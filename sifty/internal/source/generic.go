package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
)

// GenericSource implements Source using the existing JSON-LD, RSS/Atom feed, and
// HTML-link parse primitives. For listing pages it runs the waterfall in the
// order that matches the TypeScript application: JSON-LD first, then
// feed/HTML-link. For detail pages it cleans the HTML body and sets the
// description to the cleaned text.
type GenericSource struct {
	client *fetch.Client
}

// NewGenericSource creates a GenericSource backed by client.
func NewGenericSource(client *fetch.Client) *GenericSource {
	return &GenericSource{client: client}
}

// Discover fetches listingURL and returns references for every posting found.
// The waterfall tries JSON-LD first; if that yields results they are returned
// directly. Otherwise it falls back to feed parsing, then HTML-link extraction.
func (g *GenericSource) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	page, blocked, err := g.client.GetListingWithDiscipline(ctx, listingURL)
	if err != nil {
		return nil, fmt.Errorf("generic discover %q: %w", listingURL, err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("generic discover %q: blocked (%s)", listingURL, blocked.Reason)
	}

	// JSON-LD waterfall — covers platform boards that embed structured data.
	if jobs := collect.StructuredJobs(page.Body, page.URL, ""); len(jobs) > 0 {
		refs := make([]Ref, 0, len(jobs))
		for _, j := range jobs {
			refs = append(refs, jobToRef(j))
		}
		return refs, nil
	}

	// Feed waterfall — RSS/Atom boards.
	ct := strings.ToLower(page.ContentType)
	if strings.Contains(ct, "xml") || strings.Contains(ct, "rss") || strings.Contains(ct, "atom") ||
		strings.HasSuffix(strings.ToLower(listingURL), ".xml") {
		if jobs := collect.ParseFeed(page.Body, page.URL, ""); len(jobs) > 0 {
			refs := make([]Ref, 0, len(jobs))
			for _, j := range jobs {
				refs = append(refs, jobToRef(j))
			}
			return refs, nil
		}
	}

	// Generic HTML-link extraction — includes Getro hydration payload.
	jobs := collect.HTMLLinks(page.Body, page.URL, "")
	refs := make([]Ref, 0, len(jobs))
	for _, j := range jobs {
		refs = append(refs, jobToRef(j))
	}
	return refs, nil
}

// Resolve returns the full JobInput for ref. When ref.Partial already contains
// Title, Company and Description the posting is returned without a network fetch.
// Otherwise the detail page is fetched and its cleaned HTML body is used as the
// description.
func (g *GenericSource) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	if isComplete(ref.Partial) {
		return ref.Partial, nil
	}
	page, blocked, err := g.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("generic resolve %q: %w", ref.URL, err)
	}
	if blocked != nil {
		j := ref.Partial
		if j.OriginalURL == "" {
			j.OriginalURL = ref.URL
		}
		j.DescriptionStatus = "blocked: " + blocked.Reason
		return j, nil
	}

	// Try JSON-LD on the detail page first.
	if jobs := collect.StructuredJobs(page.Body, page.URL, ""); len(jobs) > 0 {
		j := jobs[0]
		if j.OriginalURL == "" {
			j.OriginalURL = ref.URL
		}
		return j, nil
	}

	// Fall back to cleaned HTML body.
	j := ref.Partial
	if j.OriginalURL == "" {
		j.OriginalURL = ref.URL
	}
	j.Description = collect.CleanHTML(page.Body)
	if j.Description != "" {
		j.DescriptionStatus = "available cleaned html"
	} else {
		j.DescriptionStatus = "missing"
	}
	return j, nil
}

// jobToRef builds a Ref from a JobInput produced by the parse primitives.
func jobToRef(j model.JobInput) Ref {
	return Ref{URL: j.OriginalURL, Partial: j}
}

// isComplete returns true when the posting does not need a detail fetch.
func isComplete(j model.JobInput) bool {
	return j.Title != "" && j.Company != "" && j.Description != ""
}
