package vacancyai

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// normRune folds the typographic variants a model (or a job board) swaps freely into one plain
// form, so a quote still counts as an excerpt when only such a variant differs. It returns 0 for a
// rune that carries no text (a zero-width mark or a soft hyphen).
func normRune(r rune) rune {
	switch r {
	case '\u2018', '\u2019', '\u201A', '\u201B', '\u2032', '\u02BC', '`', '\u00B4':
		return '\''
	case '\u201C', '\u201D', '\u201E', '\u201F', '\u2033', '\u00AB', '\u00BB':
		return '"'
	case '\u2010', '\u2011', '\u2012', '\u2013', '\u2014', '\u2015', '\u2212':
		return '-'
	case '\u200B', '\u200C', '\u200D', '\u2060', '\uFEFF', '\u00AD':
		return 0
	}
	if unicode.IsSpace(r) {
		return ' '
	}
	return r
}

// foldedText is a text reduced to normRune form with runs of whitespace collapsed to one space,
// plus the byte range of the original text each folded rune came from.
type foldedText struct {
	runes  []rune
	starts []int
	ends   []int
}

func fold(text string) foldedText {
	var f foldedText
	prevSpace := true // drops leading whitespace
	for i, r := range text {
		n := normRune(r)
		if n == 0 {
			continue
		}
		size := utf8.RuneLen(r)
		if size < 0 {
			size = 1
		}
		if n == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		f.runes = append(f.runes, n)
		f.starts = append(f.starts, i)
		f.ends = append(f.ends, i+size)
	}
	for len(f.runes) > 0 && f.runes[len(f.runes)-1] == ' ' {
		f.runes = f.runes[:len(f.runes)-1]
		f.starts = f.starts[:len(f.starts)-1]
		f.ends = f.ends[:len(f.ends)-1]
	}
	return f
}

// index finds needle in the folded text and returns the folded rune offset, or -1.
func (f foldedText) index(needle []rune) int {
	if len(needle) == 0 || len(needle) > len(f.runes) {
		return -1
	}
	for i := 0; i+len(needle) <= len(f.runes); i++ {
		if f.runes[i] != needle[0] {
			continue
		}
		ok := true
		for j := 1; j < len(needle); j++ {
			if f.runes[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// groundQuote resolves a model quote to an exact excerpt of the source. Typographic quotes and
// apostrophes, dashes, non-breaking and repeated spaces and zero-width marks are compared as their
// plain forms; the words themselves must match character for character, case included. It returns
// the excerpt as it appears in the source, so what is stored is always the original text, and
// false when the quote cannot be found or is too short to prove anything.
func groundQuote(source, quote string) (string, bool) {
	needle := fold(quote).runes
	if len(needle) < minQuoteRunes {
		return "", false
	}
	hay := fold(source)
	at := hay.index(needle)
	if at < 0 {
		return "", false
	}
	return strings.TrimSpace(source[hay.starts[at]:hay.ends[at+len(needle)-1]]), true
}
