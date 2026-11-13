package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

const (
	testExtractor = "vacancyai.extract/test"
	testMatcher   = "vacancyai.match/test"
)

func modelJob(t *testing.T) (*store.Store, int64, string) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "model.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	id, err := s.Upsert(model.JobInput{
		Title: "Backend Engineer", Company: "Acme", OriginalURL: "https://example.com/jobs/9",
		Description: "You know Go. You have 5 years of experience.", DescriptionStatus: "available",
		Source: "Wellfound", Location: "Berlin, Germany",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	src, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("CaptureMatchInput: %v", err)
	}
	return s, id, matchcontract.HashDescription(src.Description)
}

func extractionFor(id int64, hash, version string, texts ...string) matchcontract.VacancyExtraction {
	e := matchcontract.VacancyExtraction{
		SchemaVersion: matchcontract.SchemaVersion, ExtractorVersion: version, JobID: id, SourceHash: hash,
		Requirements: []matchcontract.ExtractedRequirement{},
	}
	for _, text := range texts {
		e.Requirements = append(e.Requirements, matchcontract.ExtractedRequirement{
			Text: text, Kind: matchcontract.RequirementSkill, Level: matchcontract.ImportanceRequired, Quote: "You know Go.",
		})
	}
	return e
}

func matchesFor(t *testing.T, e matchcontract.VacancyExtraction, revision int, version string) matchcontract.RequirementMatches {
	t.Helper()
	hash, err := e.RequirementsHash()
	if err != nil {
		t.Fatalf("RequirementsHash: %v", err)
	}
	m := matchcontract.RequirementMatches{
		SchemaVersion: matchcontract.SchemaVersion, MatcherVersion: version, JobID: e.JobID,
		RequirementsHash: hash, ProfileRevision: revision,
	}
	for i := range e.Requirements {
		m.Verdicts = append(m.Verdicts, matchcontract.RequirementVerdict{
			Index: i, Verdict: matchcontract.VerdictMatch,
			Evidence: &matchcontract.ProfileEvidence{Field: "skills", Value: "Go (expert)"},
		})
	}
	return m
}

func TestModelPassTablesAreAdditive(t *testing.T) {
	s, _, _ := modelJob(t)
	var version int
	if err := s.DB.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 22 {
		t.Fatalf("user_version = %d (%v), want 22", version, err)
	}
	for _, table := range []string{"match_vacancy_extractions", "match_requirement_matches"} {
		var n int
		if err := s.DB.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("table %s missing or not empty: n=%d err=%v", table, n, err)
		}
	}
}

func TestExtractionIsKeyedBySourceAndExtractorVersion(t *testing.T) {
	s, id, hash := modelJob(t)
	e := extractionFor(id, hash, testExtractor, "Go")
	if err := s.SaveVacancyExtraction(e); err != nil {
		t.Fatalf("SaveVacancyExtraction: %v", err)
	}
	got, err := s.GetVacancyExtraction(id, hash, testExtractor)
	if err != nil || got == nil || len(got.Requirements) != 1 || got.Requirements[0].Text != "Go" {
		t.Fatalf("stored extraction = %+v err=%v", got, err)
	}
	// A different extractor version or a different source hash is a different key: the pass must run.
	if other, err := s.GetVacancyExtraction(id, hash, "vacancyai.extract/other"); err != nil || other != nil {
		t.Fatalf("another extractor version must not find this record: %+v err=%v", other, err)
	}
	if other, err := s.GetVacancyExtraction(id, "deadbeef", testExtractor); err != nil || other != nil {
		t.Fatalf("another source hash must not find this record: %+v err=%v", other, err)
	}
}

func TestExtractionThatFoundNothingIsAFinalAnswer(t *testing.T) {
	s, id, hash := modelJob(t)
	if err := s.SaveVacancyExtraction(extractionFor(id, hash, testExtractor)); err != nil {
		t.Fatalf("SaveVacancyExtraction without requirements: %v", err)
	}
	got, err := s.GetVacancyExtraction(id, hash, testExtractor)
	if err != nil || got == nil || len(got.Requirements) != 0 {
		t.Fatalf("an extraction with no requirements must be stored and returned, not treated as missing: %+v err=%v", got, err)
	}
}

