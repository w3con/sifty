package quickmatch_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/quickmatch"
)

func defaultRules() matchcontract.Rules {
	return matchcontract.Rules{
		SchemaVersion:      matchcontract.SchemaVersion,
		RulesVersion:       matchcontract.RulesVersion,
		Weights:            map[matchcontract.Group]float64{matchcontract.GroupRole: 0.25, matchcontract.GroupCompensation: 0.20, matchcontract.GroupGeography: 0.15, matchcontract.GroupConditions: 0.10},
		RequiredWeight:     2,
		PreferredWeight:    1,
		UnknownGoalUtility: matchcontract.DefaultUnknownGoalUtility,
		VerdictUtility:     matchcontract.DefaultVerdictUtilities(),
	}
}

func evidence(field, quote string, section matchcontract.Section, reliability matchcontract.Reliability) matchcontract.Evidence {
	return matchcontract.Evidence{Field: field, Quote: quote, Section: section, Reliability: reliability}
}

func mustFacts(t *testing.T, f matchcontract.VacancyFacts) matchcontract.VacancyFacts {
	t.Helper()
	f.JobID = 1
	f.SchemaVersion = matchcontract.SchemaVersion
	f.ExtractorVersion = matchcontract.ExtractorVersion
	f.SourcePolicyVersion = "source-policy/1"
	f.RawSourceHash = "raw-source-hash"
	if err := f.SetFactsHash(); err != nil {
		t.Fatalf("facts fixture is invalid: %v", err)
	}
	return f
}

func findComparison(res matchcontract.Result, criterion matchcontract.Criterion) (matchcontract.Comparison, bool) {
	for _, c := range res.Comparisons {
		if c.Criterion == criterion {
			return c, true
		}
	}
	return matchcontract.Comparison{}, false
}

func hasUnknownKind(res matchcontract.Result, kind string) bool {
	for _, u := range res.Unknowns {
		if u.Kind == kind {
			return true
		}
	}
	return false
}

