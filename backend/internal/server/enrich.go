package server

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

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

	// Try JSON-LD first; fall back to cleaned HTML. When the detail page carries
	// several postings, use the one matching this job rather than another's data.
	var description, status, published, publishedBasis string
	if jobs := collect.StructuredJobs(page.Body, page.URL, ""); len(jobs) > 0 {
		if matched, ok := matchEnrichJob(jobs, job); ok {
			description = matched.Description
			status = matched.DescriptionStatus
			published = matched.PublishedAt
			publishedBasis = matched.PublishedBasis
		}
	}
	if description == "" {
		description = collect.CleanHTML(page.Body)
		if description != "" {
			status = "available cleaned html"
		} else {
			status = "missing"
		}
	}

	// Fill a genuinely missing date together with its source basis, or fill a
	// missing basis when the same date is confirmed. An established different
	// date and all workflow fields stay untouched.
	prevPublished, prevBasis := "", ""
	if job.PublishedAt != nil {
		prevPublished = *job.PublishedAt
	}
	if job.PublishedBasis != nil {
		prevBasis = *job.PublishedBasis
	}
	keepPublished, keepBasis := prevPublished, prevBasis
	if keepPublished == "" && published != "" {
		keepPublished, keepBasis = published, publishedBasis
	} else if keepPublished == published && keepBasis == "" && publishedBasis != "" {
		keepBasis = publishedBasis
	}
	_, err = s.DB.Exec(`
		UPDATE jobs SET description=?, description_status=?, published_at=?, published_basis=?
		WHERE id=?`,
		description, status, keepPublished, keepBasis, job.ID)
	if err != nil {
		return job.ID, fmt.Errorf("enrich: update job %d: %w", job.ID, err)
	}
	return job.ID, nil
}

// matchEnrichJob returns the unique reliable JSON-LD posting for job by URL,
// then by unique title+company. An ambiguous multi-posting page yields ok=false
// so another opening's date/description is never attached by guesswork.
func matchEnrichJob(jobs []model.JobInput, job *model.Job) (model.JobInput, bool) {
	var matches []model.JobInput
	for _, j := range jobs {
		if sameJobURL(j.OriginalURL, job.OriginalURL) {
			matches = append(matches, j)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	title := strings.TrimSpace(job.Title)
	company := strings.TrimSpace(job.Company)
	if title != "" {
		matches = matches[:0]
		for _, j := range jobs {
			if strings.TrimSpace(j.Title) != title {
				continue
			}
			if company != "" && strings.TrimSpace(j.Company) != company {
				continue
			}
			matches = append(matches, j)
		}
		if len(matches) == 1 {
			return matches[0], true
		}
	}
	return model.JobInput{}, false
}

var trackingJobParamRe = regexp.MustCompile(`(?i)^(utm_.*|ref|source|gh_src|fbclid|gclid)$`)

func sameJobURL(a, b string) bool {
	ca, cb := canonicalJobURL(a), canonicalJobURL(b)
	return ca != "" && ca == cb
}

func canonicalJobURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	u.Fragment = ""
	q := u.Query()
	for k := range q {
		if trackingJobParamRe.MatchString(k) {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	out := host + strings.TrimRight(u.Path, "/")
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}

// hostOf extracts the hostname from rawURL.
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Hostname()
}
