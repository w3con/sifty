package server_test

import (
	"net/http"
	"testing"
)

// A process whose database does not answer must say so on /api/health, not hold the request or
// answer 200. A closed store stands in for a database that cannot be used: the endpoint must
// answer 503 with the documented message. The wait itself is tested in the store package.
func TestHealthReportsAnUnresponsiveDatabase(t *testing.T) {
	h, s := newRouteHandler(t)
	if resp := apiDo(t, h, "GET", "/api/health", ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health on a working database: %d, want 200", resp.StatusCode)
	}
	_ = s.Close()
	resp := apiDo(t, h, "GET", "/api/health", "")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/health on a dead database: %d, want 503", resp.StatusCode)
	}
	if got := decodeMap(t, resp)["error"]; got != "database not responding" {
		t.Errorf("error = %v, want %q", got, "database not responding")
	}
}
