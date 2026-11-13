// Package background — browser discipline (pacing, concurrency, cooldown).
// See assessment.go for package-level documentation.
package background

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/url"
	"sync"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/inspect"
)

const (
	defaultBrowserIntervalMS  = 5000
	defaultBrowserJitterMS    = 3000
	defaultBrowserConcurrency = 1
	defaultBrowserCooldownMS  = 300_000
)

// Clock allows callers to control time in tests without real sleeps.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// SystemClock is the production Clock implementation backed by the wall clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time        { return time.Now() }
func (SystemClock) Sleep(d time.Duration) { time.Sleep(d) }

// BrowserDiscipline enforces per-host pacing, global concurrency limits and per-host cooldowns
// on browser inspections. It lives in the Go process — which is long-lived — so its state
// persists across the many short-lived Playwright subprocess calls.
type BrowserDiscipline struct {
	mu      sync.Mutex
	cond    *sync.Cond
	lastNav map[string]time.Time // hostname -> time the navigation slot was last claimed
	blocked map[string]time.Time // hostname -> time the cooldown started
	running int
	clock   Clock
	holder  *config.Holder
}

// NewBrowserDiscipline creates a BrowserDiscipline that reads live settings from holder.
// Use SystemClock{} in production and a fake clock in tests.
func NewBrowserDiscipline(holder *config.Holder, clock Clock) *BrowserDiscipline {
	d := &BrowserDiscipline{
		lastNav: make(map[string]time.Time),
		blocked: make(map[string]time.Time),
		clock:   clock,
		holder:  holder,
	}
	d.cond = sync.NewCond(&d.mu)
	return d
}

func (d *BrowserDiscipline) effectiveInterval() time.Duration {
	ms := d.holder.Get().Runtime.BrowserIntervalMS
	if ms <= 0 {
		ms = defaultBrowserIntervalMS
	}
	return time.Duration(ms) * time.Millisecond
}

func (d *BrowserDiscipline) effectiveJitter() time.Duration {
	ms := d.holder.Get().Runtime.BrowserJitterMS
	if ms <= 0 {
		ms = defaultBrowserJitterMS
	}
	return time.Duration(ms) * time.Millisecond
}

func (d *BrowserDiscipline) effectiveConcurrency() int {
	n := d.holder.Get().Runtime.BrowserConcurrency
	if n <= 0 {
		n = defaultBrowserConcurrency
	}
	return n
}

func (d *BrowserDiscipline) effectiveCooldown() time.Duration {
	ms := d.holder.Get().Runtime.BrowserCooldownMS
	if ms <= 0 {
		ms = defaultBrowserCooldownMS
	}
	return time.Duration(ms) * time.Millisecond
}

// jitterDuration returns a random duration in [0, jitter). Returns 0 when jitter ≤ 0,
// making test configs with no jitter fully deterministic.
func jitterDuration(jitter time.Duration) time.Duration {
	if jitter <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(jitter)))
}

// disciplineHostname extracts the hostname from rawURL, returning "" on parse error.
func disciplineHostname(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// Wrap returns an InspectFn that applies the browser discipline before and after calling inner.
// Order per call: (1) cooldown check, (2) pacing sleep, (3) semaphore acquire,
// (4) inner call, (5) cooldown registration on backed-off.
func (d *BrowserDiscipline) Wrap(inner InspectFn) InspectFn {
	return func(ctx context.Context, applyURL string) (inspect.Questionnaire, error) {
		host := disciplineHostname(applyURL)

		// --- (1) Cooldown check ---
		if host != "" {
			d.mu.Lock()
			if when, ok := d.blocked[host]; ok {
				cooldown := d.effectiveCooldown()
				if d.clock.Now().Sub(when) < cooldown {
					unblockAt := when.Add(cooldown)
					d.mu.Unlock()
					return inspect.Questionnaire{
						URL:    applyURL,
						Status: "backed-off",
						Fields: []inspect.Question{},
						Gaps: []string{fmt.Sprintf(
							"Host %s is on cooldown after a previous block; inspection paused until %s.",
							host, unblockAt.Format(time.RFC3339),
						)},
					}, nil
				}
				// Cooldown has lapsed; clear the entry and proceed.
				delete(d.blocked, host)
			}
			d.mu.Unlock()
		}

		// --- (2) Pacing: sleep until at least interval+jitter since the last navigation slot ---
		if host != "" {
			d.mu.Lock()
			last, hasLast := d.lastNav[host]
			d.mu.Unlock()
			if hasLast {
				interval := d.effectiveInterval()
				jitter := jitterDuration(d.effectiveJitter())
				need := interval + jitter
				elapsed := d.clock.Now().Sub(last)
				if elapsed < need {
					d.clock.Sleep(need - elapsed)
				}
			}
		}

		// --- (3) Semaphore acquire: wait until a concurrency slot is free ---
		d.mu.Lock()
		for d.running >= d.effectiveConcurrency() {
			d.cond.Wait()
		}
		d.running++
		if host != "" {
			// Record when this slot was claimed; pacing for the next call measures from here.
			d.lastNav[host] = d.clock.Now()
		}
		d.mu.Unlock()

		defer func() {
			d.mu.Lock()
			d.running--
			d.mu.Unlock()
			d.cond.Signal()
		}()

		// --- (4) Call inner ---
		q, err := inner(ctx, applyURL)

		// --- (5) Register cooldown if the site signalled a block ---
		if err == nil && q.Status == "backed-off" && host != "" {
			d.mu.Lock()
			d.blocked[host] = d.clock.Now()
			d.mu.Unlock()
		}

		return q, err
	}
}
