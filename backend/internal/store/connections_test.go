package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

func openTemp(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "conn.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// pragma reads one integer or text pragma on a pinned connection.
func pragma(t *testing.T, c *sql.Conn, name string) string {
	t.Helper()
	var v string
	if err := c.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&v); err != nil {
		t.Fatalf("PRAGMA %s: %v", name, err)
	}
	return v
}

// Settings that were applied once with Exec belong to one connection only. With several
// connections a missed pragma is silent: no foreign-key enforcement, or no wait on a lock. Every
// connection of both pools must carry them, and the read pool must refuse writes.
func TestEveryConnectionCarriesTheSamePragmas(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()

	// Pin all connections of the read pool at once so each one is a different connection.
	var conns []*sql.Conn
	for i := 0; i < 4; i++ {
		c, err := s.Read.Conn(ctx)
		if err != nil {
			t.Fatalf("read connection %d: %v", i, err)
		}
		defer c.Close()
		conns = append(conns, c)
	}
	for i, c := range conns {
		for pr, want := range map[string]string{"busy_timeout": "5000", "foreign_keys": "1", "query_only": "1"} {
			if got := pragma(t, c, pr); got != want {
				t.Errorf("read connection %d: PRAGMA %s = %s, want %s", i, pr, got, want)
			}
		}
		if _, err := c.ExecContext(ctx, "INSERT INTO health(source,status,message,checked,count) VALUES('x','ok','',?,0)", "now"); err == nil {
			t.Errorf("read connection %d accepted a write", i)
		}
	}

	w, err := s.DB.Conn(ctx)
	if err != nil {
		t.Fatalf("write connection: %v", err)
	}
	defer w.Close()
	for pr, want := range map[string]string{"busy_timeout": "5000", "foreign_keys": "1", "query_only": "0", "journal_mode": "wal"} {
		if got := pragma(t, w, pr); got != want {
			t.Errorf("write connection: PRAGMA %s = %s, want %s", pr, got, want)
		}
	}
}

// One stuck read, or one long write, must not stop every other request: that is what the single
// connection did. Each case holds something open and then requires an ordinary read and an
// ordinary write to finish within a deadline; the deadline is the only way to see a hang.
func TestStuckReadOrLongWriteDoesNotBlockOthers(t *testing.T) {
	newStore := func(t *testing.T) *store.Store {
		s := openTemp(t)
		if _, err := s.Upsert(model.JobInput{Title: "Rust Engineer", Company: "Acme", OriginalURL: "https://example.com/1",
			Description: "d", DescriptionStatus: "available structured description", Source: "Wellfound"}, false); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		return s
	}
	within := func(t *testing.T, what string, fn func() error) {
		t.Helper()
		done := make(chan error, 1)
		go func() { done <- fn() }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("%s: %v", what, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s did not finish: the connection is held by something else", what)
		}
	}

	t.Run("a read that never finishes", func(t *testing.T) {
		s := newStore(t)
		rows, err := s.Read.Query("SELECT id FROM jobs") // cursor left open on one read connection
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		within(t, "another read", func() error { _, err := s.List("", ""); return err })
		within(t, "a write", func() error { return s.Health("Wellfound", "ok", "fine", 1) })
		// The write must be visible to the very next read: the pool must not serve a stale view.
		within(t, "read after write", func() error {
			list, err := s.HealthList()
			if err == nil && (len(list) != 1 || list[0]["status"] != "ok") {
				t.Errorf("read after write = %v, want the row just written", list)
			}
			return err
		})
	})

	t.Run("a write transaction in progress", func(t *testing.T) {
		s := newStore(t)
		tx, err := s.DB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := tx.Exec("INSERT INTO health(source,status,message,checked,count) VALUES('Held','ok','','now',0)"); err != nil {
			t.Fatal(err)
		}
		within(t, "a read during a write transaction", func() error {
			list, err := s.HealthList()
			if err == nil && len(list) != 0 {
				t.Errorf("a reader saw uncommitted data: %v", list)
			}
			_, err2 := s.List("", "")
			if err == nil {
				err = err2
			}
			return err
		})
	})
}
