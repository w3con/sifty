package server_test

// Step 2.1 — enqueue inspect on pre-selection.
//
// These tests verify that:
//   - Patching a job to the "saved" status (which maps to the "preselected" stage) creates a
//     queued inspect operation with inspect=true and generate=false.
//   - Patching a job that is already preselected does not enqueue a second operation.
//   - The preparation worker drains the queued inspect operation to done without a model call.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/background"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/inspect"
)

// TestPatchToPreselectedEnqueuesInspect verifies that a PATCH /api/jobs/{id} that transitions
// a job to "preselected" creates a queued inspect-only preparation operation.
func TestPatchToPreselectedEnqueuesInspect(t *testing.T) {
	h, s := newRouteHandler(t)
	id := seedJob(t, s)

	// Transition to preselected via the API.
	resp := apiDo(t, h, "PATCH", fmt.Sprintf("/api/jobs/%d", id), `{"status":"saved"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /api/jobs/%d: got %d, want 200", id, resp.StatusCode)
	}
	var job map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		t.Fatalf("decode job: %v", err)
	}
	if stage, _ := job["stage"].(string); stage != "preselected" {
		t.Fatalf("job stage: got %q, want preselected", stage)
	}

	// Verify a queued inspect operation was created.
	ops, err := s.PreparationOperations(id)
	if err != nil {
		t.Fatalf("PreparationOperations: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("expected a queued preparation operation after pre-selection, got none")
	}
	op := ops[0]
	if status, _ := op["status"].(string); status != "queued" {
		t.Errorf("operation status: got %q, want queued", status)
	}
	req, _ := op["request"].(map[string]any)
	if req["inspect"] != true {
		t.Errorf("operation request.inspect: got %v, want true", req["inspect"])
	}
	if req["generate"] != false {
		t.Errorf("operation request.generate: got %v, want false", req["generate"])
	}
}

// TestPatchToPreselectedIdempotent verifies that patching an already-preselected job does not
// enqueue a second operation (the stage guard prevents a duplicate enqueue).
func TestPatchToPreselectedIdempotent(t *testing.T) {
	h, s := newRouteHandler(t)
	id := seedJob(t, s)

	// First transition — creates the operation.
	resp := apiDo(t, h, "PATCH", fmt.Sprintf("/api/jobs/%d", id), `{"status":"saved"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first PATCH: %d", resp.StatusCode)
	}

	// Second identical PATCH — prevStage is already "preselected", so no new enqueue.
	resp = apiDo(t, h, "PATCH", fmt.Sprintf("/api/jobs/%d", id), `{"status":"saved"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second PATCH: %d", resp.StatusCode)
	}

	ops, err := s.PreparationOperations(id)
	if err != nil {
		t.Fatalf("PreparationOperations: %v", err)
	}
	if len(ops) != 1 {
		t.Errorf("operation count: got %d, want 1 (no duplicate enqueue)", len(ops))
	}
}

// TestPatchToPreselectedWorkerDrainsWithoutModelCall verifies the full path:
// PATCH creates an inspect operation, and the preparation worker drains it to done
// using a stub inspect function with no model call.
func TestPatchToPreselectedWorkerDrainsWithoutModelCall(t *testing.T) {
	h, s := newRouteHandler(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	id := seedJob(t, s)

	// PATCH to preselected via the API — this enqueues the inspect operation.
	resp := apiDo(t, h, "PATCH", fmt.Sprintf("/api/jobs/%d", id), `{"status":"saved"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /api/jobs/%d: got %d", id, resp.StatusCode)
	}

	// Stub inspect function — returns a one-field questionnaire; no model involved.
	stubInspect := background.InspectFn(func(ctx context.Context, applyURL string) (inspect.Questionnaire, error) {
		return inspect.Questionnaire{
			URL:    applyURL,
			Status: "partial",
			Fields: []inspect.Question{
				{ID: "q1", Label: "Why do you want this role?", Type: "textarea", Origin: "site", SourceURL: applyURL},
			},
		}, nil
	})

	// prepareFn nil — if the worker ever tried to call it, it would panic. The inspect-only
	// operation must complete without reaching the generate phase.
	w := background.NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspect)
	w.Start()
	defer w.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ops, err := s.PreparationOperations(id)
		if err == nil && len(ops) > 0 {
			if status, _ := ops[0]["status"].(string); status == "done" {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	ops, err := s.PreparationOperations(id)
	if err != nil || len(ops) == 0 {
		t.Fatalf("PreparationOperations: %v", err)
	}
	if status, _ := ops[0]["status"].(string); status != "done" {
		t.Errorf("operation status: got %q, want done", status)
	}
}
