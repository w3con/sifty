package quickmatch_test

// TestUnrestrictedRemote covers the rule introduced by INBOX-4 stage G.1:
// a vacancy whose remote work is unrestricted (worldwide labels, or remote
// with no location label) scores as a geography match (utility 1, reason
// remote_unrestricted) when the owner has any geography data.  Cases where
// the owner has no geography data, and all restricted / remoteNoRemote cases,
// are unchanged.
//
// The last four cases (G.3) test TELECOMMUTE vocabulary and "|" separator added
// in INBOX-4 stage G.3 using the embedded regions.default.yaml vocabulary.
//
// Run as:
//
//	go test ./internal/quickmatch/ -run TestUnrestrictedRemote -v
import (
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/quickmatch"
)

func TestUnrestrictedRemote(t *testing.T) {
	rulesWithRegions := func() matchcontract.Rules {
		r := defaultRules()
		r.Regions = []matchcontract.RegionEntry{
			{Name: "Europe", Countries: []string{"FR", "DE", "NL"}},
			{Name: "Anywhere", Countries: nil},        // worldwide label
			{Name: "Worldwide", Countries: nil},       // worldwide label
			{Name: "Remote (Global)", Countries: nil}, // worldwide label
		}
		r.Separators = []string{"/", ","}
		return r
	}

	// rulesWithDefaultVocab loads the actual embedded regions.default.yaml, which
	// is the vocabulary modified by INBOX-4 G.3 (adds telecommute alias and | separator).
	// Used for TELECOMMUTE test cases.
	rulesWithDefaultVocab := func(t *testing.T) matchcontract.Rules {
		t.Helper()
		regions, err := config.DefaultRegions()
		if err != nil {
			t.Fatalf("config.DefaultRegions: %v", err)
		}
		seps, err := config.DefaultSeparators()
		if err != nil {
			t.Fatalf("config.DefaultSeparators: %v", err)
		}
		r := defaultRules()
		for _, re := range regions {
			r.Regions = append(r.Regions, matchcontract.RegionEntry{
				Name:      re.Name,
				Aliases:   append([]string(nil), re.Aliases...),
				Countries: append([]string(nil), re.Countries...),
			})
		}
		r.Separators = seps
		return r
	}

	// remoteUnrestrictedFacts returns facts for a remote vacancy with the given
	// hiring location labels (may be nil for bare "Remote").
	remoteUnrestrictedFacts := func(t *testing.T, labels []string) matchcontract.VacancyFacts {
		t.Helper()
		var locs []matchcontract.LocationFact
		for _, l := range labels {
			locs = append(locs, matchcontract.LocationFact{
				Label: l,
				Evidence: matchcontract.Evidence{
					Field:       "location",
					Quote:       l,
					Section:     matchcontract.SectionUnknown,
					Reliability: matchcontract.ReliabilityUsable,
				},
			})
		}
		return mustFacts(t, matchcontract.VacancyFacts{
			Geography: matchcontract.GeographyFacts{HiringLocations: locs},
			Conditions: matchcontract.ConditionsFacts{WorkModes: []matchcontract.WorkModeFact{{
				Mode:     matchcontract.WorkRemote,
				Evidence: matchcontract.Evidence{Field: "work_mode", Quote: "Remote", Section: matchcontract.SectionUnknown, Reliability: matchcontract.ReliabilityUsable},
			}}},
		})
	}

	// remoteRestrictedFacts returns facts for a remote vacancy restricted to Europe.
	remoteRestrictedFacts := func(t *testing.T) matchcontract.VacancyFacts {
		t.Helper()
		return mustFacts(t, matchcontract.VacancyFacts{
			Geography: matchcontract.GeographyFacts{HiringLocations: []matchcontract.LocationFact{{
				Label:   "Europe",
				Country: "",
				Evidence: matchcontract.Evidence{
					Field: "location", Quote: "Europe",
					Section: matchcontract.SectionUnknown, Reliability: matchcontract.ReliabilityUsable,
				},
			}}},
			Conditions: matchcontract.ConditionsFacts{WorkModes: []matchcontract.WorkModeFact{{
				Mode:     matchcontract.WorkRemote,
				Evidence: matchcontract.Evidence{Field: "work_mode", Quote: "Remote", Section: matchcontract.SectionUnknown, Reliability: matchcontract.ReliabilityUsable},
			}}},
		})
	}

	restrictionComparison := func(res matchcontract.Result) (matchcontract.Comparison, bool) {
		return findComparison(res, matchcontract.CriterionGeographyRemoteRestriction)
	}

	t.Run("unrestricted with authorizations only", func(t *testing.T) {
		// Owner has authorizations (FR) but no preferred locations.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "FR", Status: "eligible"}},
		}
		facts := remoteUnrestrictedFacts(t, []string{"Anywhere"})
		res, err := quickmatch.Evaluate(profile, facts, rulesWithRegions())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison, got none; comparisons: %+v", res.Comparisons)
		}
		if !c.Comparable || c.Utility == nil || *c.Utility != 1 {
			t.Fatalf("unrestricted with authorizations: want utility 1, got %+v", c)
		}
		if c.Reason != matchcontract.ReasonRemoteUnrestricted {
			t.Fatalf("unrestricted with authorizations: want reason %q, got %q", matchcontract.ReasonRemoteUnrestricted, c.Reason)
		}
		if res.Applicability[matchcontract.GroupGeography] != matchcontract.Applicable {
			t.Fatalf("geography group must be applicable: %+v", res.Applicability)
		}
	})

	t.Run("unrestricted with preferred location (Niger)", func(t *testing.T) {
		// Owner has preferred location Niger; no authorizations.  The location comparison
		// is replaced by the unrestricted match (Niger is not a known country code, but
		// the unrestricted rule does not care about location resolution).
		profile := matchcontract.ProfileSnapshot{
			ProfileID: "local",
			Locations: []matchcontract.ProfileLocation{{Country: "NE"}}, // Niger ISO code
		}
		facts := remoteUnrestrictedFacts(t, []string{"Worldwide"})
		res, err := quickmatch.Evaluate(profile, facts, rulesWithRegions())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison, got none; comparisons: %+v", res.Comparisons)
		}
		if !c.Comparable || c.Utility == nil || *c.Utility != 1 {
			t.Fatalf("unrestricted with preferred location: want utility 1, got %+v", c)
		}
		if c.Reason != matchcontract.ReasonRemoteUnrestricted {
			t.Fatalf("unrestricted with preferred location: want reason %q, got %q", matchcontract.ReasonRemoteUnrestricted, c.Reason)
		}
		// No location comparison member (replaced).
		if _, ok := findComparison(res, matchcontract.CriterionGeographyLocation); ok {
			t.Fatalf("location comparison must be absent when unrestricted rule fires: %+v", res.Comparisons)
		}
	})

	t.Run("unrestricted with no geography data — not applicable", func(t *testing.T) {
		// Owner has no locations, no authorizations, no current location.
		profile := matchcontract.ProfileSnapshot{ProfileID: "local"}
		facts := remoteUnrestrictedFacts(t, []string{"Anywhere"})
		res, err := quickmatch.Evaluate(profile, facts, rulesWithRegions())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if res.Applicability[matchcontract.GroupGeography] == matchcontract.Applicable {
			t.Fatalf("geography must not be applicable with no owner geography data: %+v", res.Applicability)
		}
		if _, ok := restrictionComparison(res); ok {
			t.Fatalf("no remote_restriction comparison expected when owner has no geography data")
		}
	})

	t.Run("bare remote with no label — unrestricted match", func(t *testing.T) {
		// No hiring location labels at all ("Remote" with nothing else).
		// jobValue must be the display constant.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "FR", Status: "eligible"}},
		}
		facts := remoteUnrestrictedFacts(t, nil) // no labels
		res, err := quickmatch.Evaluate(profile, facts, rulesWithRegions())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison, got none")
		}
		if !c.Comparable || c.Utility == nil || *c.Utility != 1 {
			t.Fatalf("bare remote: want utility 1, got %+v", c)
		}
		if c.Reason != matchcontract.ReasonRemoteUnrestricted {
			t.Fatalf("bare remote: want reason %q, got %q", matchcontract.ReasonRemoteUnrestricted, c.Reason)
		}
		if c.JobValue != "remote, no region stated" {
			t.Fatalf("bare remote job value: want %q, got %q", "remote, no region stated", c.JobValue)
		}
	})

	t.Run("restricted covered — unchanged", func(t *testing.T) {
		// FR authorization covers Europe restriction.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "FR", Status: "eligible"}},
		}
		facts := remoteRestrictedFacts(t)
		res, err := quickmatch.Evaluate(profile, facts, rulesWithRegions())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison")
		}
		if c.Reason != matchcontract.ReasonRemoteRestrictedCovered {
			t.Fatalf("restricted covered: want %q, got %q", matchcontract.ReasonRemoteRestrictedCovered, c.Reason)
		}
		if c.Utility == nil || *c.Utility != 1 {
			t.Fatalf("restricted covered: want utility 1, got %+v", c)
		}
	})

	t.Run("restricted not covered — unchanged", func(t *testing.T) {
		// NZ authorization does not cover Europe.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "NZ", Status: "eligible"}},
		}
		facts := remoteRestrictedFacts(t)
		res, err := quickmatch.Evaluate(profile, facts, rulesWithRegions())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison")
		}
		if c.Reason != matchcontract.ReasonRemoteRestrictedNotCovered {
			t.Fatalf("restricted not covered: want %q, got %q", matchcontract.ReasonRemoteRestrictedNotCovered, c.Reason)
		}
		if c.Utility == nil || *c.Utility != 0 {
			t.Fatalf("restricted not covered: want utility 0, got %+v", c)
		}
	})

	t.Run("restricted unknown — unchanged", func(t *testing.T) {
		// No authorizations; restricted vacancy produces unknown verdict.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:       "local",
			CurrentLocation: &matchcontract.ProfileLocation{Country: "NZ"},
		}
		facts := remoteRestrictedFacts(t)
		res, err := quickmatch.Evaluate(profile, facts, rulesWithRegions())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison")
		}
		if c.Reason != matchcontract.ReasonRemoteRestrictedUnknown {
			t.Fatalf("restricted unknown: want %q, got %q", matchcontract.ReasonRemoteRestrictedUnknown, c.Reason)
		}
		if c.Utility != nil {
			t.Fatalf("restricted unknown: want nil utility (not comparable), got %v", *c.Utility)
		}
	})

	// G.3 cases: TELECOMMUTE alias and "|" separator in regions.default.yaml.

	t.Run("G.3: Anywhere + TELECOMMUTE as separate labels → unrestricted", func(t *testing.T) {
		// Schema.org sources store "Anywhere" and "TELECOMMUTE" as two separate
		// hiringLocations entries, not one compound label. Each is processed
		// independently by classifyRemoteRestriction: both resolve to worldwide →
		// remoteUnrestricted.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "FR", Status: "eligible"}},
		}
		facts := remoteUnrestrictedFacts(t, []string{"Anywhere", "TELECOMMUTE"})
		res, err := quickmatch.Evaluate(profile, facts, rulesWithDefaultVocab(t))
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison, got none; comparisons: %+v", res.Comparisons)
		}
		if c.Reason != matchcontract.ReasonRemoteUnrestricted {
			t.Fatalf("Anywhere+TELECOMMUTE: want reason %q, got %q", matchcontract.ReasonRemoteUnrestricted, c.Reason)
		}
		if !c.Comparable || c.Utility == nil || *c.Utility != 1 {
			t.Fatalf("Anywhere+TELECOMMUTE: want utility 1, got %+v", c)
		}
	})

	t.Run("G.3: TELECOMMUTE alone → unrestricted", func(t *testing.T) {
		// A single "TELECOMMUTE" label (schema.org jobLocationType) means worldwide remote.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "FR", Status: "eligible"}},
		}
		facts := remoteUnrestrictedFacts(t, []string{"TELECOMMUTE"})
		res, err := quickmatch.Evaluate(profile, facts, rulesWithDefaultVocab(t))
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison, got none; comparisons: %+v", res.Comparisons)
		}
		if c.Reason != matchcontract.ReasonRemoteUnrestricted {
			t.Fatalf("TELECOMMUTE alone: want reason %q, got %q", matchcontract.ReasonRemoteUnrestricted, c.Reason)
		}
		if !c.Comparable || c.Utility == nil || *c.Utility != 1 {
			t.Fatalf("TELECOMMUTE alone: want utility 1, got %+v", c)
		}
	})

	t.Run("G.3: TELECOMMUTE + US as separate labels → not covered (owner FR)", func(t *testing.T) {
		// Two separate hiringLocations: "TELECOMMUTE" (worldwide) and "US" (restricted).
		// classifyRemoteRestriction sees one restricted label → remoteRestricted.
		// Owner has FR authorization → not covered.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "FR", Status: "eligible"}},
		}
		facts := remoteUnrestrictedFacts(t, []string{"TELECOMMUTE", "US"})
		res, err := quickmatch.Evaluate(profile, facts, rulesWithDefaultVocab(t))
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison, got none; comparisons: %+v", res.Comparisons)
		}
		if c.Reason != matchcontract.ReasonRemoteRestrictedNotCovered {
			t.Fatalf("TELECOMMUTE+US: want reason %q, got %q", matchcontract.ReasonRemoteRestrictedNotCovered, c.Reason)
		}
		if c.Utility == nil || *c.Utility != 0 {
			t.Fatalf("TELECOMMUTE+US: want utility 0, got %+v", c)
		}
	})

	t.Run("G.3: DE + TELECOMMUTE as separate labels → not covered (owner FR)", func(t *testing.T) {
		// Two separate hiringLocations: "DE" (Germany, restricted) and "TELECOMMUTE" (worldwide).
		// classifyRemoteRestriction sees one restricted label → remoteRestricted with [DE].
		// Owner has FR authorization → DE does not cover FR → not covered.
		profile := matchcontract.ProfileSnapshot{
			ProfileID:      "local",
			Authorizations: []matchcontract.ProfileAuthorization{{Country: "FR", Status: "eligible"}},
		}
		facts := remoteUnrestrictedFacts(t, []string{"DE", "TELECOMMUTE"})
		res, err := quickmatch.Evaluate(profile, facts, rulesWithDefaultVocab(t))
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		c, ok := restrictionComparison(res)
		if !ok {
			t.Fatalf("expected remote_restriction comparison, got none; comparisons: %+v", res.Comparisons)
		}
		if c.Reason != matchcontract.ReasonRemoteRestrictedNotCovered {
			t.Fatalf("DE+TELECOMMUTE: want reason %q, got %q", matchcontract.ReasonRemoteRestrictedNotCovered, c.Reason)
		}
		if c.Utility == nil || *c.Utility != 0 {
			t.Fatalf("DE+TELECOMMUTE: want utility 0, got %+v", c)
		}
	})
}
