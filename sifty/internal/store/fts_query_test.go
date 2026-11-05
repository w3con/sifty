package store

import (
	"strings"
	"testing"
)

// TestBuildFTSQueryExprs covers the OR/AND combinator logic.
func TestBuildFTSQueryExprs(t *testing.T) {
	t.Run("empty groups", func(t *testing.T) {
		if q := BuildFTSQueryExprs(nil); q != "" {
			t.Errorf("got %q, want empty", q)
		}
	})

	t.Run("single OR group one term", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{Or: []string{"CTO"}}})
		if q != `"CTO"` {
			t.Errorf("got %q, want %q", q, `"CTO"`)
		}
	})

	t.Run("single OR group multiple terms", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{Or: []string{"CTO", "Interim CTO"}}})
		if q != `("CTO" OR "Interim CTO")` {
			t.Errorf("got %q, want %q", q, `("CTO" OR "Interim CTO")`)
		}
	})

	t.Run("single AND group one term", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{And: []string{"Blockchain"}}})
		if q != `"Blockchain"` {
			t.Errorf("got %q, want %q", q, `"Blockchain"`)
		}
	})

	t.Run("single AND group multiple terms", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{And: []string{"Rust", "Engineer"}}})
		if q != `("Rust" AND "Engineer")` {
			t.Errorf("got %q, want %q", q, `("Rust" AND "Engineer")`)
		}
	})

	t.Run("multiple groups are ORed at top level", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{
			{Or: []string{"CTO", "Interim CTO"}},
			{And: []string{"Rust", "Blockchain"}},
		})
		want := `("CTO" OR "Interim CTO") OR ("Rust" AND "Blockchain")`
		if q != want {
			t.Errorf("got %q, want %q", q, want)
		}
	})

	t.Run("OR takes precedence over And when both set", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{Or: []string{"CTO"}, And: []string{"Rust"}}})
		if q != `"CTO"` {
			t.Errorf("got %q, want %q (Or must take precedence)", q, `"CTO"`)
		}
	})

	t.Run("dedup within OR group", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{Or: []string{"CTO", "cto", "CTO"}}})
		if q != `"CTO"` {
			t.Errorf("got %q, want %q", q, `"CTO"`)
		}
	})

	t.Run("dedup within AND group", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{And: []string{"Rust", "rust"}}})
		if q != `"Rust"` {
			t.Errorf("got %q, want %q", q, `"Rust"`)
		}
	})

	t.Run("all stripped terms produce empty result", func(t *testing.T) {
		q := BuildFTSQueryExprs([]SearchGroup{{Or: []string{"***", "^^^"}}})
		if q != "" {
			t.Errorf("got %q, want empty", q)
		}
	})

	t.Run("backward-compat flat OR group matches BuildFTSQuery output", func(t *testing.T) {
		terms := []string{"Fractional CTO", "CTO", "Blockchain developer"}
		old := BuildFTSQuery(terms)
		// BuildFTSQueryExprs with a single OR group should produce the same phrases,
		// wrapped in parens because there are multiple terms.
		qExprs := BuildFTSQueryExprs([]SearchGroup{{Or: terms}})
		// The exprs form wraps in parens; strip them to compare the inner phrases.
		inner := strings.TrimPrefix(strings.TrimSuffix(qExprs, ")"), "(")
		if inner != old {
			t.Errorf("inner exprs %q != BuildFTSQuery %q", inner, old)
		}
	})
}

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
