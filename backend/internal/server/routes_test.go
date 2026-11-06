package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/server"
	"github.com/w3con/sifty/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	return s
}

func newTestDirs(t *testing.T) dirs.Dirs {
	t.Helper()
	d := dirs.Dirs{Root: t.TempDir()}
	if err := d.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	return d
}

func minimalTestCfg() *config.Config {
	trueVal := true
	return &config.Config{
		Search: config.Search{
			Version: 1,
			Sites:   []config.Source{{Name: "LinkedIn", URL: "https://linkedin.com/jobs", Enabled: &trueVal}},
		},
		Runtime:  config.Runtime{Version: 1, Port: 7373},
		Providers: config.Providers{
			Version: 1,
			Stages: struct {
				Assessment  string `yaml:"assessment"`
				Extraction  string `yaml:"extraction"`
				Preparation string `yaml:"preparation"`
			}{Assessment: "p", Extraction: "p", Preparation: "p"},
			Providers: map[string]config.Provider{
				"p": {Type: "ollama", Model: "llama3", Endpoint: "http://127.0.0.1:11434"},
			},
		},
		Candidate: config.Candidate{Version: 1, CacheHours: 24},
	}
}

func newRouteHandler(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	deps := &server.Deps{Store: s, Dirs: d, Cfg: config.NewHolder(cfg)}
	mux := server.New(testPort, func(m *http.ServeMux) {
		server.RegisterRoutes(m, deps)
	})
	return server.Handler(testPort, mux), s
}

func apiDo(t *testing.T, h http.Handler, method, path, body string) *http.Response {
	t.Helper()
	var bodyR *strings.Reader
	if body != "" {
		bodyR = strings.NewReader(body)
	}
	var req *http.Request
	if bodyR != nil {
		req = httptest.NewRequest(method, path, bodyR)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Host", "127.0.0.1:7373")
	req.Host = "127.0.0.1:7373"
	if method != "GET" && method != "HEAD" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Result()
}

func seedJob(t *testing.T, s *store.Store) int64 {
	t.Helper()
	j := model.JobInput{
		Title: "Go Engineer", Company: "Acme", OriginalURL: "https://acme.com/job/1",
		Source: "LinkedIn", Description: "Build Go services.", DescriptionStatus: "available",
	}
	id, err := s.Upsert(j, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	return id
}

// --- Phase 4.1 ---

func TestListJobs(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "GET", "/api/jobs", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/jobs: got %d, want 200", resp.StatusCode)
	}
	var jobs []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("jobs count: got %d, want 1", len(jobs))
	}
}

func TestGetJob(t *testing.T) {
	h, s := newRouteHandler(t)
	id := seedJob(t, s)
	resp := apiDo(t, h, "GET", "/api/jobs/1", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/jobs/1: got %d, want 200", resp.StatusCode)
	}
	var job map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if float64(id) != job["id"].(float64) {
		t.Errorf("id: got %v, want %d", job["id"], id)
	}
}

func TestGetJobNotFound(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "GET", "/api/jobs/9999", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("GET /api/jobs/9999: got %d, want 400", resp.StatusCode)
	}
}

func TestPatchJob(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "PATCH", "/api/jobs/1", `{"status":"rejected"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("PATCH /api/jobs/1: got %d, want 200", resp.StatusCode)
	}
	var job map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if job["stage"] != "no" {
		t.Errorf("stage after rejection: got %v, want no", job["stage"])
	}
}

func TestDeleteJobWithoutConfirm(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "DELETE", "/api/jobs/1", `{"confirmed":false}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("DELETE without confirm: got %d, want 400", resp.StatusCode)
	}
}

func TestSeenJob(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "POST", "/api/jobs/1/seen", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/jobs/1/seen: got %d, want 200", resp.StatusCode)
	}
	var job map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if job["seen"] != true {
		t.Errorf("seen: got %v, want true", job["seen"])
	}
}

