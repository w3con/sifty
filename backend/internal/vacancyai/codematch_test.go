package vacancyai_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/vacancyai"
)

func defaultVocabulary(t *testing.T) *vacancyai.Vocabulary {
	t.Helper()
	skills, err := config.DefaultSkills()
	if err != nil {
		t.Fatalf("built-in skills: %v", err)
	}
	return vacancyai.NewVocabulary(skills)
}

func skillMatcher(t *testing.T, infer vacancyai.InferFunc) vacancyai.Matcher {
	m := matcher(infer)
	m.Vocabulary = defaultVocabulary(t)
	return m
}

func oneRequirement(text string, kind matchcontract.RequirementKind, alternatives ...string) matchcontract.VacancyExtraction {
	return matchcontract.VacancyExtraction{
		SchemaVersion: matchcontract.SchemaVersion, ExtractorVersion: "x", JobID: 1, SourceHash: "h",
		Requirements: []matchcontract.ExtractedRequirement{{
			Text: text, Kind: kind, Level: matchcontract.ImportanceRequired, Quote: "q", Alternatives: alternatives,
		}},
	}
}

var cvFacts = []matchcontract.ProfileFact{
	{Field: "skills", Value: "Go (expert)"},
	{Field: "skills", Value: "R (advanced)"},
	{Field: "skills", Value: "Rust"},
	{Field: "skills", Value: "Python (advanced)"},
	{Field: "experience", Value: "Founder & CTO at Acme, 2019 to present: built the platform on Postgres and led the engineers."},
}

const askedNothing = `{"verdicts":[{"index":0,"verdict":"unclear","evidenceId":"","reason":""}]}`

// A skill named in the profile must never be decided by code because of a word that only looks
// like a technology: "go to market", "Go-to-market", "R&D" and "C-level" are ordinary words. A false
// match here is silent (the owner sees a match with evidence), so it is the one thing code matching
// must not get wrong; each of these must reach the model.
func TestCodeMatchDoesNotMistakeOrdinaryWordsForTechnologies(t *testing.T) {
	for _, text := range []string{
		"Go to market strategy", "Go-to-market experience", "go to market planning",
		"R&D management", "Experience with C-level stakeholders", "Understanding of the R in ROI",
		// Nothing but the rule for ambiguous names stops this one: "or" makes it a list of
		// alternatives, so no unknown word counts against it.
		"Go or no go decisions",
	} {
		var calls int32
		e := oneRequirement(text, matchcontract.RequirementExperience)
		res, err := skillMatcher(t, stub(&calls, askedNothing)).Match(context.Background(), e, cvFacts, 1)
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		if res.CodeMatched != 0 || calls != 1 {
			t.Errorf("%q was decided by code (code matched %d, model calls %d); the profile lists Go, R and C-like names but this is not a technology", text, res.CodeMatched, calls)
		}
		if r := res.Record.Verdicts[0].Reason; r == vacancyai.ReasonNamedSkill || r == vacancyai.ReasonSomeNamedSkills {
			t.Errorf("%q: %+v", text, res.Record.Verdicts[0])
		}
	}
}

