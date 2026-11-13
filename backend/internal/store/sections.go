package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/model"
)

// saveSectionsInTx writes the section list for one job inside an open transaction.
// When sections is empty, any existing row is deleted (vacancies without structure
// carry no sections). An existing row is replaced atomically.
func saveSectionsInTx(tx *sql.Tx, jobID int64, sections []model.Section) error {
	if len(sections) == 0 {
		_, err := tx.Exec("DELETE FROM job_sections WHERE job_id=?", jobID)
		return err
	}
	blob, err := json.Marshal(sections)
	if err != nil {
		return fmt.Errorf("marshal sections: %w", err)
	}
	_, err = tx.Exec(
		"INSERT OR REPLACE INTO job_sections(job_id, sections) VALUES(?,?)",
		jobID, string(blob))
	return err
}

// GetSections returns the stored sections for one job. Returns nil when no
// sections have been built yet (description_html absent or not yet processed).
func (s *Store) GetSections(jobID int64) ([]model.Section, error) {
	var blob string
	err := s.DB.QueryRow("SELECT sections FROM job_sections WHERE job_id=?", jobID).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sections for job %d: %w", jobID, err)
	}
	var sections []model.Section
	if err := json.Unmarshal([]byte(blob), &sections); err != nil {
		return nil, fmt.Errorf("unmarshal sections for job %d: %w", jobID, err)
	}
	return sections, nil
}

// SectionCounts is returned by BuildJobSections to summarise what it did.
type SectionCounts struct {
	// Normalised is the number of rows whose description_html was updated by the normaliser.
	Normalised int
	// Sectioned is the number of rows for which sections were written (or re-written).
	Sectioned int
	// Skipped is the number of rows with no description_html (not processed).
	Skipped int
}

// BuildJobSections re-normalises the stored description_html of every vacancy
// that has one, writes sections for it, and returns aggregate counts.
//
// It is idempotent: re-running it produces the same description_html and the
// same sections. It never touches the description column and never makes network
// requests. It is called at startup before the HTTP server accepts connections,
// and again after each collection run to cover newly added vacancies.
//
// The vocab parameter is the section heading vocabulary loaded from configuration.
// Rows that have no description_html are skipped (counted as Skipped).
func (s *Store) BuildJobSections(ctx context.Context, vocab []collect.SectionClass) (SectionCounts, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT id, description_html FROM jobs WHERE description_html IS NOT NULL AND description_html != ''")
	if err != nil {
		return SectionCounts{}, fmt.Errorf("BuildJobSections query: %w", err)
	}
	type pending struct {
		id   int64
		html string
	}
	var work []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.html); err != nil {
			rows.Close()
			return SectionCounts{}, fmt.Errorf("BuildJobSections scan: %w", err)
		}
		work = append(work, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return SectionCounts{}, fmt.Errorf("BuildJobSections iterate: %w", err)
	}

	// Count vacancies without description_html as skipped.
	var totalRows int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs").Scan(&totalRows); err != nil {
		return SectionCounts{}, fmt.Errorf("BuildJobSections count: %w", err)
	}

	var counts SectionCounts
	counts.Skipped = totalRows - len(work)

	for _, p := range work {
		select {
		case <-ctx.Done():
			return counts, ctx.Err()
		default:
		}

		// Re-normalise: apply the structural improvements the normaliser adds.
		// Rows collected before L.5.1 were stored without normalisation; this
		// brings them up to date without changing their description field.
		normalised := collect.NormaliseStructuredHTML(p.html)

		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return counts, fmt.Errorf("BuildJobSections begin tx for job %d: %w", p.id, err)
		}

		// Update description_html if the normaliser changed it.
		if normalised != p.html {
			if _, err := tx.ExecContext(ctx,
				"UPDATE jobs SET description_html=? WHERE id=?", normalised, p.id); err != nil {
				_ = tx.Rollback()
				return counts, fmt.Errorf("BuildJobSections update html job %d: %w", p.id, err)
			}
			counts.Normalised++
		}

		// Build and save sections.
		sections := collect.SplitSections(normalised, vocab)
		if err := saveSectionsInTx(tx, p.id, sections); err != nil {
			_ = tx.Rollback()
			return counts, fmt.Errorf("BuildJobSections save sections job %d: %w", p.id, err)
		}
		if len(sections) > 0 {
			counts.Sectioned++
		}

		if err := tx.Commit(); err != nil {
			return counts, fmt.Errorf("BuildJobSections commit job %d: %w", p.id, err)
		}
	}
	return counts, nil
}

// LogSectionCounts prints a startup log line for BuildJobSections results.
// It prints nothing when no work was done (all rows already sectioned and
// no normalisation needed).
func LogSectionCounts(c SectionCounts) {
	if c.Normalised == 0 && c.Sectioned == 0 {
		return
	}
	log.Printf("sections: normalised=%d sectioned=%d skipped=%d", c.Normalised, c.Sectioned, c.Skipped)
}
