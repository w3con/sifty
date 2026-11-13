package background

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/source"
	"github.com/w3con/sifty/internal/store"
)

// expressionsForTest converts config.SearchExpression slices to store.SearchGroup slices,
// matching the production helper so test code can build queries the same way.
func expressionsForTest(exprs []config.SearchExpression) []store.SearchGroup {
	return expressionsToFTSGroups(exprs)
}

// matchScoreFor reads the match_score for the first job whose title equals title.
func matchScoreFor(t *testing.T, s *store.Store, title string) int {
	t.Helper()
	var score int
	err := s.DB.QueryRow("SELECT match_score FROM jobs WHERE title=?", title).Scan(&score)
	if err != nil {
		t.Fatalf("matchScoreFor %q: %v", title, err)
	}
	return score
}

// stubSource is a Source whose Discover returns a fixed set of Refs.
type stubSource struct {
	refs []source.Ref
}

func (s *stubSource) Discover(_ context.Context, _ string) ([]source.Ref, error) {
	return s.refs, nil
}
func (s *stubSource) Resolve(_ context.Context, ref source.Ref) (model.JobInput, error) {
	return ref.Partial, nil
}

func TestDefaultCollect(t *testing.T) {
	st := newTestStore(t)

	// Two postings: one with a description, one without (pending state).
	refs := []source.Ref{
		{
			URL: "https://jobs.example.com/jobs/sre-001",
			Partial: model.JobInput{
				Title: "Site Reliability Engineer", Company: "Example Corp",
				OriginalURL: "https://jobs.example.com/jobs/sre-001",
				Description: "Build and maintain reliable systems at scale.",
				DescriptionStatus: "available structured description",
				Source: "example",
			},
		},
		{
			URL: "https://jobs.example.com/jobs/pm-002",
			Partial: model.JobInput{
				Title: "Product Manager", Company: "Example Corp",
				OriginalURL: "https://jobs.example.com/jobs/pm-002",
				Description: "",
				DescriptionStatus: "pending detail fetch",
				Source: "example",
			},
		},
	}

	// Use a live httptest server so the site URL resolves; the stub source ignores it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	stub := &stubSource{refs: refs}
	reg := source.NewRegistry(stub) // fallback is the stub — routes all hosts to it

	trueVal := true
	sites := []config.Source{
		{Name: "example", URL: srv.URL, Enabled: &trueVal},
	}

	total, err := collectSources(context.Background(), st, sites, reg, "", nil)
	if err != nil {
		t.Fatalf("collectSources: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 upserted postings, got %d", total)
	}

	// Source name must be the config site name, not an internal label.
	jobs, err := allJobs(st)
	if err != nil {
		t.Fatalf("allJobs: %v", err)
	}
	for _, j := range jobs {
		if j.Source != "example" {
			t.Errorf("job %q has source %q, want %q", j.Title, j.Source, "example")
		}
	}

	// The pending-description posting must have a non-empty DescriptionStatus.
	for _, j := range jobs {
		if j.Title == "Product Manager" && j.DescriptionStatus == "" {
			t.Error("Product Manager posting has empty DescriptionStatus — expected honest pending state")
		}
	}

	// A duplicate URL must not produce a second record.
	total2, err := collectSources(context.Background(), st, sites, reg, "", nil)
	if err != nil {
		t.Fatalf("second collectSources: %v", err)
	}
	// Count distinct job rows — duplicates must not create new rows.
	var rowCount int
	if err := st.DB.QueryRow("SELECT count(*) FROM jobs").Scan(&rowCount); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if rowCount != 2 {
		t.Errorf("after second run: expected 2 job rows, got %d", rowCount)
	}
	_ = total2
}

