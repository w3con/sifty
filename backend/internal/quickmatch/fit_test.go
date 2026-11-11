package quickmatch_test

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/quickmatch"
)

// goalProfile states one goal in every group, so every group is applicable and a vacancy fact
// that is missing is a genuine unknown rather than a goal the owner does not have.
func goalProfile(revision int) matchcontract.ProfileSnapshot {
	return matchcontract.ProfileSnapshot{
		ProfileID: "local", ProfileRevision: revision,
		TargetRoles: []string{"Engineering Manager"},
		Compensation: &matchcontract.ProfileCash{
			Minimum: floatPtr(100000), Currency: "USD", Period: matchcontract.PeriodYear, TaxBasis: matchcontract.TaxGross,
		},
		Locations: []matchcontract.ProfileLocation{{Country: "DE"}},
		WorkModes: []string{matchcontract.WorkRemote},
	}
}

func titleOnlyFacts(t *testing.T, title string) matchcontract.VacancyFacts {
	t.Helper()
	return mustFacts(t, matchcontract.VacancyFacts{Role: matchcontract.RoleFacts{Title: &matchcontract.TitleFact{
		Value: title, Evidence: evidence("title", "", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
	}}})
}

func fitInput(profile matchcontract.ProfileSnapshot, facts *matchcontract.VacancyFacts) matchcontract.FitInput {
	return matchcontract.FitInput{Profile: profile, Source: matchcontract.SourceUsable, Facts: facts,
		DescriptionHash: descriptionHashOf(facts), Rules: defaultRules()}
}

func mustFit(t *testing.T, in matchcontract.FitInput) matchcontract.FitResult {
	t.Helper()
	res, err := quickmatch.EvaluateFit(in)
	if err != nil {
		t.Fatalf("EvaluateFit: %v", err)
	}
	return res
}

// descriptionHashOf stands in for matchcontract.HashDescription of the description the facts came
// from; the tests only need it to be one stable value per facts record.
func descriptionHashOf(f *matchcontract.VacancyFacts) string {
	if f == nil {
		return ""
	}
	return "description:" + f.RawSourceHash
}

func extraction(t *testing.T, facts matchcontract.VacancyFacts, reqs ...matchcontract.ExtractedRequirement) *matchcontract.VacancyExtraction {
	t.Helper()
	return &matchcontract.VacancyExtraction{
		SchemaVersion: matchcontract.SchemaVersion, ExtractorVersion: "test-extract/1",
		JobID: facts.JobID, SourceHash: descriptionHashOf(&facts), Requirements: reqs,
	}
}

func req(text string, kind matchcontract.RequirementKind, level matchcontract.Importance) matchcontract.ExtractedRequirement {
	return matchcontract.ExtractedRequirement{Text: text, Kind: kind, Level: level, Quote: "quote: " + text}
}

func matchesOf(t *testing.T, e *matchcontract.VacancyExtraction, revision int, verdicts ...matchcontract.RequirementVerdict) *matchcontract.RequirementMatches {
	t.Helper()
	h, err := e.RequirementsHash()
	if err != nil {
		t.Fatalf("requirements hash: %v", err)
	}
	return &matchcontract.RequirementMatches{
		SchemaVersion: matchcontract.SchemaVersion, MatcherVersion: "test-match/1", JobID: e.JobID,
		RequirementsHash: h, ProfileRevision: revision, Verdicts: verdicts,
	}
}

func evidenceOf(field, value string) *matchcontract.ProfileEvidence {
	return &matchcontract.ProfileEvidence{Field: field, Value: value}
}

func criteria(lines []matchcontract.FitLine) []matchcontract.Criterion {
	out := []matchcontract.Criterion{}
	for _, l := range lines {
		if l.Origin == matchcontract.OriginGoal {
			out = append(out, l.Criterion)
		}
	}
	return out
}

func subjects(lines []matchcontract.FitLine) []string {
	out := []string{}
	for _, l := range lines {
		if l.Origin == matchcontract.OriginRequirement {
			out = append(out, l.Subject)
		}
	}
	return out
}

func TestFitsYouTitleOnlyMatchIsNotAHundred(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")

	// The goal score on its own: the title is the only known group.
	goals, err := quickmatch.Evaluate(goalProfile(1), facts, defaultRules())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	// (25*1 + 20*0.5 + 15*0.5 + 10*0.5) / 70 = 47.5/70
	if goals.Score == nil || *goals.Score != 68 {
		t.Fatalf("title-only goal score = %v, want 68, never 100", goals.Score)
	}
	if goals.RawScore == nil || math.Abs(*goals.RawScore-47.5/70) > 1e-12 {
		t.Fatalf("raw goal score = %v, want 47.5/70", goals.RawScore)
	}

	res := mustFit(t, fitInput(goalProfile(1), &facts))
	if res.FitsYou.Status != matchcontract.ScoreScored || res.FitsYou.Score == nil || *res.FitsYou.Score != 68 {
		t.Fatalf("Fits you = %+v, want scored 68", res.FitsYou)
	}
	known := map[matchcontract.Group]bool{}
	for _, g := range res.FitsYou.Groups {
		known[g.Group] = g.Known
	}
	want := map[matchcontract.Group]bool{
		matchcontract.GroupRole: true, matchcontract.GroupCompensation: false,
		matchcontract.GroupGeography: false, matchcontract.GroupConditions: false,
	}
	if !reflect.DeepEqual(known, want) {
		t.Fatalf("known groups = %v, want only role known: %v", known, want)
	}
	for _, g := range res.FitsYou.Groups {
		if !g.Known && g.Utility != matchcontract.DefaultUnknownGoalUtility {
			t.Fatalf("an unknown group must stand in as the midpoint: %+v", g)
		}
	}
}

func TestFitsYouAllUnknownIsFifty(t *testing.T) {
	facts := mustFacts(t, matchcontract.VacancyFacts{}) // no title, no pay, no location, no work mode
	res := mustFit(t, fitInput(goalProfile(1), &facts))
	if res.FitsYou.Status != matchcontract.ScoreScored || res.FitsYou.Score == nil || *res.FitsYou.Score != 50 {
		t.Fatalf("all-unknown Fits you = %+v, want scored 50", res.FitsYou)
	}
	if res.FitsYou.RawScore == nil || math.Abs(*res.FitsYou.RawScore-0.5) > 1e-12 {
		t.Fatalf("raw = %v, want 0.5", res.FitsYou.RawScore)
	}
	for _, g := range res.FitsYou.Groups {
		if g.Known {
			t.Fatalf("no group has a comparable fact: %+v", g)
		}
	}
}

func TestFitsYouWithoutAnyGoalIsUnscored(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	res := mustFit(t, fitInput(matchcontract.ProfileSnapshot{ProfileID: "local"}, &facts))
	if res.FitsYou.Status != matchcontract.ScoreUnscored || res.FitsYou.Score != nil {
		t.Fatalf("a profile with no goal cannot yield Fits you: %+v", res.FitsYou)
	}
}

func TestYouFitFromVerdictsAndAuthorizationMovesHere(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	profile := matchcontract.ProfileSnapshot{ProfileID: "local", ProfileRevision: 4}
	e := extraction(t, facts,
		req("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),                               // 0 match
		req("Right to work in Germany", matchcontract.RequirementAuthorization, matchcontract.ImportanceRequired), // 1 no
		req("5 years of backend", matchcontract.RequirementExperience, matchcontract.ImportanceRequired),          // 2 partial
		req("Terraform", matchcontract.RequirementSkill, matchcontract.ImportancePreferred),                       // 3 unclear
		req("Rust", matchcontract.RequirementSkill, matchcontract.ImportancePreferred),                            // 4 match
	)
	m := matchesOf(t, e, 4,
		matchcontract.RequirementVerdict{Index: 0, Verdict: matchcontract.VerdictMatch, Evidence: evidenceOf("skills", "Go (expert)"), Reason: "listed as expert"},
		matchcontract.RequirementVerdict{Index: 1, Verdict: matchcontract.VerdictNo, Evidence: evidenceOf("authorizations", "DE=need_visa"), Reason: "needs a visa"},
		matchcontract.RequirementVerdict{Index: 2, Verdict: matchcontract.VerdictPartial, Evidence: evidenceOf("experience", "3 years backend"), Reason: "3 of 5 years"},
		matchcontract.RequirementVerdict{Index: 3, Verdict: matchcontract.VerdictUnclear},
		matchcontract.RequirementVerdict{Index: 4, Verdict: matchcontract.VerdictMatch, Evidence: evidenceOf("skills", "Rust"), Reason: "listed"},
	)
	in := fitInput(profile, &facts)
	in.Extraction, in.Matches = e, m
	res := mustFit(t, in)

	// required weight 2, preferred 1: (2*1 + 2*0 + 2*0.5 + 1*0.5 + 1*1) / (2+2+2+1+1) = 4.5/8
	if res.YouFit.Status != matchcontract.ScoreScored || res.YouFit.Score == nil || *res.YouFit.Score != 56 {
		t.Fatalf("You fit = %+v, want scored 56", res.YouFit)
	}
	if res.YouFit.RawScore == nil || math.Abs(*res.YouFit.RawScore-4.5/8) > 1e-12 {
		t.Fatalf("raw You fit = %v, want 4.5/8", res.YouFit.RawScore)
	}
	if want := (matchcontract.FitCounts{Match: 2, Partial: 1, No: 1, Unclear: 1}); res.YouFit.Counts != want {
		t.Fatalf("counts = %+v, want %+v", res.YouFit.Counts, want)
	}

	// Required before preferred; match and partial are matches (a partial line keeps its verdict
	// so the view can mark it); no is a mismatch; unclear is missing from the profile.
	if got, want := subjects(res.Lists.Matches), []string{"Go", "5 years of backend", "Rust"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("matches = %v, want %v", got, want)
	}
	if partial := res.Lists.Matches[1]; partial.Verdict != matchcontract.VerdictPartial || partial.ProfileValue != "3 years backend" {
		t.Fatalf("a partial match keeps its verdict and evidence: %+v", partial)
	}
	if got, want := subjects(res.Lists.Mismatches), []string{"Right to work in Germany"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatches = %v, want %v", got, want)
	}
	if got, want := subjects(res.Lists.Missing), []string{"Terraform"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing = %v, want %v", got, want)
	}
	auth := res.Lists.Mismatches[0]
	if auth.Kind != matchcontract.RequirementAuthorization || auth.Verdict != matchcontract.VerdictNo ||
		auth.ProfileField != "authorizations" || auth.ProfileValue != "DE=need_visa" || auth.Quote == "" {
		t.Fatalf("work authorization is a requirement line carrying its verdict and evidence: %+v", auth)
	}

	// Work authorization is no longer a goal: whatever the vacancy states, no goal comparison for it.
	for _, list := range [][]matchcontract.FitLine{res.Lists.Matches, res.Lists.Mismatches, res.Lists.ToClarify} {
		for _, l := range list {
			if l.Criterion == matchcontract.CriterionGeographyAuthorization {
				t.Fatalf("authorization appeared as a goal line: %+v", l)
			}
		}
	}
	if res.Provenance.RequirementsHash == "" || res.Provenance.ExtractorVersion != "test-extract/1" || res.Provenance.MatcherVersion != "test-match/1" {
		t.Fatalf("provenance must name the extraction and matches used: %+v", res.Provenance)
	}
}

// A demand that lists alternatives ("Kotlin, Python or Go") keeps its members on the line, so the
// view can show them; a single demand carries none and the field is absent from the JSON.
func TestRequirementLineCarriesAlternatives(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	group := req("Backend language", matchcontract.RequirementSkill, matchcontract.ImportanceRequired)
	group.Alternatives = []string{"Kotlin", "Python", "Go"}
	e := extraction(t, facts, group, req("Terraform", matchcontract.RequirementSkill, matchcontract.ImportanceRequired))
	m := matchesOf(t, e, 4,
		matchcontract.RequirementVerdict{Index: 0, Verdict: matchcontract.VerdictMatch, Evidence: evidenceOf("skills", "Go (expert)")},
		matchcontract.RequirementVerdict{Index: 1, Verdict: matchcontract.VerdictUnclear},
	)
	in := fitInput(matchcontract.ProfileSnapshot{ProfileID: "local", ProfileRevision: 4}, &facts)
	in.Extraction, in.Matches = e, m
	res := mustFit(t, in)

	if got := res.Lists.Matches[0].Alternatives; !reflect.DeepEqual(got, []string{"Kotlin", "Python", "Go"}) {
		t.Fatalf("alternatives on the group line = %v", got)
	}
	// The line owns its copy: changing it must not change the extraction it came from.
	res.Lists.Matches[0].Alternatives[0] = "changed"
	if e.Requirements[0].Alternatives[0] != "Kotlin" {
		t.Fatalf("the line shares its alternatives slice with the extraction")
	}
	single, err := json.Marshal(res.Lists.Missing[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(single), "alternatives") {
		t.Fatalf("a single demand must omit alternatives: %s", single)
	}
}

func TestYouFitNoRequirementsStatedIsNeitherZeroNorHundred(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	in := fitInput(goalProfile(1), &facts)
	in.Extraction = extraction(t, facts) // analysed, found nothing: final for this source
	res := mustFit(t, in)
	if res.YouFit.Status != matchcontract.ScoreNoRequirements || res.YouFit.Score != nil || res.YouFit.RawScore != nil {
		t.Fatalf("You fit = %+v, want no_requirements without a number", res.YouFit)
	}
	if res.YouFit.Counts != (matchcontract.FitCounts{}) {
		t.Fatalf("counts must be empty: %+v", res.YouFit.Counts)
	}
	if res.FitsYou.Status != matchcontract.ScoreScored {
		t.Fatalf("Fits you is independent of the requirements: %+v", res.FitsYou)
	}
}

func TestYouFitPendingUntilExtractedAndMatched(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	profile := goalProfile(2)

	notExtracted := mustFit(t, fitInput(profile, &facts))
	if notExtracted.YouFit.Status != matchcontract.ScorePending || notExtracted.YouFit.Score != nil {
		t.Fatalf("not extracted = %+v, want pending without a number", notExtracted.YouFit)
	}

	in := fitInput(profile, &facts)
	in.Extraction = extraction(t, facts, req("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired))
	notMatched := mustFit(t, in)
	if notMatched.YouFit.Status != matchcontract.ScorePending || notMatched.YouFit.Score != nil {
		t.Fatalf("extracted but not matched = %+v, want pending without a number", notMatched.YouFit)
	}
	if notMatched.FitsYou.Status != matchcontract.ScoreScored || notMatched.FitsYou.Score == nil {
		t.Fatalf("Fits you must not wait for the model passes: %+v", notMatched.FitsYou)
	}
}

func TestSourceUnavailableHasNoNumbersAndNoLists(t *testing.T) {
	// Even when records exist, an unavailable source shows nothing: the text they came from is gone.
	facts := titleOnlyFacts(t, "Engineering Manager")
	in := fitInput(goalProfile(1), &facts)
	in.Source = matchcontract.SourceUnavailable
	in.Extraction = extraction(t, facts, req("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired))
	res := mustFit(t, in)
	if res.FitsYou.Status != matchcontract.ScoreUnavailable || res.FitsYou.Score != nil || res.FitsYou.RawScore != nil {
		t.Fatalf("Fits you = %+v, want unavailable without a number", res.FitsYou)
	}
	if res.YouFit.Status != matchcontract.ScoreUnavailable || res.YouFit.Score != nil || res.YouFit.RawScore != nil {
		t.Fatalf("You fit = %+v, want unavailable without a number", res.YouFit)
	}
	if len(res.Lists.Matches)+len(res.Lists.Mismatches)+len(res.Lists.Missing)+len(res.Lists.ToClarify) != 0 {
		t.Fatalf("lists must be empty: %+v", res.Lists)
	}

	// No facts are needed, or read, when the source is unavailable.
	noFacts := fitInput(goalProfile(1), nil)
	noFacts.Source = matchcontract.SourceUnavailable
	if res := mustFit(t, noFacts); res.FitsYou.Status != matchcontract.ScoreUnavailable {
		t.Fatalf("unavailable source without facts = %+v", res.FitsYou)
	}
	// But a usable source without facts is a caller defect, not a score.
	if _, err := quickmatch.EvaluateFit(fitInput(goalProfile(1), nil)); err == nil {
		t.Fatalf("a usable source without facts must be refused")
	}
}

func TestGoalLinesFollowReasonCodesAndOnlyClarifyWhatTheOwnerCaresAbout(t *testing.T) {
	profile := goalProfile(1)
	profile.EngagementTypes = []string{matchcontract.EngagementEmployment}
	facts := mustFacts(t, matchcontract.VacancyFacts{
		Role: matchcontract.RoleFacts{Title: &matchcontract.TitleFact{
			Value: "Engineering Manager", Evidence: evidence("title", "", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
		}},
		Compensation: matchcontract.CompensationFacts{Cash: &matchcontract.CashFact{
			Max: floatPtr(70000), Currency: "USD", Period: matchcontract.PeriodYear, TaxBasis: matchcontract.TaxGross,
			Basis:    matchcontract.CashBasisStated,
			Evidence: evidence("", "Pay: 70,000 USD", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
		}},
		Geography: matchcontract.GeographyFacts{HiringLocations: []matchcontract.LocationFact{{
			Label: "Germany", Country: "DE", Evidence: evidence("", "Germany", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
		}}},
		Conditions: matchcontract.ConditionsFacts{WorkModes: []matchcontract.WorkModeFact{{
			Mode: matchcontract.WorkOffice, Evidence: evidence("", "Office in Berlin", matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
		}}},
	})
	res := mustFit(t, fitInput(profile, &facts))

	if got, want := criteria(res.Lists.Matches), []matchcontract.Criterion{matchcontract.CriterionRoleTitle, matchcontract.CriterionGeographyLocation}; !reflect.DeepEqual(got, want) {
		t.Fatalf("matches = %v, want %v", got, want)
	}
	if got, want := criteria(res.Lists.Mismatches), []matchcontract.Criterion{matchcontract.CriterionCompensationCash, matchcontract.CriterionConditionsWorkMode}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatches = %v, want %v", got, want)
	}
	if got, want := criteria(res.Lists.ToClarify), []matchcontract.Criterion{matchcontract.CriterionConditionsEngagement}; !reflect.DeepEqual(got, want) {
		t.Fatalf("to clarify = %v, want only the unknown engagement the profile asks for: %v", got, want)
	}

	pay := res.Lists.Mismatches[0]
	if pay.Origin != matchcontract.OriginGoal || pay.Group != matchcontract.GroupCompensation ||
		pay.ProfileValue == "" || pay.JobValue == "" || pay.Reason != matchcontract.ReasonCompensationBelowMin ||
		pay.Utility == nil || *pay.Utility >= 1 {
		t.Fatalf("pay below the minimum must carry both values, the reason code and its utility: %+v", pay)
	}
}

func TestStaleRecordsAreRefusedNotReused(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	e := extraction(t, facts, req("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired))
	goMatch := matchcontract.RequirementVerdict{Index: 0, Verdict: matchcontract.VerdictMatch, Evidence: evidenceOf("skills", "Go")}
	build := func(profileRevision int, e *matchcontract.VacancyExtraction, m *matchcontract.RequirementMatches) matchcontract.FitInput {
		in := fitInput(goalProfile(profileRevision), &facts)
		in.Extraction, in.Matches = e, m
		return in
	}

	if _, err := quickmatch.EvaluateFit(build(3, e, matchesOf(t, e, 3, goMatch))); err != nil {
		t.Fatalf("current records refused: %v", err)
	}
	if _, err := quickmatch.EvaluateFit(build(4, e, matchesOf(t, e, 3, goMatch))); err == nil {
		t.Errorf("matches for profile revision 3 must not be applied to revision 4")
	}

	other := extraction(t, facts, req("Rust", matchcontract.RequirementSkill, matchcontract.ImportanceRequired))
	if _, err := quickmatch.EvaluateFit(build(3, other, matchesOf(t, e, 3, goMatch))); err == nil {
		t.Errorf("matches for another requirement list must be refused")
	}

	foreign := *e
	foreign.SourceHash = "a-different-source"
	if _, err := quickmatch.EvaluateFit(build(3, &foreign, nil)); err == nil {
		t.Errorf("an extraction of another source text must be refused")
	}
}

func TestNewProfileRevisionReusesTheSameExtraction(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	e := extraction(t, facts,
		req("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),
		req("Rust", matchcontract.RequirementSkill, matchcontract.ImportanceRequired),
	)
	run := func(revision int, rustVerdict matchcontract.RequirementVerdict) matchcontract.FitResult {
		in := fitInput(goalProfile(revision), &facts)
		in.Extraction = e
		in.Matches = matchesOf(t, e, revision,
			matchcontract.RequirementVerdict{Index: 0, Verdict: matchcontract.VerdictMatch, Evidence: evidenceOf("skills", "Go")},
			rustVerdict,
		)
		return mustFit(t, in)
	}
	before := run(1, matchcontract.RequirementVerdict{Index: 1, Verdict: matchcontract.VerdictUnclear})
	after := run(2, matchcontract.RequirementVerdict{Index: 1, Verdict: matchcontract.VerdictMatch, Evidence: evidenceOf("skills", "Rust")})
	if *before.YouFit.Score != 75 || *after.YouFit.Score != 100 {
		t.Fatalf("You fit before/after adding Rust = %d/%d, want 75/100", *before.YouFit.Score, *after.YouFit.Score)
	}
	if before.Provenance.RequirementsHash != after.Provenance.RequirementsHash {
		t.Fatalf("the profile change must reuse the same extraction")
	}
	if *before.FitsYou.Score != *after.FitsYou.Score {
		t.Fatalf("a skill change must not move Fits you")
	}
}

func TestPolicyComesFromRulesAndEvaluationIsPureAndSerializable(t *testing.T) {
	facts := titleOnlyFacts(t, "Engineering Manager")
	e := extraction(t, facts, req("Go", matchcontract.RequirementSkill, matchcontract.ImportanceRequired))
	in := fitInput(goalProfile(1), &facts)
	in.Extraction = e
	in.Matches = matchesOf(t, e, 1, matchcontract.RequirementVerdict{Index: 0, Verdict: matchcontract.VerdictPartial, Evidence: evidenceOf("skills", "Go")})

	first, second := mustFit(t, in), mustFit(t, in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same input produced different results")
	}
	if *first.YouFit.Score != 50 || *first.FitsYou.Score != 68 {
		t.Fatalf("defaults: You fit %d Fits you %d, want 50 and 68", *first.YouFit.Score, *first.FitsYou.Score)
	}

	// Changing the injected policy changes the numbers without any other input changing.
	tuned := in
	tuned.Rules.UnknownGoalUtility = 0
	tuned.Rules.VerdictUtility.Partial = 0.25
	res := mustFit(t, tuned)
	// (25*1 + 0 + 0 + 0) / 70 = 35.7 -> 36; partial 0.25 -> 25
	if *res.FitsYou.Score != 36 || *res.YouFit.Score != 25 {
		t.Fatalf("tuned policy: Fits you %d You fit %d, want 36 and 25", *res.FitsYou.Score, *res.YouFit.Score)
	}
	if res.Provenance.RulesIdentity == first.Provenance.RulesIdentity {
		t.Fatalf("a policy change must change the rules identity so stored results are re-evaluated")
	}

	// Lists and groups are always arrays in the persisted and API JSON, so a view can read their
	// length without a null check. A number without a value is an explicit null by design.
	raw, err := json.Marshal(mustFit(t, fitInput(matchcontract.ProfileSnapshot{ProfileID: "local"}, &facts)))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, field := range []string{`"matches":[]`, `"mismatches":[]`, `"missing":[]`, `"toClarify":[]`, `"groups":[]`} {
		if !strings.Contains(string(raw), field) {
			t.Fatalf("result JSON lacks %s: %s", field, raw)
		}
	}
}