func TestCodeMatchDecidesNamedSkillsWithoutTheModel(t *testing.T) {
	cases := []struct {
		name         string
		text         string
		alternatives []string
		code         bool
		verdict      matchcontract.Verdict
	}{
		{"a named skill", "Rust", nil, true, matchcontract.VerdictMatch},
		{"a synonym of a profile skill", "Golang proficiency", nil, true, matchcontract.VerdictMatch},
		{"an ambiguous name that is the whole requirement", "Go", nil, true, matchcontract.VerdictMatch},
		{"an ambiguous name next to a technology", "Python/Go/Rust proficiency", nil, true, matchcontract.VerdictMatch},
		{"a group where one member is in the profile", "Backend language", []string{"Kotlin", "Java", "Go (Golang)"}, true, matchcontract.VerdictMatch},
		{"a short name inside a group", "Statistics tooling", []string{"R", "Stata"}, true, matchcontract.VerdictMatch},
		{"a technology in the profile's prose", "Postgres", nil, true, matchcontract.VerdictMatch},
		{"several named, some in the profile: a partial decided by code", "Python and Kubernetes", nil, true, matchcontract.VerdictPartial},
		{"an unknown word is not verified by code", "Python and observability", nil, false, ""},
		{"years of one technology are not in a skills list", "5+ years Rust", nil, false, ""},
		{"a group with no member in the profile", "Backend language", []string{"Kotlin", "Java"}, false, ""},
		{"a requirement that is not about a technology", "Leading engineering teams", nil, false, ""},
	}
	for _, c := range cases {
		var calls int32
		e := oneRequirement(c.text, matchcontract.RequirementSkill, c.alternatives...)
		res, err := skillMatcher(t, stub(&calls, askedNothing)).Match(context.Background(), e, cvFacts, 1)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.code {
			v := res.Record.Verdicts[0]
			if res.CodeMatched != 1 || calls != 0 || v.Verdict != c.verdict || v.Evidence == nil || (v.Reason != vacancyai.ReasonNamedSkill && v.Reason != vacancyai.ReasonSomeNamedSkills) {
				t.Errorf("%s: want a match decided by code with evidence and no model call: %+v (code %d, calls %d)", c.name, v, res.CodeMatched, calls)
			}
		} else if res.CodeMatched != 0 || calls != 1 {
			t.Errorf("%s: want the model to decide: code %d, calls %d", c.name, res.CodeMatched, calls)
		}
	}
}

func TestCodeMatchedRequirementsAreNotSentToTheModel(t *testing.T) {
	e := matchcontract.VacancyExtraction{
		SchemaVersion: matchcontract.SchemaVersion, ExtractorVersion: "x", JobID: 1, SourceHash: "h",
		Requirements: []matchcontract.ExtractedRequirement{
			{Text: "Rust", Kind: matchcontract.RequirementSkill, Level: matchcontract.ImportanceRequired, Quote: "q1"},
			{Text: "Kubernetes", Kind: matchcontract.RequirementSkill, Level: matchcontract.ImportanceRequired, Quote: "q2"},
		},
	}
	var request string
	infer := func(_ context.Context, _ assess.Provider, req string, _ any, out any) error {
		request = req
		return json.Unmarshal([]byte(`{"verdicts":[{"index":1,"verdict":"no","evidenceId":"","reason":"No Kubernetes in the profile"}]}`), out)
	}
	res, err := skillMatcher(t, infer).Match(context.Background(), e, cvFacts, 1)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if strings.Contains(request, "0. [required] [skill] Rust") || !strings.Contains(request, "1. [required] [skill] Kubernetes") {
		t.Errorf("only the requirement code could not decide goes to the model:\n%s", request)
	}
	v := res.Record.Verdicts
	if v[0].Verdict != matchcontract.VerdictMatch || v[1].Verdict != matchcontract.VerdictNo || res.CodeMatched != 1 || res.Asked != 1 {
		t.Errorf("verdicts %+v, code %d, asked %d", v, res.CodeMatched, res.Asked)
	}
}

