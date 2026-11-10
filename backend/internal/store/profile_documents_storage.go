package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrProfileDocumentNotFound is returned when a document id does not exist.
var ErrProfileDocumentNotFound = errors.New("profile document not found")

// ErrProfileFactNotFound is returned when a fact id does not exist.
var ErrProfileFactNotFound = errors.New("profile fact not found")

// ErrProfileFactResolved is returned when a fact has already been accepted or rejected.
var ErrProfileFactResolved = errors.New("profile fact already resolved")

// ProfileDocumentRecord is one row of profile_documents. StoragePath is relative to the data
// directory when it names a local original; AnalysisMetadata is a JSON object maintained by the
// profile service.
type ProfileDocumentRecord struct {
	ID               string
	ProfileID        string
	Kind             string
	OriginalName     string
	StoragePath      string
	SourceURL        string
	ContentHash      string
	ProcessingState  string
	ExtractedText    string
	AnalysisMetadata string
	Error            string
	CreatedAt        string
	UpdatedAt        string
}

// ProfileFactRecord is one attributed proposal or confirmed assertion.
type ProfileFactRecord struct {
	ID              string
	ProfileID       string
	DocumentID      *string
	FieldKey        string
	ValueJSON       string
	Quote           string
	SourceReference string
	Origin          string
	State           string
	SupersedesID    *string
	CreatedAt       string
	UpdatedAt       string
}

func scanProfileDocument(row interface {
	Scan(dest ...any) error
}) (*ProfileDocumentRecord, error) {
	rec := &ProfileDocumentRecord{}
	var storage, source, hash, text, metadata, errText sql.NullString
	if err := row.Scan(
		&rec.ID, &rec.ProfileID, &rec.Kind, &rec.OriginalName, &storage, &source,
		&hash, &rec.ProcessingState, &text, &metadata, &errText,
		&rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, err
	}
	rec.StoragePath = storage.String
	rec.SourceURL = source.String
	rec.ContentHash = hash.String
	rec.ExtractedText = text.String
	rec.AnalysisMetadata = metadata.String
	rec.Error = errText.String
	return rec, nil
}

const profileDocumentColumns = `id, profile_id, kind, original_name, storage_path, source_url,
content_hash, processing_state, extracted_text, analysis_metadata, error, created_at, updated_at`

// CreateProfileDocument inserts one document row. The original file, when present, has already
// been written under the data directory by the caller.
func (s *Store) CreateProfileDocument(rec *ProfileDocumentRecord) error {
	created := now()
	if rec.CreatedAt == "" {
		rec.CreatedAt = created
	}
	rec.UpdatedAt = created
	_, err := s.DB.Exec(
		`INSERT INTO profile_documents(id, profile_id, kind, original_name, storage_path, source_url,
			content_hash, processing_state, extracted_text, analysis_metadata, error, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rec.ID, rec.ProfileID, rec.Kind, rec.OriginalName, nullString(rec.StoragePath),
		nullString(rec.SourceURL), nullString(rec.ContentHash), rec.ProcessingState,
		nullString(rec.ExtractedText), nullString(rec.AnalysisMetadata), nullString(rec.Error),
		rec.CreatedAt, rec.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create profile document: %w", err)
	}
	return nil
}

// ListProfileDocuments returns all documents for one profile, newest first.
func (s *Store) ListProfileDocuments(profileID string) ([]*ProfileDocumentRecord, error) {
	rows, err := s.DB.Query(
		`SELECT `+profileDocumentColumns+` FROM profile_documents WHERE profile_id=? ORDER BY created_at, id`,
		profileID)
	if err != nil {
		return nil, fmt.Errorf("list profile documents: %w", err)
	}
	defer rows.Close()
	var out []*ProfileDocumentRecord
	for rows.Next() {
		rec, err := scanProfileDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan profile document: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// GetProfileDocument returns one document or ErrProfileDocumentNotFound.
func (s *Store) GetProfileDocument(id string) (*ProfileDocumentRecord, error) {
	rec, err := scanProfileDocument(s.DB.QueryRow(
		`SELECT `+profileDocumentColumns+` FROM profile_documents WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrProfileDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read profile document: %w", err)
	}
	return rec, nil
}