// allJobs returns all JobInput records from the store via a lightweight direct query.
func allJobs(s *store.Store) ([]jobRow, error) {
	rows, err := s.DB.Query(
		"SELECT j.title, j.company, j.description_status, si.source FROM jobs j" +
			" LEFT JOIN sightings si ON si.job_id = j.id ORDER BY j.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []jobRow
	for rows.Next() {
		var j jobRow
		var src *string
		if err := rows.Scan(&j.Title, &j.Company, &j.DescriptionStatus, &src); err != nil {
			return nil, err
		}
		if src != nil {
			j.Source = *src
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

type jobRow struct {
	Title             string
	Company           string
	DescriptionStatus string
	Source            string
}

// titleFillSource discovers refs with no title and fills the title from a map on Resolve,
// so the post-resolve stop-word check is exercised; it counts Resolve calls.
type titleFillSource struct {
	refs     []source.Ref
	titles   map[string]string
	resolved int
}

func (s *titleFillSource) Discover(_ context.Context, _ string) ([]source.Ref, error) {
	return s.refs, nil
}
func (s *titleFillSource) Resolve(_ context.Context, ref source.Ref) (model.JobInput, error) {
	s.resolved++
	j := ref.Partial
	if t, ok := s.titles[ref.URL]; ok {
		j.Title = t
	}
	return j, nil
}

// TestCollectSkipsStopWordPostings checks that a posting whose title matches an active
// stop-word is not stored, whether the title is known at discovery or only after Resolve,
// and that removing the stop-word lets the next run collect it.
func TestCollectSkipsStopWordPostings(t *testing.T) {
	st := newTestStore(t)

	mk := func(url, title, company string) source.Ref {
		return source.Ref{URL: url, Partial: model.JobInput{
			Title: title, Company: company, OriginalURL: url,
			Description: "Description for " + company, DescriptionStatus: "available structured description",
			Source: "example",
		}}
	}
	refs := []source.Ref{
		mk("https://jobs.example.com/jobs/1", "Senior Risk Engineer", "Alpha"),
		mk("https://jobs.example.com/jobs/2", "Go Engineer", "Beta"),
		mk("https://jobs.example.com/jobs/3", "", "Gamma"), // title only known after Resolve
	}
	stub := &titleFillSource{refs: refs, titles: map[string]string{
		"https://jobs.example.com/jobs/3": "RISK ENGINEER Payments",
	}}
	reg := source.NewRegistry(stub)
	trueVal := true
	sites := []config.Source{{Name: "example", URL: "https://jobs.example.com", Enabled: &trueVal}}

	sw, err := st.AddStopWord("risk engineer")
	if err != nil {
		t.Fatalf("AddStopWord: %v", err)
	}

	total, err := collectSources(context.Background(), st, sites, reg, "", nil)
	if err != nil {
		t.Fatalf("collectSources: %v", err)
	}
	if total != 1 {
		t.Errorf("upserted = %d, want 1 (only the non-matching posting)", total)
	}
	var rowCount int
	if err := st.DB.QueryRow("SELECT count(*) FROM jobs").Scan(&rowCount); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("stored jobs = %d, want 1", rowCount)
	}
	// The discovery-titled match is skipped before Resolve: only jobs 2 and 3 are resolved.
	if stub.resolved != 2 {
		t.Errorf("Resolve calls = %d, want 2 (matching discovery title must skip Resolve)", stub.resolved)
	}

	// Removing the stop-word resumes collection of the skipped postings.
	if err := st.RemoveStopWord(sw.ID); err != nil {
		t.Fatalf("RemoveStopWord: %v", err)
	}
	if _, err := collectSources(context.Background(), st, sites, reg, "", nil); err != nil {
		t.Fatalf("second collectSources: %v", err)
	}
	if err := st.DB.QueryRow("SELECT count(*) FROM jobs").Scan(&rowCount); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if rowCount != 3 {
		t.Errorf("stored jobs after removal = %d, want 3", rowCount)
	}
}

// TestCollectSetsMatchScore verifies that collectSources writes match_score for collected roles
// when given a matchQuery, without triggering any LLM call. The test does not start an assessment
// worker, so any non-zero score is evidence that BM25 (not LLM) produced it.
func TestCollectSetsMatchScore(t *testing.T) {
	st := newTestStore(t)

	refs := []source.Ref{
		{
			URL: "https://jobs.example.com/jobs/cto-001",
			Partial: model.JobInput{
				Title: "Fractional CTO", Company: "StartupA",
				OriginalURL: "https://jobs.example.com/jobs/cto-001",
				Description: "We are looking for an experienced fractional CTO to lead our engineering organisation.",
				DescriptionStatus: "available structured description",
				Source: "example",
			},
		},
		{
			URL: "https://jobs.example.com/jobs/pm-002",
			Partial: model.JobInput{
				Title: "Product Manager", Company: "StartupB",
				OriginalURL: "https://jobs.example.com/jobs/pm-002",
				Description: "Define the product roadmap and work with the engineering team.",
				DescriptionStatus: "available structured description",
				Source: "example",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	stub := &stubSource{refs: refs}
	reg := source.NewRegistry(stub)
	trueVal := true
	sites := []config.Source{{Name: "example", URL: srv.URL, Enabled: &trueVal}}

	// A query that matches "Fractional CTO" but not "Product Manager".
	matchQuery := store.BuildFTSQuery([]string{"Fractional CTO", "CTO"})
	total, err := collectSources(context.Background(), st, sites, reg, matchQuery, nil)
	if err != nil {
		t.Fatalf("collectSources: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 upserted postings, got %d", total)
	}

	ctoScore := matchScoreFor(t, st, "Fractional CTO")
	pmScore := matchScoreFor(t, st, "Product Manager")

	t.Logf("match_score: Fractional CTO=%d, Product Manager=%d", ctoScore, pmScore)

	if ctoScore == 0 {
		t.Error("Fractional CTO should have a non-zero match_score after collection with matching query")
	}
	if ctoScore < pmScore {
		t.Errorf("Fractional CTO (%d) should outscore Product Manager (%d)", ctoScore, pmScore)
	}
}

// queuedCount returns the number of rows in assessment_queue.
func queuedCount(t *testing.T, s *store.Store) int {
	t.Helper()
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM assessment_queue").Scan(&n); err != nil {
		t.Fatalf("queuedCount: %v", err)
	}
	return n
}

// TestANDQueryNarrows verifies that an AND expression narrows the match set:
// a role must contain ALL terms to earn a non-zero score, while a role that contains
// only some of them scores zero. It also confirms that ASM-4 fit scores (match_score)
// are still produced across the catalogue when OR/AND expressions are used.
func TestANDQueryNarrows(t *testing.T) {
	st := newTestStore(t)

	refs := []source.Ref{
		{
			URL: "https://jobs.example.com/jobs/full-001",
			Partial: model.JobInput{
				Title:             "Blockchain CTO",
				Company:           "ChainCo",
				OriginalURL:       "https://jobs.example.com/jobs/full-001",
				Description:       "We need an experienced CTO with deep blockchain expertise to lead our protocol engineering team.",
				DescriptionStatus: "available structured description",
				Source:            "example",
			},
		},
		{
			URL: "https://jobs.example.com/jobs/partial-002",
			Partial: model.JobInput{
				Title:             "Blockchain Developer",
				Company:           "ChainTwo",
				OriginalURL:       "https://jobs.example.com/jobs/partial-002",
				Description:       "Build decentralised applications using Solidity and Ethereum.",
				DescriptionStatus: "available structured description",
				Source:            "example",
			},
		},
		{
			URL: "https://jobs.example.com/jobs/none-003",
			Partial: model.JobInput{
				Title:             "Frontend Engineer",
				Company:           "WebCo",
				OriginalURL:       "https://jobs.example.com/jobs/none-003",
				Description:       "Build user interfaces with React and TypeScript.",
				DescriptionStatus: "available structured description",
				Source:            "example",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	stub := &stubSource{refs: refs}
	reg := source.NewRegistry(stub)
	trueVal := true
	sites := []config.Source{{Name: "example", URL: srv.URL, Enabled: &trueVal}}

	// AND query: both "Blockchain" AND "CTO" must appear. Only "Blockchain CTO" matches both.
	andExprs := []config.SearchExpression{{And: []string{"Blockchain", "CTO"}}}
	andQuery := store.BuildFTSQueryExprs(expressionsForTest(andExprs))

	total, err := collectSources(context.Background(), st, sites, reg, andQuery, nil)
	if err != nil {
		t.Fatalf("collectSources: %v", err)
	}
	if total != 3 {
		t.Errorf("expected 3 collected postings, got %d", total)
	}

	ctoScore := matchScoreFor(t, st, "Blockchain CTO")
	devScore := matchScoreFor(t, st, "Blockchain Developer")
	feScore := matchScoreFor(t, st, "Frontend Engineer")

	t.Logf("AND query match_scores: Blockchain CTO=%d, Blockchain Developer=%d, Frontend Engineer=%d",
		ctoScore, devScore, feScore)

	if ctoScore == 0 {
		t.Error("Blockchain CTO should have a non-zero score for AND(Blockchain,CTO) query")
	}
	if devScore != 0 {
		t.Errorf("Blockchain Developer should score 0 for AND(Blockchain,CTO) query (CTO absent), got %d", devScore)
	}
	if feScore != 0 {
		t.Errorf("Frontend Engineer should score 0 for AND(Blockchain,CTO) query, got %d", feScore)
	}
	if ctoScore <= devScore {
		t.Errorf("AND query must narrow: Blockchain CTO (%d) should outscore Blockchain Developer (%d)", ctoScore, devScore)
	}

	// Now verify ASM-4 fit scores are produced with an OR expression (backward-compat path).
	st2 := newTestStore(t)
	orExprs := []config.SearchExpression{{Or: []string{"Blockchain", "CTO"}}}
	orQuery := store.BuildFTSQueryExprs(expressionsForTest(orExprs))
	if _, err := collectSources(context.Background(), st2, sites, reg, orQuery, nil); err != nil {
		t.Fatalf("OR collectSources: %v", err)
	}
	ctoOr := matchScoreFor(t, st2, "Blockchain CTO")
	devOr := matchScoreFor(t, st2, "Blockchain Developer")
	if ctoOr == 0 || devOr == 0 {
		t.Errorf("OR query: both Blockchain CTO (%d) and Blockchain Developer (%d) should have non-zero scores",
			ctoOr, devOr)
	}
}

// TestCollectDoesNotEnqueue confirms that collectSources no longer mass-enqueues roles,
// and that transitioning a role to preselected enqueues exactly that role.
func TestCollectDoesNotEnqueue(t *testing.T) {
	st := newTestStore(t)

	refs := []source.Ref{
		{
			URL: "https://jobs.example.com/jobs/cto-001",
			Partial: model.JobInput{
				Title: "Fractional CTO", Company: "StartupA",
				OriginalURL:       "https://jobs.example.com/jobs/cto-001",
				Description:       "Lead engineering as fractional CTO.",
				DescriptionStatus: "available structured description",
				Source:            "example",
			},
		},
		{
			URL: "https://jobs.example.com/jobs/pm-002",
			Partial: model.JobInput{
				Title: "Product Manager", Company: "StartupB",
				OriginalURL:       "https://jobs.example.com/jobs/pm-002",
				Description:       "Define the product roadmap.",
				DescriptionStatus: "available structured description",
				Source:            "example",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	stub := &stubSource{refs: refs}
	reg := source.NewRegistry(stub)
	trueVal := true
	sites := []config.Source{{Name: "example", URL: srv.URL, Enabled: &trueVal}}

	// Collection must not enqueue anything.
	if _, err := collectSources(context.Background(), st, sites, reg, "", nil); err != nil {
		t.Fatalf("collectSources: %v", err)
	}
	if n := queuedCount(t, st); n != 0 {
		t.Errorf("after collection: assessment_queue has %d rows, want 0", n)
	}

	// Transitioning any role to preselected must enqueue exactly that role.
	var ctoID int64
	if err := st.DB.QueryRow("SELECT id FROM jobs WHERE title='Fractional CTO'").Scan(&ctoID); err != nil {
		t.Fatalf("find cto job: %v", err)
	}
	if _, err := st.Status(ctoID, "preselected", false); err != nil {
		t.Fatalf("move to preselected: %v", err)
	}
	if n := queuedCount(t, st); n != 1 {
		t.Errorf("after preselect: assessment_queue has %d rows, want 1", n)
	}

	// The other role (Product Manager) must still not be queued.
	var pmID int64
	if err := st.DB.QueryRow("SELECT id FROM jobs WHERE title='Product Manager'").Scan(&pmID); err != nil {
		t.Fatalf("find pm job: %v", err)
	}
	var pmQueued int
	if err := st.DB.QueryRow("SELECT count(*) FROM assessment_queue WHERE job_id=?", pmID).Scan(&pmQueued); err != nil {
		t.Fatalf("check pm queue: %v", err)
	}
	if pmQueued != 0 {
		t.Errorf("Product Manager should not be in queue, got %d rows", pmQueued)
	}
}
