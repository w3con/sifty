package profile_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
)

func TestMatchFactsProjectCapabilityAndNothingElse(t *testing.T) {
	p := &profile.Profile{Data: profile.ProfileData{
		Identity:     profile.Identity{DisplayName: "Private Name", Headline: "CTO and engineering leader"},
		Demographics: profile.Demographics{Gender: "x", Race: []string{"y"}},
		Skills:       []profile.Skill{{Name: "Go", Level: "expert"}, {Name: "  "}, {Name: "Rust"}},
		Experience: []profile.Experience{{
			Company: "Acme", Title: "CTO", Start: "2019", Current: true,
			Summary: "Led   four engineering teams.\nShipped payments.",
		}},
		Languages:   []profile.Language{{Name: "English", Level: "C1"}},
		TargetRoles: []string{"Staff Engineer"},
		Geography:   profile.Geography{WorkAuthorizations: []profile.WorkAuthorization{{Country: "fr", Status: "citizen"}}},
	}}
	got := profile.MatchFactsAt(p, time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC))
	want := []matchcontract.ProfileFact{
		{Field: "headline", Value: "CTO and engineering leader"},
		{Field: "skills", Value: "Go (expert)"},
		{Field: "skills", Value: "Rust"},
		{Field: "experience", Value: "CTO at Acme, 2019 to present: Led four engineering teams. Shipped payments."},
		{Field: "experienceTotal", Value: "7 years of work experience in total, computed from the dates of the experience entries"},
		{Field: "experienceLeadership", Value: "7 years in leadership roles, computed from the dates of: CTO at Acme"},
		{Field: "experienceCurrent", Value: "Has worked within the last year (a current role or one that ended in the last 12 months), computed from the dates of: CTO at Acme (current)"},
		{Field: "languages", Value: "English (C1)"},
		{Field: "workAuthorizations", Value: "FR: citizen"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MatchFacts =\n%+v\nwant\n%+v", got, want)
	}
	for _, f := range got {
		if strings.Contains(f.Value, "Private Name") || strings.Contains(f.Value, "Staff Engineer") {
			t.Fatalf("a name or a goal leaked into the matching facts: %+v", f)
		}
	}
	if profile.MatchFacts(nil) != nil {
		t.Fatal("a nil profile has no facts")
	}
}

func derivedOf(t *testing.T, now time.Time, experience ...profile.Experience) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, f := range profile.MatchFactsAt(&profile.Profile{Data: profile.ProfileData{Experience: experience}}, now) {
		if strings.HasPrefix(f.Field, "experience") && f.Field != "experience" {
			got[f.Field] = f.Value
		}
	}
	return got
}

func TestDerivedExperienceFacts(t *testing.T) {
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	got := derivedOf(t, now,
		profile.Experience{Company: "A", Title: "Software Engineer", Start: "2010-03", End: "2014-02"},
		profile.Experience{Company: "B", Title: "Co-founder", Start: "2014-03", End: "2018-02"},
		profile.Experience{Company: "C", Title: "Staff Engineer", Start: "2016", End: "2020"}, // overlaps B
		profile.Experience{Company: "D", Title: "Head of Engineering", Start: "2018-03", Current: true},
		profile.Experience{Company: "E", Title: "Advisor", Start: "unknown", End: "2019"},
	)
	if v := got["experienceTotal"]; !strings.HasPrefix(v, "16 years") {
		t.Errorf("total = %q; overlapping roles must count once and an unreadable date adds nothing", v)
	}
	if v := got["experienceLeadership"]; !strings.HasPrefix(v, "12 years") || !strings.Contains(v, "Co-founder at B") || !strings.Contains(v, "Head of Engineering at D") {
		t.Errorf("leadership = %q", v)
	}
	if v := got["experienceFounder"]; !strings.Contains(v, "Co-founder at B") {
		t.Errorf("founder = %q", v)
	}
}

func TestDerivedExperienceFactsStayAbsentWithoutEvidence(t *testing.T) {
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	if got := derivedOf(t, now); len(got) != 0 {
		t.Errorf("no experience entries, no computed facts: %v", got)
	}
	got := derivedOf(t, now, profile.Experience{Company: "A", Title: "Engineer", Start: "2024-06", End: "2025-01"})
	if len(got) != 0 {
		t.Errorf("under a year and no leading title: %v", got)
	}
	// "Leadership" in a skill or summary is not a leading title.
	got = derivedOf(t, now, profile.Experience{Company: "A", Title: "Engineer", Summary: "Led a team and founded a guild", Start: "2010", End: "2020"})
	if _, ok := got["experienceFounder"]; ok {
		t.Errorf("founder is read from titles only: %v", got)
	}
	if _, ok := got["experienceLeadership"]; ok {
		t.Errorf("leadership is read from titles only: %v", got)
	}
}

func TestDerivedCurrentRoleAndEngineeringYears(t *testing.T) {
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	got := derivedOf(t, now,
		profile.Experience{Company: "A", Title: "Principal Smart Contracts Engineer", Start: "2021-01", End: "2023-12"},
		profile.Experience{Company: "B", Title: "Chief Architect", Start: "2024-01", End: "2026-06"},
	)
	if v := got["experienceEngineering"]; !strings.HasPrefix(v, "5 years") || !strings.Contains(v, "Chief Architect at B") {
		t.Errorf("engineering = %q", v)
	}
	if v := got["experienceCurrent"]; !strings.Contains(v, "Chief Architect at B") || strings.Contains(v, "Engineer at A") {
		t.Errorf("a role that ended five months ago is within the last year, one that ended in 2023 is not: %q", v)
	}
	// Nothing reaches into the last year: the fact is absent, never a "no".
	got = derivedOf(t, now, profile.Experience{Company: "A", Title: "Engineer", Start: "2015-01", End: "2020-01"})
	if _, ok := got["experienceCurrent"]; ok {
		t.Errorf("no role in the last year, so no fact: %v", got)
	}
	for _, f := range []string{"experienceTotal", "experienceLeadership", "experienceFounder", "experienceCurrent", "experienceEngineering"} {
		if !profile.IsComputedField(f) {
			t.Errorf("%s must be a computed field", f)
		}
	}
	if profile.IsComputedField("experience") || profile.IsComputedField("skills") {
		t.Error("a field the owner wrote is not computed")
	}
}
