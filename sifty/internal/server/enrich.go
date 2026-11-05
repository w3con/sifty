package server

import (
	"context"
	"fmt"
	"net/url"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/source"
	"github.com/w3con/sifty/internal/store"
)

// FetchClient is the outbound HTTP client shared by importURL and EnrichFn.
// When nil, both paths create a default client on first call.
var defaultFetchClient *fetch.Client

func getDefaultFetchClient() *fetch.Client {
	if defaultFetchClient == nil {
		c, err := fetch.New(fetch.Options{})
		if err != nil {
			panic("server: cannot create default fetch client: " + err.Error())
		}
		defaultFetchClient = c
	}
	return defaultFetchClient
}

// buildServerRegistry returns the source registry used by the server-side import
// and enrich paths. The registry is stateless; it is cheap to construct per call.
func buildServerRegistry(client *fetch.Client) *source.Registry {
	generic := source.NewGenericSource(client)
	reg := source.NewRegistry(generic)
	reg.Register(`(?:boards|job-boards)\.greenhouse\.io`, source.NewGreenhouseAdapter(client))
	reg.Register(`jobs\.lever\.co`, source.NewLeverAdapter(client))
	reg.Register(`jobs\.ashbyhq\.com`, source.NewAshbyAdapter(client))
	return reg
}

// ImportURL fetches rawURL, parses it through the source registry and upserts it.
// It replaces the "Manual import" placeholder that the original importURL wrote.
func ImportURL(s *store.Store, client *fetch.Client, rawURL string) (int64, error) {
	if client == nil {
		client = getDefaultFetchClient()
	}
	reg := buildServerRegistry(client)
	h := hostOf(rawURL)
	src := reg.For(h)

	ctx := context.Background()
	refs, err := src.Discover(ctx, rawURL)
	if err != nil || len(refs) == 0 {
		// Fall back: resolve the URL directly as a single-posting detail page.
		ref := source.Ref{URL: rawURL, Partial: model.JobInput{
			OriginalURL:       rawURL,
			Source:            "manual",
			DescriptionStatus: "pending detail fetch",
		}}
		j, err2 := src.Resolve(ctx, ref)
		if err2 != nil {
			// Last resort: create a minimal record so the caller gets an ID.
			j = ref.Partial
			j.Title = "Imported role"
			j.Company = "Unknown"
		}
		if j.Source == "" {
			j.Source = "manual"
		}
		return s.Upsert(j, false)
	}

	// Use the first ref (import is for a single URL).
	ref := refs[0]
	ref.Source = "manual"
	j, err := src.Resolve(ctx, ref)
	if err != nil {
		j = ref.Partial
	}
	j.Source = "manual"
	return s.Upsert(j, false)
}

// EnrichJob re-fetches the detail page for id and updates the job's description.
// It returns the (resolved) job ID after the update.
func EnrichJob(s *store.Store, client *fetch.Client, id int64) (int64, error) {
	if client == nil {
		client = getDefaultFetchClient()
	}
	job, err := s.Get(id)
	if err != nil {
		return 0, fmt.Errorf("enrich: get job %d: %w", id, err)
	}

	ctx := context.Background()
	page, blocked, err := client.GetDetailWithDiscipline(ctx, job.OriginalURL)
	if err != nil {
		return job.ID, fmt.Errorf("enrich: fetch %q: %w", job.OriginalURL, err)
	}
	if blocked != nil {
		return job.ID, fmt.Errorf("enrich: %q is blocked (%s)", job.OriginalURL, blocked.Reason)
	}

	// Try JSON-LD first; fall back to cleaned HTML.
	var description, status string
	if jobs := collect.StructuredJobs(page.Body, page.URL, ""); len(jobs) > 0 {
		description = jobs[0].Description
		status = jobs[0].DescriptionStatus
	}
	if description == "" {
		description = collect.CleanHTML(page.Body)
		if description != "" {
			status = "available cleaned html"
		} else {
			status = "missing"
		}
	}

	// Update only the description fields; leave all other fields as-is.
	_, err = s.DB.Exec(
		"UPDATE jobs SET description=?, description_status=? WHERE id=?",
		description, status, job.ID)
	if err != nil {
		return job.ID, fmt.Errorf("enrich: update job %d: %w", job.ID, err)
	}
	return job.ID, nil
}

// hostOf extracts the hostname from rawURL.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Hostname()
}
