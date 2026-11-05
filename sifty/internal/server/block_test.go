package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestBlockJobCompany verifies that POST /api/jobs/{id}/block with dimension=company
// creates a company rule using normalizeText(company), regardless of URL host.
func TestBlockJobCompany(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s) // Company: "Acme" → normalizeText → "acme"

	resp := apiDo(t, h, "POST", "/api/jobs/1/block", `{"dimension":"company"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/jobs/1/block company: got %d, want 200", resp.StatusCode)
	}
	var rule map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rule); err != nil {
		t.Fatalf("decode rule: %v", err)
	}
	if rule["kind"] != "company" {
		t.Errorf("kind: got %v, want company", rule["kind"])
	}
	if rule["value"] != "acme" {
		t.Errorf("value: got %v, want acme", rule["value"])
	}
	if rule["id"] == nil {
		t.Error("id: must be present in response")
	}
	if rule["createdAt"] == nil {
		t.Error("createdAt: must be present in response")
	}
}

// TestBlockJobSource verifies that POST /api/jobs/{id}/block with dimension=source
// creates a source rule using the job's first source name.
func TestBlockJobSource(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s) // Source: "LinkedIn"

	resp := apiDo(t, h, "POST", "/api/jobs/1/block", `{"dimension":"source"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/jobs/1/block source: got %d, want 200", resp.StatusCode)
	}
	var rule map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rule); err != nil {
		t.Fatalf("decode rule: %v", err)
	}
	if rule["kind"] != "source" {
		t.Errorf("kind: got %v, want source", rule["kind"])
	}
	if rule["value"] != "LinkedIn" {
		t.Errorf("value: got %v, want LinkedIn", rule["value"])
	}
}