// "No" on what a CV does not state would make every soft requirement a mismatch. Such a "no" is
// unclear; "no" on a specific skill stands.
func TestNoOnWhatACVDoesNotStateIsUnclear(t *testing.T) {
	e := matchcontract.VacancyExtraction{
		SchemaVersion: matchcontract.SchemaVersion, ExtractorVersion: "x", JobID: 1, SourceHash: "h",
		Requirements: []matchcontract.ExtractedRequirement{
			{Text: "Excellent communication skills", Kind: matchcontract.RequirementSkill, Level: matchcontract.ImportanceRequired, Quote: "q1"},
			{Text: "Led a team of 5-8 engineers", Kind: matchcontract.RequirementExperience, Level: matchcontract.ImportanceRequired, Quote: "q2"},
			{Text: "Kubernetes", Kind: matchcontract.RequirementSkill, Level: matchcontract.ImportanceRequired, Quote: "q3"},
			{Text: "Degree in Computer Science or related", Kind: matchcontract.RequirementQualification, Level: matchcontract.ImportancePreferred, Quote: "q4"},
		},
	}
	answer := `{"verdicts":[
 {"index":0,"verdict":"no","evidenceId":"","reason":"not mentioned"},
 {"index":1,"verdict":"no","evidenceId":"","reason":"no team size"},
 {"index":2,"verdict":"no","evidenceId":"","reason":"No Kubernetes in the profile"},
 {"index":3,"verdict":"no","evidenceId":"","reason":"No degree listed"}]}`
	var calls int32
	res, err := skillMatcher(t, stub(&calls, answer)).Match(context.Background(), e, cvFacts, 1)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	v := res.Record.Verdicts
	if v[0].Verdict != matchcontract.VerdictUnclear || v[1].Verdict != matchcontract.VerdictUnclear {
		t.Errorf("soft skills and team size are not stated in a CV: %+v %+v", v[0], v[1])
	}
	if v[2].Verdict != matchcontract.VerdictNo {
		t.Errorf("a specific technology the CV does not show is a no: %+v", v[2])
	}
	if v[3].Verdict != matchcontract.VerdictUnclear {
		t.Errorf("the profile has no education section, so a degree cannot be a no: %+v", v[3])
	}
}

func TestOnlyTheConfiguredNumberOfClosestFactsIsShown(t *testing.T) {
	facts := []matchcontract.ProfileFact{
		{Field: "experience", Value: "Engineer at A: designed payment systems"},
		{Field: "experience", Value: "Engineer at B: designed payment gateways"},
		{Field: "experience", Value: "Engineer at C: designed payment ledgers"},
		{Field: "experience", Value: "Engineer at D: designed payment risk models"},
		{Field: "skills", Value: "Cooking"},
	}
	e := oneRequirement("Designed payment systems", matchcontract.RequirementExperience)
	for _, k := range []int{2, 3} {
		var request string
		infer := func(_ context.Context, _ assess.Provider, req string, _ any, out any) error {
			request = req
			return json.Unmarshal([]byte(askedNothing), out)
		}
		m := skillMatcher(t, infer)
		m.EvidencePassages = k
		if _, err := m.Match(context.Background(), e, facts, 1); err != nil {
			t.Fatalf("Match: %v", err)
		}
		shown := strings.Count(request, "| experience |")
		if shown != k || strings.Contains(request, "Cooking") {
			t.Errorf("k=%d: %d experience facts shown, want %d and nothing unrelated:\n%s", k, shown, k, request)
		}
	}
}

