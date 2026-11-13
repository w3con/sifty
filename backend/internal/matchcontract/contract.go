// Package matchcontract is the versioned, domain-only contract shared by Quick Match
// extraction and evaluation. It performs no I/O and imports no store, provider, parser,
// evaluator or presentation package. Both backend/internal/jobextract and the future pure
// backend/internal/quickmatch depend on this package and on nothing of each other's.
package matchcontract

import "fmt"

// SchemaVersion is the on-disk and in-memory shape of the retained input, facts and result
// types. A producer or consumer that does not implement this exact version must refuse the
// data instead of guessing its meaning.
const SchemaVersion = "1"

// ExtractorVersion identifies the deterministic extraction implementation that produced a
// VacancyFacts value. Change it whenever extraction output can change for the same input.
const ExtractorVersion = "jobextract/3"

// RulesVersion identifies the numerical Quick Match policy. The concrete weights and
// sub-weights are configuration data, validated into Rules; this constant only identifies the
// rule set shape. Version 2 is the two-score policy: a group with no comparable fact counts as
// the midpoint, the professional group and work authorization are no longer goals, and You fit
// is computed from model-extracted requirements.
const RulesVersion = "quickmatch.rules/2"

// Group is one of the five fit groups.
type Group string

const (
	GroupRole         Group = "role"
	GroupProfessional Group = "professional"
	GroupCompensation Group = "compensation"
	GroupGeography    Group = "geography"
	GroupConditions   Group = "conditions"
)

// Groups returns the five groups in contract order.
func Groups() []Group {
	return []Group{GroupRole, GroupProfessional, GroupCompensation, GroupGeography, GroupConditions}
}

// GoalGroups returns the groups that take part in Fits you: the four the owner states as goals.
// The professional group describes the vacancy's requirements, which You fit judges instead, so
// it carries no weight.
func GoalGroups() []Group {
	return []Group{GroupRole, GroupCompensation, GroupGeography, GroupConditions}
}

// IsGoal reports whether g is one of the goal groups that carry a weight.
func (g Group) IsGoal() bool {
	switch g {
	case GroupRole, GroupCompensation, GroupGeography, GroupConditions:
		return true
	}
	return false
}

// Valid reports whether g is a known group.
func (g Group) Valid() bool {
	switch g {
	case GroupRole, GroupProfessional, GroupCompensation, GroupGeography, GroupConditions:
		return true
	}
	return false
}

// Section records where a textual fact was stated in the retained vacancy text. It is never
// inferred from prose style: text whose original heading is unavailable stays SectionUnknown.
type Section string

const (
	SectionRequired       Section = "required"
	SectionPreferred      Section = "preferred"
	SectionCompanyContext Section = "company_context"
	SectionUnknown        Section = "unknown"
)

// Valid reports whether s is a known section.
func (s Section) Valid() bool {
	switch s {
	case SectionRequired, SectionPreferred, SectionCompanyContext, SectionUnknown:
		return true
	}
	return false
}

// Reliability is a property of a fact, not a preference weight and not a per-source score.
type Reliability string

const (
	ReliabilityUsable             Reliability = "usable"
	ReliabilityNeedsCorroboration Reliability = "needs_corroboration"
	ReliabilityUnusable           Reliability = "unusable"
)

// Valid reports whether r is a known reliability class.
func (r Reliability) Valid() bool {
	switch r {
	case ReliabilityUsable, ReliabilityNeedsCorroboration, ReliabilityUnusable:
		return true
	}
	return false
}

// Importance is the explicit requirement importance, kept separate from extraction
// confidence. It is never derived from a strength word or a tag.
type Importance string

const (
	ImportanceRequired  Importance = "required"
	ImportancePreferred Importance = "preferred"
)

// Valid reports whether i is a supported requirement importance.
func (i Importance) Valid() bool { return i == ImportanceRequired || i == ImportancePreferred }

