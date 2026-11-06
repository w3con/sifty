package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubTransport returns a fixed response for every request.
type stubTransport struct {
	status int
	body   string
}

func (s *stubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.status,
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

// newTestClient creates a Client with AllowPrivate=false and a fake lookup that
// maps the given hostname to the given address. The underlying transport is a
// stubTransport so no real network calls are made.
func newTestClient(t *testing.T, host, addr string) *Client {
	t.Helper()
	c, err := New(Options{
		Transport: &stubTransport{status: 200, body: "ok"},
		LookupHost: func(_ context.Context, h string) ([]string, error) {
			if h == host {
				return []string{addr}, nil
			}
			return nil, fmt.Errorf("unexpected host %q in test", h)
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestSSRFGuard(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		addr    string
		wantErr bool
	}{
		// Private / loopback ranges — all must be blocked.
		{name: "loopback IPv4", host: "local.example", addr: "127.0.0.1", wantErr: true},
		{name: "loopback IPv6", host: "local.example", addr: "::1", wantErr: true},
		{name: "RFC-1918 10.x", host: "intranet.example", addr: "10.0.0.1", wantErr: true},
		{name: "RFC-1918 172.16", host: "intranet.example", addr: "172.16.0.1", wantErr: true},
		{name: "RFC-1918 192.168", host: "intranet.example", addr: "192.168.1.1", wantErr: true},
		{name: "link-local 169.254", host: "ll.example", addr: "169.254.1.1", wantErr: true},
		{name: "unique-local IPv6 fc", host: "v6.example", addr: "fc00::1", wantErr: true},
		{name: "unique-local IPv6 fd", host: "v6.example", addr: "fd00::1", wantErr: true},
		{name: "unspecified IPv4", host: "z.example", addr: "0.0.0.0", wantErr: true},
		{name: "multicast", host: "mc.example", addr: "224.0.0.1", wantErr: true},
		// A plausible public address — must be allowed.
		{name: "public IPv4", host: "api.greenhouse.io", addr: "54.175.0.1", wantErr: false},
		{name: "public IPv4 2", host: "api.lever.co", addr: "104.16.0.1", wantErr: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, tc.host, tc.addr)
			_, err := c.Get(context.Background(), "https://"+tc.host+"/")
			if tc.wantErr && err == nil {
				t.Errorf("expected SSRF error for %s (%s), got nil", tc.host, tc.addr)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for %s (%s): %v", tc.host, tc.addr, err)
			}
		})
	}
}

func TestSizeCap(t *testing.T) {
	const cap = 100
	body := strings.Repeat("x", cap+1) // one byte over the cap

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c, err := New(Options{
		SizeCap:      cap,
		AllowPrivate: true, // httptest server listens on 127.0.0.1
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected size-cap error, got nil")
	}
	if !strings.Contains(err.Error(), "size cap") {
		t.Errorf("expected ErrSizeCap in error message, got: %v", err)
	}
}

func TestSizeCapExactBoundary(t *testing.T) {
	const cap = 100
	body := strings.Repeat("x", cap) // exactly at the cap — must succeed

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c, err := New(Options{
		SizeCap:      cap,
		AllowPrivate: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	page, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error at exact cap: %v", err)
	}
	if len(page.Body) != cap {
		t.Errorf("expected body length %d, got %d", cap, len(page.Body))
	}
}
