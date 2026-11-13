// Package fetch is the outbound-HTTP client for the collection layer. All sources
// fetch through this client so the SSRF guard and response-size cap cannot be
// bypassed by a source that calls a custom transport directly.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

// DefaultSizeCap is the maximum response body the client accepts: 8 MB, matching
// the network layer of the TypeScript application.
const DefaultSizeCap = 8 << 20 // 8 MB

var (
	// ErrSSRF is returned when the target URL resolves to a private, link-local or
	// loopback IP address and AllowPrivate is false.
	ErrSSRF = errors.New("fetch: destination resolves to a private or loopback address")

	// ErrSizeCap is returned when the response body exceeds the client's size cap.
	ErrSizeCap = errors.New("fetch: response body exceeds size cap")

	// ErrTooManyRedirects is returned after six consecutive redirect responses.
	ErrTooManyRedirects = errors.New("fetch: too many redirects")
)

// Options configures a Client.
type Options struct {
	// SizeCap is the maximum response body in bytes. Zero uses DefaultSizeCap.
	SizeCap int64
	// ProxyURL is an optional outbound proxy URL. Empty means no proxy.
	ProxyURL string
	// AllowPrivate disables the SSRF guard. For tests only.
	AllowPrivate bool
	// LookupHost resolves a hostname to addresses. Nil uses net.DefaultResolver.LookupHost.
	LookupHost func(ctx context.Context, host string) ([]string, error)
	// Transport is the underlying HTTP transport. Nil builds a fresh http.Transport.
	Transport http.RoundTripper
	// NowMS returns the current time in Unix milliseconds. Nil uses time.Now().UnixMilli().
	// Used by the page cache and the traffic discipline.
	NowMS func() int64
	// SleepMS sleeps for the given number of milliseconds. Nil uses time.Sleep.
	// Used by the per-domain traffic discipline.
	SleepMS func(ms int64)
	// RandIntervalMS returns a random inter-request jitter in [0, maxMS).
	// Nil uses math/rand/v2. Used by the traffic discipline.
	RandIntervalMS func(maxMS int64) int64
}

// Page is the result of a successful outbound HTTP fetch.
type Page struct {
	URL         string
	StatusCode  int
	Body        string
	ContentType string
	ETag        string
	Modified    string
	Cached      bool // true when this response was served from the page cache
}

// Client is the single outbound HTTP client for the collection layer.
type Client struct {
	opts       Options
	hc         *http.Client
	cache      *pageCache
	discipline *trafficDiscipline
}

// New creates a Client. It returns an error if the proxy URL is malformed.
func New(opts Options) (*Client, error) {
	if opts.SizeCap == 0 {
		opts.SizeCap = DefaultSizeCap
	}
	if opts.LookupHost == nil {
		opts.LookupHost = func(ctx context.Context, host string) ([]string, error) {
			return net.DefaultResolver.LookupHost(ctx, host)
		}
	}

	var transport http.RoundTripper
	if opts.Transport != nil {
		transport = opts.Transport
	} else {
		base, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			base = &http.Transport{}
		}
		t := base.Clone()
		if opts.ProxyURL != "" {
			pu, err := url.Parse(opts.ProxyURL)
			if err != nil {
				return nil, fmt.Errorf("fetch: invalid proxy URL: %w", err)
			}
			t.Proxy = http.ProxyURL(pu)
		} else {
			t.Proxy = nil
		}
		transport = t
	}

	hc := &http.Client{
		Transport: transport,
		// Disable automatic redirect following so we can re-check SSRF on each hop.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Client{
		opts:       opts,
		hc:         hc,
		cache:      newPageCache(opts.NowMS),
		discipline: newTrafficDiscipline(opts.NowMS, opts.SleepMS, opts.RandIntervalMS),
	}, nil
}

// guardHost checks that all IPs the hostname resolves to are public.
func (c *Client) guardHost(ctx context.Context, host string) error {
	if c.opts.AllowPrivate {
		return nil
	}
	addrs, err := c.opts.LookupHost(ctx, host)
	if err != nil {
		return fmt.Errorf("%w: DNS lookup %q: %s", ErrSSRF, host, err.Error())
	}
	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip == nil {
			return fmt.Errorf("fetch: cannot parse resolved address %q for %q", a, host)
		}
		if isPrivateIP(ip) {
			return fmt.Errorf("%w: %s resolves to %s", ErrSSRF, host, a)
		}
	}
	return nil
}

// guardURL validates the URL scheme and credentials, then runs the SSRF guard.
func (c *Client) guardURL(ctx context.Context, rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch: invalid URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("fetch: URL must be http or https, got %q in %q", u.Scheme, rawURL)
	}
	if u.User != nil && u.User.String() != "" {
		return nil, fmt.Errorf("fetch: URL must not contain credentials: %q", rawURL)
	}
	if err := c.guardHost(ctx, u.Hostname()); err != nil {
		return nil, err
	}
	return u, nil
}

// Get fetches rawURL and returns the page. It follows up to six redirects,
// re-checking the SSRF guard on each hop. The response body is capped at
// opts.SizeCap. No caching or traffic discipline is applied; those sit above Get.
func (c *Client) Get(ctx context.Context, rawURL string) (*Page, error) {
	current := rawURL
	for n := 0; n < 6; n++ {
		u, err := c.guardURL(ctx, current)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("fetch: build request for %q: %w", current, err)
		}
		req.Header.Set("User-Agent", "Sifty/1.0 (personal read-only job discovery)")
		req.Header.Set("Accept", "text/html,application/json,application/rss+xml,application/xml;q=0.9,*/*;q=0.5")

		resp, err := c.hc.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch: GET %q: %w", current, err)
		}

		// Follow redirects manually so we can re-check SSRF on each hop.
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			_ = resp.Body.Close()
			loc := resp.Header.Get("Location")
			if loc == "" {
				return nil, fmt.Errorf("fetch: redirect %d with no Location from %q",
					resp.StatusCode, current)
			}
			resolved, err := url.Parse(loc)
			if err != nil {
				return nil, fmt.Errorf("fetch: bad Location %q in redirect from %q: %w",
					loc, current, err)
			}
			current = u.ResolveReference(resolved).String()
			continue
		}

		lr := io.LimitReader(resp.Body, c.opts.SizeCap+1)
		body, err := io.ReadAll(lr)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("fetch: read body from %q: %w", current, err)
		}
		if int64(len(body)) > c.opts.SizeCap {
			return nil, fmt.Errorf("%w: response from %q exceeds %d bytes",
				ErrSizeCap, current, c.opts.SizeCap)
		}
		return &Page{
			URL:         current,
			StatusCode:  resp.StatusCode,
			Body:        string(body),
			ContentType: resp.Header.Get("Content-Type"),
			ETag:        resp.Header.Get("ETag"),
			Modified:    resp.Header.Get("Last-Modified"),
		}, nil
	}
	return nil, fmt.Errorf("%w from %q", ErrTooManyRedirects, rawURL)
}

// isPrivateIP returns true for IP addresses that must not be fetched as remote
// sources: loopback, link-local, unspecified, multicast, and RFC-1918 ranges.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// privateCIDRs holds the RFC-1918 and similar private address blocks.
var privateCIDRs []*net.IPNet

func init() {
	for _, s := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7", // unique-local IPv6, covers fc00:: and fd00::
	} {
		_, cidr, err := net.ParseCIDR(s)
		if err != nil {
			panic("fetch: invalid init CIDR " + s + ": " + err.Error())
		}
		privateCIDRs = append(privateCIDRs, cidr)
	}
}