// Work modes, engagement types, hours modes and authorization requirements are explicit
// structured values. Empty means unknown and is never replaced by an inferred default.
const (
	WorkRemote = "remote"
	WorkHybrid = "hybrid"
	WorkOffice = "office"

	EngagementEmployment = "employment"
	EngagementContract   = "contract"

	HoursFullTime = "full_time"
	HoursPartTime = "part_time"

	AuthorizationRequired    = "required"
	AuthorizationNotRequired = "not_required"

	PeriodYear            = "year"
	PeriodMonth           = "month"
	PeriodDay             = "day"
	PeriodHour            = "hour"
	TaxGross              = "gross"
	TaxNet                = "net"
	CashBasisRangeCeiling = "range_ceiling"
	CashBasisCeiling      = "ceiling"
	CashBasisStated       = "stated"
)

// Evidence says which retained field or exact quote supports a fact.
type Evidence struct {
	Field       string      `json:"field,omitempty"`
	Quote       string      `json:"quote,omitempty"`
	Section     Section     `json:"section"`
	Reliability Reliability `json:"reliability"`
}

// Validate refuses missing provenance and unknown enum values.
func (e Evidence) Validate() error {
	if !e.Section.Valid() {
		return fmt.Errorf("invalid evidence section %q", e.Section)
	}
	if !e.Reliability.Valid() {
		return fmt.Errorf("invalid evidence reliability %q", e.Reliability)
	}
	if e.Reliability != ReliabilityUnusable && e.Field == "" && e.Quote == "" {
		return fmt.Errorf("evidence needs a field or quote")
	}
	return nil
}

// DeclaredCash is an explicit source salary declaration. It is not a board estimate and not a
// converted or tax-adjusted value. Empty currency/period/tax basis means unknown.
type DeclaredCash struct {
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Currency string   `json:"currency,omitempty"`
	Period   string   `json:"period,omitempty"`
	TaxBasis string   `json:"taxBasis,omitempty"`
}

// HoursRange is an explicit structured hours declaration. Nil bounds mean unknown.
type HoursRange struct {
	Mode    string   `json:"mode,omitempty"`
	Weekly  *float64 `json:"weekly,omitempty"`
	WeekMin *float64 `json:"weekMin,omitempty"`
	WeekMax *float64 `json:"weekMax,omitempty"`
}

// StructuredFields are optional typed source fields supplied by an adapter that actually
// retains them. The current Job record does not carry these fields; adapters leave them nil
// rather than reading them out of free text here.
type StructuredFields struct {
	Cash              *DeclaredCash `json:"cash,omitempty"`
	WorkModes         []string      `json:"workModes,omitempty"`
	EngagementTypes   []string      `json:"engagementTypes,omitempty"`
	Hours             *HoursRange   `json:"hours,omitempty"`
	HiringLocations   []string      `json:"hiringLocations,omitempty"`
	WorkAuthorization string        `json:"workAuthorization,omitempty"`
	Timezone          string        `json:"timezone,omitempty"`
}

