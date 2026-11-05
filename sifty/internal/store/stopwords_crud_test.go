package store_test

import (
	"testing"

	"github.com/w3con/sifty/internal/store"
)

func TestStopWordCRUD(t *testing.T) {
	s := openFresh(t)

	list, err := s.ListStopWords()
	if err != nil || len(list) != 0 {
		t.Fatalf("empty store: list=%v err=%v", list, err)
	}

	sw, err := s.AddStopWord("  Risk Engineer ")
	if err != nil {
		t.Fatalf("AddStopWord: %v", err)
	}
	if sw.Word != "risk engineer" || sw.ID == 0 {
		t.Errorf("stored word = %q id=%d, want normalised \"risk engineer\"", sw.Word, sw.ID)
	}

	dup, err := s.AddStopWord("RISK ENGINEER")
	if err != nil || dup.ID != sw.ID {
		t.Errorf("duplicate add: got %+v err=%v, want same id %d", dup, err, sw.ID)
	}

	if _, err := s.AddStopWord("   "); err == nil {
		t.Error("empty stop-word must be rejected")
	}

	list, _ = s.ListStopWords()
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	if err := s.RemoveStopWord(sw.ID); err != nil {
		t.Fatalf("RemoveStopWord: %v", err)
	}
	if err := s.RemoveStopWord(sw.ID); err == nil {
		t.Error("removing a missing stop-word must error")
	}
	list, _ = s.ListStopWords()
	if len(list) != 0 {
		t.Errorf("after remove len = %d, want 0", len(list))
	}
}

// stopTestJob inserts a role with the given title and moves it to the given stage.
func stopTestJob(t *testing.T, s *store.Store, title, urlSuffix, stage string) int64 {
	t.Helper()
	in := fixtureJobInput("test-source")
	in.Title = title
	apply := "https://example.com/apply/" + urlSuffix
	in.ApplyURL = &apply
	in.Company = "Company " + urlSuffix // distinct company so Upsert never merges the test roles
	in.Description = "Description for " + urlSuffix + ". " + in.Description
	in.OriginalURL = "https://jobs.example.com/postings/" + urlSuffix
	id, err := s.Upsert(in, false)
	if err != nil {
		t.Fatalf("Upsert %q: %v", title, err)
	}
	switch stage {
	case "preselected":
		if _, err := s.Status(id, "preselected", false); err != nil {
			t.Fatalf("Status preselected: %v", err)
		}
	case "applied":
		if _, err := s.Status(id, "preselected", false); err != nil {
			t.Fatalf("Status preselected: %v", err)
		}
		if _, err := s.Status(id, "applied", true); err != nil {
			t.Fatalf("Status applied: %v", err)
		}
	case "no":
		if _, err := s.Status(id, "no", false); err != nil {
			t.Fatalf("Status no: %v", err)
		}
	}
	return id
}

func listIDs(t *testing.T, s *store.Store, status string) map[int64]bool {
	t.Helper()
	jobs, err := s.List(status, "")
	if err != nil {
		t.Fatalf("List(%q): %v", status, err)
	}
	out := map[int64]bool{}
	for _, j := range jobs {
		out[j.ID] = true
	}
	return out
}

// TestStopWordHidesFromEveryListing checks that a stop-word computes Stopped/StoppedBy on Get,
// removes the role from every status listing, and that removing the word restores it.
func TestStopWordHidesFromEveryListing(t *testing.T) {
	s := openFresh(t)
	stages := []string{"triage", "preselected", "applied", "no"}
	stopped := map[string]int64{}
	for i, st := range stages {
		stopped[st] = stopTestJob(t, s, "Senior Risk Engineer "+st, "stop"+string(rune('a'+i)), st)
	}
	keep := stopTestJob(t, s, "Go Engineer", "keep", "triage")

	sw, err := s.AddStopWord("risk engineer")
	if err != nil {
		t.Fatalf("AddStopWord: %v", err)
	}

	for _, st := range stages {
		j, err := s.Get(stopped[st])
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !j.Stopped || j.StoppedBy != "risk engineer" {
			t.Errorf("%s: Stopped=%v StoppedBy=%q, want true/\"risk engineer\"", st, j.Stopped, j.StoppedBy)
		}
		ids := listIDs(t, s, st)
		if ids[stopped[st]] {
			t.Errorf("%s listing still contains the stopped role", st)
		}
	}
	all := listIDs(t, s, "all")
	for _, st := range stages {
		if all[stopped[st]] {
			t.Errorf("\"all\" listing contains stopped %s role", st)
		}
	}
	if !all[keep] {
		t.Error("non-matching role must remain listed")
	}
	if j, _ := s.Get(keep); j.Stopped {
		t.Error("non-matching role must not be stopped")
	}

	// Reversible: removing the stop-word restores every role; nothing was deleted.
	if err := s.RemoveStopWord(sw.ID); err != nil {
		t.Fatalf("RemoveStopWord: %v", err)
	}
	for _, st := range stages {
		if !listIDs(t, s, st)[stopped[st]] {
			t.Errorf("%s role did not return after the stop-word was removed", st)
		}
		if j, _ := s.Get(stopped[st]); j.Stopped || j.StoppedBy != "" {
			t.Errorf("%s role still flagged stopped after removal", st)
		}
	}
}

// TestStopWordMatchCount checks the preview count equals the number of roles actually hidden
// after the word is added, and that previewing creates no rule.
func TestStopWordMatchCount(t *testing.T) {
	s := openFresh(t)
	stopTestJob(t, s, "Senior Risk Engineer", "m1", "triage")
	stopTestJob(t, s, "RISK ENGINEER, Payments", "m2", "preselected")
	stopTestJob(t, s, "Go Engineer", "m3", "triage")

	n, err := s.StopWordMatchCount("Risk Engineer")
	if err != nil {
		t.Fatalf("StopWordMatchCount: %v", err)
	}
	if n != 2 {
		t.Errorf("preview count = %d, want 2", n)
	}
	if list, _ := s.ListStopWords(); len(list) != 0 {
		t.Errorf("preview created %d rules, want 0", len(list))
	}
	if n, _ := s.StopWordMatchCount("  "); n != 0 {
		t.Errorf("empty word count = %d, want 0", n)
	}

	before, _ := s.List("all", "")
	if _, err := s.AddStopWord("risk engineer"); err != nil {
		t.Fatalf("AddStopWord: %v", err)
	}
	after, _ := s.List("all", "")
	if hidden := len(before) - len(after); hidden != n {
		t.Errorf("hidden after add = %d, preview said %d", hidden, n)
	}
	// Once the word is active, previewing it again hides nothing new.
	if again, _ := s.StopWordMatchCount("risk engineer"); again != 0 {
		t.Errorf("count after add = %d, want 0", again)
	}
}
