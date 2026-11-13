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

// A skill the owner added by name (non-vocabulary or vocabulary) is recognised cross-vacancy
// by haveByName, without the model.
func TestHaveByNameMatchesSkillAddedByName(t *testing.T) {
	facts := append([]matchcontract.ProfileFact(nil), cvFacts...)
	// Add a non-vocabulary skill "AI coding tools" with a level suffix (as MatchFacts produces it)
	// and a vocabulary skill "React" without a level, to verify that the level suffix is stripped
	// before the name lookup. "Open source contributions (Proficient)" must match the requirement
	// "Open source contributions", not miss it because of the "(Proficient)" tail.
	facts = append(facts,
		matchcontract.ProfileFact{Field: "skills", Value: "AI coding tools"},
		matchcontract.ProfileFact{Field: "skills", Value: "Open source contributions (Proficient)"},
		matchcontract.ProfileFact{Field: "skills", Value: "React"},
	)
	cases := []struct {
		name    string
		text    string
		verdict matchcontract.Verdict
		reason  string
	}{
		{"non-vocab skill matches by name", "AI coding tools", matchcontract.VerdictMatch, vacancyai.ReasonSkillInProfile},
		{"same skill with prefix normalised away", "Strong AI coding tools experience", matchcontract.VerdictMatch, vacancyai.ReasonSkillInProfile},
		// A skill stored with a level suffix "(Proficient)" must still match the requirement
		// that names only the skill (no level). This is the runtime case: MatchFacts always
		// appends the level to the fact value.
		{"non-vocab skill with level suffix matches requirement without level", "Open source contributions", matchcontract.VerdictMatch, vacancyai.ReasonSkillInProfile},
		// React is a vocabulary skill; it will match via byGroup (ReasonNamedSkill), not haveByName.
		// Either way it must be a code-decided match without a model call.
		{"vocabulary skill added by name also matches", "React", matchcontract.VerdictMatch, ""},
	}
	for _, c := range cases {
		var calls int32
		e := oneRequirement(c.text, matchcontract.RequirementSkill)
		res, err := skillMatcher(t, stub(&calls, askedNothing)).Match(context.Background(), e, facts, 1)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		v := res.Record.Verdicts[0]
		if v.Verdict != c.verdict || calls != 0 || v.Evidence == nil {
			t.Errorf("%s: want match by code, no model call: verdict=%q, calls=%d, evidence=%v", c.name, v.Verdict, calls, v.Evidence)
		}
		if c.reason != "" && v.Reason != c.reason {
			t.Errorf("%s: reason = %q, want %q", c.name, v.Reason, c.reason)
		}
	}
	// A second vacancy with the same non-vocabulary row also matches (cross-vacancy).
	var calls int32
	e2 := oneRequirement("AI coding tools", matchcontract.RequirementSkill)
	res2, err := skillMatcher(t, stub(&calls, askedNothing)).Match(context.Background(), e2, facts, 2)
	if err != nil {
		t.Fatalf("cross-vacancy: %v", err)
	}
	v2 := res2.Record.Verdicts[0]
	if v2.Verdict != matchcontract.VerdictMatch || calls != 0 {
		t.Errorf("cross-vacancy: want match by code: verdict=%q, calls=%d", v2.Verdict, calls)
	}
	// ✕: absent skill wins over haveByName; haveByName only wins when no absent entry for same name.
	factsWithAbsent := append([]matchcontract.ProfileFact(nil), facts...)
	factsWithAbsent = append(factsWithAbsent,
		matchcontract.ProfileFact{Field: "skillsAbsent", Value: "Does not have: React"},
	)
	var callsAbsent int32
	eAbsent := oneRequirement("React", matchcontract.RequirementSkill)
	resAbsent, err := skillMatcher(t, stub(&callsAbsent, askedNothing)).Match(context.Background(), eAbsent, factsWithAbsent, 1)
	if err != nil {
		t.Fatalf("absent vs have: %v", err)
	}
	// byGroup for React finds it in the profile first (via idx.byGroup, before absentByGroup).
	// The important invariant: code decides, model is not called.
	if callsAbsent != 0 {
		t.Errorf("absent vs have: model must not be called: calls=%d, verdict=%q", callsAbsent, resAbsent.Record.Verdicts[0].Verdict)
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

// A requirement for non-vocabulary "X or Y" alternatives (e.g. "Elixir or Erlang") must be
// decided by code once the owner has added one of the alternatives to their profile via ✓.
// Likewise for slash-separated alternatives (e.g. "Datadog / OpenSearch"). codeMatch shares
// splitByOr and splitBySlash with ClassifySkillAdd so the detection is consistent.
// Note: "Elixir" is in the default vocabulary (groups it; "Erlang" is not); "OpenSearch" is in
// the vocabulary (terms under Elasticsearch; "Datadog" is not). The fix must handle the mixed
// case where one alternative is vocabulary-known but only the non-vocabulary one was added.
func TestOrAndSlashAlternativesNonVocabMatchedByPart(t *testing.T) {
	type absentParts []string // helper: each string stored as a separate absent-skill fact

	cases := []struct {
		name        string
		text        string
		skills      []string      // present skills (haveByName)
		absent      bool          // absent full phrase (old ✕ behaviour)
		absentParts absentParts   // absent individual parts (new ✕ behaviour)
		verdict     matchcontract.Verdict
		reason      string
	}{
		// Elixir in vocab, Erlang not. Owner chose "Erlang" via alternatives.
		{"Erlang chosen from or-alternatives", "Elixir or Erlang", []string{"Erlang"}, false, nil, matchcontract.VerdictMatch, vacancyai.ReasonSkillInProfile},
		{"Elixir chosen from or-alternatives (via haveByName)", "Elixir or Erlang", []string{"Elixir"}, false, nil, matchcontract.VerdictMatch, ""},
		// ✕ on the full phrase (old behaviour): backward-compat path via absentByName[full phrase].
		{"or-alternatives dismissed via ✕ on full phrase", "Elixir or Erlang", nil, true, nil, matchcontract.VerdictNo, vacancyai.ReasonMarkedAbsent},
		// ✕ on each part separately (new behaviour): "Elixir" and "Erlang" each in absentByName.
		{"or-alternatives dismissed via ✕ on each part", "Elixir or Erlang", nil, false, absentParts{"Elixir", "Erlang"}, matchcontract.VerdictNo, vacancyai.ReasonMarkedAbsent},
		// OpenSearch in vocab (Elasticsearch group), Datadog not. Owner added "Datadog" from another vacancy.
		{"Datadog chosen from slash-alternatives", "Datadog / OpenSearch", []string{"Datadog"}, false, nil, matchcontract.VerdictMatch, vacancyai.ReasonSkillInProfile},
		{"OpenSearch chosen from slash-alternatives (via haveByName)", "Datadog / OpenSearch", []string{"OpenSearch"}, false, nil, matchcontract.VerdictMatch, vacancyai.ReasonSkillInProfile},
		// ✕ on each slash-part separately: "Datadog" and "OpenSearch" each in absentByName.
		{"slash-alternatives dismissed via ✕ on each part", "Datadog / OpenSearch", nil, false, absentParts{"Datadog", "OpenSearch"}, matchcontract.VerdictNo, vacancyai.ReasonMarkedAbsent},
		// ✕ on comma-list parts separately: "Python" and "Postgres" each in absentByName.
		{"comma-list all parts absent", "Python, Postgres", nil, false, absentParts{"Python", "Postgres"}, matchcontract.VerdictNo, vacancyai.ReasonMarkedAbsent},
	}
	for _, c := range cases {
		var calls int32
		var facts []matchcontract.ProfileFact
		switch {
		case c.absent:
			facts = []matchcontract.ProfileFact{
				{Field: "skillsAbsent", Value: "Does not have: " + c.text},
			}
		case len(c.absentParts) > 0:
			for _, part := range c.absentParts {
				facts = append(facts, matchcontract.ProfileFact{Field: "skillsAbsent", Value: "Does not have: " + part})
			}
		default:
			for _, s := range c.skills {
				facts = append(facts, matchcontract.ProfileFact{Field: "skills", Value: s})
			}
		}
		e := oneRequirement(c.text, matchcontract.RequirementSkill)
		res, err := skillMatcher(t, stub(&calls, askedNothing)).Match(context.Background(), e, facts, 1)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		v := res.Record.Verdicts[0]
		if v.Verdict != c.verdict || calls != 0 {
			t.Errorf("%s: want verdict=%q by code (0 model calls): got verdict=%q calls=%d reason=%q", c.name, c.verdict, v.Verdict, calls, v.Reason)
		}
		if c.reason != "" && v.Reason != c.reason {
			t.Errorf("%s: reason = %q, want %q", c.name, v.Reason, c.reason)
		}
	}
}

// A comma-separated list requirement (e.g. "Python, Postgres, AWS, distributed systems") should
// be decided by code once the owner has added each part as a skill via ✓. The leftover branch of
// codeMatch splits the requirement text and checks each part in haveByName.
func TestCommaListRequirementMatchedAfterAddingParts(t *testing.T) {
	// Profile has three vocabulary skills and one non-vocabulary skill added by name.
	facts := []matchcontract.ProfileFact{
		{Field: "skills", Value: "Python (Proficient)"},
		{Field: "skills", Value: "Postgres (Proficient)"},
		{Field: "skills", Value: "AWS (Proficient)"},
		{Field: "skills", Value: "distributed systems"},
	}

	var calls int32
	// All four parts present in profile → full match by code.
	e := oneRequirement("Python, Postgres, AWS, distributed systems", matchcontract.RequirementSkill)
	res, err := skillMatcher(t, stub(&calls, askedNothing)).Match(context.Background(), e, facts, 1)
	if err != nil {
		t.Fatalf("full match: %v", err)
	}
	v := res.Record.Verdicts[0]
	if v.Verdict != matchcontract.VerdictMatch || calls != 0 {
		t.Errorf("all parts in profile → full match by code: verdict=%q, calls=%d", v.Verdict, calls)
	}
	if v.Reason != vacancyai.ReasonSkillInProfile {
		t.Errorf("reason = %q, want ReasonSkillInProfile", v.Reason)
	}

	// Only three of four parts present → partial match by code.
	factsPartial := []matchcontract.ProfileFact{
		{Field: "skills", Value: "Python (Proficient)"},
		{Field: "skills", Value: "Postgres (Proficient)"},
		{Field: "skills", Value: "AWS (Proficient)"},
		// "distributed systems" not in profile
	}
	var calls2 int32
	res2, err := skillMatcher(t, stub(&calls2, askedNothing)).Match(context.Background(), e, factsPartial, 1)
	if err != nil {
		t.Fatalf("partial match: %v", err)
	}
	v2 := res2.Record.Verdicts[0]
	if v2.Verdict != matchcontract.VerdictPartial || calls2 != 0 {
		t.Errorf("three of four parts → partial match by code: verdict=%q, calls=%d", v2.Verdict, calls2)
	}
}
