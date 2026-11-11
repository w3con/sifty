package background

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/modellock"
)

// TestMain points the machine-wide model lock at a private file, so the tests neither wait for
// nor disturb the lock that real servers on this machine hold.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "sifty-modellock-")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv(dirs.ModelLockEnvVar, filepath.Join(dir, "model.lock"))
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// holdModelLock takes the machine-wide model lock for the rest of the test, simulating a run in
// another process, and fails the test if it is already held.
func holdModelLock(t *testing.T) {
	t.Helper()
	lock, err := modellock.TryAcquire()
	if err != nil {
		t.Fatalf("hold model lock: %v", err)
	}
	t.Cleanup(lock.Release)
	if l2, err := modellock.TryAcquire(); err == nil {
		l2.Release()
		t.Fatal("the model lock should be held; a second acquire must fail")
	}
}
