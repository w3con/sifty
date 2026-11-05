// Package worker is the Go core's client for the thin TypeScript workers that run browser
// automation (Playwright) and job-board scraping (jobspy). The core invokes a worker as a
// subprocess and exchanges one JSON request and one JSON response per call, over the contract
// in docs/worker-protocol.md. There is no browser-automation or scraping code in Go.
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// DefaultTimeout bounds a single worker call when a Worker sets none.
const DefaultTimeout = 120 * time.Second

// maxResponseBytes caps a worker's standard output, matching the current application's 2 MB guard.
const maxResponseBytes = 2 << 20

// Request is what the core writes to a worker's standard input.
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Response is what a worker writes to its standard output.
type Response struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Worker describes how to invoke one TypeScript worker.
type Worker struct {
	Command string        // executable, e.g. "node"
	Args    []string      // arguments, e.g. ["workers/playwright/index.mjs"]
	Dir     string        // working directory
	Env     []string      // process environment; nil means an isolated minimal environment
	Timeout time.Duration // per-call timeout; zero means DefaultTimeout
}

// IsolatedEnv returns a minimal environment that does not carry the parent's session, model or
// billing credentials, matching the current application's isolation of inference subprocesses.
func IsolatedEnv() []string {
	keys := []string{"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TMPDIR", "LANG", "TERM",
		"XDG_DATA_HOME", "XDG_CACHE_HOME", "NODE_PATH"}
	var env []string
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func truncate(s string) string {
	const limit = 4000
	if len(s) > limit {
		return s[len(s)-limit:]
	}
	return s
}

// Call runs one request against the worker, writing params as JSON to its input and decoding the
// result into out (which may be nil). A handled worker error, a crash, a timeout, or unparsable
// or oversized output are all returned as errors.
func (w Worker) Call(ctx context.Context, method string, params, out any) error {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("worker %s: marshal params: %w", method, err)
	}
	reqBytes, err := json.Marshal(Request{Method: method, Params: rawParams})
	if err != nil {
		return fmt.Errorf("worker %s: marshal request: %w", method, err)
	}

	timeout := w.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, w.Command, w.Args...)
	cmd.Dir = w.Dir
	cmd.Env = w.Env
	if cmd.Env == nil {
		cmd.Env = IsolatedEnv()
	}
	cmd.Stdin = bytes.NewReader(append(reqBytes, '\n'))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("worker %s: timed out after %s", method, timeout)
		}
		return fmt.Errorf("worker %s: %w (stderr: %s)", method, err, truncate(stderr.String()))
	}
	if stdout.Len() > maxResponseBytes {
		return fmt.Errorf("worker %s: response exceeds %d bytes", method, maxResponseBytes)
	}

	var resp Response
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &resp); err != nil {
		return fmt.Errorf("worker %s: invalid response: %w (stdout: %s)", method, err, truncate(stdout.String()))
	}
	if !resp.OK {
		return fmt.Errorf("worker %s: %s", method, resp.Error)
	}
	if out != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("worker %s: decode result: %w", method, err)
		}
	}
	return nil
}
