package vacancyai

import (
	"testing"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
)

// TestConfirmedFactsCrossVacancy verifies the cross-vacancy requirement: pressing ✓ on a
// requirement in vacancy A (which stores a confirmed fact in the profile) makes the same
// requirement ✓ in vacancy B, and ✕ makes it ✕ in vacancy B.
//
// The test does not need a full service or store: it checks the matching logic in isolation
// by building profile facts (as MatchFacts would) from confirmed fact records and verifying
// that checkConfirmedFacts returns the right verdict for a matching requirement in a second
// vacancy.
func TestConfirmedFactsCrossVacancy(t *testing.T) {
	// Simulate the profile state after the owner pressed ✓ on
	// "Financial, payments, or crypto-related systems experience" in vacancy A.
	confirmedText := "Financial, payments, or crypto-related systems experience"
	confirmedKey := profile.NormalizeFactKey(confirmedText)

	// Build the profile facts slice as MatchFacts would.
	facts := []matchcontract.ProfileFact{
		{Field: profile.MatchFieldConfirmedHave, Value: confirmedText},
	}

	// Build the confirmed facts index.
	idx := buildConfirmedFactsIndex(facts)

	// Vacancy B has the same requirement text (experience kind, "?" verdict before the fact).
	reqB := matchcontract.ExtractedRequirement{
		Text: confirmedText,
		Kind: matchcontract.RequirementExperience,
	}

	// checkConfirmedFacts should decide ✓ for vacancy B.
	ev, verdict, reason, ok := checkConfirmedFacts(reqB, idx)
	if !ok {
		t.Fatalf("cross-vacancy ✓: checkConfirmedFacts returned ok=false for %q (key=%q)", confirmedText, confirmedKey)
	}
	if verdict != matchcontract.VerdictMatch {
		t.Errorf("cross-vacancy ✓: verdict = %q, want %q", verdict, matchcontract.VerdictMatch)
	}
	if ev.Field != profile.MatchFieldConfirmedHave {
		t.Errorf("cross-vacancy ✓: evidence field = %q, want %q", ev.Field, profile.MatchFieldConfirmedHave)
	}
	if reason != ReasonConfirmedHave {
		t.Errorf("cross-vacancy ✓: reason = %q, want %q", reason, ReasonConfirmedHave)
	}

	// Now simulate the owner pressing ✕ on "High-growth or fast-moving product environments
	// experience" in vacancy A and check vacancy B.
	absentText := "High-growth or fast-moving product environments experience"
	factsWithAbsent := []matchcontract.ProfileFact{
		{Field: profile.MatchFieldConfirmedHave, Value: confirmedText},
		{Field: profile.MatchFieldConfirmedDoNotHave, Value: absentText},
	}
	idxWithAbsent := buildConfirmedFactsIndex(factsWithAbsent)

	reqBAbsent := matchcontract.ExtractedRequirement{
		Text: absentText,
		Kind: matchcontract.RequirementExperience,
	}
	evA, verdictA, reasonA, okA := checkConfirmedFacts(reqBAbsent, idxWithAbsent)
	if !okA {
		t.Fatalf("cross-vacancy ✕: checkConfirmedFacts returned ok=false for %q", absentText)
	}
	if verdictA != matchcontract.VerdictNo {
		t.Errorf("cross-vacancy ✕: verdict = %q, want %q", verdictA, matchcontract.VerdictNo)
	}
	if evA.Field != profile.MatchFieldConfirmedDoNotHave {
		t.Errorf("cross-vacancy ✕: evidence field = %q, want %q", evA.Field, profile.MatchFieldConfirmedDoNotHave)
	}
	if reasonA != ReasonConfirmedDoNotHave {
		t.Errorf("cross-vacancy ✕: reason = %q, want %q", reasonA, ReasonConfirmedDoNotHave)
	}

	// A skill-kind requirement IS now decided by confirmed facts when the same text was recorded
	// as a confirmed fact (e.g. a phrase like "Coding, debugging, reviewing" that the classifier
	// sent to confirmed facts rather than to the skills list). The guard was removed so that
	// non-vocabulary phrases confirmed via ✓ are recognised cross-vacancy.
	skillReq := matchcontract.ExtractedRequirement{
		Text: confirmedText,
		Kind: matchcontract.RequirementSkill,
	}
	evSkill, verdictSkill, _, okSkill := checkConfirmedFacts(skillReq, idxWithAbsent)
	if !okSkill {
		t.Errorf("skill requirement with matching confirmed fact must be decided; got ok=false")
	}
	if verdictSkill != matchcontract.VerdictMatch {
		t.Errorf("skill requirement confirmed as have: verdict = %q, want %q", verdictSkill, matchcontract.VerdictMatch)
	}
	if evSkill.Field != profile.MatchFieldConfirmedHave {
		t.Errorf("skill requirement confirmed as have: evidence field = %q, want %q", evSkill.Field, profile.MatchFieldConfirmedHave)
	}
}
