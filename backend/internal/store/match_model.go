package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/w3con/sifty/internal/matchcontract"
)

// ErrMatchExtractionMissing means requirement matches were offered for a requirement list that no
// stored extraction of that job holds.
var ErrMatchExtractionMissing = errors.New("no stored extraction holds the requirement list the matches judge")

// SaveVacancyExtraction stores the requirements a model extracted from one retained vacancy
// source. It refuses a record whose source is no longer the saved Job (ErrMatchSourceChanged) and
// otherwise replaces any record under the same key. Rows for an older source or extractor of the
// same job are superseded and removed in the same transaction, so a job keeps one extraction. An
// extraction with no requirements is stored like any other: it is the final answer for that
// source and extractor version.
func (s *Store) SaveVacancyExtraction(e matchcontract.VacancyExtraction) error {
	if err := e.Validate(); err != nil {
		return fmt.Errorf("extraction: %w", err)
	}
	hash, err := e.RequirementsHash()
	if err != nil {
		return err
	}
	blob, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode extraction for job %d: %w", e.JobID, err)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin extraction for job %d: %w", e.JobID, err)
	}
	defer tx.Rollback()
	if err := requireCurrentSource(tx, e.JobID, e.SourceHash); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO match_vacancy_extractions(
			job_id, source_hash, extractor_version, requirements_hash, requirement_count,
			schema_version, record, created_at)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(job_id, source_hash, extractor_version) DO UPDATE SET
			requirements_hash=excluded.requirements_hash,
			requirement_count=excluded.requirement_count,
			schema_version=excluded.schema_version,
			record=excluded.record,
			created_at=excluded.created_at`,
		e.JobID, e.SourceHash, e.ExtractorVersion, hash, len(e.Requirements),
		e.SchemaVersion, string(blob), now()); err != nil {
		return fmt.Errorf("store extraction for job %d: %w", e.JobID, err)
	}
	if _, err := tx.Exec(`
		DELETE FROM match_vacancy_extractions
		WHERE job_id=? AND NOT (source_hash=? AND extractor_version=?)`,
		e.JobID, e.SourceHash, e.ExtractorVersion); err != nil {
		return fmt.Errorf("supersede extractions for job %d: %w", e.JobID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit extraction for job %d: %w", e.JobID, err)
	}
	return nil
}

// GetLatestVacancyExtraction returns the most recently stored extraction for a job regardless of
// source hash or extractor version, or nil when no extraction exists. Used by read-only display
// paths that need any available conditions and requirements without re-running extraction.
func (s *Store) GetLatestVacancyExtraction(jobID int64) (*matchcontract.VacancyExtraction, error) {
	var blob string
	err := s.Read.QueryRow(`
		SELECT record FROM match_vacancy_extractions
		WHERE job_id=? ORDER BY created_at DESC LIMIT 1`,
		jobID).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest extraction for job %d: %w", jobID, err)
	}
	return decodeExtraction(jobID, blob)
}

// GetVacancyExtraction returns the extraction made from exactly this retained source by exactly
// this extractor version, or nil when there is none. A nil answer means the extraction pass must
// run; a record with no requirements is a real answer and is returned.
func (s *Store) GetVacancyExtraction(jobID int64, sourceHash, extractorVersion string) (*matchcontract.VacancyExtraction, error) {
	var blob string
	err := s.DB.QueryRow(`
		SELECT record FROM match_vacancy_extractions
		WHERE job_id=? AND source_hash=? AND extractor_version=?`,
		jobID, sourceHash, extractorVersion).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read extraction for job %d: %w", jobID, err)
	}
	return decodeExtraction(jobID, blob)
}

func decodeExtraction(jobID int64, blob string) (*matchcontract.VacancyExtraction, error) {
	var e matchcontract.VacancyExtraction
	if err := json.Unmarshal([]byte(blob), &e); err != nil {
		return nil, fmt.Errorf("decode extraction for job %d: %w", jobID, err)
	}
	if err := e.Validate(); err != nil {
		return nil, fmt.Errorf("stored extraction for job %d: %w", jobID, err)
	}
	return &e, nil
}

// SaveRequirementMatches stores the verdicts a model gave for one requirement list against one
// profile revision. The matches must judge a requirement list that a stored extraction of the same
// job holds, and that extraction must still be of the saved Job's source; otherwise the write is
// refused. Verdicts for an older requirement list, profile revision or matcher version of the
// same job are superseded in the same transaction.
func (s *Store) SaveRequirementMatches(m matchcontract.RequirementMatches) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin matches for job %d: %w", m.JobID, err)
	}
	defer tx.Rollback()

	var blob string
	err = tx.QueryRow(`
		SELECT record FROM match_vacancy_extractions
		WHERE job_id=? AND requirements_hash=?
		ORDER BY created_at DESC LIMIT 1`, m.JobID, m.RequirementsHash).Scan(&blob)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: job %d", ErrMatchExtractionMissing, m.JobID)
	}
	if err != nil {
		return fmt.Errorf("read extraction for job %d: %w", m.JobID, err)
	}
	e, err := decodeExtraction(m.JobID, blob)
	if err != nil {
		return err
	}
	if err := m.ValidateAgainst(*e); err != nil {
		return fmt.Errorf("matches: %w", err)
	}
	if err := requireCurrentSource(tx, m.JobID, e.SourceHash); err != nil {
		return err
	}
	out, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode matches for job %d: %w", m.JobID, err)
	}
	if _, err := tx.Exec(`
		INSERT INTO match_requirement_matches(
			job_id, requirements_hash, profile_revision, matcher_version, schema_version, record, created_at)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(job_id, requirements_hash, profile_revision, matcher_version) DO UPDATE SET
			schema_version=excluded.schema_version,
			record=excluded.record,
			created_at=excluded.created_at`,
		m.JobID, m.RequirementsHash, m.ProfileRevision, m.MatcherVersion,
		m.SchemaVersion, string(out), now()); err != nil {
		return fmt.Errorf("store matches for job %d: %w", m.JobID, err)
	}
	if _, err := tx.Exec(`
		DELETE FROM match_requirement_matches
		WHERE job_id=? AND NOT (requirements_hash=? AND profile_revision=? AND matcher_version=?)`,
		m.JobID, m.RequirementsHash, m.ProfileRevision, m.MatcherVersion); err != nil {
		return fmt.Errorf("supersede matches for job %d: %w", m.JobID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit matches for job %d: %w", m.JobID, err)
	}
	return nil
}

// GetRequirementMatches returns the verdicts for exactly this requirement list, profile revision
// and matcher version, or nil when there are none. A nil answer means only the matching pass must
// run; the extraction is reused.
func (s *Store) GetRequirementMatches(jobID int64, requirementsHash string, profileRevision int, matcherVersion string) (*matchcontract.RequirementMatches, error) {
	var blob string
	err := s.DB.QueryRow(`
		SELECT record FROM match_requirement_matches
		WHERE job_id=? AND requirements_hash=? AND profile_revision=? AND matcher_version=?`,
		jobID, requirementsHash, profileRevision, matcherVersion).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read matches for job %d: %w", jobID, err)
	}
	var m matchcontract.RequirementMatches
	if err := json.Unmarshal([]byte(blob), &m); err != nil {
		return nil, fmt.Errorf("decode matches for job %d: %w", jobID, err)
	}
	return &m, nil
}

// requireCurrentSource refuses a write whose description hash is no longer the saved Job's. Only
// the description counts: tags, sightings and views change without changing what the model read.
func requireCurrentSource(q queryer, jobID int64, sourceHash string) error {
	current, err := readRetainedSource(q, jobID)
	if err != nil {
		return err
	}
	if matchcontract.HashDescription(current.Description) != sourceHash {
		return fmt.Errorf("%w: job %d", ErrMatchSourceChanged, jobID)
	}
	return nil
}
