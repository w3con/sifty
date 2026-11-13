package collect

import (
	"testing"
)

// defaultTestVocab is a small section vocabulary used in targeted splitter tests.
var defaultTestVocab = []SectionClass{
	{Label: "About", Headings: []string{"about us", "about the company", "about"}},
	{Label: "Responsibilities", Headings: []string{"responsibilities", "key responsibilities", "what you'll do"}},
	{Label: "Requirements", Headings: []string{"requirements", "qualifications"}},
	{Label: "Benefits", Headings: []string{"benefits", "what we offer"}},
}

// TestSplitSectionsBasic verifies the core splitting behaviour: headings become
// section boundaries, pre-heading content gets class "other", vocab matches work.
func TestSplitSectionsBasic(t *testing.T) {
	input := `<p>Intro paragraph before any heading.</p>` +
		`<h3>About Us</h3><p>We are building the future.</p>` +
		`<h3>Key Responsibilities</h3><ul><li>Write Go</li><li>Review PRs</li></ul>` +
		`<h3>Requirements</h3><p>5+ years experience.</p>`

	sections := SplitSections(input, defaultTestVocab)

	if len(sections) != 4 {
		t.Fatalf("want 4 sections, got %d: %+v", len(sections), sections)
	}

	// Pre-heading content.
	s0 := sections[0]
	if s0.Heading != "" {
		t.Errorf("section 0 heading: want empty, got %q", s0.Heading)
	}
	if s0.Class != "other" {
		t.Errorf("section 0 class: want other, got %q", s0.Class)
	}
	if !contains2(s0.Content, "Intro paragraph") {
		t.Errorf("section 0 content: want intro text\ncontent: %s", s0.Content)
	}

	// About Us.
	s1 := sections[1]
	if s1.Heading != "About Us" {
		t.Errorf("section 1 heading: want About Us, got %q", s1.Heading)
	}
	if s1.Class != "About" {
		t.Errorf("section 1 class: want About, got %q", s1.Class)
	}

	// Key Responsibilities → Responsibilities class.
	s2 := sections[2]
	if s2.Class != "Responsibilities" {
		t.Errorf("section 2 class: want Responsibilities, got %q", s2.Class)
	}
	if !contains2(s2.Content, "<li>Write Go</li>") {
		t.Errorf("section 2 content: want list items\ncontent: %s", s2.Content)
	}

	// Requirements.
	s3 := sections[3]
	if s3.Class != "Requirements" {
		t.Errorf("section 3 class: want Requirements, got %q", s3.Class)
	}
}

// TestSplitSectionsUnmatchedHeading verifies that headings not in the vocabulary
// get class "other" with their heading text preserved.
func TestSplitSectionsUnmatchedHeading(t *testing.T) {
	input := `<h3>Compensation</h3><p>$150k–$200k.</p>`
	// "Compensation" is not in defaultTestVocab.
	sections := SplitSections(input, defaultTestVocab)

	if len(sections) != 1 {
		t.Fatalf("want 1 section, got %d", len(sections))
	}
	if sections[0].Class != "other" {
		t.Errorf("unmatched heading class: want other, got %q", sections[0].Class)
	}
	if sections[0].Heading != "Compensation" {
		t.Errorf("unmatched heading: want Compensation, got %q", sections[0].Heading)
	}
}

// TestSplitSectionsTrailingColonStripped verifies that "Requirements:" (with colon)
// matches the same entry as "Requirements" (without colon).
func TestSplitSectionsTrailingColonStripped(t *testing.T) {
	input := `<h3>Requirements:</h3><p>5+ years Go.</p>`
	sections := SplitSections(input, defaultTestVocab)

	if len(sections) != 1 || sections[0].Class != "Requirements" {
		t.Errorf("trailing colon not stripped: got %+v", sections)
	}
}

// TestSplitSectionsNoContentBeforeFirstHeading verifies that a missing pre-heading
// paragraph does not produce an empty "other" section.
func TestSplitSectionsNoContentBeforeFirstHeading(t *testing.T) {
	input := `<h3>About Us</h3><p>We build things.</p>`
	sections := SplitSections(input, defaultTestVocab)

	if len(sections) != 1 {
		t.Fatalf("want 1 section (no empty preamble), got %d: %+v", len(sections), sections)
	}
	if sections[0].Class != "About" {
		t.Errorf("class: want About, got %q", sections[0].Class)
	}
}