// TestBlockJobInvalidDimension verifies that an unknown dimension returns 400.
func TestBlockJobInvalidDimension(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)

	resp := apiDo(t, h, "POST", "/api/jobs/1/block", `{"dimension":"unknown"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown dimension: got %d, want 400", resp.StatusCode)
	}
}

// TestListBlockRules verifies that GET /api/block-rules returns all active rules.
func TestListBlockRules(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)

	// No rules initially.
	resp0 := apiDo(t, h, "GET", "/api/block-rules", "")
	if resp0.StatusCode != http.StatusOK {
		t.Errorf("GET /api/block-rules (empty): got %d, want 200", resp0.StatusCode)
	}
	var empty []map[string]any
	json.NewDecoder(resp0.Body).Decode(&empty)
	if len(empty) != 0 {
		t.Errorf("empty rules: got %d, want 0", len(empty))
	}

	// Create a rule.
	apiDo(t, h, "POST", "/api/jobs/1/block", `{"dimension":"company"}`)

	resp := apiDo(t, h, "GET", "/api/block-rules", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/block-rules: got %d, want 200", resp.StatusCode)
	}
	var rules []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rules); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("rules count: got %d, want 1", len(rules))
	}
}

// TestRemoveBlockRuleClearsBlocked verifies that DELETE /api/block-rules/{id} removes the rule
// and that the previously blocked job is no longer blocked after removal.
func TestRemoveBlockRuleClearsBlocked(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s)

	// Block the job by company.
	resp1 := apiDo(t, h, "POST", "/api/jobs/1/block", `{"dimension":"company"}`)
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("POST block: got %d, want 200", resp1.StatusCode)
	}
	var rule map[string]any
	json.NewDecoder(resp1.Body).Decode(&rule)
	ruleID := int(rule["id"].(float64))

	// Verify the job is now blocked.
	jobResp := apiDo(t, h, "GET", "/api/jobs/1", "")
	var job map[string]any
	json.NewDecoder(jobResp.Body).Decode(&job)
	if job["blocked"] != true {
		t.Error("job should be blocked after rule insertion")
	}

	// Remove the rule.
	resp2 := apiDo(t, h, "DELETE", fmt.Sprintf("/api/block-rules/%d", ruleID), "")
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("DELETE /api/block-rules/%d: got %d, want 200", ruleID, resp2.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp2.Body).Decode(&body)
	if body["removed"] == nil {
		t.Error("response must include removed id")
	}

	// Verify the job is no longer blocked.
	jobResp2 := apiDo(t, h, "GET", "/api/jobs/1", "")
	var job2 map[string]any
	json.NewDecoder(jobResp2.Body).Decode(&job2)
	if job2["blocked"] == true {
		t.Error("job should not be blocked after rule removal")
	}
}

// TestRemoveBlockRuleNotFound verifies that deleting a non-existent rule returns 400.
func TestRemoveBlockRuleNotFound(t *testing.T) {
	h, _ := newRouteHandler(t)
	resp := apiDo(t, h, "DELETE", "/api/block-rules/9999", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("non-existent rule: got %d, want 400", resp.StatusCode)
	}
}

// TestListSources verifies that GET /api/sources returns the distinct source names with a role
// count each and a blocked flag that follows the active source rules.
func TestListSources(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s) // one role on "LinkedIn"

	type src struct {
		Name    string `json:"name"`
		Roles   int    `json:"roles"`
		Blocked bool   `json:"blocked"`
	}
	list := func() []src {
		resp := apiDo(t, h, "GET", "/api/sources", "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /api/sources: got %d, want 200", resp.StatusCode)
		}
		var out []src
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return out
	}

	got := list()
	if len(got) != 1 || got[0].Name != "LinkedIn" || got[0].Roles != 1 || got[0].Blocked {
		t.Fatalf("sources before block: got %+v, want [{LinkedIn 1 false}]", got)
	}

	apiDo(t, h, "POST", "/api/jobs/1/block", `{"dimension":"source"}`)
	got = list()
	if len(got) != 1 || !got[0].Blocked {
		t.Errorf("sources after block: got %+v, want LinkedIn blocked", got)
	}
}

// TestCreateSourceBlockRule verifies that POST /api/block-rules creates a source rule directly,
// returns the stored rule, and that the matching role is then blocked.
func TestCreateSourceBlockRule(t *testing.T) {
	h, s := newRouteHandler(t)
	seedJob(t, s) // Source: "LinkedIn"

	resp := apiDo(t, h, "POST", "/api/block-rules", `{"kind":"source","value":"LinkedIn"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/block-rules: got %d, want 200", resp.StatusCode)
	}
	var rule map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rule); err != nil {
		t.Fatalf("decode rule: %v", err)
	}
	if rule["kind"] != "source" || rule["value"] != "LinkedIn" || rule["id"] == nil || rule["createdAt"] == nil {
		t.Errorf("rule: got %v, want source/LinkedIn with id and createdAt", rule)
	}

	// The role on that source is now blocked.
	jobResp := apiDo(t, h, "GET", "/api/jobs/1", "")
	var job map[string]any
	json.NewDecoder(jobResp.Body).Decode(&job)
	if job["blocked"] != true {
		t.Error("job should be blocked after a source rule is created from Settings")
	}

	// Creating the same rule again is idempotent: same id, still one rule.
	resp2 := apiDo(t, h, "POST", "/api/block-rules", `{"kind":"source","value":"LinkedIn"}`)
	var rule2 map[string]any
	json.NewDecoder(resp2.Body).Decode(&rule2)
	if rule2["id"] != rule["id"] {
		t.Errorf("duplicate create: id %v, want %v", rule2["id"], rule["id"])
	}
	listResp := apiDo(t, h, "GET", "/api/block-rules", "")
	var rules []map[string]any
	json.NewDecoder(listResp.Body).Decode(&rules)
	if len(rules) != 1 {
		t.Errorf("rules count: got %d, want 1", len(rules))
	}

	// Removing the rule restores the role.
	apiDo(t, h, "DELETE", fmt.Sprintf("/api/block-rules/%d", int(rule["id"].(float64))), "")
	jobResp2 := apiDo(t, h, "GET", "/api/jobs/1", "")
	var job2 map[string]any
	json.NewDecoder(jobResp2.Body).Decode(&job2)
	if job2["blocked"] == true {
		t.Error("job should not be blocked after the rule is removed")
	}
}

// TestCreateBlockRuleRejectsBadInput verifies that a non-source kind or an empty value returns 400.
func TestCreateBlockRuleRejectsBadInput(t *testing.T) {
	h, _ := newRouteHandler(t)
	for _, body := range []string{
		`{"kind":"company","value":"acme"}`,
		`{"kind":"source","value":"   "}`,
		`{"kind":"source"}`,
	} {
		resp := apiDo(t, h, "POST", "/api/block-rules", body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("POST /api/block-rules %s: got %d, want 400", body, resp.StatusCode)
		}
	}
}
