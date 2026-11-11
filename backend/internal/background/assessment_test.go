package background

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	return s
}

func newTestDirs(t *testing.T) dirs.Dirs {
	t.Helper()
	d := dirs.Dirs{Root: t.TempDir()}
	if err := d.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	return d
}

func minimalTestCfg() *config.Config {
	trueVal := true
	return &config.Config{
		Search: config.Search{
			Version: 1,
			Sites:   []config.Source{{Name: "LinkedIn", URL: "https://linkedin.com/jobs", Enabled: &trueVal}},
		},
		Runtime: config.Runtime{Version: 1, Port: 7373},
		Providers: config.Providers{
			Version: 1,
			Stages: struct {
				Assessment  string `yaml:"assessment"`
				Extraction  string `yaml:"extraction"`
				Preparation string `yaml:"preparation"`
				Matching    string `yaml:"matching,omitempty"`
			}{Assessment: "p", Extraction: "p", Preparation: "p"},
			Providers: map[string]config.Provider{
				"p": {Type: "ollama", Model: "llama3", Endpoint: "http://127.0.0.1:11434", ContextChars: 8192},
			},
		},
		Candidate: config.Candidate{Version: 1, CacheHours: 24},
	}
}

// stubAssessJob records each call and marks the job as assessed.
type stubAssessJob struct {
	mu           sync.Mutex
	calledJobIDs []int64
	err          error
}

func (st *stubAssessJob) fn(s *store.Store, d dirs.Dirs, cfg *config.Config, jobID int64, providerName string, inferFn assess.Inference) error {
	st.mu.Lock()
	st.calledJobIDs = append(st.calledJobIDs, jobID)
	st.mu.Unlock()
	if st.err != nil {
		return st.err
	}
	// Record a minimal successful assessment so the job's assessment column is set.
	return s.Assessment(jobID, map[string]any{"score": 75, "stub": true})
}

func (st *stubAssessJob) count() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.calledJobIDs)
}

// TestAssessmentWorkerDrainsQueue verifies that after Start(), enqueued pending jobs are
// processed: assessFn is called for each job and the queue row transitions to 'done'.
func TestAssessmentWorkerDrainsQueue(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	// Save a minimal candidate context so defaultAssessJob (if used) would not fail.
	if err := s.SaveContext(map[string]any{"version": "v1", "preferences": map[string]any{}}); err != nil {
		t.Fatalf("SaveContext: %v", err)
	}

	// Insert two jobs.
	jobID1, err := s.Upsert(model.JobInput{Title: "SWE", Company: "Acme", OriginalURL: "https://example.com/1", Description: "Go developer", DescriptionStatus: "full", Source: "manual"}, false)
	if err != nil {
		t.Fatalf("Upsert job1: %v", err)
	}
	jobID2, err := s.Upsert(model.JobInput{Title: "CTO", Company: "Startup", OriginalURL: "https://example.com/2", Description: "lead engineer", DescriptionStatus: "full", Source: "manual"}, false)
	if err != nil {
		t.Fatalf("Upsert job2: %v", err)
	}

	// Manually enqueue both jobs.
	if _, err := s.EnqueueJob(jobID1, "fp1", "p", 0); err != nil {
		t.Fatalf("EnqueueJob1: %v", err)
	}
	if _, err := s.EnqueueJob(jobID2, "fp2", "p", 0); err != nil {
		t.Fatalf("EnqueueJob2: %v", err)
	}

	// Enable the queue.
	if err := s.SetQueueEnabled(true); err != nil {
		t.Fatalf("SetQueueEnabled: %v", err)
	}

	stub := &stubAssessJob{}
	w := NewAssessmentWorker(s, d, config.NewHolder(cfg), stub.fn, nil)
	w.Start()

	// Poll until both jobs are processed or timeout.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if stub.count() >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	w.Stop()

	if stub.count() < 2 {
		t.Fatalf("expected assessFn to be called at least 2 times, got %d", stub.count())
	}

	// Both queue rows should now be 'done'.
	for _, jobID := range []int64{jobID1, jobID2} {
		row, err := s.QueueRow(jobID)
		if err != nil {
			t.Fatalf("QueueRow(%d): %v", jobID, err)
		}
		if row == nil {
			t.Fatalf("QueueRow(%d): no row", jobID)
		}
		if row.Status != "done" {
			t.Errorf("job %d: expected queue status 'done', got %q", jobID, row.Status)
		}
	}
}

// TestAssessmentWorkerFailureRetries verifies that when assessFn returns an error, the queue
// row transitions to 'failed' rather than 'done'.
func TestAssessmentWorkerFailureRetries(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID, err := s.Upsert(model.JobInput{Title: "SWE", Company: "Fail Inc", OriginalURL: "https://example.com/fail", Description: "fail", DescriptionStatus: "full", Source: "manual"}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := s.EnqueueJob(jobID, "fpfail", "p", 0); err != nil {
		t.Fatalf("EnqueueJob: %v", err)
	}
	if err := s.SetQueueEnabled(true); err != nil {
		t.Fatalf("SetQueueEnabled: %v", err)
	}

	stub := &stubAssessJob{err: fmt.Errorf("provider unavailable")}
	w := NewAssessmentWorker(s, d, config.NewHolder(cfg), stub.fn, nil)
	w.Start()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if stub.count() >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	w.Stop()

	if stub.count() == 0 {
		t.Fatal("assessFn was never called")
	}

	row, err := s.QueueRow(jobID)
	if err != nil {
		t.Fatalf("QueueRow: %v", err)
	}
	if row == nil {
		t.Fatal("QueueRow: no row")
	}
	if row.Status != "failed" {
		t.Errorf("expected queue status 'failed', got %q", row.Status)
	}
}

// TestAssessmentWorkerStopIsIdempotent verifies Stop can be called multiple times without panic.
func TestAssessmentWorkerStopIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	stub := &stubAssessJob{}
	w := NewAssessmentWorker(s, d, config.NewHolder(cfg), stub.fn, nil)
	w.Start()
	w.Stop()
	w.Stop() // second call must not panic
}
