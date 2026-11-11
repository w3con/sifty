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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// LocalProfileID is the single local profile this build manages. Multi-user accounts are not
// part of the current scope.
const LocalProfileID = "local"

// Skill levels. A mention of a technology is not production experience; the level is explicit.
// An empty level means unknown and is allowed for a mentioned skill awaiting user confirmation.
// The current levels are Beginner/Proficient/Expert. Production/Familiar/Learning are retained
// so older stored values survive unchanged until the owner chooses a current level; they are
// never mapped automatically.
const (
	SkillBeginner   = "Beginner"
	SkillProficient = "Proficient"
	SkillExpert     = "Expert"

	// Legacy levels, retained for stored profiles and compatibility with the previous form.
	SkillProduction = "Production"
	SkillFamiliar   = "Familiar"
	SkillLearning   = "Learning"
)

// canonicalSkillLevel maps clear case variants of a known level to its canonical spelling. It
// leaves an unknown or empty level untouched rather than inventing one.
func canonicalSkillLevel(level string) string {
	level = strings.TrimSpace(level)
	for _, valid := range allSkillLevels() {
		if strings.EqualFold(level, valid) {
			return valid
		}
	}
	return level
}

func allSkillLevels() []string {
	return []string{SkillBeginner, SkillProficient, SkillExpert, SkillProduction, SkillFamiliar, SkillLearning}
}

// SkillLevels returns the legacy explicit levels exposed by the pre-existing capabilities field.
// It is kept unchanged so an older client keeps a working select until the current level editor
// consumes SkillLevelOptions. New profiles use SkillLevelOptions.
func SkillLevels() []string {
	return []string{SkillProduction, SkillFamiliar, SkillLearning}
}

// SkillLevelOptions returns the current explicit levels in contract order. "Not set" is the empty
// value and is deliberately not a stored level.
func SkillLevelOptions() []string {
	return []string{SkillBeginner, SkillProficient, SkillExpert}
}

// Gender and the optional demographics of an application form. Empty means not set; a separate
// "prefer not to say" value is a deliberate answer and must not be collapsed into empty.
const (
	GenderWoman        = "woman"
	GenderMan          = "man"
	GenderNonBinary    = "non_binary"
	GenderSelfDescribe = "self_describe"
	GenderPreferNotSay = "prefer_not_to_say"
)

// Race values. Race is a multi-value field; Prefer not to say is exclusive.
const (
	RaceAmericanIndian = "american_indian_or_alaska_native"
	RaceAsian          = "asian"
	RaceBlack          = "black_or_african_american"
	RaceNativeHawaiian = "native_hawaiian_or_other_pacific_islander"
	RaceWhite          = "white"
	RaceSelfDescribe   = "self_describe"
	RacePreferNotSay   = "prefer_not_to_say"
)

// Ethnicity values. Empty means not set; Prefer not to say is a deliberate answer.
const (
	EthnicityHispanic     = "hispanic_or_latino"
	EthnicityNotHispanic  = "not_hispanic_or_latino"
	EthnicityPreferNotSay = "prefer_not_to_say"
)

