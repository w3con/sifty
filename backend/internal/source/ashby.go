package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
)

// ashbyHostRe matches the Ashby job-board domain.
var ashbyHostRe = regexp.MustCompile(`(?i)jobs\.ashbyhq\.com`)

// AshbyAdapter implements Source for boards hosted on Ashby HQ. It uses the
// public api.ashbyhq.com/posting-api/job-board endpoint to list and describe
// postings in JSON.
type AshbyAdapter struct {
	client *fetch.Client
}

// NewAshbyAdapter creates an AshbyAdapter backed by client.
func NewAshbyAdapter(client *fetch.Client) *AshbyAdapter {
	return &AshbyAdapter{client: client}
}

// ashbyCompanySlug derives the company slug from a jobs.ashbyhq.com board URL.
func ashbyCompanySlug(boardURL string) (string, error) {
	u, err := url.Parse(boardURL)
	if err != nil {
		return "", fmt.Errorf("ashby: bad board URL %q: %w", boardURL, err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return "", fmt.Errorf("ashby: cannot derive company slug from %q", boardURL)
	}
	return parts[0], nil
}

type ashbyPosting struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	LocationName    string `json:"locationName"`
	IsListed        bool   `json:"isListed"`
	PublishedDate   string `json:"publishedDate"`
	DescriptionHTML string `json:"descriptionHtml"`
	JobURL          string `json:"jobUrl"`
	ApplyURL        string `json:"applyUrl"`
	DepartmentName  string `json:"departmentName"`
}

type ashbyBoardResponse struct {
	Success bool `json:"success"`
	Data    struct {
		JobBoard struct {
			JobPostings []ashbyPosting `json:"jobPostings"`
		} `json:"jobBoard"`
		Organization struct {
			Name string `json:"name"`
		} `json:"organization"`
	} `json:"data"`
}

// Discover fetches the api.ashbyhq.com/posting-api/job-board endpoint and
// returns one Ref per listed posting.
func (a *AshbyAdapter) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	slug, err := ashbyCompanySlug(listingURL)
	if err != nil {
		return nil, err
	}
	apiURL := "https://api.ashbyhq.com/posting-api/job-board/" + slug
	page, blocked, err := a.client.GetListingWithDiscipline(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("ashby discover: %w", err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("ashby discover %q: blocked (%s)", apiURL, blocked.Reason)
	}

	var resp ashbyBoardResponse
	if err := json.Unmarshal([]byte(page.Body), &resp); err != nil {
		return nil, fmt.Errorf("ashby: parse board JSON: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("ashby: board API returned success=false for %q", listingURL)
	}

	companyName := resp.Data.Organization.Name
	if companyName == "" {
		companyName = titleCase(slug)
	}

	refs := make([]Ref, 0)
	for _, p := range resp.Data.JobBoard.JobPostings {
		if !p.IsListed || p.Title == "" || p.JobURL == "" {
			continue
		}
		refs = append(refs, Ref{
			URL: p.JobURL,
			Partial: model.JobInput{
				Title:             p.Title,
				Company:           companyName,
				OriginalURL:       p.JobURL,
				Description:       collect.CleanHTML(p.DescriptionHTML),
				DescriptionStatus: descStatus(p.DescriptionHTML),
				Location:          p.LocationName,
				PublishedAt:       model.NormalizePublishedAt(p.PublishedDate),
				PublishedBasis:    model.PublicationBasis(model.NormalizePublishedAt(p.PublishedDate), "published_date"),
			},
		})
	}
	return refs, nil
}

// Resolve returns the posting from the Ref.Partial when complete.
func (a *AshbyAdapter) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	if isComplete(ref.Partial) {
		return ref.Partial, nil
	}
	page, blocked, err := a.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("ashby resolve %q: %w", ref.URL, err)
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
	j.Description = collect.CleanHTML(page.Body)
	j.DescriptionStatus = descStatus(j.Description)
	return j, nil
}
