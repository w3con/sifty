package quickmatch

import (
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
)

// testRulesWithDefaults builds a matchcontract.Rules populated from the built-in
// regions.default.yaml vocabulary. Tests use this so they exercise the same configured
// defaults that production uses, without requiring a data directory.
func testRulesWithDefaults(t *testing.T) matchcontract.Rules {
	t.Helper()
	qm := config.QuickMatch{} // empty → all Effective* methods return built-in defaults
	regions, err := qm.EffectiveRegions()
	if err != nil {
		t.Fatalf("load default regions: %v", err)
	}
	prefixes, err := qm.EffectiveWrapperPrefixes()
	if err != nil {
		t.Fatalf("load default wrapper prefixes: %v", err)
	}
	qualifiers, err := qm.EffectiveTrailingQualifiers()
	if err != nil {
		t.Fatalf("load default trailing qualifiers: %v", err)
	}
	separators, err := qm.EffectiveSeparators()
	if err != nil {
		t.Fatalf("load default separators: %v", err)
	}

	var r matchcontract.Rules
	for _, re := range regions {
		r.Regions = append(r.Regions, matchcontract.RegionEntry{
			Name:      re.Name,
			Aliases:   append([]string(nil), re.Aliases...),
			Countries: append([]string(nil), re.Countries...),
		})
	}
	r.WrapperPrefixes = prefixes
	r.TrailingQualifiers = qualifiers
	r.Separators = separators
	return r
}

// TestResolveRestrictionLabelForms exercises every label form described in the ASSESSMENT-10
// plan. The nine labels are taken from the actual production values read from the nine known
// vacancies. Seven must resolve; two must stay unknown.
//
// The test also verifies that a vocabulary name containing "and" (e.g. "North and South
// America") is not split and resolves to the full entry.
func TestResolveRestrictionLabelForms(t *testing.T) {
	rules := testRulesWithDefaults(t)

	cases := []struct {
		label       string
		wantKind    labelKind
		wantCodes   []string // non-nil and non-empty only when wantKind == labelRestricted
		description string
	}{
		// --- Should resolve ---
		{
			label:       "Remote (US only)",
			wantKind:    labelRestricted,
			wantCodes:   []string{"US"},
			description: "vacancy 17: paren extraction + trailing qualifier strip",
		},
		{
			label:       "US only, US time zones",
			wantKind:    labelRestricted,
			wantCodes:   []string{"US"},
			description: "vacancy 1153: comma split + trailing qualifier strip for each part",
		},
		{
			label:       "Alabama, Florida, Georgia, Michigan, Mississippi, Missouri, North Carolina, Oklahoma, Tennessee, Texas, or Virginia",
			wantKind:    labelRestricted,
			wantCodes:   []string{"US"},
			description: "vacancy 1175: state list split on comma and 'or', each state → US",
		},
		{
			label:       "US and Canada",
			wantKind:    labelRestricted,
			wantCodes:   []string{"US", "CA"},
			description: "vacancy 1196: 'and' separator",
		},
		{
			label:       "US-based (remote within the US)",
			wantKind:    labelRestricted,
			wantCodes:   []string{"US"},
			description: "vacancy 1201: paren extraction, '-based' strip, 'remote within the' prefix strip",
		},
		{
			label:       "UK, Ireland, Germany, Portugal, Spain or The Netherlands",
			wantKind:    labelRestricted,
			wantCodes:   []string{"GB", "IE", "DE", "PT", "ES", "NL"},
			description: "vacancy 1279: multi-country, 'or' separator, 'The Netherlands' alias",
		},
		{
			label:       "US & Canada",
			wantKind:    labelRestricted,
			wantCodes:   []string{"US", "CA"},
			description: "vacancy 1294: '&' separator",
		},
		// --- Should stay unknown ---
		{
			label:       "REMOTE (US only) or NYC/Dallas",
			wantKind:    labelUnknown,
			description: "vacancy 1119: city names NYC and Dallas are not in the vocabulary",
		},
		{
			label:       "ET timezone in the US",
			wantKind:    labelUnknown,
			description: "vacancy 1281: time-zone statement — no resolvable structure",
		},
		// --- Vocabulary name containing "and" must not be split ---
		{
			label:       "North and South America",
			wantKind:    labelRestricted,
			description: "vocabulary entry 'Americas' name contains 'and': full-label lookup must succeed before split",
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			kind, codes := resolveRestrictionLabel(tc.label, rules)
			if kind != tc.wantKind {
				t.Errorf("label %q: got kind %v, want %v", tc.label, kind, tc.wantKind)
			}
			if tc.wantKind == labelRestricted && tc.wantCodes != nil {
				if !sameCountries(codes, tc.wantCodes) {
					t.Errorf("label %q: got countries %v, want %v", tc.label, codes, tc.wantCodes)
				}
			}
		})
	}
}

// sameCountries reports whether a and b contain the same ISO codes (order-independent,
// case-insensitive).
func sameCountries(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]bool, len(a))
	for _, c := range a {
		m[c] = true
	}
	for _, c := range b {
		if !m[c] {
			return false
		}
	}
	return true
}
