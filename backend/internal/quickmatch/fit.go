package quickmatch

import (
	"fmt"
	"sort"

	"github.com/w3con/sifty/internal/matchcontract"
)

// EvaluateFit is the pure two-score evaluator (docs/specs/quick-match.md section 14). Fits you
// comes from the deterministic goal comparison (Evaluate); You fit comes from the requirements a
// model extracted from the vacancy text and the verdicts a second pass gave against the profile.
// It reads only its input: it calls no model, store or clock. A stale record (another source, a
// different requirement list, another profile revision) is refused, never silently reused.
func EvaluateFit(in matchcontract.FitInput) (matchcontract.FitResult, error) {
	if !in.Source.Valid() {
		return matchcontract.FitResult{}, fmt.Errorf("invalid source state %q", in.Source)
	}
	if err := in.Profile.Validate(); err != nil {
		return matchcontract.FitResult{}, fmt.Errorf("profile snapshot: %w", err)
	}
	if err := in.Rules.Validate(); err != nil {
		return matchcontract.FitResult{}, fmt.Errorf("quick match rules: %w", err)
	}

	res := matchcontract.FitResult{
		Source:             in.Source,
		MatchingOff:        in.MatchingOff,
		ConditionConflicts: append([]matchcontract.ConditionConflict{}, in.ConditionConflicts...),
		Lists: matchcontract.FitLists{
			Matches:    []matchcontract.FitLine{},
			Mismatches: []matchcontract.FitLine{},
			Missing:    []matchcontract.FitLine{},
			ToClarify:  []matchcontract.FitLine{},
		},
		Provenance: matchcontract.FitProvenance{
			SchemaVersion:   matchcontract.SchemaVersion,
			ProfileID:       in.Profile.ProfileID,
			ProfileRevision: in.Profile.ProfileRevision,
			RulesVersion:    in.Rules.RulesVersion,
		},
	}
	if identity, err := matchcontract.RulesIdentity(in.Rules); err == nil {
		res.Provenance.RulesIdentity = identity
	}

	if in.Source == matchcontract.SourceUnavailable {
		res.FitsYou = matchcontract.FitsYou{Status: matchcontract.ScoreUnavailable, Groups: []matchcontract.GroupFit{}}
		res.YouFit = matchcontract.YouFit{Status: matchcontract.ScoreUnavailable}
		return finishFit(res)
	}
	if in.Facts == nil {
		return matchcontract.FitResult{}, fmt.Errorf("vacancy facts are required when the source is usable")
	}

	goals, err := Evaluate(in.Profile, *in.Facts, in.Rules)
	if err != nil {
		return matchcontract.FitResult{}, err
	}
	res.Provenance.RawSourceHash = in.Facts.RawSourceHash
	res.Provenance.FactsHash = in.Facts.FactsHash
	res.FitsYou = fitsYou(goals)
	addGoalLines(&res.Lists, goals)

	if err := addYouFit(&res, in); err != nil {
		return matchcontract.FitResult{}, err
	}
	return finishFit(res)
}

func finishFit(res matchcontract.FitResult) (matchcontract.FitResult, error) {
	if err := res.Validate(); err != nil {
		return matchcontract.FitResult{}, fmt.Errorf("evaluator produced an invalid fit result: %w", err)
	}
	return res, nil
}

func fitsYou(goals matchcontract.Result) matchcontract.FitsYou {
	out := matchcontract.FitsYou{Status: matchcontract.ScoreUnscored, Groups: []matchcontract.GroupFit{}}
	if goals.Status == matchcontract.StatusScored {
		out.Status = matchcontract.ScoreScored
		out.Score = goals.Score
		out.RawScore = goals.RawScore
	}
	for _, c := range goals.Contributions {
		out.Groups = append(out.Groups, matchcontract.GroupFit{
			Group: c.Group, Weight: c.Weight, Utility: c.Utility, Known: !c.Assumed,
		})
	}
	return out
}

// listSection is the list a fit line belongs to.
type listSection int

