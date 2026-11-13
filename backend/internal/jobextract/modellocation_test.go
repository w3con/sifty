package jobextract_test

import (
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchconfig"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/quickmatch"
)

// TestModelLocationUsableOnlyWhenQuoteStatesRestriction catches a model-read location deciding a
// remote restriction from a bare place name, and a restriction quote ("US/Canada only") staying
// "To clarify" because the phrases were lost between regions.default.yaml and the rules.
func TestModelLocationUsableOnlyWhenQuoteStatesRestriction(t *testing.T) {
	rules, err := matchconfig.Rules(config.DefaultQuickMatch())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules.RestrictionPhrases) == 0 {
		t.Fatal("built-in restriction phrases did not reach the rules")
	}
	owner := matchcontract.ProfileSnapshot{ProfileID: "local", Authorizations: []matchcontract.ProfileAuthorization{
		{Country: "FR", Status: "eligible"}, {Country: "NZ", Status: "eligible"}, {Country: "AU", Status: "eligible"},
	}}
	const description = "We are a distributed team. Everyone works remotely."

	for _, tc := range []struct {
		name, value, quote string
		reliability        matchcontract.Reliability
		list               func(matchcontract.FitLists) []matchcontract.FitLine
		reason             string
	}{
		{"restriction quote", "US/Canada", "Remote (US/Canada only)", matchcontract.ReliabilityUsable,
			func(l matchcontract.FitLists) []matchcontract.FitLine { return l.Mismatches }, matchcontract.ReasonRemoteRestrictedNotCovered},
		{"bare country", "United States", "United States", matchcontract.ReliabilityNeedsCorroboration,
			func(l matchcontract.FitLists) []matchcontract.FitLine { return l.ToClarify }, matchcontract.ReasonRemoteRestrictedUnknown},
		{"bare region", "Europe", "Remote, Europe", matchcontract.ReliabilityNeedsCorroboration,
			func(l matchcontract.FitLists) []matchcontract.FitLine { return l.ToClarify }, matchcontract.ReasonRemoteRestrictedUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := factsFor(t, description)
			if len(f.Geography.HiringLocations) != 0 {
				t.Fatalf("fixture must have no location found by the ordinary parse: %+v", f.Geography.HiringLocations)
			}
			merged, _, err := jobextract.MergeModelConditions(f, matchcontract.ExtractedConditions{
				WorkMode: cond("remote", "Everyone works remotely"),
				Location: cond(tc.value, tc.quote),
			}, rules.RestrictionPhrases)
			if err != nil {
				t.Fatal(err)
			}
			for _, loc := range merged.Geography.HiringLocations {
				if loc.Evidence.Reliability != tc.reliability {
					t.Fatalf("location %q: reliability %q, want %q", loc.Label, loc.Evidence.Reliability, tc.reliability)
				}
			}
			res, err := quickmatch.EvaluateFit(matchcontract.FitInput{
				Profile: owner, Source: matchcontract.SourceUsable, Facts: &merged,
				DescriptionHash: matchcontract.HashDescription(description), Rules: rules,
			})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, line := range tc.list(res.Lists) {
				if line.Criterion == matchcontract.CriterionGeographyRemoteRestriction && line.Reason == tc.reason {
					found = true
				}
			}
			if !found {
				t.Fatalf("want a %s line with reason %s; lists: %+v", matchcontract.CriterionGeographyRemoteRestriction, tc.reason, res.Lists)
			}
		})
	}
}
