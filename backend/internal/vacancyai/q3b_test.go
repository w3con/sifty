package vacancyai_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/matchcontract"
)

func extractOf(t *testing.T, description, answer string) (matchcontract.VacancyExtraction, []string) {
	t.Helper()
	var calls int32
	res, err := extractor(stub(&calls, answer)).Extract(context.Background(), source(description))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return res.Record, res.Skipped
}

func TestAlternativesGroupIsOneRequirement(t *testing.T) {
	desc := "Experience with a backend language such as Kotlin, Python or Go. Strong interest or background in distributed systems."
	rec, skipped := extractOf(t, desc, `{"requirements":[
	 {"text":"Backend language","kind":"skill","level":"required","quote":"Experience with a backend language such as Kotlin, Python or Go.","alternatives":["Kotlin","Python","Go","Haskell"]},
	 {"text":"Distributed systems","kind":"experience","level":"preferred","quote":"Strong interest or background in distributed systems.","alternatives":[]}],
	 "conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`)
	if len(rec.Requirements) != 2 {
		t.Fatalf("requirements = %+v", rec.Requirements)
	}
	got := rec.Requirements[0].Alternatives
	if strings.Join(got, ",") != "Kotlin,Python,Go" {
		t.Errorf("alternatives = %v; an item the text does not name must be dropped", got)
	}
	if !strings.Contains(strings.Join(skipped, "\n"), "Haskell") {
		t.Errorf("the dropped alternative is not explained: %v", skipped)
	}
	if rec.Requirements[1].Level != matchcontract.ImportancePreferred || rec.Requirements[1].Alternatives != nil {
		t.Errorf("single demand = %+v", rec.Requirements[1])
	}
}

func TestOneAlternativeIsNotAGroup(t *testing.T) {
	rec, _ := extractOf(t, "You know Go well.", `{"requirements":[
	 {"text":"Go","kind":"skill","level":"required","quote":"You know Go well.","alternatives":["Go"]}],
	 "conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`)
	if rec.Requirements[0].Alternatives != nil {
		t.Errorf("a one-member group is a single demand: %v", rec.Requirements[0].Alternatives)
	}
}

func TestAlternativesChangeTheRequirementsHash(t *testing.T) {
	a := matchcontract.VacancyExtraction{SchemaVersion: "1", ExtractorVersion: "x", SourceHash: "h",
		Requirements: []matchcontract.ExtractedRequirement{{Text: "Backend language", Kind: "skill", Level: "required", Quote: "q"}}}
	b := a
	b.Requirements = []matchcontract.ExtractedRequirement{{Text: "Backend language", Kind: "skill", Level: "required", Quote: "q", Alternatives: []string{"Go", "Rust"}}}
	ha, _ := a.RequirementsHash()
	hb, _ := b.RequirementsHash()
	if ha == hb {
		t.Fatal("verdicts for a plain requirement must not be reused for a group")
	}
}

func TestQuoteToleratesTypographyButStoresTheOriginal(t *testing.T) {
	desc := "We’re looking for someone with deep   Rust experience — ideally 5+ years.\nCandidates “must” be reachable."
	rec, skipped := extractOf(t, desc, `{"requirements":[
	 {"text":"Rust","kind":"skill","level":"required","quote":"We're looking for someone with deep Rust experience - ideally 5+ years.","alternatives":[]},
	 {"text":"Reachable","kind":"other","level":"required","quote":"Candidates \"must\" be reachable.","alternatives":[]},
	 {"text":"Invented","kind":"other","level":"required","quote":"We pay in gold coins.","alternatives":[]}],
	 "conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`)
	if len(rec.Requirements) != 2 {
		t.Fatalf("requirements = %+v skipped = %v", rec.Requirements, skipped)
	}
	for _, r := range rec.Requirements {
		if !strings.Contains(desc, r.Quote) {
			t.Errorf("the stored quote must be the source's own text, got %q", r.Quote)
		}
	}
	if !strings.Contains(rec.Requirements[0].Quote, "’") || !strings.Contains(rec.Requirements[0].Quote, "—") {
		t.Errorf("the original apostrophe and dash must be kept: %q", rec.Requirements[0].Quote)
	}
}

