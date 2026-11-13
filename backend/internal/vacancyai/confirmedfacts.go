package vacancyai

import (
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
)

// ReasonConfirmedHave is the reason stored on a "match" verdict decided by a confirmed fact.
const ReasonConfirmedHave = "Confirmed by you"

// ReasonConfirmedDoNotHave is the reason stored on a "no" verdict decided by a confirmed fact.
const ReasonConfirmedDoNotHave = "You said you do not have this"

// confirmedFactsIndex maps a normalized requirement key to the profile fact evidence for it.
type confirmedFactsIndex struct {
	have      map[string]matchcontract.ProfileEvidence
	doNotHave map[string]matchcontract.ProfileEvidence
}

// buildConfirmedFactsIndex builds a lookup from the profile facts slice.
func buildConfirmedFactsIndex(facts []matchcontract.ProfileFact) confirmedFactsIndex {
	idx := confirmedFactsIndex{
		have:      make(map[string]matchcontract.ProfileEvidence),
		doNotHave: make(map[string]matchcontract.ProfileEvidence),
	}
	for _, f := range facts {
		switch f.Field {
		case profile.MatchFieldConfirmedHave:
			key := profile.NormalizeFactKey(f.Value)
			if key != "" {
				idx.have[key] = matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}
			}
		case profile.MatchFieldConfirmedDoNotHave:
			key := profile.NormalizeFactKey(f.Value)
			if key != "" {
				idx.doNotHave[key] = matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}
			}
		}
	}
	return idx
}

// checkConfirmedFacts decides a requirement whose normalized text matches a confirmed fact. It
// handles all requirement kinds and runs before codeMatch. A skill-kind requirement may also be
// decided here when the owner has explicitly confirmed it as a fact (e.g. a phrase like
// "Coding, debugging, reviewing" that the classifier recorded as a confirmed fact rather than a skill).
// Returns (evidence, verdict, reason, true) when decided, or (zero, "", "", false) when not found.
func checkConfirmedFacts(r matchcontract.ExtractedRequirement, idx confirmedFactsIndex) (matchcontract.ProfileEvidence, matchcontract.Verdict, string, bool) {
	none := matchcontract.ProfileEvidence{}
	key := profile.NormalizeFactKey(r.Text)
	if key == "" {
		return none, "", "", false
	}
	if ev, ok := idx.have[key]; ok {
		return ev, matchcontract.VerdictMatch, ReasonConfirmedHave, true
	}
	if ev, ok := idx.doNotHave[key]; ok {
		return ev, matchcontract.VerdictNo, ReasonConfirmedDoNotHave, true
	}
	return none, "", "", false
}
