// Package modellock is the one lock every Sifty process on this machine takes around a model
// call: the background fit worker, the detailed assessment queue and application preparation.
// Every server on the machine (production, staging, review copies) uses the same local model,
// and each has its own data directory, so the lock is an operating-system file lock on a path
// that does not depend on the data directory (dirs.ModelLockPath). The operating system drops the
// lock when its holder exits or crashes, so a dead process never leaves it held.
package modellock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/w3con/sifty/internal/dirs"
)

// ErrBusy means another holder, in this process or another, has the lock.
var ErrBusy = errors.New("the model lock is held by another run")

// Lock is a held model lock. Release it when the model call is finished.
type Lock struct{ f *os.File }

// TryAcquire takes the lock without waiting. It returns ErrBusy when the lock is held.
func TryAcquire() (*Lock, error) {
	path := dirs.ModelLockPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("model lock: create %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("model lock: open %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, fmt.Errorf("model lock: lock %s: %w", path, err)
	}
	return &Lock{f: f}, nil
}

// Release gives the lock up. It is safe to call more than once.
func (l *Lock) Release() {
	if l == nil || l.f == nil {
		return
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	_ = l.f.Close()
	l.f = nil
}
