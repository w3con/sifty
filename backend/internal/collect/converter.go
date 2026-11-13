package collect

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/atom"
)

// inputForm is the detected form of a raw description string.
type inputForm int

const (
	formPlain    inputForm = iota // plain text, possibly with bullet/numbered lines
	formHTML                      // real HTML tags
	formEscaped                   // entity-escaped HTML (&lt;p&gt;…)
	formMarkdown                  // Markdown (#, **, []())
)

var (
	// escapedHTMLRe detects entity-escaped HTML: &lt; immediately before a tag-name character.
	escapedHTMLRe = regexp.MustCompile(`&lt;[a-zA-Z/]`)

	// realTagRe detects real HTML tags that appear in listing descriptions.
	realTagRe = regexp.MustCompile(`(?i)<(?:p|br|li|ul|ol|h[1-6]|div|strong|em|b|i|span|section|article|table|tbody|tr|td|th|header|footer|nav|a)\b`)

	// markdownRe detects Markdown headings, bold markers, or inline links.
	markdownRe = regexp.MustCompile(`(?m)^#{1,6} |\*\*[^*\n]+\*\*|\[.+?\]\(https?://`)

	// mdHeadingRe matches a Markdown heading line and captures its level (#...) and text.
	mdHeadingRe = regexp.MustCompile(`^(#{1,6}) +(.+)$`)

	// mdBoldRe matches **text**.
	mdBoldRe = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)

	// mdLinkRe matches [text](url).
	mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

	// mdBulletRe matches a bullet list line: "- " or "* " prefix.
	mdBulletRe = regexp.MustCompile(`^[-*] +(.+)$`)

	// mdNumberedRe matches a numbered list line: "1. " prefix.
	mdNumberedRe = regexp.MustCompile(`^\d+\. +(.+)$`)

	// plainBulletRe matches a plain-text bullet line.
	plainBulletRe = regexp.MustCompile(`^[-*•] +(.+)$`)

	// plainNumberedRe matches a plain-text numbered line.
	plainNumberedRe = regexp.MustCompile(`^\d+\. +(.+)$`)

	// plainColonRe matches a short line ending with ":" that is not a URL.
	plainColonRe = regexp.MustCompile(`^[^:\n]{1,60}[^/]:$`)

	// structuredBlankLinesRe collapses multiple blank lines to one.
	structuredBlankLinesRe = regexp.MustCompile(`\n{3,}`)
)

// detectForm classifies s into one of the four input forms. The order is fixed:
// escaped is checked before html so that partially-decoded strings are handled
// correctly; markdown is checked last because plain text can contain * characters.
func detectForm(s string) inputForm {
	if escapedHTMLRe.MatchString(s) {
		return formEscaped
	}
	if realTagRe.MatchString(s) {
		return formHTML
	}
	if markdownRe.MatchString(s) {
		return formMarkdown
	}
	return formPlain
}

// allowedTag reports whether the HTML tag name is in the sanitised subset.
func allowedTag(name string) bool {
	switch name {
	case "h2", "h3", "h4", "p", "ul", "ol", "li", "strong", "em", "br":
		return true
	}
	return false
}

// StructuredHTML converts a raw description string in any of the four input
// forms into a sanitised HTML fragment containing only the allowed subset:
// h2–h4, p, ul, ol, li, strong, em, br. Every other tag is unwrapped to its
// text. No attributes, no scripts, no styles, no event handlers. Links (a) are
// unwrapped to their link text.
func StructuredHTML(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	var raw string
	switch detectForm(s) {
	case formEscaped:
		raw = structuredFromEscaped(s)
	case formHTML:
		raw = structuredFromHTML(s)
	case formMarkdown:
		raw = structuredFromMarkdown(s)
	default:
		raw = structuredFromPlain(s)
	}
	return normaliseStructuredHTML(raw)
}

