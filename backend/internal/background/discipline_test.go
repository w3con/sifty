package background

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/inspect"
)

// fakeClock is a deterministic Clock for tests. Sleep advances the internal time
// by the requested duration without blocking the goroutine.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f *fakeClock) Sleep(d time.Duration) { f.advance(d) }

// disciplineHolder builds a minimal *config.Holder for use in discipline tests.
// Zero values in browser fields trigger the BrowserDiscipline defaults.
func disciplineHolder(bIntervalMS, bJitterMS, bConcurrency, bCooldownMS int) *config.Holder {
	return config.NewHolder(&config.Config{
		Runtime: config.Runtime{
			BrowserIntervalMS:  bIntervalMS,
			BrowserJitterMS:    bJitterMS,
			BrowserConcurrency: bConcurrency,
			BrowserCooldownMS:  bCooldownMS,
		},
	})
}

// TestPacingSpacesInspections: two inspections to the same host are spaced by at least
// the configured interval (fake clock, no real sleep).
func TestPacingSpacesInspections(t *testing.T) {
	fc := newFakeClock()
	h := disciplineHolder(500, 0, 1, 300_000)
	d := NewBrowserDiscipline(h, fc)

	var calls []time.Time
	inner := func(_ context.Context, _ string) (inspect.Questionnaire, error) {
		calls = append(calls, fc.Now())
		return inspect.Questionnaire{Status: "partial", Fields: []inspect.Question{{ID: "q1"}}}, nil
	}

	wrapped := d.Wrap(inner)
	ctx := context.Background()
	for i := range 2 {
		if _, err := wrapped(ctx, "https://example.com/jobs/apply"); err != nil {
			t.Fatalf("inspection %d: %v", i, err)
		}
	}

	if len(calls) != 2 {
		t.Fatalf("want 2 inner calls, got %d", len(calls))
	}
	gap := calls[1].Sub(calls[0])
	want := 500 * time.Millisecond
	if gap < want {
		t.Errorf("gap between inspections %v < configured interval %v", gap, want)
	}
}

// TestBackedOffRegistersAndBlocksSecondCall: a backed-off result puts the host on cooldown;
// a second inspection within the cooldown window does not invoke the worker.
func TestBackedOffRegistersAndBlocksSecondCall(t *testing.T) {
	fc := newFakeClock()
	h := disciplineHolder(1, 0, 1, 300_000) // 1ms interval so pacing is negligible
	d := NewBrowserDiscipline(h, fc)

	var calls int32
	inner := func(_ context.Context, _ string) (inspect.Questionnaire, error) {
		atomic.AddInt32(&calls, 1)
		return inspect.Questionnaire{
			URL:    "https://ats.example.com/apply",
			Status: "backed-off",
			Gaps:   []string{"HTTP 429; host put on cooldown."},
		}, nil
	}

	wrapped := d.Wrap(inner)
	ctx := context.Background()

	r1, err := wrapped(ctx, "https://ats.example.com/apply")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if r1.Status != "backed-off" {
		t.Errorf("first call status = %q, want backed-off", r1.Status)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("first call: inner called %d times, want 1", calls)
	}

	// Second call within cooldown: must not reach inner.
	r2, err := wrapped(ctx, "https://ats.example.com/apply")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if r2.Status != "backed-off" {
		t.Errorf("second call status = %q, want backed-off", r2.Status)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("second call within cooldown: inner was called again (want still 1, got %d)", calls)
	}
	if len(r2.Gaps) == 0 {
		t.Error("second call: cooldown result must include a gap message")
	}
}

// TestCooldownExpiresAfterInterval: after the cooldown has elapsed the worker is invoked again.
func TestCooldownExpiresAfterInterval(t *testing.T) {
	fc := newFakeClock()
	h := disciplineHolder(1, 0, 1, 500) // 1ms interval, 500ms cooldown
	d := NewBrowserDiscipline(h, fc)

	var calls int32
	inner := func(_ context.Context, _ string) (inspect.Questionnaire, error) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			return inspect.Questionnaire{Status: "backed-off", Gaps: []string{"blocked"}}, nil
		}
		return inspect.Questionnaire{Status: "partial", Fields: []inspect.Question{{ID: "q1"}}}, nil
	}

	wrapped := d.Wrap(inner)
	ctx := context.Background()

	// First call: blocked → registers cooldown.
	if _, err := wrapped(ctx, "https://slow.example.com/apply"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("first call: inner called %d times, want 1", calls)
	}

	// Advance clock past cooldown.
	fc.advance(600 * time.Millisecond)

	// Second call: cooldown expired → inner is invoked again.
	r, err := wrapped(ctx, "https://slow.example.com/apply")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if r.Status == "backed-off" {
		t.Errorf("second call after cooldown must not be backed-off, got %q", r.Status)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("second call after cooldown: inner called %d times, want 2", calls)
	}
}

// TestConcurrencyBound: concurrent inspections never exceed the configured concurrency limit.
func TestConcurrencyBound(t *testing.T) {
	h := disciplineHolder(0, 0, 2, 300_000) // limit 2; 0 triggers defaults for interval
	d := NewBrowserDiscipline(h, SystemClock{})

	var mu sync.Mutex
	var peak, concurrent int

	inner := func(_ context.Context, _ string) (inspect.Questionnaire, error) {
		mu.Lock()
		concurrent++
		if concurrent > peak {
			peak = concurrent
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond) // hold the slot open so goroutines pile up

		mu.Lock()
		concurrent--
		mu.Unlock()
		return inspect.Questionnaire{Status: "partial", Fields: []inspect.Question{{ID: "q1"}}}, nil
	}

	wrapped := d.Wrap(inner)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Different host per goroutine so pacing does not interfere.
			url := fmt.Sprintf("https://host%d.example.com/apply", i)
			if _, err := wrapped(ctx, url); err != nil {
				t.Errorf("goroutine %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	if peak > 2 {
		t.Errorf("peak concurrency %d exceeds limit 2", peak)
	}
	if peak == 0 {
		t.Error("no inspections ran")
	}
}

// TestDefaultsApplyWhenConfigKeysAreZero: absent browser config fields (zero values) fall
// back to the documented conservative defaults.
func TestDefaultsApplyWhenConfigKeysAreZero(t *testing.T) {
	h := config.NewHolder(&config.Config{}) // all zero values
	d := NewBrowserDiscipline(h, SystemClock{})

	if got, want := d.effectiveInterval(), time.Duration(defaultBrowserIntervalMS)*time.Millisecond; got != want {
		t.Errorf("effectiveInterval() = %v, want %v", got, want)
	}
	if got, want := d.effectiveJitter(), time.Duration(defaultBrowserJitterMS)*time.Millisecond; got != want {
		t.Errorf("effectiveJitter() = %v, want %v", got, want)
	}
	if got, want := d.effectiveConcurrency(), defaultBrowserConcurrency; got != want {
		t.Errorf("effectiveConcurrency() = %v, want %v", got, want)
	}
	if got, want := d.effectiveCooldown(), time.Duration(defaultBrowserCooldownMS)*time.Millisecond; got != want {
		t.Errorf("effectiveCooldown() = %v, want %v", got, want)
	}
}
