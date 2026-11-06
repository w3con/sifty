package store

import (
	"os"
	"syscall"
)

// isProcessAlive reports whether the process with the given PID is running.
// It uses signal 0 (a no-op that probes existence) on Unix. Returns false for
// PID 0 or when the process does not exist.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}
