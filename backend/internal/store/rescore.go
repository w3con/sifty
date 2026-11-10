package store

import (
	"database/sql"
	"fmt"
)

// Rescore recomputes the match_score column for every role in the jobs table using BM25
// relevance from the FTS5 index. The score is derived by:
//
//  1. Running the given FTS5 MATCH query against jobs_fts. SQLite's bm25() returns a
//     negative value where a more negative value means a better match (the sign is
//     reversed relative to the conventional BM25 definition).
//
//  2. Negating the bm25() output so that a higher value means a better match.
//
//  3. Scaling the negated scores linearly to [0, 100] using min-max normalisation across
//     the current corpus. A role that matches no term receives a score of 0 regardless.
//
// When matchQuery is empty, all match_score values are reset to 0 and the function returns
// without querying the FTS index.
func (s *Store) Rescore(matchQuery string) error {
	return s.RescoreForProfile(matchQuery, 0)
}

// RescoreForProfile is Rescore with the confirmed profile revision that produced the query
// recorded on every role. Reads compare that revision with the current stored profile, so an
// old score is never presented as current after a profile save or a failed refresh.
func (s *Store) RescoreForProfile(matchQuery string, profileRevision int) error {
	if matchQuery == "" {
		if _, err := s.DB.Exec("UPDATE jobs SET match_score=0, match_profile_revision=?", profileRevision); err != nil {
			return fmt.Errorf("rescore: reset scores: %w", err)
		}
		return nil
	}

	// Step 1: collect raw BM25 scores for every role that has an FTS entry.
	// bm25() returns negative values; we negate so that higher = better.
	rows, err := s.DB.Query(
		"SELECT rowid, -bm25(jobs_fts) AS raw FROM jobs_fts WHERE jobs_fts MATCH ?",
		matchQuery,
	)
	if err != nil {
		return fmt.Errorf("rescore: BM25 query: %w", err)
	}
	type scored struct {
		id  int64
		raw float64
	}
	var matched []scored
	var minRaw, maxRaw float64
	first := true
	for rows.Next() {
		var sc scored
		if err := rows.Scan(&sc.id, &sc.raw); err != nil {
			rows.Close()
			return fmt.Errorf("rescore: scan: %w", err)
		}
		matched = append(matched, sc)
		if first || sc.raw < minRaw {
			minRaw = sc.raw
		}
		if first || sc.raw > maxRaw {
			maxRaw = sc.raw
		}
		first = false
	}
	rows.Close()

	// Step 2: build a map of id → scaled score (0–100).
	// Scale to [1, 100] for matched roles (unmatched roles stay at 0, set below).
	// The minimum matched role gets 1 so that any match is distinguishable from no match;
	// the maximum matched role gets 100.
	scores := make(map[int64]int64, len(matched))
	span := maxRaw - minRaw
	for _, sc := range matched {
		var v int64
		if span > 0 {
			v = int64((sc.raw-minRaw)/span*99+0.5) + 1 // rounds to [1, 100]
		} else {
			v = 100 // single matched role, or all roles have equal BM25 score
		}
		scores[sc.id] = v
	}

	// Step 3: reset all scores to 0 first so unmatched roles get 0, then apply the scaled
	// scores in one transaction.
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("rescore: begin tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Exec("UPDATE jobs SET match_score=0, match_profile_revision=?", profileRevision); err != nil {
		return fmt.Errorf("rescore: reset: %w", err)
	}

	for id, score := range scores {
		if _, err := tx.Exec("UPDATE jobs SET match_score=? WHERE id=?", score, id); err != nil {
			return fmt.Errorf("rescore: update id=%d: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("rescore: commit: %w", err)
	}
	tx = nil
	return nil
}

// FastRankingRevision reports the profile revision attribution currently stored on jobs.
// known=false means there are no roles to rank; uniform=false means roles carry different
// revisions (for example a new role inserted after the last rescore). Callers must not present
// either case as a current score.
func (s *Store) FastRankingRevision() (revision int, known, uniform bool, err error) {
	var count int
	var min, max sql.NullInt64
	if err := s.DB.QueryRow(
		"SELECT COUNT(*), MIN(match_profile_revision), MAX(match_profile_revision) FROM jobs").
		Scan(&count, &min, &max); err != nil {
		return 0, false, false, fmt.Errorf("fast ranking state: %w", err)
	}
	if count == 0 {
		return 0, false, false, nil
	}
	if !min.Valid || !max.Valid {
		return 0, true, false, nil
	}
	return int(min.Int64), true, min.Int64 == max.Int64, nil
}