// RetainedSource is the authoritative saved Job snapshot used for extraction. Description is
// preserved exactly as stored; it is never re-fetched or reconstructed. Sources are the saved
// sightings that produced the job, used only to select source-field policy.
type RetainedSource struct {
	JobID             int64             `json:"jobId"`
	Title             string            `json:"title"`
	Company           string            `json:"company"`
	Location          string            `json:"location,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	Description       string            `json:"description"`
	DescriptionStatus string            `json:"descriptionStatus"`
	Sources           []string          `json:"sources,omitempty"`
	Structured        *StructuredFields `json:"structured,omitempty"`
}

// TitleFact is the actual vacancy title with its provenance.
type TitleFact struct {
	Value    string   `json:"value"`
	Evidence Evidence `json:"evidence"`
}

// RoleFacts carries the vacancy-side role evidence. Target titles live in ProfileSnapshot.
type RoleFacts struct {
	Title *TitleFact `json:"title,omitempty"`
}

// RequirementFact is one explicit required or preferred item. Text preserves the retained
// wording; Label is the deterministic item label used for comparison and display.
type RequirementFact struct {
	Label      string     `json:"label"`
	Text       string     `json:"text,omitempty"`
	Importance Importance `json:"importance"`
	Evidence   Evidence   `json:"evidence"`
}

// ContextMention is a company-stack or other contextual mention. It is retained for
// explanation and is never treated as a requirement.
type ContextMention struct {
	Label    string   `json:"label"`
	Evidence Evidence `json:"evidence"`
}

// ProfessionalFacts contains only explicit vacancy-side professional evidence.
type ProfessionalFacts struct {
	Requirements    []RequirementFact `json:"requirements,omitempty"`
	ContextMentions []ContextMention  `json:"contextMentions,omitempty"`
}

// CashFact is an explicit employer cash declaration with provenance. A range lower bound is
// retained but is not a guaranteed offer; Max is the potential ceiling.
type CashFact struct {
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Currency string   `json:"currency,omitempty"`
	Period   string   `json:"period,omitempty"`
	TaxBasis string   `json:"taxBasis,omitempty"`
	Basis    string   `json:"basis,omitempty"`
	Evidence Evidence `json:"evidence"`
}

// CompensationFacts carries at most one explicit declared cash band.
type CompensationFacts struct {
	Cash *CashFact `json:"cash,omitempty"`
}

// LocationFact is one explicit hiring-location statement. Country/City stay empty when the
// retained wording is not an exact supported mapping; Label is always preserved.
type LocationFact struct {
	Label    string   `json:"label"`
	Country  string   `json:"country,omitempty"`
	City     string   `json:"city,omitempty"`
	Evidence Evidence `json:"evidence"`
}

// AuthorizationFact is an explicit work-authorization requirement statement.
type AuthorizationFact struct {
	Country     string   `json:"country,omitempty"`
	Requirement string   `json:"requirement,omitempty"` // required | not_required
	Evidence    Evidence `json:"evidence"`
}

// GeographyFacts separates permitted hiring locations from work authorization. Remote work
// mode is a condition, not proof that every location is permitted.
type GeographyFacts struct {
	HiringLocations   []LocationFact     `json:"hiringLocations,omitempty"`
	WorkAuthorization *AuthorizationFact `json:"workAuthorization,omitempty"`
}

// WorkModeFact is an explicit remote/hybrid/office statement.
type WorkModeFact struct {
	Mode     string   `json:"mode"`
	Evidence Evidence `json:"evidence"`
}

// EngagementFact is an explicit employment/contract statement.
type EngagementFact struct {
	Type     string   `json:"type"`
	Evidence Evidence `json:"evidence"`
}

// HoursFact is an explicit full-time/part-time statement with optional structured hours.
type HoursFact struct {
	Mode     string   `json:"mode,omitempty"`
	Weekly   *float64 `json:"weekly,omitempty"`
	WeekMin  *float64 `json:"weekMin,omitempty"`
	WeekMax  *float64 `json:"weekMax,omitempty"`
	Evidence Evidence `json:"evidence"`
}

// TimezoneFact is an explicit timezone or overlap statement, retained verbatim.
type TimezoneFact struct {
	Value    string   `json:"value"`
	Evidence Evidence `json:"evidence"`
}

// ConditionsFacts contains non-pay working conditions only.
type ConditionsFacts struct {
	WorkModes  []WorkModeFact  `json:"workModes,omitempty"`
	Engagement *EngagementFact `json:"engagement,omitempty"`
	Hours      *HoursFact      `json:"hours,omitempty"`
	Timezone   *TimezoneFact   `json:"timezone,omitempty"`
}

// Ambiguity records a retained ambiguity without turning it into a guessed fact.
type Ambiguity struct {
	Group    Group    `json:"group"`
	Field    string   `json:"field"`
	Kind     string   `json:"kind,omitempty"`
	Detail   string   `json:"detail"`
	Evidence Evidence `json:"evidence"`
}

// VacancyFacts is the complete persisted extraction output. It is produced only from a
// RetainedSource and validated before storage. FactsHash covers everything except itself.
type VacancyFacts struct {
	SchemaVersion       string            `json:"schemaVersion"`
	ExtractorVersion    string            `json:"extractorVersion"`
	SourcePolicyVersion string            `json:"sourcePolicyVersion"`
	JobID               int64             `json:"jobId"`
	RawSourceHash       string            `json:"rawSourceHash"`
	FactsHash           string            `json:"factsHash"`
	Role                RoleFacts         `json:"role"`
	Professional        ProfessionalFacts `json:"professional"`
	Compensation        CompensationFacts `json:"compensation"`
	Geography           GeographyFacts    `json:"geography"`
	Conditions          ConditionsFacts   `json:"conditions"`
	Ambiguities         []Ambiguity       `json:"ambiguities,omitempty"`
}

// SourcePolicyRule is the configurable usability of one source's structured fields.
type SourcePolicyRule struct {
	Location Reliability `json:"location,omitempty"`
}

// SourcePolicy is the injected extractor policy. It carries no preference coefficient and is
// not consulted by the evaluator. A board location field defaults to needs-corroboration: a
// city or office address is not by itself proof of a hiring restriction, and remote is not
// worldwide. Explicit job text can corroborate or contradict it.
type SourcePolicy struct {
	Version string                      `json:"version"`
	Sources map[string]SourcePolicyRule `json:"sources,omitempty"`
}

// LocationReliability returns the declared usability of a source's location field. When a
// role was seen on several sources, the most conservative declared rule applies.
func (p SourcePolicy) LocationReliability(sources []string) Reliability {
	if len(sources) == 0 {
		return ReliabilityNeedsCorroboration
	}
	out := ReliabilityUsable
	for _, name := range sources {
		reliability := ReliabilityNeedsCorroboration
		if rule, ok := p.Sources[name]; ok && rule.Location != "" {
			if !rule.Location.Valid() {
				reliability = ReliabilityUnusable
			} else {
				reliability = rule.Location
			}
		}
		out = moreConservative(out, reliability)
	}
	return out
}

func moreConservative(a, b Reliability) Reliability {
	rank := func(r Reliability) int {
		switch r {
		case ReliabilityUsable:
			return 0
		case ReliabilityNeedsCorroboration:
			return 1
		default:
			return 2
		}
	}
	if rank(b) > rank(a) {
		return b
	}
	return a
}

// ProfileSnapshot is the explicit, demographic-free matching projection of the confirmed
// profile. It carries no name, photo, nationality, age, gender, race or ethnicity fields at
// all, so those values cannot reach evaluation by omission.
type ProfileSnapshot struct {
	ProfileID       string                 `json:"profileId"`
	ProfileRevision int                    `json:"profileRevision"`
	TargetRoles     []string               `json:"targetRoles,omitempty"`
	Skills          []ProfileSkill         `json:"skills,omitempty"`
	Compensation    *ProfileCash           `json:"compensation,omitempty"`
	WorkModes       []string               `json:"workModes,omitempty"`
	EngagementTypes []string               `json:"engagementTypes,omitempty"`
	Hours           *ProfileHours          `json:"hours,omitempty"`
	Locations       []ProfileLocation      `json:"locations,omitempty"`
	Authorizations  []ProfileAuthorization `json:"authorizations,omitempty"`
	Timezones       []string               `json:"timezones,omitempty"`
	// CurrentLocation is the owner's current residence, used to compare remote work
	// restrictions when no preferred locations are set. It carries no demographics.
	CurrentLocation *ProfileLocation `json:"currentLocation,omitempty"`
}

// ProfileProvenance identifies the saved source of a confirmed profile fact. It is
// demographic-free and descriptive only: the evaluator never changes a comparison because a
// fact came from a document or was entered manually.
type ProfileProvenance struct {
	Kind      string `json:"kind,omitempty"`
	Reference string `json:"reference,omitempty"`
}

// ProfileSkill is a confirmed saved skill. Empty level means unknown, not absent. SkillID and
// Provenance retain the authoritative saved identity so a result can point back to it.
type ProfileSkill struct {
	SkillID    string             `json:"skillId,omitempty"`
	Name       string             `json:"name"`
	Level      string             `json:"level,omitempty"`
	Context    string             `json:"context,omitempty"`
	Provenance *ProfileProvenance `json:"provenance,omitempty"`
}

// ProfileCash is the confirmed minimum acceptable cash. Target is retained for context only;
// evaluation must use Minimum. Empty currency/period/tax basis means incomparable.
type ProfileCash struct {
	Minimum  *float64 `json:"minimum,omitempty"`
	Target   *float64 `json:"target,omitempty"`
	Currency string   `json:"currency,omitempty"`
	Period   string   `json:"period,omitempty"`
	TaxBasis string   `json:"taxBasis,omitempty"`
}

// ProfileHours is the explicit working-time preference.
type ProfileHours struct {
	Mode    string   `json:"mode,omitempty"`
	Weekly  *float64 `json:"weekly,omitempty"`
	WeekMin *float64 `json:"weekMin,omitempty"`
	WeekMax *float64 `json:"weekMax,omitempty"`
}

// ProfileLocation is one explicit preferred location.
type ProfileLocation struct {
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
}

// ProfileAuthorization is one explicit work-authorization answer for a country.
type ProfileAuthorization struct {
	Country string `json:"country,omitempty"`
	Status  string `json:"status,omitempty"`
}

// RegionEntry maps one named region (and its aliases) to a set of ISO 3166-1 alpha-2 country
// codes. An empty Countries list means worldwide / unrestricted. The vocabulary is configuration
// data: no country list may be hard-coded in evaluator logic.
type RegionEntry struct {
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases,omitempty"`
	Countries []string `json:"countries"`
}

