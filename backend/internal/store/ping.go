package store

import (
	"context"
	"time"
)

// pingStuckAfter bounds one probe on its own, so that a probe that is eventually answered frees
// its goroutine. Callers wait for far less (the health endpoint gives up after a couple of seconds).
const pingStuckAfter = 30 * time.Second

// pingProbe is one probe in flight; every caller that arrives meanwhile waits for the same one.
type pingProbe struct {
	done chan struct{}
	err  error
}

// Ping reports whether the database answers, waiting no longer than ctx allows. It reads the
// schema page through a connection reserved for this purpose, so a stuck read, a slow read or a
// long write does not make it fail: only a database that does not answer at all does. That is a
// hung driver, or a file that cannot be read.
//
// The probe runs in its own goroutine and Ping returns when ctx ends even if the probe has not:
// a lock inside the driver is not interrupted by cancelling a context. While one probe is stuck,
// later calls share it instead of starting more goroutines.
func (s *Store) Ping(ctx context.Context) error {
	s.pingMu.Lock()
	p := s.pingRun
	if p == nil {
		p = &pingProbe{done: make(chan struct{})}
		s.pingRun = p
		go func() {
			pctx, cancel := context.WithTimeout(context.Background(), pingStuckAfter)
			defer cancel()
			var n int
			p.err = s.ping.QueryRowContext(pctx, "SELECT count(*) FROM sqlite_master").Scan(&n)
			s.pingMu.Lock()
			s.pingRun = nil
			s.pingMu.Unlock()
			close(p.done)
		}()
	}
	s.pingMu.Unlock()
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
