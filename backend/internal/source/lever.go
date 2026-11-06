package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
)

// leverHostRe matches the Lever job-board domains.
var leverHostRe = regexp.MustCompile(`(?i)jobs\.lever\.co`)

// LeverAdapter implements Source for boards hosted on Lever. It uses the public
// api.lever.co/v0/postings endpoint to list and describe postings in JSON.
type LeverAdapter struct {
	client *fetch.Client
}

// NewLeverAdapter creates a LeverAdapter backed by client.
func NewLeverAdapter(client *fetch.Client) *LeverAdapter {
	return &LeverAdapter{client: client}
}

// leverCompanySlug derives the company slug from a jobs.lever.co board URL.
// E.g. "https://jobs.lever.co/betacorp" → "betacorp".
func leverCompanySlug(boardURL string) (string, error) {
	u, err := url.Parse(boardURL)
	if err != nil {
		return "", fmt.Errorf("lever: bad board URL %q: %w", boardURL, err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return "", fmt.Errorf("lever: cannot derive company slug from %q", boardURL)
	}
	return parts[0], nil
}

type leverPosting struct {
	ID         string `json:"id"`
	Text       string `json:"text"` // posting title
	HostedURL  string `json:"hostedUrl"`
	ApplyURL   string `json:"applyUrl"`
	CreatedAt  int64  `json:"createdAt"` // Unix milliseconds
	Categories struct {
		Location   string `json:"location"`
		Department string `json:"department"`
		Commitment string `json:"commitment"`
	} `json:"categories"`
	Description      string `json:"description"`
	DescriptionPlain string `json:"descriptionPlain"`
	Lists            []struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	} `json:"lists"`
	Additional string `json:"additional"`
}

// Discover fetches the api.lever.co/v0/postings endpoint and returns one Ref per
// posting. The slug is derived from the board URL path.
func (l *LeverAdapter) Discover(ctx context.Context, listingURL string) ([]Ref, error) {
	slug, err := leverCompanySlug(listingURL)
	if err != nil {
		return nil, err
	}
	apiURL := "https://api.lever.co/v0/postings/" + slug + "?mode=json"
	page, blocked, err := l.client.GetListingWithDiscipline(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("lever discover: %w", err)
	}
	if blocked != nil {
		return nil, fmt.Errorf("lever discover %q: blocked (%s)", apiURL, blocked.Reason)
	}

	var postings []leverPosting
	if err := json.Unmarshal([]byte(page.Body), &postings); err != nil {
		return nil, fmt.Errorf("lever: parse postings JSON: %w", err)
	}

	refs := make([]Ref, 0, len(postings))
	for _, p := range postings {
		if p.Text == "" || p.HostedURL == "" {
			continue
		}
		desc := leverDescription(p)
		refs = append(refs, Ref{
			URL: p.HostedURL,
			Partial: model.JobInput{
				Title:             p.Text,
				Company:           titleCase(slug),
				OriginalURL:       p.HostedURL,
				Description:       collect.CleanHTML(desc),
				DescriptionStatus: descStatus(desc),
				Location:          p.Categories.Location,
				PublishedAt:       leverPublishedAt(p.CreatedAt),
			},
		})
	}
	return refs, nil
}

// Resolve returns the posting from the Ref.Partial when complete.
func (l *LeverAdapter) Resolve(ctx context.Context, ref Ref) (model.JobInput, error) {
	if isComplete(ref.Partial) {
		return ref.Partial, nil
	}
	page, blocked, err := l.client.GetDetailWithDiscipline(ctx, ref.URL)
	if err != nil {
		return model.JobInput{}, fmt.Errorf("lever resolve %q: %w", ref.URL, err)
	}
	if blocked != nil {
		j := ref.Partial
		j.DescriptionStatus = "blocked: " + blocked.Reason
		return j, nil
	}
	if jobs := collect.StructuredJobs(page.Body, page.URL, ""); len(jobs) > 0 {
		return jobs[0], nil
	}
	j := ref.Partial
	j.Description = collect.CleanHTML(page.Body)
	j.DescriptionStatus = descStatus(j.Description)
	return j, nil
}

// leverDescription concatenates description + list items + additional into one HTML string.
func leverDescription(p leverPosting) string {
	var sb strings.Builder
	sb.WriteString(p.Description)
	for _, li := range p.Lists {
		if li.Content != "" {
			sb.WriteString("\n<ul>")
			sb.WriteString(li.Content)
			sb.WriteString("</ul>")
		}
	}
	if p.Additional != "" {
		sb.WriteString("\n")
		sb.WriteString(p.Additional)
	}
	return sb.String()
}

// leverPublishedAt converts a Unix-milliseconds timestamp to an ISO 8601 string.
func leverPublishedAt(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

// titleCase converts a slug like "betacorp" to "Betacorp".
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
