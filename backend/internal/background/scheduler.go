// Package background — daily collection scheduler component.
// See assessment.go for package-level documentation.
package background

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/store"
)

// CollectionFn is the injectable function that executes the daily collection: refresh context,
// run search across all enabled sources, enqueue results. The real implementation calls the Go
// collection core; tests inject a stub.
type CollectionFn func(s *store.Store, d dirs.Dirs, cfg *config.Config) error

// DailyScheduler wakes up once per minute and fires the configured daily collection exactly once
// per calendar day at or after the configured hour and minute. It serialises through the
// database 'daily' lock so a concurrent CLI call does not duplicate a run.
type DailyScheduler struct {
	store        *store.Store
	dirs         dirs.Dirs
	cfg          *config.Holder
	collectFn    CollectionFn
	afterCollect func() error
	nowFn        func() time.Time // injectable for tests

	mu      sync.Mutex
	busy    bool
	stopped bool
	cancel  context.CancelFunc
}

// NewDailyScheduler creates a scheduler. collectFn is injectable; nil uses defaultCollect.
// nowFn is injectable for tests to simulate time; nil uses time.Now.
func NewDailyScheduler(s *store.Store, d dirs.Dirs, cfg *config.Holder, collectFn CollectionFn, nowFn func() time.Time) *DailyScheduler {
	if collectFn == nil {
		collectFn = defaultCollect
	}
	if nowFn == nil {
		nowFn = time.Now
	}
	return &DailyScheduler{
		store: s, dirs: d, cfg: cfg,
		collectFn: collectFn, nowFn: nowFn,
	}
}

// SetAfterCollect wires the live post-collection effect (Quick Match under the authoritative
// config holder). The collector keeps the config snapshot it used for fetching.
func (sc *DailyScheduler) SetAfterCollect(fn func() error) { sc.afterCollect = fn }

// Start launches the scheduler goroutine, which checks every minute whether the daily run is due.
func (sc *DailyScheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	sc.cancel = cancel
	go func() {
		sc.tick()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sc.tick()
			}
		}
	}()
}

// Stop signals the scheduler goroutine to stop; it does not wait for an in-flight run.
func (sc *DailyScheduler) Stop() {
	sc.mu.Lock()
	sc.stopped = true
	sc.mu.Unlock()
	if sc.cancel != nil {
		sc.cancel()
	}
}

func (sc *DailyScheduler) tick() {
	sc.mu.Lock()
	if sc.busy || sc.stopped {
		sc.mu.Unlock()
		return
	}

	cfg := sc.cfg.Get()
	if cfg.Runtime.CollectionDisabled {
		sc.mu.Unlock()
		return
	}
	now := sc.nowFn()
	hour := cfg.Runtime.DailyHour
	minute := cfg.Runtime.DailyMinute

	// Check if the configured time has passed today.
	due := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if now.Before(due) {
		sc.mu.Unlock()
		return
	}

	// Check if already run today (using the store, not the lock) before claiming busy.
	if sc.store.HasDailyRunToday(hour, minute) {
		sc.mu.Unlock()
		return
	}

	sc.busy = true
	sc.mu.Unlock()

	go func() {
		defer func() {
			sc.mu.Lock()
			sc.busy = false
			sc.mu.Unlock()
		}()
		sc.runOnce()
	}()
}

func (sc *DailyScheduler) runOnce() {
	cfg := sc.cfg.Get()
	pid := os.Getpid()

	// Acquire the 'daily' lock to prevent parallel runs across processes.
	if err := sc.store.AcquireLock("daily", pid); err != nil {
		return // another process is running the daily collection
	}
	defer func() { _ = sc.store.ReleaseLock("daily", pid) }()

	// Re-check inside the lock in case another goroutine or process completed a run while we
	// were acquiring the lock.
	if sc.store.HasDailyRunToday(cfg.Runtime.DailyHour, cfg.Runtime.DailyMinute) {
		return
	}

	runID, err := sc.store.StartDailyRun()
	if err != nil {
		log.Printf("daily scheduler: start run: %v", err)
		return
	}

	err = sc.collectFn(sc.store, sc.dirs, cfg)
	if err != nil {
		log.Printf("daily scheduler: collection failed: %v", err)
		_ = sc.store.FinishDailyRun(runID, "failed", err.Error())
		return
	}
	details := map[string]any{"triggered": "scheduler"}
	if sc.afterCollect != nil {
		if qerr := sc.afterCollect(); qerr != nil {
			log.Printf("daily scheduler: post-collection quick match: %v", qerr)
			details["quickMatchWarning"] = qerr.Error()
		}
	}
	_ = sc.store.FinishDailyRun(runID, "complete", details)
}

// defaultCollect is defined in collect.go.
