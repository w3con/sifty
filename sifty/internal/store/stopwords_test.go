package store

import "testing"

func TestTitleMatchesStopWords(t *testing.T) {
	words := func(ws ...string) []*StopWord {
		var out []*StopWord
		for _, w := range ws {
			out = append(out, &StopWord{Word: w})
		}
		return out
	}
	tests := []struct {
		name  string
		title string
		words []*StopWord
		want  bool
		by    string
	}{
		{"single word", "Sales Manager", words("sales"), true, "sales"},
		{"single word no hit", "Engineering Manager", words("sales"), false, ""},
		{"multi-word sequence", "Senior Risk Engineer", words("risk engineer"), true, "risk engineer"},
		{"multi-word broken by other word", "Risk and Engineer", words("risk engineer"), false, ""},
		{"case-insensitive title", "SENIOR RISK ENGINEER", words("risk engineer"), true, "risk engineer"},
		{"case-insensitive word", "senior risk engineer", words("Risk Engineer"), true, "Risk Engineer"},
		{"substring inside a word", "Administrator", words("admin"), true, "admin"},
		{"no words", "Sales Manager", nil, false, ""},
		{"second word matches", "Account Executive", words("sales", "account"), true, "account"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, by := titleMatchesStopWords(tc.title, tc.words)
			if got != tc.want || by != tc.by {
				t.Errorf("titleMatchesStopWords(%q) = (%v, %q), want (%v, %q)", tc.title, got, by, tc.want, tc.by)
			}
		})
	}
}
