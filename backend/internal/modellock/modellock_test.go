package modellock

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/store"
)

const holderEnv = "SIFTY_MODELLOCK_TEST_HOLD"

// TestHelperHoldLock is not a test: it is the body of the child process that TestTwoStoresWithDifferentDataDirsCannotHoldItAtOnce starts.
func TestHelperHoldLock(t *testing.T) {
	if os.Getenv(holderEnv) != "1" {
		t.Skip("helper process only")
	}
	lock, err := TryAcquire()
	if err != nil {
		os.Exit(3)
	}
	defer lock.Release()
	_, _ = os.Stdout.WriteString("held\n")
	time.Sleep(30 * time.Second) // the parent kills this process
}

// Two stores with different data directories, each in its own process, must not hold the lock at
// once; and a holder that dies must not leave it held.
func TestTwoStoresWithDifferentDataDirsCannotHoldItAtOnce(t *testing.T) {
	root := t.TempDir()
	t.Setenv(dirs.ModelLockEnvVar, filepath.Join(root, "locks", "model.lock"))

	dirA, dirB := filepath.Join(root, "data-a"), filepath.Join(root, "data-b")
	for _, d := range []string{dirA, dirB} {
		dd := dirs.Dirs{Root: d}
		if err := dd.Ensure(); err != nil {
			t.Fatal(err)
		}
		s, err := store.Open(dd.Store())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
	}
	if dirs.ModelLockPath() != filepath.Join(root, "locks", "model.lock") {
		t.Fatalf("the lock path must not depend on a data directory: %s", dirs.ModelLockPath())
	}

	// Process "A" (a child with its own data directory) takes the lock.
	child := exec.Command(os.Args[0], "-test.run=^TestHelperHoldLock$")
	child.Env = append(os.Environ(), holderEnv+"=1", dirs.EnvVar+"="+dirA)
	out, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	buf := make([]byte, 16)
	if n, _ := out.Read(buf); string(buf[:n]) != "held\n" {
		t.Fatalf("child did not take the lock: %q", buf[:n])
	}

	// Process "B" (this one, data directory B) must be refused.
	t.Setenv(dirs.EnvVar, dirB)
	if lock, err := TryAcquire(); !errors.Is(err, ErrBusy) {
		if lock != nil {
			lock.Release()
		}
		t.Fatalf("second holder with another data directory: err = %v, want ErrBusy", err)
	}

	// The holder dies without releasing; the lock is free again.
	_ = child.Process.Kill()
	_ = child.Wait()
	lock, err := TryAcquire()
	if err != nil {
		t.Fatalf("after the holder died: %v", err)
	}
	defer lock.Release()

	// And within one process a second acquire is refused while the first is held.
	if l2, err := TryAcquire(); !errors.Is(err, ErrBusy) {
		if l2 != nil {
			l2.Release()
		}
		t.Fatalf("second acquire in the same process: err = %v, want ErrBusy", err)
	}
}
