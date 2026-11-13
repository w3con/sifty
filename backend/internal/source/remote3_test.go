package source

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/fetch"
)

// newRemote3Fixture returns a Remote3Source whose HTTP client always responds
// with the contents of fixturePath, regardless of the requested URL.
func newRemote3Fixture(t *testing.T, fixturePath string) *Remote3Source {
	t.Helper()
	body, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}
	c, err := fetch.New(fetch.Options{
		AllowPrivate: true,
		Transport:    &fixedBodyTransport{body: string(body), ct: "text/html; charset=utf-8"},
	})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	return NewRemote3Source(c)
}

// TestRemote3RegistryPattern verifies that Remote3Pattern routes remote3.co to a
// *Remote3Source and not to the generic fallback.
func TestRemote3RegistryPattern(t *testing.T) {
	c, err := fetch.New(fetch.Options{AllowPrivate: true})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	fallback := NewGenericSource(c)
	r3 := NewRemote3Source(c)

	reg := NewRegistry(fallback)
	reg.Register(Remote3Pattern, r3)

	got := reg.For("remote3.co")
	if _, ok := got.(*Remote3Source); !ok {
		t.Errorf("reg.For(remote3.co) = %T, want *Remote3Source", got)
	}
	got = reg.For("www.remote3.co")
	if _, ok := got.(*Remote3Source); !ok {
		t.Errorf("reg.For(www.remote3.co) = %T, want *Remote3Source", got)
	}
	if _, ok := reg.For("remote4.co").(*Remote3Source); ok {
		t.Error("reg.For(remote4.co) must not match Remote3Pattern")
	}
}

// TestRemote3DiscoverUsesRSS verifies that Discover parses the RSS feed
// (testdata/remote3_listing.html, which is a real remote3.co RSS response)
// and returns refs whose URLs all start with /remote-jobs/. It must not
// return refs for the metadata-only RSS description — those descriptions are
// discarded so Resolve always fetches the individual page.
func TestRemote3DiscoverUsesRSS(t *testing.T) {
	src := newRemote3Fixture(t, "testdata/remote3_listing.html")
	refs, err := src.Discover(context.Background(), "https://www.remote3.co/")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("Discover returned 0 refs; fixture must have RSS items")
	}
	for _, ref := range refs {
		if !strings.Contains(ref.URL, "/remote-jobs/") {
			t.Errorf("ref.URL %q does not contain /remote-jobs/", ref.URL)
		}
		if ref.Partial.Description != "" {
			t.Errorf("ref %q: Description must be empty at discovery stage (RSS snippet should be discarded), got %q",
				ref.URL, ref.Partial.Description[:min(len(ref.Partial.Description), 80)])
		}
		if ref.Partial.DescriptionStatus != "pending body fetch" {
			t.Errorf("ref %q: DescriptionStatus = %q, want \"pending body fetch\"",
				ref.URL, ref.Partial.DescriptionStatus)
		}
		if ref.URL == "" {
			t.Error("ref.URL is empty")
		}
	}
}

// TestRemote3ResolveExtractsContentOnly verifies that Resolve extracts the
// description from [class*="RemoteJobs_jobDescription"] on the real Robinhood
// Wallet job page (testdata/remote3_job.html), and that the page chrome
// (Apply button, navigation) is excluded.
//
// The fixture is a real remote3.co vacancy page. The description block has h2
// headings, paragraphs, and lists.
func TestRemote3ResolveExtractsContentOnly(t *testing.T) {
	src := newRemote3Fixture(t, "testdata/remote3_job.html")
	ref := Ref{URL: "https://remote3.co/remote-jobs/job-application-for-software-engineer-wallet-at-robinhood-robinhood"}

	j, err := src.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if j.DescriptionHTML == "" {
		t.Fatal("DescriptionHTML is empty; Resolve did not extract from the page")
	}

	// Output must start with a tag (no bare text from page chrome at the top).
	if len(j.DescriptionHTML) > 0 && j.DescriptionHTML[0] != '<' {
		t.Errorf("DescriptionHTML starts with bare text:\n%q",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 200)])
	}

	// The description must have heading and list markup from the real job page.
	if !strings.Contains(j.DescriptionHTML, "<h2>") {
		t.Errorf("DescriptionHTML has no <h2>; real page has heading sections:\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}
	if !strings.Contains(j.DescriptionHTML, "<li>") {
		t.Errorf("DescriptionHTML has no <li>; real page has bulleted lists:\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}

	// No raw HTML tags in the plain description.
	for _, tag := range []string{"<p>", "<li>", "<h2>", "<ul>"} {
		if strings.Contains(j.Description, tag) {
			t.Errorf("plain description contains raw tag %q", tag)
		}
	}

	// No amp;amp double-encoding.
	if strings.Contains(j.DescriptionHTML, "amp;amp") {
		t.Errorf("double-encoded entity found in DescriptionHTML")
	}

	// "Back to jobs" breadcrumb must be stripped from both outputs.
	if strings.Contains(j.DescriptionHTML, "Back to jobs") {
		t.Errorf("DescriptionHTML contains \"Back to jobs\" breadcrumb after stripping:\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}
	if strings.Contains(j.Description, "Back to jobs") {
		t.Errorf("plain description contains \"Back to jobs\" breadcrumb after stripping")
	}
}