// UpdateProfileDocumentText records a completed local text extraction.
func (s *Store) UpdateProfileDocumentText(id, text, hash, state, metadata, errMsg string) error {
	_, err := s.DB.Exec(
		`UPDATE profile_documents SET extracted_text=?, content_hash=?, processing_state=?,
			analysis_metadata=?, error=?, updated_at=? WHERE id=?`,
		nullString(text), nullString(hash), state, nullString(metadata), nullString(errMsg), now(), id)
	if err != nil {
		return fmt.Errorf("update profile document text: %w", err)
	}
	return nil
}

// UpdateProfileDocumentState records a state/error change without touching extracted text.
func (s *Store) UpdateProfileDocumentState(id, state, metadata, errMsg string) error {
	_, err := s.DB.Exec(
		`UPDATE profile_documents SET processing_state=?, analysis_metadata=?, error=?, updated_at=?
		 WHERE id=?`,
		state, nullString(metadata), nullString(errMsg), now(), id)
	if err != nil {
		return fmt.Errorf("update profile document state: %w", err)
	}
	return nil
}

// FailProfileDocumentAnalysis makes a publication failure visible only when the run is still the
// current analyzing run. It is a no-op for a cancelled or superseded run, so it never overwrites
// a newer run's state.
func (s *Store) FailProfileDocumentAnalysis(id, runID, metadata, errMsg string) (bool, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return false, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	var state, currentMetadata string
	if err := tx.QueryRow(
		`SELECT processing_state, COALESCE(analysis_metadata, '') FROM profile_documents WHERE id=?`,
		id).Scan(&state, &currentMetadata); err != nil {
		if err == sql.ErrNoRows {
			return false, ErrProfileDocumentNotFound
		}
		return false, fmt.Errorf("read document for failure: %w", err)
	}
	if state != "analyzing" || analysisRunID(currentMetadata) != runID {
		return false, nil
	}
	if _, err := tx.Exec(
		`UPDATE profile_documents SET processing_state='error', analysis_metadata=?, error=?, updated_at=?
		 WHERE id=?`,
		nullString(metadata), nullString(errMsg), now(), id); err != nil {
		return false, fmt.Errorf("fail profile document analysis: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	tx = nil
	return true, nil
}

// StartProfileDocumentAnalysis marks a document analyzing for one run id. It overwrites a stale
// analyzing state from a previous process; the run id is the late-result guard.
func (s *Store) StartProfileDocumentAnalysis(id, metadata string) error {
	_, err := s.DB.Exec(
		`UPDATE profile_documents SET processing_state='analyzing', analysis_metadata=?, error='',
			updated_at=? WHERE id=?`,
		metadata, now(), id)
	if err != nil {
		return fmt.Errorf("start profile document analysis: %w", err)
	}
	return nil
}

// CompleteProfileDocumentAnalysis publishes analysis facts only if the document is still
// analyzing under runID. A cancelled or superseded run returns false and writes nothing. The
// replacement of prior proposed facts, insertion of new proposals and document state update share
// one transaction; accepted/rejected/conflict facts are never deleted.
func (s *Store) CompleteProfileDocumentAnalysis(id, runID, state, metadata, errMsg string, facts []*ProfileFactRecord) (bool, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return false, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var currentState, currentMetadata, currentHash string
	if err := tx.QueryRow(
		`SELECT processing_state, COALESCE(analysis_metadata, ''), COALESCE(content_hash, '')
		 FROM profile_documents WHERE id=?`,
		id).Scan(&currentState, &currentMetadata, &currentHash); err != nil {
		if err == sql.ErrNoRows {
			return false, ErrProfileDocumentNotFound
		}
		return false, fmt.Errorf("read document for completion: %w", err)
	}
	if currentState != "analyzing" || analysisRunID(currentMetadata) != runID {
		return false, nil
	}
	if expected := analysisContentHash(currentMetadata); expected != "" && expected != currentHash {
		return false, nil
	}

	if facts != nil {
		if _, err := tx.Exec(
			`DELETE FROM profile_facts WHERE document_id=? AND state='proposed'`, id); err != nil {
			return false, fmt.Errorf("replace proposed facts: %w", err)
		}
		for _, fact := range facts {
			if err := insertProfileFact(tx, fact); err != nil {
				return false, err
			}
		}
	}
	if _, err := tx.Exec(
		`UPDATE profile_documents SET processing_state=?, analysis_metadata=?, error=?, updated_at=?
		 WHERE id=?`,
		state, nullString(metadata), nullString(errMsg), now(), id); err != nil {
		return false, fmt.Errorf("complete profile document analysis: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	tx = nil
	return true, nil
}

func analysisContentHash(metadata string) string {
	if metadata == "" {
		return ""
	}
	var m struct {
		ContentHash string `json:"contentHash"`
	}
	if json.Unmarshal([]byte(metadata), &m) != nil {
		return ""
	}
	return m.ContentHash
}

func analysisRunID(metadata string) string {
	if metadata == "" {
		return ""
	}
	var m struct {
		RunID string `json:"runId"`
	}
	if json.Unmarshal([]byte(metadata), &m) != nil {
		return ""
	}
	return m.RunID
}

const profileFactColumns = `id, profile_id, document_id, field_key, value_json, quote,
source_reference, origin, state, supersedes_id, created_at, updated_at`

func scanProfileFact(row interface {
	Scan(dest ...any) error
}) (*ProfileFactRecord, error) {
	rec := &ProfileFactRecord{}
	var documentID, quote, sourceRef, supersedes sql.NullString
	if err := row.Scan(
		&rec.ID, &rec.ProfileID, &documentID, &rec.FieldKey, &rec.ValueJSON, &quote,
		&sourceRef, &rec.Origin, &rec.State, &supersedes, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, err
	}
	if documentID.Valid {
		v := documentID.String
		rec.DocumentID = &v
	}
	rec.Quote = quote.String
	rec.SourceReference = sourceRef.String
	if supersedes.Valid {
		v := supersedes.String
		rec.SupersedesID = &v
	}
	return rec, nil
}

// ListProfileFacts returns facts for one profile, optionally narrowed to one document.
func (s *Store) ListProfileFacts(profileID, documentID string) ([]*ProfileFactRecord, error) {
	query := `SELECT ` + profileFactColumns + ` FROM profile_facts WHERE profile_id=?`
	args := []any{profileID}
	if documentID != "" {
		query += ` AND document_id=?`
		args = append(args, documentID)
	}
	query += ` ORDER BY created_at, id`
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list profile facts: %w", err)
	}
	defer rows.Close()
	var out []*ProfileFactRecord
	for rows.Next() {
		rec, err := scanProfileFact(rows)
		if err != nil {
			return nil, fmt.Errorf("scan profile fact: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// GetProfileFact returns one fact or ErrProfileFactNotFound.
func (s *Store) GetProfileFact(id string) (*ProfileFactRecord, error) {
	rec, err := scanProfileFact(s.DB.QueryRow(
		`SELECT `+profileFactColumns+` FROM profile_facts WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrProfileFactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read profile fact: %w", err)
	}
	return rec, nil
}

// UpdateProfileFactState changes a fact's state when it is still in one of fromStates.
func (s *Store) UpdateProfileFactState(id string, fromStates []string, state string) (*ProfileFactRecord, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	rec, err := scanProfileFact(tx.QueryRow(`SELECT `+profileFactColumns+` FROM profile_facts WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, ErrProfileFactNotFound
	}
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, from := range fromStates {
		if rec.State == from {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrProfileFactResolved
	}
	updated := now()
	if _, err := tx.Exec(`UPDATE profile_facts SET state=?, updated_at=? WHERE id=?`, state, updated, id); err != nil {
		return nil, fmt.Errorf("update profile fact state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	rec.State = state
	rec.UpdatedAt = updated
	return rec, nil
}

// SetProfileFactStateCAS changes a fact state only when the profile revision still matches and
// the fact is still in one of fromStates. It is the atomic decision path for an accepted fact that
// makes no profile change (or records a conflict), closing the race between the service's read
// and the state update.
func (s *Store) SetProfileFactStateCAS(factID string, expected int, fromStates []string, state string) (*ProfileFactRecord, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	current := &ProfileRecord{}
	if err := tx.QueryRow(
		`SELECT id, revision, data, created_at, updated_at FROM user_profiles ORDER BY id LIMIT 1`).
		Scan(&current.ID, &current.Revision, &current.Data, &current.CreatedAt, &current.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("read profile for fact state: %w", err)
	}
	if current.Revision != expected {
		return nil, &RevisionConflictError{Current: current}
	}
	rec, err := scanProfileFact(tx.QueryRow(`SELECT `+profileFactColumns+` FROM profile_facts WHERE id=?`, factID))
	if err == sql.ErrNoRows {
		return nil, ErrProfileFactNotFound
	}
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, from := range fromStates {
		if rec.State == from {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrProfileFactResolved
	}
	updated := now()
	if _, err := tx.Exec(`UPDATE profile_facts SET state=?, updated_at=? WHERE id=?`, state, updated, factID); err != nil {
		return nil, fmt.Errorf("set profile fact state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	rec.State = state
	rec.UpdatedAt = updated
	return rec, nil
}

// AcceptProfileFact writes the accepted fact state, the new current profile and its immutable
// revision snapshot in one transaction. The compare-and-swap revision check and the fact state
// check happen inside the same transaction, so a stale or failed confirmation leaves both the
// profile and the fact untouched.
func (s *Store) AcceptProfileFact(factID string, expected int, data, reason, origin, newState string) (*ProfileRecord, *ProfileFactRecord, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	current := &ProfileRecord{}
	if err := tx.QueryRow(
		`SELECT id, revision, data, created_at, updated_at FROM user_profiles ORDER BY id LIMIT 1`).
		Scan(&current.ID, &current.Revision, &current.Data, &current.CreatedAt, &current.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrProfileNotFound
		}
		return nil, nil, fmt.Errorf("read profile for fact acceptance: %w", err)
	}
	if current.Revision != expected {
		return nil, nil, &RevisionConflictError{Current: current}
	}
	fact, err := scanProfileFact(tx.QueryRow(`SELECT `+profileFactColumns+` FROM profile_facts WHERE id=?`, factID))
	if err == sql.ErrNoRows {
		return nil, nil, ErrProfileFactNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if fact.ProfileID != current.ID {
		return nil, nil, fmt.Errorf("fact belongs to another profile")
	}
	if fact.State != "proposed" && fact.State != "conflict" {
		return nil, nil, ErrProfileFactResolved
	}

	updated := now()
	if _, err := tx.Exec(`UPDATE profile_facts SET state=?, updated_at=? WHERE id=?`, newState, updated, factID); err != nil {
		return nil, nil, fmt.Errorf("accept profile fact: %w", err)
	}
	fact.State = newState
	fact.UpdatedAt = updated

	next := expected + 1
	res, err := tx.Exec(
		`UPDATE user_profiles SET revision=?, data=?, updated_at=? WHERE id=? AND revision=?`,
		next, data, updated, current.ID, expected)
	if err != nil {
		return nil, nil, fmt.Errorf("update profile for fact: %w", err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return nil, nil, fmt.Errorf("update profile for fact rows: %w", err)
	}
	if changed != 1 {
		return nil, nil, &RevisionConflictError{Current: current}
	}
	if _, err := tx.Exec(
		`INSERT INTO profile_revisions(profile_id, revision, data, reason, origin, created_at)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		current.ID, next, data, reason, origin, updated); err != nil {
		return nil, nil, fmt.Errorf("write profile revision for fact: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	tx = nil
	current.Revision = next
	current.Data = data
	current.UpdatedAt = updated
	return current, fact, nil
}

// RejectProfileFact marks a proposed/conflict fact rejected without touching the profile.
func (s *Store) RejectProfileFact(id string) (*ProfileFactRecord, error) {
	return s.UpdateProfileFactState(id, []string{"proposed", "conflict"}, "rejected")
}

func insertProfileFact(tx *sql.Tx, rec *ProfileFactRecord) error {
	created := now()
	if rec.CreatedAt == "" {
		rec.CreatedAt = created
	}
	rec.UpdatedAt = created
	_, err := tx.Exec(
		`INSERT INTO profile_facts(id, profile_id, document_id, field_key, value_json, quote,
			source_reference, origin, state, supersedes_id, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		rec.ID, rec.ProfileID, nullStringPtr(rec.DocumentID), rec.FieldKey, rec.ValueJSON,
		nullString(rec.Quote), nullString(rec.SourceReference), rec.Origin, rec.State,
		nullStringPtr(rec.SupersedesID), rec.CreatedAt, rec.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert profile fact: %w", err)
	}
	return nil
}

func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullStringPtr(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}
