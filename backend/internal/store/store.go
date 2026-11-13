// Package store manages the SQLite database that holds all operational state for the Sifty
// backend: roles and sightings, assessments and the queue, workflow history, application drafts,
// collection telemetry, feedback and context indexes, cross-process locks. It opens the database
// with WAL journal mode, busy_timeout and foreign-key enforcement, runs the idempotent migration
// sequence to user_version 22, and exposes the underlying handles so that higher-level packages
// can prepare and execute their own statements.
//
// The database is reached through two handles. DB is the only write connection: every write and
// every transaction goes through it, and its transactions start IMMEDIATE so that two writers never
// deadlock on a lock upgrade. Read is a small pool of read-only connections for plain reads (lists,
// status, profile, queue state): in WAL mode they run alongside the writer, so one slow or stuck
// read blocks neither the writes nor the other reads.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	_ "modernc.org/sqlite" // registers the "sqlite" driver name
)

const (
	// busyTimeoutMS is how long a connection waits for a lock held by another connection or
	// process before it reports "database is locked". Every connection gets it.
	busyTimeoutMS = "5000"
	// readConns is the size of the read-only pool. Reads are short; four keeps a few slow ones
	// from queuing every other reader.
	readConns = 4
)

// Store holds the open, migrated handles to the operational SQLite database.
type Store struct {
	// DB is the write connection (a pool of exactly one). Other packages prepare statements
	// against it directly; the store layer owns only the schema and the connection lifecycle.
	DB *sql.DB
	// Read is the pool of read-only connections for plain reads. It sees committed data only.
	// Anything inside a transaction, any write and any read whose result decides a write uses DB.
	Read *sql.DB

	// ping is a pool of one read-only connection that nothing else uses, so that Ping is not
	// queued behind a stuck read or a long write. See Ping.
	ping    *sql.DB
	pingMu  sync.Mutex
	pingRun *pingProbe
}

// dsn appends connection parameters to a database path. Parameters in the connection string are
// applied by the driver to every connection it opens, which a pragma run once through Exec is not.
func dsn(path string, params url.Values) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + params.Encode()
}

// Open opens or creates the SQLite database at path. It sets WAL journal mode, a 5000 ms
// busy timeout and foreign-key enforcement on every connection, then calls migrate to bring the
// schema to the current user_version. An existing database already at that version is opened
// without any schema modification. A fresh database is brought to that version in one pass.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn(path, url.Values{
		"_busy_timeout": {busyTimeoutMS},
		"_foreign_keys": {"1"},
		"_txlock":       {"immediate"},
	}))
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	// One write connection serialises all writes. Readers have their own pool (below); WAL lets
	// them run while a write is in progress.
	db.SetMaxOpenConns(1)

	// PRAGMA journal_mode returns the active mode name; use QueryRow to consume the row. The mode
	// is stored in the database file, so the read connections pick it up by themselves.
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}

	// The read pool opens after migration, so the file and its schema exist. query_only makes
	// the pool refuse writes, so a read routed to it by mistake fails loudly instead of racing
	// the writer.
	read, err := sql.Open("sqlite", dsn(path, url.Values{
		"_busy_timeout": {busyTimeoutMS},
		"_foreign_keys": {"1"},
		"_query_only":   {"1"},
	}))
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open readers %s: %w", path, err)
	}
	read.SetMaxOpenConns(readConns)
	read.SetMaxIdleConns(readConns)

	ping, err := sql.Open("sqlite", dsn(path, url.Values{
		"_busy_timeout": {busyTimeoutMS},
		"_query_only":   {"1"},
	}))
	if err != nil {
		read.Close()
		db.Close()
		return nil, fmt.Errorf("open ping connection %s: %w", path, err)
	}
	ping.SetMaxOpenConns(1)
	ping.SetMaxIdleConns(1)
	return &Store{DB: db, Read: read, ping: ping}, nil
}

// Close releases all connection pools.
func (s *Store) Close() error {
	return errors.Join(s.ping.Close(), s.Read.Close(), s.DB.Close())
}
