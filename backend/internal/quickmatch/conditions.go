package quickmatch

import (
	"sort"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

func evaluateConditions(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, rules matchcontract.Rules) *groupEval {
	g := &groupEval{group: matchcontract.GroupConditions, applicability: matchcontract.NotApplicable}
	hasWorkModes := len(profile.WorkModes) > 0
	hasEngagement := len(profile.EngagementTypes) > 0
	hasHours := profile.Hours != nil && (profile.Hours.Mode != "" || profile.Hours.Weekly != nil ||
		profile.Hours.WeekMin != nil || profile.Hours.WeekMax != nil)
	hasTimezone := len(profile.Timezones) > 0
	if !hasWorkModes && !hasEngagement && !hasHours && !hasTimezone {
		return g
	}
	g.applicability = matchcontract.Applicable

	if hasWorkModes {
		restricted := hasRemoteWithRestriction(facts, rules)
		g.members = append(g.members, evaluateWorkModes(profile.WorkModes, facts, restricted))
	}
	if hasEngagement {
		g.members = append(g.members, evaluateEngagement(profile.EngagementTypes, facts))
	}
	if hasHours {
		g.members = append(g.members, evaluateHours(profile.Hours, facts))
	}
	if hasTimezone {
		g.members = append(g.members, evaluateTimezone(profile.Timezones, facts))
	}
	return g
}

// evaluateWorkModes compares the owner's preferred work modes against the vacancy.
// When restricted is true, the vacancy offers remote work but with a regional restriction;
// in that case a plain "remote matches remote" verdict is suppressed and the reason code
// ReasonWorkModeRemoteRestricted is returned with no utility (geography carries the verdict).
func evaluateWorkModes(profileModes []string, facts matchcontract.VacancyFacts, restricted bool) member {
	base := member{
		group: matchcontract.GroupConditions, criterion: matchcontract.CriterionConditionsWorkMode,
		profileField: "workModes", profileValue: joinNonEmpty(profileModes),
		jobFact: "workModes", applicability: matchcontract.Applicable, weight: 1,
	}
	want := make(map[string]bool)
	for _, mode := range profileModes {
		if normalized := normalizeWorkMode(mode); normalized != "" {
			want[normalized] = true
		}
	}
	if len(want) == 0 {
		base.reason = matchcontract.ReasonWorkModeMissing
		return base
	}
	ordered := append([]matchcontract.WorkModeFact(nil), facts.Conditions.WorkModes...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Mode != ordered[j].Mode {
			return ordered[i].Mode < ordered[j].Mode
		}
		if ordered[i].Evidence.Field != ordered[j].Evidence.Field {
			return ordered[i].Evidence.Field < ordered[j].Evidence.Field
		}
		return ordered[i].Evidence.Quote < ordered[j].Evidence.Quote
	})

	var jobModes []string
	var unreliable *matchcontract.WorkModeFact
	for _, f := range ordered {
		if f.Evidence.Reliability == matchcontract.ReliabilityUsable {
			jobModes = append(jobModes, normalizeWorkMode(f.Mode))
		} else if unreliable == nil {
			fact := f
			unreliable = &fact
		}
	}
	if len(jobModes) == 0 {
		if unreliable != nil {
			fillWorkModeEvidence(&base, *unreliable)
			base.reason = matchcontract.ReasonWorkModeUnusable
		} else {
			base.reason = matchcontract.ReasonWorkModeMissing
		}
		return base
	}
	sort.Strings(jobModes)
	base.jobValue = joinNonEmpty(jobModes)
	if unreliable != nil {
		base.jobValue += ", needs corroboration: " + unreliable.Mode
	}
	for _, m := range ordered {
		if m.Evidence.Reliability == matchcontract.ReliabilityUsable {
			base.section = m.Evidence.Section
			base.evidenceField = m.Evidence.Field
			base.evidenceQuote = m.Evidence.Quote
			base.reliability = m.Evidence.Reliability
			break
		}
	}
	matched := false
	for _, mode := range jobModes {
		if want[mode] {
			matched = true
			break
		}
	}
	if !matched && unreliable != nil {
		fillWorkModeEvidence(&base, *unreliable)
		base.reason = matchcontract.ReasonWorkModeUnusable
		return base
	}
	// When the vacancy has a regional remote restriction and the owner wants remote, suppress
	// the plain match: the geography group carries the actual coverage verdict, so it is not
	// correct to show "✓ remote" here. The work mode fact is still present and applicable; it
	// just has no utility until geography resolves whether the owner is covered.
	if matched && restricted && want[normalizeWorkMode(matchcontract.WorkRemote)] {
		base.reason = matchcontract.ReasonWorkModeRemoteRestricted
		return base // no utility set → midpoint in Fits you
	}
	utility := 0.0
	base.reason = matchcontract.ReasonWorkModeDisjoint
	if matched {
		utility = 1
		base.reason = matchcontract.ReasonWorkModeCompatible
	}
	base.utility = &utility
	return base
}

