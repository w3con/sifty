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

// Remote3Pattern is the registry host-pattern for remote3.co.
const Remote3Pattern = `remote3\.co$`

// remote3RSSPath is the RSS endpoint on the Remote3 site. The listing page
// (remote3.co/) is a Next.js application that renders job cards via JavaScript;
// the job links are not present in the server-side-rendered HTML, so HTML
// scraping returns nothing. The RSS feed contains the same listings with direct
// links to the /remote-jobs/<slug> vacancy pages.
const remote3RSSPath = "/api/rss"

// Remote3Source implements Source for remote3.co. Discover reads the RSS feed;
// Resolve reads the individual vacancy page (which has real markup).
type Remote3Source struct {
	client *fetch.Client
}

// NewRemote3Source creates a Remote3Source backed by client.
func NewRemote3Source(client *fetch.Client) *Remote3Source {
	return &Remote3Source{client: client}
}

// remote3RSSURL builds the RSS feed URL from the listing URL by replacing the
// path with the fixed RSS path, so the host from search.yaml is preserved.
func remote3RSSURL(listingURL string) (string, error) {
	u, err := url.Parse(listingURL)
	if err != nil {
		return "", err
	}
	u.Path = remote3RSSPath
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// Discover fetches the Remote3 RSS feed and returns refs for each item. The RSS
// description field is a one-line metadata snippet ("at Bybit - Full-Time -
// Worldwide"), not a job description; all descriptions are cleared and Resolve
// fetches the individual vacancy page for the real content.
func (s *Remote3Source) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	rssURL, err := remote3RSSURL(listingURL)
	if err != nil {
		return nil, fmt.Errorf("remote3 discover: build rss url from %q: %w", listingURL, err)
	}
	page, blocked, err := s.client.GetListingWithDiscipline(ctx, rssURL)
	if err != nil {
		return nil, fmt.Errorf("remote3 discover %q: %w", rssURL, err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("remote3 discover %q: blocked (%s)", rssURL, blocked.Reason)
	}
	items := collect.ParseFeed(page.Body, page.URL, "")
	refs := make([]Ref, 0, len(items))
	for _, j := range items {
		if j.OriginalURL == "" {
			continue
		}
		j.Description = ""
		j.DescriptionHTML = ""
		j.DescriptionStatus = "pending body fetch"
		refs = append(refs, Ref{URL: j.OriginalURL, Partial: j})
	}
	return refs, nil
}

// Resolve fetches the individual vacancy page and extracts the description from
// the [class*="RemoteJobs_jobDescription"] block, which contains the real job
// markup (h2, p, ul, li, links). The class uses a Next.js hash suffix that may
// change between deployments, so a substring match is used.
func (s *Remote3Source) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	page, blocked, err := s.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("remote3 resolve %q: %w", ref.URL, err)
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
	h := remote3ExtractDescription(page.Body)
	if h == "" {
		// Fallback to full-body extraction when the page layout changes.
		h = collect.StructuredHTML(page.Body)
	}
	j.DescriptionHTML = h
	j.Description = plainOrFallback(h, page.Body)
	j.DescriptionStatus = descStatus(j.Description)
	j.Original = []byte(page.Body)
	j.OriginalKind = "html/page"
	return j, nil
}

// remote3ExtractDescription extracts the job description from a Remote3 vacancy
// page by selecting the element whose class contains "RemoteJobs_jobDescription".
// The class uses a Next.js hash suffix that varies between deployments; the
// substring match makes the selector resilient to that.
//
// Before passing the inner HTML to collect.StructuredHTML, it strips leading
// chrome elements: image-only links (empty text) and "Back to jobs" breadcrumb
// links that appear above the real h2 heading on many vacancy pages. Exported for testing.
func remote3ExtractDescription(body string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ""
	}
	var inner string
	doc.Find(`[class*="RemoteJobs_jobDescription"]`).Each(func(_ int, s *goquery.Selection) {
		if inner != "" {
			return
		}
		// Strip leading chrome: logo image links (empty text) and "Back to jobs"
		// breadcrumb links that appear before the real heading content.
		for {
			first := s.Children().First()
			if first.Length() == 0 {
				break
			}
			text := strings.TrimSpace(first.Text())
			lower := strings.ToLower(text)
			if lower == "" || lower == "back to jobs" {
				first.Remove()
			} else {
				break
			}
		}
		html, err := s.Html()
		if err == nil {
			inner = html
		}
	})
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	return collect.StructuredHTML(inner)
}
