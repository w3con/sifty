package collect

import (
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// headingTermRe matches terminal punctuation that marks a sentence, not a section heading.
var headingTermRe = regexp.MustCompile(`[.!?,;]$`)

// NormaliseStructuredHTML applies structural improvements to a sanitised HTML
// fragment produced by StructuredHTML or loaded from storage:
//   - <p><strong>text</strong></p> where text looks like a heading → <h3>text</h3>
//   - <p><strong>heading</strong>body…</p> → <h3>heading</h3><p>body…</p>
//   - two or more consecutive <br> as direct children of <p> → paragraph split
//   - empty <p> elements removed
//   - list items (li) are never modified
//
// Idempotent: existing headings and clean paragraphs pass through unchanged.
// Never touches description or any field other than the structured HTML fragment.
func NormaliseStructuredHTML(htmlStr string) string {
	return normaliseStructuredHTML(htmlStr)
}

func normaliseStructuredHTML(htmlStr string) string {
	if htmlStr == "" {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + htmlStr + "</body>"))
	if err != nil {
		return htmlStr
	}

	var out strings.Builder
	doc.Find("body").Contents().Each(func(_ int, n *goquery.Selection) {
		name := goquery.NodeName(n)
		switch name {
		case "#text":
			// Top-level text nodes: preserve only if non-empty.
			if strings.TrimSpace(n.Text()) != "" {
				out.WriteString(html.EscapeString(n.Get(0).Data))
			}
		case "p":
			out.WriteString(normaliseP(n))
		default:
			outer, e := goquery.OuterHtml(n)
			if e == nil {
				out.WriteString(outer)
			}
		}
	})
	return demoteFragmentHeadings(strings.TrimSpace(out.String()))
}

// demoteFragmentHeadings is a post-normalisation safety pass. Any h2–h4 node whose
// text fails isHeadingLike (starts with a lowercase letter) is demoted to a plain
// paragraph. When the immediately preceding sibling is a plain paragraph whose text
// does not end with sentence-closing punctuation (., !, ?), the two are merged into
// one paragraph: "<p>previous text demoted text</p>".
// The pass is idempotent: a heading that passes isHeadingLike is never touched.
func demoteFragmentHeadings(htmlStr string) string {
	if htmlStr == "" {
		return htmlStr
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + htmlStr + "</body>"))
	if err != nil {
		return htmlStr
	}

	var parts []string
	prevPText := ""  // text content of the last plain <p> in parts
	prevPIdx := -1   // index into parts of that last plain <p>

	doc.Find("body").Contents().Each(func(_ int, n *goquery.Selection) {
		name := goquery.NodeName(n)
		switch name {
		case "#text":
			raw := n.Get(0).Data
			if strings.TrimSpace(raw) != "" {
				parts = append(parts, html.EscapeString(raw))
				prevPIdx = -1
				prevPText = ""
			}
			// Whitespace-only text nodes between elements do not reset prevP.
		case "p":
			outer, e := goquery.OuterHtml(n)
			if e != nil {
				return
			}
			parts = append(parts, outer)
			pText := strings.TrimSpace(n.Text())
			// Track this paragraph for a potential merge only when normaliseP returned a
			// single <p> (no embedded headings). The heuristic: outer ends with </p> and
			// does not contain an opening heading tag.
			if strings.HasSuffix(outer, "</p>") &&
				!strings.Contains(outer, "<h2>") &&
				!strings.Contains(outer, "<h3>") &&
				!strings.Contains(outer, "<h4>") {
				prevPIdx = len(parts) - 1
				prevPText = pText
			} else {
				prevPIdx = -1
				prevPText = ""
			}
		case "h2", "h3", "h4":
			text := strings.TrimSpace(n.Text())
			if isHeadingLike(text) {
				outer, e := goquery.OuterHtml(n)
				if e == nil {
					parts = append(parts, outer)
				}
				prevPIdx = -1
				prevPText = ""
				return
			}
			// Fragment heading: demote to paragraph, optionally merging with the preceding
			// paragraph when it does not end with sentence-closing punctuation.
			escaped := html.EscapeString(text)
			if prevPIdx >= 0 && !headingTermRe.MatchString(prevPText) {
				p := parts[prevPIdx]
				// p ends with "</p>"; replace it with merged content.
				parts[prevPIdx] = p[:len(p)-4] + " " + escaped + "</p>"
			} else {
				parts = append(parts, "<p>"+escaped+"</p>")
			}
			prevPIdx = -1
			prevPText = ""
		default:
			outer, e := goquery.OuterHtml(n)
			if e == nil {
				parts = append(parts, outer)
			}
			prevPIdx = -1
			prevPText = ""
		}
	})
	return strings.TrimSpace(strings.Join(parts, ""))
}