// TestSplitSectionsEmptyInput returns nil for empty or whitespace-only input.
func TestSplitSectionsEmptyInput(t *testing.T) {
	if s := SplitSections("", defaultTestVocab); s != nil {
		t.Errorf("empty input: want nil, got %v", s)
	}
	if s := SplitSections("   ", defaultTestVocab); s != nil {
		t.Errorf("whitespace input: want nil, got %v", s)
	}
}

// TestSplitSectionsCaseInsensitive verifies that classification is case-insensitive.
func TestSplitSectionsCaseInsensitive(t *testing.T) {
	input := `<h3>ABOUT US</h3><p>Content.</p>`
	sections := SplitSections(input, defaultTestVocab)
	if len(sections) != 1 || sections[0].Class != "About" {
		t.Errorf("case-insensitive match failed: got %+v", sections)
	}
}

// TestSplitSectionsH2AndH4 verifies that h2 and h4 (not only h3) trigger section splits.
func TestSplitSectionsH2AndH4(t *testing.T) {
	input := `<h2>About Us</h2><p>Company intro.</p>` +
		`<h4>Benefits</h4><p>Great perks.</p>`
	sections := SplitSections(input, defaultTestVocab)
	if len(sections) != 2 {
		t.Fatalf("want 2 sections, got %d", len(sections))
	}
	if sections[0].Class != "About" {
		t.Errorf("h2 section class: want About, got %q", sections[0].Class)
	}
	if sections[1].Class != "Benefits" {
		t.Errorf("h4 section class: want Benefits, got %q", sections[1].Class)
	}
}

// TestSplitSectionsTypographicApostrophe verifies that typographic apostrophes in headings
// match vocabulary entries that use plain apostrophes (and vice-versa).
func TestSplitSectionsTypographicApostrophe(t *testing.T) {
	// ’ is the right single quotation mark ('), common in HTML from word processors.
	input := "<h3>What you’ll do</h3><p>Build things.</p>"
	sections := SplitSections(input, defaultTestVocab)
	if len(sections) != 1 || sections[0].Class != "Responsibilities" {
		t.Errorf("typographic apostrophe: want Responsibilities, got %+v", sections)
	}
}

// TestSplitSectionsPrefixMatch verifies that the prefix rule matches "About <Name>".
var prefixTestVocab = []SectionClass{
	{Label: "About", Headings: []string{"about us"}, Prefix: []string{"about"}},
	{Label: "Responsibilities", Headings: []string{"responsibilities"}},
}

func TestSplitSectionsPrefixMatch(t *testing.T) {
	input := `<h3>About Chainlink</h3><p>Company description.</p>`
	sections := SplitSections(input, prefixTestVocab)
	if len(sections) != 1 || sections[0].Class != "About" {
		t.Errorf("prefix match: want About, got %+v", sections)
	}
}

// TestSplitSectionsContainsMatch verifies that the contains rule matches headings that
// include a target word.
var containsTestVocab = []SectionClass{
	{Label: "Responsibilities", Headings: []string{"responsibilities"}, Contains: []string{"responsibilities"}},
	{Label: "Requirements", Headings: []string{"requirements"}, Contains: []string{"requirements", "qualifications"}},
	{Label: "Benefits", Headings: []string{"benefits"}, Contains: []string{"benefits"}},
}

func TestSplitSectionsContainsMatch(t *testing.T) {
	cases := []struct {
		heading string
		want    string
	}{
		{"Job Responsibilities", "Responsibilities"},
		{"Additional Requirements", "Requirements"},
		{"Preferred Skills and Qualifications", "Requirements"},
		{"Internship Benefits", "Benefits"},
	}
	for _, tc := range cases {
		input := "<h3>" + tc.heading + "</h3><p>Content.</p>"
		sections := SplitSections(input, containsTestVocab)
		if len(sections) != 1 || sections[0].Class != tc.want {
			t.Errorf("contains(%q): want %s, got %+v", tc.heading, tc.want, sections)
		}
	}
}

// TestNormalizeHeading verifies the heading normaliser on representative inputs.
func TestNormalizeHeading(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Requirements:", "requirements"},
		{"What you’ll do", "what you'll do"}, // typographic '
		{"What you‘ll do", "what you'll do"}, // typographic '
		{"Responsibilities & Duties:", "responsibilities and duties"},
		{"  About   Us  ", "about us"},
		{"Benefits.", "benefits"},
	}
	for _, tc := range cases {
		got := normalizeHeading(tc.in)
		if got != tc.want {
			t.Errorf("normalizeHeading(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// contains2 is a simple string-contains helper for use inside the sections_test package
// (the test file uses check/contains from normalise_test.go which are in the same package,
// but to keep them independent we use a local helper here).
func contains2(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
