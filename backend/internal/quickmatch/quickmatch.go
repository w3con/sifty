// Package quickmatch is the pure, deterministic Quick Match evaluator. Its only dependency is
// the shared matchcontract package and the Go standard library: it imports no extractor, store,
// provider, configuration, network, filesystem, clock or corpus package. Every input is passed
// explicitly, so the same profile revision, facts and rules produce the same result anywhere.
package quickmatch

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// member is one typed criterion inside a group. weight is the criterion's internal weight;
// coverage uses every member with a positive weight, while utility uses comparable members only.
type member struct {
	group         matchcontract.Group
	criterion     matchcontract.Criterion
	profileField  string
	profileValue  string
	profileNumber *float64
	skillID       string
	provenance    *matchcontract.ProfileProvenance
	jobFact       string
	jobValue      string
	jobNumber     *float64
	section       matchcontract.Section
	evidenceField string
	evidenceQuote string
	reliability   matchcontract.Reliability
	applicability matchcontract.Applicability
	weight        float64
	utility       *float64
	reason        string
}

type groupEval struct {
	group         matchcontract.Group
	applicability matchcontract.Applicability
	members       []member
	coverage      float64
	utility       *float64
	knownWeight   float64
}

func (g *groupEval) finish() {
	if g.applicability != matchcontract.Applicable {
		g.coverage = 0
		g.utility = nil
		g.knownWeight = 0
		return
	}
	var total, known, weighted float64
	for _, m := range g.members {
		total += m.weight
		if m.utility == nil {
			continue
		}
		known += m.weight
		weighted += m.weight * *m.utility
	}
	g.knownWeight = known
	if total > 0 {
		g.coverage = known / total
	} else {
		g.coverage = 0
	}
	if known > 0 {
		u := weighted / known
		g.utility = &u
	}
}

// Evaluate computes the goals score (Fits you) from one vacancy's prepared facts and the
// confirmed demographic-free profile. Only the role, compensation, geography and conditions
// groups take part; skills and work authorization are requirements, judged by EvaluateFit. A
// goal group the profile cares about but the vacancy gives no comparable fact for counts as the
// configured midpoint instead of being dropped. It returns an error for an incompatible
// contract, a tampered or missing facts hash, or non-finite/invalid rules; it never invents a
// score from invalid input.
func Evaluate(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, rules matchcontract.Rules) (matchcontract.Result, error) {
	if err := profile.Validate(); err != nil {
		return matchcontract.Result{}, fmt.Errorf("profile snapshot: %w", err)
	}
	if err := facts.Validate(); err != nil {
		return matchcontract.Result{}, fmt.Errorf("vacancy facts: %w", err)
	}
	if err := facts.VerifyFactsHash(); err != nil {
		return matchcontract.Result{}, fmt.Errorf("vacancy facts: %w", err)
	}
	if err := rules.Validate(); err != nil {
		return matchcontract.Result{}, fmt.Errorf("quick match rules: %w", err)
	}

	groups := []*groupEval{
		evaluateRole(profile, facts),
		evaluateCompensation(profile, facts),
		evaluateGeography(profile, facts),
		evaluateConditions(profile, facts),
	}
	for _, g := range groups {
		g.finish()
	}

	result := assemble(profile, facts, rules, groups)
	stabilize(&result)
	if err := result.Validate(); err != nil {
		return matchcontract.Result{}, fmt.Errorf("evaluator produced an invalid result: %w", err)
	}
	return result, nil
}