func TestQuoteDoesNotForgiveChangedWords(t *testing.T) {
	desc := "You will write Rust services."
	if _, skipped := extractOf(t, desc, `{"requirements":[
	 {"text":"Rust","kind":"skill","level":"required","quote":"You will write Go services.","alternatives":[]},
	 {"text":"Rust","kind":"skill","level":"required","quote":"You will write Rust services.","alternatives":[]}],
	 "conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`); len(skipped) != 1 {
		t.Fatalf("skipped = %v, want exactly the Go quote", skipped)
	}
}

func TestHybridDutiesIsNotAWorkMode(t *testing.T) {
	desc := "This is a 50/50 hybrid role between engineering and product. We offer 30 days of vacation."
	rec, skipped := extractOf(t, desc, `{"requirements":[],
	 "conditions":{"compensation":null,"workMode":{"value":"hybrid","quote":"This is a 50/50 hybrid role between engineering and product."},
	 "engagement":null,"location":null,"timezone":null,"hours":null}}`)
	if rec.Conditions.WorkMode != nil {
		t.Fatalf("a mix of duties is not a work mode: %+v", rec.Conditions.WorkMode)
	}
	if !strings.Contains(strings.Join(skipped, "\n"), "place of work") {
		t.Errorf("skipped = %v", skipped)
	}
}

func TestWorkModeFromPlaceOfWorkSentences(t *testing.T) {
	for _, tc := range []struct{ sentence, value, want string }{
		{"This is a hybrid role with three days in the office.", "hybrid", "hybrid"},
		{"The position is fully remote.", "Remote (fully)", "remote"},
		{"You will work on-site in Berlin.", "on-site", "office"},
		{"Hybrid office model in London.", "hybrid", "hybrid"},
	} {
		rec, _ := extractOf(t, tc.sentence, `{"requirements":[],"conditions":{"compensation":null,
		 "workMode":{"value":"`+tc.value+`","quote":"`+tc.sentence+`"},"engagement":null,"location":null,"timezone":null,"hours":null}}`)
		if rec.Conditions.WorkMode == nil || rec.Conditions.WorkMode.Value != tc.want {
			t.Errorf("%q: work mode = %+v, want %s", tc.sentence, rec.Conditions.WorkMode, tc.want)
		}
	}
}

func TestHoursOnlyFromExplicitHours(t *testing.T) {
	desc := "This is a permanent position with real ownership. We expect full-time availability. Around 20 hours per week for the first month."
	cond := func(value, quote string) string {
		return `{"requirements":[],"conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,
		 "hours":{"value":"` + value + `","quote":"` + quote + `"}}}`
	}
	if rec, _ := extractOf(t, desc, cond("full-time", "This is a permanent position with real ownership.")); rec.Conditions.Hours != nil {
		t.Errorf("hours inferred from 'permanent' must be refused: %+v", rec.Conditions.Hours)
	}
	if rec, _ := extractOf(t, desc, cond("full-time", "We expect full-time availability.")); rec.Conditions.Hours == nil {
		t.Error("an explicit full-time statement must be kept")
	}
	if rec, _ := extractOf(t, desc, cond("20 hours per week", "Around 20 hours per week for the first month.")); rec.Conditions.Hours == nil {
		t.Error("explicit weekly hours must be kept")
	}
}

func TestEngagementOnlyFromAStatedKindOfAgreement(t *testing.T) {
	// Quotes taken from real vacancies where a model had read "employment" out of other sentences.
	refused := []string{
		"All qualified applicants will be treated with respect and receive equal consideration for employment.",
		"Competitive salary package.",
		"This position will be open until it is successfully filled.",
		"The base salary for this role is between $230,000-$280,000 + equity + benefits.",
		"Check participates in E-Verify and will provide Form I-9 information from all new employees.",
	}
	kept := []string{
		"Job type Full-Time",
		"Team members will be taken on as a contractor via our UK entity.",
		"This is a permanent position.",
		"Freelance, 3 months, extendable.",
	}
	cond := func(quote string) string {
		return `{"requirements":[],"conditions":{"compensation":null,"workMode":null,"location":null,"timezone":null,"hours":null,
		 "engagement":{"value":"employment","quote":"` + quote + `"}}}`
	}
	for _, q := range refused {
		if rec, skipped := extractOf(t, q, cond(q)); rec.Conditions.Engagement != nil {
			t.Errorf("%q: an engagement without a stated agreement must be refused", q)
		} else if !strings.Contains(strings.Join(skipped, "\n"), "kind of agreement") {
			t.Errorf("%q: skipped = %v", q, skipped)
		}
	}
	for _, q := range kept {
		if rec, _ := extractOf(t, q, cond(q)); rec.Conditions.Engagement == nil {
			t.Errorf("%q: a stated kind of agreement must be kept", q)
		}
	}
}

