package assess

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ollamaStubResponse returns a minimal valid ollama chat response whose content field
// is already valid JSON for decodeModelJSON.
func ollamaStubResponse() any {
	content := `{"score":0,"summary":"ok","evidence":[],"gaps":[],"confirmedMismatches":[],"candidateUnknowns":[],"employerUnknowns":[]}`
	return map[string]any{
		"message": map[string]any{"content": content},
	}
}

// captureServer starts a local HTTP server that captures the first request body and responds with
// the given response payload. Caller must call srv.Close() when done.
func captureServer(t *testing.T, response any) (srv *httptest.Server, captured *[]byte) {
	t.Helper()
	body := make([]byte, 0)
	captured = &body
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	return srv, captured
}

// TestInferHTTPOllamaNumCtx verifies that an ollama inference request carries
// options.num_ctx derived from the provider's configured context budget.
func TestInferHTTPOllamaNumCtx(t *testing.T) {
	const contextChars = 32000
	// 1 token ≈ 4 chars (conservative). At this size the upper bound is not hit.
	const wantNumCtx = contextChars / 4

	srv, captured := captureServer(t, ollamaStubResponse())
	defer srv.Close()

	p := Provider{
		Type:         "ollama",
		Model:        "llama3",
		Endpoint:     srv.URL,
		TimeoutMS:    5000,
		ContextChars: contextChars,
	}
	if _, err := inferHTTP(context.Background(), p, "test prompt", map[string]any{"type": "object"}); err != nil {
		t.Fatalf("inferHTTP: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(*captured, &body); err != nil {
		t.Fatalf("parse captured request body: %v", err)
	}
	opts, ok := body["options"].(map[string]any)
	if !ok {
		t.Fatalf("ollama request has no options block; body: %s", *captured)
	}
	// JSON numbers decode as float64.
	got, ok := opts["num_ctx"].(float64)
	if !ok {
		t.Fatalf("options.num_ctx is not a number; opts: %v", opts)
	}
	if int(got) != wantNumCtx {
		t.Errorf("options.num_ctx = %d, want %d", int(got), wantNumCtx)
	}
}

// TestInferHTTPOllamaThinkingOption verifies that an explicit Think value reaches the ollama
// payload, and that leaving Think nil does not invent a thinking preference. This is the request
// option that keeps a reasoning model from consuming the whole extraction budget.
func TestInferHTTPOllamaThinkingOption(t *testing.T) {
	noThinking := false

	srv, captured := captureServer(t, ollamaStubResponse())
	defer srv.Close()

	p := Provider{
		Type: "ollama", Model: "llama3", Endpoint: srv.URL,
		TimeoutMS: 5000, ContextChars: 32000, Think: &noThinking,
	}
	if _, err := inferHTTP(context.Background(), p, "test prompt", map[string]any{"type": "object"}); err != nil {
		t.Fatalf("inferHTTP: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(*captured, &body); err != nil {
		t.Fatalf("parse captured request body: %v", err)
	}
	if think, ok := body["think"].(bool); !ok || think {
		t.Fatalf("ollama think = %v (present %v), want explicit false; body: %s", body["think"], ok, *captured)
	}

	p.Think = nil
	if _, err := inferHTTP(context.Background(), p, "test prompt", map[string]any{"type": "object"}); err != nil {
		t.Fatalf("inferHTTP without Think: %v", err)
	}
	var plain map[string]any
	if err := json.Unmarshal(*captured, &plain); err != nil {
		t.Fatalf("parse captured request body: %v", err)
	}
	if _, present := plain["think"]; present {
		t.Fatalf("unset Think must not add a think field; body: %s", *captured)
	}
}

// TestInferHTTPOllamaNumCtxUpperBound verifies that a very large context budget is capped
// at the safe upper bound (131072 tokens) rather than passed through verbatim.
func TestInferHTTPOllamaNumCtxUpperBound(t *testing.T) {
	// 10× the per-token cap in chars — should be clamped to 131072.
	const veryLargeContextChars = 131072 * 4 * 10

	srv, captured := captureServer(t, ollamaStubResponse())
	defer srv.Close()

	p := Provider{
		Type:         "ollama",
		Model:        "llama3",
		Endpoint:     srv.URL,
		TimeoutMS:    5000,
		ContextChars: veryLargeContextChars,
	}
	if _, err := inferHTTP(context.Background(), p, "test prompt", map[string]any{"type": "object"}); err != nil {
		t.Fatalf("inferHTTP: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(*captured, &body); err != nil {
		t.Fatalf("parse captured request body: %v", err)
	}
	opts, ok := body["options"].(map[string]any)
	if !ok {
		t.Fatalf("ollama request has no options block; body: %s", *captured)
	}
	got, ok := opts["num_ctx"].(float64)
	if !ok {
		t.Fatalf("options.num_ctx is not a number; opts: %v", opts)
	}
	const wantCap = 131072
	if int(got) != wantCap {
		t.Errorf("options.num_ctx = %d, want cap of %d", int(got), wantCap)
	}
}

// TestInferHTTPCompatibleNoOptions verifies that the OpenAI-compatible path does NOT set
// an options block, leaving the compatible provider's payload exactly as before.
func TestInferHTTPCompatibleNoOptions(t *testing.T) {
	compatResponse := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": `{"score":0,"summary":"ok","evidence":[],"gaps":[],"confirmedMismatches":[],"candidateUnknowns":[],"employerUnknowns":[]}`,
				},
			},
		},
	}
	srv, captured := captureServer(t, compatResponse)
	defer srv.Close()

	p := Provider{
		Type:         "compatible",
		Model:        "gpt-4o",
		Endpoint:     srv.URL,
		TimeoutMS:    5000,
		ContextChars: 32000,
	}
	if _, err := inferHTTP(context.Background(), p, "test prompt", map[string]any{"type": "object"}); err != nil {
		t.Fatalf("inferHTTP: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(*captured, &body); err != nil {
		t.Fatalf("parse captured request body: %v", err)
	}
	if _, present := body["options"]; present {
		t.Errorf("compatible request must not carry an options block; body: %s", *captured)
	}
}
