package collect

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/w3con/sifty/internal/model"
)

// SectionClass maps one class label to the heading patterns that belong to it.
// Matching uses normalizeHeading on both the heading and each pattern.
// Prefix, when set, also matches any heading that starts with one of its entries
// followed by a space (e.g. "about" matches "about chainlink").
// Contains, when set, also matches any heading whose words include one of its entries.
type SectionClass struct {
	Label    string
	Headings []string
	Prefix   []string
	Contains []string
}

// SplitSections splits a normalised HTML fragment into ordered sections at h2, h3
// and h4 elements. Content that appears before the first heading is emitted as its
// own section with an empty heading and class "other". Each heading is classified
// against vocab; headings not found in the vocabulary use class "other".
// Returns nil when htmlStr is empty or has no block content.
func SplitSections(htmlStr string, vocab []SectionClass) []model.Section {
	if strings.TrimSpace(htmlStr) == "" {
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + htmlStr + "</body>"))
	if err != nil {
		return nil
	}

	var sections []model.Section
	var curHeading string
	var curClass = "other"
	var curContent strings.Builder

	flush := func() {
		content := strings.TrimSpace(curContent.String())
		if content != "" {
			sections = append(sections, model.Section{
				Heading: curHeading,
				Class:   curClass,
				Content: content,
			})
		}
		curContent.Reset()
	}

	doc.Find("body").Contents().Each(func(_ int, n *goquery.Selection) {
		name := goquery.NodeName(n)
		switch name {
		case "h2", "h3", "h4":
			flush()
			curHeading = strings.TrimSpace(n.Text())
			curClass = classifySection(curHeading, vocab)
		default:
			outer, e := goquery.OuterHtml(n)
			if e == nil {
				curContent.WriteString(outer)
			}
		}
	})
	flush()
	return sections
}

// normalizeHeading normalises a section heading for classification:
// typographic apostrophes (‘, ‘) → plain apostrophe; ampersand → "and";
// commas stripped (allows "In this role, you’ll" to match "in this role you’ll");
// trailing colon, period, question mark, exclamation mark stripped;
// runs of whitespace collapsed; lower-cased.
func normalizeHeading(s string) string {
	// Typographic apostrophes.
	s = strings.ReplaceAll(s, "’", "'") // ’
	s = strings.ReplaceAll(s, "‘", "'") // ‘
	s = strings.ReplaceAll(s, "ʼ", "'") // ʼ
	// Ampersand (with and without spaces).
	s = strings.ReplaceAll(s, " & ", " and ")
	s = strings.ReplaceAll(s, "&", " and ")
	// Commas: "In this role, you’ll" → "In this role you’ll".
	s = strings.ReplaceAll(s, ",", "")
	// Lower-case first so whitespace collapse and trim work on ASCII.
	s = strings.ToLower(strings.TrimSpace(s))
	// Strip trailing colon, period, question mark and exclamation mark.
	s = strings.TrimRight(s, ":.?!")
	s = strings.TrimSpace(s)
	// Collapse internal whitespace.
	var b strings.Builder
	prevSpace := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// classifySection returns the vocabulary class for a heading string.
// Matching is performed on the normalised form of the heading (see normalizeHeading).
// Three levels are tried in order:
//  1. Exact match against each class's Headings list.
//  2. Prefix match: the heading starts with one of a class's Prefix entries followed by a space.
//  3. Contains match: the heading contains one of a class's Contains entries as a whole word.
//
// Returns "other" when no entry matches.
func classifySection(heading string, vocab []SectionClass) string {
	h := normalizeHeading(heading)
	for _, v := range vocab {
		// Level 1: exact match.
		for _, pat := range v.Headings {
			if normalizeHeading(pat) == h {
				return v.Label
			}
		}
	}
	for _, v := range vocab {
		// Level 2: prefix match — heading starts with "<prefix> ".
		for _, pfx := range v.Prefix {
			p := normalizeHeading(pfx)
			if strings.HasPrefix(h, p+" ") {
				return v.Label
			}
		}
	}
	for _, v := range vocab {
		// Level 3: contains match — heading contains the word as a standalone token
		// (preceded and followed by a word boundary: space, start, or end).
		for _, word := range v.Contains {
			w := normalizeHeading(word)
			if containsWord(h, w) {
				return v.Label
			}
		}
	}
	return "other"
}

// containsWord reports whether s contains word as a whole-word match.
// A word is bounded by a space, start-of-string, or end-of-string.
func containsWord(s, word string) bool {
	if s == word {
		return true
	}
	if strings.HasPrefix(s, word+" ") {
		return true
	}
	if strings.HasSuffix(s, " "+word) {
		return true
	}
	return strings.Contains(s, " "+word+" ")
}