func assemble(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, rules matchcontract.Rules, groups []*groupEval) matchcontract.Result {
	res := matchcontract.Result{
		Coverage: matchcontract.Coverage{
			Groups: make(map[matchcontract.Group]float64, len(groups)),
		},
		Applicability: make(map[matchcontract.Group]matchcontract.Applicability, len(groups)),
		Provenance: matchcontract.Provenance{
			ProfileID:           profile.ProfileID,
			ProfileRevision:     profile.ProfileRevision,
			JobHash:             facts.RawSourceHash,
			FactsHash:           facts.FactsHash,
			RawSourceHash:       facts.RawSourceHash,
			ExtractorVersion:    facts.ExtractorVersion,
			SourcePolicyVersion: facts.SourcePolicyVersion,
			SchemaVersion:       facts.SchemaVersion,
			RulesVersion:        rules.RulesVersion,
		},
	}
	if identity, err := matchcontract.RulesIdentity(rules); err == nil {
		res.Provenance.RulesIdentity = identity
	}

	var coverageDen, coverageNum, scoreDen, scoreNum float64
	for _, g := range groups {
		res.Applicability[g.group] = g.applicability
		res.Coverage.Groups[g.group] = g.coverage
		if g.applicability == matchcontract.Applicable {
			w := rules.Weights[g.group]
			coverageDen += w
			coverageNum += w * g.coverage
			utility, assumed := rules.UnknownGoalUtility, true
			if g.utility != nil && g.coverage > 0 {
				utility, assumed = *g.utility, false
			}
			scoreDen += w
			scoreNum += w * utility
			res.Contributions = append(res.Contributions, matchcontract.Contribution{
				Group:            g.group,
				Weight:           w,
				EvidenceFraction: g.coverage,
				Utility:          utility,
				Weighted:         w * utility,
				Status:           matchcontract.ReliabilityUsable,
				Assumed:          assumed,
			})
		}
		for _, m := range g.members {
			res.Comparisons = append(res.Comparisons, m.comparison(g.knownWeight))
			if m.applicability == matchcontract.NotApplicable {
				continue
			}
			res.Reasons = append(res.Reasons, m.reason)
			if m.utility == nil {
				res.Unknowns = append(res.Unknowns, matchcontract.Unknown{
					Group:   g.group,
					Fact:    firstNonEmpty(m.jobFact, string(m.criterion)),
					Section: m.section,
					Kind:    m.reason,
					Quote:   m.evidenceQuote,
				})
			}
		}
	}
	if coverageDen > 0 {
		res.Coverage.Weighted = coverageNum / coverageDen
	}
	if scoreDen > 0 {
		raw := scoreNum / scoreDen
		score := int(math.Round(raw * 100))
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		res.Status = matchcontract.StatusScored
		res.RawScore = &raw
		res.Score = &score
	} else {
		res.Status = matchcontract.StatusUnscored
		res.Reasons = append(res.Reasons, matchcontract.ReasonNoComparableEvidence)
	}

	for _, a := range facts.Ambiguities {
		if a.Kind == "source_conflict" {
			res.Conflicts = append(res.Conflicts, matchcontract.Conflict{
				Group:  a.Group,
				Fact:   a.Field,
				Kind:   matchcontract.ReasonSourceAmbiguity,
				Detail: a.Detail,
			})
		}
		res.Unknowns = append(res.Unknowns, matchcontract.Unknown{
			Group:   a.Group,
			Fact:    a.Field,
			Section: a.Evidence.Section,
			Kind:    matchcontract.ReasonSourceAmbiguity,
			Quote:   a.Evidence.Quote,
		})
		res.Reasons = append(res.Reasons, matchcontract.ReasonSourceAmbiguity)
	}

	res.Reliability = countReliability(facts)
	return res
}

func (m member) comparison(knownWeight float64) matchcontract.Comparison {
	c := matchcontract.Comparison{
		Group:             m.group,
		Criterion:         m.criterion,
		ProfileField:      m.profileField,
		ProfileValue:      m.profileValue,
		ProfileNumber:     m.profileNumber,
		SkillID:           m.skillID,
		ProfileProvenance: m.provenance,
		JobFact:           m.jobFact,
		JobValue:          m.jobValue,
		JobNumber:         m.jobNumber,
		Section:           m.section,
		EvidenceField:     m.evidenceField,
		EvidenceQuote:     m.evidenceQuote,
		Reliability:       m.reliability,
		Applicability:     m.applicability,
		Comparable:        m.utility != nil,
		Utility:           m.utility,
		Weight:            m.weight,
		Reason:            m.reason,
	}
	if m.utility != nil && knownWeight > 0 {
		c.Contribution = m.weight * *m.utility / knownWeight
	}
	return c
}

