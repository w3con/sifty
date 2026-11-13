package background

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/store"
)

// stubCollect returns a CollectionFn that increments a counter on each call.
func stubCollect(called *int32) CollectionFn {
	return func(_ *store.Store, _ dirs.Dirs, _ *config.Config) error {
		atomic.AddInt32(called, 1)
		return nil
	}
}

// TestTriggerCollectionForced verifies that a forced call runs the dispatch and records start+finish.
func TestTriggerCollectionForced(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	var called int32
	res, err := TriggerCollection(s, d, cfg, TriggerOpts{Force: true, CollectFn: stubCollect(&called)})
	if err != nil {
		t.Fatalf("TriggerCollection: %v", err)
	}
	if res.Skipped {
		t.Fatal("forced call must not be skipped")
	}
	if atomic.LoadInt32(&called) == 0 {
		t.Fatal("dispatch was not called on a forced trigger")
	}
	if res.RunID == 0 {
		t.Fatal("RunID must be non-zero after a forced run")
	}
	if res.Status != "complete" {
		t.Errorf("status = %q; want complete", res.Status)
	}

	// The run row must exist with a finished timestamp.
	run, err := s.CollectionRunByID(res.RunID)
	if err != nil || run == nil {
		t.Fatalf("CollectionRunByID(%d): %v (run=%v)", res.RunID, err, run)
	}
	if _, ok := run["finished"]; !ok {
		t.Error("collection_runs row has no finished field")
	}
	if run["status"] != "complete" {
		t.Errorf("collection_runs status = %v; want complete", run["status"])
	}
}

// TestTriggerCollectionSkippedWithinCooldown verifies that an unforced call within the cooldown is
// skipped and records no new run.
func TestTriggerCollectionSkippedWithinCooldown(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	// Plant a completed run right now.
	recentID, err := s.StartCollectionRun(0, "test", "")
	if err != nil {
		t.Fatalf("StartCollectionRun: %v", err)
	}
	if err := s.FinishCollectionRun(recentID, "complete"); err != nil {
		t.Fatalf("FinishCollectionRun: %v", err)
	}

	var called int32
	// nowFn = time.Now: the planted run was just created, so we are well within the cooldown.
	res, err := TriggerCollection(s, d, cfg, TriggerOpts{
		Force:     false,
		CollectFn: stubCollect(&called),
		Cooldown:  defaultCooldown,
	})
	if err != nil {
		t.Fatalf("TriggerCollection: %v", err)
	}
	if !res.Skipped {
		t.Fatal("expected unforced call within cooldown to be skipped")
	}
	if res.Status != "skipped" {
		t.Errorf("status = %q; want skipped", res.Status)
	}
	if atomic.LoadInt32(&called) != 0 {
		t.Fatal("dispatch must not be called when skipped")
	}

	// The latest run must still be recentID — no new row was inserted.
	latest, err := s.LatestCollectionRun()
	if err != nil {
		t.Fatalf("LatestCollectionRun: %v", err)
	}
	if latest == nil {
		t.Fatal("unexpected nil latest run")
	}
	id, _ := latest["id"].(int64)
	if id != recentID {
		t.Errorf("latest run ID = %d; want %d (no new run should have been created)", id, recentID)
	}
}

// TestTriggerCollectionDailyRunSuppressesUnforced verifies that a recent daily run (recorded in the
// runs table by the scheduler) suppresses an unforced trigger, and that an older daily run does not.
func TestTriggerCollectionDailyRunSuppressesUnforced(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	// Record a daily run just now via the scheduler's own store methods.
	dailyID, err := s.StartDailyRun()
	if err != nil {
		t.Fatalf("StartDailyRun: %v", err)
	}
	if err := s.FinishDailyRun(dailyID, "complete", nil); err != nil {
		t.Fatalf("FinishDailyRun: %v", err)
	}

	var called int32
	stub := stubCollect(&called)

	// --- recent daily run: should suppress unforced trigger ---
	res, err := TriggerCollection(s, d, cfg, TriggerOpts{
		Force:     false,
		CollectFn: stub,
		Cooldown:  defaultCooldown,
	})
	if err != nil {
		t.Fatalf("TriggerCollection (recent daily): %v", err)
	}
	if !res.Skipped {
		t.Error("recent daily run must suppress an unforced trigger")
	}
	if atomic.LoadInt32(&called) != 0 {
		t.Error("dispatch must not run when suppressed by a recent daily run")
	}

	// --- older daily run: 4 hours in the simulated future, should not suppress ---
	atomic.StoreInt32(&called, 0)
	pastNow := func() time.Time { return time.Now().UTC().Add(4 * time.Hour) }
	res, err = TriggerCollection(s, d, cfg, TriggerOpts{
		Force:     false,
		CollectFn: stub,
		NowFn:     pastNow,
		Cooldown:  defaultCooldown,
	})
	if err != nil {
		t.Fatalf("TriggerCollection (old daily): %v", err)
	}
	if res.Skipped {
		t.Error("old daily run must not suppress an unforced trigger past the cooldown")
	}
	if atomic.LoadInt32(&called) == 0 {
		t.Error("dispatch must run when cooldown has elapsed since the daily run")
	}
}

