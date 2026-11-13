package source

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/fetch"
)

// newWeb3CareersFixture returns a Web3CareersSource whose HTTP client always
// responds with the contents of fixturePath, regardless of the requested URL.
// fixedBodyTransport is defined in greenhouse_test.go (same package).
func newWeb3CareersFixture(t *testing.T, fixturePath string) *Web3CareersSource {
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
	return NewWeb3CareersSource(c)
}

// TestWeb3CareerRegistryPattern verifies that Web3CareerPattern routes both
// web3.career (vacancy links) and web3.careers (config URL) to a
// *Web3CareersSource and not to the generic fallback.
func TestWeb3CareerRegistryPattern(t *testing.T) {
	c, err := fetch.New(fetch.Options{AllowPrivate: true})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	fallback := NewGenericSource(c)
	w3 := NewWeb3CareersSource(c)

	reg := NewRegistry(fallback)
	reg.Register(Web3CareerPattern, w3)

	for _, host := range []string{"web3.career", "web3.careers"} {
		got := reg.For(host)
		if _, ok := got.(*Web3CareersSource); !ok {
			t.Errorf("reg.For(%q) = %T, want *Web3CareersSource", host, got)
		}
	}

	// Generic fallback must still fire for unrelated hosts.
	if _, ok := reg.For("jobs.lever.co").(*GenericSource); !ok {
		t.Error("reg.For(jobs.lever.co) should fall back to *GenericSource")
	}
}

// TestWeb3CareersDiscoverExtractsAnchorLinks verifies that Discover extracts
// individual job links from the listing page by matching the web3.career URL
// pattern /<slug>/<numeric-id-5+digits>. It must:
//   - return exactly the 3 unique job refs from the fixture (not 4, since one
//     is a duplicate of the first)
//   - exclude category and filter links (/web3-jobs, /companies, /salary, etc.)
//   - exclude short numeric paths with fewer than 5 digits
//   - set DescriptionStatus = "pending body fetch" on every ref (descriptions
//     are always read from the individual page body, not from the listing)
func TestWeb3CareersDiscoverExtractsAnchorLinks(t *testing.T) {
	src := newWeb3CareersFixture(t, "testdata/web3career_listing.html")
	refs, err := src.Discover(context.Background(), "https://web3.career")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 3 {
		t.Fatalf("Discover returned %d refs, want 3 (3 unique job pages, 1 duplicate, category links excluded)", len(refs))
	}
	wantURLs := map[string]bool{
		"https://web3.career/senior-go-engineer-acme-protocol/155406": true,
		"https://web3.career/blockchain-data-analyst-ether-fi/155407":  true,
		"https://web3.career/smart-contract-auditor-trailofbits/155408": true,
	}
	for _, ref := range refs {
		if !wantURLs[ref.URL] {
			t.Errorf("unexpected ref URL %q", ref.URL)
		}
		if ref.Partial.Description != "" {
			t.Errorf("ref %q: Description must be empty at discovery stage", ref.URL)
		}
		if ref.Partial.DescriptionHTML != "" {
			t.Errorf("ref %q: DescriptionHTML must be empty at discovery stage", ref.URL)
		}
		if ref.Partial.DescriptionStatus != "pending body fetch" {
			t.Errorf("ref %q: DescriptionStatus = %q, want \"pending body fetch\"", ref.URL, ref.Partial.DescriptionStatus)
		}
	}
}