func TestEvaluateDeterministicAndCorpusIndependent(t *testing.T) {
	profile := matchcontract.ProfileSnapshot{
		ProfileID: "local", ProfileRevision: 3,
		TargetRoles: []string{"Rust Engineer"},
		Skills: []matchcontract.ProfileSkill{{
			SkillID: "skill-rust", Name: "Rust", Level: "Expert",
			Provenance: &matchcontract.ProfileProvenance{Kind: "manual"},
		}},
	}
	jobA := mustFacts(t, matchcontract.VacancyFacts{
		Role: matchcontract.RoleFacts{Title: &matchcontract.TitleFact{
			Value:    "Rust Rust Rust Engineer",
			Evidence: evidence("title", "", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
		}},
		Professional: matchcontract.ProfessionalFacts{Requirements: []matchcontract.RequirementFact{
			{Label: "Rust", Text: "- Rust", Importance: matchcontract.ImportanceRequired, Evidence: evidence("", "- Rust", matchcontract.SectionRequired, matchcontract.ReliabilityUsable)},
			{Label: "Rust", Text: "- Rust", Importance: matchcontract.ImportanceRequired, Evidence: evidence("", "- Rust", matchcontract.SectionRequired, matchcontract.ReliabilityUsable)},
			{Label: "Rust and Go", Text: "- Rust and Go", Importance: matchcontract.ImportancePreferred, Evidence: evidence("", "- Rust and Go", matchcontract.SectionPreferred, matchcontract.ReliabilityUsable)},
		}},
	})
	jobB := mustFacts(t, matchcontract.VacancyFacts{
		Role: matchcontract.RoleFacts{Title: &matchcontract.TitleFact{
			Value:    "Product Designer",
			Evidence: evidence("title", "", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
		}},
	})

	first, err := quickmatch.Evaluate(profile, jobA, defaultRules())
	if err != nil {
		t.Fatalf("evaluate A: %v", err)
	}
	second, err := quickmatch.Evaluate(profile, jobA, defaultRules())
	if err != nil {
		t.Fatalf("evaluate A again: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same inputs produced different results")
	}
	if _, err := quickmatch.Evaluate(profile, jobB, defaultRules()); err != nil {
		t.Fatalf("evaluate B: %v", err)
	}
	third, err := quickmatch.Evaluate(profile, jobA, defaultRules())
	if err != nil {
		t.Fatalf("evaluate A after B: %v", err)
	}
	if !reflect.DeepEqual(first, third) {
		t.Fatalf("another vacancy changed this vacancy's result")
	}

	role, ok := findComparison(first, matchcontract.CriterionRoleTitle)
	if !ok || !role.Comparable || role.Utility == nil || math.Abs(*role.Utility-1) > 1e-12 {
		t.Fatalf("repeated title tokens should not lower role F1: %+v", role)
	}
	for _, c := range first.Comparisons {
		if c.Group == matchcontract.GroupProfessional {
			t.Fatalf("vacancy requirements are judged by You fit, not by the goal score: %+v", c)
		}
	}
	if _, ok := first.Applicability[matchcontract.GroupProfessional]; ok {
		t.Fatalf("the professional group is not part of the goal score: %+v", first.Applicability)
	}
}

func TestEvaluateSalaryCoverageAndZeroVsUnscored(t *testing.T) {
	profile := matchcontract.ProfileSnapshot{
		ProfileID: "local", ProfileRevision: 1,
		TargetRoles: []string{"Rust Engineer"},
		Skills:      []matchcontract.ProfileSkill{{Name: "Rust"}},
		Compensation: &matchcontract.ProfileCash{
			Minimum: floatPtr(90000), Currency: "USD", Period: matchcontract.PeriodYear, TaxBasis: matchcontract.TaxGross,
		},
		Locations: []matchcontract.ProfileLocation{{Country: "DE"}},
		WorkModes: []string{matchcontract.WorkRemote},
	}
	fullFacts := func(cash *matchcontract.CashFact) matchcontract.VacancyFacts {
		return mustFacts(t, matchcontract.VacancyFacts{
			Role: matchcontract.RoleFacts{Title: &matchcontract.TitleFact{Value: "Rust Engineer", Evidence: evidence("title", "", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)}},
			Professional: matchcontract.ProfessionalFacts{Requirements: []matchcontract.RequirementFact{{
				Label: "Rust", Importance: matchcontract.ImportanceRequired,
				Evidence: evidence("", "- Rust", matchcontract.SectionRequired, matchcontract.ReliabilityUsable),
			}}},
			Compensation: matchcontract.CompensationFacts{Cash: cash},
			Geography: matchcontract.GeographyFacts{HiringLocations: []matchcontract.LocationFact{{
				Label: "Germany", Country: "DE",
				Evidence: evidence("", "Location: Germany", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
			}}},
			Conditions: matchcontract.ConditionsFacts{WorkModes: []matchcontract.WorkModeFact{{
				Mode:     matchcontract.WorkRemote,
				Evidence: evidence("", "Remote", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
			}}},
		})
	}
	cash := &matchcontract.CashFact{
		Max: floatPtr(80000), Currency: "USD", Period: matchcontract.PeriodYear, TaxBasis: matchcontract.TaxGross,
		Basis:    matchcontract.CashBasisRangeCeiling,
		Evidence: evidence("", "Salary: 80,000 USD per year gross", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
	}
	res, err := quickmatch.Evaluate(profile, fullFacts(cash), defaultRules())
	if err != nil {
		t.Fatalf("evaluate full: %v", err)
	}
	// (25*1 + 20*8/9 + 15*1 + 10*1) / 70: role, compensation, geography, conditions only.
	if res.Status != matchcontract.StatusScored || res.Score == nil || *res.Score != 97 {
		t.Fatalf("known example score = %v/%v, want 97", res.Score, res.Status)
	}
	if res.RawScore == nil || math.Abs(*res.RawScore-0.9682539682539683) > 1e-9 {
		t.Fatalf("raw score = %v, want 0.96825", res.RawScore)
	}
	comp, ok := findComparison(res, matchcontract.CriterionCompensationCash)
	if !ok || !comp.Comparable || comp.Utility == nil || math.Abs(*comp.Utility-0.8888888888888888) > 1e-9 {
		t.Fatalf("compensation comparison = %+v", comp)
	}
	if math.Abs(res.Coverage.Weighted-1) > 1e-12 {
		t.Fatalf("full example coverage = %v, want 1", res.Coverage.Weighted)
	}

	missing, err := quickmatch.Evaluate(profile, fullFacts(nil), defaultRules())
	if err != nil {
		t.Fatalf("evaluate missing salary: %v", err)
	}
	if missing.Applicability[matchcontract.GroupCompensation] != matchcontract.Applicable ||
		missing.Coverage.Groups[matchcontract.GroupCompensation] != 0 {
		t.Fatalf("missing salary must stay applicable with zero coverage: %+v", missing.Coverage.Groups)
	}
	var assumed *matchcontract.Contribution
	for i, c := range missing.Contributions {
		if c.Group == matchcontract.GroupCompensation {
			assumed = &missing.Contributions[i]
		}
	}
	if assumed == nil || !assumed.Assumed || assumed.Utility != matchcontract.DefaultUnknownGoalUtility {
		t.Fatalf("missing salary must count as the midpoint and say so: %+v", missing.Contributions)
	}
	// (25*1 + 20*0.5 + 15*1 + 10*1) / 70 = 60/70: unknown pay is neither dropped nor a zero.
	if missing.Score == nil || *missing.Score != 86 {
		t.Fatalf("missing salary score = %v, want 86", missing.Score)
	}
	if math.Abs(missing.Coverage.Weighted-0.5/0.7) > 1e-12 {
		t.Fatalf("missing salary coverage = %v, want 5/7", missing.Coverage.Weighted)
	}
	if !hasUnknownKind(missing, matchcontract.ReasonCompensationMissing) {
		t.Fatalf("missing salary should be an explicit unknown")
	}

	skillsOnly := matchcontract.ProfileSnapshot{ProfileID: "local", Skills: []matchcontract.ProfileSkill{{Name: "Rust"}, {Name: "Go"}}}
	requirementsOnly := mustFacts(t, matchcontract.VacancyFacts{Professional: matchcontract.ProfessionalFacts{Requirements: []matchcontract.RequirementFact{
		{Label: "Rust", Importance: matchcontract.ImportanceRequired, Evidence: evidence("", "- Rust", matchcontract.SectionRequired, matchcontract.ReliabilityUsable)},
	}}})
	notGoals, err := quickmatch.Evaluate(skillsOnly, requirementsOnly, defaultRules())
	if err != nil {
		t.Fatalf("evaluate skills only: %v", err)
	}
	if notGoals.Status != matchcontract.StatusUnscored || notGoals.Score != nil {
		t.Fatalf("skills and requirements are not goals, so they cannot produce Fits you: %+v", notGoals)
	}

	zeroProfile := matchcontract.ProfileSnapshot{ProfileID: "local", TargetRoles: []string{"Designer"}}
	zeroFacts := mustFacts(t, matchcontract.VacancyFacts{Role: matchcontract.RoleFacts{Title: &matchcontract.TitleFact{
		Value: "Rust Engineer", Evidence: evidence("title", "", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
	}}})
	zero, err := quickmatch.Evaluate(zeroProfile, zeroFacts, defaultRules())
	if err != nil {
		t.Fatalf("evaluate zero: %v", err)
	}
	if zero.Status != matchcontract.StatusScored || zero.Score == nil || *zero.Score != 0 || zero.RawScore == nil || *zero.RawScore != 0 {
		t.Fatalf("genuine zero = %+v, want scored 0", zero)
	}

	unscored, err := quickmatch.Evaluate(matchcontract.ProfileSnapshot{ProfileID: "local"}, mustFacts(t, matchcontract.VacancyFacts{
		Role: matchcontract.RoleFacts{Title: &matchcontract.TitleFact{Value: "Rust Engineer", Evidence: evidence("title", "", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)}},
	}), defaultRules())
	if err != nil {
		t.Fatalf("evaluate unscored: %v", err)
	}
	if unscored.Status != matchcontract.StatusUnscored || unscored.Score != nil || unscored.RawScore != nil {
		t.Fatalf("no applicable goal must be unscored, not zero: %+v", unscored)
	}
	found := false
	for _, reason := range unscored.Reasons {
		if reason == matchcontract.ReasonNoComparableEvidence {
			found = true
		}
	}
	if !found {
		t.Fatalf("unscored result needs no_comparable_evidence reason")
	}
}

func TestEvaluateGeographyAndConditionsHonesty(t *testing.T) {
	location := func(label, country, city string, rel matchcontract.Reliability) matchcontract.LocationFact {
		return matchcontract.LocationFact{Label: label, Country: country, City: city, Evidence: evidence("", label, matchcontract.SectionUnknown, rel)}
	}
	evalGeo := func(t *testing.T, profile matchcontract.ProfileSnapshot, locs []matchcontract.LocationFact) matchcontract.Result {
		t.Helper()
		facts := mustFacts(t, matchcontract.VacancyFacts{Geography: matchcontract.GeographyFacts{HiringLocations: locs}})
		res, err := quickmatch.Evaluate(profile, facts, defaultRules())
		if err != nil {
			t.Fatalf("evaluate geography: %v", err)
		}
		return res
	}
	locationComparison := func(res matchcontract.Result) matchcontract.Comparison {
		t.Helper()
		c, ok := findComparison(res, matchcontract.CriterionGeographyLocation)
		if !ok {
			t.Fatalf("missing location comparison: %+v", res.Comparisons)
		}
		return c
	}

	t.Run("exact country is compatible", func(t *testing.T) {
		res := evalGeo(t, matchcontract.ProfileSnapshot{ProfileID: "local", Locations: []matchcontract.ProfileLocation{{Country: "DE"}}}, []matchcontract.LocationFact{location("Germany", "DE", "", matchcontract.ReliabilityUsable)})
		c := locationComparison(res)
		if !c.Comparable || c.Utility == nil || *c.Utility != 1 || c.Reason != matchcontract.ReasonLocationCompatible {
			t.Fatalf("compatible country = %+v", c)
		}
	})

	t.Run("country versus unmapped city stays unknown", func(t *testing.T) {
		res := evalGeo(t, matchcontract.ProfileSnapshot{ProfileID: "local", Locations: []matchcontract.ProfileLocation{{Country: "DE"}}}, []matchcontract.LocationFact{
			location("Berlin", "", "Berlin", matchcontract.ReliabilityUsable),
			location("France", "FR", "", matchcontract.ReliabilityUsable),
		})
		c := locationComparison(res)
		if c.Comparable || c.Utility != nil || c.Reason != matchcontract.ReasonLocationMappingUnsupported {
			t.Fatalf("unmapped city must not become a false conflict: %+v", c)
		}
	})

	t.Run("exclusive country is disjoint", func(t *testing.T) {
		res := evalGeo(t, matchcontract.ProfileSnapshot{ProfileID: "local", Locations: []matchcontract.ProfileLocation{{Country: "DE"}}}, []matchcontract.LocationFact{location("France", "FR", "", matchcontract.ReliabilityUsable)})
		c := locationComparison(res)
		if !c.Comparable || c.Utility == nil || *c.Utility != 0 || c.Reason != matchcontract.ReasonLocationDisjoint {
			t.Fatalf("disjoint country = %+v", c)
		}
	})

	t.Run("labels are not substring locations", func(t *testing.T) {
		res := evalGeo(t, matchcontract.ProfileSnapshot{ProfileID: "local", Locations: []matchcontract.ProfileLocation{{Country: "US"}}}, []matchcontract.LocationFact{location("Australia", "", "", matchcontract.ReliabilityUsable)})
		c := locationComparison(res)
		if c.Comparable || c.Utility != nil {
			t.Fatalf("US must not substring-match Australia: %+v", c)
		}
	})

	t.Run("unreliable alternative blocks disjoint", func(t *testing.T) {
		res := evalGeo(t, matchcontract.ProfileSnapshot{ProfileID: "local", Locations: []matchcontract.ProfileLocation{{Country: "DE"}}}, []matchcontract.LocationFact{location("France", "FR", "", matchcontract.ReliabilityNeedsCorroboration)})
		c := locationComparison(res)
		if c.Comparable || c.Utility != nil || c.Reason != matchcontract.ReasonLocationUnusable {
			t.Fatalf("needs-corroboration location must stay unknown: %+v", c)
		}
	})

	t.Run("work authorization is not a goal", func(t *testing.T) {
		facts := mustFacts(t, matchcontract.VacancyFacts{Geography: matchcontract.GeographyFacts{WorkAuthorization: &matchcontract.AuthorizationFact{
			Country: "DE", Requirement: matchcontract.AuthorizationRequired,
			Evidence: evidence("", "must be authorized in Germany", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
		}}})
		for _, status := range []string{"eligible", "need_visa"} {
			profile := matchcontract.ProfileSnapshot{ProfileID: "local", Authorizations: []matchcontract.ProfileAuthorization{{Country: "DE", Status: status}}}
			res, err := quickmatch.Evaluate(profile, facts, defaultRules())
			if err != nil {
				t.Fatalf("evaluate auth %s: %v", status, err)
			}
			if _, ok := findComparison(res, matchcontract.CriterionGeographyAuthorization); ok {
				t.Fatalf("%s: authorization must not be compared as a goal: %+v", status, res.Comparisons)
			}
			if res.Status != matchcontract.StatusUnscored || res.Applicability[matchcontract.GroupGeography] == matchcontract.Applicable {
				t.Fatalf("%s: authorization alone must not make a goal applicable: %+v", status, res)
			}
		}
	})

	t.Run("conditions partial unknown and timezone unsupported", func(t *testing.T) {
		profile := matchcontract.ProfileSnapshot{
			ProfileID: "local",
			WorkModes: []string{matchcontract.WorkRemote},
			Hours:     &matchcontract.ProfileHours{Mode: matchcontract.HoursFullTime, Weekly: floatPtr(40)},
			Timezones: []string{"Europe/Berlin"},
		}
		facts := mustFacts(t, matchcontract.VacancyFacts{Conditions: matchcontract.ConditionsFacts{
			WorkModes: []matchcontract.WorkModeFact{{Mode: matchcontract.WorkOffice, Evidence: evidence("", "Office", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)}},
			Hours:     &matchcontract.HoursFact{Mode: matchcontract.HoursFullTime, Evidence: evidence("", "Full-time", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)},
			Timezone:  &matchcontract.TimezoneFact{Value: "CET", Evidence: evidence("", "CET", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)},
		}})
		res, err := quickmatch.Evaluate(profile, facts, defaultRules())
		if err != nil {
			t.Fatalf("evaluate conditions: %v", err)
		}
		mode, _ := findComparison(res, matchcontract.CriterionConditionsWorkMode)
		if !mode.Comparable || mode.Utility == nil || *mode.Utility != 0 || mode.Reason != matchcontract.ReasonWorkModeDisjoint {
			t.Fatalf("work mode = %+v", mode)
		}
		hours, _ := findComparison(res, matchcontract.CriterionConditionsHours)
		if hours.Comparable || hours.Utility != nil || hours.Reason != matchcontract.ReasonHoursPartialUnknown {
			t.Fatalf("partial hours = %+v", hours)
		}
		tz, _ := findComparison(res, matchcontract.CriterionConditionsTimezone)
		if tz.Comparable || tz.Utility != nil || tz.Reason != matchcontract.ReasonTimezoneUnsupported {
			t.Fatalf("timezone = %+v", tz)
		}
		if math.Abs(res.Coverage.Groups[matchcontract.GroupConditions]-1.0/3.0) > 1e-12 {
			t.Fatalf("conditions coverage = %v, want 1/3", res.Coverage.Groups[matchcontract.GroupConditions])
		}
	})

	t.Run("missing condition fact is unknown not zero", func(t *testing.T) {
		res, err := quickmatch.Evaluate(matchcontract.ProfileSnapshot{ProfileID: "local", WorkModes: []string{matchcontract.WorkRemote}}, mustFacts(t, matchcontract.VacancyFacts{}), defaultRules())
		if err != nil {
			t.Fatalf("evaluate conditions: %v", err)
		}
		c, _ := findComparison(res, matchcontract.CriterionConditionsWorkMode)
		if c.Comparable || c.Utility != nil || c.Reason != matchcontract.ReasonWorkModeMissing {
			t.Fatalf("missing work mode = %+v", c)
		}
	})

	t.Run("unreliable work mode alternative blocks disjoint", func(t *testing.T) {
		facts := mustFacts(t, matchcontract.VacancyFacts{Conditions: matchcontract.ConditionsFacts{WorkModes: []matchcontract.WorkModeFact{
			{Mode: matchcontract.WorkOffice, Evidence: evidence("", "Office", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)},
			{Mode: matchcontract.WorkRemote, Evidence: evidence("", "Remote possible", matchcontract.SectionUnknown, matchcontract.ReliabilityNeedsCorroboration)},
		}}})
		res, err := quickmatch.Evaluate(matchcontract.ProfileSnapshot{ProfileID: "local", WorkModes: []string{matchcontract.WorkRemote}}, facts, defaultRules())
		if err != nil {
			t.Fatalf("evaluate work modes: %v", err)
		}
		c, _ := findComparison(res, matchcontract.CriterionConditionsWorkMode)
		if c.Comparable || c.Utility != nil || c.Reason != matchcontract.ReasonWorkModeUnusable ||
			c.EvidenceQuote != "Remote possible" {
			t.Fatalf("unreliable alternative must keep the group unknown: %+v", c)
		}
	})

	t.Run("hours mode missing is not credited by an overlapping range", func(t *testing.T) {
		profile := matchcontract.ProfileSnapshot{
			ProfileID: "local",
			Hours:     &matchcontract.ProfileHours{Mode: matchcontract.HoursFullTime, Weekly: floatPtr(40)},
		}
		facts := mustFacts(t, matchcontract.VacancyFacts{Conditions: matchcontract.ConditionsFacts{
			Hours: &matchcontract.HoursFact{Weekly: floatPtr(40), Evidence: evidence("", "40 hours", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)},
		}})
		res, err := quickmatch.Evaluate(profile, facts, defaultRules())
		if err != nil {
			t.Fatalf("evaluate hours: %v", err)
		}
		c, _ := findComparison(res, matchcontract.CriterionConditionsHours)
		if c.Comparable || c.Utility != nil || c.Reason != matchcontract.ReasonHoursPartialUnknown {
			t.Fatalf("missing job hours mode must not become full fit: %+v", c)
		}
	})
}

func floatPtr(v float64) *float64 { return &v }