const (
	sectionMatches listSection = iota
	sectionMismatches
	sectionMissing
)

// goalReasonSection places a comparable goal comparison by its stable reason code, because a
// numeric utility alone cannot tell a weak title overlap (a match) from pay just under the
// minimum (a mismatch).
var goalReasonSection = map[string]listSection{
	matchcontract.ReasonRoleTitleAligned:           sectionMatches,
	matchcontract.ReasonCompensationAligned:        sectionMatches,
	matchcontract.ReasonLocationCompatible:         sectionMatches,
	matchcontract.ReasonWorkModeCompatible:         sectionMatches,
	matchcontract.ReasonEngagementCompatible:       sectionMatches,
	matchcontract.ReasonHoursCompatible:            sectionMatches,
	matchcontract.ReasonRemoteRestrictedCovered:    sectionMatches,
	matchcontract.ReasonRemoteUnrestricted:         sectionMatches,
	matchcontract.ReasonRoleTitleNoOverlap:         sectionMismatches,
	matchcontract.ReasonCompensationBelowMin:       sectionMismatches,
	matchcontract.ReasonLocationDisjoint:           sectionMismatches,
	matchcontract.ReasonWorkModeDisjoint:           sectionMismatches,
	matchcontract.ReasonEngagementDisjoint:         sectionMismatches,
	matchcontract.ReasonHoursDisjoint:              sectionMismatches,
	matchcontract.ReasonRemoteRestrictedNotCovered: sectionMismatches,
}

func addGoalLines(lists *matchcontract.FitLists, goals matchcontract.Result) {
	for _, c := range goals.Comparisons {
		if c.Applicability == matchcontract.NotApplicable {
			continue
		}
		line := matchcontract.FitLine{
			Origin:       matchcontract.OriginGoal,
			Group:        c.Group,
			Criterion:    c.Criterion,
			Utility:      c.Utility,
			ProfileField: c.ProfileField,
			ProfileValue: c.ProfileValue,
			JobValue:     c.JobValue,
			Reason:       c.Reason,
			Quote:        c.EvidenceQuote,
		}
		if !c.Comparable {
			// The job title is always read from the posting; an unreadable one is a data defect,
			// not something the owner can clarify.
			if c.Criterion == matchcontract.CriterionRoleTitle {
				continue
			}
			lists.ToClarify = append(lists.ToClarify, line)
			continue
		}
		section, ok := goalReasonSection[c.Reason]
		if !ok {
			// A reason this table does not know yet: a full utility is a match, anything less is not.
			section = sectionMismatches
			if c.Utility != nil && *c.Utility >= 1 {
				section = sectionMatches
			}
		}
		if section == sectionMatches {
			lists.Matches = append(lists.Matches, line)
		} else {
			lists.Mismatches = append(lists.Mismatches, line)
		}
	}
}

