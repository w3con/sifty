package server_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/fetch"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/server"
)

// fixedTransport returns a fixed status, body and Content-Type for every request.
type fixedTransport struct {
	status int
	body   string
	ct     string
}

func (f *fixedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	code := f.status
	if code == 0 {
		code = 200
	}
	return &http.Response{
		StatusCode: code,
		Header:     http.Header{"Content-Type": []string{f.ct}},
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Request:    req,
	}, nil
}

const detailPageHTML = `<!DOCTYPE html>
<html><head><title>Senior Go Engineer at TechCo</title></head>
<body>
<script type="application/ld+json">
{
  "@context":"https://schema.org",
  "@type":"JobPosting",
  "title":"Senior Go Engineer",
  "url":"https://jobs.techco.example/go-engineer",
  "hiringOrganization":{"@type":"Organization","name":"TechCo"},
  "datePosted":"2026-10-01",
  "description":"Build distributed systems in Go. Work on the platform team.",
  "jobLocationType":"TELECOMMUTE"
}
</script>
</body></html>`

func TestImportURLFetch(t *testing.T) {
	s := newTestStore(t)

	c, err := fetch.New(fetch.Options{
		AllowPrivate: true,
		Transport: &fixedTransport{
			body: detailPageHTML,
			ct:   "text/html",
		},
	})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}

	id, err := server.ImportURL(s, c, "https://jobs.techco.example/go-engineer")
	if err != nil {
		t.Fatalf("ImportURL: %v", err)
	}
	if id == 0 {
		t.Fatal("ImportURL returned id=0")
	}

	// Verify the stored posting has real content, not the "Manual import" placeholder.
	job, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get(%d): %v", id, err)
	}
	if job.Title == "Manual import" || job.Title == "" {
		t.Errorf("Title is %q — expected parsed title from detail page", job.Title)
	}
	if !strings.Contains(job.Description, "distributed systems in Go") {
		t.Errorf("Description missing expected phrase; got: %q", job.Description)
	}
}

func TestEnrichFn(t *testing.T) {
	s := newTestStore(t)

	// Seed a job with a pending description.
	seedID, err := s.Upsert(model.JobInput{
		Title: "Go Engineer", Company: "TechCo",
		OriginalURL: "https://jobs.techco.example/go-engineer",
		Source:      "test", DescriptionStatus: "pending detail fetch",
	}, false)
	if err != nil {
		t.Fatalf("Upsert seed: %v", err)
	}

	c, err := fetch.New(fetch.Options{
		AllowPrivate: true,
		Transport: &fixedTransport{
			body: detailPageHTML,
			ct:   "text/html",
		},
	})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}

	returnedID, err := server.EnrichJob(s, c, seedID)
	if err != nil {
		t.Fatalf("EnrichJob: %v", err)
	}
	if returnedID != seedID {
		t.Errorf("EnrichJob returned id %d, want %d", returnedID, seedID)
	}

	// The job's description must now contain text from the detail page.
	job, err := s.Get(seedID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(job.Description, "distributed systems in Go") {
		t.Errorf("Description not updated; got: %q", job.Description)
	}
	if job.DescriptionStatus == "pending detail fetch" {
		t.Error("DescriptionStatus still 'pending detail fetch' after enrich")
	}
	if job.PublishedAt == nil || *job.PublishedAt != "2026-10-01" {
		t.Errorf("missing publication date not filled from datePosted: %v", job.PublishedAt)
	}
	if job.PublishedBasis == nil || *job.PublishedBasis != "datePosted" {
		t.Errorf("publication basis not attached atomically: %v", job.PublishedBasis)
	}
	if job.Stage != "triage" || job.Status != "new" {
		t.Errorf("enrich changed workflow: stage=%q status=%q", job.Stage, job.Status)
	}
}

// TestImportURLRoute tests the POST /api/import route end-to-end using an injected FetchClient.
func TestImportURLRoute(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	c, _ := fetch.New(fetch.Options{
		AllowPrivate: true,
		Transport:    &fixedTransport{body: detailPageHTML, ct: "text/html"},
	})

	deps := &server.Deps{
		Store:       s,
		Dirs:        d,
		Cfg:         config.NewHolder(cfg),
		FetchClient: c,
	}
	mux := server.New(testPort, func(m *http.ServeMux) {
		server.RegisterRoutes(m, deps)
	})
	h := server.Handler(testPort, mux)

	resp := apiDo(t, h, "POST", "/api/import", `{"url":"https://jobs.techco.example/go-engineer"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("POST /api/import: status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"id"`) {
		t.Errorf("response has no id field: %s", body)
	}
}
