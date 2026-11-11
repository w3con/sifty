package background

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestNewestKeyMatchesTheInbox checks the server's Newest key against the fixture generated from
// the browser implementation (frontend/dates.ts publicationDate and frontend/filters.ts). The same
// fixture is read by frontend/filters.test.mjs, so the two cannot drift apart unnoticed.
func TestNewestKeyMatchesTheInbox(t *testing.T) {
	raw, err := os.ReadFile("testdata/newest-key.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Now      string `json:"now"`
		Timezone string `json:"timezone"`
		Cases    []struct {
			Raw    string `json:"raw"`
			Basis  bool   `json:"basis"`
			Rank   int    `json:"rank"`
			Millis int64  `json:"millis"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if fx.Timezone != "UTC" {
		t.Fatalf("fixture timezone %q", fx.Timezone)
	}
	saved := time.Local
	time.Local = time.UTC
	defer func() { time.Local = saved }()
	now, err := time.Parse(time.RFC3339, fx.Now)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range fx.Cases {
		basis := ""
		if c.Basis {
			basis = "api"
		}
		got := NewestKeyOf(c.Raw, basis, now)
		if got.Rank != c.Rank || got.Millis != c.Millis {
			t.Errorf("%q (basis %v): key %+v, want rank %d millis %d", c.Raw, c.Basis, got, c.Rank, c.Millis)
		}
	}
}

func TestNewestKeyOrder(t *testing.T) {
	a := NewestKey{Rank: 0, Millis: 100} // dated with a basis, older
	b := NewestKey{Rank: 0, Millis: 200}
	c := NewestKey{Rank: 1, Millis: 900} // dated without a basis comes after any basis-backed date
	d := NewestKey{Rank: 2}
	e := NewestKey{Rank: 2}
	if !b.Before(a) || a.Before(b) {
		t.Error("within a rank the later date must come first")
	}
	if !a.Before(c) || c.Before(a) || !c.Before(d) {
		t.Error("ranks must order 0, 1, 2")
	}
	if d.Before(e) || e.Before(d) {
		t.Error("unknown dates are equal and left to the job id")
	}
}
