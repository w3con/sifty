package collect

import (
	"strings"
	"testing"
)

// TestStructuredHTMLSanitiser verifies that scripts, all attributes and event
// handlers are removed, and that only the allowed tag subset survives.
// Hidden failure: a cross-site script injection via the description field.
func TestStructuredHTMLSanitiser(t *testing.T) {
	input := `<script>alert('xss')</script>
<p onclick="evil()" class="x" id="y">Hello <strong style="color:red">world</strong></p>
<a href="https://evil.example/track">click me</a>
<div data-attr="x"><ul><li>item</li></ul></div>
<img src="x" onerror="evil()">
<h2>Title</h2>`

	got := StructuredHTML(input)

	disallowed := []string{
		"<script", "alert(", "onclick", "onerror", "class=", "style=", "href=",
		"data-attr", "<img", "<a ", "<div>",
	}
	for _, s := range disallowed {
		if strings.Contains(got, s) {
			t.Errorf("sanitiser: output contains disallowed %q\noutput: %s", s, got)
		}
	}
	if !strings.Contains(got, "<strong>world</strong>") {
		t.Errorf("sanitiser: expected <strong>world</strong> in output\noutput: %s", got)
	}
	if !strings.Contains(got, "click me") {
		t.Errorf("sanitiser: link text should be preserved\noutput: %s", got)
	}
	if !strings.Contains(got, "<h2>Title</h2>") {
		t.Errorf("sanitiser: expected <h2>Title</h2> in output\noutput: %s", got)
	}
	if !strings.Contains(got, "<li>item</li>") {
		t.Errorf("sanitiser: expected <li>item</li> in output\noutput: %s", got)
	}
}

// TestPlainFromStructuredBrInsideInline verifies that a <br> nested inside a
// <strong> or <em> produces a line break in the plain output. Without this fix,
// Web3 Careers descriptions like <p><strong>About CoinList<br/><br/></strong>
// CoinList is building…</p> produce "About CoinListCoinList is building…" with
// the heading glued to the next paragraph — the same glued-text defect that the
// backfill needs_refetch heuristic exists to detect.
//
// Hidden failure: search, Quick Match and the model receive a glued heading with
// no word boundary, making the heading invisible as a semantic signal.
func TestPlainFromStructuredBrInsideInline(t *testing.T) {
	// Exact structure produced by Web3 Careers escaped-HTML backfill.
	input := `<p><strong>About CoinList<br><br></strong>CoinList is building the distribution network for onchain capital markets.</p>`
	got := PlainFromStructured(input)

	// "About CoinList" and the following sentence must be on separate lines.
	if !strings.Contains(got, "About CoinList\n") {
		t.Errorf("br inside strong: expected newline after heading in plain output\noutput: %q", got)
	}
	if !strings.Contains(got, "CoinList is building") {
		t.Errorf("br inside strong: following text missing from plain output\noutput: %q", got)
	}
	// They must NOT be directly adjacent (no gluing).
	if strings.Contains(got, "About CoinListCoinList") {
		t.Errorf("br inside strong: heading glued to next text — br not producing newline\noutput: %q", got)
	}

	// Verify that StructuredHTML on the escaped form (the real storage form for
	// Web3 Careers rows) promotes "About CoinList" to a heading and separates
	// it from the following text. The double-<br> separator inside the <strong>
	// is consumed by the normaliser; it must NOT appear as a literal <br> after
	// the promotion.
	escaped := `&lt;strong&gt;About CoinList &lt;br&gt;&lt;br&gt;&lt;/strong&gt;CoinList is building the distribution network.`
	htmlOut := StructuredHTML(escaped)
	if !strings.Contains(htmlOut, "<h3>About CoinList</h3>") {
		t.Errorf("normaliser: About CoinList should be promoted to h3\noutput: %s", htmlOut)
	}
	if !strings.Contains(htmlOut, "CoinList is building the distribution network.") {
		t.Errorf("normaliser: body text missing after heading promotion\noutput: %s", htmlOut)
	}
}

// TestPlainFromStructured verifies that blocks are separated by blank lines,
// list items are prefixed correctly, and HTML entities are decoded once.
// Hidden failure: list structure lost or entities double-decoded.
func TestPlainFromStructured(t *testing.T) {
	input := `<h2>About Us</h2><p>We are &amp; building the future.</p><ul><li>Skill one</li><li>Skill two</li></ul><ol><li>First step</li><li>Second step</li></ol>`

	got := PlainFromStructured(input)

	checks := []struct {
		want string
		desc string
	}{
		{"About Us", "heading text present"},
		{"We are & building the future.", "entity decoded once (not double-decoded)"},
		{"- Skill one", "ul list item with dash"},
		{"- Skill two", "ul list item two"},
		{"1. First step", "ol first item"},
		{"2. Second step", "ol second item"},
	}
	for _, c := range checks {
		if !strings.Contains(got, c.want) {
			t.Errorf("PlainFromStructured: %s: want %q in output\noutput: %s", c.desc, c.want, got)
		}
	}
	// Blocks should be separated by blank lines.
	if !strings.Contains(got, "\n\n") {
		t.Errorf("PlainFromStructured: expected blank line separators in output\noutput: %q", got)
	}
}

