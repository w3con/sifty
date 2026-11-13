package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/fetch"
)

// fixtureServer returns a test server that always serves the given file content
// with the given Content-Type header.
func fixtureServer(t *testing.T, path, contentType string) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write(body)
	}))
}

func newTestGenericSource(t *testing.T) *GenericSource {
	t.Helper()
	c, err := fetch.New(fetch.Options{AllowPrivate: true})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	return NewGenericSource(c)
}

func TestGenericSourceJSONLD(t *testing.T) {
	srv := fixtureServer(t, "testdata/jsonld_listing.html", "text/html")
	defer srv.Close()

	g := newTestGenericSource(t)
	refs, err := g.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	// Compare against the parse primitive directly.
	body, _ := os.ReadFile("testdata/jsonld_listing.html")
	want := collect.StructuredJobs(string(body), srv.URL, "")
	if len(refs) != len(want) {
		t.Fatalf("got %d refs, want %d", len(refs), len(want))
	}
	for i, ref := range refs {
		if ref.Partial.Title != want[i].Title {
			t.Errorf("ref[%d].Title = %q, want %q", i, ref.Partial.Title, want[i].Title)
		}
		if ref.Partial.Company != want[i].Company {
			t.Errorf("ref[%d].Company = %q, want %q", i, ref.Partial.Company, want[i].Company)
		}
	}

	// Resolve must return the partial without a network round-trip (complete fields).
	for _, ref := range refs {
		j, err := g.Resolve(context.Background(), ref)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if j.Title != ref.Partial.Title {
			t.Errorf("Resolve title mismatch: got %q, want %q", j.Title, ref.Partial.Title)
		}
	}
}

func TestGenericSourceFeed(t *testing.T) {
	srv := fixtureServer(t, "testdata/feed_listing.xml", "application/rss+xml")
	defer srv.Close()

	g := newTestGenericSource(t)
	refs, err := g.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	body, _ := os.ReadFile("testdata/feed_listing.xml")
	want := collect.ParseFeed(string(body), srv.URL, "")
	if len(refs) != len(want) {
		t.Fatalf("got %d refs, want %d", len(refs), len(want))
	}
	for i, ref := range refs {
		if ref.Partial.Title != want[i].Title {
			t.Errorf("ref[%d].Title = %q, want %q", i, ref.Partial.Title, want[i].Title)
		}
	}
}

func TestGenericSourceHTMLLinks(t *testing.T) {
	srv := fixtureServer(t, "testdata/html_listing.html", "text/html")
	defer srv.Close()

	g := newTestGenericSource(t)
	refs, err := g.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	body, _ := os.ReadFile("testdata/html_listing.html")
	want := collect.HTMLLinks(string(body), srv.URL, "")
	if len(refs) != len(want) {
		t.Fatalf("got %d refs, want %d (refs: %+v)", len(refs), len(want), refs)
	}
	for i, ref := range refs {
		if ref.Partial.Title != want[i].Title {
			t.Errorf("ref[%d].Title = %q, want %q", i, ref.Partial.Title, want[i].Title)
		}
		if ref.URL == "" {
			t.Errorf("ref[%d].URL is empty", i)
		}
	}
}
