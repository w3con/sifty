// Package profile defines the typed local user-profile contract and the application service
// that reads and saves it. The profile is the authoritative, user-confirmed record of the
// candidate; legacy search preferences and candidate-context snapshots are retained once as
// provenance by the bootstrap path rather than imported as confirmed typed fields or re-read
// as a second master.
//
// Persistence lives in internal/store (SQLite tables user_profiles, profile_documents,
// profile_facts and profile_revisions). HTTP wiring lives in internal/server. This package never
// calls a model provider, enqueues an assessment or writes provider/config selections.
package profile

import (
	"fmt"
	"strings"
)

// LocalProfileID is the single local profile this build manages. Multi-user accounts are not
// part of the current scope.
const LocalProfileID = "local"

// Skill levels. A mention of a technology is not production experience; the level is explicit.
// An empty level means unknown and is allowed for a mentioned skill awaiting user confirmation.
const (
	SkillProduction = "Production"
	SkillFamiliar   = "Familiar"
	SkillLearning   = "Learning"
)

// canonicalSkillLevel maps clear case variants of a known level to its canonical spelling. It
// leaves an unknown or empty level untouched rather than inventing one.
func canonicalSkillLevel(level string) string {
	level = strings.TrimSpace(level)
	for _, valid := range []string{SkillProduction, SkillFamiliar, SkillLearning} {
		if strings.EqualFold(level, valid) {
			return valid
		}
	}
	return level
}

// SkillLevels returns the supported explicit levels in contract order.
func SkillLevels() []string {
	return []string{SkillProduction, SkillFamiliar, SkillLearning}
}

// Engagement types are independent of working hours and work modes.
const (
	EngagementEmployee   = "employee"
	EngagementContractor = "contractor"
)

// Hours modes are independent of engagement type and work mode.
const (
	HoursFullTime = "full_time"
	HoursPartTime = "part_time"
)

// Work modes are independent of hours and engagement type.
const (
	WorkRemote = "remote"
	WorkHybrid = "hybrid"
	WorkOffice = "office"
)

// Compensation qualifiers. An empty value means unknown, never an assumed default.
const (
	GrossNetGross = "gross"
	GrossNetNet   = "net"

	PeriodYear  = "year"
	PeriodMonth = "month"
	PeriodDay   = "day"
	PeriodHour  = "hour"
)

// Sponsorship values. An empty value means unknown; it must never be read as ineligibility.
const (
	SponsorshipRequired    = "required"
	SponsorshipNotRequired = "not_required"
)

// Provenance records where an attributed value came from. It is optional on manual values.
type Provenance struct {
	Kind      string `json:"kind,omitempty"`      // document, manual, legacy-config, legacy-context
	Reference string `json:"reference,omitempty"` // document id, config path or context version
	Quote     string `json:"quote,omitempty"`     // supporting excerpt, when available
}

// Identity is the basic presentation data shown on the profile. The physical residence lives in
// Geography.Current; identity intentionally carries no second copy of it.
type Identity struct {
	DisplayName string `json:"displayName"`
	Headline    string `json:"headline"`
}

// Skill is an explicitly levelled skill with optional context and provenance.
type Skill struct {
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	Level      string      `json:"level"`
	Context    string      `json:"context,omitempty"`
	Provenance *Provenance `json:"provenance,omitempty"`
}

// Experience is one role in the candidate's history.
type Experience struct {
	Company string   `json:"company"`
	Title   string   `json:"title"`
	Start   string   `json:"start,omitempty"`
	End     string   `json:"end,omitempty"`
	Summary string   `json:"summary,omitempty"`
	Skills  []string `json:"skills"`
}

// DayRates holds contract day-rate expectations separately from annual cash compensation.
type DayRates struct {
	Min      *float64 `json:"min"`
	Target   *float64 `json:"target"`
	Currency string   `json:"currency"`
	Period   string   `json:"period"`
}

// Compensation separates guaranteed cash from additional equity preference. Nil numeric values
// mean "unknown", not zero.
type Compensation struct {
	CashMin     *float64 `json:"cashMin"`
	CashTarget  *float64 `json:"cashTarget"`
	Currency    string   `json:"currency"`
	GrossNet    string   `json:"grossNet"`
	Period      string   `json:"period"`
	DayRates    DayRates `json:"dayRates"`
	EquityOnTop *bool    `json:"equityOnTop"`
}

// Engagement describes employee versus contractor work. Both may be acceptable.
type Engagement struct {
	Types []string `json:"types"`
}

