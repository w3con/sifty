package store_test

import (
	"time"
	"testing"
)

// TestWorkflowHistory verifies that Status transitions are recorded in workflow_history.
func TestWorkflowHistory(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	// triage → rejected
	if _, err := s.Status(id, "rejected", false); err != nil {
		t.Fatalf("→ rejected: %v", err)
	}
	// rejected → preselected
	if _, err := s.Status(id, "saved", false); err != nil {
		t.Fatalf("→ saved: %v", err)
	}

	hist, err := s.WorkflowHistory(id)
	if err != nil {
		t.Fatalf("WorkflowHistory: %v", err)
	}
	if len(hist) != 2 {
		t.Fatalf("want 2 history entries, got %d", len(hist))
	}
	if hist[0]["fromStage"] != "triage" || hist[0]["toStage"] != "no" {
		t.Errorf("history[0]: got fromStage=%v toStage=%v, want triage→no", hist[0]["fromStage"], hist[0]["toStage"])
	}
	if hist[1]["fromStage"] != "no" || hist[1]["toStage"] != "preselected" {
		t.Errorf("history[1]: got fromStage=%v toStage=%v, want no→preselected", hist[1]["fromStage"], hist[1]["toStage"])
	}
}

// TestAssessmentQueue verifies enqueue, counts and row retrieval.
func TestAssessmentQueue(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	row, err := s.EnqueueJob(id, "fp1", "ollama", 0)
	if err != nil {
		t.Fatalf("EnqueueJob: %v", err)
	}
	if row == nil {
		t.Fatal("EnqueueJob returned nil row")
	}
	if row.Status != "pending" {
		t.Errorf("queue row status: got %q, want pending", row.Status)
	}
	if row.Fingerprint != "fp1" {
		t.Errorf("fingerprint: got %q, want fp1", row.Fingerprint)
	}

	counts, err := s.QueueCounts()
	if err != nil {
		t.Fatalf("QueueCounts: %v", err)
	}
	if counts["pending"] != 1 {
		t.Errorf("pending count: got %d, want 1", counts["pending"])
	}

	// Re-enqueue with higher priority: status stays pending, priority updated.
	row2, err := s.EnqueueJob(id, "fp2", "ollama", 5)
	if err != nil {
		t.Fatalf("EnqueueJob #2: %v", err)
	}
	if row2.Priority < 5 {
		t.Errorf("priority after re-enqueue: got %d, want ≥5", row2.Priority)
	}

	allRows, err := s.AllQueueRows()
	if err != nil {
		t.Fatalf("AllQueueRows: %v", err)
	}
	if len(allRows) != 1 {
		t.Errorf("AllQueueRows: got %d rows, want 1", len(allRows))
	}
}

// TestAssessmentRecord verifies that Assessment and AssessmentFailure write to the database.
func TestAssessmentRecord(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	// Record a failure first.
	if err := s.AssessmentFailure(id, "provider unavailable", nil); err != nil {
		t.Fatalf("AssessmentFailure: %v", err)
	}
	job, _ := s.Get(id)
	if job.AssessmentError == nil || *job.AssessmentError == "" {
		t.Error("AssessmentError: want non-empty after failure")
	}

	// Record success.
	result := map[string]any{
		"score": 85.0, "summary": "Excellent match.",
		"evidence": []any{}, "gaps": []any{},
		"confirmedMismatches": []any{},
		"candidateUnknowns":   []any{},
		"employerUnknowns":    []any{},
		"jobVersion":          "deadbeef",
	}
	if err := s.Assessment(id, result); err != nil {
		t.Fatalf("Assessment: %v", err)
	}
	job2, _ := s.Get(id)
	if job2.Assessment == nil {
		t.Fatal("Assessment: want non-nil after recording")
	}
	m, ok := job2.Assessment.(map[string]any)
	if !ok {
		t.Fatalf("Assessment: want map, got %T", job2.Assessment)
	}
	if m["score"] != 85.0 {
		t.Errorf("score: got %v, want 85", m["score"])
	}
	if job2.AssessmentError != nil {
		t.Error("AssessmentError should be cleared after success")
	}
}

// TestQueueState verifies the queue enabled/disabled toggle.
func TestQueueState(t *testing.T) {
	s := openFresh(t)

	state, err := s.GetQueueState()
	if err != nil {
		t.Fatalf("GetQueueState: %v", err)
	}
	if !state.Enabled {
		t.Error("default queue state should be enabled")
	}

	if err := s.SetQueueEnabled(false); err != nil {
		t.Fatalf("SetQueueEnabled(false): %v", err)
	}
	state2, _ := s.GetQueueState()
	if state2.Enabled {
		t.Error("queue should be disabled after SetQueueEnabled(false)")
	}

	if err := s.SetQueueEnabled(true); err != nil {
		t.Fatalf("SetQueueEnabled(true): %v", err)
	}
	state3, _ := s.GetQueueState()
	if !state3.Enabled {
		t.Error("queue should be enabled after SetQueueEnabled(true)")
	}
}

// TestContext verifies SaveContext and Context round-trip.
func TestContext(t *testing.T) {
	s := openFresh(t)

	// No context initially.
	ctx, err := s.Context()
	if err != nil {
		t.Fatalf("Context (empty): %v", err)
	}
	if ctx != nil {
		t.Error("Context: want nil on empty database")
	}

	// Save and read back.
	c := map[string]any{
		"version":   "v1abc",
		"createdAt": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"sources":   []any{},
	}
	if err := s.SaveContext(c); err != nil {
		t.Fatalf("SaveContext: %v", err)
	}
	ctx2, err := s.Context()
	if err != nil {
		t.Fatalf("Context after save: %v", err)
	}
	if ctx2 == nil {
		t.Fatal("Context: want non-nil after save")
	}
	if ctx2["version"] != "v1abc" {
		t.Errorf("version: got %v, want v1abc", ctx2["version"])
	}
}

// TestEnqueueAllSkipsStoppedRoles verifies that EnqueueAll does not queue a role hidden by an
// active stop-word, still queues the others, and queues the hidden role once the word is removed.
func TestEnqueueAllSkipsStoppedRoles(t *testing.T) {
	s := openFresh(t)
	hidden := stopTestJob(t, s, "Risk Engineer", "hidden", "triage")
	keep := stopTestJob(t, s, "Go Engineer", "keep", "triage")

	sw, err := s.AddStopWord("risk engineer")
	if err != nil {
		t.Fatalf("AddStopWord: %v", err)
	}
	if err := s.EnqueueAll(); err != nil {
		t.Fatalf("EnqueueAll: %v", err)
	}
	if row, err := s.QueueRow(hidden); err != nil || row != nil {
		t.Errorf("stop-word-hidden role: queue row = %+v err=%v, want none", row, err)
	}
	if row, err := s.QueueRow(keep); err != nil || row == nil {
		t.Errorf("visible role: queue row = %+v err=%v, want a queued row", row, err)
	}

	if err := s.RemoveStopWord(sw.ID); err != nil {
		t.Fatalf("RemoveStopWord: %v", err)
	}
	if err := s.EnqueueAll(); err != nil {
		t.Fatalf("EnqueueAll after removal: %v", err)
	}
	if row, err := s.QueueRow(hidden); err != nil || row == nil {
		t.Errorf("after removing the stop-word: queue row = %+v err=%v, want a queued row", row, err)
	}
}
