package store_test

import (
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

// seqBefore reads the current catalogue sequence counter.
func seqBefore(t *testing.T, s *store.Store) int64 {
	t.Helper()
	v, err := s.CatalogueSeq()
	if err != nil {
		t.Fatalf("CatalogueSeq: %v", err)
	}
	return v
}

// assertBumped verifies the counter increased by exactly one after an operation.
func assertBumped(t *testing.T, s *store.Store, before int64, op string) {
	t.Helper()
	after, err := s.CatalogueSeq()
	if err != nil {
		t.Fatalf("%s: CatalogueSeq after: %v", op, err)
	}
	if after != before+1 {
		t.Errorf("%s: catalogueSeq = %d, want %d", op, after, before+1)
	}
}

// seqJob returns a minimal JobInput for sequence tests.
func seqJob(source string) model.JobInput {
	return model.JobInput{
		Title: "Seq Test Engineer", Company: "SeqCo",
		OriginalURL:       "https://jobs.example.com/seq/1",
		Description:       "Build and maintain distributed systems.",
		DescriptionStatus: "full",
		Source:            source,
		Location:          "Remote",
	}
}

// TestCatalogueSeqBumpsOnWrite verifies every write path that changes the jobs list
// increments the in-memory counter by exactly one.
func TestCatalogueSeqBumpsOnWrite(t *testing.T) {
	t.Run("Upsert new vacancy", func(t *testing.T) {
		s := openFresh(t)
		before := seqBefore(t, s)
		if _, err := s.Upsert(seqJob("src"), false); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		assertBumped(t, s, before, "Upsert new")
	})

	t.Run("Upsert re-upsert", func(t *testing.T) {
		s := openFresh(t)
		j := seqJob("src")
		if _, err := s.Upsert(j, false); err != nil {
			t.Fatalf("first Upsert: %v", err)
		}
		before := seqBefore(t, s)
		j.Description = "Updated description."
		if _, err := s.Upsert(j, false); err != nil {
			t.Fatalf("re-Upsert: %v", err)
		}
		assertBumped(t, s, before, "re-Upsert")
	})

	t.Run("Seen", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		before := seqBefore(t, s)
		if _, err := s.Seen(id); err != nil {
			t.Fatalf("Seen: %v", err)
		}
		assertBumped(t, s, before, "Seen")
	})

	t.Run("StatusWithReason", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		before := seqBefore(t, s)
		if _, err := s.StatusWithReason(id, "preselected", false, ""); err != nil {
			t.Fatalf("StatusWithReason: %v", err)
		}
		assertBumped(t, s, before, "StatusWithReason")
	})

	t.Run("Delete", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		// Delete is only allowed outside Triage; move to Preselected first.
		if _, err := s.StatusWithReason(id, "preselected", false, ""); err != nil {
			t.Fatalf("StatusWithReason (setup): %v", err)
		}
		before := seqBefore(t, s)
		if _, _, err := s.Delete(id); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		assertBumped(t, s, before, "Delete")
	})

	t.Run("Assessment", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		before := seqBefore(t, s)
		if err := s.Assessment(id, map[string]any{"score": 85}); err != nil {
			t.Fatalf("Assessment: %v", err)
		}
		assertBumped(t, s, before, "Assessment")
	})

	t.Run("AssessmentFailure", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		before := seqBefore(t, s)
		if err := s.AssessmentFailure(id, "provider timeout", nil); err != nil {
			t.Fatalf("AssessmentFailure: %v", err)
		}
		assertBumped(t, s, before, "AssessmentFailure")
	})

	t.Run("SaveFitResult", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		score := 72
		raw := 0.72
		r := matchcontract.FitResult{
			Source:  matchcontract.SourceUsable,
			FitsYou: matchcontract.FitsYou{Status: matchcontract.ScoreScored, Score: &score, RawScore: &raw, Groups: []matchcontract.GroupFit{}},
			YouFit:  matchcontract.YouFit{Status: matchcontract.ScorePending},
			Lists: matchcontract.FitLists{
				Matches: []matchcontract.FitLine{}, Mismatches: []matchcontract.FitLine{},
				Missing: []matchcontract.FitLine{}, ToClarify: []matchcontract.FitLine{},
			},
			Provenance: matchcontract.FitProvenance{
				SchemaVersion: matchcontract.SchemaVersion, ProfileRevision: 1, RulesVersion: "r1",
			},
		}
		before := seqBefore(t, s)
		if err := s.SaveFitResult(id, r, store.FitKey{RulesIdentity: "rules-seq", DescriptionHash: "h1"}); err != nil {
			t.Fatalf("SaveFitResult: %v", err)
		}
		assertBumped(t, s, before, "SaveFitResult")
	})

	t.Run("RescoreForProfile empty query resets scores", func(t *testing.T) {
		s := openFresh(t)
		if _, err := s.Upsert(seqJob("src"), false); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		before := seqBefore(t, s)
		if err := s.RescoreForProfile("", 0); err != nil {
			t.Fatalf("RescoreForProfile: %v", err)
		}
		assertBumped(t, s, before, "RescoreForProfile")
	})

	t.Run("StoreRefetchResult", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		before := seqBefore(t, s)
		update := seqJob("src")
		update.Description = "Refreshed description with more detail."
		update.DescriptionStatus = "full"
		if err := s.StoreRefetchResult(id, update); err != nil {
			t.Fatalf("StoreRefetchResult: %v", err)
		}
		assertBumped(t, s, before, "StoreRefetchResult")
	})

	t.Run("BlockJob", func(t *testing.T) {
		s := openFresh(t)
		id, err := s.Upsert(seqJob("src"), false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		before := seqBefore(t, s)
		if _, err := s.BlockJob(id, "company"); err != nil {
			t.Fatalf("BlockJob: %v", err)
		}
		assertBumped(t, s, before, "BlockJob")
	})

	t.Run("AddStopWord", func(t *testing.T) {
		s := openFresh(t)
		before := seqBefore(t, s)
		if _, err := s.AddStopWord("blockchain developer"); err != nil {
			t.Fatalf("AddStopWord: %v", err)
		}
		assertBumped(t, s, before, "AddStopWord")
	})

	t.Run("RemoveStopWord", func(t *testing.T) {
		s := openFresh(t)
		sw, err := s.AddStopWord("web3 protocol")
		if err != nil {
			t.Fatalf("AddStopWord: %v", err)
		}
		before := seqBefore(t, s)
		if err := s.RemoveStopWord(sw.ID); err != nil {
			t.Fatalf("RemoveStopWord: %v", err)
		}
		assertBumped(t, s, before, "RemoveStopWord")
	})
}

