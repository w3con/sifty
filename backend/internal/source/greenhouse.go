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

// greenhouseHostRe matches the Greenhouse job-board domains.
var greenhouseHostRe = regexp.MustCompile(`(?i)(?:boards|job-boards)\.greenhouse\.io`)

// GreenhouseAdapter implements Source for boards hosted on Greenhouse. It uses
// the public boards-api.greenhouse.io JSON endpoint to list and describe
// postings without rendering any HTML.
type GreenhouseAdapter struct {
	client *fetch.Client
}

// NewGreenhouseAdapter creates a GreenhouseAdapter backed by client.
func NewGreenhouseAdapter(client *fetch.Client) *GreenhouseAdapter {
	return &GreenhouseAdapter{client: client}
}

// companySlug extracts the board slug from a boards.greenhouse.io URL.
// E.g. "https://boards.greenhouse.io/acmecorp" → "acmecorp".
func companySlug(boardURL string) (string, error) {
	u, err := url.Parse(boardURL)
	if err != nil {
		return "", fmt.Errorf("greenhouse: bad board URL %q: %w", boardURL, err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return "", fmt.Errorf("greenhouse: cannot derive company slug from %q", boardURL)
	}
	return parts[0], nil
}

type ghJob struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	AbsoluteURL    string `json:"absolute_url"`
	UpdatedAt      string `json:"updated_at"`      // modification time, not publication
	FirstPublished string `json:"first_published"` // genuine publication when supplied
	Location       struct {
		Name string `json:"name"`
	} `json:"location"`
}

type ghBoard struct {
	Jobs []ghJob `json:"jobs"`
}

// Discover fetches the boards-api.greenhouse.io JSON endpoint and returns one
// Ref per posting. Each Ref carries a complete JobInput because the endpoint
// returns the full description.
func (g *GreenhouseAdapter) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	slug, err := companySlug(listingURL)
	if err != nil {
		return nil, err
	}
	apiURL := "https://boards-api.greenhouse.io/v1/boards/" + slug + "/jobs?content=true"
	page, blocked, err := g.client.GetListingWithDiscipline(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("greenhouse discover: %w", err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("greenhouse discover %q: blocked (%s)", apiURL, blocked.Reason)
	}

	var board ghBoard
	if err := json.Unmarshal([]byte(page.Body), &board); err != nil {
		return nil, fmt.Errorf("greenhouse: parse board JSON: %w", err)
	}

	refs := make([]Ref, 0, len(board.Jobs))
	for _, j := range board.Jobs {
		if j.Title == "" || j.AbsoluteURL == "" {
			continue
		}
		refs = append(refs, Ref{
			URL: j.AbsoluteURL,
			Partial: model.JobInput{
				Title:             j.Title,
				Company:           slug, // refined by Resolve if company name is available
				OriginalURL:       j.AbsoluteURL,
				Description:       collect.CleanHTML(j.Content),
				DescriptionStatus: descStatus(j.Content),
				Location:          j.Location.Name,
				PublishedAt:       model.NormalizePublishedAt(j.FirstPublished),
				PublishedBasis:    model.PublicationBasis(model.NormalizePublishedAt(j.FirstPublished), "first_published"),
			},
		})
	}
	return refs, nil
}

// Resolve returns the posting from the Ref.Partial when it is already complete,
// otherwise the detail page is fetched. For Greenhouse the board API already
// returns full descriptions, so the detail fetch is rarely needed.
func (g *GreenhouseAdapter) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	if isComplete(ref.Partial) {
		return ref.Partial, nil
	}
	page, blocked, err := g.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("greenhouse resolve %q: %w", ref.URL, err)
	}
	if blocked != nil {
		j := ref.Partial
		j.DescriptionStatus = "blocked: " + blocked.Reason
		return j, nil
	}
	// Try JSON-LD on the detail page.
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

// descStatus returns the description status string based on whether content is present.
func descStatus(content string) string {
	if strings.TrimSpace(content) != "" {
		return "available structured description"
	}
	return "missing"
}

// isGreenhouseURL returns true when the URL is hosted on a Greenhouse board domain.
func isGreenhouseURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return greenhouseHostRe.MatchString(u.Hostname())
}
