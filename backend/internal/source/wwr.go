package source

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
)

// WWRPattern is the registry host-pattern for weworkremotely.com.
const WWRPattern = `weworkremotely\.com$`

// wwrRSSPath is appended to the listing host to derive the RSS feed URL.
const wwrRSSPath = "/remote-jobs.rss"

// WWRSource implements Source for weworkremotely.com. The site's individual
// job pages are blocked by Cloudflare for a plain client, but the RSS feed
// provides entity-escaped HTML descriptions with real headings and lists.
// Resolve is a no-op for refs discovered from the RSS (they are already
// complete); for refs that require a page fetch it records the block.
type WWRSource struct {
	client *fetch.Client
}

// NewWWRSource creates a WWRSource backed by client.
func NewWWRSource(client *fetch.Client) *WWRSource {
	return &WWRSource{client: client}
}

// Discover derives the RSS URL from listingURL, fetches it, and returns refs.
// The RSS description field carries entity-escaped HTML with h3, p, ul, li,
// which ParseFeed already passes through collect.StructuredHTML.
func (s *WWRSource) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	rssURL, err := wwrRSSURL(listingURL)
	if err != nil {
		return nil, fmt.Errorf("wwr discover: build rss url from %q: %w", listingURL, err)
	}
	page, blocked, err := s.client.GetListingWithDiscipline(ctx, rssURL)
	if err != nil {
		return nil, fmt.Errorf("wwr discover %q: %w", rssURL, err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("wwr discover %q: blocked (%s)", rssURL, blocked.Reason)
	}
	jobs := collect.ParseFeed(page.Body, page.URL, "")
	refs := make([]Ref, 0, len(jobs))
	for _, j := range jobs {
		refs = append(refs, Ref{URL: j.OriginalURL, Partial: j})
	}
	return refs, nil
}

// wwrRSSURL builds the RSS feed URL from the listing URL by replacing the path
// with wwrRSSPath, keeping the scheme and host.
func wwrRSSURL(listingURL string) (string, error) {
	u, err := url.Parse(listingURL)
	if err != nil {
		return "", err
	}
	u.Path = wwrRSSPath
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// Resolve returns the posting from ref.Partial when it already has a complete
// description (the typical case when the ref came from the RSS feed). If the
// description is missing, it fetches the job page; since We Work Remotely pages
// are frequently blocked by Cloudflare, it records the block rather than
// retrying. For unblocked pages it extracts the job content block from the body,
// stripping the page navigation chrome.
func (s *WWRSource) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	if isComplete(ref.Partial) {
		return ref.Partial, nil
	}
	page, blocked, err := s.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("wwr resolve %q: %w", ref.URL, err)
	}
	if blocked != nil {
		j := ref.Partial
		if j.OriginalURL == "" {
			j.OriginalURL = ref.URL
		}
		j.DescriptionStatus = "blocked: " + blocked.Reason
		return j, nil
	}

	j := ref.Partial
	if j.OriginalURL == "" {
		j.OriginalURL = ref.URL
	}

	// Extract the description block from the page body, stripping navigation
	// chrome ("× All jobs …", "Posted N day ago").
	bodyForConversion := wwrExtractContent(page.Body)
	h := collect.StructuredHTML(bodyForConversion)
	j.DescriptionHTML = h
	j.Description = plainOrFallback(h, bodyForConversion)
	j.DescriptionStatus = descStatus(j.Description)
	j.Original = []byte(page.Body)
	j.OriginalKind = "html/page"
	return j, nil
}

// wwrExtractContent extracts the job description block from a We Work Remotely
// page body. It looks for the listing-container section that holds the actual
// job description, skipping the navigation chrome and footer. When the
// container is not found (page structure change), the full body is returned.
func wwrExtractContent(body string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return body
	}

	// Try known container selectors in order of specificity.
	for _, sel := range []string{
		"section.listing-container",
		".listing-container",
		"section.container-with-block",
		"article.listing",
		"[class*='listing-container']",
	} {
		if s := doc.Find(sel); s.Length() > 0 {
			html, err := s.Html()
			if err == nil && strings.TrimSpace(html) != "" {
				return html
			}
		}
	}

	// No known container found — return the full body and let StructuredHTML
	// remove nav/header/footer elements as far as it can.
	return body
}

// wwrExtractContentFromBody is an exported variant for direct testing without
// an HTTP client.
func wwrExtractContentFromBody(body string) string {
	return wwrExtractContent(body)
}

// resolveWWRBody processes a raw page body string into the structured forms.
// Exported for unit testing the body conversion path without an HTTP client.
func resolveWWRBody(body string) (descriptionHTML, description string) {
	content := wwrExtractContent(body)
	h := collect.StructuredHTML(content)
	plain := plainOrFallback(h, content)
	return h, plain
}