func TestExtractionForAChangedSourceIsRefusedAndASourceChangeSupersedes(t *testing.T) {
	s, id, hash := modelJob(t)
	if err := s.SaveVacancyExtraction(extractionFor(id, hash, testExtractor, "Go")); err != nil {
		t.Fatalf("SaveVacancyExtraction: %v", err)
	}
	if _, err := s.DB.Exec("UPDATE jobs SET description=? WHERE id=?", "You know Rust.", id); err != nil {
		t.Fatalf("update source: %v", err)
	}
	err := s.SaveVacancyExtraction(extractionFor(id, hash, testExtractor, "Rust"))
	if !errors.Is(err, store.ErrMatchSourceChanged) {
		t.Fatalf("a write for the old source: error = %v, want ErrMatchSourceChanged", err)
	}
	src, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("CaptureMatchInput: %v", err)
	}
	newHash := matchcontract.HashDescription(src.Description)
	if newHash == hash {
		t.Fatal("the source hash did not change with the description")
	}
	if cur, err := s.GetVacancyExtraction(id, newHash, testExtractor); err != nil || cur != nil {
		t.Fatalf("the new source has no extraction yet, so the pass must run: %+v err=%v", cur, err)
	}
	if err := s.SaveVacancyExtraction(extractionFor(id, newHash, testExtractor, "Rust")); err != nil {
		t.Fatalf("SaveVacancyExtraction for the new source: %v", err)
	}
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM match_vacancy_extractions WHERE job_id=?", id).Scan(&n); err != nil || n != 1 {
		t.Fatalf("a job keeps one extraction, the one for its current source: n=%d err=%v", n, err)
	}
}

func TestNewSightingOfTheSameTextKeepsTheExtraction(t *testing.T) {
	s, id, hash := modelJob(t)
	if err := s.SaveVacancyExtraction(extractionFor(id, hash, testExtractor, "Go")); err != nil {
		t.Fatalf("SaveVacancyExtraction: %v", err)
	}
	// A later collection sees the same text on another source, with another location and tags.
	if _, err := s.Upsert(model.JobInput{
		Title: "Backend Engineer", Company: "Acme", OriginalURL: "https://example.com/jobs/9",
		Description: "You know Go. You have 5 years of experience.", DescriptionStatus: "available",
		Source: "Remotive", Location: "Paris, France", Tags: []string{"go", "backend"},
	}, false); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	src, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("CaptureMatchInput: %v", err)
	}
	if got := matchcontract.HashDescription(src.Description); got != hash {
		t.Fatalf("description hash changed without a change of text")
	}
	if cur, err := s.GetVacancyExtraction(id, hash, testExtractor); err != nil || cur == nil {
		t.Fatalf("the extraction must survive a new sighting of the same text: %+v err=%v", cur, err)
	}
}

func TestNewProfileRevisionReRunsOnlyMatching(t *testing.T) {
	s, id, hash := modelJob(t)
	e := extractionFor(id, hash, testExtractor, "Go", "Kubernetes")
	if err := s.SaveVacancyExtraction(e); err != nil {
		t.Fatalf("SaveVacancyExtraction: %v", err)
	}
	reqHash, _ := e.RequirementsHash()
	if err := s.SaveRequirementMatches(matchesFor(t, e, 3, testMatcher)); err != nil {
		t.Fatalf("SaveRequirementMatches: %v", err)
	}
	if m, err := s.GetRequirementMatches(id, reqHash, 3, testMatcher); err != nil || m == nil || len(m.Verdicts) != 2 {
		t.Fatalf("matches for revision 3 = %+v err=%v", m, err)
	}
	// The profile changed: no matches for revision 4, but the extraction is still current.
	if m, err := s.GetRequirementMatches(id, reqHash, 4, testMatcher); err != nil || m != nil {
		t.Fatalf("a new profile revision must not find old matches: %+v err=%v", m, err)
	}
	if got, err := s.GetVacancyExtraction(id, hash, testExtractor); err != nil || got == nil {
		t.Fatalf("a profile change must leave the extraction in place: %+v err=%v", got, err)
	}
	if err := s.SaveRequirementMatches(matchesFor(t, e, 4, testMatcher)); err != nil {
		t.Fatalf("SaveRequirementMatches revision 4: %v", err)
	}
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM match_requirement_matches WHERE job_id=?", id).Scan(&n); err != nil || n != 1 {
		t.Fatalf("the older revision is superseded: n=%d err=%v", n, err)
	}
	if m, err := s.GetRequirementMatches(id, reqHash, 4, "vacancyai.match/other"); err != nil || m != nil {
		t.Fatalf("a different matcher version is a different key: %+v err=%v", m, err)
	}
}

