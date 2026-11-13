package matchcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// CheckSchema refuses contract data produced for a different schema. Empty is incompatible:
// an old row without a version must not be guessed into the current meaning.
func CheckSchema(version string) error {
	if version != SchemaVersion {
		return fmt.Errorf("incompatible match schema %q; need %q", version, SchemaVersion)
	}
	return nil
}

// HashRetainedSource returns the canonical SHA-256 of the retained fields. Tags, sightings and
// structured list values are sorted first, so ordering alone never changes the source hash.
// Title, company, location and description bytes are retained exactly.
func HashRetainedSource(src RetainedSource) (string, error) {
	c := src
	c.Tags = sortedUnique(c.Tags)
	c.Sources = sortedUnique(c.Sources)
	if src.Structured != nil {
		s := *src.Structured
		s.WorkModes = sortedUnique(s.WorkModes)
		s.EngagementTypes = sortedUnique(s.EngagementTypes)
		s.HiringLocations = sortedUnique(s.HiringLocations)
		c.Structured = &s
	}
	return hashJSON(c)
}

// HashDescription returns the SHA-256 of a retained vacancy description exactly as stored. It is
// the key of a model extraction: the model reads the description and nothing else, so only a
// change of that text may ask it again.
func HashDescription(description string) string {
	sum := sha256.Sum256([]byte(description))
	return hex.EncodeToString(sum[:])
}

// HashProfileSnapshot returns the canonical SHA-256 of the confirmed matching projection.
func HashProfileSnapshot(p ProfileSnapshot) (string, error) {
	c := p
	c.TargetRoles = sortedUnique(c.TargetRoles)
	c.WorkModes = sortedUnique(c.WorkModes)
	c.EngagementTypes = sortedUnique(c.EngagementTypes)
	c.Timezones = sortedUnique(c.Timezones)
	c.Skills = append([]ProfileSkill(nil), p.Skills...)
	sort.SliceStable(c.Skills, func(i, j int) bool {
		if c.Skills[i].Name != c.Skills[j].Name {
			return c.Skills[i].Name < c.Skills[j].Name
		}
		if c.Skills[i].Level != c.Skills[j].Level {
			return c.Skills[i].Level < c.Skills[j].Level
		}
		if c.Skills[i].Context != c.Skills[j].Context {
			return c.Skills[i].Context < c.Skills[j].Context
		}
		if c.Skills[i].SkillID != c.Skills[j].SkillID {
			return c.Skills[i].SkillID < c.Skills[j].SkillID
		}
		return provenanceKey(c.Skills[i].Provenance) < provenanceKey(c.Skills[j].Provenance)
	})
	c.Locations = append([]ProfileLocation(nil), p.Locations...)
	sort.SliceStable(c.Locations, func(i, j int) bool {
		if c.Locations[i].Country != c.Locations[j].Country {
			return c.Locations[i].Country < c.Locations[j].Country
		}
		return c.Locations[i].City < c.Locations[j].City
	})
	c.Authorizations = append([]ProfileAuthorization(nil), p.Authorizations...)
	sort.SliceStable(c.Authorizations, func(i, j int) bool {
		if c.Authorizations[i].Country != c.Authorizations[j].Country {
			return c.Authorizations[i].Country < c.Authorizations[j].Country
		}
		return c.Authorizations[i].Status < c.Authorizations[j].Status
	})
	return hashJSON(c)
}

// HashVacancyFacts returns the canonical SHA-256 of facts with FactsHash excluded and every
// repeated fact list in a stable, content-derived order.
func HashVacancyFacts(f VacancyFacts) (string, error) {
	c := canonicalFacts(f)
	c.FactsHash = ""
	return hashJSON(c)
}

// SetFactsHash validates the facts and stores their canonical hash.
func (f *VacancyFacts) SetFactsHash() error {
	if err := f.Validate(); err != nil {
		return err
	}
	h, err := HashVacancyFacts(*f)
	if err != nil {
		return err
	}
	f.FactsHash = h
	return nil
}

