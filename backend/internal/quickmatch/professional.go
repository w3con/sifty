package quickmatch

import (
	"sort"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

func evaluateProfessional(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, rules matchcontract.Rules) *groupEval {
	g := &groupEval{group: matchcontract.GroupProfessional, applicability: matchcontract.NotApplicable}
	skills := sortedSkills(profile.Skills)
	hasExplicitRequirement := false
	for _, r := range facts.Professional.Requirements {
		if r.Evidence.Section == matchcontract.SectionRequired || r.Evidence.Section == matchcontract.SectionPreferred {
			hasExplicitRequirement = true
		}
	}
	if len(skills) == 0 && !hasExplicitRequirement {
		return g
	}
	g.applicability = matchcontract.Applicable

	reqs := append([]matchcontract.RequirementFact(nil), facts.Professional.Requirements...)
	sort.SliceStable(reqs, func(i, j int) bool {
		a, b := reqs[i], reqs[j]
		if sectionPriority(a.Evidence.Section) != sectionPriority(b.Evidence.Section) {
			return sectionPriority(a.Evidence.Section) < sectionPriority(b.Evidence.Section)
		}
		if importancePriority(a.Importance) != importancePriority(b.Importance) {
			return importancePriority(a.Importance) < importancePriority(b.Importance)
		}
		if reliabilityPriority(a.Evidence.Reliability) != reliabilityPriority(b.Evidence.Reliability) {
			return reliabilityPriority(a.Evidence.Reliability) < reliabilityPriority(b.Evidence.Reliability)
		}
		if a.Label != b.Label {
			return a.Label < b.Label
		}
		return a.Text < b.Text
	})

	seen := make(map[string]bool)
	for _, req := range reqs {
		if req.Evidence.Section != matchcontract.SectionRequired && req.Evidence.Section != matchcontract.SectionPreferred {
			if req.Evidence.Section != matchcontract.SectionCompanyContext {
				g.members = append(g.members, requirementUnknown(req, 0, matchcontract.ReasonSkillSectionUnknown))
			}
			continue
		}
		key := strings.ToLower(strings.Join(strings.Fields(req.Label), " "))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		weight := importanceWeight(req.Importance, rules)
		if !usableRequirement(req) {
			g.members = append(g.members, requirementUnknown(req, weight, matchcontract.ReasonSkillRequirementUnusable))
			continue
		}
		text := req.Label + " " + req.Text
		switch {
		case hasYearsConstraint(text):
			g.members = append(g.members, requirementUnknown(req, weight, matchcontract.ReasonSkillYearsUnsupported))
			continue
		case hasLevelConstraint(text):
			g.members = append(g.members, requirementUnknown(req, weight, matchcontract.ReasonSkillLevelUnsupported))
			continue
		}

		parts, alternative := requirementParts(req.Label)
		if len(parts) == 0 {
			g.members = append(g.members, requirementUnknown(req, weight, matchcontract.ReasonSkillNotEvidenced))
			continue
		}
		matched := make([]matchcontract.ProfileSkill, 0, len(parts))
		for _, part := range parts {
			if skill, ok := matchSkill(part, skills); ok {
				matched = append(matched, skill)
			}
		}
		compound := len(parts) > 1
		fullySupported := false
		switch {
		case !compound:
			fullySupported = len(matched) == 1
		case alternative:
			fullySupported = len(matched) > 0
		default:
			fullySupported = len(matched) == len(parts)
		}
		if !fullySupported {
			m := requirementUnknown(req, weight, matchcontract.ReasonSkillNotEvidenced)
			if compound {
				m.reason = matchcontract.ReasonSkillRequirementCompound
				m.profileValue = joinSkillNames(matched)
			}
			g.members = append(g.members, m)
			continue
		}
		utility := 1.0
		m := requirementMember(req, weight, &utility, matchcontract.ReasonSkillEvidenced)
		m.profileValue = joinSkillNames(matched)
		if len(matched) == 1 {
			m.skillID = matched[0].SkillID
			m.provenance = matched[0].Provenance
			m.profileValue = matched[0].Name
		}
		g.members = append(g.members, m)
	}

	if !hasExplicitRequirement {
		g.members = append(g.members, member{
			group: matchcontract.GroupProfessional, criterion: matchcontract.CriterionProfessionalRequirement,
			profileField: "skills", profileValue: joinSkillNames(skills),
			jobFact: "requirements", applicability: matchcontract.Applicable, weight: 1,
			reason: matchcontract.ReasonSkillRequirementsMissing,
		})
	}
	return g
}

func sectionPriority(s matchcontract.Section) int {
	switch s {
	case matchcontract.SectionRequired:
		return 0
	case matchcontract.SectionPreferred:
		return 1
	case matchcontract.SectionCompanyContext:
		return 2
	default:
		return 3
	}
}

func importancePriority(i matchcontract.Importance) int {
	if i == matchcontract.ImportanceRequired {
		return 0
	}
	return 1
}

func reliabilityPriority(r matchcontract.Reliability) int {
	switch r {
	case matchcontract.ReliabilityUsable:
		return 0
	case matchcontract.ReliabilityNeedsCorroboration:
		return 1
	default:
		return 2
	}
}

func importanceWeight(i matchcontract.Importance, rules matchcontract.Rules) float64 {
	if i == matchcontract.ImportanceRequired {
		return rules.RequiredWeight
	}
	return rules.PreferredWeight
}

func usableRequirement(r matchcontract.RequirementFact) bool {
	return r.Evidence.Reliability == matchcontract.ReliabilityUsable &&
		r.Importance.Valid() &&
		(r.Evidence.Section == matchcontract.SectionRequired || r.Evidence.Section == matchcontract.SectionPreferred)
}

func requirementMember(req matchcontract.RequirementFact, weight float64, utility *float64, reason string) member {
	return member{
		group: matchcontract.GroupProfessional, criterion: matchcontract.CriterionProfessionalRequirement,
		profileField: "skills", jobFact: req.Label, jobValue: req.Text,
		section: req.Evidence.Section, evidenceField: req.Evidence.Field, evidenceQuote: req.Evidence.Quote,
		reliability: req.Evidence.Reliability, applicability: matchcontract.Applicable,
		weight: weight, utility: utility, reason: reason,
	}
}

func requirementUnknown(req matchcontract.RequirementFact, weight float64, reason string) member {
	return requirementMember(req, weight, nil, reason)
}

func sortedSkills(in []matchcontract.ProfileSkill) []matchcontract.ProfileSkill {
	out := append([]matchcontract.ProfileSkill(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := skillKey(out[i].Name), skillKey(out[j].Name)
		if ki != kj {
			return ki < kj
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].SkillID < out[j].SkillID
	})
	return out
}

func matchSkill(name string, skills []matchcontract.ProfileSkill) (matchcontract.ProfileSkill, bool) {
	key := skillKey(name)
	if key == "" {
		return matchcontract.ProfileSkill{}, false
	}
	for _, skill := range skills {
		if skillKey(skill.Name) == key {
			return skill, true
		}
	}
	return matchcontract.ProfileSkill{}, false
}

func joinSkillNames(skills []matchcontract.ProfileSkill) string {
	var names []string
	for _, s := range skills {
		if strings.TrimSpace(s.Name) != "" {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, ", ")
}
