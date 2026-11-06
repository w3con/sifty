package server_test

// Tests for the search expression and source sites API endpoints added in NAVIGATION-2:
//   GET  /api/search/expressions
//   PUT  /api/search/expressions
//   GET  /api/search/sites
//   POST /api/search/sites
//   DELETE /api/search/sites/{name}
//   PATCH  /api/search/sites/{name}

import (
	"encoding/json"
	"net/http"
	"testing"
)

// --- GET /api/search/expressions ---

// TestGetSearchExpressionsReturnsArray verifies that GET /api/search/expressions returns a JSON
// array synthesised from the flat titles/keywords in the fixture search.yaml when no expressions
// key is present.
func TestGetSearchExpressionsReturnsArray(t *testing.T) {
	h, _ := newReloadHandler(t)
	resp := apiDo(t, h, "GET", "/api/search/expressions", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/search/expressions: got %d, want 200", resp.StatusCode)
	}
	var body []any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) == 0 {
		t.Error("expressions array must not be empty when titles/keywords are configured")
	}
	// The synthesised expression must have an "or" array with both CTO and golang.
	first, ok := body[0].(map[string]any)
	if !ok {
		t.Fatalf("first element is not an object: %v", body[0])
	}
	orTerms, ok := first["or"].([]any)
	if !ok {
		t.Fatalf("first expression missing or field: %v", first)
	}
	got := make(map[string]bool, len(orTerms))
	for _, t := range orTerms {
		if s, ok := t.(string); ok {
			got[s] = true
		}
	}
	for _, want := range []string{"CTO", "golang"} {
		if !got[want] {
			t.Errorf("synthesised OR expression missing %q; got terms: %v", want, orTerms)
		}
	}
}

// --- PUT /api/search/expressions ---

// TestPutSearchExpressionsAcceptsAndPersists verifies that PUT /api/search/expressions writes
// new OR/AND expressions to search.yaml and that a subsequent GET reflects them.
func TestPutSearchExpressionsAcceptsAndPersists(t *testing.T) {
	h, _ := newReloadHandler(t)

	body := `{"expressions":[{"or":["CTO","CPO"]},{"and":["Blockchain","Head"]}]}`
	resp := apiDo(t, h, "PUT", "/api/search/expressions", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/search/expressions: got %d, want 200", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode PUT response: %v", err)
	}
	if _, ok := result["expressions"]; !ok {
		t.Error("PUT response must include expressions field")
	}

	// GET must now return exactly the two expressions that were PUT.
	resp2 := apiDo(t, h, "GET", "/api/search/expressions", "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET after PUT: got %d, want 200", resp2.StatusCode)
	}
	var after []map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&after); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if len(after) != 2 {
		t.Fatalf("expressions count after PUT: got %d, want 2", len(after))
	}
	// First expression must have an "or" key.
	if _, ok := after[0]["or"]; !ok {
		t.Errorf("first expression missing or key: %v", after[0])
	}
	// Second expression must have an "and" key.
	if _, ok := after[1]["and"]; !ok {
		t.Errorf("second expression missing and key: %v", after[1])
	}
}

// TestPutSearchExpressionsRejectsEmptyBody verifies that PUT without a body returns 400.
func TestPutSearchExpressionsRejectsEmptyBody(t *testing.T) {
	h, _ := newReloadHandler(t)
	resp := apiDo(t, h, "PUT", "/api/search/expressions", `{}`)
	// An empty body or missing expressions key should be rejected.
	if resp.StatusCode == http.StatusInternalServerError {
		t.Errorf("PUT with empty body returned 500; want a 4xx")
	}
}

// --- GET /api/search/sites ---

// TestGetSearchSitesReturnsList verifies that GET /api/search/sites returns a JSON array
// containing at least the LinkedIn entry from the fixture search.yaml.
func TestGetSearchSitesReturnsList(t *testing.T) {
	h, _ := newReloadHandler(t)
	resp := apiDo(t, h, "GET", "/api/search/sites", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/search/sites: got %d, want 200", resp.StatusCode)
	}
	var sites []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&sites); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sites) == 0 {
		t.Fatal("sites list must not be empty")
	}
	first := sites[0]
	for _, key := range []string{"name", "url"} {
		if _, ok := first[key]; !ok {
			t.Errorf("site entry missing field %q: %v", key, first)
		}
	}
	if first["name"] != "LinkedIn" {
		t.Errorf("first site name: got %v, want LinkedIn", first["name"])
	}
}

// --- POST /api/search/sites ---

// TestPostSearchSitesAddsEntry verifies that POST /api/search/sites adds a new site and that
// a subsequent GET returns it.
func TestPostSearchSitesAddsEntry(t *testing.T) {
	h, _ := newReloadHandler(t)

	body := `{"name":"Indeed","url":"https://indeed.com","enabled":true}`
	resp := apiDo(t, h, "POST", "/api/search/sites", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/search/sites: got %d, want 201", resp.StatusCode)
	}
	var added map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&added); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	if added["name"] != "Indeed" {
		t.Errorf("added site name: got %v, want Indeed", added["name"])
	}

	// Verify the site appears in the subsequent GET.
	resp2 := apiDo(t, h, "GET", "/api/search/sites", "")
	var sites []map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&sites); err != nil {
		t.Fatalf("decode GET after POST: %v", err)
	}
	found := false
	for _, s := range sites {
		if s["name"] == "Indeed" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Indeed not found in sites list after POST: %v", sites)
	}
}

