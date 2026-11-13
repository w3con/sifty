package profile

// FieldUse names one consumer of a profile field. The catalog below is the single declarative
// place that decides where a field may appear; the derived context, the fast-rank query and the
// document-extraction schema are built from it rather than from copies in views or prompts.
type FieldUse string

const (
	UseProfileDisplay     FieldUse = "profile_display"
	UseAnswerForm         FieldUse = "answer_form"
	UseFastRanking        FieldUse = "fast_ranking"
	UseDetailedAssessment FieldUse = "detailed_assessment"
	UsePreparation        FieldUse = "preparation"
	UseDocumentExtraction FieldUse = "document_extraction"
)

// FieldDescriptor is one field's declared uses and vocabulary. It is exposed to the UI through
// the profile capabilities endpoint; it carries no candidate values.
type FieldDescriptor struct {
	Key     string     `json:"key"`
	Kind    string     `json:"kind"`
	Uses    []FieldUse `json:"uses"`
	Options string     `json:"options,omitempty"`
	Format  string     `json:"format,omitempty"`
	Legacy  bool       `json:"legacy,omitempty"`
	Note    string     `json:"note,omitempty"`
}

// OptionSet is one named vocabulary. Values are stable storage values; Notes explain retained
// legacy values that a current editor must not silently rewrite.
type OptionSet struct {
	Name    string            `json:"name"`
	Values  []string          `json:"values"`
	Default string            `json:"default,omitempty"`
	Notes   map[string]string `json:"notes,omitempty"`
}

