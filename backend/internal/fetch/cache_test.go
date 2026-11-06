package fetch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newCachedClient creates a Client backed by a real httptest server and a
// controlled clock. The optional preAdvance shifts the initial clock forward.
func newCachedClient(t *testing.T, srv *httptest.Server, nowMS *int64) *Client {
	t.Helper()
	c, err := New(Options{
		AllowPrivate: true,
		NowMS:        func() int64 { return atomic.LoadInt64(nowMS) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestCacheListing(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		fmt.Fprint(w, "listing page")
	}))
	defer srv.Close()

	var nowMS int64 = 0
	c := newCachedClient(t, srv, &nowMS)

	// First fetch — hits the server.
	p1, blocked, err := c.GetListing(context.Background(), srv.URL)
	if err != nil || blocked != nil {
		t.Fatalf("first GetListing: err=%v blocked=%v", err, blocked)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected 1 server hit, got %d", atomic.LoadInt32(&hits))
	}
	if p1.Cached {
		t.Error("first response must not be marked Cached")
	}

	// Second fetch within TTL — served from cache.
	atomic.AddInt64(&nowMS, int64(ListingTTL/2/time.Millisecond)) // half the TTL
	p2, blocked, err := c.GetListing(context.Background(), srv.URL)
	if err != nil || blocked != nil {
		t.Fatalf("second GetListing: err=%v blocked=%v", err, blocked)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected still 1 server hit after cache hit, got %d", atomic.LoadInt32(&hits))
	}
	if !p2.Cached {
		t.Error("second response must be marked Cached")
	}

	// Advance past the TTL — cache miss, server is hit again.
	atomic.AddInt64(&nowMS, int64(ListingTTL/time.Millisecond))
	p3, blocked, err := c.GetListing(context.Background(), srv.URL)
	if err != nil || blocked != nil {
		t.Fatalf("third GetListing: err=%v blocked=%v", err, blocked)
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Fatalf("expected 2 server hits after TTL expiry, got %d", atomic.LoadInt32(&hits))
	}
	if p3.Cached {
		t.Error("post-expiry response must not be marked Cached")
	}
}

func TestCacheDetail(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		fmt.Fprint(w, "detail page")
	}))
	defer srv.Close()

	var nowMS int64 = 0
	c := newCachedClient(t, srv, &nowMS)

	// First fetch.
	_, blocked, err := c.GetDetail(context.Background(), srv.URL)
	if err != nil || blocked != nil {
		t.Fatalf("first GetDetail: err=%v blocked=%v", err, blocked)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", atomic.LoadInt32(&hits))
	}

	// Within detail TTL (24 h) — cache hit.
	atomic.AddInt64(&nowMS, int64(12*time.Hour/time.Millisecond))
	p, _, _ := c.GetDetail(context.Background(), srv.URL)
	if !p.Cached {
		t.Error("within-TTL response must be Cached")
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected still 1 hit after cache hit, got %d", atomic.LoadInt32(&hits))
	}
}

func TestBlocked403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Access Denied")
	}))
	defer srv.Close()

	var nowMS int64 = 0
	c := newCachedClient(t, srv, &nowMS)

	page, blocked, err := c.GetListing(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != nil {
		t.Error("expected nil page on blocked result")
	}
	if blocked == nil {
		t.Fatal("expected BlockedResult, got nil")
	}
	if blocked.Reason != "403" {
		t.Errorf("expected reason 403, got %q", blocked.Reason)
	}
}

func TestBlockedCaptcha(t *testing.T) {
	bodies := []string{
		"<html>Please verify you are human to continue</html>",
		"<div class='cf-chl-'></div>",
		"<div class='captcha-container'></div>",
		"<title>Just a moment...</title>",
	}
	for _, body := range bodies {
		body := body
		t.Run(body[:20], func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, body)
			}))
			defer srv.Close()

			var nowMS int64 = 0
			c := newCachedClient(t, srv, &nowMS)
			page, blocked, err := c.GetListing(context.Background(), srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if page != nil {
				t.Error("expected nil page on CAPTCHA block")
			}
			if blocked == nil {
				t.Fatal("expected BlockedResult for CAPTCHA body")
			}
			if blocked.Reason != "captcha" {
				t.Errorf("expected reason 'captcha', got %q", blocked.Reason)
			}
		})
	}
}