// Engagement types are independent of working hours and work modes. Employment/Contract are the
// current labels; employee/contractor remain accepted for stored profiles and older clients.
const (
	EngagementEmployment = "employment"
	EngagementContract   = "contract"

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

// Payment method preference. CryptoAccepted is the default meaning "no restriction on the
// payment method"; it is not a requirement. An explicit stored value is never overridden.
const (
	PaymentFiatOnly       = "fiat_only"
	PaymentCryptoAccepted = "crypto_accepted"
	PaymentCryptoOnly     = "crypto_only"
)

// Work authorization status for one country. Empty means Not set, not ineligible.
const (
	AuthorizationEligible = "eligible"
	AuthorizationNeedVisa = "need_visa"
	AuthorizationNotSet   = ""
)

// Notice period units. The number and unit are structured; NoticePeriod remains the retained
// free-form legacy value.
const (
	NoticeUnitDay   = "day"
	NoticeUnitWeek  = "week"
	NoticeUnitMonth = "month"
)

// Language levels. Pre-A1 is an honest beginner label and is not presented as official CEFR A0.
const (
	LanguagePreA1  = "Pre-A1"
	LanguageA1     = "A1"
	LanguageA2     = "A2"
	LanguageB1     = "B1"
	LanguageB2     = "B2"
	LanguageC1     = "C1"
	LanguageC2     = "C2"
	LanguageNative = "Native"
)

// Provenance records where an attributed value came from. It is optional on manual values.
type Provenance struct {
	Kind      string `json:"kind,omitempty"`      // document, manual, legacy-config, legacy-context
	Reference string `json:"reference,omitempty"` // document id, config path or context version
	Quote     string `json:"quote,omitempty"`     // supporting excerpt, when available
}

// Identity is the basic presentation data shown on the profile. The physical residence lives in
// Geography; identity intentionally carries no second copy of it. Headline is legacy data kept
// for existing profiles; the current editor no longer offers it.
type Identity struct {
	DisplayName string `json:"displayName"`
	LegalName   string `json:"legalName,omitempty"`
	Headline    string `json:"headline,omitempty"`
}

// Demographics are optional application-form answers. They are deliberately separate from the
// fit/preparation context: no current search, quick score, assessment or preparation input may
// include them, and document extraction cannot propose them. They are read back only for the
// profile/application-form pathway, and nothing is ever sent automatically.
type Demographics struct {
	Gender            string   `json:"gender,omitempty"`
	GenderDescription string   `json:"genderDescription,omitempty"`
	Race              []string `json:"race"`
	RaceDescription   string   `json:"raceDescription,omitempty"`
	Ethnicity         string   `json:"ethnicity,omitempty"`
}

// Skill is an explicitly levelled skill with optional context and provenance.
type Skill struct {
	ID         string      `json:"id,omitempty"`
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	Level      string      `json:"level"`
	Context    string      `json:"context,omitempty"`
	Provenance *Provenance `json:"provenance,omitempty"`
}

// Experience is one role in the candidate's history. Start/End are month/year strings for the
// current editor; existing free-form values are retained. Current is explicit and must never be
// inferred from an empty End. Summary is the authoritative plain-text derivative of
// SummaryDocument; the supported rich document is optional and existing plain text is retained.
// Skills is legacy per-row data removed from the editor and preserved without loss.
type Experience struct {
	ID              string          `json:"id,omitempty"`
	Company         string          `json:"company"`
	Title           string          `json:"title"`
	Start           string          `json:"start,omitempty"`
	End             string          `json:"end,omitempty"`
	Current         bool            `json:"current,omitempty"`
	Summary         string          `json:"summary,omitempty"`
	SummaryDocument json.RawMessage `json:"summaryDocument,omitempty"`
	Skills          []string        `json:"skills"`
}

// DayRates holds contract day-rate expectations separately from annual cash compensation.
type DayRates struct {
	Min      *float64 `json:"min"`
	Target   *float64 `json:"target"`
	Currency string   `json:"currency"`
	Period   string   `json:"period"`
}

// Compensation separates guaranteed cash from additional equity preference. Nil numeric values
// mean "unknown", not zero. Crypto is the payment-method preference, not a currency; the cash
// currency remains EUR/USD in the current editor. EquityOnTop nil/false means no additional
// preference, not a prohibition, and never overrides an explicit stored value.
type Compensation struct {
	CashMin     *float64 `json:"cashMin"`
	CashTarget  *float64 `json:"cashTarget"`
	Currency    string   `json:"currency"`
	GrossNet    string   `json:"grossNet"`
	Period      string   `json:"period"`
	DayRates    DayRates `json:"dayRates"`
	EquityOnTop *bool    `json:"equityOnTop"`
	Crypto      string   `json:"crypto,omitempty"`
}

// Engagement describes employment versus contract work. Both may be acceptable. Stored legacy
// employee/contractor values remain valid and are never rewritten automatically.
type Engagement struct {
	Types []string `json:"types"`
}

// Hours describes full/part-time availability. Weekly is the desired weekly hours exposed by the
// current part-time editor; older profiles may have it from legacy data. WeeklyMin/WeeklyMax are
// the optional part-time range. Stored values are preserved for full-time or unset working time
// but are not effective inputs then. WeeklyRangeSet records that the current editor handled the
// range; it is compatibility metadata and does not suppress the desired hours.
type Hours struct {
	Mode           string   `json:"mode"`
	Weekly         *float64 `json:"weekly"`
	WeeklyMin      *float64 `json:"weeklyMin,omitempty"`
	WeeklyMax      *float64 `json:"weeklyMax,omitempty"`
	WeeklyRangeSet bool     `json:"weeklyRangeSet,omitempty"`
}

// WorkModes describes remote/hybrid/office acceptance. Limits is retained legacy free text;
// the current editor moves those conditions into OtherRequirements without deleting them.
type WorkModes struct {
	Modes  []string `json:"modes"`
	Limits []string `json:"limits"`
}

// Location is a country (ISO 3166-1 alpha-2) and an optional manually entered city. It makes no
// promise that every city is known locally.
type Location struct {
	Country string `json:"country"`
	City    string `json:"city,omitempty"`
}

// WorkAuthorization is one country-specific authorization answer. Empty status means Not set.
type WorkAuthorization struct {
	Country string `json:"country"`
	Status  string `json:"status,omitempty"`
}

// Geography separates current residence, desired locations, work authorization, relocation,
// sponsorship and timezone overlap. Empty eligibility means unknown, not ineligible. Current and
// Desired are retained free-form legacy strings; old ambiguous values stay unstructured until
// manually replaced. CurrentLocation/PreferredLocations are the current typed fields.
type Geography struct {
	Current             string              `json:"current"`
	Desired             []string            `json:"desired"`
	Eligibility         []string            `json:"eligibility"`
	Relocation          []string            `json:"relocation"`
	WillingToRelocate   *bool               `json:"willingToRelocate"`
	Sponsorship         string              `json:"sponsorship"`
	Timezones           []string            `json:"timezones"`
	CurrentLocation     *Location           `json:"currentLocation,omitempty"`
	PreferredLocations  []Location          `json:"preferredLocations"`
	WorkAuthorizations  []WorkAuthorization `json:"workAuthorizations"`
	TimezoneIDs         []string            `json:"timezoneIds"`
	DesiredOverlapHours *float64            `json:"desiredOverlapHours,omitempty"`

	// Effective-source markers distinguish "never handled by the current editor" from an
	// explicit empty selection. They prevent a cleared current choice from reviving retained
	// legacy text. They are set by section saves, not guessed from values.
	CurrentLocationSet    bool `json:"currentLocationSet,omitempty"`
	PreferredLocationsSet bool `json:"preferredLocationsSet,omitempty"`
	WorkAuthorizationsSet bool `json:"workAuthorizationsSet,omitempty"`
	TimezonesSet          bool `json:"timezonesSet,omitempty"`
}

// Language is a spoken language and its level. Level stays a string so an existing arbitrary
// value is preserved until manually replaced; current options are exposed separately.
type Language struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

// Availability holds notice period, start date and timezone. NoticeValue/NoticeUnit are the
// structured current fields; NoticePeriod and Timezone are retained legacy values, not a second
// active master.
type Availability struct {
	NoticePeriod string   `json:"noticePeriod"`
	StartDate    string   `json:"startDate"`
	Timezone     string   `json:"timezone"`
	NoticeValue  *float64 `json:"noticeValue,omitempty"`
	NoticeUnit   string   `json:"noticeUnit,omitempty"`
	NoticeSet    bool     `json:"noticeSet,omitempty"`
}

// OtherRequirements is the current free-text block with optional simple rich text. Text is the
// authoritative plain-text derivative and Document is the supported document; legacy
// Dealbreakers and WorkModes.Limits remain stored and are used when this block is empty.
type OtherRequirements struct {
	Text     string          `json:"text"`
	Document json.RawMessage `json:"document,omitempty"`
	Selected bool            `json:"selected,omitempty"`
}

// AbsentSkill is a skill the owner has said they do not have, kept beside the skills they do have.
// It is a confirmed absence: a requirement for it is answered "no" without asking a model, and it
// is not offered for adding again. Key follows the same rule as Skill.Key.
type AbsentSkill struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ProfileData is the complete typed profile contract. Empty strings, nil numbers and empty
// slices mean "unknown"; they must not be displayed or scored as confirmed zero/false values.
type ProfileData struct {
	Identity             Identity          `json:"identity"`
	Demographics         Demographics      `json:"demographics"`
	Skills               []Skill           `json:"skills"`
	AbsentSkills         []AbsentSkill     `json:"absentSkills"`
	Experience           []Experience      `json:"experience"`
	TargetRoles          []string          `json:"targetRoles"`
	TargetIndustries     []string          `json:"targetIndustries"`
	Compensation         Compensation      `json:"compensation"`
	Engagement           Engagement        `json:"engagement"`
	Hours                Hours             `json:"hours"`
	WorkModes            WorkModes         `json:"workModes"`
	Geography            Geography         `json:"geography"`
	Languages            []Language        `json:"languages"`
	Availability         Availability      `json:"availability"`
	Dealbreakers         []string          `json:"dealbreakers"`
	OtherRequirements    OtherRequirements `json:"otherRequirements"`
	LegacyPreferences    map[string]any    `json:"legacyPreferences,omitempty"`
	LegacyContextVersion string            `json:"legacyContextVersion,omitempty"`
}

// Normalize replaces nil slices with empty slices so the JSON contract always exposes arrays.
// It changes no value: unknown numbers remain nil and unknown enums remain empty.
func (p *ProfileData) Normalize() {
	if p.Demographics.Race == nil {
		p.Demographics.Race = []string{}
	}
	if p.Skills == nil {
		p.Skills = []Skill{}
	}
	if p.AbsentSkills == nil {
		p.AbsentSkills = []AbsentSkill{}
	}
	if p.Experience == nil {
		p.Experience = []Experience{}
	}
	for i := range p.Skills {
		p.Skills[i].Level = canonicalSkillLevel(p.Skills[i].Level)
		if p.Skills[i].ID == "" {
			// Legacy rows had no identity. A deterministic read-time id is stable while the
			// row's visible identity is unchanged; the first save persists it, and the editor
			// echoes it so a later rename or reorder cannot attach hidden data to another row.
			p.Skills[i].ID = stableSkillID(p.Skills[i])
		}
	}
	skillIDs := make([]string, len(p.Skills))
	for i := range p.Skills {
		skillIDs[i] = p.Skills[i].ID
	}
	dedupeIDs(skillIDs)
	for i := range p.Skills {
		p.Skills[i].ID = skillIDs[i]
	}
	for i := range p.Experience {
		if p.Experience[i].Skills == nil {
			p.Experience[i].Skills = []string{}
		}
		if p.Experience[i].ID == "" {
			p.Experience[i].ID = stableExperienceID(p.Experience[i])
		}
		if p.Experience[i].Current {
			// A current role has no acting end date. The client keeps its previous draft
			// value; the stored active contract does not claim one.
			p.Experience[i].End = ""
		}
	}
	experienceIDs := make([]string, len(p.Experience))
	for i := range p.Experience {
		experienceIDs[i] = p.Experience[i].ID
	}
	dedupeIDs(experienceIDs)
	for i := range p.Experience {
		p.Experience[i].ID = experienceIDs[i]
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
	if p.Geography.PreferredLocations == nil {
		p.Geography.PreferredLocations = []Location{}
	}
	if p.Geography.WorkAuthorizations == nil {
		p.Geography.WorkAuthorizations = []WorkAuthorization{}
	}
	if p.Geography.TimezoneIDs == nil {
		p.Geography.TimezoneIDs = []string{}
	}
	if p.Languages == nil {
		p.Languages = []Language{}
	}
	if p.Dealbreakers == nil {
		p.Dealbreakers = []string{}
	}
}

func stableSkillID(s Skill) string {
	return stableRowID("sk", s.Key, s.Name, s.Level, s.Context)
}

func stableExperienceID(e Experience) string {
	return stableRowID("ex", e.Company, e.Title, e.Start, e.End,
		e.Summary, strings.Join(e.Skills, "\x1f"), fmt.Sprint(e.Current))
}

func stableRowID(prefix string, parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("%s-%x", prefix, sum[:8])
}

// dedupeIDs keeps server row identity unique even for legacy rows whose content is identical.
// Identical legacy rows are interchangeable; a suffix makes each persisted id addressable.
func dedupeIDs(ids []string) {
	seen := map[string]bool{}
	for i, id := range ids {
		if id == "" {
			continue
		}
		candidate := id
		for suffix := 2; seen[candidate]; suffix++ {
			candidate = fmt.Sprintf("%s-%d", id, suffix)
		}
		ids[i] = candidate
		seen[candidate] = true
	}
}

// SyncPlainText refreshes the authoritative plain-text derivative from any supported rich-text
// document. A section without a document keeps its existing plain text; this is the compatibility
// path for values stored before rich text existed. It never invents text and never writes storage.
func (p *ProfileData) SyncPlainText() error {
	if hasRichTextDocument(p.OtherRequirements.Document) {
		text, err := RichTextPlainText(p.OtherRequirements.Document)
		if err != nil {
			return fmt.Errorf("otherRequirements.document: %w", err)
		}
		p.OtherRequirements.Text = text
	}
	for i := range p.Experience {
		if !hasRichTextDocument(p.Experience[i].SummaryDocument) {
			continue
		}
		text, err := RichTextPlainText(p.Experience[i].SummaryDocument)
		if err != nil {
			return fmt.Errorf("experience[%d].summaryDocument: %w", i, err)
		}
		p.Experience[i].Summary = text
	}
	return nil
}

var (
	validSkillLevels = map[string]bool{
		SkillBeginner: true, SkillProficient: true, SkillExpert: true,
		SkillProduction: true, SkillFamiliar: true, SkillLearning: true,
	}
	validEngagementTypes = map[string]bool{
		EngagementEmployment: true, EngagementContract: true,
		EngagementEmployee: true, EngagementContractor: true,
	}
	validHoursModes = map[string]bool{HoursFullTime: true, HoursPartTime: true}
	validWorkModes  = map[string]bool{WorkRemote: true, WorkHybrid: true, WorkOffice: true}
	validGrossNet   = map[string]bool{GrossNetGross: true, GrossNetNet: true}
	validPeriods    = map[string]bool{PeriodYear: true, PeriodMonth: true, PeriodDay: true, PeriodHour: true}
	validSponsor    = map[string]bool{SponsorshipRequired: true, SponsorshipNotRequired: true}
	validGender     = map[string]bool{
		GenderWoman: true, GenderMan: true, GenderNonBinary: true,
		GenderSelfDescribe: true, GenderPreferNotSay: true,
	}
	validRace = map[string]bool{
		RaceAmericanIndian: true, RaceAsian: true, RaceBlack: true, RaceNativeHawaiian: true,
		RaceWhite: true, RaceSelfDescribe: true, RacePreferNotSay: true,
	}
	validEthnicity = map[string]bool{
		EthnicityHispanic: true, EthnicityNotHispanic: true, EthnicityPreferNotSay: true,
	}
	validPayments = map[string]bool{
		PaymentFiatOnly: true, PaymentCryptoAccepted: true, PaymentCryptoOnly: true,
	}
	validAuthorizations = map[string]bool{
		AuthorizationEligible: true, AuthorizationNeedVisa: true, AuthorizationNotSet: true,
	}
	validNoticeUnits = map[string]bool{NoticeUnitDay: true, NoticeUnitWeek: true, NoticeUnitMonth: true}
)

// Validate rejects malformed typed values before they reach storage. Empty enum values are
// accepted as "unknown"; invalid non-empty values are refused rather than silently coerced.
func (p *ProfileData) Validate() error {
	have := map[string]bool{}
	for _, s := range p.Skills {
		have[strings.TrimSpace(s.Key)] = true
	}
	absent := map[string]bool{}
	for i, a := range p.AbsentSkills {
		if strings.TrimSpace(a.Key) == "" || strings.TrimSpace(a.Name) == "" {
			return fmt.Errorf("Skill you do not have %d: enter a name", i+1)
		}
		if absent[a.Key] {
			return fmt.Errorf("%q is listed twice among the skills you do not have", a.Name)
		}
		if have[a.Key] {
			return fmt.Errorf("%q is both in your skills and among the skills you do not have", a.Name)
		}
		absent[a.Key] = true
	}
	seenKeys := map[string]bool{}
	seenSkillIDs := map[string]bool{}
	for i, s := range p.Skills {
		if strings.TrimSpace(s.Key) == "" {
			return fmt.Errorf("Skill %d: enter a name", i+1)
		}
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("Skill %d: enter a name", i+1)
		}
		if s.Level != "" && !validSkillLevels[s.Level] {
			return fmt.Errorf("skills[%d]: level %q must be Beginner, Proficient, Expert or empty (unknown); legacy Production/Familiar/Learning values are retained until replaced", i, s.Level)
		}
		if seenKeys[s.Key] {
			return fmt.Errorf("skills[%d]: duplicate key %q", i, s.Key)
		}
		seenKeys[s.Key] = true
		if s.ID != "" {
			if seenSkillIDs[s.ID] {
				return fmt.Errorf("skills[%d]: duplicate id %q", i, s.ID)
			}
			seenSkillIDs[s.ID] = true
		}
	}
	if err := validateDemographics(p.Demographics); err != nil {
		return err
	}
	for i, e := range p.Engagement.Types {
		if !validEngagementTypes[e] {
			return fmt.Errorf("engagement.types[%d]: %q must be employment or contract (legacy employee/contractor are retained)", i, e)
		}
	}
	if p.Hours.Mode != "" && !validHoursModes[p.Hours.Mode] {
		return fmt.Errorf("hours.mode %q must be full_time or part_time", p.Hours.Mode)
	}
	if err := positiveOrNil("hours.weekly", p.Hours.Weekly); err != nil {
		return err
	}
	if err := positiveOrNil("hours.weeklyMin", p.Hours.WeeklyMin); err != nil {
		return err
	}
	if err := positiveOrNil("hours.weeklyMax", p.Hours.WeeklyMax); err != nil {
		return err
	}
	if p.Hours.WeeklyMin != nil && p.Hours.WeeklyMax != nil && *p.Hours.WeeklyMax < *p.Hours.WeeklyMin {
		return fmt.Errorf("hours.weeklyMax must be greater than or equal to weeklyMin")
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
	if p.Compensation.Crypto != "" && !validPayments[p.Compensation.Crypto] {
		return fmt.Errorf("compensation.crypto %q must be fiat_only, crypto_accepted or crypto_only", p.Compensation.Crypto)
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
	if p.Geography.CurrentLocation != nil {
		if err := validateLocation("geography.currentLocation", *p.Geography.CurrentLocation); err != nil {
			return err
		}
	}
	for i, loc := range p.Geography.PreferredLocations {
		if err := validateLocation(fmt.Sprintf("geography.preferredLocations[%d]", i), loc); err != nil {
			return err
		}
	}
	for i, auth := range p.Geography.WorkAuthorizations {
		if strings.TrimSpace(auth.Country) == "" {
			return fmt.Errorf("geography.workAuthorizations[%d]: country is required", i)
		}
		if !validAuthorizations[auth.Status] {
			return fmt.Errorf("geography.workAuthorizations[%d].status %q must be eligible, need_visa or empty", i, auth.Status)
		}
	}
	if err := nonNegativeOrNil("geography.desiredOverlapHours", p.Geography.DesiredOverlapHours); err != nil {
		return err
	}
	if p.Availability.NoticeUnit != "" && !validNoticeUnits[p.Availability.NoticeUnit] {
		return fmt.Errorf("availability.noticeUnit %q must be day, week or month", p.Availability.NoticeUnit)
	}
	if p.Availability.NoticeValue != nil {
		if *p.Availability.NoticeValue <= 0 {
			return fmt.Errorf("availability.noticeValue must be positive when provided")
		}
		if p.Availability.NoticeUnit == "" {
			return fmt.Errorf("availability.noticeUnit is required when noticeValue is provided")
		}
	}
	seenExperienceIDs := map[string]bool{}
	for i := range p.Experience {
		if p.Experience[i].ID != "" {
			if seenExperienceIDs[p.Experience[i].ID] {
				return fmt.Errorf("experience[%d]: duplicate id %q", i, p.Experience[i].ID)
			}
			seenExperienceIDs[p.Experience[i].ID] = true
		}
		if hasRichTextDocument(p.Experience[i].SummaryDocument) {
			if err := validateRichTextDocument(p.Experience[i].SummaryDocument); err != nil {
				return fmt.Errorf("experience[%d].summaryDocument: %w", i, err)
			}
		}
	}
	if hasRichTextDocument(p.OtherRequirements.Document) {
		if err := validateRichTextDocument(p.OtherRequirements.Document); err != nil {
			return fmt.Errorf("otherRequirements.document: %w", err)
		}
	}
	for i, l := range p.Languages {
		if strings.TrimSpace(l.Name) == "" {
			return fmt.Errorf("languages[%d]: name is required", i)
		}
	}
	return nil
}

func validateDemographics(d Demographics) error {
	if d.Gender != "" && !validGender[d.Gender] {
		return fmt.Errorf("demographics.gender %q is not a supported answer", d.Gender)
	}
	seen := map[string]bool{}
	prefer := false
	for i, race := range d.Race {
		if !validRace[race] {
			return fmt.Errorf("demographics.race[%d] %q is not a supported answer", i, race)
		}
		if seen[race] {
			return fmt.Errorf("demographics.race[%d] duplicates %q", i, race)
		}
		seen[race] = true
		if race == RacePreferNotSay {
			prefer = true
		}
	}
	if prefer && len(d.Race) > 1 {
		return fmt.Errorf("demographics.race: prefer_not_to_say cannot be combined with another answer")
	}
	if d.Ethnicity != "" && !validEthnicity[d.Ethnicity] {
		return fmt.Errorf("demographics.ethnicity %q is not a supported answer", d.Ethnicity)
	}
	return nil
}

func validateLocation(name string, loc Location) error {
	if strings.TrimSpace(loc.Country) == "" {
		return fmt.Errorf("%s: country is required", name)
	}
	return nil
}

func nonNegativeOrNil(name string, v *float64) error {
	if v != nil && *v < 0 {
		return fmt.Errorf("%s must not be negative when provided", name)
	}
	return nil
}

func positiveOrNil(name string, v *float64) error {
	if v != nil && *v <= 0 {
		return fmt.Errorf("%s must be positive when provided (use null for unknown)", name)
	}
	return nil
}