func TestExtractionKeyIsTheDescriptionAlone(t *testing.T) {
	var calls int32
	e := extractor(stub(&calls, goodExtraction))
	a, err := e.Extract(context.Background(), source(glued))
	if err != nil {
		t.Fatal(err)
	}
	moved := source(glued)
	moved.Location, moved.Tags, moved.Sources, moved.Title, moved.Company = "Paris", []string{"x"}, []string{"Another"}, "Renamed", "Other"
	b, err := e.Extract(context.Background(), moved)
	if err != nil {
		t.Fatal(err)
	}
	if a.Record.SourceHash != b.Record.SourceHash || a.Record.SourceHash != matchcontract.HashDescription(glued) {
		t.Fatalf("the key moved with something other than the text: %s %s", a.Record.SourceHash, b.Record.SourceHash)
	}
	if c, _ := e.Extract(context.Background(), source(glued+" ")); c.Record.SourceHash == a.Record.SourceHash {
		t.Fatal("a changed text must change the key")
	}
}

func TestMatcherShowsAlternativesAndComputedFactsToTheModel(t *testing.T) {
	e := matchcontract.VacancyExtraction{
		SchemaVersion: matchcontract.SchemaVersion, ExtractorVersion: "x", JobID: 1, SourceHash: "h",
		Requirements: []matchcontract.ExtractedRequirement{{
			Text: "Backend language", Kind: matchcontract.RequirementSkill, Level: matchcontract.ImportanceRequired,
			Quote: "q", Alternatives: []string{"Kotlin", "Python", "Go"},
		}},
	}
	facts := []matchcontract.ProfileFact{{Field: "skills", Value: "Python scripting (Proficient)"}}
	var request string
	infer := func(_ context.Context, _ assess.Provider, req string, _ any, out any) error {
		request = req
		return json.Unmarshal([]byte(`{"verdicts":[{"index":0,"verdict":"match","evidenceId":"P1","reason":"Python is in the group"}]}`), out)
	}
	res, err := matcher(infer).Match(context.Background(), e, facts, 1)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if !strings.Contains(request, "(any of: Kotlin | Python | Go)") {
		t.Errorf("the group is not shown to the model:\n%s", request)
	}
	if v := res.Record.Verdicts[0]; v.Verdict != matchcontract.VerdictMatch || v.Evidence == nil || v.Evidence.Value != "Python scripting (Proficient)" {
		t.Errorf("verdict = %+v", v)
	}
}

const emptyAnswer = `{"requirements":[],"conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`

func TestEmptyAnswerForATextThatStatesRequirementsIsAskedAgain(t *testing.T) {
	desc := "Requirements\n5+ years of Go experience.\nYou have shipped services."
	good := `{"requirements":[{"text":"5+ years Go","kind":"experience","level":"required","quote":"5+ years of Go experience.","alternatives":[]}],
	 "conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`
	var calls int32
	res, err := extractor(stub(&calls, emptyAnswer, good)).Extract(context.Background(), source(desc))
	if err != nil || calls != 2 || len(res.Record.Requirements) != 1 {
		t.Fatalf("err=%v calls=%d requirements=%d; an empty answer for a text with requirements must be retried", err, calls, len(res.Record.Requirements))
	}
	// An answer that stays empty through every attempt is accepted as final, so the vacancy is not
	// stuck pending forever.
	calls = 0
	res, err = extractor(stub(&calls, emptyAnswer)).Extract(context.Background(), source(desc))
	if err != nil || calls != 3 || len(res.Record.Requirements) != 0 {
		t.Fatalf("err=%v calls=%d requirements=%d; want a final empty answer after three attempts", err, calls, len(res.Record.Requirements))
	}
	// A text with no wording of a requirement is final at once.
	calls = 0
	res, err = extractor(stub(&calls, emptyAnswer)).Extract(context.Background(), source("A warm welcome to our page."))
	if err != nil || calls != 1 || len(res.Record.Requirements) != 0 {
		t.Fatalf("err=%v calls=%d; a text that states no requirement is final on the first answer", err, calls)
	}
}