func fillWorkModeEvidence(m *member, fact matchcontract.WorkModeFact) {
	m.section = fact.Evidence.Section
	m.evidenceField = fact.Evidence.Field
	m.evidenceQuote = fact.Evidence.Quote
	m.reliability = fact.Evidence.Reliability
}

func evaluateEngagement(profileTypes []string, facts matchcontract.VacancyFacts) member {
	base := member{
		group: matchcontract.GroupConditions, criterion: matchcontract.CriterionConditionsEngagement,
		profileField: "engagementTypes", profileValue: joinNonEmpty(profileTypes),
		jobFact: "engagement", applicability: matchcontract.Applicable, weight: 1,
	}
	want := make(map[string]bool)
	for _, t := range profileTypes {
		if normalized := normalizeEngagement(t); normalized != "" {
			want[normalized] = true
		}
	}
	if len(want) == 0 {
		base.reason = matchcontract.ReasonEngagementMissing
		return base
	}
	fact := facts.Conditions.Engagement
	if fact == nil {
		base.reason = matchcontract.ReasonEngagementMissing
		return base
	}
	base.section = fact.Evidence.Section
	base.evidenceField = fact.Evidence.Field
	base.evidenceQuote = fact.Evidence.Quote
	base.reliability = fact.Evidence.Reliability
	base.jobValue = fact.Type
	if fact.Evidence.Reliability != matchcontract.ReliabilityUsable {
		base.reason = matchcontract.ReasonEngagementUnusable
		return base
	}
	jobType := normalizeEngagement(fact.Type)
	if jobType == "" {
		base.reason = matchcontract.ReasonEngagementUnusable
		return base
	}
	utility := 0.0
	base.reason = matchcontract.ReasonEngagementDisjoint
	if want[jobType] {
		utility = 1
		base.reason = matchcontract.ReasonEngagementCompatible
	}
	base.utility = &utility
	return base
}

func evaluateHours(profileHours *matchcontract.ProfileHours, facts matchcontract.VacancyFacts) member {
	base := member{
		group: matchcontract.GroupConditions, criterion: matchcontract.CriterionConditionsHours,
		profileField: "hours", profileValue: formatProfileHours(profileHours),
		jobFact: "hours", applicability: matchcontract.Applicable, weight: 1,
	}
	fact := facts.Conditions.Hours
	if fact == nil {
		base.reason = matchcontract.ReasonHoursMissing
		return base
	}
	base.section = fact.Evidence.Section
	base.evidenceField = fact.Evidence.Field
	base.evidenceQuote = fact.Evidence.Quote
	base.reliability = fact.Evidence.Reliability
	base.jobValue = formatHoursFact(fact)
	if fact.Evidence.Reliability != matchcontract.ReliabilityUsable {
		base.reason = matchcontract.ReasonHoursUnusable
		return base
	}

	profileMode := normalizeWorkMode(profileHours.Mode)
	jobMode := normalizeWorkMode(fact.Mode)
	profileHasRange := hasProfileRange(profileHours)
	jobHasRange := hasFactRange(fact)
	compared := false

	if profileMode != "" {
		if jobMode == "" {
			base.reason = matchcontract.ReasonHoursPartialUnknown
			return base
		}
		compared = true
		if profileMode != jobMode {
			utility := 0.0
			base.utility = &utility
			base.reason = matchcontract.ReasonHoursDisjoint
			return base
		}
	}
	if profileHasRange {
		if !jobHasRange {
			base.reason = matchcontract.ReasonHoursPartialUnknown
			return base
		}
		disjoint, known := hoursRangesDisjoint(profileHours, fact)
		if !known {
			base.reason = matchcontract.ReasonHoursPartialUnknown
			return base
		}
		compared = true
		if disjoint {
			utility := 0.0
			base.utility = &utility
			base.reason = matchcontract.ReasonHoursDisjoint
			return base
		}
	}
	if !compared {
		base.reason = matchcontract.ReasonHoursMissing
		return base
	}
	utility := 1.0
	base.utility = &utility
	base.reason = matchcontract.ReasonHoursCompatible
	return base
}

