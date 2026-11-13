package source

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/fetch"
)

// TestWWRRegistryPattern verifies that WWRPattern routes weworkremotely.com to
// *WWRSource and not to the generic fallback.
func TestWWRRegistryPattern(t *testing.T) {
	c, err := fetch.New(fetch.Options{AllowPrivate: true})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	fallback := NewGenericSource(c)
	wwr := NewWWRSource(c)

	reg := NewRegistry(fallback)
	reg.Register(WWRPattern, wwr)

	got := reg.For("weworkremotely.com")
	if _, ok := got.(*WWRSource); !ok {
		t.Errorf("reg.For(weworkremotely.com) = %T, want *WWRSource", got)
	}
	if _, ok := reg.For("remote3.co").(*GenericSource); !ok {
		t.Error("reg.For(remote3.co) should fall back to *GenericSource")
	}
}

// TestWWRDiscoverRSSEntityEscaped verifies that Discover fetches the RSS feed
// and processes entity-escaped HTML descriptions correctly. The fixture is built
// from the first two items of a real weworkremotely.com RSS response. The first
// item (Smart Suites) has entity-escaped HTML with &lt;strong&gt;, &lt;ul&gt;,
// &lt;li&gt; that must become real markup in description_html.
func TestWWRDiscoverRSSEntityEscaped(t *testing.T) {
	srv := fixtureServer(t, "testdata/wwr_rss.xml", "application/rss+xml; charset=utf-8")
	defer srv.Close()

	c, err := fetch.New(fetch.Options{AllowPrivate: true})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	src := NewWWRSource(c)

	// Discover passes the fixture server URL as the listing URL; the adapter
	// will append /remote-jobs.rss, which the test server ignores (serves the
	// same fixture for any path).
	refs, err := src.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("Discover returned no refs; RSS fixture must have items")
	}

	// First item (Smart Suites) has a full entity-escaped HTML description with
	// <strong>, <ul> and <li> that must be decoded and present in description_html.
	first := refs[0].Partial
	if first.DescriptionHTML == "" {
		t.Fatal("first item: DescriptionHTML is empty; entity-escaped HTML not processed")
	}
	// &lt;strong&gt; must become <strong> in the output.
	if !strings.Contains(first.DescriptionHTML, "<strong>") {
		t.Errorf("first item: DescriptionHTML has no <strong> (entity-escaped strong not decoded):\n%s",
			first.DescriptionHTML[:min(len(first.DescriptionHTML), 400)])
	}
	// &lt;ul&gt;/&lt;li&gt; must become <ul>/<li>.
	if !strings.Contains(first.DescriptionHTML, "<li>") {
		t.Errorf("first item: DescriptionHTML has no <li> (entity-escaped li not decoded):\n%s",
			first.DescriptionHTML[:min(len(first.DescriptionHTML), 400)])
	}
	// Plain description must not contain escaped entities or raw tags.
	if strings.Contains(first.Description, "&lt;") || strings.Contains(first.Description, "&amp;amp;") {
		t.Errorf("first item: Description contains escaped entities: %q",
			first.Description[:min(len(first.Description), 200)])
	}
	for _, tag := range []string{"<strong>", "<ul>", "<li>", "<p>"} {
		if strings.Contains(first.Description, tag) {
			t.Errorf("first item: plain Description contains raw tag %q", tag)
		}
	}
}

// TestWWRPageBodyChromeStripping verifies that wwrExtractContentFromBody finds
// the listing-container section in the fixture page body and that the returned
// content excludes the navigation chrome ("× All jobs …") and footer
// ("Posted 1 day ago").
func TestWWRPageBodyChromeStripping(t *testing.T) {
	body := readFixture(t, "testdata/wwr_job_body.html")

	content := wwrExtractContentFromBody(body)

	// The chrome text must not appear in the extracted content.
	if strings.Contains(content, "× All jobs") {
		t.Error("extracted content still contains navigation chrome '× All jobs'")
	}
	if strings.Contains(content, "Posted 1 day ago") {
		t.Error("extracted content still contains footer chrome 'Posted 1 day ago'")
	}
	// The actual job description must be present.
	if !strings.Contains(content, "Senior Backend Engineer") && !strings.Contains(content, "About the Role") {
		t.Errorf("extracted content does not contain expected job description text; content[:300] = %q",
			content[:min(len(content), 300)])
	}
}

// TestWWRResolveBodyConversion verifies the full body-to-description path:
// resolveWWRBody processes a page body with chrome into structured HTML and
// plain text without raw tags or chrome text.
func TestWWRResolveBodyConversion(t *testing.T) {
	body := readFixture(t, "testdata/wwr_job_body.html")

	descHTML, descPlain := resolveWWRBody(body)

	if descHTML == "" {
		t.Fatal("descriptionHTML is empty")
	}
	if strings.Contains(descPlain, "<") {
		t.Errorf("plain description contains raw HTML tags: %q", descPlain[:min(len(descPlain), 200)])
	}
	// Plain description must contain some job description content.
	if !strings.Contains(descPlain, "Senior Backend Engineer") &&
		!strings.Contains(descPlain, "About the Role") &&
		!strings.Contains(descPlain, "Responsibilities") {
		t.Errorf("plain description does not contain expected content: %q", descPlain[:min(len(descPlain), 300)])
	}
}

// readFixture reads the contents of a testdata file and returns it as a string.
func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return string(data)
}
