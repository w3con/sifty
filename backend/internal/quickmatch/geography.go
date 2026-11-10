package quickmatch

import (
	"sort"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

func evaluateGeography(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts) *groupEval {
	g := &groupEval{group: matchcontract.GroupGeography, applicability: matchcontract.NotApplicable}
	locations := sortedProfileLocations(profile.Locations)
	if len(profile.Locations) > 0 && len(locations) == 0 && len(profile.Authorizations) == 0 {
		g.applicability = matchcontract.ApplicabilityUnknown
		return g
	}
	if len(locations) == 0 && len(profile.Authorizations) == 0 {
		return g
	}
	g.applicability = matchcontract.Applicable

	if len(locations) > 0 {
		g.members = append(g.members, evaluateLocations(locations, facts))
	}
	if m, ok := evaluateAuthorization(profile, facts); ok {
		g.members = append(g.members, m)
	}
	return g
}

func evaluateLocations(profileLocs []matchcontract.ProfileLocation, facts matchcontract.VacancyFacts) member {
	base := member{
		group: matchcontract.GroupGeography, criterion: matchcontract.CriterionGeographyLocation,
		profileField: "geography.preferredLocations", profileValue: formatProfileLocations(profileLocs),
		jobFact: "hiringLocations", applicability: matchcontract.Applicable, weight: 1,
	}
	usable, unreliable := splitLocations(facts.Geography.HiringLocations)
	if len(usable) == 0 && len(unreliable) == 0 {
		base.reason = matchcontract.ReasonLocationMissing
		return base
	}
	if len(usable) == 0 {
		base.reason = matchcontract.ReasonLocationUnusable
		base.section = unreliable[0].Evidence.Section
		base.evidenceField = unreliable[0].Evidence.Field
		base.evidenceQuote = unreliable[0].Evidence.Quote
		base.reliability = unreliable[0].Evidence.Reliability
		return base
	}

	var compatible *locationPair
	var disjoint *locationPair
	anyUnknown := false
	for _, p := range profileLocs {
		for _, j := range usable {
			switch compareLocation(p, j) {
			case locationCompatible:
				if compatible == nil {
					pair := locationPair{profile: p, job: j}
					compatible = &pair
				}
			case locationDisjoint:
				if disjoint == nil {
					pair := locationPair{profile: p, job: j}
					disjoint = &pair
				}
			default:
				anyUnknown = true
			}
		}
	}

	if compatible != nil {
		utility := 1.0
		base.utility = &utility
		base.reason = matchcontract.ReasonLocationCompatible
		fillJobLocation(&base, compatible.job)
		base.profileValue = formatProfileLocation(compatible.profile)
		return base
	}
	if anyUnknown || len(unreliable) > 0 {
		if anyUnknown {
			base.reason = matchcontract.ReasonLocationMappingUnsupported
		} else {
			base.reason = matchcontract.ReasonLocationUnusable
			base.section = unreliable[0].Evidence.Section
			base.evidenceField = unreliable[0].Evidence.Field
			base.evidenceQuote = unreliable[0].Evidence.Quote
			base.reliability = unreliable[0].Evidence.Reliability
		}
		return base
	}
	if disjoint != nil {
		utility := 0.0
		base.utility = &utility
		base.reason = matchcontract.ReasonLocationDisjoint
		fillJobLocation(&base, disjoint.job)
		base.profileValue = formatProfileLocation(disjoint.profile)
		return base
	}
	base.reason = matchcontract.ReasonLocationMappingUnsupported
	return base
}

func evaluateAuthorization(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts) (member, bool) {
	base := member{
		group: matchcontract.GroupGeography, criterion: matchcontract.CriterionGeographyAuthorization,
		profileField: "geography.workAuthorizations", applicability: matchcontract.Applicable, weight: 1,
	}
	jobAuth := facts.Geography.WorkAuthorization
	hasProfileAuth := len(profile.Authorizations) > 0
	if jobAuth == nil {
		if !hasProfileAuth {
			return member{}, false
		}
		base.applicability = matchcontract.NotApplicable
		base.weight = 0
		base.reason = matchcontract.ReasonAuthorizationJobMissing
		base.profileValue = formatAuthorizations(profile.Authorizations)
		return base, true
	}
	if !hasProfileAuth && len(profile.Locations) == 0 {
		return member{}, false
	}

	base.jobFact = "workAuthorization"
	base.jobValue = formatJobAuthorization(jobAuth)
	base.section = jobAuth.Evidence.Section
	base.evidenceField = jobAuth.Evidence.Field
	base.evidenceQuote = jobAuth.Evidence.Quote
	base.reliability = jobAuth.Evidence.Reliability
	base.profileValue = formatAuthorizations(profile.Authorizations)

	if jobAuth.Evidence.Reliability != matchcontract.ReliabilityUsable {
		base.reason = matchcontract.ReasonAuthorizationUnusable
		return base, true
	}
	if strings.TrimSpace(jobAuth.Country) == "" {
		base.reason = matchcontract.ReasonAuthorizationCountryMissing
		return base, true
	}
	if jobAuth.Requirement != matchcontract.AuthorizationRequired && jobAuth.Requirement != matchcontract.AuthorizationNotRequired {
		base.reason = matchcontract.ReasonAuthorizationUnusable
		return base, true
	}

	status := matchingAuthorization(profile.Authorizations, jobAuth.Country)
	if status == "" {
		base.reason = matchcontract.ReasonAuthorizationProfileMissing
		return base, true
	}
	if status == "ambiguous" {
		base.reason = matchcontract.ReasonAuthorizationAmbiguous
		return base, true
	}
	if status == "missing" {
		base.reason = matchcontract.ReasonAuthorizationStatusMissing
		return base, true
	}
	if jobAuth.Requirement == matchcontract.AuthorizationNotRequired {
		utility := 1.0
		base.utility = &utility
		base.reason = matchcontract.ReasonAuthorizationCompatible
		return base, true
	}
	switch status {
	case "eligible":
		utility := 1.0
		base.utility = &utility
		base.reason = matchcontract.ReasonAuthorizationCompatible
	case "need_visa":
		utility := 0.0
		base.utility = &utility
		base.reason = matchcontract.ReasonAuthorizationDisjoint
	default:
		base.reason = matchcontract.ReasonAuthorizationStatusMissing
	}
	return base, true
}

type locationPair struct {
	profile matchcontract.ProfileLocation
	job     matchcontract.LocationFact
}

type locationOutcome int

const (
	locationUnknown locationOutcome = iota
	locationCompatible
	locationDisjoint
)

// compareLocation compares only explicitly normalized country/city fields. A retained label on
// its own is never substring-matched and never mapped to a country or city.
func compareLocation(p matchcontract.ProfileLocation, j matchcontract.LocationFact) locationOutcome {
	pCountry, pCode := normalizeCountry(p.Country)
	jCountry, jCode := normalizeCountry(j.Country)
	pCity, jCity := normalizeCity(p.City), normalizeCity(j.City)

	switch {
	case pCountry != "" && jCountry != "":
		if pCode && jCode {
			if pCountry != jCountry {
				return locationDisjoint
			}
			return compareCity(pCity, jCity)
		}
		if pCountry != jCountry {
			return locationUnknown
		}
		return compareCity(pCity, jCity)
	case pCountry != "" || jCountry != "":
		if pCity != "" && jCity != "" {
			return compareCity(pCity, jCity)
		}
		return locationUnknown
	default:
		if pCity == "" || jCity == "" {
			return locationUnknown
		}
		return compareCity(pCity, jCity)
	}
}

func compareCity(a, b string) locationOutcome {
	if a == "" || b == "" {
		return locationCompatible
	}
	if a == b {
		return locationCompatible
	}
	return locationDisjoint
}

func splitLocations(in []matchcontract.LocationFact) (usable, unreliable []matchcontract.LocationFact) {
	out := append([]matchcontract.LocationFact(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Label != out[j].Label {
			return out[i].Label < out[j].Label
		}
		if out[i].Country != out[j].Country {
			return out[i].Country < out[j].Country
		}
		return out[i].City < out[j].City
	})
	for _, l := range out {
		if l.Evidence.Reliability == matchcontract.ReliabilityUsable {
			usable = append(usable, l)
		} else {
			unreliable = append(unreliable, l)
		}
	}
	return usable, unreliable
}

func sortedProfileLocations(in []matchcontract.ProfileLocation) []matchcontract.ProfileLocation {
	var out []matchcontract.ProfileLocation
	for _, l := range in {
		if strings.TrimSpace(l.Country) != "" || strings.TrimSpace(l.City) != "" {
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Country != out[j].Country {
			return out[i].Country < out[j].Country
		}
		return out[i].City < out[j].City
	})
	return out
}

func fillJobLocation(m *member, j matchcontract.LocationFact) {
	m.jobValue = j.Label
	m.section = j.Evidence.Section
	m.evidenceField = j.Evidence.Field
	m.evidenceQuote = j.Evidence.Quote
	m.reliability = j.Evidence.Reliability
}

func formatProfileLocation(l matchcontract.ProfileLocation) string {
	switch {
	case strings.TrimSpace(l.Country) != "" && strings.TrimSpace(l.City) != "":
		return l.Country + "/" + l.City
	case strings.TrimSpace(l.Country) != "":
		return l.Country
	default:
		return l.City
	}
}

func formatProfileLocations(locs []matchcontract.ProfileLocation) string {
	var out []string
	for _, l := range locs {
		out = append(out, formatProfileLocation(l))
	}
	return joinNonEmpty(out)
}

func formatAuthorizations(in []matchcontract.ProfileAuthorization) string {
	var out []string
	for _, a := range in {
		if strings.TrimSpace(a.Country) == "" {
			continue
		}
		value := strings.ToUpper(strings.TrimSpace(a.Country))
		if strings.TrimSpace(a.Status) != "" {
			value += "=" + a.Status
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return joinNonEmpty(out)
}

func formatJobAuthorization(a *matchcontract.AuthorizationFact) string {
	out := a.Requirement
	if strings.TrimSpace(a.Country) != "" {
		out += " " + strings.ToUpper(strings.TrimSpace(a.Country))
	}
	return out
}

// matchingAuthorization returns the status for an exact explicit country, "missing" when the
// country is present without a status, or "ambiguous" when the saved entries disagree.
func matchingAuthorization(in []matchcontract.ProfileAuthorization, country string) string {
	want, _ := normalizeCountry(country)
	statuses := make(map[string]bool)
	found := false
	for _, a := range in {
		got, _ := normalizeCountry(a.Country)
		if got != want || got == "" {
			continue
		}
		found = true
		status := strings.ToLower(strings.TrimSpace(a.Status))
		if status == "" {
			status = "missing"
		}
		statuses[status] = true
	}
	if !found {
		return ""
	}
	if len(statuses) > 1 {
		return "ambiguous"
	}
	for status := range statuses {
		return status
	}
	return "missing"
}
