package source

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
)

// web3CareerJobPathRe matches the path pattern of web3.career individual job
// pages: /<title-slug>/<numeric-id>, e.g. /analytics-engineer-bcbgroup/155406.
// The id is 5 or more digits; category and other non-job pages use short
// non-numeric paths like /web3-jobs, /companies, /salary.
var web3CareerJobPathRe = regexp.MustCompile(`^/[a-z0-9][a-z0-9-]*/\d{5,}$`)

// Web3CareerPattern is the registry host-pattern that covers both web3.career
// (the short domain used in vacancy URLs) and web3.careers (the main site,
// used in search.yaml). Register only this constant — never the literal string.
const Web3CareerPattern = `web3\.careers?$`

// Web3CareersSource implements Source for web3.career / web3.careers. The
// site's JSON-LD description field is unreliable: apostrophes are stripped and
// entities are double-encoded ("You ll be…", "Data &amp;amp; Analytics"). This
// source always reads the job page BODY — which carries real h3, h4, p, ul,
// li, br markup — instead of the JSON-LD string.
type Web3CareersSource struct {
	client *fetch.Client
}

// NewWeb3CareersSource creates a Web3CareersSource backed by client.
// The client's existing rate limits and discipline rules are used unchanged.
func NewWeb3CareersSource(client *fetch.Client) *Web3CareersSource {
	return &Web3CareersSource{client: client}
}

// Discover fetches the listing page and extracts individual job links by
// matching anchors whose path satisfies web3CareerJobPathRe
// (/<title-slug>/<numeric-id>). HTMLLinks does not match this pattern (its
// regex requires a /job(s)/ prefix) and StructuredJobs falls back to the page
// URL when the JSON-LD has no url field, producing home-page refs. The custom
// extractor deduplicates by absolute URL and clears all descriptions; Resolve
// reads the individual page body for the real description.
func (s *Web3CareersSource) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	page, blocked, err := s.client.GetListingWithDiscipline(ctx, listingURL)
	if err != nil {
		return nil, fmt.Errorf("web3careers discover %q: %w", listingURL, err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("web3careers discover %q: blocked (%s)", listingURL, blocked.Reason)
	}
	return web3CareerJobRefs(page.Body, page.URL), nil
}

// web3CareerJobRefs extracts job refs from a web3.career listing page by
// finding anchors whose path matches web3CareerJobPathRe. It is exposed so
// tests can call it without an HTTP client.
func web3CareerJobRefs(body, pageURL string) []Ref {
	base, _ := url.Parse(pageURL)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var refs []Ref
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		u, err := url.Parse(href)
		if err != nil || !web3CareerJobPathRe.MatchString(u.Path) {
			return
		}
		abs := base.ResolveReference(u)
		abs.RawQuery = ""
		abs.Fragment = ""
		key := abs.String()
		if seen[key] {
			return
		}
		seen[key] = true
		label := strings.TrimSpace(sel.Text())
		refs = append(refs, Ref{
			URL: key,
			Partial: model.JobInput{
				OriginalURL:       key,
				Title:             label,
				DescriptionStatus: "pending body fetch",
			},
		})
	})
	return refs
}

// Resolve fetches the individual job page and extracts the description from the
// content sections only — skipping the job header chrome (title, salary, Apply
// button, "Join talent pool", timestamps) and skipping JSON-LD entirely (which
// is damaged for this source: stripped apostrophes, double-encoded entities).
func (s *Web3CareersSource) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	page, blocked, err := s.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("web3careers resolve %q: %w", ref.URL, err)
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
	h := web3CareerExtractDescription(page.Body)
	if h == "" {
		// Fallback: full-body extraction if the page has an unexpected layout.
		h = collect.StructuredHTML(page.Body)
	}
	j.DescriptionHTML = h
	j.Description = plainOrFallback(h, page.Body)
	if j.Description != "" {
		j.DescriptionStatus = "available body html"
	} else {
		j.DescriptionStatus = "missing"
	}
	j.Original = []byte(page.Body)
	j.OriginalKind = "html/page"
	return j, nil
}

// web3CareerExtractDescription extracts the job description from a web3.career
// job page, handling two distinct page layouts:
//
//  1. Greenhouse-hosted postings: the page's content blocks carry
//     [data-qa=job-description], [data-qa=posting-requirements],
//     [data-qa=closing-description]. The header chrome (title, salary, "Apply",
//     "Join talent pool", timestamps) lives before those blocks and is excluded.
//
//  2. Native web3.career postings: the description is in a .trix-content block.
//     The block also embeds a Bondex "Join talent pool" widget at the end;
//     all anchor elements that point to bondex.app are removed before extraction.
//
// Returns the sanitised HTML fragment or "" when neither selector matches.
// Exported for testing without an HTTP client.
func web3CareerExtractDescription(body string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ""
	}

	// Layout 1 — Greenhouse-hosted.
	var parts strings.Builder
	doc.Find(`[data-qa=job-description], [data-qa=posting-requirements], [data-qa=closing-description]`).Each(func(_ int, s *goquery.Selection) {
		html, err := s.Html()
		if err == nil && strings.TrimSpace(html) != "" {
			parts.WriteString(html)
			parts.WriteString("\n")
		}
	})
	if raw := strings.TrimSpace(parts.String()); raw != "" {
		return collect.StructuredHTML(raw)
	}

	// Layout 2 — Native web3.career posting.
	// The description lives inside div.text-dark-grey-text. The page HTML has a
	// malformed attribute sequence ("style=…"class=trix-content, no space between
	// attributes), so the HTML parser drops the second class attribute and the
	// .trix-content selector matches nothing. div.text-dark-grey-text is the
	// stable outer container that the parser does preserve.
	//
	// Remove Bondex "Join talent pool" widget anchors and their parent containers
	// first — the widget is embedded inside the content block but is not part of
	// the job description.
	doc.Find(`a[href*="bondex.app"]`).Each(func(_ int, s *goquery.Selection) {
		s.Parent().Remove()
	})
	container := doc.Find(`div.text-dark-grey-text`).First()
	if container.Length() == 0 {
		return ""
	}
	// Remove trailing "Apply Now:" page chrome. The native layout's container ends
	// with <p>Apply Now:</p> which is a site button label, not part of the job
	// description. Remove only the last child when its text is exactly "Apply Now:"
	// or "Apply Now" (case-insensitive) so the word is preserved when it appears
	// inside the actual job text.
	if last := container.Children().Last(); last.Length() > 0 {
		t := strings.TrimSpace(strings.ToLower(last.Text()))
		if t == "apply now:" || t == "apply now" {
			last.Remove()
		}
	}
	raw, err := container.Html()
	if err != nil || strings.TrimSpace(raw) == "" {
		return ""
	}
	return collect.StructuredHTML(raw)
}
