package store_test

import (
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

func openFresh(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func fixtureJobInput(source string) model.JobInput {
	applyURL := "https://example.com/apply/123"
	return model.JobInput{
		Title:             "Senior Go Engineer",
		Company:           "Acme Corp",
		Tags:              []string{"go", "backend"},
		OriginalURL:       "https://jobs.example.com/postings/123",
		ApplyURL:          &applyURL,
		ApplyStatus:       "direct",
		Description:       "We are looking for a senior Go engineer with 5+ years of experience building distributed systems and microservices.",
		DescriptionStatus: "full",
		Location:          "Remote",
		PublishedAt:       "2026-11-01T10:00:00.000Z",
		Source:            source,
	}
}

// TestJobRoundTrip verifies that a job inserted with Upsert comes back intact via Get.
func TestJobRoundTrip(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("test-source")

	id, err := s.Upsert(j, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if id == 0 {
		t.Fatal("Upsert returned 0 id for a non-deleted job")
	}

	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Verify the documented shape fields.
	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.Title != j.Title {
		t.Errorf("Title: got %q, want %q", got.Title, j.Title)
	}
	if got.Company != j.Company {
		t.Errorf("Company: got %q, want %q", got.Company, j.Company)
	}
	if got.Stage != "triage" {
		t.Errorf("Stage: got %q, want triage (new job)", got.Stage)
	}
	if got.Status != "new" {
		t.Errorf("Status: got %q, want new", got.Status)
	}
	if got.Seen {
		t.Error("Seen: new job should not be seen")
	}
	if got.SeenAt != nil {
		t.Errorf("SeenAt: want nil, got %q", *got.SeenAt)
	}
	if got.Description != j.Description {
		t.Errorf("Description: got %q…, want %q…", got.Description[:20], j.Description[:20])
	}
	if got.ApplyURL == nil || *got.ApplyURL == "" {
		t.Error("ApplyURL: want non-empty")
	}
	if got.ApplyStatus != j.ApplyStatus {
		t.Errorf("ApplyStatus: got %q, want %q", got.ApplyStatus, j.ApplyStatus)
	}
	if len(got.Tags) != len(j.Tags) {
		t.Errorf("Tags: got %v, want %v", got.Tags, j.Tags)
	}
	if len(got.Sources) == 0 {
		t.Error("Sources: want at least one sighting")
	} else {
		if got.Sources[0].Name != j.Source {
			t.Errorf("Sources[0].Name: got %q, want %q", got.Sources[0].Name, j.Source)
		}
	}
	if got.FirstSeen == "" {
		t.Error("FirstSeen: must not be empty")
	}
	if got.LastSeen == "" {
		t.Error("LastSeen: must not be empty")
	}
	// Assessment is nil for a brand-new role.
	if got.Assessment != nil {
		t.Errorf("Assessment: want nil for new role, got %v", got.Assessment)
	}
}

// TestJobList verifies that List returns all inserted jobs in score-sorted order
// and filters correctly by status and query text.
func TestJobList(t *testing.T) {
	s := openFresh(t)

	applyURL := "https://b.example.com/apply"
	jobs := []model.JobInput{
		fixtureJobInput("src"),
		{
			Title:             "Product Manager",
			Company:           "WidgetCo",
			OriginalURL:       "https://widgetco.com/jobs/pm",
			ApplyURL:          &applyURL,
			Description:       "Lead product strategy for our growing platform. Work closely with engineering and design teams.",
			DescriptionStatus: "full",
			Source:            "src",
		},
	}

	var ids []int64
	for _, j := range jobs {
		id, err := s.Upsert(j, false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		ids = append(ids, id)
	}

	all, err := s.List("", "")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List all: got %d jobs, want 2", len(all))
	}

	// Status filter: 'new' resolves to triage.
	triage, err := s.List("new", "")
	if err != nil {
		t.Fatalf("List new: %v", err)
	}
	if len(triage) != 2 {
		t.Errorf("List new: got %d, want 2", len(triage))
	}

	// Query filter.
	filtered, err := s.List("", "product manager")
	if err != nil {
		t.Fatalf("List with query: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("List query 'product manager': got %d, want 1", len(filtered))
	}

	// Status transition: move first job to preselected (saved).
	updated, err := s.Status(ids[0], "saved", false)
	if err != nil {
		t.Fatalf("Status to saved: %v", err)
	}
	if updated.Stage != "preselected" {
		t.Errorf("after Status(saved): Stage = %q, want preselected", updated.Stage)
	}

	saved, err := s.List("saved", "")
	if err != nil {
		t.Fatalf("List saved: %v", err)
	}
	if len(saved) != 1 {
		t.Errorf("List saved: got %d, want 1", len(saved))
	}
}

// TestJobSeen verifies that Seen sets seen_at and subsequent calls are no-ops.
func TestJobSeen(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, err := s.Upsert(j, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	after, err := s.Seen(id)
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if !after.Seen {
		t.Error("Seen: job should be marked seen after Seen()")
	}
	if after.SeenAt == nil {
		t.Error("SeenAt: should be set after Seen()")
	}
	firstSeenAt := *after.SeenAt

	// Second call must not change seen_at.
	after2, err := s.Seen(id)
	if err != nil {
		t.Fatalf("Seen (2nd): %v", err)
	}
	if *after2.SeenAt != firstSeenAt {
		t.Errorf("SeenAt changed on 2nd Seen call: %q → %q", firstSeenAt, *after2.SeenAt)
	}
}

// TestJobStatusTransitions verifies workflow guards and that applied requires confirmation.
func TestJobStatusTransitions(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	// Move to saved (preselected).
	saved, err := s.Status(id, "saved", false)
	if err != nil {
		t.Fatalf("→ saved: %v", err)
	}
	if saved.Stage != "preselected" {
		t.Errorf("→ saved: stage = %q, want preselected", saved.Stage)
	}

	// Applied without confirmation must fail.
	if _, err := s.Status(id, "applied", false); err == nil {
		t.Error("→ applied without confirmed: want error")
	}

	// Applied with confirmation must succeed.
	applied, err := s.Status(id, "applied", true)
	if err != nil {
		t.Fatalf("→ applied with confirmed: %v", err)
	}
	if applied.Stage != "applied" {
		t.Errorf("→ applied: stage = %q, want applied", applied.Stage)
	}
	if applied.AppliedAt == nil {
		t.Error("applied_at must be set after moving to applied")
	}
}

// TestJobGuardedDelete verifies that a triage role cannot be deleted directly.
func TestJobGuardedDelete(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	// Delete from triage must fail.
	if _, _, err := s.Delete(id); err == nil {
		t.Error("Delete from triage: want error")
	}

	// Move to 'no' (rejected), then delete.
	if _, err := s.Status(id, "rejected", false); err != nil {
		t.Fatalf("→ rejected: %v", err)
	}
	deleted, _, err := s.Delete(id)
	if err != nil {
		t.Fatalf("Delete from rejected: %v", err)
	}
	if deleted != id {
		t.Errorf("Delete returned id %d, want %d", deleted, id)
	}

	// Role should now be gone.
	if _, err := s.Get(id); err == nil {
		t.Error("Get after delete: want error")
	}
}

// TestJobDeduplication verifies that upserting the same canonical URL twice returns the same id
// (a second sighting is not added because the URL conflict updates `seen` in place, matching
// the TypeScript ON CONFLICT(url) DO UPDATE SET seen=excluded.seen behaviour).
func TestJobDeduplication(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src-a")
	id1, err := s.Upsert(j, false)
	if err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Same URL, different source — same id, sighting updated in place.
	j2 := j
	j2.Source = "src-b"
	id2, err := s.Upsert(j2, false)
	if err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if id1 != id2 {
		t.Errorf("Upsert returned different ids for same URL: %d vs %d", id1, id2)
	}

	// One sighting: the url is unique, second upsert updates seen_at in place.
	job, err := s.Get(id1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(job.Sources) != 1 {
		t.Errorf("Sources: want 1 sighting after two same-URL upserts, got %d", len(job.Sources))
	}

	// A different board listing for the same apply_url deduplicates by the second condition
	// (apply_url match). The identity guard with different apply_urls on both sides blocks the
	// third condition, so we use the same apply_url here.
	sameApply := "https://example.com/apply/123"
	j3 := model.JobInput{
		Title:             j.Title,
		Company:           j.Company,
		OriginalURL:       "https://boards2.example.com/job/senior-456",
		ApplyURL:          &sameApply,
		Description:       j.Description,
		DescriptionStatus: j.DescriptionStatus,
		Source:            "src-c",
	}
	id3, err := s.Upsert(j3, false)
	if err != nil {
		t.Fatalf("third Upsert (different board, same apply URL): %v", err)
	}
	if id3 != id1 {
		t.Errorf("same apply URL: want id %d, got %d", id1, id3)
	}
	job2, _ := s.Get(id1)
	if len(job2.Sources) != 2 {
		t.Errorf("after different-board same-apply upsert: want 2 sightings, got %d", len(job2.Sources))
	}
}
