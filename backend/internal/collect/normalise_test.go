package collect

import (
	"strings"
	"testing"
)

// TestNormaliseStructuredHTML covers the structural improvements applied by the
// normaliser to sanitised HTML fragments. Each sub-test names the real-world
// fragment or rule it exercises.
func TestNormaliseStructuredHTML(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		checks  []check
		noMatch []string
	}{
		{
			// LinkedIn uses <p><strong>Key Responsibilities</strong></p> to mark
			// section headings. The normaliser must promote these to <h3>.
			name:  "LinkedIn bold-only paragraph becomes h3",
			input: `<p><strong>Key Responsibilities</strong></p><ul><li>Write Go</li></ul>`,
			checks: []check{
				contains("<h3>Key Responsibilities</h3>"),
				contains("<li>Write Go</li>"),
			},
			noMatch: []string{"<p><strong>Key Responsibilities"},
		},
		{
			// Web3 Careers (CoinList): <p><strong>About CoinList<br/><br/></strong>
			// CoinList is building…</p>. The <br><br> inside <strong> acts as a
			// separator. Result: heading + paragraph, no blank-line gaps.
			name:  "CoinList leading bold with trailing br becomes h3+p",
			input: `<p><strong>About CoinList<br/><br/></strong>CoinList is building the distribution network for onchain capital markets.</p>`,
			checks: []check{
				contains("<h3>About CoinList</h3>"),
				contains("<p>CoinList is building the distribution network"),
			},
			noMatch: []string{"<br/>", "<br>"},
		},
		{
			// Glued form after wrapTopLevelInlines: <p><strong>About Us</strong>Parity
			// is building the future of Web3.</p>. The strong is the first child and is
			// followed by body text directly.
			name:  "glued heading+body becomes h3+p",
			input: `<p><strong>About Us</strong>Parity is building the future of Web3.</p>`,
			checks: []check{
				contains("<h3>About Us</h3>"),
				contains("<p>Parity is building the future of Web3.</p>"),
			},
		},
		{
			// Double <br> as direct children of <p> (paragraph break in disguise)
			// must split into two separate paragraphs.
			name:  "double br in paragraph becomes two paragraphs",
			input: `<p>First paragraph content.<br/><br/>Second paragraph content.</p>`,
			checks: []check{
				contains("<p>First paragraph content.</p>"),
				contains("<p>Second paragraph content.</p>"),
			},
			noMatch: []string{"<br/>", "<br>"},
		},
		{
			// A single <br> is a soft line break and must be preserved.
			name:  "single br inside paragraph is preserved",
			input: `<p>Line one.<br/>Line two.</p>`,
			checks: []check{
				contains("Line one."),
				contains("Line two."),
				containsEither("<br/>", "<br>"),
			},
		},
		{
			// A bold paragraph whose text ends with a full stop is a sentence,
			// not a heading. It must stay as <p><strong>…</strong></p>.
			// Hidden failure: promotional heuristic triggers on running prose.
			name:  "bold sentence ending with period stays as paragraph",
			input: `<p><strong>CoinList is building the distribution network for onchain capital markets.</strong></p>`,
			checks: []check{
				contains("<p><strong>"),
			},
			noMatch: []string{"<h3>"},
		},
		{
			// A bold text longer than 80 characters is prose, not a heading.
			name:  "bold text over 80 chars stays as paragraph",
			input: `<p><strong>This is a long bold sentence that describes the company mission and goes well past the heading length limit.</strong></p>`,
			checks: []check{
				contains("<p><strong>"),
			},
			noMatch: []string{"<h3>"},
		},
		{
			// A bold paragraph starting with a lowercase letter is a sentence fragment
			// lifted from the middle of a paragraph — it must NOT become a heading.
			// Real-world example: "is as follows" (LinkedIn, 231 vacancies).
			name:  "bold text starting with lowercase stays as paragraph",
			input: `<p><strong>is as follows</strong></p>`,
			checks: []check{
				contains("<p>"),
			},
			noMatch: []string{"<h3>"},
		},
		{
			// Bold items inside <li> are list content — the normaliser must not
			// touch list items at all.
			name:  "bold item inside li is never promoted",
			input: `<ul><li><strong>Go experience</strong></li><li>TypeScript knowledge</li></ul>`,
			checks: []check{
				contains("<li><strong>Go experience</strong></li>"),
				contains("<li>TypeScript knowledge</li>"),
			},
			noMatch: []string{"<h3>"},
		},
		{
			// Existing headings must pass through unchanged (idempotency precondition).
			name:  "existing headings pass through",
			input: `<h3>Key Responsibilities</h3><p>Some text.</p>`,
			checks: []check{
				contains("<h3>Key Responsibilities</h3>"),
				contains("<p>Some text.</p>"),
			},
		},
		{
			// Safety pass: a sentence fragment promoted to h3 by the plain-text converter
			// (e.g. "is as follows:") must be demoted back to a paragraph and merged with
			// the preceding paragraph when that paragraph has no terminal punctuation.
			// Real occurrence: job 227 — "Classification of protected categories\nis as follows:"
			name:  "fragment heading merged with preceding unfinished paragraph",
			input: `<p>Classification of protected categories</p><h3>is as follows</h3>`,
			checks: []check{
				contains("<p>Classification of protected categories is as follows</p>"),
			},
			noMatch: []string{"<h3>", "<h3>is as follows</h3>"},
		},
		{
			// Safety pass: a sentence fragment h3 that follows a sentence-terminal paragraph
			// must be demoted to its own paragraph rather than merged.
			name:  "fragment heading after terminal paragraph becomes own paragraph",
			input: `<p>See the full details below.</p><h3>is as follows</h3>`,
			checks: []check{
				contains("<p>See the full details below.</p>"),
				contains("<p>is as follows</p>"),
			},
			noMatch: []string{"<h3>is as follows</h3>"},
		},
		{
			// Safety pass: uppercase headings (isHeadingLike = true) must pass through
			// even when demoteFragmentHeadings runs — no regression on valid headings.
			name:  "valid heading after non-terminal paragraph is not merged",
			input: `<p>See the requirements</p><h3>Requirements</h3><ul><li>Go</li></ul>`,
			checks: []check{
				contains("<h3>Requirements</h3>"),
				contains("<p>See the requirements</p>"),
			},
			noMatch: []string{"<p>See the requirements Requirements</p>"},
		},
		{
			// A paragraph where the strong is NOT the first child must not be promoted.
			// Hidden failure: heuristic triggers on mid-sentence bold runs.
			name:  "strong not first child — no promotion",
			input: `<p>We build <strong>great</strong> products for everyone.</p>`,
			checks: []check{
				contains("<p>"),
				contains("<strong>great</strong>"),
			},
			noMatch: []string{"<h3>"},
		},
		{
			// Empty paragraphs must be removed.
			name:  "empty paragraph is removed",
			input: `<p></p><p>Real content.</p><p>  </p>`,
			checks: []check{
				contains("<p>Real content.</p>"),
			},
			noMatch: []string{"<p></p>", "<p>  </p>"},
		},
		{
			// Applying the normaliser twice must yield the same result.
			name:  "idempotency: normalising twice gives same result",
			input: `<p><strong>Key Responsibilities</strong></p><p><strong>About Us</strong>We build things.</p>`,
			// The check function verifies this programmatically in the test body below.
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NormaliseStructuredHTML(c.input)
			for _, ch := range c.checks {
				if !ch.fn(got) {
					t.Errorf("%s: want %q in output\noutput: %s", c.name, ch.desc, got)
				}
			}
			for _, bad := range c.noMatch {
				if strings.Contains(got, bad) {
					t.Errorf("%s: must not contain %q\noutput: %s", c.name, bad, got)
				}
			}
		})
	}

	// Idempotency: apply twice, compare results.
	t.Run("idempotency: twice same as once", func(t *testing.T) {
		inputs := []string{
			`<p><strong>Key Responsibilities</strong></p><ul><li>Write Go</li></ul>`,
			`<p><strong>About CoinList<br/><br/></strong>CoinList is building.</p>`,
			`<p>First.<br/><br/>Second.</p>`,
			`<p>We build <strong>great</strong> products.</p>`,
			`<h2>Existing heading</h2><p>Content.</p>`,
			`<p>Classification of protected categories</p><h3>is as follows</h3>`,
			`<p>See the full details below.</p><h3>is as follows</h3>`,
		}
		for _, in := range inputs {
			once := NormaliseStructuredHTML(in)
			twice := NormaliseStructuredHTML(once)
			if once != twice {
				t.Errorf("not idempotent for input %q\nonce:  %s\ntwice: %s", in, once, twice)
			}
		}
	})
}