func TestAssessEnqueue(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "POST", "/api/jobs/1/assess", `{}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("POST /api/jobs/1/assess: got %d, want 202", resp.StatusCode)
	}
	var row map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if row["status"] != "pending" {
		t.Errorf("queue row status: got %v, want pending", row["status"])
	}
}

// --- Phase 4.2 ---

func TestSearchStartRun(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "POST", "/api/search", `{"source":"LinkedIn"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("POST /api/search: got %d, want 202", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "running" {
		t.Errorf("status: got %v, want running", body["status"])
	}
}

func TestGetCollection(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "GET", "/api/collection", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/collection: got %d, want 200", resp.StatusCode)
	}
}

// newRouteHandlerWithCollect builds a test handler wired with an injectable CollectFn.
func newRouteHandlerWithCollect(t *testing.T, collectFn func(string, bool) (int64, string, error)) (http.Handler, *store.Store) {
	t.Helper()
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	deps := &server.Deps{Store: s, Dirs: d, Cfg: config.NewHolder(cfg), CollectFn: collectFn}
	mux := server.New(testPort, func(m *http.ServeMux) {
		server.RegisterRoutes(m, deps)
	})
	return server.Handler(testPort, mux), s
}

// TestPostSearchForcedStartsRun verifies that a forced POST starts a run that is visible in the
// store and that GET /api/collection reports it.
func TestPostSearchForcedStartsRun(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	var capturedRunID int64
	deps := &server.Deps{
		Store: s, Dirs: d, Cfg: config.NewHolder(cfg),
		CollectFn: func(source string, force bool) (int64, string, error) {
			if !force {
				t.Error("CollectFn called with force=false in a forced-POST test")
			}
			runID, err := s.StartCollectionRun(0, source, "")
			if err != nil {
				return 0, "", err
			}
			capturedRunID = runID
			_ = s.FinishCollectionRun(runID, "complete")
			return runID, "running", nil
		},
	}
	mux := server.New(testPort, func(m *http.ServeMux) { server.RegisterRoutes(m, deps) })
	h := server.Handler(testPort, mux)

	resp := apiDo(t, h, "POST", "/api/search", `{"source":"test","force":true}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /api/search: got %d, want 202", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "running" {
		t.Errorf("status = %v; want running", body["status"])
	}
	if capturedRunID == 0 {
		t.Fatal("CollectFn was not called or did not create a run row")
	}

	// GET /api/collection must report the run.
	cResp := apiDo(t, h, "GET", "/api/collection", "")
	if cResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/collection: got %d, want 200", cResp.StatusCode)
	}
	var cBody map[string]any
	if err := json.NewDecoder(cResp.Body).Decode(&cBody); err != nil {
		t.Fatalf("decode collection: %v", err)
	}
	if cBody == nil {
		t.Fatal("GET /api/collection returned null; expected run data")
	}
	if cBody["status"] != "complete" {
		t.Errorf("collection status = %v; want complete", cBody["status"])
	}
}

// TestPostSearchUnforcedWithinCooldownReturnsSkipped verifies that an unforced POST within the
// cooldown returns a skipped status and does not create a new run.
func TestPostSearchUnforcedWithinCooldownReturnsSkipped(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	callCount := 0
	deps := &server.Deps{
		Store: s, Dirs: d, Cfg: config.NewHolder(cfg),
		CollectFn: func(source string, force bool) (int64, string, error) {
			callCount++
			return 0, "skipped", nil
		},
	}
	mux := server.New(testPort, func(m *http.ServeMux) { server.RegisterRoutes(m, deps) })
	h := server.Handler(testPort, mux)

	resp := apiDo(t, h, "POST", "/api/search", `{"source":"test","force":false}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /api/search: got %d, want 202", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "skipped" {
		t.Errorf("status = %v; want skipped", body["status"])
	}
	if callCount != 1 {
		t.Errorf("CollectFn called %d times; want 1", callCount)
	}

	// No new run must exist.
	run, err := s.LatestCollectionRun()
	if err != nil {
		t.Fatalf("LatestCollectionRun: %v", err)
	}
	if run != nil {
		t.Errorf("expected no run in store after a skipped trigger, got: %v", run)
	}
}

func TestImportURL(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "POST", "/api/import", `{"url":"https://example.com/job/42"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/import: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["id"] == nil {
		t.Error("import: response missing id")
	}
}

// --- Phase 4.3 ---

func TestGetQueue(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "GET", "/api/queue", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/queue: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["enabled"] == nil {
		t.Error("queue: response missing enabled")
	}
}

func TestToggleQueue(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "PUT", "/api/queue/assessment", `{"enabled":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("PUT /api/queue/assessment: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf("enabled: got %v, want false", body["enabled"])
	}
}

func TestAssessmentOptions(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "GET", "/api/assessment-options", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/assessment-options: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["selected"] == nil {
		t.Error("assessment-options: missing selected")
	}
}

// --- Phase 4.4 ---

func TestGetContext(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "GET", "/api/context", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/context: got %d, want 200", resp.StatusCode)
	}
}

func TestFilterOptions(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "GET", "/api/filter-options", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/filter-options: got %d, want 200", resp.StatusCode)
	}
}

func TestGetConfig(t *testing.T) {
	// Write a config file so there's something to read.

	// Use a handler with the dirs that has the file.
	cfg := minimalTestCfg()
	s := newTestStore(t)
	d2 := newTestDirs(t)
	os.WriteFile(filepath.Join(d2.Config(), "runtime.yaml"), []byte("version: 1\nport: 7373\n"), 0o600) //nolint:errcheck
	deps := &server.Deps{Store: s, Dirs: d2, Cfg: config.NewHolder(cfg)}
	mux := server.New(testPort, func(m *http.ServeMux) {
		server.RegisterRoutes(m, deps)
	})
	h2 := server.Handler(testPort, mux)

	resp := apiDo(t, h2, "GET", "/api/config/runtime", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/config/runtime: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["content"] == nil {
		t.Error("config response missing content")
	}
}

// --- Phase 4.5 ---

func TestGetFeedback(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "GET", "/api/jobs/1/feedback", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/jobs/1/feedback: got %d, want 200", resp.StatusCode)
	}
	var body []any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// No feedback yet — should return empty array.
	if len(body) != 0 {
		t.Errorf("feedback: got %d, want 0", len(body))
	}
}

// --- Phase 4.6 ---

func TestGetApplication(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "GET", "/api/jobs/1/application", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/jobs/1/application: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["draft"] == nil {
		t.Error("application: missing draft")
	}
	if body["operations"] == nil {
		t.Error("application: missing operations")
	}
}

func TestAddQuestion(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)
	resp := apiDo(t, h, "POST", "/api/jobs/1/application/questions", `{"label":"Why apply?","type":"textarea"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/jobs/1/application/questions: got %d, want 200", resp.StatusCode)
	}
	var draft map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&draft); err != nil {
		t.Fatalf("decode: %v", err)
	}
	fields, ok := draft["fields"].([]any)
	if !ok || len(fields) != 1 {
		t.Errorf("fields: got %v, want 1 item", draft["fields"])
	}
}