// Rules is the validated numerical policy supplied to the evaluator. Weights hold exactly the four
// goal groups; Fits you divides by the sum of the weights of the groups it uses, so the weights
// need not add up to 1. RequiredWeight and PreferredWeight weigh requirements in You fit.
// Regions is the optional region vocabulary for restricted-remote detection; when empty the
// restriction check is skipped and remote works as before. RestrictionPhrases is the wording that
// makes a model-read location usable for that check (docs/specs/quick-match.md 14.5); when empty
// every model-read location needs corroboration. Both enter the rules identity.
type Rules struct {
	SchemaVersion      string            `json:"schemaVersion"`
	RulesVersion       string            `json:"rulesVersion"`
	Weights            map[Group]float64 `json:"weights"`
	RequiredWeight     float64           `json:"requiredWeight"`
	PreferredWeight    float64           `json:"preferredWeight"`
	UnknownGoalUtility float64           `json:"unknownGoalUtility"`
	VerdictUtility     VerdictUtilities  `json:"verdictUtility"`
	// ExchangeRates maps ISO 4217 upper-case currency codes to their value relative to EUR (the
	// base; "EUR" maps to 1.0). The compensation evaluator uses it to convert a vacancy's stated
	// currency into the profile currency before comparison. An empty map means cross-currency
	// comparisons are incomparable rather than refused.
	ExchangeRates      map[string]float64 `json:"exchangeRates,omitempty"`
	Regions            []RegionEntry      `json:"regions,omitempty"`
	RestrictionPhrases []string           `json:"restrictionPhrases,omitempty"`
	// WrapperPrefixes are leading words stripped from a token before region lookup.
	WrapperPrefixes []string `json:"wrapperPrefixes,omitempty"`
	// TrailingQualifiers are suffixes stripped from a token before region lookup.
	TrailingQualifiers []string `json:"trailingQualifiers,omitempty"`
	// Separators split a compound label into individual region tokens. Character separators
	// match literally; letter separators match at word boundaries (case-insensitive).
	Separators []string `json:"separators,omitempty"`
}

