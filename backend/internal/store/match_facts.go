package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/w3con/sifty/internal/matchcontract"
)

// ErrMatchSourceChanged means the authoritative saved Job no longer matches the snapshot the
// facts were extracted from. The write is refused; the old fact row is left untouched.
var ErrMatchSourceChanged = errors.New("match source changed since extraction")

// ErrMatchInputNotRetained means facts were saved before their retained input snapshot.
var ErrMatchInputNotRetained = errors.New("retained match input has not been captured")

// CaptureMatchInput reads the authoritative saved Job fields, stores them as the retained
// extraction input and returns the immutable snapshot. The read and write share one SQLite
// transaction so a concurrent source change cannot produce a snapshot written under a
// different hash. It never reads or fetches the network.
func (s *Store) CaptureMatchInput(jobID int64) (matchcontract.RetainedSource, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return matchcontract.RetainedSource{}, fmt.Errorf("begin retained input for job %d: %w", jobID, err)
	}
	defer tx.Rollback()
	src, err := readRetainedSource(tx, jobID)
	if err != nil {
		return matchcontract.RetainedSource{}, err
	}
	if err := src.Validate(); err != nil {
		return matchcontract.RetainedSource{}, fmt.Errorf("job %d retained input: %w", jobID, err)
	}
	hash, err := matchcontract.HashRetainedSource(src)
	if err != nil {
		return matchcontract.RetainedSource{}, err
	}
	blob, err := json.Marshal(src)
	if err != nil {
		return matchcontract.RetainedSource{}, fmt.Errorf("encode retained input: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO match_job_inputs(job_id, raw_source_hash, snapshot, captured_at)
		VALUES(?,?,?,?)
		ON CONFLICT(job_id) DO UPDATE SET
			raw_source_hash=excluded.raw_source_hash,
			snapshot=excluded.snapshot,
			captured_at=excluded.captured_at`,
		jobID, hash, string(blob), now()); err != nil {
		return matchcontract.RetainedSource{}, fmt.Errorf("store retained input for job %d: %w", jobID, err)
	}
	if err := tx.Commit(); err != nil {
		return matchcontract.RetainedSource{}, fmt.Errorf("commit retained input for job %d: %w", jobID, err)
	}
	return src, nil
}

// GetMatchInput returns the stored retained snapshot, or nil when none exists.
func (s *Store) GetMatchInput(jobID int64) (*matchcontract.RetainedSource, error) {
	var blob string
	err := s.DB.QueryRow("SELECT snapshot FROM match_job_inputs WHERE job_id=?", jobID).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read retained input for job %d: %w", jobID, err)
	}
	var src matchcontract.RetainedSource
	if err := json.Unmarshal([]byte(blob), &src); err != nil {
		return nil, fmt.Errorf("decode retained input for job %d: %w", jobID, err)
	}
	return &src, nil
}

// SaveMatchFacts validates facts and stores them only when the current saved Job still hashes
// to the facts' RawSourceHash and a matching retained input exists. It never invalidates or
// rewrites an existing result; that is the later orchestration stage.
func (s *Store) SaveMatchFacts(f matchcontract.VacancyFacts) error {
	if err := f.Validate(); err != nil {
		return fmt.Errorf("facts: %w", err)
	}
	if err := f.VerifyFactsHash(); err != nil {
		return fmt.Errorf("facts: %w", err)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin facts for job %d: %w", f.JobID, err)
	}
	defer tx.Rollback()

	var storedInputHash string
	err = tx.QueryRow("SELECT raw_source_hash FROM match_job_inputs WHERE job_id=?", f.JobID).Scan(&storedInputHash)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: job %d", ErrMatchInputNotRetained, f.JobID)
	}
	if err != nil {
		return fmt.Errorf("read retained input for job %d: %w", f.JobID, err)
	}
	current, err := readRetainedSource(tx, f.JobID)
	if err != nil {
		return err
	}
	currentHash, err := matchcontract.HashRetainedSource(current)
	if err != nil {
		return err
	}
	if storedInputHash != f.RawSourceHash || currentHash != f.RawSourceHash {
		return fmt.Errorf("%w: job %d", ErrMatchSourceChanged, f.JobID)
	}

	blob, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("encode facts for job %d: %w", f.JobID, err)
	}
	if _, err := tx.Exec(`
		INSERT INTO match_job_facts(
			job_id, facts_hash, raw_source_hash, schema_version, extractor_version,
			source_policy_version, facts, updated_at)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(job_id) DO UPDATE SET
			facts_hash=excluded.facts_hash,
			raw_source_hash=excluded.raw_source_hash,
			schema_version=excluded.schema_version,
			extractor_version=excluded.extractor_version,
			source_policy_version=excluded.source_policy_version,
			facts=excluded.facts,
			updated_at=excluded.updated_at`,
		f.JobID, f.FactsHash, f.RawSourceHash, f.SchemaVersion, f.ExtractorVersion,
		f.SourcePolicyVersion, string(blob), now()); err != nil {
		return fmt.Errorf("store facts for job %d: %w", f.JobID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit facts for job %d: %w", f.JobID, err)
	}
	return nil
}

// GetMatchFacts returns validated stored facts, or nil when none exist. An incompatible or
// corrupt schema is refused instead of being interpreted as current facts.
func (s *Store) GetMatchFacts(jobID int64) (*matchcontract.VacancyFacts, error) {
	var blob string
	err := s.DB.QueryRow("SELECT facts FROM match_job_facts WHERE job_id=?", jobID).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read facts for job %d: %w", jobID, err)
	}
	var f matchcontract.VacancyFacts
	if err := json.Unmarshal([]byte(blob), &f); err != nil {
		return nil, fmt.Errorf("decode facts for job %d: %w", jobID, err)
	}
	if err := matchcontract.CheckSchema(f.SchemaVersion); err != nil {
		return nil, fmt.Errorf("job %d facts: %w", jobID, err)
	}
	if err := f.Validate(); err != nil {
		return nil, fmt.Errorf("job %d facts: %w", jobID, err)
	}
	if err := f.VerifyFactsHash(); err != nil {
		return nil, fmt.Errorf("job %d facts: %w", jobID, err)
	}
	return &f, nil
}

// queryer is the read surface shared by *sql.DB and *sql.Tx, so a source snapshot can be read
// inside the same transaction as its write.
type queryer interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

// readRetainedSource reads the authoritative saved Job columns. It does not write and does not
// consult the network. The saved sighting names are retained for policy selection.
func readRetainedSource(q queryer, jobID int64) (matchcontract.RetainedSource, error) {
	var (
		title, company, description, descriptionStatus string
		location, tagsJSON                             sql.NullString
	)
	err := q.QueryRow(`
		SELECT title, company, description, description_status, location, tags
		FROM jobs WHERE id=?`, jobID).
		Scan(&title, &company, &description, &descriptionStatus, &location, &tagsJSON)
	if err == sql.ErrNoRows {
		return matchcontract.RetainedSource{}, fmt.Errorf("job %d not found", jobID)
	}
	if err != nil {
		return matchcontract.RetainedSource{}, fmt.Errorf("read job %d: %w", jobID, err)
	}
	var tags []string
	if tagsJSON.Valid && tagsJSON.String != "" && tagsJSON.String != "null" {
		_ = json.Unmarshal([]byte(tagsJSON.String), &tags)
	}
	sources, err := matchSightings(q, jobID)
	if err != nil {
		return matchcontract.RetainedSource{}, err
	}
	return matchcontract.RetainedSource{
		JobID:             jobID,
		Title:             title,
		Company:           company,
		Location:          location.String,
		Tags:              tags,
		Description:       description,
		DescriptionStatus: descriptionStatus,
		Sources:           sources,
	}, nil
}

func matchSightings(q queryer, jobID int64) ([]string, error) {
	rows, err := q.Query("SELECT source FROM sightings WHERE job_id=? ORDER BY seen, url", jobID)
	if err != nil {
		return nil, fmt.Errorf("read sightings for job %d: %w", jobID, err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan sighting for job %d: %w", jobID, err)
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sightings for job %d: %w", jobID, err)
	}
	return out, nil
}

// MatchSourceCurrent reports whether the retained extraction input still matches the
// authoritative saved Job and can produce current facts. It reads only; it never captures,
// extracts, fetches or rewrites a row.
func (s *Store) MatchSourceCurrent(jobID int64) (bool, error) {
	var storedHash string
	err := s.DB.QueryRow("SELECT raw_source_hash FROM match_job_inputs WHERE job_id=?", jobID).Scan(&storedHash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read retained input for job %d: %w", jobID, err)
	}
	current, err := readRetainedSource(s.DB, jobID)
	if err != nil {
		return false, err
	}
	currentHash, err := matchcontract.HashRetainedSource(current)
	if err != nil {
		return false, err
	}
	return storedHash == currentHash, nil
}

// AllJobIDs returns every saved role id in stable id order. Callers use it for a bounded local
// recomputation; it performs no filtering by query or stage.
func (s *Store) AllJobIDs() ([]int64, error) {
	rows, err := s.DB.Query("SELECT id FROM jobs ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list job ids: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan job id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate job ids: %w", err)
	}
	return out, nil
}
