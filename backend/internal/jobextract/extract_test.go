package jobextract_test

import (
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
)

func TestExtractEvidenceAndSectionPreservation(t *testing.T) {
	src := matchcontract.RetainedSource{
		JobID:             42,
		Title:             "Senior Rust Engineer",
		Company:           "Acme",
		Location:          "Berlin, Germany",
		Tags:              []string{"Rust", "Go"},
		DescriptionStatus: "available structured description",
		Sources:           []string{"Wellfound"},
		Description: strings.Join([]string{
			"About us:",
			"- Our stack is Rust and Postgres",
			"",
			"Requirements:",
			"- Rust",
			"- Go",
			"",
			"Preferred:",
			"- Kubernetes",
			"",
			"Benefits:",
			"- free lunch",
			"",
			"Salary: USD 80,000 - 100,000 per year gross",
			"Permitted hiring locations: Berlin, Germany",
			"Work mode: Hybrid",
		}, "\n"),
	}
	policy := jobextract.Policy{Source: matchcontract.SourcePolicy{
		Version: "source-policy/1",
		Sources: map[string]matchcontract.SourcePolicyRule{
			"Wellfound": {Location: matchcontract.ReliabilityNeedsCorroboration},
		},
	}}

	f, err := jobextract.Extract(src, policy)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if f.FactsHash == "" || f.RawSourceHash == "" {
		t.Fatalf("facts need both hashes: %+v", f)
	}
	if err := f.VerifyFactsHash(); err != nil {
		t.Fatalf("facts hash verification: %v", err)
	}
	if len(f.Professional.Requirements) != 3 {
		t.Fatalf("requirements = %d, want 3: %+v", len(f.Professional.Requirements), f.Professional.Requirements)
	}
	requiredRust, ok := findByLabel(f.Professional.Requirements, "Rust")
	if !ok {
		t.Fatalf("missing required Rust fact: %+v", f.Professional.Requirements)
	}
	if requiredRust.Importance != matchcontract.ImportanceRequired ||
		requiredRust.Evidence.Section != matchcontract.SectionRequired ||
		requiredRust.Evidence.Quote != "- Rust" {
		t.Errorf("required evidence = %+v, want exact retained quote in required section", requiredRust)
	}
	preferred, ok := findByLabel(f.Professional.Requirements, "Kubernetes")
	if !ok || preferred.Importance != matchcontract.ImportancePreferred ||
		preferred.Evidence.Section != matchcontract.SectionPreferred {
		t.Errorf("preferred fact = %+v, want preferred Kubernetes with preferred section", preferred)
	}
	if len(f.Professional.ContextMentions) != 1 ||
		f.Professional.ContextMentions[0].Evidence.Section != matchcontract.SectionCompanyContext {
		t.Errorf("context mentions = %+v, want company-context stack mention", f.Professional.ContextMentions)
	}
	if c := f.Compensation.Cash; c == nil {
		t.Fatal("expected explicit salary fact")
	} else if c.Currency != "USD" || c.Period != matchcontract.PeriodYear ||
		c.TaxBasis != matchcontract.TaxGross || c.Max == nil || *c.Max != 100000 ||
		c.Min == nil || *c.Min != 80000 || c.Evidence.Reliability != matchcontract.ReliabilityUsable {
		t.Errorf("cash fact = %+v, want usable USD/year/gross 80k-100k range", c)
	}
	if len(f.Geography.HiringLocations) != 1 ||
		f.Geography.HiringLocations[0].Evidence.Reliability != matchcontract.ReliabilityUsable {
		t.Errorf("locations = %+v, want one corroborated usable Berlin location", f.Geography.HiringLocations)
	}
	if len(f.Conditions.WorkModes) != 1 || f.Conditions.WorkModes[0].Mode != matchcontract.WorkHybrid {
		t.Errorf("work modes = %+v, want hybrid", f.Conditions.WorkModes)
	}
	if _, ok := findByLabel(f.Professional.Requirements, "free lunch"); ok {
		t.Errorf("Benefits bullet leaked into requirements: %+v", f.Professional.Requirements)
	}

	again, err := jobextract.Extract(src, policy)
	if err != nil {
		t.Fatalf("second Extract: %v", err)
	}
	if again.FactsHash != f.FactsHash {
		t.Errorf("facts hash changed between identical runs: %s vs %s", f.FactsHash, again.FactsHash)
	}
	reordered := src
	reordered.Tags = []string{"Go", "Rust"}
	reorderedHash, err := matchcontract.HashRetainedSource(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if reorderedHash != f.RawSourceHash {
		t.Errorf("tag order changed source hash: %s vs %s", f.RawSourceHash, reorderedHash)
	}
	if err := matchcontract.CheckSchema("2"); err == nil {
		t.Error("incompatible schema was accepted")
	}
	if err := matchcontract.CheckSchema(""); err == nil {
		t.Error("missing schema version was accepted")
	}

	structured := src
	min, max := 90000.0, 110000.0
	structured.Structured = &matchcontract.StructuredFields{Cash: &matchcontract.DeclaredCash{
		Min: &min, Max: &max, Currency: "EUR", Period: matchcontract.PeriodYear, TaxBasis: matchcontract.TaxGross,
	}}
	sf, err := jobextract.Extract(structured, policy)
	if err != nil {
		t.Fatalf("structured Extract: %v", err)
	}
	if sf.Compensation.Cash == nil || sf.Compensation.Cash.Evidence.Field != "structured.cash" ||
		sf.Compensation.Cash.Currency != "EUR" || sf.Compensation.Cash.Max == nil || *sf.Compensation.Cash.Max != max {
		t.Errorf("structured cash did not take precedence: %+v", sf.Compensation.Cash)
	}

	generic := src
	generic.Location = "Berlin, Germany"
	generic.Description = "Location: Berlin, Germany"
	gf, err := jobextract.Extract(generic, policy)
	if err != nil {
		t.Fatalf("generic location Extract: %v", err)
	}
	if len(gf.Geography.HiringLocations) != 1 ||
		gf.Geography.HiringLocations[0].Evidence.Reliability != matchcontract.ReliabilityNeedsCorroboration {
		t.Errorf("generic Location label became a hiring restriction: %+v", gf.Geography.HiringLocations)
	}

	conflict := src
	conflict.Location = "San Francisco, CA"
	conflict.Description = "Permitted hiring locations: Berlin, Germany"
	conflictFacts, err := jobextract.Extract(conflict, policy)
	if err != nil {
		t.Fatalf("conflicting location Extract: %v", err)
	}
	if len(conflictFacts.Geography.HiringLocations) != 2 {
		t.Fatalf("conflicting locations = %+v, want both board and text preserved", conflictFacts.Geography.HiringLocations)
	}
	for _, loc := range conflictFacts.Geography.HiringLocations {
		if loc.Evidence.Reliability != matchcontract.ReliabilityNeedsCorroboration {
			t.Errorf("conflicting location %q reliability = %s, want needs_corroboration", loc.Label, loc.Evidence.Reliability)
		}
	}
	if len(conflictFacts.Ambiguities) == 0 || conflictFacts.Ambiguities[0].Group != matchcontract.GroupGeography {
		t.Errorf("conflicting locations did not record geography ambiguity: %+v", conflictFacts.Ambiguities)
	}
	pinned := matchcontract.SourcePolicy{Version: "source-policy/1", Sources: map[string]matchcontract.SourcePolicyRule{
		"Wellfound": {Location: matchcontract.ReliabilityUnusable},
	}}
	if got := pinned.LocationReliability([]string{"Wellfound", "Unknown"}); got != matchcontract.ReliabilityUnusable {
		t.Errorf("unknown source masked unusable declared source: got %s", got)
	}

	cashCases := []struct {
		description string
		wantNil     bool
		wantMaxNil  bool
		wantRel     matchcontract.Reliability
	}{
		{"Salary survey: USD 80,000 - 100,000 per year gross", true, false, ""},
		{"Base salary: from USD 80,000 per year gross", false, true, matchcontract.ReliabilityNeedsCorroboration},
		{"Base salary: USD 80,000 per year", false, false, matchcontract.ReliabilityNeedsCorroboration},
		{"Base salary: USD 80,000 - 100,000 per year gross plus bonus", true, false, ""},
	}
	for _, tc := range cashCases {
		c := src
		c.Description = tc.description
		cf, err := jobextract.Extract(c, policy)
		if err != nil {
			t.Fatalf("cash Extract %q: %v", tc.description, err)
		}
		if tc.wantNil {
			if cf.Compensation.Cash != nil {
				t.Errorf("ambiguous salary %q produced cash %+v", tc.description, cf.Compensation.Cash)
			}
			continue
		}
		if cf.Compensation.Cash == nil {
			t.Errorf("explicit salary %q produced no cash fact", tc.description)
			continue
		}
		if tc.wantMaxNil != (cf.Compensation.Cash.Max == nil) {
			t.Errorf("salary %q Max nil = %v, want %v", tc.description, cf.Compensation.Cash.Max == nil, tc.wantMaxNil)
		}
		if cf.Compensation.Cash.Evidence.Reliability != tc.wantRel {
			t.Errorf("salary %q reliability = %s, want %s", tc.description, cf.Compensation.Cash.Evidence.Reliability, tc.wantRel)
		}
	}

	authCases := []struct {
		description string
		wantNil     bool
		want        string
	}{
		{"Work authorization: no sponsorship needed", true, ""},
		{"Work authorization: required in Germany", false, matchcontract.AuthorizationRequired},
		{"You must be authorized to work in Germany.", false, matchcontract.AuthorizationRequired},
	}
	for _, tc := range authCases {
		a := src
		a.Description = tc.description
		af, err := jobextract.Extract(a, policy)
		if err != nil {
			t.Fatalf("authorization Extract %q: %v", tc.description, err)
		}
		got := af.Geography.WorkAuthorization
		if tc.wantNil {
			if got != nil {
				t.Errorf("ambiguous authorization %q produced %+v", tc.description, got)
			}
			continue
		}
		if got == nil || got.Requirement != tc.want {
			t.Errorf("authorization %q = %+v, want %s", tc.description, got, tc.want)
		}
	}
}

func findByLabel(reqs []matchcontract.RequirementFact, label string) (matchcontract.RequirementFact, bool) {
	for _, r := range reqs {
		if r.Label == label {
			return r, true
		}
	}
	return matchcontract.RequirementFact{}, false
}

func TestExtractDoesNotInferFlattenedSections(t *testing.T) {
	src := matchcontract.RetainedSource{
		JobID:             7,
		Title:             "Backend Engineer",
		Company:           "Acme",
		Location:          "Remote",
		DescriptionStatus: "feed description; may be excerpt",
		Sources:           []string{"Wellfound"},
		Description:       "We build developer tools for small teams.\n- Rust\n- Go\n- Postgres",
	}
	policy := jobextract.Policy{Source: matchcontract.SourcePolicy{
		Version: "source-policy/1",
		Sources: map[string]matchcontract.SourcePolicyRule{
			"Wellfound": {Location: matchcontract.ReliabilityNeedsCorroboration},
		},
	}}
	f, err := jobextract.Extract(src, policy)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(f.Professional.Requirements) != 0 {
		t.Fatalf("flattened bullets became requirements: %+v", f.Professional.Requirements)
	}
	if len(f.Ambiguities) != 1 ||
		!strings.Contains(f.Ambiguities[0].Detail, "bullets have no retained required/preferred heading") ||
		f.Ambiguities[0].Evidence.Section != matchcontract.SectionUnknown {
		t.Errorf("ambiguities = %+v, want one unknown-section flattened-input note", f.Ambiguities)
	}
	if len(f.Geography.HiringLocations) != 0 {
		t.Errorf("remote board field became a hiring location: %+v", f.Geography.HiringLocations)
	}
	if len(f.Conditions.WorkModes) != 1 || f.Conditions.WorkModes[0].Mode != matchcontract.WorkRemote {
		t.Errorf("work modes = %+v, want explicit remote condition", f.Conditions.WorkModes)
	}
}
