package quickmatch

import (
	"regexp"
	"sort"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// remoteNoRegionStated is the jobValue for a remote vacancy that carries no location label.
const remoteNoRegionStated = "remote, no region stated"

// evaluateGeography compares preferred locations and evaluates remote work restrictions.
// The geography group has two overlapping paths:
//
//  1. Preferred locations: applies when profile.Locations is non-empty. When the vacancy is
//     remoteUnrestricted the location comparison is replaced by a single unrestricted member
//     (utility 1); otherwise the location comparison is added to the group, with an optional
//     restriction member alongside it.
//  2. Remote restriction: applies whenever the vacancy states a regional restriction and the
//     owner has work authorizations or a current location, regardless of preferred locations.
//     The restriction comparison is added in addition to any location member from path 1.
//     When the vacancy is remoteUnrestricted and the owner has any geography data, a single
//     unrestricted member (utility 1) replaces the restriction check.
//
// When only path 2 applies (no preferred locations), the group contributes only the restriction
// verdict, or the unrestricted verdict when the vacancy is remoteUnrestricted. When both apply,
// both members contribute and the group utility is their weighted mean (unless the unrestricted
// shortcut fires, which produces a single member with utility 1).
func evaluateGeography(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, rules matchcontract.Rules) *groupEval {
	g := &groupEval{group: matchcontract.GroupGeography, applicability: matchcontract.NotApplicable}

	// Path 1: preferred locations.
	locations := sortedProfileLocations(profile.Locations)
	if len(profile.Locations) > 0 && len(locations) == 0 {
		g.applicability = matchcontract.ApplicabilityUnknown
		return g
	}
	if len(locations) > 0 {
		// Unrestricted remote matches any location preference; replace the location comparison
		// (which cannot compare a worldwide label against a city or country) with a single match.
		if classifyRemoteRestriction(facts, rules) == remoteUnrestricted {
			g.applicability = matchcontract.Applicable
			g.members = append(g.members, buildUnrestrictedMember(facts))
			return g
		}
		g.applicability = matchcontract.Applicable
		g.members = append(g.members, evaluateLocations(locations, facts))
		// Path 2 in addition: add restriction member when owner has authorizations or current location.
		if m, ok := buildRestrictionMember(profile, facts, rules); ok {
			g.members = append(g.members, m)
		}
		return g
	}

	// Path 2 only: no preferred locations; restriction check is the only geography member.
	restriction := classifyRemoteRestriction(facts, rules)
	if restriction == remoteNoRemote {
		return g // not applicable; vacancy has no remote work mode
	}
	ownerCountries, ownerPresent := ownerCoverageCountries(profile)
	if !ownerPresent {
		return g // owner has no data to compare against; not applicable
	}
	if restriction == remoteUnrestricted {
		// Vacancy is unrestricted remote and owner has geography data: a match.
		g.applicability = matchcontract.Applicable
		g.members = append(g.members, buildUnrestrictedMember(facts))
		return g
	}
	g.applicability = matchcontract.Applicable
	g.members = append(g.members, evaluateRemoteRestriction(facts, rules, ownerCountries))
	return g
}

// buildUnrestrictedMember returns the geography member for a vacancy whose remote work is
// unrestricted (worldwide labels or no location label). The member always has utility 1 and
// reason remote_unrestricted. The job value is taken from the first hiring location label; when
// no labels are present (bare "Remote" with no location), it falls back to remoteNoRegionStated.
func buildUnrestrictedMember(facts matchcontract.VacancyFacts) member {
	utility := 1.0
	m := member{
		group:         matchcontract.GroupGeography,
		criterion:     matchcontract.CriterionGeographyRemoteRestriction,
		profileField:  "geography",
		jobFact:       "hiringLocations",
		applicability: matchcontract.Applicable,
		weight:        1,
		utility:       &utility,
		reason:        matchcontract.ReasonRemoteUnrestricted,
	}
	if len(facts.Geography.HiringLocations) > 0 {
		loc := facts.Geography.HiringLocations[0]
		m.jobValue = loc.Label
		m.section = loc.Evidence.Section
		m.evidenceField = loc.Evidence.Field
		m.evidenceQuote = loc.Evidence.Quote
		m.reliability = loc.Evidence.Reliability
	} else {
		m.jobValue = remoteNoRegionStated
	}
	return m
}

// buildRestrictionMember evaluates the restriction member when a vacancy carries a remote
// restriction and the owner has authorizations or a current location to compare against.
// Returns the member and ok=true only when both conditions hold.
func buildRestrictionMember(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, rules matchcontract.Rules) (member, bool) {
	restriction := classifyRemoteRestriction(facts, rules)
	if restriction == remoteUnrestricted || restriction == remoteNoRemote {
		return member{}, false
	}
	ownerCountries, ownerPresent := ownerCoverageCountries(profile)
	if !ownerPresent {
		return member{}, false
	}
	return evaluateRemoteRestriction(facts, rules, ownerCountries), true
}

// remoteRestriction classifies the vacancy's remote work restriction status.
type remoteRestriction int

const (
	remoteNoRemote      remoteRestriction = iota // no remote work mode in the vacancy
	remoteUnrestricted                           // remote is worldwide / unrestricted
	remoteRestricted                             // remote is restricted to a known country set
	remoteUnknownLabel                           // remote has locations but they can't be fully resolved
)

// classifyRemoteRestriction inspects all hiring location facts (usable and
// needs-corroboration) when the vacancy offers a remote work mode. It returns:
//
//   - remoteNoRemote:     the vacancy has no remote work mode
//   - remoteUnrestricted: the vacancy offers unrestricted / worldwide remote
//   - remoteRestricted:   the vacancy restricts remote to a resolvable country set
//   - remoteUnknownLabel: the vacancy has locations but at least one cannot be resolved
//
// When the vocabulary is empty the function always returns remoteNoRemote, so the
// new code path is inactive until the owner adds regions to the configuration.
func classifyRemoteRestriction(facts matchcontract.VacancyFacts, rules matchcontract.Rules) remoteRestriction {
	if len(rules.Regions) == 0 {
		return remoteNoRemote
	}
	hasRemote := false
	for _, m := range facts.Conditions.WorkModes {
		if m.Mode == matchcontract.WorkRemote {
			hasRemote = true
			break
		}
	}
	if !hasRemote {
		return remoteNoRemote
	}
	// Collect all location labels regardless of reliability; reliability determines the
	// verdict strength, not whether the restriction is considered at all.
	var labels []string
	for _, loc := range facts.Geography.HiringLocations {
		labels = append(labels, loc.Label)
	}
	if len(labels) == 0 {
		return remoteUnrestricted
	}
	// Resolve each label through the vocabulary.
	// A single unresolvable label makes the whole restriction unknown.
	allWorldwide := true
	for _, label := range labels {
		kind, _ := resolveRestrictionLabel(label, rules)
		switch kind {
		case labelWorldwide:
			// worldwide label: does not restrict, skip
		case labelRestricted:
			allWorldwide = false
		case labelUnknown:
			return remoteUnknownLabel
		}
	}
	if allWorldwide {
		return remoteUnrestricted
	}
	return remoteRestricted
}

// labelKind is the result of resolving one location label against the region vocabulary.
type labelKind int

const (
	labelUnknown    labelKind = iota // label not found in the vocabulary
	labelWorldwide                   // label found and maps to empty countries (unrestricted)
	labelRestricted                  // label found and maps to a specific country set
)

// resolveRestrictionLabel looks up one location label in the vocabulary. It tries the full
// label first (protecting vocabulary names that contain "and"), then uses paren-aware
// decomposition and the configured separators to split compound labels. Every part must
// resolve; a single unresolvable part makes the whole label unknown.
func resolveRestrictionLabel(label string, rules matchcontract.Rules) (labelKind, []string) {
	// Step 1: Try the full label. This ensures a vocabulary name that contains "and" or "or"
	// is never accidentally split.
	if kind, countries := lookupRegion(label, rules); kind != labelUnknown {
		return kind, countries
	}

	// Step 2: Build the parts list, using paren-aware decomposition when parens are present.
	var parts []string
	if strings.ContainsRune(label, '(') {
		// Extract content inside each paren group and the concatenated outside text.
		inside, outside := extractParenSegments(label)
		// Split each paren group on separators (handles nested comma-separated content).
		for _, g := range inside {
			parts = append(parts, splitOnSeparators(g, rules.Separators)...)
		}
		// Split the outside-paren text on separators (may include word separators like "or").
		parts = append(parts, splitOnSeparators(outside, rules.Separators)...)
	} else {
		// No parens: split the whole label on the configured separators.
		parts = splitOnSeparators(label, rules.Separators)
	}

	// Filter to non-empty, deduplicating by case-insensitive value.
	var nonEmpty []string
	seen := map[string]bool{}
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			low := strings.ToLower(p)
			if !seen[low] {
				seen[low] = true
				nonEmpty = append(nonEmpty, p)
			}
		}
	}
	if len(nonEmpty) <= 1 {
		return labelUnknown, nil // no progress from splitting
	}

	// All parts must resolve; one unknown makes the whole label unknown.
	return resolveAllParts(nonEmpty, rules)
}

