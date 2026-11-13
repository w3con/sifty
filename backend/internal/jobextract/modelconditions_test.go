package jobextract_test

import (
	"testing"

	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
)

func factsFor(t *testing.T, description string) matchcontract.VacancyFacts {
	t.Helper()
	f, err := jobextract.Extract(matchcontract.RetainedSource{
		JobID: 7, Title: "Engineer", Company: "Acme", DescriptionStatus: "available",
		Description: description,
	}, jobextract.Policy{})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return f
}

func cond(value, quote string) *matchcontract.ExtractedCondition {
	return &matchcontract.ExtractedCondition{Value: value, Quote: quote}
}

func TestMergeModelConditionsFillsOnlyWhatTheParseMissed(t *testing.T) {
	f := factsFor(t, "We are a distributed team. Everyone works remotely.")
	if len(f.Conditions.WorkModes) != 0 || f.Compensation.Cash != nil || f.Conditions.Engagement != nil {
		t.Fatalf("fixture must have no conditions found by the ordinary parse: %+v", f.Conditions)
	}
	merged, conflicts, err := jobextract.MergeModelConditions(f, matchcontract.ExtractedConditions{
		WorkMode:     cond("remote", "Everyone works remotely"),
		Engagement:   cond("permanent employee", "full-time employees"),
		Hours:        cond("full-time", "full-time"),
		Timezone:     cond("CET", "CET"),
		Compensation: cond("USD 120,000 - 150,000 per year gross", "USD 120,000 - 150,000 per year gross"),
	})
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("merge: err=%v conflicts=%+v", err, conflicts)
	}
	if len(merged.Conditions.WorkModes) != 1 || merged.Conditions.WorkModes[0].Mode != matchcontract.WorkRemote {
		t.Fatalf("work mode not filled: %+v", merged.Conditions.WorkModes)
	}
	if merged.Conditions.Engagement == nil || merged.Conditions.Engagement.Type != matchcontract.EngagementEmployment {
		t.Fatalf("engagement not filled: %+v", merged.Conditions.Engagement)
	}
	if merged.Conditions.Hours == nil || merged.Conditions.Hours.Mode != matchcontract.HoursFullTime {
		t.Fatalf("hours not filled: %+v", merged.Conditions.Hours)
	}
	if merged.Conditions.Timezone == nil || merged.Conditions.Timezone.Value != "CET" {
		t.Fatalf("timezone not filled: %+v", merged.Conditions.Timezone)
	}
	cash := merged.Compensation.Cash
	if cash == nil || cash.Min == nil || *cash.Min != 120000 || cash.Max == nil || *cash.Max != 150000 || cash.Currency != "USD" {
		t.Fatalf("compensation not filled: %+v", cash)
	}
	if err := merged.VerifyFactsHash(); err != nil || merged.FactsHash == f.FactsHash {
		t.Fatalf("merged facts must carry a fresh, verifiable hash: %v", err)
	}
	if len(f.Conditions.WorkModes) != 0 {
		t.Fatalf("the input facts were modified")
	}
}

func TestMergeModelConditionsLetsTheDeterministicFactWinAndKeepsTheConflict(t *testing.T) {
	f := factsFor(t, "Work mode: Hybrid\nEmployment type: Contract\nSalary: USD 80,000 - 100,000 per year gross")
	merged, conflicts, err := jobextract.MergeModelConditions(f, matchcontract.ExtractedConditions{
		WorkMode:     cond("fully remote", "fully remote"),
		Engagement:   cond("permanent employee", "permanent employee"),
		Compensation: cond("USD 120,000 - 150,000 per year gross", "USD 120,000 - 150,000 per year gross"),
	})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(merged.Conditions.WorkModes) != 1 || merged.Conditions.WorkModes[0].Mode != matchcontract.WorkHybrid {
		t.Fatalf("the deterministic work mode must stand: %+v", merged.Conditions.WorkModes)
	}
	if merged.Conditions.Engagement.Type != matchcontract.EngagementContract {
		t.Fatalf("the deterministic engagement must stand: %+v", merged.Conditions.Engagement)
	}
	if *merged.Compensation.Cash.Max != 100000 {
		t.Fatalf("the deterministic pay must stand: %+v", merged.Compensation.Cash)
	}
	fields := map[string]bool{}
	for _, c := range conflicts {
		fields[c.Field] = true
		if c.Deterministic == "" || c.Model == "" || c.Quote == "" {
			t.Errorf("a conflict must carry both sides and the quote: %+v", c)
		}
	}
	for _, want := range []string{"work_mode", "engagement", "compensation"} {
		if !fields[want] {
			t.Errorf("missing conflict for %s: %+v", want, conflicts)
		}
	}
}

func TestMergeModelConditionsIgnoresValuesItCannotReadAsTyped(t *testing.T) {
	f := factsFor(t, "Nothing stated here.")
	merged, conflicts, err := jobextract.MergeModelConditions(f, matchcontract.ExtractedConditions{
		WorkMode:     cond("flexible", "flexible"),
		Engagement:   cond("interesting", "interesting"),
		Compensation: cond("competitive", "competitive salary"),
	})
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("merge: err=%v conflicts=%+v", err, conflicts)
	}
	if len(merged.Conditions.WorkModes) != 0 || merged.Conditions.Engagement != nil || merged.Compensation.Cash != nil {
		t.Fatalf("an unreadable value must not become a fact: %+v", merged)
	}
}
