package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/w3con/sifty/internal/matchcontract"
)

// FitKey names the inputs a stored two-score result was computed from, beyond what the result's
// own provenance carries: the rules in force and the hash of the description the extraction read.
type FitKey struct {
	RulesIdentity   string
	DescriptionHash string
}

// StoredFit is a persisted two-score result with the key it was computed under.
type StoredFit struct {
	Result          matchcontract.FitResult
	ProfileRevision int
	Key             FitKey
	ComputedAt      string
}

// SaveFitResult stores the two-score result of one vacancy, replacing the previous one. It is the
// only writer of match_fit_results and touches no other table.
func (s *Store) SaveFitResult(jobID int64, r matchcontract.FitResult, key FitKey) error {
	if err := r.Validate(); err != nil {
		return fmt.Errorf("fit result for job %d: %w", jobID, err)
	}
	blob, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode fit result for job %d: %w", jobID, err)
	}
	if _, err := s.DB.Exec(`
		INSERT INTO match_fit_results(job_id, profile_revision, rules_identity, description_hash,
			fits_raw, you_raw, schema_version, record, computed_at)
		VALUES(?,?,?,?,?,?,?,?,?)
		ON CONFLICT(job_id) DO UPDATE SET
			profile_revision=excluded.profile_revision, rules_identity=excluded.rules_identity,
			description_hash=excluded.description_hash, fits_raw=excluded.fits_raw, you_raw=excluded.you_raw,
			schema_version=excluded.schema_version, record=excluded.record, computed_at=excluded.computed_at`,
		jobID, r.Provenance.ProfileRevision, key.RulesIdentity, key.DescriptionHash,
		r.FitsYou.RawScore, r.YouFit.RawScore, matchcontract.SchemaVersion, string(blob), now()); err != nil {
		return fmt.Errorf("store fit result for job %d: %w", jobID, err)
	}
	return nil
}

// GetFitResult returns the stored two-score result of one vacancy, or nil when there is none.
func (s *Store) GetFitResult(jobID int64) (*StoredFit, error) {
	row := s.DB.QueryRow(`SELECT profile_revision, rules_identity, description_hash, record, computed_at
		FROM match_fit_results WHERE job_id=?`, jobID)
	f, err := scanFit(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read fit result for job %d: %w", jobID, err)
	}
	return f, nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanFit(row rowScanner) (*StoredFit, error) {
	var f StoredFit
	var blob string
	if err := row.Scan(&f.ProfileRevision, &f.Key.RulesIdentity, &f.Key.DescriptionHash, &blob, &f.ComputedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(blob), &f.Result); err != nil {
		return nil, fmt.Errorf("decode fit result: %w", err)
	}
	return &f, nil
}

// ModelWorkRow is one vacancy as the model-pass worker sees it: the retained text and the state of
// its stored extraction and matches. Absent extraction or matches are the zero values with
// HasExtraction / HasMatches false.
type ModelWorkRow struct {
	Source matchcontract.RetainedSource
	// PublishedAt and PublishedBasis are the stored source publication date and its basis; they
	// give the Newest order.
	PublishedAt    string
	PublishedBasis string
	// FitsRaw is the stored Fits you, or nil when no two-score result exists yet.
	FitsRaw *float64

	HasExtraction    bool
	ExtractionSource string // description hash the stored extraction read
	ExtractorVersion string
	RequirementCount int

	HasMatches      bool
	MatcherVersion  string
	ProfileRevision int
}

// ModelWorkRows lists every vacancy with its retained text and the state of its stored model-pass
// output, in one read and without writing. Deciding what to do with a row is the caller's job.
func (s *Store) ModelWorkRows() ([]ModelWorkRow, error) {
	rows, err := s.DB.Query(`
		SELECT j.id, j.title, j.company, j.description, j.description_status, j.tags, COALESCE(j.published_at,''), COALESCE(j.published_basis,''),
		       f.fits_raw,
		       e.job_id IS NOT NULL, COALESCE(e.source_hash,''), COALESCE(e.extractor_version,''), COALESCE(e.requirement_count,0),
		       m.job_id IS NOT NULL, COALESCE(m.matcher_version,''), COALESCE(m.profile_revision,0)
		FROM jobs j
		LEFT JOIN match_fit_results f ON f.job_id = j.id
		LEFT JOIN match_vacancy_extractions e ON e.job_id = j.id
		LEFT JOIN match_requirement_matches m ON m.job_id = j.id AND m.requirements_hash = e.requirements_hash
		ORDER BY j.id`)
	if err != nil {
		return nil, fmt.Errorf("list model work: %w", err)
	}
	defer rows.Close()
	var out []ModelWorkRow
	for rows.Next() {
		var r ModelWorkRow
		var tags string
		var fits sql.NullFloat64
		if err := rows.Scan(&r.Source.JobID, &r.Source.Title, &r.Source.Company, &r.Source.Description,
			&r.Source.DescriptionStatus, &tags, &r.PublishedAt, &r.PublishedBasis, &fits,
			&r.HasExtraction, &r.ExtractionSource, &r.ExtractorVersion, &r.RequirementCount,
			&r.HasMatches, &r.MatcherVersion, &r.ProfileRevision); err != nil {
			return nil, fmt.Errorf("scan model work: %w", err)
		}
		if fits.Valid {
			v := fits.Float64
			r.FitsRaw = &v
		}
		if tags != "" && tags != "null" {
			_ = json.Unmarshal([]byte(tags), &r.Source.Tags)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountExtractionsSince counts the extractions stored at or after the given UTC timestamp
// (store format). It is the day's progress against a configured daily limit.
func (s *Store) CountExtractionsSince(since string) (int, error) {
	var n int
	if err := s.DB.QueryRow(`SELECT count(*) FROM match_vacancy_extractions WHERE created_at >= ?`, since).Scan(&n); err != nil {
		return 0, fmt.Errorf("count extractions: %w", err)
	}
	return n, nil
}
