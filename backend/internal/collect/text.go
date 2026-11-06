package collect

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	spacesTabsRe = regexp.MustCompile(`[ \t]+`)
	blankLinesRe = regexp.MustCompile(`\n\s*\n+`)
)

// CleanHTML strips non-content tags from an HTML fragment, inserts newlines before
// block elements and collapses whitespace, mirroring the TypeScript cleanHtml helper.
func CleanHTML(fragment string) string { return cleanHTML(fragment) }

// cleanHTML mirrors the TypeScript cleanHtml: it strips non-content tags, inserts a newline
// before block elements, takes the body text, and collapses whitespace.
func cleanHTML(fragment string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(fragment))
	if err != nil {
		return ""
	}
	doc.Find("script,style,nav,footer,header,noscript,svg").Remove()
	doc.Find("p,br,div,li,h1,h2,h3,section").Each(func(_ int, s *goquery.Selection) {
		s.PrependHtml("\n")
	})
	txt := doc.Find("body").Text()
	txt = spacesTabsRe.ReplaceAllString(txt, " ")
	txt = blankLinesRe.ReplaceAllString(txt, "\n\n")
	return strings.TrimSpace(txt)
}

// text mirrors the TypeScript text() helper: cleanHtml of the fragment wrapped in a body.
func text(s string) string {
	return cleanHTML("<body>" + s + "</body>")
}

// abs resolves u against base, returning the absolute HTTP(S) URL or "" (the empty string stands
// for the TypeScript null).
func abs(u, base string) string {
	if u == "" {
		return ""
	}
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(u)
	if err != nil {
		return ""
	}
	v := b.ResolveReference(ref)
	if v.Scheme != "http" && v.Scheme != "https" {
		return ""
	}
	return v.String()
}