// ResultStatus distinguishes a numeric score, no usable evidence, and a stale saved result.
type ResultStatus string

const (
	StatusScored   ResultStatus = "scored"
	StatusUnscored ResultStatus = "unscored"
	StatusStale    ResultStatus = "stale"
)

// Applicability is decided from saved profile goals and vacancy facts before scoring.
type Applicability string

const (
	Applicable           Applicability = "applicable"
	NotApplicable        Applicability = "not_applicable"
	ApplicabilityUnknown Applicability = "unknown"
)

// Coverage separates evidence coverage from the preference score.
type Coverage struct {
	Weighted float64           `json:"weighted"`
	Groups   map[Group]float64 `json:"groups"`
}

// ReliabilityCounts reports facts by reliability class.
type ReliabilityCounts struct {
	Usable             int `json:"usable"`
	NeedsCorroboration int `json:"needsCorroboration"`
	Unusable           int `json:"unusable"`
}

// Criterion identifies one stable, typed comparison. Views render the code, the carried values
// and the reason; they never recompute a utility or invent an explanation.
type Criterion string

const (
	CriterionRoleTitle                  Criterion = "role.title"
	CriterionProfessionalRequirement    Criterion = "professional.requirement"
	CriterionCompensationCash           Criterion = "compensation.cash"
	CriterionGeographyLocation          Criterion = "geography.location"
	CriterionGeographyAuthorization     Criterion = "geography.authorization"
	CriterionGeographyRemoteRestriction Criterion = "geography.remote_restriction"
	CriterionConditionsWorkMode         Criterion = "conditions.work_mode"
	CriterionConditionsEngagement       Criterion = "conditions.engagement"
	CriterionConditionsHours            Criterion = "conditions.hours"
	CriterionConditionsTimezone         Criterion = "conditions.timezone"
)