// resolveAllParts resolves every part through the vocabulary and returns the aggregate kind.
// Returns unknown if any single part is unknown.
func resolveAllParts(parts []string, rules matchcontract.Rules) (labelKind, []string) {
	var allCountries []string
	anyRestricted := false
	for _, part := range parts {
		kind, countries := lookupRegion(part, rules)
		if kind == labelUnknown {
			return labelUnknown, nil
		}
		if kind == labelRestricted {
			anyRestricted = true
			allCountries = appendUnique(allCountries, countries...)
		}
	}
	if anyRestricted {
		return labelRestricted, allCountries
	}
	return labelWorldwide, nil
}

// extractParenSegments extracts the content of each top-level paren group and the
// concatenated outside-paren text. For "Remote (US only)" it returns
// inside=["US only"], outside="Remote". For "REMOTE (US only) or NYC/Dallas" it
// returns inside=["US only"], outside="REMOTE  or NYC/Dallas".
func extractParenSegments(label string) (inside []string, outside string) {
	var outsideBuf strings.Builder
	i, n := 0, len(label)
	for i < n {
		if label[i] == '(' {
			j, depth := i+1, 1
			for j < n && depth > 0 {
				switch label[j] {
				case '(':
					depth++
				case ')':
					depth--
				}
				j++
			}
			if depth == 0 {
				content := strings.TrimSpace(label[i+1 : j-1])
				if content != "" {
					inside = append(inside, content)
				}
				i = j
				continue
			}
		}
		outsideBuf.WriteByte(label[i])
		i++
	}
	outside = strings.TrimSpace(outsideBuf.String())
	return inside, outside
}

