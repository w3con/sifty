package store

import (
	"strings"
	"testing"
)

func TestSanitiseFTS5(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// No-ops
		{"Blockchain developer", "Blockchain developer"},
		{"Fractional CTO", "Fractional CTO"},
		{"Rust", "Rust"},
		// FTS5 operators stripped
		{`"exact phrase"`, "exact phrase"},
		{"half-time", "half time"},
		{"C++", "C"},
		{"(Solidity)", "Solidity"},
		{"term*", "term"},
		{"^anchor", "anchor"},
		{`back\slash`, "backslash"},   // backslash stripped without space
		{"single'quote", "singlequote"}, // apostrophe stripped without space
		// Whitespace collapsed
		{"  two  spaces  ", "two spaces"},
		{"tab\there", "tab here"},
		// Control characters treated as space
		{"line\nbreak", "line break"},
		// Empty / all-stripped
		{"", ""},
		{`"*^()+- `, ""},
	}
	for _, c := range cases {
		got := sanitiseFTS5(c.in)
		if got != c.want {
			t.Errorf("sanitiseFTS5(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildFTSQuery(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		if q := BuildFTSQuery(nil); q != "" {
			t.Errorf("got %q, want empty", q)
		}
	})

	t.Run("all stripped terms", func(t *testing.T) {
		if q := BuildFTSQuery([]string{"***", "^^^", ""}); q != "" {
			t.Errorf("got %q, want empty", q)
		}
	})

	t.Run("single term", func(t *testing.T) {
		q := BuildFTSQuery([]string{"Rust"})
		if q != `"Rust"` {
			t.Errorf("got %q, want %q", q, `"Rust"`)
		}
	})

	t.Run("multi-word phrase wrapped in quotes", func(t *testing.T) {
		q := BuildFTSQuery([]string{"Fractional CTO"})
		if q != `"Fractional CTO"` {
			t.Errorf("got %q, want %q", q, `"Fractional CTO"`)
		}
	})

	t.Run("multiple terms joined with OR", func(t *testing.T) {
		q := BuildFTSQuery([]string{"Blockchain developer", "Rust", "Solidity"})
		parts := strings.Split(q, " OR ")
		if len(parts) != 3 {
			t.Fatalf("expected 3 OR-parts, got %d: %q", len(parts), q)
		}
		if parts[0] != `"Blockchain developer"` {
			t.Errorf("first part = %q, want %q", parts[0], `"Blockchain developer"`)
		}
	})

	t.Run("deduplication case-insensitive", func(t *testing.T) {
		q := BuildFTSQuery([]string{"Rust", "rust", "RUST"})
		if strings.Count(q, "OR") > 0 {
			t.Errorf("duplicates not removed: %q", q)
		}
		if q != `"Rust"` {
			t.Errorf("got %q, want %q", q, `"Rust"`)
		}
	})

	t.Run("operators stripped before quoting", func(t *testing.T) {
		q := BuildFTSQuery([]string{`"exact"`, "half-time"})
		// "exact" → exact → "exact"; half-time → half time → "half time"
		if !strings.Contains(q, `"exact"`) {
			t.Errorf("stripped term not wrapped: %q", q)
		}
		if !strings.Contains(q, `"half time"`) {
			t.Errorf("sanitised hyphen term not wrapped: %q", q)
		}
	})

	t.Run("typical search.yaml titles", func(t *testing.T) {
		titles := []string{
			"Blockchain developer",
			"Fractional CTO",
			"Senior Software Engineer",
			"CTO",
			"Interim CTO",
		}
		q := BuildFTSQuery(titles)
		// All five should appear as OR-parts.
		parts := strings.Split(q, " OR ")
		if len(parts) != 5 {
			t.Errorf("expected 5 OR-parts, got %d: %q", len(parts), q)
		}
		// Each part must be a quoted phrase (no unquoted operators).
		for _, p := range parts {
			if !strings.HasPrefix(p, `"`) || !strings.HasSuffix(p, `"`) {
				t.Errorf("part not quoted: %q", p)
			}
		}
	})
}