func TestRequirementMatchesAreValidatedAgainstTheStoredExtraction(t *testing.T) {
	s, id, hash := modelJob(t)
	e := extractionFor(id, hash, testExtractor, "Go")
	if err := s.SaveVacancyExtraction(e); err != nil {
		t.Fatalf("SaveVacancyExtraction: %v", err)
	}

	foreign := matchesFor(t, extractionFor(id, hash, testExtractor, "Rust"), 1, testMatcher)
	if err := s.SaveRequirementMatches(foreign); !errors.Is(err, store.ErrMatchExtractionMissing) {
		t.Fatalf("matches for a requirement list nobody extracted: error = %v, want ErrMatchExtractionMissing", err)
	}
	unsupported := matchesFor(t, e, 1, testMatcher)
	unsupported.Verdicts[0].Evidence = nil
	if err := s.SaveRequirementMatches(unsupported); err == nil {
		t.Fatal("a match verdict without profile evidence was stored")
	}
	if _, err := s.DB.Exec("UPDATE jobs SET description=? WHERE id=?", "You know Rust.", id); err != nil {
		t.Fatalf("update source: %v", err)
	}
	if err := s.SaveRequirementMatches(matchesFor(t, e, 1, testMatcher)); !errors.Is(err, store.ErrMatchSourceChanged) {
		t.Fatalf("matches for an extraction of an old source: error = %v, want ErrMatchSourceChanged", err)
	}
}

func TestModelPassRowsGoWithTheirJob(t *testing.T) {
	s, id, hash := modelJob(t)
	e := extractionFor(id, hash, testExtractor, "Go")
	if err := s.SaveVacancyExtraction(e); err != nil {
		t.Fatalf("SaveVacancyExtraction: %v", err)
	}
	if err := s.SaveRequirementMatches(matchesFor(t, e, 1, testMatcher)); err != nil {
		t.Fatalf("SaveRequirementMatches: %v", err)
	}
	if _, err := s.DB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("foreign keys: %v", err)
	}
	// Older tables carry no cascade of their own; clear them first, as the application's delete does.
	for _, table := range []string{"sightings", "snapshots", "assessments"} {
		if _, err := s.DB.Exec("DELETE FROM "+table+" WHERE job_id=?", id); err != nil {
			t.Fatalf("delete from %s: %v", table, err)
		}
	}
	if _, err := s.DB.Exec("DELETE FROM jobs WHERE id=?", id); err != nil {
		t.Fatalf("delete job: %v", err)
	}
	for _, table := range []string{"match_vacancy_extractions", "match_requirement_matches"} {
		var n int
		if err := s.DB.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s kept rows of a deleted job: n=%d err=%v", table, n, err)
		}
	}
}

func TestFitResultTableIsAdditiveAndRoundTrips(t *testing.T) {
	s, id, hash := modelJob(t)
	if got, err := s.GetFitResult(id); err != nil || got != nil {
		t.Fatalf("a fresh job has no fit result: %+v err=%v", got, err)
	}
	score := 61
	raw := 0.61
	r := matchcontract.FitResult{
		Source:  matchcontract.SourceUsable,
		FitsYou: matchcontract.FitsYou{Status: matchcontract.ScoreScored, Score: &score, RawScore: &raw, Groups: []matchcontract.GroupFit{}},
		YouFit:  matchcontract.YouFit{Status: matchcontract.ScorePending},
		Lists: matchcontract.FitLists{Matches: []matchcontract.FitLine{}, Mismatches: []matchcontract.FitLine{},
			Missing: []matchcontract.FitLine{}, ToClarify: []matchcontract.FitLine{}},
		Provenance: matchcontract.FitProvenance{SchemaVersion: matchcontract.SchemaVersion, ProfileRevision: 3, RulesVersion: "r"},
	}
	if err := s.SaveFitResult(id, r, store.FitKey{RulesIdentity: "rules-1", DescriptionHash: hash}); err != nil {
		t.Fatalf("SaveFitResult: %v", err)
	}
	got, err := s.GetFitResult(id)
	if err != nil || got == nil || got.ProfileRevision != 3 || got.Key.DescriptionHash != hash || got.Result.FitsYou.Score == nil || *got.Result.FitsYou.Score != 61 {
		t.Fatalf("round trip = %+v err=%v", got, err)
	}
	r.YouFit = matchcontract.YouFit{Status: matchcontract.ScoreNoRequirements}
	if err := s.SaveFitResult(id, r, store.FitKey{RulesIdentity: "rules-2", DescriptionHash: hash}); err != nil {
		t.Fatalf("second save: %v", err)
	}
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM match_fit_results WHERE job_id=?", id).Scan(&n); err != nil || n != 1 {
		t.Fatalf("a job keeps one fit result: n=%d err=%v", n, err)
	}
	// A result that breaks the score invariants is refused, not stored.
	bad := r
	bad.FitsYou = matchcontract.FitsYou{Status: matchcontract.ScoreScored}
	if err := s.SaveFitResult(id, bad, store.FitKey{}); err == nil {
		t.Fatal("a scored result without a number must be refused")
	}
}