func addYouFit(res *matchcontract.FitResult, in matchcontract.FitInput) error {
	res.YouFit = matchcontract.YouFit{Status: matchcontract.ScorePending}
	if in.Extraction == nil {
		return nil
	}
	e := *in.Extraction
	if err := e.Validate(); err != nil {
		return fmt.Errorf("vacancy extraction: %w", err)
	}
	if e.JobID != in.Facts.JobID || e.SourceHash != in.DescriptionHash {
		return fmt.Errorf("vacancy extraction is not for this description")
	}
	hash, err := e.RequirementsHash()
	if err != nil {
		return err
	}
	res.Provenance.ExtractorVersion = e.ExtractorVersion
	res.Provenance.RequirementsHash = hash

	if len(e.Requirements) == 0 {
		res.YouFit = matchcontract.YouFit{Status: matchcontract.ScoreNoRequirements}
		return nil
	}
	if in.Matches == nil {
		if in.MatchingOff {
			res.YouFit = matchcontract.YouFit{Status: matchcontract.ScoreOff}
			addRequirementLines(&res.Lists, e, unclearVerdicts(len(e.Requirements)))
		}
		return nil
	}
	m := *in.Matches
	if err := m.ValidateAgainst(e); err != nil {
		return fmt.Errorf("requirement matches: %w", err)
	}
	if m.ProfileRevision != in.Profile.ProfileRevision {
		return fmt.Errorf("requirement matches are for profile revision %d, not %d", m.ProfileRevision, in.Profile.ProfileRevision)
	}
	res.Provenance.MatcherVersion = m.MatcherVersion

	verdicts := make([]matchcontract.RequirementVerdict, len(e.Requirements))
	for _, v := range m.Verdicts {
		verdicts[v.Index] = v
	}
	var num, den float64
	var counts matchcontract.FitCounts
	for i, req := range e.Requirements {
		v := verdicts[i]
		weight := in.Rules.PreferredWeight
		if req.Level == matchcontract.ImportanceRequired {
			weight = in.Rules.RequiredWeight
		}
		utility, _ := in.Rules.VerdictUtility.Of(v.Verdict)
		num += weight * utility
		den += weight
		switch v.Verdict {
		case matchcontract.VerdictMatch:
			counts.Match++
		case matchcontract.VerdictPartial:
			counts.Partial++
		case matchcontract.VerdictNo:
			counts.No++
		case matchcontract.VerdictUnclear:
			counts.Unclear++
		}
	}
	if in.MatchingOff {
		res.YouFit = matchcontract.YouFit{Status: matchcontract.ScoreOff}
		addRequirementLines(&res.Lists, e, verdicts)
		return nil
	}
	raw := num / den
	score := matchcontract.ScoreFromRaw(raw)
	res.YouFit = matchcontract.YouFit{Status: matchcontract.ScoreScored, Score: &score, RawScore: &raw, Counts: counts}
	addRequirementLines(&res.Lists, e, verdicts)
	return nil
}

func unclearVerdicts(n int) []matchcontract.RequirementVerdict {
	out := make([]matchcontract.RequirementVerdict, n)
	for i := range out {
		out[i] = matchcontract.RequirementVerdict{Index: i, Verdict: matchcontract.VerdictUnclear}
	}
	return out
}

// requirementSection is the one place a verdict is placed in a list. A requirement the profile
// meets, wholly or partly, is a match; the line keeps its verdict so a view can mark a partial
// one. A negative verdict is a known conflict. An unclear verdict is something the profile cannot
// confirm, so it is listed as missing from the profile.
func requirementSection(v matchcontract.Verdict) listSection {
	switch v {
	case matchcontract.VerdictMatch, matchcontract.VerdictPartial:
		return sectionMatches
	case matchcontract.VerdictNo:
		return sectionMismatches
	default:
		return sectionMissing
	}
}

func addRequirementLines(lists *matchcontract.FitLists, e matchcontract.VacancyExtraction, verdicts []matchcontract.RequirementVerdict) {
	order := make([]int, len(e.Requirements))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return levelRank(e.Requirements[order[a]].Level) < levelRank(e.Requirements[order[b]].Level)
	})
	for _, i := range order {
		req, v := e.Requirements[i], verdicts[i]
		line := matchcontract.FitLine{
			Origin:  matchcontract.OriginRequirement,
			Kind:    req.Kind,
			Level:   req.Level,
			Verdict: v.Verdict,
			Subject: req.Text,
			Reason:  v.Reason,
			Quote:   req.Quote,
		}
		if len(req.Alternatives) > 0 {
			line.Alternatives = append([]string(nil), req.Alternatives...)
		}
		if v.Evidence != nil {
			line.ProfileField = v.Evidence.Field
			line.ProfileValue = v.Evidence.Value
		}
		switch requirementSection(v.Verdict) {
		case sectionMatches:
			lists.Matches = append(lists.Matches, line)
		case sectionMismatches:
			lists.Mismatches = append(lists.Mismatches, line)
		default:
			lists.Missing = append(lists.Missing, line)
		}
	}
}

func levelRank(l matchcontract.Importance) int {
	if l == matchcontract.ImportanceRequired {
		return 0
	}
	return 1
}