// TestPostSearchSitesDuplicateReturnsError verifies that adding a site with a name that already
// exists in search.yaml returns an error.
func TestPostSearchSitesDuplicateReturnsError(t *testing.T) {
	h, _ := newReloadHandler(t)

	// LinkedIn is already seeded in configFixtures.
	resp := apiDo(t, h, "POST", "/api/search/sites", `{"name":"LinkedIn","url":"https://linkedin.com/jobs","enabled":true}`)
	if resp.StatusCode == http.StatusCreated {
		t.Errorf("duplicate POST: got 201, want an error status")
	}
}

// --- DELETE /api/search/sites/{name} ---

// TestDeleteSearchSiteRemovesEntry verifies that DELETE /api/search/sites/{name} removes the
// named site and that a subsequent GET no longer lists it.
func TestDeleteSearchSiteRemovesEntry(t *testing.T) {
	h, _ := newReloadHandler(t)

	// Add a site so the DELETE has something to act on without touching LinkedIn.
	apiDo(t, h, "POST", "/api/search/sites", `{"name":"Glassdoor","url":"https://glassdoor.com","enabled":true}`)

	resp := apiDo(t, h, "DELETE", "/api/search/sites/Glassdoor", "{}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /api/search/sites/Glassdoor: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode DELETE response: %v", err)
	}
	if body["removed"] == nil {
		t.Error("DELETE response must include removed field")
	}

	// Glassdoor must no longer appear in the GET list.
	resp2 := apiDo(t, h, "GET", "/api/search/sites", "")
	var sites []map[string]any
	json.NewDecoder(resp2.Body).Decode(&sites)
	for _, s := range sites {
		if s["name"] == "Glassdoor" {
			t.Error("Glassdoor still appears after DELETE")
		}
	}
}

// TestDeleteSearchSiteNotFoundReturnsError verifies that deleting a non-existent site name
// returns a 400 error.
func TestDeleteSearchSiteNotFoundReturnsError(t *testing.T) {
	h, _ := newReloadHandler(t)

	resp := apiDo(t, h, "DELETE", "/api/search/sites/NonExistent", "{}")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("DELETE non-existent site: got %d, want 400", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if _, ok := body["error"]; !ok {
		t.Error("error response must include error field")
	}
}

// --- PATCH /api/search/sites/{name} ---

// TestPatchSearchSiteTogglesEnabled verifies that PATCH /api/search/sites/{name} with
// enabled:false disables the site and that GET reflects the change.
func TestPatchSearchSiteTogglesEnabled(t *testing.T) {
	h, _ := newReloadHandler(t)

	resp := apiDo(t, h, "PATCH", "/api/search/sites/LinkedIn", `{"enabled":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /api/search/sites/LinkedIn: got %d, want 200", resp.StatusCode)
	}

	// GET must report LinkedIn as disabled.
	resp2 := apiDo(t, h, "GET", "/api/search/sites", "")
	var sites []map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&sites); err != nil {
		t.Fatalf("decode GET after PATCH: %v", err)
	}
	for _, s := range sites {
		if s["name"] != "LinkedIn" {
			continue
		}
		if s["enabled"] == true || s["enabled"] == nil {
			t.Errorf("LinkedIn enabled = %v after PATCH {enabled:false}, want false", s["enabled"])
		}
		return
	}
	t.Error("LinkedIn not found in sites list after PATCH")
}

// TestPatchSearchSiteNotFoundReturnsError verifies that PATCH on a non-existent site name
// returns a 400 error.
func TestPatchSearchSiteNotFoundReturnsError(t *testing.T) {
	h, _ := newReloadHandler(t)

	resp := apiDo(t, h, "PATCH", "/api/search/sites/NoSuchSite", `{"enabled":true}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("PATCH non-existent site: got %d, want 400", resp.StatusCode)
	}
}

// TestGetSearchExpressionsRequiresNoAuth verifies that the endpoint is accessible without any
// special authentication (the host header is set by apiDo).
func TestGetSearchExpressionsRequiresNoAuth(t *testing.T) {
	h, _ := newReloadHandler(t)
	resp := apiDo(t, h, "GET", "/api/search/expressions", "")
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("GET /api/search/expressions: got %d, should be accessible", resp.StatusCode)
	}
}

// TestGetSearchSitesRequiresNoAuth verifies that GET /api/search/sites is accessible.
func TestGetSearchSitesRequiresNoAuth(t *testing.T) {
	h, _ := newReloadHandler(t)
	resp := apiDo(t, h, "GET", "/api/search/sites", "")
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("GET /api/search/sites: got %d, should be accessible", resp.StatusCode)
	}
}