// normaliseP processes a single <p> element and returns replacement HTML
// (may be one or more elements: <h3>, <p>, or multiple <p> from splits).
func normaliseP(p *goquery.Selection) string {
	// Collect groups of content, splitting when two or more consecutive <br>
	// appear as direct children of this <p>.
	type group struct {
		parts []string
	}
	var groups []group
	cur := group{}
	brRun := 0

	p.Contents().Each(func(_ int, n *goquery.Selection) {
		name := goquery.NodeName(n)
		if name == "br" {
			brRun++
			if brRun == 1 {
				// Keep the first <br> as a line break within the group.
				cur.parts = append(cur.parts, "<br/>")
			}
			if brRun == 2 {
				// Second <br>: remove the tentative first one and flush the group.
				if len(cur.parts) > 0 && cur.parts[len(cur.parts)-1] == "<br/>" {
					cur.parts = cur.parts[:len(cur.parts)-1]
				}
				groups = append(groups, cur)
				cur = group{}
				// brRun stays > 1 so further <br> continue to be absorbed without output.
			}
		} else {
			brRun = 0
			if name == "#text" {
				raw := n.Get(0).Data
				if strings.TrimSpace(raw) != "" {
					cur.parts = append(cur.parts, html.EscapeString(raw))
				}
			} else {
				outer, _ := goquery.OuterHtml(n)
				cur.parts = append(cur.parts, outer)
			}
		}
	})
	groups = append(groups, cur)

	var result strings.Builder
	for _, g := range groups {
		inner := strings.TrimSpace(strings.Join(g.parts, ""))
		if inner == "" {
			continue
		}
		result.WriteString(classifySegmentHTML(inner))
	}
	return result.String()
}

// classifySegmentHTML inspects the structure of an inner HTML segment and
// returns <h3>…</h3>, <h3>…</h3><p>…</p>, or <p>…</p>.
func classifySegmentHTML(inner string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + inner + "</body>"))
	if err != nil {
		return "<p>" + inner + "</p>"
	}
	body := doc.Find("body")

	// Collect meaningful (non-whitespace-only) content nodes.
	meaningful := body.Contents().FilterFunction(func(_ int, n *goquery.Selection) bool {
		if goquery.NodeName(n) == "#text" && strings.TrimSpace(n.Text()) == "" {
			return false
		}
		return true
	})

	if meaningful.Length() == 0 {
		return ""
	}

	first := meaningful.First()
	firstName := goquery.NodeName(first)

	// Pattern A: sole meaningful child is <strong> whose text looks like a heading.
	if meaningful.Length() == 1 && firstName == "strong" {
		headText := strongText(first)
		if isHeadingLike(headText) {
			return "<h3>" + html.EscapeString(headText) + "</h3>"
		}
		return "<p>" + inner + "</p>"
	}

	// Pattern B: leading <strong> (possibly with trailing <br> separators) + body text.
	if firstName == "strong" && meaningful.Length() > 1 {
		headText := strongText(first)
		if isHeadingLike(headText) {
			rest := contentAfterNode(body, first)
			if trimmed := strings.TrimSpace(rest); trimmed != "" {
				return "<h3>" + html.EscapeString(headText) + "</h3><p>" + trimmed + "</p>"
			}
		}
	}

	return "<p>" + inner + "</p>"
}

// strongText returns the text content of a <strong> element, skipping any
// <br> children (trailing separators used in Web3 Careers descriptions).
func strongText(strong *goquery.Selection) string {
	var sb strings.Builder
	strong.Contents().Each(func(_ int, n *goquery.Selection) {
		if goquery.NodeName(n) == "#text" {
			sb.WriteString(n.Get(0).Data)
		}
		// <br> children are intentionally skipped.
	})
	return strings.TrimSpace(sb.String())
}

// isHeadingLike reports whether text looks like a section heading:
// non-empty, at most 80 characters, starts with an uppercase letter, and has
// no terminal sentence punctuation. A lowercase start indicates a sentence
// fragment lifted from the middle of a paragraph by the normaliser.
func isHeadingLike(text string) bool {
	if text == "" || len(text) > 80 {
		return false
	}
	// A heading must start with an uppercase or title-case letter (or a digit).
	// Lowercase start means a sentence continuation, not a heading.
	for _, r := range text {
		if unicode.IsLower(r) {
			return false
		}
		break // only check the first rune
	}
	return !headingTermRe.MatchString(text)
}

// contentAfterNode serializes all content in body that follows the given node.
func contentAfterNode(body *goquery.Selection, after *goquery.Selection) string {
	target := after.Get(0)
	var sb strings.Builder
	found := false
	body.Contents().Each(func(_ int, n *goquery.Selection) {
		if n.Get(0) == target {
			found = true
			return
		}
		if !found {
			return
		}
		name := goquery.NodeName(n)
		if name == "#text" {
			sb.WriteString(html.EscapeString(n.Get(0).Data))
		} else {
			outer, _ := goquery.OuterHtml(n)
			sb.WriteString(outer)
		}
	})
	return sb.String()
}