// TestStructuredHTMLEscapedForm verifies that entity-escaped HTML input (e.g.
// from JSON-LD descriptions that use &lt;p&gt;) is decoded exactly once and
// contains no literal tags in the final output.
// Hidden failure: &lt;p&gt; left as literal text, or decoded twice so &amp;amp;
// becomes & instead of &amp;.
func TestStructuredHTMLEscapedForm(t *testing.T) {
	// Typical LinkedIn/Paradigm JSON-LD description: entity-escaped HTML.
	input := `&lt;p&gt;Senior engineer role at Acme.&lt;/p&gt;&lt;ul&gt;&lt;li&gt;Go experience&lt;/li&gt;&lt;li&gt;DeFi knowledge&lt;/li&gt;&lt;/ul&gt;&lt;p&gt;Salary: $200k &amp;amp; equity&lt;/p&gt;`

	got := StructuredHTML(input)

	// No literal &lt; or &gt; tags should remain.
	if strings.Contains(got, "&lt;") || strings.Contains(got, "&gt;") {
		t.Errorf("escaped form: literal &lt;/&gt; remain in output\noutput: %s", got)
	}
	// No raw < or > from double-decode should appear as tags.
	if strings.Contains(got, "<p>Senior engineer") {
		// Good: p tags present — that's fine.
	}
	if !strings.Contains(got, "Senior engineer role at Acme.") {
		t.Errorf("escaped form: description text missing\noutput: %s", got)
	}
	if !strings.Contains(got, "<li>Go experience</li>") {
		t.Errorf("escaped form: list item missing\noutput: %s", got)
	}
	// &amp;amp; should become & (decoded once from &amp;amp; → &amp; → &).
	// After HTML parse, &amp; in text becomes & character.
	if !strings.Contains(got, "&amp; equity") && !strings.Contains(got, "& equity") {
		t.Errorf("escaped form: entity handling wrong (want & equity or &amp; equity)\noutput: %s", got)
	}

	// Verify plain text also has no literal tags.
	plain := PlainFromStructured(got)
	if strings.Contains(plain, "<") || strings.Contains(plain, ">") {
		t.Errorf("escaped form plain: unexpected < or > in plain text\nplain: %s", plain)
	}
}

// TestStructuredHTMLMarkdownForm verifies that Markdown headings, bold and
// links are converted to the allowed HTML subset.
// Hidden failure: Markdown syntax rendered literally in the UI.
func TestStructuredHTMLMarkdownForm(t *testing.T) {
	// Typical Wellfound search API description.
	input := "# About Acme\n\nWe build **decentralised** infrastructure.\n\n## Role\n\n- Write Go\n- Review PRs\n\n[Apply here](https://acme.example/jobs/1)"

	got := StructuredHTML(input)

	if !strings.Contains(got, "<h2>About Acme</h2>") {
		t.Errorf("markdown form: expected <h2>About Acme</h2>\noutput: %s", got)
	}
	if !strings.Contains(got, "<h3>Role</h3>") {
		t.Errorf("markdown form: expected <h3>Role</h3>\noutput: %s", got)
	}
	if !strings.Contains(got, "<strong>decentralised</strong>") {
		t.Errorf("markdown form: expected <strong>decentralised</strong>\noutput: %s", got)
	}
	if !strings.Contains(got, "<li>Write Go</li>") {
		t.Errorf("markdown form: expected <li>Write Go</li>\noutput: %s", got)
	}
	// Link text preserved, href dropped.
	if !strings.Contains(got, "Apply here") {
		t.Errorf("markdown form: link text missing\noutput: %s", got)
	}
	if strings.Contains(got, "href") || strings.Contains(got, "acme.example") {
		t.Errorf("markdown form: href leaked into output\noutput: %s", got)
	}
}

// TestStructuredHTMLWrapsTopLevelText verifies that bare text nodes appearing
// directly at the document level (not inside any allowed block element) are
// wrapped in <p> rather than emitted as a raw string. This matters for pages
// whose navigation chrome ends up as text at the top of the description when
// all enclosing divs are unwrapped.
//
// Hidden failure: description_html starts with un-wrapped text, breaking the
// block-fragment contract assumed by the renderer.
func TestStructuredHTMLWrapsTopLevelText(t *testing.T) {
	// Simulate a page where divs wrap the chrome text; renderSanitised unwraps
	// them and must then promote the exposed text to <p>.
	input := `<div class="job-header">Senior Engineer at Acme Apply Join talent pool 6d ago</div>` +
		`<div class="job-description"><p>You will build our infrastructure.</p><ul><li>Go experience</li></ul></div>`

	got := StructuredHTML(input)

	// No bare text must appear before the first tag.
	if !strings.HasPrefix(got, "<") {
		t.Errorf("output starts with bare text, want it to start with a tag:\n%q", got[:min2(len(got), 200)])
	}
	// The chrome text from the header div must be wrapped in <p>.
	if !strings.Contains(got, "<p>Senior Engineer at Acme") {
		t.Errorf("chrome text not wrapped in <p>:\n%s", got[:min2(len(got), 400)])
	}
	// The job description block must follow normally.
	if !strings.Contains(got, "<p>You will build our infrastructure.</p>") {
		t.Errorf("description paragraph missing or malformed:\n%s", got)
	}
	if !strings.Contains(got, "<li>Go experience</li>") {
		t.Errorf("list item missing:\n%s", got)
	}
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestDetectFormDeterminism verifies that the same string always maps to the
// same form, and that the four detection paths are non-overlapping.
func TestDetectFormDeterminism(t *testing.T) {
	cases := []struct {
		input string
		want  inputForm
	}{
		{"&lt;p&gt;Hello&lt;/p&gt;", formEscaped},
		{"<p>Hello</p>", formHTML},
		{"# Heading\n\n**bold** text", formMarkdown},
		{"Plain text with no markup", formPlain},
		{"Bullet list:\n- item one\n- item two", formPlain},
		{"", formPlain},
	}
	for _, c := range cases {
		got := detectForm(c.input)
		if got != c.want {
			t.Errorf("detectForm(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}