// Stable reason codes. They drive deterministic UI templates and are never free text.
const (
	ReasonNoComparableEvidence = "no_comparable_evidence"
	ReasonSourceAmbiguity      = "source_ambiguity"

	ReasonRoleTitleAligned   = "role_title_aligned"
	ReasonRoleTitleNoOverlap = "role_title_no_overlap"
	ReasonRoleTitleMissing   = "role_title_missing"
	ReasonRoleTitleUnusable  = "role_title_unusable"

	ReasonSkillEvidenced           = "skill_evidenced"
	ReasonSkillNotEvidenced        = "skill_not_evidenced"
	ReasonSkillRequirementCompound = "skill_requirement_compound"
	ReasonSkillYearsUnsupported    = "skill_years_unsupported"
	ReasonSkillLevelUnsupported    = "skill_level_unsupported"
	ReasonSkillRequirementUnusable = "skill_requirement_unusable"
	ReasonSkillSectionUnknown      = "skill_requirement_section_unknown"
	ReasonSkillRequirementsMissing = "skill_requirements_missing"

	ReasonCompensationAligned      = "compensation_aligned"
	ReasonCompensationBelowMin     = "compensation_below_minimum"
	ReasonCompensationRangeCovers  = "compensation_range_covers_minimum" // lower < min ≤ upper; not counted
	ReasonCompensationMissing      = "compensation_job_missing"
	ReasonCompensationIncomparable = "compensation_incomparable"
	ReasonCompensationUnusable     = "compensation_fact_unusable"

	ReasonLocationCompatible         = "location_compatible"
	ReasonLocationDisjoint           = "location_disjoint"
	ReasonLocationMappingUnsupported = "location_mapping_unsupported"
	ReasonLocationMissing            = "location_missing"
	ReasonLocationUnusable           = "location_unusable"

	// Remote restriction verdicts. A vacancy saying "Remote (US/Canada only)" is restricted;
	// these codes say whether the owner's authorizations or current location cover the restriction.
	ReasonRemoteRestrictedCovered    = "remote_restricted_covered"
	ReasonRemoteRestrictedNotCovered = "remote_restricted_not_covered"
	ReasonRemoteRestrictedUnknown    = "remote_restricted_unknown"
	// ReasonRemoteUnrestricted means the vacancy offers unrestricted remote work (worldwide labels
	// or no location label) and the owner has at least one geography data point; it is always a
	// match (utility 1) and appears in the Matches section.
	ReasonRemoteUnrestricted = "remote_unrestricted"

	// ReasonWorkModeRemoteRestricted means the work mode is remote but a regional restriction
	// applies; the geography group carries the coverage verdict instead.
	ReasonWorkModeRemoteRestricted = "work_mode_remote_restricted"

	ReasonAuthorizationCompatible     = "authorization_compatible"
	ReasonAuthorizationDisjoint       = "authorization_disjoint"
	ReasonAuthorizationCountryMissing = "authorization_country_missing"
	ReasonAuthorizationProfileMissing = "authorization_profile_missing"
	ReasonAuthorizationStatusMissing  = "authorization_profile_status_missing"
	ReasonAuthorizationAmbiguous      = "authorization_profile_ambiguous"
	ReasonAuthorizationJobMissing     = "authorization_job_missing"
	ReasonAuthorizationUnusable       = "authorization_fact_unusable"

	ReasonWorkModeCompatible   = "work_mode_compatible"
	ReasonWorkModeDisjoint     = "work_mode_disjoint"
	ReasonWorkModeMissing      = "work_mode_missing"
	ReasonWorkModeUnusable     = "work_mode_unusable"
	ReasonEngagementCompatible = "engagement_compatible"
	ReasonEngagementDisjoint   = "engagement_disjoint"
	ReasonEngagementMissing    = "engagement_missing"
	ReasonEngagementUnusable   = "engagement_unusable"
	ReasonHoursCompatible      = "hours_compatible"
	ReasonHoursDisjoint        = "hours_disjoint"
	ReasonHoursMissing         = "hours_missing"
	ReasonHoursPartialUnknown  = "hours_partial_unknown"
	ReasonHoursUnusable        = "hours_unusable"
	ReasonTimezoneUnsupported  = "timezone_unsupported"
	ReasonTimezoneMissing      = "timezone_missing"
)

