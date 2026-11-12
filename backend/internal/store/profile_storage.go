package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrProfileNotFound is returned when a write targets a profile that does not exist. Reads use
// (nil, nil) for the absent case so callers can bootstrap.
var ErrProfileNotFound = errors.New("profile not found")

// RevisionConflictError reports a compare-and-swap save whose expected revision is stale.
// Current is the stored row at the moment the conflict was detected.
type RevisionConflictError struct {
	Current *ProfileRecord
}

func (e *RevisionConflictError) Error() string {
	if e.Current == nil {
		return "profile revision conflict"
	}
	return fmt.Sprintf("profile revision conflict: current revision %d", e.Current.Revision)
}

// ProfileRecord is one row of user_profiles. Data is the encoded typed ProfileData contract.
type ProfileRecord struct {
	ID        string
	Revision  int
	Data      string
	CreatedAt string
	UpdatedAt string
}

// CurrentProfile returns the single stored profile, or nil when none exists yet. It reads through
// the read pool; write paths read the row inside their own transaction.
func (s *Store) CurrentProfile() (*ProfileRecord, error) {
	return currentProfileOn(s.Read)
}

func currentProfileOn(q queryer) (*ProfileRecord, error) {
	rec := &ProfileRecord{}
	err := q.QueryRow(
		`SELECT id, revision, data, created_at, updated_at FROM user_profiles ORDER BY id LIMIT 1`).
		Scan(&rec.ID, &rec.Revision, &rec.Data, &rec.CreatedAt, &rec.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read current profile: %w", err)
	}
	return rec, nil
}

// CreateProfile inserts a profile at revision 1 together with its initial revision row. It is
// idempotent for concurrent bootstrap: when the row already exists it returns that row unchanged
// and writes nothing. The insert of the profile and its first history row share one transaction.
func (s *Store) CreateProfile(id, data string) (*ProfileRecord, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	created := now()
	res, err := tx.Exec(
		`INSERT INTO user_profiles(id, revision, data, created_at, updated_at)
		 VALUES(?, 1, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		id, data, created, created)
	if err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("create profile rows: %w", err)
	}
	if inserted == 1 {
		if _, err := tx.Exec(
			`INSERT INTO profile_revisions(profile_id, revision, data, reason, origin, created_at)
			 VALUES(?, 1, ?, ?, ?, ?)`,
			id, data, "bootstrap", "bootstrap", created); err != nil {
			return nil, fmt.Errorf("write initial profile revision: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil

	rec, err := currentProfileOn(s.DB)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("create profile %q: row missing after insert", id)
	}
	return rec, nil
}

// ProfileMutation reads the current row inside one transaction and returns either new encoded
// data or a no-op. The callback must not call the store: the single write connection is held by
// the transaction for the whole merge.
type ProfileMutation func(current *ProfileRecord) (data string, changed bool, reason, origin string, err error)

// MutateProfile performs one atomic read-merge-write of the current profile. The section merge
// runs inside the transaction, so concurrent writers compose on the latest stored row instead of
// overwriting each other with a stale full profile. A no-op writes nothing and keeps the
// revision. A changed write still uses the revision CAS and writes its immutable history row in
// the same transaction.
func (s *Store) MutateProfile(id string, fn ProfileMutation) (*ProfileRecord, bool, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, false, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	current := &ProfileRecord{}
	err = tx.QueryRow(
		`SELECT id, revision, data, created_at, updated_at FROM user_profiles WHERE id=?`,
		id).Scan(&current.ID, &current.Revision, &current.Data, &current.CreatedAt, &current.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, false, ErrProfileNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("read profile for mutation: %w", err)
	}

	data, changed, reason, origin, err := fn(current)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		if err := tx.Rollback(); err != nil {
			return nil, false, err
		}
		tx = nil
		return current, false, nil
	}

	next := current.Revision + 1
	updated := now()
	res, err := tx.Exec(
		`UPDATE user_profiles SET revision=?, data=?, updated_at=? WHERE id=? AND revision=?`,
		next, data, updated, id, current.Revision)
	if err != nil {
		return nil, false, fmt.Errorf("update profile: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, false, fmt.Errorf("update profile rows: %w", err)
	}
	if affected != 1 {
		return nil, false, &RevisionConflictError{Current: current}
	}
	if _, err := tx.Exec(
		`INSERT INTO profile_revisions(profile_id, revision, data, reason, origin, created_at)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		id, next, data, reason, origin, updated); err != nil {
		return nil, false, fmt.Errorf("write profile revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	tx = nil
	return &ProfileRecord{
		ID:        id,
		Revision:  next,
		Data:      data,
		CreatedAt: current.CreatedAt,
		UpdatedAt: updated,
	}, true, nil
}

// SaveProfile applies a compare-and-swap revision write. The new current row and the immutable
// profile_revisions snapshot are written in one transaction: if the history insert fails, the
// current row is rolled back too. A stale expected revision returns *RevisionConflictError and
// leaves the stored profile untouched.
func (s *Store) SaveProfile(id string, expected int, data, reason, origin string) (*ProfileRecord, error) {
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
	err = tx.QueryRow(
		`SELECT id, revision, data, created_at, updated_at FROM user_profiles WHERE id=?`,
		id).Scan(&current.ID, &current.Revision, &current.Data, &current.CreatedAt, &current.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read profile for save: %w", err)
	}
	if current.Revision != expected {
		return nil, &RevisionConflictError{Current: current}
	}

	next := expected + 1
	updated := now()
	res, err := tx.Exec(
		`UPDATE user_profiles SET revision=?, data=?, updated_at=? WHERE id=? AND revision=?`,
		next, data, updated, id, expected)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update profile rows: %w", err)
	}
	if changed != 1 {
		// A concurrent writer reached the row despite the expected revision; treat it as stale.
		return nil, &RevisionConflictError{Current: current}
	}

	if _, err := tx.Exec(
		`INSERT INTO profile_revisions(profile_id, revision, data, reason, origin, created_at)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		id, next, data, reason, origin, updated); err != nil {
		return nil, fmt.Errorf("write profile revision: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return &ProfileRecord{
		ID:        id,
		Revision:  next,
		Data:      data,
		CreatedAt: current.CreatedAt,
		UpdatedAt: updated,
	}, nil
}
