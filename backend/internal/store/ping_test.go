package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// A database that does not answer must be reported at the caller's deadline, however it is stuck.
// Holding the one connection reserved for the probe stands in for a hang: the probe cannot run,
// and Ping must still return when its context ends. It must also not fail because ordinary reads
// and writes are busy, and it must recover when the hang ends.
func TestPingReturnsAtTheDeadlineAndIgnoresBusyReadersAndWriter(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "ping.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	// Every reader connection and the writer are held: Ping must still succeed.
	var held []func()
	for i := 0; i < readConns; i++ {
		c, err := s.Read.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, func() { c.Close() })
	}
	w, err := s.DB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	held = append(held, func() { w.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("Ping with every other connection busy: %v", err)
	}
	cancel()
	for _, f := range held {
		f()
	}

	// The probe's own connection is held: a hang. Ping must return at the deadline.
	stuck, err := s.ping.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := s.Ping(ctx); err == nil {
		t.Fatal("Ping answered although the probe could not run")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("Ping took %v, want about the 200ms deadline", d)
	}

	// The hang ends: the next Ping answers.
	stuck.Close()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	if err := s.Ping(ctx2); err != nil {
		t.Fatalf("Ping after the hang ended: %v", err)
	}
}