func TestMatcherVersionFollowsTheVocabularyAndPassageCount(t *testing.T) {
	base := &config.Config{}
	skills, err := config.DefaultSkills()
	if err != nil {
		t.Fatal(err)
	}
	withSkills := &config.Config{Skills: skills}
	extra := skills
	extra.Groups = append(append([]config.SkillGroup(nil), skills.Groups...), config.SkillGroup{Name: "Zig"})
	changed := &config.Config{Skills: extra}
	more := &config.Config{Skills: skills}
	more.Runtime.QuickMatch = config.DefaultQuickMatch()
	more.Runtime.QuickMatch.ModelPasses.EvidencePassages = 5

	// A different model is a different matcher, and stages.matching picks it without touching extraction.
	small := &config.Config{Skills: skills}
	small.Providers.Providers = map[string]config.Provider{"small": {Model: "m-small"}, "large": {Model: "m-large"}}
	small.Providers.Stages.Extraction = "small"
	large := &config.Config{Skills: skills, Providers: small.Providers}
	large.Providers.Stages.Extraction = "small"
	large.Providers.Stages.Matching = "large"
	if got := large.Providers.MatchingStage(); got != "large" {
		t.Errorf("matching stage = %q, want large", got)
	}
	if got := small.Providers.MatchingStage(); got != "small" {
		t.Errorf("matching stage without its own provider = %q, want the extraction provider small", got)
	}

	seen := map[string]string{}
	for name, cfg := range map[string]*config.Config{"none": base, "default": withSkills, "changed": changed, "more": more, "small": small, "large": large} {
		v := vacancyai.MatcherVersion(cfg)
		if prev, dup := seen[v]; dup {
			t.Errorf("%s and %s share the matcher version %s, so a change would not re-run the matching", name, prev, v)
		}
		seen[v] = name
	}
	// The version a built matcher stamps on its records is the one the worker plans with.
	m := skillMatcher(t, stub(new(int32), askedNothing))
	res, err := m.Match(context.Background(), oneRequirement("Rust", matchcontract.RequirementSkill), cvFacts, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Record.MatcherVersion != vacancyai.MatcherVersion(testModelConfig(withSkills)) {
		t.Errorf("records carry %s, the worker plans with %s", res.Record.MatcherVersion, vacancyai.MatcherVersion(testModelConfig(withSkills)))
	}
}

// A requirement for "cloud platform experience" must be shown the passage that says AWS, which
// shares no word with it; otherwise the model sees an empty profile and says "no".
func TestCategoryWordsBringTheNamesOfThatKindOfTechnology(t *testing.T) {
	facts := []matchcontract.ProfileFact{
		{Field: "experience", Value: "Backend Go/AWS, frontend React at Immutable"},
		{Field: "experience", Value: "Cooked dinner for a large family"},
	}
	e := oneRequirement("Cloud platform experience", matchcontract.RequirementSkill)
	var request string
	infer := func(_ context.Context, _ assess.Provider, req string, _ any, out any) error {
		request = req
		return json.Unmarshal([]byte(askedNothing), out)
	}
	if _, err := skillMatcher(t, infer).Match(context.Background(), e, facts, 1); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request, "| facts: P1") || strings.Contains(request, "Cooked dinner") {
		t.Errorf("the AWS passage must be offered for a cloud requirement and the unrelated one not:\n%s", request)
	}
}

// A skill the owner has said they do not have is a confirmed "no": decided by code, without a
// model, and not guessed from a different wording of another skill.
func TestSkillTheOwnerSaidTheyLackIsANoWithoutTheModel(t *testing.T) {
	facts := append([]matchcontract.ProfileFact(nil), cvFacts...)
	facts = append(facts,
		matchcontract.ProfileFact{Field: "skillsAbsent", Value: "Does not have: Kubernetes"},
		matchcontract.ProfileFact{Field: "skillsAbsent", Value: "Does not have: ML sphere"},
		matchcontract.ProfileFact{Field: "skillsAbsent", Value: "Does not have: IaC"},
	)
	cases := []struct {
		name         string
		text         string
		alternatives []string
		no           bool
	}{
		{"a technology of the vocabulary", "Kubernetes expertise", nil, true},
		{"the same skill worded another way", "Strong Kubernetes experience", nil, true},
		{"a skill the vocabulary does not know, by the name the owner saw", "ML sphere", nil, true},
		{"a group in which every member is absent", "Container orchestration", []string{"Kubernetes", "K8s"}, true},
		{"a group in which one member is not marked", "Orchestration", []string{"Kubernetes", "Nomad"}, false},
		{"one skill of a line that names several is not the line", "Kubernetes, IaC and observability", nil, false},
	}
	for _, c := range cases {
		var calls int32
		e := oneRequirement(c.text, matchcontract.RequirementSkill, c.alternatives...)
		res, err := skillMatcher(t, stub(&calls, askedNothing)).Match(context.Background(), e, facts, 1)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		v := res.Record.Verdicts[0]
		if c.no {
			if v.Verdict != matchcontract.VerdictNo || calls != 0 || v.Reason != vacancyai.ReasonMarkedAbsent || v.Evidence == nil || v.Evidence.Field != "skillsAbsent" {
				t.Errorf("%s: want a no decided by code from the owner's own list: %+v (calls %d)", c.name, v, calls)
			}
		} else if calls != 1 || v.Reason == vacancyai.ReasonMarkedAbsent {
			t.Errorf("%s: the model must decide: %+v (calls %d)", c.name, v, calls)
		}
	}
}