// Comparison is one deterministic criterion explanation. ProfileValue/JobValue and the optional
// numbers carry the actual compared values for positive and negative outcomes, not only for an
// unknown quote. Contribution is this criterion's share of its group utility.
type Comparison struct {
	Group             Group              `json:"group"`
	Criterion         Criterion          `json:"criterion"`
	ProfileField      string             `json:"profileField,omitempty"`
	ProfileValue      string             `json:"profileValue,omitempty"`
	ProfileNumber     *float64           `json:"profileNumber,omitempty"`
	SkillID           string             `json:"skillId,omitempty"`
	ProfileProvenance *ProfileProvenance `json:"profileProvenance,omitempty"`
	JobFact           string             `json:"jobFact,omitempty"`
	JobValue          string             `json:"jobValue,omitempty"`
	JobNumber         *float64           `json:"jobNumber,omitempty"`
	Section           Section            `json:"section,omitempty"`
	EvidenceField     string             `json:"evidenceField,omitempty"`
	EvidenceQuote     string             `json:"evidenceQuote,omitempty"`
	Reliability       Reliability        `json:"reliability,omitempty"`
	Applicability     Applicability      `json:"applicability"`
	Comparable        bool               `json:"comparable"`
	Utility           *float64           `json:"utility,omitempty"`
	Weight            float64            `json:"weight,omitempty"`
	Contribution      float64            `json:"contribution,omitempty"`
	Reason            string             `json:"reason"`
}