// TestWeb3CareersResolveExtractsContentOnly verifies that Resolve extracts
// description from the content sections ([data-qa=job-description],
// [data-qa=posting-requirements], [data-qa=closing-description]) only —
// excluding the job header chrome ("Apply", "Join talent pool", timestamps)
// that precedes those sections on the page.
//
// The fixture is a real saved web3.career page. The page redirected from the
// original URL and landed on a Member of People Experience role at Anchorage
// Digital. All the properties below are properties of that real page.
func TestWeb3CareersResolveExtractsContentOnly(t *testing.T) {
	src := newWeb3CareersFixture(t, "testdata/web3career_job.html")
	ref := Ref{URL: "https://web3.career/member-of-people-experience-anchorage/155407"}

	j, err := src.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// description_html must be set.
	if j.DescriptionHTML == "" {
		t.Fatal("DescriptionHTML is empty; Resolve did not read the body")
	}

	// description_html must start with a tag — no bare text from the page chrome.
	if len(j.DescriptionHTML) > 0 && j.DescriptionHTML[0] != '<' {
		t.Errorf("DescriptionHTML starts with bare text (not a tag):\n%q",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 200)])
	}

	// Page chrome must not appear in the extracted description.
	for _, chrome := range []string{"Join talent pool", "Apply"} {
		if strings.Contains(j.DescriptionHTML, chrome) {
			t.Errorf("DescriptionHTML contains page chrome %q", chrome)
		}
		if strings.Contains(j.Description, chrome) {
			t.Errorf("Description contains page chrome %q", chrome)
		}
	}

	// No double-encoded entity.
	if strings.Contains(j.Description, "amp;amp") || strings.Contains(j.DescriptionHTML, "amp;amp") {
		t.Errorf("double-encoded entity &amp;amp; found in output")
	}

	// No raw HTML tags in the plain description.
	for _, tag := range []string{"<p>", "<li>", "<h3>", "<ul>", "<strong>"} {
		if strings.Contains(j.Description, tag) {
			t.Errorf("plain description contains raw tag %q", tag)
		}
	}

	// description_html must contain at least <li> (the requirements sections on
	// this page are bulleted lists) and <strong> (the opening paragraph uses bold).
	if !strings.Contains(j.DescriptionHTML, "<li>") {
		t.Errorf("DescriptionHTML contains no <li>; list markup not preserved:\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}
	if !strings.Contains(j.DescriptionHTML, "<strong>") {
		t.Errorf("DescriptionHTML contains no <strong>; bold markup not preserved:\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}

	// Apostrophes must survive (the page has "industry's", "Anchorage Digital's").
	if !strings.ContainsRune(j.Description, '\'') {
		t.Errorf("Description has no apostrophes; they were stripped:\n%q",
			j.Description[:min(len(j.Description), 400)])
	}
}

// TestWeb3CareersResolveNativeLayout verifies that Resolve correctly handles
// native web3.career job pages (jobs posted directly on the site, not via
// Greenhouse). These pages use a .trix-content block for the description and
// embed a Bondex "Join talent pool" widget at the end of that block. The
// widget must be excluded from the extracted description.
//
// The fixture is a real saved native web3.career page (Junior Market Analyst
// at MEYA HOLDINGS LTD). It has h3, ul/li structure and the Bondex widget.
func TestWeb3CareersResolveNativeLayout(t *testing.T) {
	src := newWeb3CareersFixture(t, "testdata/web3career_job_native.html")
	ref := Ref{URL: "https://web3.career/junior-market-analyst-remote-meya-holdings-ltd/155983"}

	j, err := src.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if j.DescriptionHTML == "" {
		t.Fatal("DescriptionHTML is empty; Resolve did not extract from the native page")
	}

	// Output must start with a tag (no bare text from page chrome at the top).
	if len(j.DescriptionHTML) > 0 && j.DescriptionHTML[0] != '<' {
		t.Errorf("DescriptionHTML starts with bare text:\n%q",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 200)])
	}

	// Bondex widget must be excluded.
	if strings.Contains(j.DescriptionHTML, "Join talent pool") {
		t.Errorf("DescriptionHTML contains Bondex widget text \"Join talent pool\":\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}
	if strings.Contains(j.Description, "Join talent pool") {
		t.Errorf("plain Description contains Bondex widget text \"Join talent pool\"")
	}

	// No amp;amp double-encoding.
	if strings.Contains(j.DescriptionHTML, "amp;amp") {
		t.Errorf("double-encoded entity found in DescriptionHTML")
	}

	// No raw HTML tags in the plain description.
	for _, tag := range []string{"<p>", "<li>", "<h3>", "<ul>"} {
		if strings.Contains(j.Description, tag) {
			t.Errorf("plain description contains raw tag %q", tag)
		}
	}

	// The description must have the expected structure from the fixture:
	// h3 headings (e.g. "Key Responsibilities") and list items.
	if !strings.Contains(j.DescriptionHTML, "<h3>") {
		t.Errorf("DescriptionHTML has no <h3>; native page has heading sections:\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}
	if !strings.Contains(j.DescriptionHTML, "<li>") {
		t.Errorf("DescriptionHTML has no <li>; native page has lists:\n%s",
			j.DescriptionHTML[:min(len(j.DescriptionHTML), 400)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