// Hours describes full/part-time availability and an optional weekly hour figure.
type Hours struct {
	Mode   string   `json:"mode"`
	Weekly *float64 `json:"weekly"`
}

// WorkModes describes remote/hybrid/office acceptance and free-form limits (travel, on-site
// days, on-call) that are not yet structured.
type WorkModes struct {
	Modes  []string `json:"modes"`
	Limits []string `json:"limits"`
}

// Geography separates current residence, desired locations, work authorization, relocation,
// sponsorship and timezone overlap. Empty eligibility means unknown, not ineligible.
type Geography struct {
	Current           string   `json:"current"`
	Desired           []string `json:"desired"`
	Eligibility       []string `json:"eligibility"`
	Relocation        []string `json:"relocation"`
	WillingToRelocate *bool    `json:"willingToRelocate"`
	Sponsorship       string   `json:"sponsorship"`
	Timezones         []string `json:"timezones"`
}

// Language is a spoken language and its level. Level is free-form because degrees and CEFR
// labels are both in use; an empty level means unknown.
type Language struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

// Availability holds notice period, start date and timezone as free-form strings.
type Availability struct {
	NoticePeriod string `json:"noticePeriod"`
	StartDate    string `json:"startDate"`
	Timezone     string `json:"timezone"`
}

// ProfileData is the complete typed profile contract. Empty strings, nil numbers and empty
// slices mean "unknown"; they must not be displayed or scored as confirmed zero/false values.
type ProfileData struct {
	Identity             Identity       `json:"identity"`
	Skills               []Skill        `json:"skills"`
	Experience           []Experience   `json:"experience"`
	TargetRoles          []string       `json:"targetRoles"`
	TargetIndustries     []string       `json:"targetIndustries"`
	Compensation         Compensation   `json:"compensation"`
	Engagement           Engagement     `json:"engagement"`
	Hours                Hours          `json:"hours"`
	WorkModes            WorkModes      `json:"workModes"`
	Geography            Geography      `json:"geography"`
	Languages            []Language     `json:"languages"`
	Availability         Availability   `json:"availability"`
	Dealbreakers         []string       `json:"dealbreakers"`
	LegacyPreferences    map[string]any `json:"legacyPreferences,omitempty"`
	LegacyContextVersion string         `json:"legacyContextVersion,omitempty"`
}

// Normalize replaces nil slices with empty slices so the JSON contract always exposes arrays.
// It changes no value: unknown numbers remain nil and unknown enums remain empty.
func (p *ProfileData) Normalize() {
	if p.Skills == nil {
		p.Skills = []Skill{}
	}
	if p.Experience == nil {
		p.Experience = []Experience{}
	}
	for i := range p.Skills {
		p.Skills[i].Level = canonicalSkillLevel(p.Skills[i].Level)
	}
	for i := range p.Experience {
		if p.Experience[i].Skills == nil {
			p.Experience[i].Skills = []string{}
		}
	}
	if p.TargetRoles == nil {
		p.TargetRoles = []string{}
	}
	if p.TargetIndustries == nil {
		p.TargetIndustries = []string{}
	}
	if p.Engagement.Types == nil {
		p.Engagement.Types = []string{}
	}
	if p.WorkModes.Modes == nil {
		p.WorkModes.Modes = []string{}
	}
	if p.WorkModes.Limits == nil {
		p.WorkModes.Limits = []string{}
	}
	if p.Geography.Desired == nil {
		p.Geography.Desired = []string{}
	}
	if p.Geography.Eligibility == nil {
		p.Geography.Eligibility = []string{}
	}
	if p.Geography.Relocation == nil {
		p.Geography.Relocation = []string{}
	}
	if p.Geography.Timezones == nil {
		p.Geography.Timezones = []string{}
	}
	if p.Languages == nil {
		p.Languages = []Language{}
	}
	if p.Dealbreakers == nil {
		p.Dealbreakers = []string{}
	}
}

var (
	validSkillLevels = map[string]bool{
		SkillProduction: true, SkillFamiliar: true, SkillLearning: true,
	}
	validEngagementTypes = map[string]bool{
		EngagementEmployee: true, EngagementContractor: true,
	}
	validHoursModes = map[string]bool{HoursFullTime: true, HoursPartTime: true}
	validWorkModes  = map[string]bool{WorkRemote: true, WorkHybrid: true, WorkOffice: true}
	validGrossNet   = map[string]bool{GrossNetGross: true, GrossNetNet: true}
	validPeriods    = map[string]bool{PeriodYear: true, PeriodMonth: true, PeriodDay: true, PeriodHour: true}
	validSponsor    = map[string]bool{SponsorshipRequired: true, SponsorshipNotRequired: true}
)