// PlainFromStructured converts a sanitised HTML fragment (the output of
// StructuredHTML) to plain text: entities decoded, block elements separated by
// blank lines, list items prefixed with "- " (ul) or "N. " (ol).
func PlainFromStructured(htmlStr string) string {
	if strings.TrimSpace(htmlStr) == "" {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + htmlStr + "</body>"))
	if err != nil {
		return ""
	}
	var sb strings.Builder
	renderPlain(doc.Find("body"), &sb)
	out := strings.TrimSpace(sb.String())
	out = structuredBlankLinesRe.ReplaceAllString(out, "\n\n")
	return out
}

// renderPlain writes the plain text representation of sel's children into sb.
func renderPlain(sel *goquery.Selection, sb *strings.Builder) {
	sel.Contents().Each(func(_ int, n *goquery.Selection) {
		name := goquery.NodeName(n)
		switch name {
		case "#text":
			sb.WriteString(n.Text())
		case "br":
			sb.WriteString("\n")
		case "h2", "h3", "h4":
			sb.WriteString("\n\n")
			sb.WriteString(strings.TrimSpace(n.Text()))
			sb.WriteString("\n\n")
		case "p":
			sb.WriteString("\n\n")
			renderPlain(n, sb)
			sb.WriteString("\n\n")
		case "ul":
			sb.WriteString("\n")
			n.Find("li").Each(func(_ int, li *goquery.Selection) {
				sb.WriteString("- ")
				sb.WriteString(strings.TrimSpace(li.Text()))
				sb.WriteString("\n")
			})
		case "ol":
			sb.WriteString("\n")
			counter := 0
			n.Find("li").Each(func(_ int, li *goquery.Selection) {
				counter++
				sb.WriteString(fmt.Sprintf("%d. ", counter))
				sb.WriteString(strings.TrimSpace(li.Text()))
				sb.WriteString("\n")
			})
		case "strong", "em":
			// Recurse so that <br> inside inline elements produces a line break.
			// Using n.Text() would flatten all descendants and strip <br> positions.
			renderPlain(n, sb)
		}
	})
}

// ── Form-specific converters ────────────────────────────────────────────────

// structuredFromEscaped decodes entity-escaped HTML once, then delegates to
// structuredFromHTML. This handles JSON-LD descriptions that contain &lt;p&gt;
// instead of real tags.
func structuredFromEscaped(s string) string {
	return structuredFromHTML(html.UnescapeString(s))
}

// structuredFromHTML parses the HTML fragment and produces a sanitised subset.
func structuredFromHTML(s string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + s + "</body>"))
	if err != nil {
		return ""
	}
	// Remove non-content elements entirely.
	doc.Find("script, style, nav, footer, header, noscript, svg, meta, link").Remove()
	var sb strings.Builder
	renderSanitised(doc.Find("body"), &sb)
	raw := strings.TrimSpace(sb.String())
	// A second pass wraps consecutive top-level inline runs (bare text, strong,
	// em, br) in <p> so the output is a valid block-level fragment. Without
	// this, divs that were unwrapped by renderSanitised leave their text content
	// floating outside any block element.
	return wrapTopLevelInlines(raw)
}

// wrapTopLevelInlines scans the top-level nodes of htmlStr (which must be a
// sanitised fragment containing only h2–h4, p, ul, ol, li, strong, em, br)
// and groups consecutive inline runs (text nodes, strong, em, br) into <p>
// elements. Block nodes (h*, p, ul, ol) are passed through unchanged.
func wrapTopLevelInlines(htmlStr string) string {
	if htmlStr == "" {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + htmlStr + "</body>"))
	if err != nil {
		return htmlStr
	}

	isTopLevelBlock := func(name string) bool {
		switch name {
		case "h2", "h3", "h4", "ul", "ol", "p":
			return true
		}
		return false
	}

	var out strings.Builder
	var buf strings.Builder // accumulates inline content for the current implicit <p>

	flushBuf := func() {
		content := strings.TrimSpace(buf.String())
		if content != "" {
			out.WriteString("<p>")
			out.WriteString(content)
			out.WriteString("</p>")
		}
		buf.Reset()
	}

	doc.Find("body").Contents().Each(func(_ int, n *goquery.Selection) {
		name := goquery.NodeName(n)
		if isTopLevelBlock(name) {
			// Skip empty block elements: a <p> (or any block) with no text content
			// adds visual whitespace without information and can appear when page
			// chrome divs are unwrapped or when a source ends with a bare label.
			if strings.TrimSpace(n.Text()) == "" {
				return
			}
			flushBuf()
			outer, err := goquery.OuterHtml(n)
			if err == nil {
				out.WriteString(outer)
			}
		} else if name == "#text" {
			text := n.Text()
			if strings.TrimSpace(text) != "" {
				buf.WriteString(text)
			}
		} else {
			// Inline element at top level (strong, em, br): accumulate.
			outer, err := goquery.OuterHtml(n)
			if err == nil {
				buf.WriteString(outer)
			}
		}
	})
	flushBuf()
	return strings.TrimSpace(out.String())
}

