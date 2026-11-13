package background

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/prepare"
	"github.com/w3con/sifty/internal/store"
)

// callLog is a goroutine-safe log of job IDs passed to a stub function.
type callLog struct {
	mu  sync.Mutex
	ids []int64
}

func (c *callLog) add(id int64) {
	c.mu.Lock()
	c.ids = append(c.ids, id)
	c.mu.Unlock()
}

func (c *callLog) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.ids)
}

// makePrepStub returns a PrepareJob stub and a goroutine-safe log of job IDs it was called with.
func makePrepStub(returnErr error) (PrepareJob, *callLog) {
	called := &callLog{}
	fn := PrepareJob(func(s *store.Store, d dirs.Dirs, cfg *config.Config, jobID, opID int64, request map[string]any, progressFn func(string), inferFn prepare.Inference) error {
		called.add(jobID)
		progressFn("stub progress")
		return returnErr
	})
	return fn, called
}

// queueAndEnablePrep queues a preparation operation for a job and enables nothing special (prep
// worker does not use a separate enabled flag).
func queueAndEnablePrep(t *testing.T, s *store.Store, jobID int64) {
	t.Helper()
	_, err := s.QueuePreparation(jobID, map[string]any{"generate": true, "provider": "p"})
	if err != nil {
		t.Fatalf("QueuePreparation: %v", err)
	}
}

// TestPreparationWorkerDrainsQueue verifies that queued operations are processed, prepareFn is
// called and the operation transitions to 'done'.
func TestPreparationWorkerDrainsQueue(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID, err := s.Upsert(model.JobInput{
		Title: "SWE", Company: "Acme", OriginalURL: "https://example.com/1",
		Description: "Go developer", DescriptionStatus: "full", Source: "manual",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	// Pre-select the job (stage must be 'preselected' for preparation).
	// "saved" maps to the "preselected" stage.
	if _, err := s.Status(jobID, "saved", false); err != nil {
		t.Fatalf("Status: %v", err)
	}
	queueAndEnablePrep(t, s, jobID)

	stub, called := makePrepStub(nil)
	w := NewPreparationWorker(s, d, config.NewHolder(cfg), stub, nil, nil)
	w.Start()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if called.len() >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	w.Stop()

	if called.len() == 0 {
		t.Fatal("prepareFn was never called")
	}

	ops, err := s.PreparationOperations(jobID)
	if err != nil {
		t.Fatalf("PreparationOperations: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("no operations found")
	}
	status, _ := ops[0]["status"].(string)
	if status != "done" {
		t.Errorf("expected operation status 'done', got %q", status)
	}
}

// TestPreparationWorkerFailure verifies that when prepareFn returns an error the operation is
// marked 'failed'.
func TestPreparationWorkerFailure(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID, err := s.Upsert(model.JobInput{
		Title: "PM", Company: "FailCo", OriginalURL: "https://example.com/fail",
		Description: "fail", DescriptionStatus: "full", Source: "manual",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := s.Status(jobID, "saved", false); err != nil {
		t.Fatalf("Status: %v", err)
	}
	queueAndEnablePrep(t, s, jobID)

	stub, called := makePrepStub(fmt.Errorf("provider unavailable"))
	w := NewPreparationWorker(s, d, config.NewHolder(cfg), stub, nil, nil)
	w.Start()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if called.len() >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	w.Stop()

	if called.len() == 0 {
		t.Fatal("prepareFn was never called")
	}

	ops, err := s.PreparationOperations(jobID)
	if err != nil {
		t.Fatalf("PreparationOperations: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("no operations found")
	}
	status, _ := ops[0]["status"].(string)
	if status != "failed" {
		t.Errorf("expected operation status 'failed', got %q", status)
	}
}

// TestPreparationWorkerStopIdempotent verifies Stop can be called multiple times safely.
func TestPreparationWorkerStopIdempotent(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	stub, _ := makePrepStub(nil)
	w := NewPreparationWorker(s, d, config.NewHolder(cfg), stub, nil, nil)
	w.Start()
	w.Stop()
	w.Stop()
}

// TestPreparationWorkerNoDoubleRun verifies that a second queued operation waits while the
// first is running (single concurrency guarantee).
func TestPreparationWorkerNoDoubleRun(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	for i := 0; i < 2; i++ {
		jobID, err := s.Upsert(model.JobInput{
			Title: fmt.Sprintf("Role %d", i), Company: "Co", OriginalURL: fmt.Sprintf("https://example.com/%d", i),
			Description: "desc", DescriptionStatus: "full", Source: "manual",
		}, false)
		if err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
		if _, err := s.Status(jobID, "saved", false); err != nil {
			t.Fatalf("Status %d: %v", i, err)
		}
		queueAndEnablePrep(t, s, jobID)
	}

	// A stub that records the maximum concurrency observed.
	var mu sync.Mutex
	maxConcurrent := 0
	current := 0
	called := &callLog{}
	stub := PrepareJob(func(s *store.Store, d dirs.Dirs, cfg *config.Config, jobID, opID int64, request map[string]any, progressFn func(string), inferFn prepare.Inference) error {
		mu.Lock()
		current++
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()
		called.add(jobID)
		time.Sleep(100 * time.Millisecond) // hold for a moment
		mu.Lock()
		current--
		mu.Unlock()
		return nil
	})

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), stub, nil, nil)
	w.Start()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if called.len() >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	w.Stop()

	mu.Lock()
	mc := maxConcurrent
	mu.Unlock()
	if mc > 1 {
		t.Errorf("preparation worker ran %d operations concurrently; expected at most 1", mc)
	}
}