// TestTriggerCollectionCooldownFromConfig verifies that the cooldown interval is read from
// cfg.Runtime.CollectionCooldownMS: a call within the interval is skipped and a call past it runs.
func TestTriggerCollectionCooldownFromConfig(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	cfg.Runtime.CollectionCooldownMS = 100 // 100 ms cooldown for the test

	// Plant a run right now.
	seedID, err := s.StartCollectionRun(0, "test", "")
	if err != nil {
		t.Fatalf("StartCollectionRun: %v", err)
	}
	if err := s.FinishCollectionRun(seedID, "complete"); err != nil {
		t.Fatalf("FinishCollectionRun: %v", err)
	}

	var called int32
	stub := stubCollect(&called)

	// --- within cooldown: 50 ms after the planted run ---
	withinNow := func() time.Time { return time.Now().UTC().Add(50 * time.Millisecond) }
	res, err := TriggerCollection(s, d, cfg, TriggerOpts{Force: false, CollectFn: stub, NowFn: withinNow})
	if err != nil {
		t.Fatalf("within-cooldown call: %v", err)
	}
	if !res.Skipped {
		t.Error("within-cooldown call should be skipped")
	}
	if atomic.LoadInt32(&called) != 0 {
		t.Error("dispatch must not run within cooldown")
	}

	// --- past cooldown: 200 ms after the planted run ---
	pastNow := func() time.Time { return time.Now().UTC().Add(200 * time.Millisecond) }
	res, err = TriggerCollection(s, d, cfg, TriggerOpts{Force: false, CollectFn: stub, NowFn: pastNow})
	if err != nil {
		t.Fatalf("past-cooldown call: %v", err)
	}
	if res.Skipped {
		t.Error("past-cooldown call must not be skipped")
	}
	if atomic.LoadInt32(&called) == 0 {
		t.Error("dispatch must run past cooldown")
	}
}

// TestTriggerCollectionPastCooldown verifies that an unforced call past the cooldown runs the
// dispatch and records a new run.
func TestTriggerCollectionPastCooldown(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	// Plant a run, then pretend it is 4 hours in the past by advancing nowFn.
	recentID, err := s.StartCollectionRun(0, "test", "")
	if err != nil {
		t.Fatalf("StartCollectionRun: %v", err)
	}
	if err := s.FinishCollectionRun(recentID, "complete"); err != nil {
		t.Fatalf("FinishCollectionRun: %v", err)
	}

	var called int32
	futureNow := func() time.Time { return time.Now().UTC().Add(4 * time.Hour) }
	res, err := TriggerCollection(s, d, cfg, TriggerOpts{
		Force:     false,
		CollectFn: stubCollect(&called),
		NowFn:     futureNow,
		Cooldown:  defaultCooldown,
	})
	if err != nil {
		t.Fatalf("TriggerCollection: %v", err)
	}
	if res.Skipped {
		t.Fatal("expected unforced call past cooldown to run, got skipped")
	}
	if atomic.LoadInt32(&called) == 0 {
		t.Fatal("dispatch was not called")
	}
	if res.Status != "complete" {
		t.Errorf("status = %q; want complete", res.Status)
	}
	if res.RunID == recentID {
		t.Error("RunID must be a new row, not the previously planted one")
	}
}

// TestCollectionDisabledInConfiguration verifies that collection_disabled stops manual runs, the
// asynchronous start used by POST /api/search, and the daily scheduler, and that none of them
// records a run or calls the dispatch.
func TestCollectionDisabledInConfiguration(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	cfg.Runtime.CollectionDisabled = true
	cfg.Runtime.DailyHour, cfg.Runtime.DailyMinute = 0, 0 // due at any time of day

	var called int32
	res, err := TriggerCollection(s, d, cfg, TriggerOpts{Force: true, CollectFn: stubCollect(&called)})
	if err != nil || !res.Skipped || res.RunID != 0 {
		t.Fatalf("TriggerCollection: res=%+v err=%v; want skipped without a run", res, err)
	}
	runID, status, err := TriggerAsync(s, d, cfg, TriggerOpts{Force: true, CollectFn: stubCollect(&called)})
	if err != nil || runID != 0 || status != "skipped" {
		t.Fatalf("TriggerAsync: runID=%d status=%q err=%v; want 0, skipped", runID, status, err)
	}
	sc := NewDailyScheduler(s, d, config.NewHolder(cfg), stubCollect(&called), simulatedTime(time.Now()))
	sc.tick()
	time.Sleep(200 * time.Millisecond)
	sc.Stop()

	if n := atomic.LoadInt32(&called); n != 0 {
		t.Fatalf("dispatch called %d times while collection is disabled", n)
	}
	if run, _ := s.LatestCollectionRun(); run != nil {
		t.Fatalf("a collection run was recorded: %v", run)
	}
}
