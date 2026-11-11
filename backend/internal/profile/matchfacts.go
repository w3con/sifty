package profile

import (
	"fmt"
	"strings"
	"time"

	"github.com/w3con/sifty/internal/matchcontract"
)

// Profile fields offered to the requirement-matching pass. They are the evidence names a verdict
// may cite.
const (
	MatchFieldHeadline = "headline"
	MatchFieldSkills   = "skills"
	// MatchFieldSkillsAbsent carries a skill the owner has said they do not have.
	MatchFieldSkillsAbsent   = "skillsAbsent"
	MatchFieldExperience     = "experience"
	MatchFieldLanguages      = "languages"
	MatchFieldAuthorizations = "workAuthorizations"
)

// AbsentSkillPrefix opens the value of a skillsAbsent fact; the skill's name follows it.
const AbsentSkillPrefix = "Does not have: "

// maxExperienceSummaryChars bounds one experience summary so a long CV cannot crowd out the rest.
const maxExperienceSummaryChars = 400

// MatchFacts projects the confirmed profile into the capability facts the requirement-matching
// pass compares vacancy requirements with: headline, skills with levels, experience entries,
// languages and work authorizations. Goals (target roles, pay, work mode), demographics, name and
// photo are not included, and the profile model has no education section to include. The result
// is a read-only projection in a stable order; it creates no second source of truth. Besides the
// profile's own facts it carries three computed from the experience entries (total years, years
// in leading roles, founder), because a model cannot be trusted to do date arithmetic.
func MatchFacts(p *Profile) []matchcontract.ProfileFact { return MatchFactsAt(p, time.Now()) }

// MatchFactsAt is MatchFacts with the clock injected: the years of a current role end at now.
func MatchFactsAt(p *Profile, now time.Time) []matchcontract.ProfileFact {
	if p == nil {
		return nil
	}
	d := p.Data
	var out []matchcontract.ProfileFact
	add := func(field, value string) {
		if value = strings.Join(strings.Fields(value), " "); value != "" {
			out = append(out, matchcontract.ProfileFact{Field: field, Value: value})
		}
	}
	add(MatchFieldHeadline, d.Identity.Headline)
	for _, s := range d.Skills {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			continue
		}
		value := name
		if level := strings.TrimSpace(s.Level); level != "" {
			value += " (" + level + ")"
		}
		add(MatchFieldSkills, value)
	}
	for _, a := range d.AbsentSkills {
		if name := strings.TrimSpace(a.Name); name != "" {
			add(MatchFieldSkillsAbsent, AbsentSkillPrefix+name)
		}
	}
	for _, e := range d.Experience {
		title, company := strings.TrimSpace(e.Title), strings.TrimSpace(e.Company)
		if title == "" && company == "" {
			continue
		}
		value := title
		if company != "" {
			value += " at " + company
		}
		if period := experiencePeriod(e); period != "" {
			value += ", " + period
		}
		if summary := strings.Join(strings.Fields(e.Summary), " "); summary != "" {
			if r := []rune(summary); len(r) > maxExperienceSummaryChars {
				summary = string(r[:maxExperienceSummaryChars])
			}
			value += ": " + summary
		}
		add(MatchFieldExperience, value)
	}
	out = append(out, derivedExperienceFacts(d.Experience, now)...)
	for _, l := range d.Languages {
		name := strings.TrimSpace(l.Name)
		if name == "" {
			continue
		}
		value := name
		if level := strings.TrimSpace(l.Level); level != "" {
			value += " (" + level + ")"
		}
		add(MatchFieldLanguages, value)
	}
	for _, a := range d.Geography.WorkAuthorizations {
		country, status := strings.ToUpper(strings.TrimSpace(a.Country)), strings.TrimSpace(a.Status)
		if country == "" && status == "" {
			continue
		}
		add(MatchFieldAuthorizations, fmt.Sprintf("%s: %s", country, status))
	}
	return out
}

func experiencePeriod(e Experience) string {
	start, end := strings.TrimSpace(e.Start), strings.TrimSpace(e.End)
	if e.Current {
		end = "present"
	}
	switch {
	case start != "" && end != "":
		return start + " to " + end
	case start != "":
		return "from " + start
	case end != "":
		return "until " + end
	}
	return ""
}
