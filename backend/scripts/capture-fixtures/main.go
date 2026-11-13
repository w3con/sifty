// capture-fixtures hits a running Sifty API server and writes one JSON fixture file per endpoint
// to the specified output directory. Run against the TypeScript application to capture the
// reference implementation's responses, which the replay harness then diffs against the Go backend.
//
// Usage:
//
//	go run scripts/capture-fixtures/main.go --base-url http://127.0.0.1:4317 --out internal/server/testdata/fixtures
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type fixture struct {
	Description     string `json:"description"`
	Method          string `json:"method"`
	Path            string `json:"path"`
	RequestBody     any    `json:"requestBody"`
	ResponseStatus  int    `json:"responseStatus"`
	ResponseBody    any    `json:"responseBody,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	Source          string `json:"source"`
}

// endpoint is one API endpoint to capture. jobID is substituted into paths containing {id}.
type endpoint struct {
	file        string
	description string
	method      string
	path        string
	body        any
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:4317", "base URL of the running Sifty API")
	outDir := flag.String("out", "internal/server/testdata/fixtures", "output directory for fixture files")
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Second}

	// Resolve a job ID first: we need one for the :id endpoints.
	jobID, err := resolveFirstJobID(client, *baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not resolve a job ID (%v); id-specific endpoints will use id=1\n", err)
		jobID = 1
	}
	idStr := fmt.Sprintf("%d", jobID)

	endpoints := buildEndpoints(idStr)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", *outDir, err)
		os.Exit(1)
	}

	ok, failed := 0, 0
	for _, ep := range endpoints {
		if err := capture(client, *baseURL, *outDir, ep); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s %s: %v\n", ep.method, ep.path, err)
			failed++
		} else {
			fmt.Printf("OK   %s %s → %s\n", ep.method, ep.path, ep.file)
			ok++
		}
	}
	fmt.Printf("\nCaptured %d/%d endpoints\n", ok, ok+failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func resolveFirstJobID(c *http.Client, base string) (int64, error) {
	req, _ := http.NewRequest("GET", base+"/api/jobs", nil)
	req.Header.Set("Host", req.Host)
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var jobs []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil || len(jobs) == 0 {
		return 0, fmt.Errorf("no jobs returned")
	}
	switch v := jobs[0]["id"].(type) {
	case float64:
		return int64(v), nil
	}
	return 0, fmt.Errorf("unexpected id type")
}

func capture(c *http.Client, base, outDir string, ep endpoint) error {
	var bodyReader io.Reader
	if ep.body != nil {
		b, _ := json.Marshal(ep.body)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(ep.method, base+ep.path, bodyReader)
	if err != nil {
		return err
	}
	if ep.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Set Host to satisfy the security middleware.
	host := req.URL.Host
	req.Host = host

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var parsedBody any
	ct := resp.Header.Get("Content-Type")
	if ct == "application/pdf" || ct == "application/octet-stream" {
		parsedBody = fmt.Sprintf("<binary %d bytes>", len(rawBody))
	} else {
		_ = json.Unmarshal(rawBody, &parsedBody)
		if parsedBody == nil {
			parsedBody = string(rawBody)
		}
	}

	headers := map[string]string{
		"Content-Type":           ct,
		"X-Content-Type-Options": resp.Header.Get("X-Content-Type-Options"),
		"Referrer-Policy":        resp.Header.Get("Referrer-Policy"),
	}

	f := fixture{
		Description:     ep.description,
		Method:          ep.method,
		Path:            ep.path,
		RequestBody:     ep.body,
		ResponseStatus:  resp.StatusCode,
		ResponseBody:    parsedBody,
		ResponseHeaders: headers,
		Source:          "captured",
	}

	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, ep.file), append(out, '\n'), 0o644)
}

func buildEndpoints(id string) []endpoint {
	return []endpoint{
		{"01_GET_jobs.json", "GET /api/jobs — list all roles", "GET", "/api/jobs", nil},
		{"02_GET_job.json", "GET /api/jobs/:id — get one role", "GET", "/api/jobs/" + id, nil},
		{"03_PATCH_job.json", "PATCH /api/jobs/:id — move stage", "PATCH", "/api/jobs/" + id, map[string]any{"status": "new"}},
		{"04_DELETE_job.json", "DELETE /api/jobs/:id — delete (skipped: destructive)", "DELETE", "/api/jobs/" + id, map[string]any{"confirmed": true}},
		{"05_POST_seen.json", "POST /api/jobs/:id/seen — mark seen", "POST", "/api/jobs/" + id + "/seen", map[string]any{}},
		{"06_POST_read.json", "POST /api/jobs/:id/read — enrich", "POST", "/api/jobs/" + id + "/read", map[string]any{}},
		{"07_POST_assess.json", "POST /api/jobs/:id/assess — enqueue assessment", "POST", "/api/jobs/" + id + "/assess", map[string]any{}},
		{"08_POST_search.json", "POST /api/search — start collection", "POST", "/api/search", map[string]any{}},
		{"09_GET_collection.json", "GET /api/collection — collection progress", "GET", "/api/collection", nil},
		{"10_GET_health.json", "GET /api/health — source health", "GET", "/api/health", nil},
		{"11_POST_import.json", "POST /api/import — import URL", "POST", "/api/import", map[string]any{"url": "https://example.com/sample-job"}},
		{"12_GET_queue.json", "GET /api/queue — queue status", "GET", "/api/queue", nil},
		{"13_PUT_queue_assessment.json", "PUT /api/queue/assessment — set enabled", "PUT", "/api/queue/assessment", map[string]any{"enabled": true}},
		{"14_GET_assessment_options.json", "GET /api/assessment-options — provider list", "GET", "/api/assessment-options", nil},
		{"15_GET_context.json", "GET /api/context — candidate context", "GET", "/api/context", nil},
		{"16_POST_context.json", "POST /api/context — refresh context", "POST", "/api/context", map[string]any{}},
		{"17_GET_filter_options.json", "GET /api/filter-options — search config", "GET", "/api/filter-options", nil},
		{"18_GET_providers.json", "GET /api/providers — provider diagnostics", "GET", "/api/providers", nil},
		{"19_POST_provider_check.json", "POST /api/providers/:name/check — smoke check (first provider)", "POST", "/api/providers/local/check", map[string]any{}},
		{"20_GET_config.json", "GET /api/config/runtime — read config", "GET", "/api/config/runtime", nil},
		{"21_PUT_config.json", "PUT /api/config/runtime — save config (roundtrip)", "PUT", "/api/config/runtime", nil}, // body set from GET response
		{"22_GET_feedback.json", "GET /api/jobs/:id/feedback — feedback list", "GET", "/api/jobs/" + id + "/feedback", nil},
		{"23_POST_feedback.json", "POST /api/jobs/:id/feedback — save feedback", "POST", "/api/jobs/" + id + "/feedback", map[string]any{"text": "Test feedback", "scope": "general"}},
		{"24_GET_application.json", "GET /api/jobs/:id/application — draft+ops", "GET", "/api/jobs/" + id + "/application", nil},
		{"25_POST_prepare.json", "POST /api/jobs/:id/application/prepare — queue", "POST", "/api/jobs/" + id + "/application/prepare", map[string]any{"generate": true}},
		{"26_POST_cancel.json", "POST /api/jobs/:id/application/cancel — cancel", "POST", "/api/jobs/" + id + "/application/cancel", map[string]any{}},
		{"27_PATCH_application.json", "PATCH /api/jobs/:id/application — update draft", "PATCH", "/api/jobs/" + id + "/application", map[string]any{"selectedFile": "profile.pdf", "revision": 0}},
		{"28_POST_questions.json", "POST /api/jobs/:id/application/questions — add question", "POST", "/api/jobs/" + id + "/application/questions", map[string]any{"label": "Motivation?"}},
		{"29_POST_remember.json", "POST /api/jobs/:id/application/remember — remember answer", "POST", "/api/jobs/" + id + "/application/remember", map[string]any{"fieldId": "f1", "revision": 0, "scope": "general", "confirmed": true}},
		{"30_GET_memories.json", "GET /api/jobs/:id/application/memories — memories", "GET", "/api/jobs/" + id + "/application/memories", nil},
		{"31_GET_application_file.json", "GET /api/application-files/profile.pdf — serve PDF", "GET", "/api/application-files/profile.pdf", nil},
	}
}