// VerifyFactsHash checks a stored facts value against its own content.
func (f VacancyFacts) VerifyFactsHash() error {
	if f.FactsHash == "" {
		return fmt.Errorf("facts hash is missing")
	}
	h, err := HashVacancyFacts(f)
	if err != nil {
		return err
	}
	if h != f.FactsHash {
		return fmt.Errorf("facts hash mismatch: stored %s, computed %s", f.FactsHash, h)
	}
	return nil
}

// Validate checks the retained source shape without modifying it.
func (s RetainedSource) Validate() error {
	if s.JobID <= 0 {
		return fmt.Errorf("retained source needs a positive job id")
	}
	if strings.TrimSpace(s.Title) == "" {
		return fmt.Errorf("retained source needs a title")
	}
	if strings.TrimSpace(s.DescriptionStatus) == "" {
		return fmt.Errorf("retained source needs a description status")
	}
	if s.Structured != nil {
		if err := s.Structured.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s StructuredFields) validate() error {
	if s.Cash != nil {
		if err := s.Cash.validate(); err != nil {
			return err
		}
	}
	for _, m := range s.WorkModes {
		if !validWorkMode(m) {
			return fmt.Errorf("invalid structured work mode %q", m)
		}
	}
	for _, e := range s.EngagementTypes {
		if !validEngagement(e) {
			return fmt.Errorf("invalid structured engagement %q", e)
		}
	}
	if s.WorkAuthorization != "" && s.WorkAuthorization != AuthorizationRequired && s.WorkAuthorization != AuthorizationNotRequired {
		return fmt.Errorf("invalid structured work authorization %q", s.WorkAuthorization)
	}
	if s.Hours != nil {
		if s.Hours.Mode != "" && !validHoursMode(s.Hours.Mode) {
			return fmt.Errorf("invalid structured hours mode %q", s.Hours.Mode)
		}
		if err := finiteHours("structured hours", s.Hours.Weekly, s.Hours.WeekMin, s.Hours.WeekMax); err != nil {
			return err
		}
	}
	return nil
}

func (c DeclaredCash) validate() error {
	if err := finiteCash("declared cash minimum", c.Min); err != nil {
		return err
	}
	if err := finiteCash("declared cash maximum", c.Max); err != nil {
		return err
	}
	if c.Min != nil && *c.Min < 0 {
		return fmt.Errorf("declared cash minimum must not be negative")
	}
	if c.Max != nil && *c.Max < 0 {
		return fmt.Errorf("declared cash maximum must not be negative")
	}
	if c.Min != nil && c.Max != nil && *c.Min > *c.Max {
		return fmt.Errorf("declared cash range minimum exceeds maximum")
	}
	if c.Period != "" && !validPeriod(c.Period) {
		return fmt.Errorf("invalid cash period %q", c.Period)
	}
	if c.TaxBasis != "" && !validTax(c.TaxBasis) {
		return fmt.Errorf("invalid cash tax basis %q", c.TaxBasis)
	}
	return nil
}

// Validate checks the facts contract. It accepts an empty FactsHash so a producer can compute
// it; persisted facts additionally call VerifyFactsHash.
func (f VacancyFacts) Validate() error {
	if err := CheckSchema(f.SchemaVersion); err != nil {
		return err
	}
	if f.JobID <= 0 {
		return fmt.Errorf("facts need a positive job id")
	}
	if strings.TrimSpace(f.ExtractorVersion) == "" {
		return fmt.Errorf("facts need an extractor version")
	}
	if strings.TrimSpace(f.RawSourceHash) == "" {
		return fmt.Errorf("facts need a raw source hash")
	}
	if f.Role.Title != nil {
		if strings.TrimSpace(f.Role.Title.Value) == "" {
			return fmt.Errorf("role title fact has no value")
		}
		if err := f.Role.Title.Evidence.Validate(); err != nil {
			return fmt.Errorf("role title: %w", err)
		}
	}
	for i, r := range f.Professional.Requirements {
		if strings.TrimSpace(r.Label) == "" {
			return fmt.Errorf("professional requirement %d has no label", i)
		}
		if !r.Importance.Valid() {
			return fmt.Errorf("professional requirement %d has invalid importance %q", i, r.Importance)
		}
		if err := r.Evidence.Validate(); err != nil {
			return fmt.Errorf("professional requirement %d: %w", i, err)
		}
	}
	for i, m := range f.Professional.ContextMentions {
		if strings.TrimSpace(m.Label) == "" {
			return fmt.Errorf("context mention %d has no label", i)
		}
		if err := m.Evidence.Validate(); err != nil {
			return fmt.Errorf("context mention %d: %w", i, err)
		}
	}
	if c := f.Compensation.Cash; c != nil {
		if err := finiteCash("cash fact minimum", c.Min); err != nil {
			return err
		}
		if err := finiteCash("cash fact maximum", c.Max); err != nil {
			return err
		}
		if c.Min != nil && *c.Min < 0 || c.Max != nil && *c.Max < 0 {
			return fmt.Errorf("cash fact must not have negative values")
		}
		if c.Min != nil && c.Max != nil && *c.Min > *c.Max {
			return fmt.Errorf("cash fact range minimum exceeds maximum")
		}
		if c.Period != "" && !validPeriod(c.Period) {
			return fmt.Errorf("invalid cash fact period %q", c.Period)
		}
		if c.TaxBasis != "" && !validTax(c.TaxBasis) {
			return fmt.Errorf("invalid cash fact tax basis %q", c.TaxBasis)
		}
		if c.Basis != "" && c.Basis != CashBasisRangeCeiling && c.Basis != CashBasisCeiling && c.Basis != CashBasisStated {
			return fmt.Errorf("invalid cash fact basis %q", c.Basis)
		}
		if err := c.Evidence.Validate(); err != nil {
			return fmt.Errorf("cash fact: %w", err)
		}
	}
	for i, l := range f.Geography.HiringLocations {
		if strings.TrimSpace(l.Label) == "" {
			return fmt.Errorf("hiring location %d has no label", i)
		}
		if err := l.Evidence.Validate(); err != nil {
			return fmt.Errorf("hiring location %d: %w", i, err)
		}
	}
	if a := f.Geography.WorkAuthorization; a != nil {
		if a.Requirement != AuthorizationRequired && a.Requirement != AuthorizationNotRequired {
			return fmt.Errorf("invalid work authorization requirement %q", a.Requirement)
		}
		if err := a.Evidence.Validate(); err != nil {
			return fmt.Errorf("work authorization: %w", err)
		}
	}
	for i, w := range f.Conditions.WorkModes {
		if !validWorkMode(w.Mode) {
			return fmt.Errorf("condition work mode %d is invalid %q", i, w.Mode)
		}
		if err := w.Evidence.Validate(); err != nil {
			return fmt.Errorf("condition work mode %d: %w", i, err)
		}
	}
	if e := f.Conditions.Engagement; e != nil {
		if !validEngagement(e.Type) {
			return fmt.Errorf("invalid engagement %q", e.Type)
		}
		if err := e.Evidence.Validate(); err != nil {
			return fmt.Errorf("engagement: %w", err)
		}
	}
	if h := f.Conditions.Hours; h != nil {
		if h.Mode != "" && !validHoursMode(h.Mode) {
			return fmt.Errorf("invalid hours mode %q", h.Mode)
		}
		if err := finiteHours("hours fact", h.Weekly, h.WeekMin, h.WeekMax); err != nil {
			return err
		}
		if err := h.Evidence.Validate(); err != nil {
			return fmt.Errorf("hours: %w", err)
		}
	}
	if t := f.Conditions.Timezone; t != nil {
		if strings.TrimSpace(t.Value) == "" {
			return fmt.Errorf("timezone fact has no value")
		}
		if err := t.Evidence.Validate(); err != nil {
			return fmt.Errorf("timezone: %w", err)
		}
	}
	for i, a := range f.Ambiguities {
		if !a.Group.Valid() {
			return fmt.Errorf("ambiguity %d has invalid group %q", i, a.Group)
		}
		if strings.TrimSpace(a.Detail) == "" {
			return fmt.Errorf("ambiguity %d has no detail", i)
		}
		if err := a.Evidence.Validate(); err != nil {
			return fmt.Errorf("ambiguity %d: %w", i, err)
		}
	}
	return nil
}

// Validate refuses weights that do not sum to 1 or omit a group.
func (r Rules) Validate() error {
	if err := CheckSchema(r.SchemaVersion); err != nil {
		return err
	}
	if strings.TrimSpace(r.RulesVersion) == "" {
		return fmt.Errorf("rules need a version")
	}
	if !isFinite(r.RequiredWeight) || !isFinite(r.PreferredWeight) ||
		r.RequiredWeight <= 0 || r.PreferredWeight <= 0 {
		return fmt.Errorf("requirement weights must be finite and positive")
	}
	sum := 0.0
	for _, g := range GoalGroups() {
		w, ok := r.Weights[g]
		if !ok {
			return fmt.Errorf("rules are missing goal group %q", g)
		}
		if !isFinite(w) || w <= 0 {
			return fmt.Errorf("goal group weight %q must be finite and positive", g)
		}
		sum += w
	}
	if sum <= 0 {
		return fmt.Errorf("goal group weights must have a positive sum")
	}
	for g := range r.Weights {
		if !g.IsGoal() {
			return fmt.Errorf("rules contain a weight for %q, which is not a goal group", g)
		}
	}
	if !isFinite(r.UnknownGoalUtility) || r.UnknownGoalUtility < 0 || r.UnknownGoalUtility > 1 {
		return fmt.Errorf("unknown goal utility must be within 0..1")
	}
	if err := r.VerdictUtility.validate(); err != nil {
		return err
	}
	for code, rate := range r.ExchangeRates {
		if !isFinite(rate) || rate <= 0 {
			return fmt.Errorf("exchange rate for %q must be finite and positive", code)
		}
	}
	return nil
}

// Validate checks profile snapshot identity. Demographics are structurally absent.
func (p ProfileSnapshot) Validate() error {
	if strings.TrimSpace(p.ProfileID) == "" {
		return fmt.Errorf("profile snapshot needs an id")
	}
	if p.ProfileRevision < 0 {
		return fmt.Errorf("profile revision must not be negative")
	}
	if c := p.Compensation; c != nil {
		if err := finiteCash("profile cash minimum", c.Minimum); err != nil {
			return err
		}
		if err := finiteCash("profile cash target", c.Target); err != nil {
			return err
		}
		if c.Minimum != nil && *c.Minimum < 0 {
			return fmt.Errorf("profile cash minimum must not be negative")
		}
		if c.Target != nil && *c.Target < 0 {
			return fmt.Errorf("profile cash target must not be negative")
		}
		if c.Period != "" && !validPeriod(c.Period) {
			return fmt.Errorf("invalid profile cash period %q", c.Period)
		}
		if c.TaxBasis != "" && !validTax(c.TaxBasis) {
			return fmt.Errorf("invalid profile cash tax basis %q", c.TaxBasis)
		}
	}
	if h := p.Hours; h != nil {
		if h.Mode != "" && !validHoursMode(h.Mode) {
			return fmt.Errorf("invalid profile hours mode %q", h.Mode)
		}
		if err := finiteHours("profile hours", h.Weekly, h.WeekMin, h.WeekMax); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks the result contract's enum and score invariants.
func (r Result) Validate() error {
	switch r.Status {
	case StatusScored:
		if r.Score == nil || r.RawScore == nil {
			return fmt.Errorf("scored result needs score and raw score")
		}
		if *r.Score < 0 || *r.Score > 100 || !isFinite(*r.RawScore) || *r.RawScore < 0 || *r.RawScore > 1 {
			return fmt.Errorf("scored result is outside its range")
		}
	case StatusUnscored:
		if r.Score != nil || r.RawScore != nil {
			return fmt.Errorf("unscored result must not carry a numeric score")
		}
	case StatusStale:
		// A stale row preserves its previous value until the effects layer recomputes it.
		if r.Score != nil && (*r.Score < 0 || *r.Score > 100) {
			return fmt.Errorf("stale result score is outside its range")
		}
		if r.RawScore != nil && (!isFinite(*r.RawScore) || *r.RawScore < 0 || *r.RawScore > 1) {
			return fmt.Errorf("stale result raw score is outside its range")
		}
	default:
		return fmt.Errorf("invalid result status %q", r.Status)
	}
	if !isFinite(r.Coverage.Weighted) || r.Coverage.Weighted < 0 || r.Coverage.Weighted > 1 {
		return fmt.Errorf("coverage weight is outside 0..1")
	}
	for g, c := range r.Coverage.Groups {
		if !g.Valid() {
			return fmt.Errorf("coverage has invalid group %q", g)
		}
		if !isFinite(c) || c < 0 || c > 1 {
			return fmt.Errorf("coverage for %q is outside 0..1", g)
		}
	}
	for g, a := range r.Applicability {
		if !g.Valid() {
			return fmt.Errorf("applicability has invalid group %q", g)
		}
		switch a {
		case Applicable, NotApplicable, ApplicabilityUnknown:
		default:
			return fmt.Errorf("applicability for %q is invalid %q", g, a)
		}
	}
	for _, c := range r.Contributions {
		if !c.Group.Valid() {
			return fmt.Errorf("contribution has invalid group %q", c.Group)
		}
		if !isFinite(c.Weight) || !isFinite(c.EvidenceFraction) || !isFinite(c.Utility) ||
			!isFinite(c.Weighted) || c.Weight < 0 || c.EvidenceFraction < 0 ||
			c.EvidenceFraction > 1 || c.Utility < 0 || c.Utility > 1 {
			return fmt.Errorf("contribution for %q carries an invalid number", c.Group)
		}
	}
	for _, c := range r.Comparisons {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks one comparison's typed invariants.
func (c Comparison) Validate() error {
	if !c.Group.Valid() {
		return fmt.Errorf("comparison has invalid group %q", c.Group)
	}
	if strings.TrimSpace(string(c.Criterion)) == "" {
		return fmt.Errorf("comparison has no criterion")
	}
	switch c.Applicability {
	case Applicable, NotApplicable, ApplicabilityUnknown:
	default:
		return fmt.Errorf("comparison applicability is invalid %q", c.Applicability)
	}
	if c.Applicability == NotApplicable && c.Comparable {
		return fmt.Errorf("not-applicable comparison cannot be comparable")
	}
	if c.Comparable {
		if c.Utility == nil {
			return fmt.Errorf("comparable criterion %q has no utility", c.Criterion)
		}
		if !isFinite(*c.Utility) || *c.Utility < 0 || *c.Utility > 1 {
			return fmt.Errorf("comparable criterion %q utility is outside 0..1", c.Criterion)
		}
		if c.Weight <= 0 {
			return fmt.Errorf("comparable criterion %q has no weight", c.Criterion)
		}
	} else if c.Utility != nil {
		return fmt.Errorf("unknown criterion %q must not carry a utility", c.Criterion)
	}
	if c.Section != "" && !c.Section.Valid() {
		return fmt.Errorf("comparison has invalid section %q", c.Section)
	}
	if c.Reliability != "" && !c.Reliability.Valid() {
		return fmt.Errorf("comparison has invalid reliability %q", c.Reliability)
	}
	if !isFinite(c.Weight) || c.Weight < 0 {
		return fmt.Errorf("comparison weight must be finite and not negative")
	}
	if !isFinite(c.Contribution) || c.Contribution < 0 {
		return fmt.Errorf("comparison contribution must be finite and not negative")
	}
	if c.ProfileNumber != nil && !isFinite(*c.ProfileNumber) {
		return fmt.Errorf("comparison profile number is not finite")
	}
	if c.JobNumber != nil && !isFinite(*c.JobNumber) {
		return fmt.Errorf("comparison job number is not finite")
	}
	if strings.TrimSpace(c.Reason) == "" {
		return fmt.Errorf("comparison has no reason code")
	}
	return nil
}

func canonicalFacts(f VacancyFacts) VacancyFacts {
	c := f
	reqs := append([]RequirementFact(nil), f.Professional.Requirements...)
	sort.SliceStable(reqs, func(i, j int) bool {
		a, b := reqs[i], reqs[j]
		if sectionOrder(a.Evidence.Section) != sectionOrder(b.Evidence.Section) {
			return sectionOrder(a.Evidence.Section) < sectionOrder(b.Evidence.Section)
		}
		if a.Importance != b.Importance {
			return a.Importance < b.Importance
		}
		if a.Label != b.Label {
			return a.Label < b.Label
		}
		return a.Text < b.Text
	})
	c.Professional.Requirements = reqs
	mentions := append([]ContextMention(nil), f.Professional.ContextMentions...)
	sort.SliceStable(mentions, func(i, j int) bool {
		if sectionOrder(mentions[i].Evidence.Section) != sectionOrder(mentions[j].Evidence.Section) {
			return sectionOrder(mentions[i].Evidence.Section) < sectionOrder(mentions[j].Evidence.Section)
		}
		return mentions[i].Label < mentions[j].Label
	})
	c.Professional.ContextMentions = mentions
	locs := append([]LocationFact(nil), f.Geography.HiringLocations...)
	sort.SliceStable(locs, func(i, j int) bool {
		if locs[i].Label != locs[j].Label {
			return locs[i].Label < locs[j].Label
		}
		if locs[i].Country != locs[j].Country {
			return locs[i].Country < locs[j].Country
		}
		return locs[i].City < locs[j].City
	})
	c.Geography.HiringLocations = locs
	modes := append([]WorkModeFact(nil), f.Conditions.WorkModes...)
	sort.SliceStable(modes, func(i, j int) bool {
		if modes[i].Mode != modes[j].Mode {
			return modes[i].Mode < modes[j].Mode
		}
		return modes[i].Evidence.Quote < modes[j].Evidence.Quote
	})
	c.Conditions.WorkModes = modes
	amb := append([]Ambiguity(nil), f.Ambiguities...)
	sort.SliceStable(amb, func(i, j int) bool {
		if amb[i].Group != amb[j].Group {
			return amb[i].Group < amb[j].Group
		}
		if amb[i].Field != amb[j].Field {
			return amb[i].Field < amb[j].Field
		}
		return amb[i].Detail < amb[j].Detail
	})
	c.Ambiguities = amb
	return c
}

func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func finiteCash(name string, v *float64) error {
	if v != nil && !isFinite(*v) {
		return fmt.Errorf("%s must be finite", name)
	}
	return nil
}

func finiteHours(name string, values ...*float64) error {
	for _, v := range values {
		if v == nil {
			continue
		}
		if !isFinite(*v) || *v < 0 {
			return fmt.Errorf("%s must be finite and not negative", name)
		}
	}
	return nil
}

func hashJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("canonical marshal: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func sortedUnique(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := append([]string(nil), in...)
	sort.Strings(out)
	n := 0
	for _, s := range out {
		if n > 0 && out[n-1] == s {
			continue
		}
		out[n] = s
		n++
	}
	return out[:n]
}

func provenanceKey(p *ProfileProvenance) string {
	if p == nil {
		return ""
	}
	return p.Kind + "\x00" + p.Reference
}

func sectionOrder(s Section) int {
	switch s {
	case SectionRequired:
		return 0
	case SectionPreferred:
		return 1
	case SectionCompanyContext:
		return 2
	default:
		return 3
	}
}

func validWorkMode(m string) bool   { return m == WorkRemote || m == WorkHybrid || m == WorkOffice }
func validEngagement(e string) bool { return e == EngagementEmployment || e == EngagementContract }
func validHoursMode(h string) bool  { return h == HoursFullTime || h == HoursPartTime }
func validPeriod(p string) bool {
	return p == PeriodYear || p == PeriodMonth || p == PeriodDay || p == PeriodHour
}
func validTax(t string) bool { return t == TaxGross || t == TaxNet }
