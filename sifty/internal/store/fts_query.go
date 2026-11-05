package store

import (
	"strings"
	"unicode"
)

// BuildFTSQuery constructs a sanitised FTS5 MATCH query from a list of search terms.
//
// Each term is wrapped in double-quotes (phrase match) so that multi-word phrases like
// "Fractional CTO" match as a phrase rather than as independent tokens. FTS5 special
// characters are stripped from the content of each phrase before quoting. Terms that
// collapse to empty after stripping are silently dropped. Duplicate terms (after case
// folding) are deduplicated. The phrases are joined with OR.
//
// The resulting string is safe to pass as the value parameter of:
//
//	SELECT bm25(jobs_fts) FROM jobs_fts WHERE jobs_fts MATCH ?
//
// An empty return value ("") means no terms survived sanitisation; callers should treat
// that as "no match constraint" and avoid passing it to MATCH.
func BuildFTSQuery(terms []string) string {
	seen := make(map[string]bool)
	var phrases []string
	for _, raw := range terms {
		clean := sanitiseFTS5(raw)
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if seen[key] {
			continue
		}
		seen[key] = true
		phrases = append(phrases, `"`+clean+`"`)
	}
	return strings.Join(phrases, " OR ")
}

// sanitiseFTS5 strips characters that carry special meaning in an FTS5 query string.
// The stripped characters are: " ' * ^ ( ) + - \ and all ASCII control characters.
// Runs of whitespace are collapsed to a single space and the result is trimmed.
func sanitiseFTS5(s string) string {
	var b strings.Builder
	prevSpace := true // treat leading edge as "had space" to avoid a leading space
	for _, r := range s {
		switch {
		case r == '-':
			// Hyphen doubles as a word separator in compound terms (half-time → half time);
			// replace with a space.
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		case r == '"' || r == '\'' || r == '*' || r == '^' ||
			r == '(' || r == ')' || r == '+' || r == '\\':
			// FTS5 operator that is not a word separator — strip silently.
		case unicode.IsControl(r) || r == '\n' || r == '\r' || r == '\t':
			// Control character — treat as space
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		case unicode.IsSpace(r):
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		default:
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimRight(b.String(), " ")
}
