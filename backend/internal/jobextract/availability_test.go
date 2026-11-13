package jobextract_test

import (
	"testing"

	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
)

func TestSourceStateOf(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  matchcontract.RetainedSource
		want matchcontract.SourceState
	}{
		{"structured description", matchcontract.RetainedSource{Description: "You know Go.", DescriptionStatus: "available structured description"}, matchcontract.SourceUsable},
		{"short genuine text", matchcontract.RetainedSource{Description: "Go dev, remote.", DescriptionStatus: "available"}, matchcontract.SourceUsable},
		{"skills metadata", matchcontract.RetainedSource{Description: "Go, Rust, SQL", DescriptionStatus: "skills metadata only; full description pending"}, matchcontract.SourceUnavailable},
		{"fetch failed", matchcontract.RetainedSource{Description: "detail unavailable: HTTP 429", DescriptionStatus: "detail unavailable: Error: HTTP 429"}, matchcontract.SourceUnavailable},
		{"blocked", matchcontract.RetainedSource{DescriptionStatus: "blocked: 403"}, matchcontract.SourceUnavailable},
		{"empty", matchcontract.RetainedSource{DescriptionStatus: "available"}, matchcontract.SourceUnavailable},
		{"title repeated", matchcontract.RetainedSource{Title: "Rust Engineer", Description: "Rust Engineer", DescriptionStatus: "available"}, matchcontract.SourceUnavailable},
		{"tags repeated", matchcontract.RetainedSource{Title: "Dev", Tags: []string{"go", "rust"}, Description: "Dev go, rust", DescriptionStatus: "available"}, matchcontract.SourceUnavailable},
	} {
		if got := jobextract.SourceStateOf(tc.src); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
}
