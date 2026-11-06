package background

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/store"
)

// collectionLockName is the cross-process lock the daily scheduler and the on-demand trigger
// both use, so they serialise against each other.
const collectionLockName = "daily"

// defaultCooldown is the fallback when no cooldown is configured.
const defaultCooldown = 3 * time.Hour

// TriggerOpts configures a single TriggerCollection call.
type TriggerOpts struct {
	// Source is stored in the collection_runs row (blank means "all").
	Source string
	// Force bypasses the cooldown check; it does not bypass per-domain traffic pacing.
	Force bool
	// CollectFn is the collection implementation; nil uses defaultCollect.
	CollectFn CollectionFn
	// NowFn returns the current time; nil uses time.Now.
	NowFn func() time.Time
	// Cooldown overrides the default 3-hour cooldown; zero uses defaultCooldown.
	Cooldown time.Duration
}

// TriggerResult is the outcome of TriggerCollection.
type TriggerResult struct {
	RunID   int64
	Status  string // "complete", "failed", or "skipped"
	Skipped bool
	Reason  string // non-empty when Skipped is true
}

// TriggerCollection runs a collection cycle under the same cross-process lock the daily scheduler
// uses. When Force is false and the most recent collection_runs row started within the cooldown,
// the call returns immediately with Status "skipped" and records nothing. Otherwise it records a
// new collection_runs row, runs the dispatch, and finishes the row.
func TriggerCollection(s *store.Store, d dirs.Dirs, cfg *config.Config, opts TriggerOpts) (TriggerResult, error) {
	collectFn := opts.CollectFn
	if collectFn == nil {
		collectFn = defaultCollect
	}
	nowFn := opts.NowFn
	if nowFn == nil {
		nowFn = time.Now
	}
	cooldown := opts.Cooldown
	if cooldown == 0 && cfg.Runtime.CollectionCooldownMS > 0 {
		cooldown = time.Duration(cfg.Runtime.CollectionCooldownMS) * time.Millisecond
	}
	if cooldown == 0 {
		cooldown = defaultCooldown
	}

	pid := os.Getpid()

	if err := s.AcquireLock(collectionLockName, pid); err != nil {
		return TriggerResult{}, fmt.Errorf("acquire collection lock: %w", err)
	}
	defer func() { _ = s.ReleaseLock(collectionLockName, pid) }()

	// Cooldown gate: skip if unforced and a recent run exists.
	if !opts.Force {
		last, err := s.LatestCollectionRun()
		if err != nil {
			return TriggerResult{}, fmt.Errorf("latest collection run: %w", err)
		}
		if last != nil {
			if started, ok := last["started"].(string); ok {
				if t, err := parseRunTime(started); err == nil {
					if nowFn().Sub(t) < cooldown {
						return TriggerResult{Status: "skipped", Skipped: true, Reason: "cooldown"}, nil
					}
				}
			}
		}
	}

	runID, err := s.StartCollectionRun(pid, opts.Source, "")
	if err != nil {
		return TriggerResult{}, fmt.Errorf("start collection run: %w", err)
	}

	collectErr := collectFn(s, d, cfg)
	if collectErr != nil {
		_ = s.FinishCollectionRun(runID, "failed")
		return TriggerResult{RunID: runID, Status: "failed"}, collectErr
	}

	if err := s.FinishCollectionRun(runID, "complete"); err != nil {
		return TriggerResult{RunID: runID, Status: "complete"}, fmt.Errorf("finish collection run: %w", err)
	}
	return TriggerResult{RunID: runID, Status: "complete"}, nil
}

// TriggerAsync starts a collection run asynchronously. The cooldown check and lock acquisition
// happen synchronously so the caller immediately knows whether the run was skipped or started.
// When not skipped, collection runs in a goroutine; the run row is marked complete or failed
// when it finishes. Errors from the goroutine are logged and recorded in the run row but are
// not returned to the caller.
//
// Returns: runID (0 when skipped), status ("running" or "skipped"), error (setup failures only).
func TriggerAsync(s *store.Store, d dirs.Dirs, cfg *config.Config, opts TriggerOpts) (int64, string, error) {
	collectFn := opts.CollectFn
	if collectFn == nil {
		collectFn = defaultCollect
	}
	nowFn := opts.NowFn
	if nowFn == nil {
		nowFn = time.Now
	}
	cooldown := opts.Cooldown
	if cooldown == 0 && cfg.Runtime.CollectionCooldownMS > 0 {
		cooldown = time.Duration(cfg.Runtime.CollectionCooldownMS) * time.Millisecond
	}
	if cooldown == 0 {
		cooldown = defaultCooldown
	}

	pid := os.Getpid()

	if err := s.AcquireLock(collectionLockName, pid); err != nil {
		return 0, "", fmt.Errorf("acquire collection lock: %w", err)
	}
	// Lock is released in the goroutine (or immediately on skip/error below).

	if !opts.Force {
		last, err := s.LatestCollectionRun()
		if err != nil {
			_ = s.ReleaseLock(collectionLockName, pid)
			return 0, "", fmt.Errorf("latest collection run: %w", err)
		}
		if last != nil {
			if started, ok := last["started"].(string); ok {
				if t, err := parseRunTime(started); err == nil {
					if nowFn().Sub(t) < cooldown {
						_ = s.ReleaseLock(collectionLockName, pid)
						return 0, "skipped", nil
					}
				}
			}
		}
	}

	runID, err := s.StartCollectionRun(pid, opts.Source, "")
	if err != nil {
		_ = s.ReleaseLock(collectionLockName, pid)
		return 0, "", fmt.Errorf("start collection run: %w", err)
	}

	// Collection runs in a goroutine; the lock is released when it finishes.
	go func() {
		defer func() { _ = s.ReleaseLock(collectionLockName, pid) }()
		collectErr := collectFn(s, d, cfg)
		if collectErr != nil {
			log.Printf("trigger: collection failed (run %d): %v", runID, collectErr)
			_ = s.FinishCollectionRun(runID, "failed")
			return
		}
		if err := s.FinishCollectionRun(runID, "complete"); err != nil {
			log.Printf("trigger: finish run %d: %v", runID, err)
		}
	}()

	return runID, "running", nil
}

// parseRunTime parses a stored run timestamp in one of the formats the store writes.
func parseRunTime(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("parseRunTime: cannot parse %q", s)
}
