package store

import "testing"

// TestRoleMatchesExclusions verifies that RoleMatchesExclusions checks both title and company.
func TestRoleMatchesExclusions(t *testing.T) {
	words := func(ws ...string) []*StopWord {
		var out []*StopWord
		for _, w := range ws {
			out = append(out, &StopWord{Word: w})
		}
		return out
	}
	tests := []struct {
		name    string
		title   string
		company string
		words   []*StopWord
		want    bool
	}{
		{"match in title", "Sales Manager", "GoodCo", words("sales"), true},
		{"match in company", "Engineer", "Acme Sales Corp", words("sales"), true},
		{"match in neither", "Engineer", "GoodCo", words("sales"), false},
		{"case insensitive title", "SALES MANAGER", "GoodCo", words("sales"), true},
		{"case insensitive company", "Engineer", "ACME SALES CORP", words("sales"), true},
		{"empty title match company", "", "Sales Corp", words("sales"), true},
		{"empty company match title", "Sales Lead", "", words("sales"), true},
		{"both empty no match", "", "", words("sales"), false},
		{"no words", "Sales Manager", "GoodCo", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := RoleMatchesExclusions(tc.title, tc.company, tc.words)
			if got != tc.want {
				t.Errorf("RoleMatchesExclusions(%q, %q) = %v, want %v", tc.title, tc.company, got, tc.want)
			}
		})
	}
}

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