// TestNormaliseStructuredHTMLViaConverter verifies that StructuredHTML calls the
// normaliser for all input forms, using the LinkedIn and CoinList real patterns.
func TestNormaliseStructuredHTMLViaConverter(t *testing.T) {
	// HTML form: LinkedIn-style bold-only paragraph from jobspy.
	linkedInHTML := `<p><strong>Key Responsibilities</strong></p><ul><li>Lead the team</li></ul>`
	got := StructuredHTML(linkedInHTML)
	if !strings.Contains(got, "<h3>Key Responsibilities</h3>") {
		t.Errorf("HTML form: Key Responsibilities not promoted to h3\noutput: %s", got)
	}

	// Escaped form: Web3 Careers row from backfill (entity-escaped HTML).
	escapedCoinList := `&lt;strong&gt;About This Role&lt;/strong&gt;Join us to build the future.`
	got = StructuredHTML(escapedCoinList)
	if !strings.Contains(got, "<h3>About This Role</h3>") {
		t.Errorf("escaped form: About This Role not promoted to h3\noutput: %s", got)
	}
	if !strings.Contains(got, "Join us to build the future.") {
		t.Errorf("escaped form: body text missing\noutput: %s", got)
	}
}

// check is a named assertion used in table-driven subtests.
type check struct {
	desc string
	fn   func(string) bool
}

func contains(want string) check {
	return check{desc: want, fn: func(s string) bool { return strings.Contains(s, want) }}
}

func containsEither(a, b string) check {
	return check{
		desc: a + " or " + b,
		fn:   func(s string) bool { return strings.Contains(s, a) || strings.Contains(s, b) },
	}
}