// Contribution is one group's deterministic contribution to the score.
type Contribution struct {
	Group            Group       `json:"group"`
	Weight           float64     `json:"weight"`
	EvidenceFraction float64     `json:"evidenceFraction"`
	Utility          float64     `json:"utility"`
	Weighted         float64     `json:"weighted"`
	Status           Reliability `json:"status"`
	// Assumed marks a group that had no comparable fact: Utility is the configured midpoint,
	// not an observed comparison.
	Assumed bool `json:"assumed,omitempty"`
}

// Unknown is one missing or ambiguous fact shown to the owner.
type Unknown struct {
	Group       Group   `json:"group"`
	Fact        string  `json:"fact"`
	Section     Section `json:"section"`
	Kind        string  `json:"kind"`
	Quote       string  `json:"quote,omitempty"`
	ProfileEdit string  `json:"profileEdit,omitempty"`
}

// Conflict is one retained disagreement between sources.
type Conflict struct {
	Group  Group  `json:"group"`
	Fact   string `json:"fact"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// Provenance identifies the exact inputs that produced a result.
type Provenance struct {
	ProfileID            string `json:"profileId"`
	ProfileRevision      int    `json:"profileRevision"`
	JobHash              string `json:"jobHash"`
	FactsHash            string `json:"factsHash"`
	RawSourceHash        string `json:"rawSourceHash"`
	ExtractorVersion     string `json:"extractorVersion"`
	SourcePolicyVersion  string `json:"sourcePolicyVersion"`
	SourcePolicyIdentity string `json:"sourcePolicyIdentity,omitempty"`
	SchemaVersion        string `json:"schemaVersion"`
	RulesVersion         string `json:"rulesVersion"`
	RulesIdentity        string `json:"rulesIdentity,omitempty"`
}

// View is the API-facing Quick Match envelope. Result is the complete stored result; Summary is
// the lightweight catalogue projection; StaleReasons explain why a preserved result is not
// current. A stale view keeps the previous values but never presents them as current.
type View struct {
	Result       Result   `json:"result"`
	Summary      Summary  `json:"summary"`
	Stale        bool     `json:"stale"`
	StaleReasons []string `json:"staleReasons,omitempty"`
	// Facts carries the validated vacancy-side facts only while they are the exact production
	// that produced the current result. It is presentation input distinct from the profile
	// preferences and from every computed score.
	Facts *VacancyFacts `json:"facts,omitempty"`
}

// Result is the deterministic evaluator output and renderer input. It carries no model text.
type Result struct {
	Status        ResultStatus            `json:"status"`
	Score         *int                    `json:"score"`
	RawScore      *float64                `json:"rawScore"`
	Coverage      Coverage                `json:"coverage"`
	Applicability map[Group]Applicability `json:"applicability"`
	Reliability   ReliabilityCounts       `json:"reliability"`
	Contributions []Contribution          `json:"contributions"`
	Comparisons   []Comparison            `json:"comparisons"`
	Unknowns      []Unknown               `json:"unknowns"`
	Conflicts     []Conflict              `json:"conflicts"`
	Reasons       []string                `json:"reasons"`
	Provenance    Provenance              `json:"provenance"`
}

// Summary is the lightweight catalogue projection of a result. It carries no recomputed
// utilities; every value is copied from the stored Result.
type Summary struct {
	Status    ResultStatus `json:"status"`
	Score     *int         `json:"score"`
	RawScore  *float64     `json:"rawScore"`
	Coverage  float64      `json:"coverage"`
	Unknowns  int          `json:"unknowns"`
	Conflicts int          `json:"conflicts"`
}
