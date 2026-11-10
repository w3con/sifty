package profile

import (
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// QuickMatchSnapshot projects the current confirmed, typed profile into the demographic-free
// Quick Match input. It is a read-only projection: it creates no second master, ignores
// demographics/name/photo/legacy arbitrary preferences, and preserves skill identity and
// provenance for explanations. Explicit section markers decide whether a typed collection or
// its retained legacy text is effective; legacy free-text locations are never guessed into
// countries.
func QuickMatchSnapshot(p *Profile) matchcontract.ProfileSnapshot {
	if p == nil {
		return matchcontract.ProfileSnapshot{}
	}
	d := p.Data
	out := matchcontract.ProfileSnapshot{
		ProfileID:       p.ID,
		ProfileRevision: p.Revision,
	}
	for _, role := range d.TargetRoles {
		if role = strings.TrimSpace(role); role != "" {
			out.TargetRoles = append(out.TargetRoles, role)
		}
	}
	for _, s := range d.Skills {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			continue
		}
		skill := matchcontract.ProfileSkill{
			SkillID: s.ID,
			Name:    name,
			Level:   strings.TrimSpace(s.Level),
			Context: s.Context,
		}
		if skill.SkillID == "" {
			skill.SkillID = s.Key
		}
		if s.Provenance != nil {
			skill.Provenance = &matchcontract.ProfileProvenance{
				Kind:      s.Provenance.Kind,
				Reference: s.Provenance.Reference,
			}
		}
		out.Skills = append(out.Skills, skill)
	}
	c := d.Compensation
	// Day rates are deliberately not converted into annual pay and are never scored here.
	if c.CashMin != nil || c.CashTarget != nil || c.Currency != "" || c.GrossNet != "" || c.Period != "" {
		out.Compensation = &matchcontract.ProfileCash{
			Minimum:  c.CashMin,
			Target:   c.CashTarget,
			Currency: strings.ToUpper(strings.TrimSpace(c.Currency)),
			Period:   strings.TrimSpace(c.Period),
			TaxBasis: strings.TrimSpace(c.GrossNet),
		}
	}
	out.WorkModes = cleanModes(d.WorkModes.Modes)
	out.EngagementTypes = cleanEngagement(d.Engagement.Types)
	if h := effectiveHours(d.Hours); h != nil {
		out.Hours = h
	}
	if d.Geography.PreferredLocationsSet || len(d.Geography.PreferredLocations) > 0 {
		for _, loc := range d.Geography.PreferredLocations {
			country := strings.ToUpper(strings.TrimSpace(loc.Country))
			city := strings.TrimSpace(loc.City)
			if country == "" && city == "" {
				continue
			}
			out.Locations = append(out.Locations, matchcontract.ProfileLocation{Country: country, City: city})
		}
	}
	if d.Geography.WorkAuthorizationsSet || len(d.Geography.WorkAuthorizations) > 0 {
		for _, auth := range d.Geography.WorkAuthorizations {
			country := strings.ToUpper(strings.TrimSpace(auth.Country))
			status := strings.TrimSpace(auth.Status)
			if country == "" && status == "" {
				continue
			}
			out.Authorizations = append(out.Authorizations, matchcontract.ProfileAuthorization{Country: country, Status: status})
		}
	}
	if d.Geography.TimezonesSet || len(d.Geography.TimezoneIDs) > 0 {
		for _, tz := range d.Geography.TimezoneIDs {
			if tz = strings.TrimSpace(tz); tz != "" {
				out.Timezones = append(out.Timezones, tz)
			}
		}
	}
	return out
}

func effectiveHours(h Hours) *matchcontract.ProfileHours {
	mode := strings.TrimSpace(h.Mode)
	if mode == "" {
		return nil
	}
	out := &matchcontract.ProfileHours{Mode: mode}
	if mode == HoursPartTime {
		out.Weekly = h.Weekly
		out.WeekMin = h.WeeklyMin
		out.WeekMax = h.WeeklyMax
	}
	return out
}

func cleanModes(in []string) []string {
	var out []string
	for _, m := range in {
		switch m = strings.TrimSpace(m); m {
		case WorkRemote, WorkHybrid, WorkOffice:
			out = append(out, m)
		}
	}
	return out
}

func cleanEngagement(in []string) []string {
	var out []string
	for _, e := range in {
		switch strings.TrimSpace(e) {
		case EngagementEmployment, EngagementEmployee:
			out = append(out, matchcontract.EngagementEmployment)
		case EngagementContract, EngagementContractor:
			out = append(out, matchcontract.EngagementContract)
		}
	}
	return out
}
