// Package store manages the SQLite database that holds all operational state for the Sifty
// backend: roles and sightings, assessments and the queue, workflow history, application drafts,
// collection telemetry, feedback and context indexes, cross-process locks. It opens the database
// with WAL journal mode, busy_timeout and foreign-key enforcement, runs the idempotent migration
// sequence to user_version 16, and exposes the underlying *sql.DB so that higher-level packages
// can prepare and execute their own statements.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver name
)

// Store holds an open, migrated connection to the operational SQLite database.
type Store struct {
	// DB is the underlying database connection. Other packages prepare statements against it
	// directly; the store layer owns only the schema and the connection lifecycle.
	DB *sql.DB
}

// Open opens or creates the SQLite database at path. It sets WAL journal mode, a 5000 ms
// busy timeout and foreign-key enforcement, then calls migrate to bring the schema to
// user_version 16. An existing database already at user_version 16 is opened without
// any schema modification. A fresh database is brought to that version in one pass.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	// One write connection serialises all writes and ensures connection-level PRAGMAs persist
	// for the lifetime of the process. SQLite WAL can still serve concurrent readers from
	// the same process through the same connection pool; MaxOpenConns(1) only limits writers.
	db.SetMaxOpenConns(1)

	// PRAGMA journal_mode returns the active mode name; use QueryRow to consume the row.
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	// These PRAGMAs do not return rows.
	for _, stmt := range []string{
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("%s: %w", stmt, err)
		}
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}
	return &Store{DB: db}, nil
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.DB.Close()
}
