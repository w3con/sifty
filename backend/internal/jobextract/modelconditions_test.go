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
	}, nil)
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
	}, nil)
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

// TestMergeModelConditionsZeroFactsWithSalaryExtraction covers the routes.go fix for
// vacancy IDs that have no stored VacancyFacts (facts == nil): routes.go seeds mergedFacts
// from extraction metadata, then calls MergeModelConditions — which must populate Cash from
// the extraction's compensation condition and produce a valid, verifiable hash.
func TestMergeModelConditionsZeroFactsWithSalaryExtraction(t *testing.T) {
	// Mirrors the seed routes.go builds when facts == nil but extraction != nil.
	zero := matchcontract.VacancyFacts{
		SchemaVersion:    matchcontract.SchemaVersion,
		JobID:            1011,
		ExtractorVersion: matchcontract.ExtractorVersion,
		RawSourceHash:    "deadbeef",
	}
	merged, conflicts, err := jobextract.MergeModelConditions(zero, matchcontract.ExtractedConditions{
		Compensation: cond("USD 150,000 - 250,000 per year gross", "base salary $150,000–$250,000"),
	}, nil)
	if err != nil {
		t.Fatalf("MergeModelConditions on zero facts: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
	}
	cash := merged.Compensation.Cash
	if cash == nil {
		t.Fatal("cash must be populated from extraction when facts are zero")
	}
	if cash.Min == nil || *cash.Min != 150000 {
		t.Errorf("cash.Min = %v, want 150000", cash.Min)
	}
	if cash.Max == nil || *cash.Max != 250000 {
		t.Errorf("cash.Max = %v, want 250000", cash.Max)
	}
	if cash.Currency != "USD" {
		t.Errorf("cash.Currency = %q, want USD", cash.Currency)
	}
	if err := merged.VerifyFactsHash(); err != nil {
		t.Errorf("merged facts hash invalid: %v", err)
	}
}

// TestCashTextCurrencySymbolDetection covers the fix for cashText losing currency when the
// extraction value contains a "$" sign but no ISO code (e.g. "$230,000-$280,000 per year").
// Also verifies that a bare number range with no sign or code leaves currency empty.
func TestCashTextCurrencySymbolDetection(t *testing.T) {
	zero := matchcontract.VacancyFacts{
		SchemaVersion:    matchcontract.SchemaVersion,
		JobID:            1011,
		ExtractorVersion: matchcontract.ExtractorVersion,
		RawSourceHash:    "deadbeef",
	}

	// "$230,000-$280,000 per year" — dollar sign in value, no ISO code → must resolve to USD.
	merged, _, err := jobextract.MergeModelConditions(zero, matchcontract.ExtractedConditions{
		Compensation: cond("$230,000-$280,000 per year", "The base salary is between $230,000-$280,000 per year."),
	}, nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	cash := merged.Compensation.Cash
	if cash == nil {
		t.Fatal("cash must be populated")
	}
	if cash.Currency != "USD" {
		t.Errorf("dollar sign: cash.Currency = %q, want USD", cash.Currency)
	}
	if cash.Min == nil || *cash.Min != 230000 || cash.Max == nil || *cash.Max != 280000 {
		t.Errorf("dollar sign: cash range = %v–%v, want 230000–280000", cash.Min, cash.Max)
	}

	// "230,000-280,000 per year" — no sign, no code → currency must be empty (unknown).
	zeroB := matchcontract.VacancyFacts{
		SchemaVersion:    matchcontract.SchemaVersion,
		JobID:            1012,
		ExtractorVersion: matchcontract.ExtractorVersion,
		RawSourceHash:    "deadbeef",
	}
	mergedB, _, err := jobextract.MergeModelConditions(zeroB, matchcontract.ExtractedConditions{
		Compensation: cond("230,000-280,000 per year", "salary 230,000-280,000 per year"),
	}, nil)
	if err != nil {
		t.Fatalf("merge no-sign: %v", err)
	}
	cashB := mergedB.Compensation.Cash
	if cashB == nil {
		t.Fatal("cash must be populated even without currency")
	}
	if cashB.Currency != "" {
		t.Errorf("no sign/code: cash.Currency = %q, want empty", cashB.Currency)
	}
}

func TestMergeModelConditionsIgnoresValuesItCannotReadAsTyped(t *testing.T) {
	f := factsFor(t, "Nothing stated here.")
	merged, conflicts, err := jobextract.MergeModelConditions(f, matchcontract.ExtractedConditions{
		WorkMode:     cond("flexible", "flexible"),
		Engagement:   cond("interesting", "interesting"),
		Compensation: cond("competitive", "competitive salary"),
	}, nil)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("merge: err=%v conflicts=%+v", err, conflicts)
	}
	if len(merged.Conditions.WorkModes) != 0 || merged.Conditions.Engagement != nil || merged.Compensation.Cash != nil {
		t.Fatalf("an unreadable value must not become a fact: %+v", merged)
	}
}
