package fetch

import (
	"context"
	"regexp"
	"sync"
	"time"
)

// ListingTTL is the freshness window for listing pages (3 hours).
const ListingTTL = 3 * time.Hour

// DetailTTL is the freshness window for detail pages (24 hours).
const DetailTTL = 24 * time.Hour

// BlockedResult is a typed result returned when the server blocks access with a
// 403 response or a CAPTCHA challenge. It is not an error; callers must check for
// it explicitly and treat the page as unavailable without retrying immediately.
type BlockedResult struct {
	URL    string
	Reason string // "403" or "captcha"
}

// captchaPattern matches page bodies that signal a CAPTCHA or human-verification
// challenge. The list mirrors the TypeScript application's network layer.
var captchaPattern = regexp.MustCompile(
	`(?i)verify you are human|cf-chl-|captcha-container|just a moment\.\.\.`)

// isBlocked returns a non-empty reason when the response indicates blocking.
func isBlocked(status int, body string) string {
	if status == 403 {
		return "403"
	}
	if captchaPattern.MatchString(body) {
		return "captcha"
	}
	return ""
}

// cacheEntry stores a cached page and the time (in Unix milliseconds) at which it expires.
type cacheEntry struct {
	page      *Page
	expiresMS int64
}

// pageCache is an in-memory cache keyed by URL. Callers choose the TTL per put.
type pageCache struct {
	mu    sync.Mutex
	items map[string]cacheEntry
	nowMS func() int64
}

func newPageCache(nowMS func() int64) *pageCache {
	if nowMS == nil {
		nowMS = func() int64 { return time.Now().UnixMilli() }
	}
	return &pageCache{items: make(map[string]cacheEntry), nowMS: nowMS}
}

func (c *pageCache) get(url string) *Page {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[url]
	if !ok || c.nowMS() > e.expiresMS {
		delete(c.items, url)
		return nil
	}
	p := *e.page
	p.Cached = true
	return &p
}

func (c *pageCache) put(url string, page *Page, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[url] = cacheEntry{
		page:      page,
		expiresMS: c.nowMS() + ttl.Milliseconds(),
	}
}

// GetListing returns a cached listing page, or nil on a cache miss.
func (cl *Client) GetListing(ctx context.Context, rawURL string) (*Page, *BlockedResult, error) {
	if p := cl.cache.get(rawURL); p != nil {
		return p, nil, nil
	}
	page, err := cl.Get(ctx, rawURL)
	if err != nil {
		return nil, nil, err
	}
	if reason := isBlocked(page.StatusCode, page.Body); reason != "" {
		return nil, &BlockedResult{URL: page.URL, Reason: reason}, nil
	}
	cl.cache.put(rawURL, page, ListingTTL)
	return page, nil, nil
}

// GetDetail returns a cached detail page, or nil on a cache miss.
func (cl *Client) GetDetail(ctx context.Context, rawURL string) (*Page, *BlockedResult, error) {
	if p := cl.cache.get(rawURL); p != nil {
		return p, nil, nil
	}
	page, err := cl.Get(ctx, rawURL)
	if err != nil {
		return nil, nil, err
	}
	if reason := isBlocked(page.StatusCode, page.Body); reason != "" {
		return nil, &BlockedResult{URL: page.URL, Reason: reason}, nil
	}
	cl.cache.put(rawURL, page, DetailTTL)
	return page, nil, nil
}
