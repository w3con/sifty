package fetch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// controlledClient returns a Client with a fake clock, a recording sleep, and
// zero jitter, backed by a real httptest server.
func controlledClient(t *testing.T, nowMS *int64, sleptMS *int64, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(Options{
		AllowPrivate: true,
		NowMS:        func() int64 { return atomic.LoadInt64(nowMS) },
		SleepMS: func(ms int64) {
			atomic.AddInt64(sleptMS, ms)
			atomic.AddInt64(nowMS, ms) // fake clock advances with sleep
		},
		RandIntervalMS: func(_ int64) int64 { return 0 }, // no jitter
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestTrafficInterval(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	var nowMS int64 = 0
	var sleptMS int64 = 0
	c := controlledClient(t, &nowMS, &sleptMS, srv)

	// First request — no previous request, no sleep expected.
	_, _, err := c.GetListingWithDiscipline(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	if atomic.LoadInt64(&sleptMS) != 0 {
		t.Errorf("first request slept %d ms, want 0", atomic.LoadInt64(&sleptMS))
	}

	// Second request — must wait at least minIntervalMS (3000 ms). With jitter=0 it
	// sleeps exactly 3000 ms less any elapsed time (0 ms elapsed since last request).
	_, _, err = c.GetListingWithDiscipline(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	got := atomic.LoadInt64(&sleptMS)
	if got < minIntervalMS {
		t.Errorf("second request slept %d ms, want at least %d ms", got, minIntervalMS)
	}
}

func TestTrafficCooldown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "denied")
	}))
	defer srv.Close()

	var nowMS int64 = 0
	var sleptMS int64 = 0
	c := controlledClient(t, &nowMS, &sleptMS, srv)

	// This request triggers a 403 → BlockedResult → 24 h cooldown set.
	_, blocked, err := c.GetListingWithDiscipline(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("blocked request: %v", err)
	}
	if blocked == nil {
		t.Fatal("expected BlockedResult, got nil")
	}

	// Immediately after: domain is on cooldown — must be refused.
	_, _, err = c.GetListingWithDiscipline(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected cooldown error, got nil")
	}

	// Advance clock by just under 24 h — still on cooldown.
	atomic.AddInt64(&nowMS, int64(blockCooldownH.Milliseconds()-1))
	_, _, err = c.GetListingWithDiscipline(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected cooldown error at 24h-1ms, got nil")
	}

	// Advance clock past the 24 h cooldown — the request goes through.
	atomic.AddInt64(&nowMS, 2) // now 1 ms past the cooldown
	atomic.StoreInt64(&sleptMS, 0)
	// The server still returns 403, but the discipline itself should not refuse the request.
	_, blocked2, err2 := c.GetListingWithDiscipline(context.Background(), srv.URL)
	if err2 != nil {
		t.Fatalf("post-cooldown request returned error: %v", err2)
	}
	if blocked2 == nil {
		t.Fatal("post-cooldown request: expected BlockedResult from 403 server")
	}
}

// TestCloudflare403NoHTTPRetry verifies that a Cloudflare-style 403 response
// is recorded as a block and that no second HTTP request is made: the domain
// enters a 24-hour cooldown and all subsequent calls are refused before
// reaching the network.
func TestCloudflare403NoHTTPRetry(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "<!DOCTYPE html><title>Just a moment...</title>") // Cloudflare body
	}))
	defer srv.Close()

	var nowMS int64 = 0
	var sleptMS int64 = 0
	c := controlledClient(t, &nowMS, &sleptMS, srv)

	// First request — server returns 403, client records it as blocked.
	_, blocked, err := c.GetDetailWithDiscipline(context.Background(), srv.URL+"/job/123")
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	if blocked == nil || blocked.Reason != "403" {
		t.Fatalf("expected BlockedResult reason=403, got %v", blocked)
	}
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Fatalf("expected 1 HTTP request after first call, got %d", n)
	}

	// Second request — domain is on cooldown; no HTTP request must be made.
	_, _, err2 := c.GetDetailWithDiscipline(context.Background(), srv.URL+"/job/123")
	if err2 == nil {
		t.Fatal("expected cooldown error on second call, got nil")
	}
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Fatalf("expected still 1 HTTP request after second (cooled-down) call, got %d", n)
	}
}

func TestTrafficBackoff(t *testing.T) {
	// A server that always returns a network-level error is hard to arrange with httptest.
	// Instead, test the backoff by driving the discipline directly.
	var nowMS int64 = 0
	var sleptMS int64 = 0
	td := newTrafficDiscipline(
		func() int64 { return atomic.LoadInt64(&nowMS) },
		func(ms int64) {
			atomic.AddInt64(&sleptMS, ms)
			atomic.AddInt64(&nowMS, ms)
		},
		func(_ int64) int64 { return 0 }, // no jitter
	)
	const domain = "backoff.example"

	// Record failure 1 — next wait = 30 s.
	td.record(domain, false, false)
	atomic.StoreInt64(&sleptMS, 0)
	if err := td.wait(context.Background(), domain); err != nil {
		t.Fatalf("wait after 1 failure: %v", err)
	}
	got := atomic.LoadInt64(&sleptMS)
	if got != backoffBaseMS {
		t.Errorf("after 1 failure expected sleep %d ms, got %d", backoffBaseMS, got)
	}

	// Record failure 2 — next wait = 60 s.
	td.record(domain, false, false)
	atomic.StoreInt64(&sleptMS, 0)
	if err := td.wait(context.Background(), domain); err != nil {
		t.Fatalf("wait after 2 failures: %v", err)
	}
	got = atomic.LoadInt64(&sleptMS)
	if got != backoffBaseMS*2 {
		t.Errorf("after 2 failures expected sleep %d ms, got %d", backoffBaseMS*2, got)
	}

	// Success resets failures.
	td.record(domain, false, true)
	atomic.StoreInt64(&sleptMS, 0)
	atomic.StoreInt64(&nowMS, 0)
	// Reset last-req so we isolate the backoff test.
	s := td.stateFor(domain)
	s.hasReq = false
	s.lastReqMS = 0
	if err := td.wait(context.Background(), domain); err != nil {
		t.Fatalf("wait after success reset: %v", err)
	}
	if atomic.LoadInt64(&sleptMS) != 0 {
		t.Errorf("after success reset expected no sleep, got %d ms", atomic.LoadInt64(&sleptMS))
	}
}
