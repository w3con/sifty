package server_test

// Step 3.1 — queue runtime fix regression test.
//
// The old queueStatus handler called s.LatestCollectionRun() for the "runtime" field. That
// function reads the collection_runs table and returns a structured object — it never reads the
// queue_state 'runtime' row, which is where the TypeScript worker stores the current scheduler
// state. The frontend then tried to access runtime.data and got null or a type mismatch,
// crashing the catalogue page.
//
// The fix is GetQueueRuntime(), which reads queue_state WHERE key='runtime' and returns
// {"data": "<stored string>"} or null.
//
// These tests verify the correct shape is returned by GET /api/queue and that a stored
// runtime string is surfaced verbatim.

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestQueueRuntimeNullWhenAbsent verifies that GET /api/queue returns runtime: null
// when no queue_state 'runtime' row has been written.
func TestQueueRuntimeNullWhenAbsent(t *testing.T) {
	h, _ := newRouteHandler(t)

	resp := apiDo(t, h, "GET", "/api/queue", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/queue: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// runtime must be present in the response and must be null when no row exists.
	runtime, exists := body["runtime"]
	if !exists {
		t.Fatal("GET /api/queue response missing 'runtime' key")
	}
	if runtime != nil {
		t.Errorf("runtime: got %v (%T), want null — old handler returned LatestCollectionRun output which is non-null", runtime, runtime)
	}
}

// TestQueueRuntimeDataIsVerbatimString verifies that once a queue_state 'runtime' row is
// present, GET /api/queue returns runtime.data as the exact stored string — not decoded,
// not wrapped, not null.
//
// This test fails against the old handler shape (LatestCollectionRun), which never reads
// queue_state.key='runtime' at all and would return runtime=null or a collection-run object.
func TestQueueRuntimeDataIsVerbatimString(t *testing.T) {
	h, s := newRouteHandler(t)

	// Seed the queue_state 'runtime' row — this is what the TypeScript worker writes.
	runtimePayload := `{"status":"idle","lastRun":"2026-11-03T12:00:00Z","nextRun":"2026-11-04T06:00:00Z"}`
	_, err := s.DB.Exec(
		"INSERT INTO queue_state(key,data) VALUES('runtime',?) ON CONFLICT(key) DO UPDATE SET data=excluded.data",
		runtimePayload)
	if err != nil {
		t.Fatalf("seed queue_state runtime: %v", err)
	}

	resp := apiDo(t, h, "GET", "/api/queue", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/queue: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// runtime must be present and be a non-null object.
	runtime, exists := body["runtime"]
	if !exists {
		t.Fatal("GET /api/queue response missing 'runtime' key")
	}
	if runtime == nil {
		t.Fatal("runtime is null; expected {\"data\": \"<string>\"}")
	}
	runtimeObj, ok := runtime.(map[string]any)
	if !ok {
		t.Fatalf("runtime: got %T, want map — old handler returned LatestCollectionRun shape", runtime)
	}

	// runtime.data must be the exact string stored in queue_state.
	data, ok := runtimeObj["data"].(string)
	if !ok {
		t.Fatalf("runtime.data: got %T, want string", runtimeObj["data"])
	}
	if data != runtimePayload {
		t.Errorf("runtime.data: got %q, want %q", data, runtimePayload)
	}

	// Verify the data is valid JSON (the frontend parses it).
	var parsed map[string]any
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		t.Errorf("runtime.data is not valid JSON: %v", err)
	}
}
