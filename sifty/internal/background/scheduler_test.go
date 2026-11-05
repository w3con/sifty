package background

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/store"
)

// simulatedTime returns a nowFn that returns a fixed time.
func simulatedTime(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// TestDailySchedulerFiresAtConfiguredTime verifies that the scheduler invokes collectFn when the
// current (simulated) time is at or after the configured hour and minute and no run has occurred
// today.
func TestDailySchedulerFiresAtConfiguredTime(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	cfg.Runtime.DailyHour = 8
	cfg.Runtime.DailyMinute = 30

	// Simulate a time that is past the configured run time.
	now := time.Now()
	simulated := time.Date(now.Year(), now.Month(), now.Day(), 8, 31, 0, 0, time.Local)

	var called int32
	collectFn := func(s *store.Store, d dirs.Dirs, cfg *config.Config) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	sc := NewDailyScheduler(s, d, config.NewHolder(cfg), collectFn, simulatedTime(simulated))
	sc.tick()

	// Give the goroutine launched by tick a moment to complete.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&called) >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	sc.Stop()

	if atomic.LoadInt32(&called) == 0 {
		t.Fatal("collectFn was not called when scheduled time had passed")
	}
}

// TestDailySchedulerSkipsBeforeTime verifies that the scheduler does not fire before the
// configured hour and minute.
func TestDailySchedulerSkipsBeforeTime(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	cfg.Runtime.DailyHour = 23
	cfg.Runtime.DailyMinute = 59

	// Simulate a time well before the configured run time.
	simulated := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

	var called int32
	collectFn := func(s *store.Store, d dirs.Dirs, cfg *config.Config) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	sc := NewDailyScheduler(s, d, config.NewHolder(cfg), collectFn, simulatedTime(simulated))
	sc.tick()
	time.Sleep(100 * time.Millisecond) // give any mistaken goroutine time to fire
	sc.Stop()

	if atomic.LoadInt32(&called) != 0 {
		t.Fatal("collectFn was called before the scheduled time")
	}
}

// TestDailySchedulerSkipsWhenAlreadyRanToday verifies that the scheduler does not fire a second
// time after a successful run has been recorded for today.
func TestDailySchedulerSkipsWhenAlreadyRanToday(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	cfg.Runtime.DailyHour = 8
	cfg.Runtime.DailyMinute = 0

	// Record a completed daily run at 08:01 today.
	runID, err := s.StartDailyRun()
	if err != nil {
		t.Fatalf("StartDailyRun: %v", err)
	}
	if err := s.FinishDailyRun(runID, "complete", nil); err != nil {
		t.Fatalf("FinishDailyRun: %v", err)
	}

	// Simulate a time past the configured time (so the time-gate passes).
	now := time.Now()
	simulated := time.Date(now.Year(), now.Month(), now.Day(), 8, 5, 0, 0, time.Local)

	var called int32
	collectFn := func(s *store.Store, d dirs.Dirs, cfg *config.Config) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	sc := NewDailyScheduler(s, d, config.NewHolder(cfg), collectFn, simulatedTime(simulated))
	sc.tick()
	time.Sleep(100 * time.Millisecond)
	sc.Stop()

	if atomic.LoadInt32(&called) != 0 {
		t.Fatal("collectFn was called despite a completed run already being recorded today")
	}
}

// TestDailySchedulerStopIdempotent verifies Stop can be called multiple times without panic.
func TestDailySchedulerStopIdempotent(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	sc := NewDailyScheduler(s, d, config.NewHolder(cfg), nil, nil)
	sc.Start()
	sc.Stop()
	sc.Stop()
}
