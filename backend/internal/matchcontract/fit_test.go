package matchcontract_test

import (
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/matchcontract"
)

func extractionOf(reqs ...matchcontract.ExtractedRequirement) matchcontract.VacancyExtraction {
	return matchcontract.VacancyExtraction{
		SchemaVersion: matchcontract.SchemaVersion, ExtractorVersion: "test/1",
		JobID: 7, SourceHash: "source-hash", Requirements: reqs,
	}
}

func requirement(text string, kind matchcontract.RequirementKind, level matchcontract.Importance) matchcontract.ExtractedRequirement {
	return matchcontract.ExtractedRequirement{Text: text, Kind: kind, Level: level, Quote: "quote of " + text}
}

func TestVacancyExtractionValidateRefusesUntrustedRecords(t *testing.T) {
	good := extractionOf(
		requirement("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),
		requirement("French", matchcontract.RequirementLanguage, matchcontract.ImportancePreferred),
	)
	if err := good.Validate(); err != nil {
		t.Fatalf("valid extraction refused: %v", err)
	}
	empty := extractionOf()
	if err := empty.Validate(); err != nil {
		t.Fatalf("an extraction that found no requirements is a valid final answer: %v", err)
	}

	cases := map[string]func(e *matchcontract.VacancyExtraction){
		"schema":      func(e *matchcontract.VacancyExtraction) { e.SchemaVersion = "0" },
		"extractor":   func(e *matchcontract.VacancyExtraction) { e.ExtractorVersion = " " },
		"source hash": func(e *matchcontract.VacancyExtraction) { e.SourceHash = "" },
		"kind":        func(e *matchcontract.VacancyExtraction) { e.Requirements[0].Kind = "wish" },
		"level":       func(e *matchcontract.VacancyExtraction) { e.Requirements[0].Level = "nice" },
		"text":        func(e *matchcontract.VacancyExtraction) { e.Requirements[0].Text = "  " },
		"quote":       func(e *matchcontract.VacancyExtraction) { e.Requirements[0].Quote = "" },
		"duplicate": func(e *matchcontract.VacancyExtraction) {
			e.Requirements[1] = requirement("  go ", matchcontract.RequirementSkill, matchcontract.ImportancePreferred)
		},
		"condition quote": func(e *matchcontract.VacancyExtraction) {
			e.Conditions.WorkMode = &matchcontract.ExtractedCondition{Value: "remote"}
		},
	}
	for name, mutate := range cases {
		e := extractionOf(
			requirement("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),
			requirement("French", matchcontract.RequirementLanguage, matchcontract.ImportancePreferred),
		)
		mutate(&e)
		if err := e.Validate(); err != nil {
			continue
		}
		t.Errorf("%s: invalid extraction was accepted", name)
	}
}

func TestRequirementsHashPinsTheOrderedListNotTheQuotes(t *testing.T) {
	base := extractionOf(
		requirement("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),
		requirement("French", matchcontract.RequirementLanguage, matchcontract.ImportancePreferred),
	)
	hash := func(e matchcontract.VacancyExtraction) string {
		t.Helper()
		h, err := e.RequirementsHash()
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		return h
	}
	want := hash(base)

	requoted := extractionOf(
		requirement("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),
		requirement("French", matchcontract.RequirementLanguage, matchcontract.ImportancePreferred),
	)
	requoted.Requirements[0].Quote = "a different supporting sentence"
	requoted.ExtractorVersion = "test/2"
	requoted.Requirements[1].Text = "French  "
	if hash(requoted) != want {
		t.Errorf("quotes, extractor version and surrounding whitespace must not change what is matched")
	}

	swapped := extractionOf(base.Requirements[1], base.Requirements[0])
	if hash(swapped) == want {
		t.Errorf("verdicts refer to requirements by index, so the order is part of the key")
	}
	relevelled := extractionOf(base.Requirements[0], base.Requirements[1])
	relevelled.Requirements[1].Level = matchcontract.ImportanceRequired
	if hash(relevelled) == want {
		t.Errorf("a changed level must change the key")
	}
}

func matchesFor(t *testing.T, e matchcontract.VacancyExtraction, verdicts ...matchcontract.RequirementVerdict) matchcontract.RequirementMatches {
	t.Helper()
	h, err := e.RequirementsHash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return matchcontract.RequirementMatches{
		SchemaVersion: matchcontract.SchemaVersion, MatcherVersion: "test/1", JobID: e.JobID,
		RequirementsHash: h, ProfileRevision: 4, Verdicts: verdicts,
	}
}

func TestRequirementMatchesValidateAgainstTheirExtraction(t *testing.T) {
	e := extractionOf(
		requirement("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),
		requirement("Rust", matchcontract.RequirementSkill, matchcontract.ImportancePreferred),
	)
	evidence := &matchcontract.ProfileEvidence{Field: "skills", Value: "Go (expert)"}
	good := matchesFor(t, e,
		matchcontract.RequirementVerdict{Index: 0, Verdict: matchcontract.VerdictMatch, Evidence: evidence, Reason: "listed"},
		matchcontract.RequirementVerdict{Index: 1, Verdict: matchcontract.VerdictUnclear},
	)
	if err := good.ValidateAgainst(e); err != nil {
		t.Fatalf("valid matches refused: %v", err)
	}

	cases := map[string]func(m *matchcontract.RequirementMatches){
		"another list":      func(m *matchcontract.RequirementMatches) { m.RequirementsHash = "other" },
		"another job":       func(m *matchcontract.RequirementMatches) { m.JobID = 8 },
		"missing verdict":   func(m *matchcontract.RequirementMatches) { m.Verdicts = m.Verdicts[:1] },
		"duplicate index":   func(m *matchcontract.RequirementMatches) { m.Verdicts[1].Index = 0 },
		"index out of list": func(m *matchcontract.RequirementMatches) { m.Verdicts[1].Index = 5 },
		"verdict":           func(m *matchcontract.RequirementMatches) { m.Verdicts[1].Verdict = "maybe" },
		"match no evidence": func(m *matchcontract.RequirementMatches) { m.Verdicts[0].Evidence = nil },
		"partial blank evidence": func(m *matchcontract.RequirementMatches) {
			m.Verdicts[0].Verdict = matchcontract.VerdictPartial
			m.Verdicts[0].Evidence = &matchcontract.ProfileEvidence{Field: "skills"}
		},
		"matcher version":   func(m *matchcontract.RequirementMatches) { m.MatcherVersion = "" },
		"negative revision": func(m *matchcontract.RequirementMatches) { m.ProfileRevision = -1 },
	}
	for name, mutate := range cases {
		m := matchesFor(t, e,
			matchcontract.RequirementVerdict{Index: 0, Verdict: matchcontract.VerdictMatch, Evidence: evidence},
			matchcontract.RequirementVerdict{Index: 1, Verdict: matchcontract.VerdictUnclear},
		)
		mutate(&m)
		if err := m.ValidateAgainst(e); err == nil {
			t.Errorf("%s: invalid matches were accepted", name)
		}
	}
}

func TestRulesRefuseOutOfRangePolicy(t *testing.T) {
	valid := func() matchcontract.Rules {
		return matchcontract.Rules{
			SchemaVersion: matchcontract.SchemaVersion, RulesVersion: matchcontract.RulesVersion,
			Weights: map[matchcontract.Group]float64{
				matchcontract.GroupRole: 0.25, matchcontract.GroupCompensation: 0.20,
				matchcontract.GroupGeography: 0.15, matchcontract.GroupConditions: 0.10,
			},
			RequiredWeight: 2, PreferredWeight: 1,
			UnknownGoalUtility: matchcontract.DefaultUnknownGoalUtility,
			VerdictUtility:     matchcontract.DefaultVerdictUtilities(),
		}
	}
	if err := valid().Validate(); err != nil {
		t.Fatalf("valid rules refused: %v", err)
	}
	// The four goal weights are divided by their own sum, so 0.70 in total is valid.
	if sum := 0.25 + 0.20 + 0.15 + 0.10; sum > 0.71 || sum < 0.69 {
		t.Fatalf("fixture weights changed: %v", sum)
	}
	for name, mutate := range map[string]func(r *matchcontract.Rules){
		"midpoint above one": func(r *matchcontract.Rules) { r.UnknownGoalUtility = 1.2 },
		"midpoint negative":  func(r *matchcontract.Rules) { r.UnknownGoalUtility = -0.1 },
		"verdict above one":  func(r *matchcontract.Rules) { r.VerdictUtility.Match = 2 },
		"verdict negative":   func(r *matchcontract.Rules) { r.VerdictUtility.No = -1 },
		"professional weight": func(r *matchcontract.Rules) {
			r.Weights[matchcontract.GroupProfessional] = 0.30
		},
		"missing goal group": func(r *matchcontract.Rules) { delete(r.Weights, matchcontract.GroupConditions) },
	} {
		r := valid()
		mutate(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: invalid rules were accepted", name)
		}
	}
}

func TestFitResultValidateKeepsNumbersAndStatusesConsistent(t *testing.T) {
	score, raw := 70, 0.7
	ok := matchcontract.FitResult{
		Source:  matchcontract.SourceUsable,
		FitsYou: matchcontract.FitsYou{Status: matchcontract.ScoreScored, Score: &score, RawScore: &raw},
		YouFit:  matchcontract.YouFit{Status: matchcontract.ScorePending},
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid result refused: %v", err)
	}

	pendingWithNumber := ok
	pendingWithNumber.YouFit = matchcontract.YouFit{Status: matchcontract.ScorePending, Score: &score, RawScore: &raw}
	if err := pendingWithNumber.Validate(); err == nil || !strings.Contains(err.Error(), "must not carry a number") {
		t.Errorf("a pending score must not carry a number: %v", err)
	}

	scoredWithoutNumber := ok
	scoredWithoutNumber.FitsYou = matchcontract.FitsYou{Status: matchcontract.ScoreScored}
	if err := scoredWithoutNumber.Validate(); err == nil {
		t.Errorf("a scored result needs its number")
	}

	unavailableButScored := ok
	unavailableButScored.Source = matchcontract.SourceUnavailable
	if err := unavailableButScored.Validate(); err == nil {
		t.Errorf("an unavailable source must leave both scores unavailable")
	}

	wrongKind := ok
	wrongKind.YouFit = matchcontract.YouFit{Status: matchcontract.ScoreUnscored}
	if err := wrongKind.Validate(); err == nil {
		t.Errorf("unscored is a Fits you status only")
	}
}