// renderSanitised writes the sanitised HTML representation of sel's children.
// Allowed tags are preserved (without attributes); all others are unwrapped.
// Links are unwrapped to their text. Event handler attributes are never kept.
func renderSanitised(sel *goquery.Selection, sb *strings.Builder) {
	sel.Contents().Each(func(_ int, n *goquery.Selection) {
		name := goquery.NodeName(n)
		switch name {
		case "#text":
			sb.WriteString(n.Text())
		case "a":
			// Links: emit link text only.
			sb.WriteString(n.Text())
		case "br":
			sb.WriteString("<br>")
		case "h1":
			// h1 → h2 (map to nearest allowed heading).
			sb.WriteString("<h2>")
			renderSanitised(n, sb)
			sb.WriteString("</h2>")
		case "h5", "h6":
			// h5/h6 → h4 (map to nearest allowed heading).
			sb.WriteString("<h4>")
			renderSanitised(n, sb)
			sb.WriteString("</h4>")
		default:
			if allowedTag(name) {
				// Self-closing br is handled above; other allowed tags have children.
				sb.WriteString("<" + name + ">")
				renderSanitised(n, sb)
				sb.WriteString("</" + name + ">")
			} else if isInlineOrBlock(name) {
				// Unknown/disallowed tag: unwrap to its content.
				renderSanitised(n, sb)
			}
			// Entirely opaque nodes (comments, processing instructions) are dropped.
		}
	})
}

// isInlineOrBlock returns true for HTML element nodes that should be unwrapped
// rather than dropped entirely. Unknown atoms (custom elements) are also unwrapped.
func isInlineOrBlock(name string) bool {
	if name == "" || name == "#text" || name == "#comment" || name == "#document" {
		return false
	}
	a := atom.Lookup([]byte(name))
	// Drop truly non-content atoms (meta, link, noscript, etc. are already removed).
	switch a {
	case atom.Script, atom.Style, atom.Head, atom.Meta, atom.Link,
		atom.Noscript, atom.Svg, atom.Canvas, atom.Video, atom.Audio,
		atom.Object, atom.Embed, atom.Iframe, atom.Frame, atom.Frameset:
		return false
	}
	return true
}

