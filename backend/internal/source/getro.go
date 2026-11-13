package source

import (
	"context"
	"fmt"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
)

// GetroAdapter implements Source for Getro-powered portfolio job boards. It
// fetches the board page and uses the existing getroJobs parser (via HTMLLinks)
// to extract postings from the #__NEXT_DATA__ hydration payload.
type GetroAdapter struct {
	client *fetch.Client
}

// NewGetroAdapter creates a GetroAdapter backed by client.
func NewGetroAdapter(client *fetch.Client) *GetroAdapter {
	return &GetroAdapter{client: client}
}

// Discover fetches the Getro board page and parses the #__NEXT_DATA__ payload.
func (g *GetroAdapter) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	page, blocked, err := g.client.GetListingWithDiscipline(ctx, listingURL)
	if err != nil {
		return nil, fmt.Errorf("getro discover %q: %w", listingURL, err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("getro discover %q: blocked (%s)", listingURL, blocked.Reason)
	}

	// HTMLLinks already detects and parses the Getro #__NEXT_DATA__ payload.
	jobs := collect.HTMLLinks(page.Body, page.URL, "")
	refs := make([]Ref, 0, len(jobs))
	for _, j := range jobs {
		refs = append(refs, Ref{URL: j.OriginalURL, Partial: j})
	}
	return refs, nil
}

// Resolve returns the posting from Ref.Partial when it is already complete.
// Getro postings from the board page include a skills list rather than a full
// description; a detail fetch is attempted when description is empty.
func (g *GetroAdapter) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	if isComplete(ref.Partial) {
		return ref.Partial, nil
	}
	page, blocked, err := g.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("getro resolve %q: %w", ref.URL, err)
	}
	if blocked != nil {
		j := ref.Partial
		j.DescriptionStatus = "blocked: " + blocked.Reason
		return j, nil
	}
	if jobs := collect.StructuredJobs(page.Body, page.URL, ""); len(jobs) > 0 {
		if j, ok := matchStructuredJob(jobs, ref); ok {
			return j, nil
		}
	}
	j := ref.Partial
	h := collect.StructuredHTML(page.Body)
	j.Description = plainOrFallback(h, page.Body)
	j.DescriptionHTML = h
	j.DescriptionStatus = descStatus(j.Description)
	return j, nil
}

// getroRef builds a Ref from a JobInput produced by HTMLLinks/getroJobs.
func getroRef(j model.JobInput) Ref {
	return Ref{URL: j.OriginalURL, Partial: j}
}
