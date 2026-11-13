package store

import (
	"fmt"
	"strings"

	"github.com/w3con/sifty/internal/collect"
)

// BackfillCounts is the result of BackfillDescriptionHTML.
type BackfillCounts struct {
	// Converted is the number of rows where description_html was set.
	Converted int
	// NeedsRefetch is the number of rows additionally marked needs_refetch=1.
	// These are "glued" rows: long single-paragraph descriptions without any
	// line structure that likely represent multiple concatenated sections.
	NeedsRefetch int
	// Skipped is the number of rows not processed (description empty or
	// description_html already set).
	Skipped int
}

// BackfillDescriptionHTML fills description_html for every vacancy row that has
// none. It is idempotent: rows that already have description_html are skipped.
// It never touches the description column and never makes network requests.
//
// For glued rows (plain text without literal HTML tags, without newlines, and at
// least 300 bytes — the inventory threshold for "glued"), it also sets
// needs_refetch=1 so the refetch path will recover the full HTML when the vacancy
// is opened or a Refresh is triggered.
//
// Rows that already contain real HTML tags (e.g. LinkedIn ATS rows with <p>,
// <strong>, etc.) receive description_html without the needs_refetch mark: their
// description is already structured; marking them would trigger a fetch to a
// protected page and change the description hash, invalidating stored assessment
// results. Escaped markup such as &lt;p&gt; has no literal tag and remains marked.
//
// Hidden failure caught by tests: a backfill that changes description would
// invalidate every stored assessment hash and trigger re-extraction of ~2800
// vacancies. This function only sets description_html.
func (s *Store) BackfillDescriptionHTML() (BackfillCounts, error) {
	rows, err := s.DB.Query(
		"SELECT id, description FROM jobs WHERE description_html IS NULL AND description != ''")
	if err != nil {
		return BackfillCounts{}, fmt.Errorf("backfill query: %w", err)
	}
	type pending struct {
		id   int64
		desc string
	}
	var work []pending
	for rows.Next() {
		var r pending
		if err := rows.Scan(&r.id, &r.desc); err != nil {
			rows.Close()
			return BackfillCounts{}, fmt.Errorf("backfill scan: %w", err)
		}
		work = append(work, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return BackfillCounts{}, fmt.Errorf("backfill iterate: %w", err)
	}

	var counts BackfillCounts
	for _, r := range work {
		if len(strings.TrimSpace(r.desc)) < 10 {
			counts.Skipped++
			continue
		}
		h := collect.StructuredHTML(r.desc)
		if h == "" {
			// Fall back to wrapping in a single paragraph.
			h = "<p>" + r.desc + "</p>"
		}
		// Heuristic for glued plain text: no literal HTML tags (no '<' followed by
		// a letter or '/'), no newlines, and at least 300 bytes (the card's inventory
		// threshold). Rows with real HTML tags are already structured and must not be
		// marked: a successful refetch would change the description hash and trigger
		// model re-reads of correctly-structured vacancies.
		isGlued := !strings.Contains(r.desc, "\n") && len(r.desc) >= 300 && !descHasHTMLTag(r.desc)
		refetch := 0
		if isGlued {
			refetch = 1
			counts.NeedsRefetch++
		}
		if _, err := s.DB.Exec(
			"UPDATE jobs SET description_html=?, needs_refetch=? WHERE id=?",
			h, refetch, r.id,
		); err != nil {
			return counts, fmt.Errorf("backfill update job %d: %w", r.id, err)
		}
		counts.Converted++
	}
	return counts, nil
}

// descHasHTMLTag reports whether s contains a literal HTML tag opener: '<'
// followed by an ASCII letter or '/'. This distinguishes plain text and escaped
// HTML (such as "&lt;p&gt;") from text that is already structured HTML.
func descHasHTMLTag(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '<' {
			next := s[i+1]
			if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '/' {
				return true
			}
		}
	}
	return false
}
