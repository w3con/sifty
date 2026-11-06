package worker

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// stubWorker locates the Node stub worker relative to the repository root (two levels up from
// this package's directory). The test is skipped when Node is not installed.
func stubWorker(t *testing.T) Worker {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed; skipping worker round-trip")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return Worker{
		Command: "node",
		Args:    []string{filepath.Join(root, "workers", "stub", "index.mjs")},
		Dir:     root,
		Timeout: 30 * time.Second,
	}
}

func TestStubWorkerRoundTrip(t *testing.T) {
	w := stubWorker(t)
	type params struct {
		URL   string `json:"url"`
		Depth int    `json:"depth"`
	}
	type echo struct {
		EchoedMethod string `json:"echoedMethod"`
		EchoedParams params `json:"echoedParams"`
	}
	var got echo
	in := params{URL: "https://example.com/apply", Depth: 2}
	if err := w.Call(context.Background(), "inspectForm", in, &got); err != nil {
		t.Fatalf("call: %v", err)
	}
	if got.EchoedMethod != "inspectForm" {
		t.Fatalf("method not round-tripped: %q", got.EchoedMethod)
	}
	if got.EchoedParams != in {
		t.Fatalf("params not round-tripped: got %+v want %+v", got.EchoedParams, in)
	}
}

func TestWorkerReportsHandledError(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	// A worker that always returns a handled error.
	w := Worker{
		Command: "node",
		Args:    []string{"-e", `process.stdout.write(JSON.stringify({ok:false,error:"boom"}))`},
		Timeout: 30 * time.Second,
	}
	err := w.Call(context.Background(), "anything", map[string]any{}, nil)
	if err == nil {
		t.Fatalf("expected an error from a worker reporting ok:false")
	}
}

func TestWorkerTimeout(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	// A worker that never finishes within the deadline.
	w := Worker{
		Command: "node",
		Args:    []string{"-e", `setInterval(()=>{}, 1000)`},
		Timeout: 300 * time.Millisecond,
	}
	err := w.Call(context.Background(), "hang", map[string]any{}, nil)
	if err == nil {
		t.Fatalf("expected a timeout error")
	}
}