// splitOnSeparators splits label on the configured separators. Character-only separators
// (e.g. "/", ",", "&", ";") are matched literally; letter-containing separators (e.g.
// "and", "or") are matched at case-insensitive word boundaries so "Florida" is not split
// on "or". Returns a list of trimmed, non-empty tokens.
func splitOnSeparators(label string, separators []string) []string {
	re := buildSeparatorRegex(separators)
	if re == nil {
		if s := strings.TrimSpace(label); s != "" {
			return []string{s}
		}
		return nil
	}
	var out []string
	for _, part := range re.Split(label, -1) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// buildSeparatorRegex compiles a regex from the configured separators. Character-only
// separators are escaped; letter-containing separators are wrapped in \b word boundaries.
// Returns nil when the separator list is empty.
func buildSeparatorRegex(separators []string) *regexp.Regexp {
	if len(separators) == 0 {
		return nil
	}
	hasLetter := regexp.MustCompile(`[a-zA-Z]`)
	var alts []string
	for _, sep := range separators {
		if hasLetter.MatchString(sep) {
			alts = append(alts, `(?i:\b`+regexp.QuoteMeta(sep)+`\b)`)
		} else {
			alts = append(alts, regexp.QuoteMeta(sep))
		}
	}
	return regexp.MustCompile(strings.Join(alts, "|"))
}

// lookupRegion matches one label token against the region vocabulary, trying
// progressively stripped forms (wrapper prefixes and trailing qualifiers from Rules).
func lookupRegion(label string, rules matchcontract.Rules) (labelKind, []string) {
	for _, stripped := range strippedForms(label, rules.WrapperPrefixes, rules.TrailingQualifiers) {
		for _, r := range rules.Regions {
			if strings.EqualFold(r.Name, stripped) {
				return regionKind(r), r.Countries
			}
			for _, a := range r.Aliases {
				if strings.EqualFold(a, stripped) {
					return regionKind(r), r.Countries
				}
			}
		}
	}
	return labelUnknown, nil
}

func regionKind(r matchcontract.RegionEntry) labelKind {
	if len(r.Countries) == 0 {
		return labelWorldwide
	}
	return labelRestricted
}

// strippedForms returns the label and qualifier/prefix-stripped variants to try in lookup order.
// It first applies trailing qualifiers, then wrapper prefixes (longest first), then prefix +
// qualifier combinations. The original label is always the first form tried.
func strippedForms(label string, prefixes, qualifiers []string) []string {
	s := strings.TrimSpace(label)

	// Sort prefixes longest-first so the most specific prefix is tried before shorter ones.
	sortedPfx := make([]string, len(prefixes))
	copy(sortedPfx, prefixes)
	sort.Slice(sortedPfx, func(i, j int) bool { return len(sortedPfx[i]) > len(sortedPfx[j]) })

	seen := map[string]bool{}
	var forms []string
	add := func(f string) {
		f = strings.TrimSpace(f)
		if f != "" && !seen[strings.ToLower(f)] {
			seen[strings.ToLower(f)] = true
			forms = append(forms, f)
		}
	}

	add(s)

	// Apply trailing qualifiers to the original.
	sLow := strings.ToLower(s)
	for _, q := range qualifiers {
		if strings.HasSuffix(sLow, strings.ToLower(q)) {
			add(s[:len(s)-len(q)])
		}
	}

	// Apply wrapper prefixes (longest first), then qualifier-strip the result.
	for _, pfx := range sortedPfx {
		pfxLow := strings.ToLower(pfx)
		if !strings.HasPrefix(sLow, pfxLow) {
			continue
		}
		after := s[len(pfx):]
		// The character immediately after the prefix must be whitespace or end-of-string.
		if len(after) > 0 && after[0] != ' ' && after[0] != '\t' {
			continue
		}
		stripped := strings.TrimSpace(after)
		add(stripped)
		// Also try with trailing qualifiers applied to the prefix-stripped form.
		strippedLow := strings.ToLower(stripped)
		for _, q := range qualifiers {
			if strings.HasSuffix(strippedLow, strings.ToLower(q)) {
				add(stripped[:len(stripped)-len(q)])
			}
		}
	}

	return forms
}

// ownerCoverageCountries returns the ISO country codes from the owner's eligible work
// authorizations, and a boolean indicating whether any owner location data is present
// (authorizations or current location). The two concerns are intentionally separated:
// presence drives group applicability; coverage drives the covered/not-covered/unknown
// verdict. Current location is not a work right and therefore never contributes to
// coverage (spec §6.2).
func ownerCoverageCountries(profile matchcontract.ProfileSnapshot) ([]string, bool) {
	present := len(profile.Authorizations) > 0 || profile.CurrentLocation != nil
	var countries []string
	for _, a := range profile.Authorizations {
		if a.Country != "" && strings.ToLower(strings.TrimSpace(a.Status)) == "eligible" {
			countries = appendUnique(countries, strings.ToUpper(a.Country))
		}
	}
	// Current location is deliberately excluded: it establishes presence but does not
	// grant work eligibility in a country restricted by the vacancy.
	return countries, present
}

// evaluateRemoteRestriction produces the geography.remote_restriction comparison member.
func evaluateRemoteRestriction(facts matchcontract.VacancyFacts, rules matchcontract.Rules, ownerCountries []string) member {
	base := member{
		group:        matchcontract.GroupGeography,
		criterion:    matchcontract.CriterionGeographyRemoteRestriction,
		profileField: "geography.authorizations",
		profileValue: joinNonEmpty(ownerCountries),
		jobFact:      "hiringLocations",
		applicability: matchcontract.Applicable,
		weight:       1,
	}
	// Pick a representative usable location fact for evidence (prefer usable, fall back to any).
	var evidenceLoc *matchcontract.LocationFact
	var restrictionLabels []string
	for i := range facts.Geography.HiringLocations {
		loc := &facts.Geography.HiringLocations[i]
		restrictionLabels = append(restrictionLabels, loc.Label)
		if evidenceLoc == nil || (evidenceLoc.Evidence.Reliability != matchcontract.ReliabilityUsable &&
			loc.Evidence.Reliability == matchcontract.ReliabilityUsable) {
			evidenceLoc = loc
		}
	}
	if evidenceLoc != nil {
		base.jobValue = joinNonEmpty(restrictionLabels)
		base.section = evidenceLoc.Evidence.Section
		base.evidenceField = evidenceLoc.Evidence.Field
		base.evidenceQuote = evidenceLoc.Evidence.Quote
		base.reliability = evidenceLoc.Evidence.Reliability
	}

	// Determine whether any location fact is from a usable (confirmed) source.
	hasUsableLocation := false
	for _, loc := range facts.Geography.HiringLocations {
		if loc.Evidence.Reliability == matchcontract.ReliabilityUsable {
			hasUsableLocation = true
			break
		}
	}
	if !hasUsableLocation {
		// All location facts need corroboration: restriction exists but can't be acted on.
		base.reason = matchcontract.ReasonRemoteRestrictedUnknown
		return base
	}

	// Collect the full restriction country set from usable location facts.
	restrictionCountries, allResolved := collectRestrictionCountries(facts.Geography.HiringLocations, rules)
	if !allResolved {
		base.reason = matchcontract.ReasonRemoteRestrictedUnknown
		return base
	}
	if len(restrictionCountries) == 0 {
		// All usable locations resolved to worldwide — treat as unrestricted.
		base.reason = matchcontract.ReasonRemoteRestrictedUnknown
		return base
	}

	if len(ownerCountries) == 0 {
		// Owner data is present (checked before this call) but no eligible countries yet.
		base.reason = matchcontract.ReasonRemoteRestrictedUnknown
		return base
	}

	// Compare: is any owner country in the restriction set?
	restrictionSet := make(map[string]bool, len(restrictionCountries))
	for _, c := range restrictionCountries {
		restrictionSet[strings.ToUpper(c)] = true
	}
	for _, oc := range ownerCountries {
		if restrictionSet[strings.ToUpper(oc)] {
			utility := 1.0
			base.utility = &utility
			base.reason = matchcontract.ReasonRemoteRestrictedCovered
			return base
		}
	}
	utility := 0.0
	base.utility = &utility
	base.reason = matchcontract.ReasonRemoteRestrictedNotCovered
	return base
}

// collectRestrictionCountries resolves only the usable location facts to country codes.
// Returns the country set and whether all usable labels were resolvable.
func collectRestrictionCountries(locs []matchcontract.LocationFact, rules matchcontract.Rules) ([]string, bool) {
	var countries []string
	for _, loc := range locs {
		if loc.Evidence.Reliability != matchcontract.ReliabilityUsable {
			continue
		}
		kind, cs := resolveRestrictionLabel(loc.Label, rules)
		if kind == labelUnknown {
			return nil, false
		}
		if kind == labelRestricted {
			countries = appendUnique(countries, cs...)
		}
		// labelWorldwide: skip (no restriction)
	}
	return countries, true
}

// appendUnique appends values to dst, skipping duplicates (case-insensitive).
func appendUnique(dst []string, values ...string) []string {
	seen := make(map[string]bool, len(dst))
	for _, d := range dst {
		seen[strings.ToUpper(d)] = true
	}
	for _, v := range values {
		if up := strings.ToUpper(v); !seen[up] {
			seen[up] = true
			dst = append(dst, v)
		}
	}
	return dst
}

// hasRemoteWithRestriction reports whether the vacancy offers a remote work mode and has
// any hiring location facts (of any reliability). Used by the conditions evaluator to
// suppress the plain remote match when restrictions exist.
func hasRemoteWithRestriction(facts matchcontract.VacancyFacts, rules matchcontract.Rules) bool {
	if len(rules.Regions) == 0 {
		return false
	}
	r := classifyRemoteRestriction(facts, rules)
	return r == remoteRestricted || r == remoteUnknownLabel
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