var fieldCatalog = []FieldDescriptor{
	{Key: "identity.displayName", Kind: "string", Uses: []FieldUse{UseProfileDisplay, UseAnswerForm, UseDocumentExtraction}},
	{Key: "identity.legalName", Kind: "string", Uses: []FieldUse{UseProfileDisplay, UseAnswerForm}},
	{Key: "identity.headline", Kind: "string", Uses: []FieldUse{UseProfileDisplay}, Legacy: true, Note: "Retained for existing profiles; the current editor no longer offers it. Document extraction no longer proposes it; use target roles and experience for role matching."},
	{Key: "demographics.gender", Kind: "enum", Uses: []FieldUse{UseAnswerForm}, Options: "genders", Note: "Optional application-form answer; never a fit, ranking or preparation input."},
	{Key: "demographics.genderDescription", Kind: "string", Uses: []FieldUse{UseAnswerForm}, Note: "Self-describe text; retained only for the answer/form pathway."},
	{Key: "demographics.race", Kind: "enum_list", Uses: []FieldUse{UseAnswerForm}, Options: "races", Note: "Optional multi-value application-form answer; never a fit, ranking or preparation input."},
	{Key: "demographics.raceDescription", Kind: "string", Uses: []FieldUse{UseAnswerForm}, Note: "Self-describe text; retained only for the answer/form pathway."},
	{Key: "demographics.ethnicity", Kind: "enum", Uses: []FieldUse{UseAnswerForm}, Options: "ethnicities", Note: "Optional application-form answer; never a fit, ranking or preparation input."},
	{Key: "targetRoles", Kind: "string_list", Uses: []FieldUse{UseFastRanking, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "targetIndustries", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "skills", Kind: "object_list", Uses: []FieldUse{UseFastRanking, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "absentSkills", Kind: "object_list", Uses: []FieldUse{UseProfileDisplay, UseFastRanking}, Note: "Skills the owner has said they do not have: a requirement for one is answered no without a model, and it is not offered for adding again."},
	{Key: "skills.level", Kind: "enum", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Options: "skill_levels"},
	{Key: "skills.context", Kind: "rich_text_plain", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Legacy: true, Note: "Removed from the editor; retained text is not deleted."},
	{Key: "languages", Kind: "object_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "languages.level", Kind: "enum_or_free_text", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Options: "language_levels", Note: "Existing arbitrary levels are preserved until manually replaced."},
	{Key: "experience", Kind: "object_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "experience.current", Kind: "boolean", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}},
	{Key: "experience.summaryDocument", Kind: "rich_text", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Format: "tiptap_document", Note: "The plain summary is derived from this document."},
	{Key: "experience.skills", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Legacy: true, Note: "Removed from the editor; old values are retained."},
	{Key: "compensation.cashMin", Kind: "number", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "compensation.cashTarget", Kind: "number", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "compensation.currency", Kind: "string", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "currencies"},
	{Key: "compensation.grossNet", Kind: "enum", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "gross_net"},
	{Key: "compensation.period", Kind: "enum", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "compensation_periods", Note: "Unselected tax basis or period is never presented as confirmed."},
	{Key: "compensation.dayRates.min", Kind: "number", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true},
	{Key: "compensation.dayRates.target", Kind: "number", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true},
	{Key: "compensation.dayRates.currency", Kind: "string", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "currencies", Legacy: true},
	{Key: "compensation.dayRates.period", Kind: "enum", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true},
	{Key: "compensation.equityOnTop", Kind: "boolean", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Note: "Off means no additional preference, not a prohibition; it never overrides an explicit value."},
	{Key: "compensation.crypto", Kind: "enum", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation}, Options: "crypto_payments", Note: "Crypto accepted is the no-restriction default; it never overrides an explicit stored answer."},
	{Key: "engagement.types", Kind: "enum_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "engagement_types"},
	{Key: "hours.mode", Kind: "enum", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "hours_modes"},
	{Key: "hours.weekly", Kind: "number", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Note: "Desired part-time hours; effective only when working time is explicitly part-time, and retained but inactive for full-time or unset."},
	{Key: "hours.weeklyMin", Kind: "number", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}},
	{Key: "hours.weeklyMax", Kind: "number", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}},
	{Key: "workModes.modes", Kind: "enum_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "work_modes"},
	{Key: "workModes.limits", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true, Note: "Moved into Other requirements; old values stay stored and are used until replaced."},
	{Key: "geography.current", Kind: "string", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true, Note: "Old ambiguous location text is retained until manually replaced."},
	{Key: "geography.currentLocation", Kind: "location", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation}, Format: "country_city"},
	{Key: "geography.desired", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Legacy: true, Note: "Old ambiguous location text is retained until manually replaced."},
	{Key: "geography.preferredLocations", Kind: "location_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Format: "country_city"},
	{Key: "geography.eligibility", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Legacy: true, Note: "Old authorization text is retained until manually replaced."},
	{Key: "geography.workAuthorizations", Kind: "authorization_list", Uses: []FieldUse{UseAnswerForm, UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Options: "authorization_statuses", Format: "country_status"},
	{Key: "geography.relocation", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Legacy: true},
	{Key: "geography.willingToRelocate", Kind: "boolean", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}},
	{Key: "geography.sponsorship", Kind: "enum", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true, Options: "sponsorship", Note: "Replaced by country-specific authorization; retained until replaced."},
	{Key: "geography.timezones", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true, Note: "Old free-form timezones are retained; current fields use IANA IDs."},
	{Key: "geography.timezoneIds", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Format: "iana_timezone"},
	{Key: "geography.desiredOverlapHours", Kind: "number", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}},
	{Key: "availability.noticePeriod", Kind: "string", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true, Note: "Retained free-form notice; not rewritten into an exact date."},
	{Key: "availability.noticeValue", Kind: "number", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}},
	{Key: "availability.noticeUnit", Kind: "enum", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Options: "notice_units"},
	{Key: "availability.startDate", Kind: "string", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Format: "date_dd_mm_yyyy"},
	{Key: "availability.timezone", Kind: "string", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true, Note: "Retained; timezone is not duplicated as a second active master."},
	{Key: "dealbreakers", Kind: "string_list", Uses: []FieldUse{UseDetailedAssessment, UsePreparation, UseDocumentExtraction}, Legacy: true, Note: "Other requirements replaces this in the editor; old values are retained and used until replaced."},
	{Key: "otherRequirements", Kind: "rich_text", Uses: []FieldUse{UseDetailedAssessment, UsePreparation}, Format: "tiptap_document"},
	{Key: "legacyPreferences", Kind: "object", Uses: []FieldUse{UseProfileDisplay}, Legacy: true, Note: "Server-owned bootstrap provenance; not editable."},
	{Key: "legacyContextVersion", Kind: "string", Uses: []FieldUse{UseProfileDisplay}, Legacy: true, Note: "Server-owned bootstrap provenance; not editable."},
}

var optionSets = []OptionSet{
	{Name: "skill_levels", Values: SkillLevelOptions(), Notes: map[string]string{
		SkillProduction: "Legacy value retained; choose Beginner, Proficient or Expert to replace it.",
		SkillFamiliar:   "Legacy value retained; choose Beginner, Proficient or Expert to replace it.",
		SkillLearning:   "Legacy value retained; choose Beginner, Proficient or Expert to replace it.",
	}},
	{Name: "genders", Values: []string{GenderWoman, GenderMan, GenderNonBinary, GenderSelfDescribe, GenderPreferNotSay}},
	{Name: "races", Values: []string{RaceAmericanIndian, RaceAsian, RaceBlack, RaceNativeHawaiian, RaceWhite, RaceSelfDescribe, RacePreferNotSay}},
	{Name: "ethnicities", Values: []string{EthnicityHispanic, EthnicityNotHispanic, EthnicityPreferNotSay}},
	{Name: "currencies", Values: []string{"EUR", "USD"}},
	{Name: "gross_net", Values: []string{GrossNetGross, GrossNetNet}},
	{Name: "compensation_periods", Values: []string{PeriodMonth, PeriodYear}, Notes: map[string]string{
		PeriodDay:  "Legacy value retained for an old amount; not produced by the current cash editor.",
		PeriodHour: "Legacy value retained for an old amount; not produced by the current cash editor.",
	}},
	{Name: "crypto_payments", Values: []string{PaymentFiatOnly, PaymentCryptoAccepted, PaymentCryptoOnly}, Default: PaymentCryptoAccepted, Notes: map[string]string{
		PaymentCryptoAccepted: "No restriction on the payment method; not a requirement.",
	}},
	{Name: "engagement_types", Values: []string{EngagementEmployment, EngagementContract}, Notes: map[string]string{
		EngagementEmployee:   "Legacy value retained; the current label is Employment.",
		EngagementContractor: "Legacy value retained; consultant work belongs to Contract.",
	}},
	{Name: "hours_modes", Values: []string{HoursFullTime, HoursPartTime}},
	{Name: "work_modes", Values: []string{WorkRemote, WorkHybrid, WorkOffice}},
	{Name: "authorization_statuses", Values: []string{AuthorizationEligible, AuthorizationNeedVisa}, Notes: map[string]string{
		AuthorizationNotSet: "Not set is represented by an empty status.",
	}},
	{Name: "notice_units", Values: []string{NoticeUnitDay, NoticeUnitWeek, NoticeUnitMonth}},
	{Name: "language_levels", Values: []string{LanguagePreA1, LanguageA1, LanguageA2, LanguageB1, LanguageB2, LanguageC1, LanguageC2, LanguageNative}, Notes: map[string]string{
		LanguagePreA1: "Honest beginner label; not an official CEFR A0 level.",
	}},
	{Name: "sponsorship", Values: []string{SponsorshipRequired, SponsorshipNotRequired}},
}

// FieldDescriptors returns the declarative field-use catalog. The caller may not mutate it.
func FieldDescriptors() []FieldDescriptor {
	out := make([]FieldDescriptor, len(fieldCatalog))
	for i, field := range fieldCatalog {
		out[i] = field
		out[i].Uses = append([]FieldUse(nil), field.Uses...)
	}
	return out
}

// OptionSets returns the declarative vocabularies. The caller may not mutate it.
func OptionSets() []OptionSet {
	out := make([]OptionSet, len(optionSets))
	for i, set := range optionSets {
		out[i] = set
		out[i].Values = append([]string(nil), set.Values...)
		if set.Notes != nil {
			notes := make(map[string]string, len(set.Notes))
			for k, v := range set.Notes {
				notes[k] = v
			}
			out[i].Notes = notes
		}
	}
	return out
}

// FieldUses returns the declared uses for one field key, or nil when the key is not in the
// catalog. It is used by the consumer tests and by future editor wiring.
func FieldUses(key string) []FieldUse {
	for _, field := range fieldCatalog {
		if field.Key == key {
			return append([]FieldUse(nil), field.Uses...)
		}
	}
	return nil
}

// HasUse reports whether a field declares a given consumer.
func (f FieldDescriptor) HasUse(use FieldUse) bool {
	for _, candidate := range f.Uses {
		if candidate == use {
			return true
		}
	}
	return false
}

// DocumentFactFieldKeys returns the field paths the document-extraction schema may propose. It is
// derived from the same catalog as the consumer mapping, so the optional demographics can never
// become document-derived facts.
func DocumentFactFieldKeys() []string {
	out := []string{}
	for _, field := range fieldCatalog {
		if field.HasUse(UseDocumentExtraction) {
			out = append(out, field.Key)
		}
	}
	return out
}
