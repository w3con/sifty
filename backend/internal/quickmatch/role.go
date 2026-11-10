package quickmatch

import (
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

func evaluateRole(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts) *groupEval {
	g := &groupEval{group: matchcontract.GroupRole, applicability: matchcontract.NotApplicable}
	targets := sortedUniqueStrings(profile.TargetRoles)
	if len(targets) == 0 {
		return g
	}
	g.applicability = matchcontract.Applicable

	title := facts.Role.Title
	if title == nil || strings.TrimSpace(title.Value) == "" {
		g.members = append(g.members, member{
			group: matchcontract.GroupRole, criterion: matchcontract.CriterionRoleTitle,
			profileField: "targetRoles", profileValue: joinNonEmpty(targets),
			jobFact: "title", applicability: matchcontract.Applicable, weight: 1,
			reason: matchcontract.ReasonRoleTitleMissing,
		})
		return g
	}
	ev := title.Evidence
	if ev.Reliability != matchcontract.ReliabilityUsable {
		g.members = append(g.members, member{
			group: matchcontract.GroupRole, criterion: matchcontract.CriterionRoleTitle,
			profileField: "targetRoles", profileValue: joinNonEmpty(targets),
			jobFact: "title", jobValue: title.Value, section: ev.Section,
			evidenceField: ev.Field, evidenceQuote: ev.Quote, reliability: ev.Reliability,
			applicability: matchcontract.Applicable, weight: 1,
			reason: matchcontract.ReasonRoleTitleUnusable,
		})
		return g
	}

	best := -1.0
	bestTarget := ""
	for _, target := range targets {
		f1 := tokenF1(target, title.Value)
		if f1 > best || (f1 == best && (bestTarget == "" || target < bestTarget)) {
			best = f1
			bestTarget = target
		}
	}
	utility := best
	reason := matchcontract.ReasonRoleTitleNoOverlap
	if utility > 0 {
		reason = matchcontract.ReasonRoleTitleAligned
	}
	g.members = append(g.members, member{
		group: matchcontract.GroupRole, criterion: matchcontract.CriterionRoleTitle,
		profileField: "targetRoles", profileValue: bestTarget,
		jobFact: "title", jobValue: title.Value, section: ev.Section,
		evidenceField: ev.Field, evidenceQuote: ev.Quote, reliability: ev.Reliability,
		applicability: matchcontract.Applicable, weight: 1, utility: &utility, reason: reason,
	})
	return g
}