func stabilize(res *matchcontract.Result) {
	sort.SliceStable(res.Comparisons, func(i, j int) bool {
		a, b := res.Comparisons[i], res.Comparisons[j]
		if groupRank(a.Group) != groupRank(b.Group) {
			return groupRank(a.Group) < groupRank(b.Group)
		}
		if a.Criterion != b.Criterion {
			return a.Criterion < b.Criterion
		}
		if a.ProfileValue != b.ProfileValue {
			return a.ProfileValue < b.ProfileValue
		}
		if a.JobValue != b.JobValue {
			return a.JobValue < b.JobValue
		}
		return a.Reason < b.Reason
	})
	sort.SliceStable(res.Contributions, func(i, j int) bool {
		return groupRank(res.Contributions[i].Group) < groupRank(res.Contributions[j].Group)
	})
	res.Unknowns = uniqueUnknowns(res.Unknowns)
	res.Conflicts = uniqueConflicts(res.Conflicts)
	res.Reasons = uniqueStrings(res.Reasons)
	// The API and persisted JSON must carry empty lists, never null, so presentation can read
	// lengths and arrays without a nil-slice crash on a fully known or unscored result.
	if res.Contributions == nil {
		res.Contributions = []matchcontract.Contribution{}
	}
	if res.Comparisons == nil {
		res.Comparisons = []matchcontract.Comparison{}
	}
	if res.Unknowns == nil {
		res.Unknowns = []matchcontract.Unknown{}
	}
	if res.Conflicts == nil {
		res.Conflicts = []matchcontract.Conflict{}
	}
	if res.Reasons == nil {
		res.Reasons = []string{}
	}
}

func countReliability(f matchcontract.VacancyFacts) matchcontract.ReliabilityCounts {
	var counts matchcontract.ReliabilityCounts
	add := func(r matchcontract.Reliability) {
		switch r {
		case matchcontract.ReliabilityUsable:
			counts.Usable++
		case matchcontract.ReliabilityNeedsCorroboration:
			counts.NeedsCorroboration++
		case matchcontract.ReliabilityUnusable:
			counts.Unusable++
		}
	}
	if f.Role.Title != nil {
		add(f.Role.Title.Evidence.Reliability)
	}
	for _, r := range f.Professional.Requirements {
		add(r.Evidence.Reliability)
	}
	for _, m := range f.Professional.ContextMentions {
		add(m.Evidence.Reliability)
	}
	if c := f.Compensation.Cash; c != nil {
		add(c.Evidence.Reliability)
	}
	for _, l := range f.Geography.HiringLocations {
		add(l.Evidence.Reliability)
	}
	if a := f.Geography.WorkAuthorization; a != nil {
		add(a.Evidence.Reliability)
	}
	for _, w := range f.Conditions.WorkModes {
		add(w.Evidence.Reliability)
	}
	if e := f.Conditions.Engagement; e != nil {
		add(e.Evidence.Reliability)
	}
	if h := f.Conditions.Hours; h != nil {
		add(h.Evidence.Reliability)
	}
	if t := f.Conditions.Timezone; t != nil {
		add(t.Evidence.Reliability)
	}
	for _, a := range f.Ambiguities {
		add(a.Evidence.Reliability)
	}
	return counts
}

func uniqueStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	n := 0
	for _, s := range out {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if n > 0 && out[n-1] == s {
			continue
		}
		out[n] = s
		n++
	}
	return out[:n]
}

func uniqueUnknowns(in []matchcontract.Unknown) []matchcontract.Unknown {
	out := append([]matchcontract.Unknown(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if groupRank(a.Group) != groupRank(b.Group) {
			return groupRank(a.Group) < groupRank(b.Group)
		}
		if a.Fact != b.Fact {
			return a.Fact < b.Fact
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Quote != b.Quote {
			return a.Quote < b.Quote
		}
		return a.ProfileEdit < b.ProfileEdit
	})
	n := 0
	for _, u := range out {
		if n > 0 && out[n-1] == u {
			continue
		}
		out[n] = u
		n++
	}
	return out[:n]
}

func uniqueConflicts(in []matchcontract.Conflict) []matchcontract.Conflict {
	out := append([]matchcontract.Conflict(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if groupRank(a.Group) != groupRank(b.Group) {
			return groupRank(a.Group) < groupRank(b.Group)
		}
		if a.Fact != b.Fact {
			return a.Fact < b.Fact
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Detail < b.Detail
	})
	n := 0
	for _, c := range out {
		if n > 0 && out[n-1] == c {
			continue
		}
		out[n] = c
		n++
	}
	return out[:n]
}

func groupRank(g matchcontract.Group) int {
	switch g {
	case matchcontract.GroupRole:
		return 0
	case matchcontract.GroupProfessional:
		return 1
	case matchcontract.GroupCompensation:
		return 2
	case matchcontract.GroupGeography:
		return 3
	case matchcontract.GroupConditions:
		return 4
	}
	return 5
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func floatPtr(v float64) *float64 { return &v }