// structuredFromMarkdown converts a Markdown string (Wellfound search API
// format) to sanitised HTML using the allowed subset.
func structuredFromMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	var sb strings.Builder
	type listState struct {
		kind  string // "ul" or "ol"
		items []string
	}
	var lst listState

	flushList := func() {
		if len(lst.items) == 0 {
			return
		}
		sb.WriteString("<" + lst.kind + ">")
		for _, item := range lst.items {
			sb.WriteString("<li>")
			sb.WriteString(item)
			sb.WriteString("</li>")
		}
		sb.WriteString("</" + lst.kind + ">")
		lst = listState{}
	}

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")

		// Blank line.
		if strings.TrimSpace(trimmed) == "" {
			flushList()
			continue
		}

		// Heading.
		if m := mdHeadingRe.FindStringSubmatch(trimmed); m != nil {
			flushList()
			level := len(m[1]) // number of # characters
			tag := mdHeadingTag(level)
			sb.WriteString("<" + tag + ">")
			sb.WriteString(mdInline(m[2]))
			sb.WriteString("</" + tag + ">")
			continue
		}

		// Bullet list item.
		if m := mdBulletRe.FindStringSubmatch(trimmed); m != nil {
			if lst.kind == "ol" {
				flushList()
			}
			lst.kind = "ul"
			lst.items = append(lst.items, mdInline(m[1]))
			continue
		}

		// Numbered list item.
		if m := mdNumberedRe.FindStringSubmatch(trimmed); m != nil {
			if lst.kind == "ul" {
				flushList()
			}
			lst.kind = "ol"
			lst.items = append(lst.items, mdInline(m[1]))
			continue
		}

		// Regular paragraph.
		flushList()
		sb.WriteString("<p>")
		sb.WriteString(mdInline(trimmed))
		sb.WriteString("</p>")
	}
	flushList()
	return strings.TrimSpace(sb.String())
}

// mdHeadingTag returns the allowed heading tag for a Markdown heading level,
// shifting by one so that # (level 1) maps to h2 and ## (level 2) maps to h3.
// All deeper levels map to h4.
func mdHeadingTag(level int) string {
	switch level {
	case 1:
		return "h2"
	case 2:
		return "h3"
	default:
		return "h4"
	}
}

// mdInline converts inline Markdown within a single line: links are unwrapped
// to their text, **bold** is wrapped in <strong>. The result is HTML-escaped
// text with inline elements.
func mdInline(s string) string {
	// Links: [text](url) → text
	s = mdLinkRe.ReplaceAllString(s, "$1")
	// Bold: **text** → <strong>text</strong>
	s = mdBoldRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[2 : len(m)-2]
		return "<strong>" + inner + "</strong>"
	})
	return s
}

// structuredFromPlain converts plain text with optional line structure to
// sanitised HTML. Bullet lines become <ul><li>, numbered lines become <ol><li>,
// short lines ending with ":" become <h3>, and regular lines become <p>.
func structuredFromPlain(s string) string {
	lines := strings.Split(s, "\n")
	var sb strings.Builder
	type listState struct {
		kind  string
		items []string
	}
	var lst listState

	flushList := func() {
		if len(lst.items) == 0 {
			return
		}
		sb.WriteString("<" + lst.kind + ">")
		for _, item := range lst.items {
			sb.WriteString("<li>")
			sb.WriteString(item)
			sb.WriteString("</li>")
		}
		sb.WriteString("</" + lst.kind + ">")
		lst = listState{}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Blank line.
		if trimmed == "" {
			flushList()
			continue
		}

		// Bullet line.
		if m := plainBulletRe.FindStringSubmatch(trimmed); m != nil {
			if lst.kind == "ol" {
				flushList()
			}
			lst.kind = "ul"
			lst.items = append(lst.items, m[1])
			continue
		}

		// Numbered line.
		if m := plainNumberedRe.FindStringSubmatch(trimmed); m != nil {
			if lst.kind == "ul" {
				flushList()
			}
			lst.kind = "ol"
			lst.items = append(lst.items, m[1])
			continue
		}

		// Short line ending with ":" that is not a URL — treat as a heading when the
		// trimmed text looks like one. Fragment sentences (e.g. "is as follows:") that
		// start with a lowercase letter are emitted as paragraphs instead.
		if plainColonRe.MatchString(trimmed) && lst.kind == "" {
			headText := strings.TrimSuffix(trimmed, ":")
			if isHeadingLike(headText) {
				flushList()
				sb.WriteString("<h3>")
				sb.WriteString(headText)
				sb.WriteString("</h3>")
				continue
			}
		}

		// Regular paragraph.
		flushList()
		sb.WriteString("<p>")
		sb.WriteString(trimmed)
		sb.WriteString("</p>")
	}
	flushList()
	return strings.TrimSpace(sb.String())
}
