package store

import (
	"fmt"
	"strings"

	"github.com/w3con/sifty/internal/model"
)

// StopWord is one durable rule that hides, and stops collecting, every role whose title
// contains Word as a case-insensitive literal substring.
type StopWord struct {
	ID        int64  `json:"id"`
	Word      string `json:"word"`
	CreatedAt string `json:"createdAt"`
}

// normalizeStopWord trims surrounding whitespace and lower-cases the word. The result is
// what is stored and what is matched, so UNIQUE(word) is case-insensitive in effect.
func normalizeStopWord(word string) string {
	return strings.ToLower(strings.TrimSpace(word))
}

// AddStopWord inserts a stop-word, returning the existing rule when the normalised word is
// already present. An empty word is rejected.
func (s *Store) AddStopWord(word string) (*StopWord, error) {
	w := normalizeStopWord(word)
	if w == "" {
		return nil, fmt.Errorf("stop-word must not be empty")
	}
	if _, err := s.DB.Exec(
		`INSERT OR IGNORE INTO stop_words(word, created_at) VALUES(?, ?)`, w, now(),
	); err != nil {
		return nil, fmt.Errorf("insert stop word: %w", err)
	}
	var sw StopWord
	if err := s.DB.QueryRow(
		`SELECT id, word, created_at FROM stop_words WHERE word=?`, w,
	).Scan(&sw.ID, &sw.Word, &sw.CreatedAt); err != nil {
		return nil, fmt.Errorf("get stop word: %w", err)
	}
	return &sw, nil
}

// ListStopWords returns all stop-words ordered by creation time descending.
func (s *Store) ListStopWords() ([]*StopWord, error) {
	rows, err := s.DB.Query(
		`SELECT id, word, created_at FROM stop_words ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list stop words: %w", err)
	}
	defer rows.Close()
	out := []*StopWord{}
	for rows.Next() {
		var sw StopWord
		if err := rows.Scan(&sw.ID, &sw.Word, &sw.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan stop word: %w", err)
		}
		out = append(out, &sw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stop words: %w", err)
	}
	return out, nil
}

// RemoveStopWord deletes the stop-word with the given id. Returns an error when it does not exist.
func (s *Store) RemoveStopWord(id int64) error {
	res, err := s.DB.Exec(`DELETE FROM stop_words WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete stop word: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("stop word %d not found", id)
	}
	return nil
}

// TitleMatchesStopWords reports whether title contains any of the stop-words as a
// case-insensitive literal substring, returning the first matching word. Exported so the
// collection loop applies exactly the same rule as the store.
func TitleMatchesStopWords(title string, words []*StopWord) (bool, string) {
	return titleMatchesStopWords(title, words)
}

// RoleMatchesExclusions reports whether a role (identified by its title and company) matches
// any of the stop-words as a case-insensitive literal substring of either field.
// It returns the first matching word. The match is tried on title first, then on company,
// so the same stop-word list covers both old title-only exclusions and new company exclusions.
func RoleMatchesExclusions(title, company string, words []*StopWord) (bool, string) {
	if hit, w := titleMatchesStopWords(title, words); hit {
		return true, w
	}
	if hit, w := titleMatchesStopWords(company, words); hit {
		return true, w
	}
	return false, ""
}

func titleMatchesStopWords(title string, words []*StopWord) (bool, string) {
	if len(words) == 0 {
		return false, ""
	}
	lower := strings.ToLower(title)
	for _, w := range words {
		needle := normalizeStopWord(w.Word)
		if needle != "" && strings.Contains(lower, needle) {
			return true, w.Word
		}
	}
	return false, ""
}

// stopMatch is the job-level wrapper used by Get. It delegates to RoleMatchesExclusions so that
// the same stop-word list covers both the job's title and its company name.
func stopMatch(job *model.Job, words []*StopWord) (bool, string) {
	return RoleMatchesExclusions(job.Title, job.Company, words)
}

// StopWordMatchCount returns how many stored roles adding word would newly hide: roles whose
// title or company contains the normalised word and that no active stop-word already hides.
// It creates nothing. Matching runs in Go (not SQL lower()) so it is identical to the matcher
// used by Get.
func (s *Store) StopWordMatchCount(word string) (int, error) {
	w := normalizeStopWord(word)
	if w == "" {
		return 0, nil
	}
	active, err := s.ListStopWords()
	if err != nil {
		return 0, err
	}
	rows, err := s.DB.Query(`SELECT title, company FROM jobs`)
	if err != nil {
		return 0, fmt.Errorf("count stop word matches: %w", err)
	}
	defer rows.Close()
	candidate := []*StopWord{{Word: w}}
	n := 0
	for rows.Next() {
		var title, company string
		if err := rows.Scan(&title, &company); err != nil {
			return 0, fmt.Errorf("scan title/company: %w", err)
		}
		if hit, _ := RoleMatchesExclusions(title, company, candidate); !hit {
			continue
		}
		if already, _ := RoleMatchesExclusions(title, company, active); already {
			continue
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate title/company: %w", err)
	}
	return n, nil
}