// Validate rejects malformed typed values before they reach storage. Empty enum values are
// accepted as "unknown"; invalid non-empty values are refused rather than silently coerced.
func (p *ProfileData) Validate() error {
	seenKeys := map[string]bool{}
	for i, s := range p.Skills {
		if strings.TrimSpace(s.Key) == "" {
			return fmt.Errorf("Skill %d: enter a name", i+1)
		}
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("Skill %d: enter a name", i+1)
		}
		if s.Level != "" && !validSkillLevels[s.Level] {
			return fmt.Errorf("skills[%d]: level %q must be Production, Familiar, Learning or empty (unknown)", i, s.Level)
		}
		if seenKeys[s.Key] {
			return fmt.Errorf("skills[%d]: duplicate key %q", i, s.Key)
		}
		seenKeys[s.Key] = true
	}
	for i, e := range p.Engagement.Types {
		if !validEngagementTypes[e] {
			return fmt.Errorf("engagement.types[%d]: %q must be employee or contractor", i, e)
		}
	}
	if p.Hours.Mode != "" && !validHoursModes[p.Hours.Mode] {
		return fmt.Errorf("hours.mode %q must be full_time or part_time", p.Hours.Mode)
	}
	if err := positiveOrNil("hours.weekly", p.Hours.Weekly); err != nil {
		return err
	}
	for i, m := range p.WorkModes.Modes {
		if !validWorkModes[m] {
			return fmt.Errorf("workModes.modes[%d]: %q must be remote, hybrid or office", i, m)
		}
	}
	if p.Compensation.GrossNet != "" && !validGrossNet[p.Compensation.GrossNet] {
		return fmt.Errorf("compensation.grossNet %q must be gross or net", p.Compensation.GrossNet)
	}
	if p.Compensation.Period != "" && !validPeriods[p.Compensation.Period] {
		return fmt.Errorf("compensation.period %q is not a supported period", p.Compensation.Period)
	}
	if err := positiveOrNil("compensation.cashMin", p.Compensation.CashMin); err != nil {
		return err
	}
	if err := positiveOrNil("compensation.cashTarget", p.Compensation.CashTarget); err != nil {
		return err
	}
	if err := positiveOrNil("compensation.dayRates.min", p.Compensation.DayRates.Min); err != nil {
		return err
	}
	if err := positiveOrNil("compensation.dayRates.target", p.Compensation.DayRates.Target); err != nil {
		return err
	}
	if p.Compensation.CashMin != nil && p.Compensation.CashTarget != nil &&
		*p.Compensation.CashTarget < *p.Compensation.CashMin {
		return fmt.Errorf("compensation.cashTarget must be greater than or equal to cashMin")
	}
	if (p.Compensation.CashMin != nil || p.Compensation.CashTarget != nil) &&
		strings.TrimSpace(p.Compensation.Currency) == "" {
		return fmt.Errorf("compensation.currency is required when a cash amount is provided")
	}
	if p.Compensation.DayRates.Min != nil && p.Compensation.DayRates.Target != nil &&
		*p.Compensation.DayRates.Target < *p.Compensation.DayRates.Min {
		return fmt.Errorf("compensation.dayRates.target must be greater than or equal to min")
	}
	if (p.Compensation.DayRates.Min != nil || p.Compensation.DayRates.Target != nil) &&
		strings.TrimSpace(p.Compensation.DayRates.Currency) == "" {
		return fmt.Errorf("compensation.dayRates.currency is required when a day rate is provided")
	}
	// A day rate is never annual pay: when its period is set at all, it must be "day".
	if p.Compensation.DayRates.Period != "" && p.Compensation.DayRates.Period != PeriodDay {
		return fmt.Errorf("compensation.dayRates.period must be %q", PeriodDay)
	}
	if p.Geography.Sponsorship != "" && !validSponsor[p.Geography.Sponsorship] {
		return fmt.Errorf("geography.sponsorship %q must be required or not_required", p.Geography.Sponsorship)
	}
	for i, l := range p.Languages {
		if strings.TrimSpace(l.Name) == "" {
			return fmt.Errorf("languages[%d]: name is required", i)
		}
	}
	return nil
}

func positiveOrNil(name string, v *float64) error {
	if v != nil && *v <= 0 {
		return fmt.Errorf("%s must be positive when provided (use null for unknown)", name)
	}
	return nil
}