func evaluateTimezone(profileTimezones []string, facts matchcontract.VacancyFacts) member {
	base := member{
		group: matchcontract.GroupConditions, criterion: matchcontract.CriterionConditionsTimezone,
		profileField: "timezones", profileValue: joinNonEmpty(profileTimezones),
		jobFact: "timezone", applicability: matchcontract.Applicable, weight: 1,
	}
	fact := facts.Conditions.Timezone
	if fact == nil {
		base.reason = matchcontract.ReasonTimezoneMissing
		return base
	}
	base.jobValue = fact.Value
	base.section = fact.Evidence.Section
	base.evidenceField = fact.Evidence.Field
	base.evidenceQuote = fact.Evidence.Quote
	base.reliability = fact.Evidence.Reliability
	base.reason = matchcontract.ReasonTimezoneUnsupported
	return base
}

type hoursBounds struct {
	low, high       float64
	hasLow, hasHigh bool
}

func boundsFrom(weekly, weekMin, weekMax *float64) hoursBounds {
	if weekly != nil {
		return hoursBounds{low: *weekly, high: *weekly, hasLow: true, hasHigh: true}
	}
	b := hoursBounds{}
	if weekMin != nil {
		b.low, b.hasLow = *weekMin, true
	}
	if weekMax != nil {
		b.high, b.hasHigh = *weekMax, true
	}
	return b
}

// hoursRangesDisjoint reports a clear interval separation. Unknown/partial bounds never become
// a disjoint claim.
func hoursRangesDisjoint(profile *matchcontract.ProfileHours, fact *matchcontract.HoursFact) (disjoint bool, known bool) {
	p := boundsFrom(profile.Weekly, profile.WeekMin, profile.WeekMax)
	j := boundsFrom(fact.Weekly, fact.WeekMin, fact.WeekMax)
	if (!p.hasLow && !p.hasHigh) || (!j.hasLow && !j.hasHigh) {
		return false, false
	}
	if p.hasHigh && j.hasLow && j.low > p.high {
		return true, true
	}
	if j.hasHigh && p.hasLow && p.low > j.high {
		return true, true
	}
	return false, true
}

func hasProfileRange(h *matchcontract.ProfileHours) bool {
	return h.Weekly != nil || h.WeekMin != nil || h.WeekMax != nil
}

func hasFactRange(h *matchcontract.HoursFact) bool {
	return h.Weekly != nil || h.WeekMin != nil || h.WeekMax != nil
}

func formatProfileHours(h *matchcontract.ProfileHours) string {
	var parts []string
	if h.Mode != "" {
		parts = append(parts, h.Mode)
	}
	if h.Weekly != nil {
		parts = append(parts, fmtFloat(*h.Weekly)+"h/week")
	} else if h.WeekMin != nil || h.WeekMax != nil {
		parts = append(parts, formatRange(h.WeekMin, h.WeekMax)+"h/week")
	}
	return strings.Join(parts, ", ")
}

func formatHoursFact(h *matchcontract.HoursFact) string {
	var parts []string
	if h.Mode != "" {
		parts = append(parts, h.Mode)
	}
	if h.Weekly != nil {
		parts = append(parts, fmtFloat(*h.Weekly)+"h/week")
	} else if h.WeekMin != nil || h.WeekMax != nil {
		parts = append(parts, formatRange(h.WeekMin, h.WeekMax)+"h/week")
	}
	return strings.Join(parts, ", ")
}

func formatRange(low, high *float64) string {
	switch {
	case low != nil && high != nil:
		return fmtFloat(*low) + "-" + fmtFloat(*high)
	case low != nil:
		return fmtFloat(*low) + "+"
	case high != nil:
		return "up to " + fmtFloat(*high)
	default:
		return ""
	}
}
