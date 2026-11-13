package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/w3con/sifty/internal/model"
)

func decodeMap(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return m
}

func listJobIDs(t *testing.T, h http.Handler, path string) map[int64]map[string]any {
	t.Helper()
	resp := apiDo(t, h, "GET", path, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %d", path, resp.StatusCode)
	}
	var jobs []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		t.Fatalf("decode jobs: %v", err)
	}
	out := map[int64]map[string]any{}
	for _, j := range jobs {
		out[int64(j["id"].(float64))] = j
	}
	return out
}

// TestStopWordEndpoints covers add, list, preview, remove, and that removal clears the stopped
// flag on affected jobs.
func TestStopWordEndpoints(t *testing.T) {
	h, s := newRouteHandler(t)
	risk, err := s.Upsert(model.JobInput{
		Title: "Senior Risk Engineer", Company: "Bank", OriginalURL: "https://bank.example/1",
		Source: "LinkedIn", Description: "Assess risk.", DescriptionStatus: "available",
	}, false)
	if err != nil {
		t.Fatalf("Upsert risk: %v", err)
	}
	goJob := seedJob(t, s) // "Go Engineer"

	// Preview counts without creating anything.
	resp := apiDo(t, h, "GET", "/api/stop-words/preview?word=Risk%20Engineer", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview: %d", resp.StatusCode)
	}
	pv := decodeMap(t, resp)
	if pv["count"] != float64(1) {
		t.Errorf("preview count = %v, want 1", pv["count"])
	}
	var list []map[string]any
	resp = apiDo(t, h, "GET", "/api/stop-words", "")
	_ = json.NewDecoder(resp.Body).Decode(&list)
	if len(list) != 0 {
		t.Fatalf("preview created a rule: %v", list)
	}

	// Empty preview word counts zero; empty add is rejected.
	if m := decodeMap(t, apiDo(t, h, "GET", "/api/stop-words/preview", "")); m["count"] != float64(0) {
		t.Errorf("empty preview count = %v, want 0", m["count"])
	}
	if r := apiDo(t, h, "POST", "/api/stop-words", `{"word":"   "}`); r.StatusCode != http.StatusBadRequest {
		t.Errorf("empty add: got %d, want 400", r.StatusCode)
	}

	// Add returns the stored rule, normalised.
	resp = apiDo(t, h, "POST", "/api/stop-words", `{"word":"Risk Engineer"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add: %d", resp.StatusCode)
	}
	rule := decodeMap(t, resp)
	if rule["word"] != "risk engineer" || rule["id"] == nil || rule["createdAt"] == nil {
		t.Fatalf("add response = %v", rule)
	}
	id := int64(rule["id"].(float64))

	// The role is hidden from every listing; the other stays.
	for _, path := range []string{"/api/jobs", "/api/jobs?status=triage", "/api/jobs?status=all"} {
		got := listJobIDs(t, h, path)
		if _, ok := got[risk]; ok {
			t.Errorf("%s still lists the stopped role", path)
		}
		if _, ok := got[goJob]; !ok {
			t.Errorf("%s dropped the non-matching role", path)
		}
	}
	// Direct fetch still returns the role with the flag.
	job := decodeMap(t, apiDo(t, h, "GET", fmt.Sprintf("/api/jobs/%d", risk), ""))
	if job["stopped"] != true || job["stoppedBy"] != "risk engineer" {
		t.Errorf("job stopped=%v stoppedBy=%v", job["stopped"], job["stoppedBy"])
	}

	// List shows the rule.
	resp = apiDo(t, h, "GET", "/api/stop-words", "")
	list = nil
	_ = json.NewDecoder(resp.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// Remove: the stopped flag clears and the role returns.
	resp = apiDo(t, h, "DELETE", fmt.Sprintf("/api/stop-words/%d", id), "{}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remove: %d", resp.StatusCode)
	}
	if got := listJobIDs(t, h, "/api/jobs"); got[risk] == nil {
		t.Error("role did not return after the stop-word was removed")
	}
	job = decodeMap(t, apiDo(t, h, "GET", fmt.Sprintf("/api/jobs/%d", risk), ""))
	if job["stopped"] != false {
		t.Errorf("stopped = %v after removal, want false", job["stopped"])
	}
	if r := apiDo(t, h, "DELETE", fmt.Sprintf("/api/stop-words/%d", id), "{}"); r.StatusCode != http.StatusBadRequest {
		t.Errorf("remove missing: got %d, want 400", r.StatusCode)
	}
}