// TestCatalogueSeqReadDoesNotBump verifies that read-only operations leave the counter unchanged.
func TestCatalogueSeqReadDoesNotBump(t *testing.T) {
	s := openFresh(t)
	id, err := s.Upsert(seqJob("src"), false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	before := seqBefore(t, s)

	if _, err := s.Get(id); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, err := s.List("", ""); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := s.CatalogueSeq(); err != nil {
		t.Fatalf("CatalogueSeq: %v", err)
	}

	after, _ := s.CatalogueSeq()
	if after != before {
		t.Errorf("read operations bumped counter: got %d, want %d", after, before)
	}
}

// TestCatalogueSeqRestoredAfterReopen verifies that after reopening the database the in-memory
// counter is at least as large as the last persisted value (never goes backwards).
func TestCatalogueSeqRestoredAfterReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seq.sqlite")

	s1, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open first: %v", err)
	}
	if _, err := s1.Upsert(seqJob("src"), false); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := s1.Upsert(seqJob("src2"), false); err != nil {
		t.Fatalf("Upsert second: %v", err)
	}
	written, _ := s1.CatalogueSeq()
	s1.Close()

	s2, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open second: %v", err)
	}
	defer s2.Close()
	restored, _ := s2.CatalogueSeq()
	if restored < written {
		t.Errorf("counter went backwards after reopen: got %d, had %d", restored, written)
	}
}
