package profile

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/w3con/sifty/internal/store"
)

// Fact API states. A proposal never changes the profile by itself.
const (
	FactProposed      = "proposed"
	FactAccepted      = "accepted"
	FactRejected      = "rejected"
	FactStateConflict = "conflict"
)

// FactStates returns the visible fact states.
func FactStates() []string {
	return []string{FactProposed, FactAccepted, FactRejected, FactStateConflict}
}

// Fact is the API-facing attributed assertion. Value is the exact typed JSON that would be
// applied to the profile when accepted. Quote is an exact excerpt of the source document.
type Fact struct {
	ID              string          `json:"factId"`
	ProfileID       string          `json:"profileId"`
	DocumentID      *string         `json:"documentId,omitempty"`
	DocumentName    string          `json:"documentName,omitempty"`
	FieldKey        string          `json:"fieldKey"`
	Value           json.RawMessage `json:"value"`
	Quote           string          `json:"quote"`
	SourceReference string          `json:"sourceReference,omitempty"`
	Origin          string          `json:"origin"`
	State           string          `json:"state"`
	SupersedesID    *string         `json:"supersedesId,omitempty"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

// FactConflict is a visible unresolved choice between the stored value and a proposed fact.
type FactConflict struct {
	FieldKey        string `json:"fieldKey"`
	Current         any    `json:"current"`
	Proposed        any    `json:"proposed"`
	FactID          string `json:"factId"`
	CurrentRevision int    `json:"currentRevision"`
	Message         string `json:"message"`
}

// FactConflictError tells the caller that confirming the fact would overwrite a different
// confirmed value. The fact is stored as "conflict"; the profile is unchanged.
type FactConflictError struct {
	Conflict *FactConflict
}

func (e *FactConflictError) Error() string {
	if e.Conflict == nil {
		return "profile fact conflict"
	}
	return e.Conflict.Message
}

// FactDecision is returned after accept/reject/resolve. When a conflict was detected the profile
// is nil and Conflict carries the choice.
type FactDecision struct {
	Fact           *Fact         `json:"fact"`
	Profile        *Profile      `json:"profile,omitempty"`
	ProfileChanged bool          `json:"profileChanged"`
	Conflict       *FactConflict `json:"conflict,omitempty"`
}

type factKind int

const (
	factString factKind = iota
	factNumber
	factBool
	factStringList
	factEnumList
	factSkill
	factLanguage
	factExperience
)

type factField struct {
	kind  factKind
	enum  map[string]bool
	label string
}

var factFields = map[string]factField{
	"identity.displayName":           {kind: factString},
	"identity.headline":              {kind: factString},
	"targetRoles":                    {kind: factStringList},
	"targetIndustries":               {kind: factStringList},
	"skills":                         {kind: factSkill},
	"languages":                      {kind: factLanguage},
	"experience":                     {kind: factExperience},
	"compensation.cashMin":           {kind: factNumber},
	"compensation.cashTarget":        {kind: factNumber},
	"compensation.currency":          {kind: factString},
	"compensation.grossNet":          {kind: factString, enum: validGrossNet},
	"compensation.period":            {kind: factString, enum: validPeriods},
	"compensation.dayRates.min":      {kind: factNumber},
	"compensation.dayRates.target":   {kind: factNumber},
	"compensation.dayRates.currency": {kind: factString},
	"compensation.dayRates.period":   {kind: factString, enum: map[string]bool{PeriodDay: true}},
	"compensation.equityOnTop":       {kind: factBool},
	"engagement.types":               {kind: factEnumList, enum: validEngagementTypes},
	"hours.mode":                     {kind: factString, enum: validHoursModes},
	"hours.weekly":                   {kind: factNumber},
	"workModes.modes":                {kind: factEnumList, enum: validWorkModes},
	"workModes.limits":               {kind: factStringList},
	"geography.current":              {kind: factString},
	"geography.desired":              {kind: factStringList},
	"geography.eligibility":          {kind: factStringList},
	"geography.relocation":           {kind: factStringList},
	"geography.willingToRelocate":    {kind: factBool},
	"geography.sponsorship":          {kind: factString, enum: validSponsor},
	"geography.timezones":            {kind: factStringList},
	"availability.noticePeriod":      {kind: factString},
	"availability.startDate":         {kind: factString},
	"availability.timezone":          {kind: factString},
	"dealbreakers":                   {kind: factStringList},
}

// AllowedFactFieldKeys returns the schema-approved field paths in a stable order.
func AllowedFactFieldKeys() []string {
	return []string{
		"identity.displayName", "identity.headline",
		"targetRoles", "targetIndustries", "skills", "languages", "experience",
		"compensation.cashMin", "compensation.cashTarget", "compensation.currency",
		"compensation.grossNet", "compensation.period",
		"compensation.dayRates.min", "compensation.dayRates.target",
		"compensation.dayRates.currency", "compensation.dayRates.period",
		"compensation.equityOnTop",
		"engagement.types", "hours.mode", "hours.weekly",
		"workModes.modes", "workModes.limits",
		"geography.current", "geography.desired", "geography.eligibility",
		"geography.relocation", "geography.willingToRelocate",
		"geography.sponsorship", "geography.timezones",
		"availability.noticePeriod", "availability.startDate", "availability.timezone",
		"dealbreakers",
	}
}

// ValidateFactValue checks the field key and value type and returns the canonical JSON. Unknown
// keys and values are refused rather than stored as free-form facts.
func ValidateFactValue(fieldKey string, raw json.RawMessage) (json.RawMessage, error) {
	field, ok := factFields[fieldKey]
	if !ok {
		return nil, fmt.Errorf("unknown profile field %q", fieldKey)
	}
	switch field.kind {
	case factString:
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("%s must be a string", fieldKey)
		}
		v = strings.TrimSpace(v)
		if v == "" {
			return nil, fmt.Errorf("%s must not be empty", fieldKey)
		}
		if field.enum != nil && !field.enum[v] {
			return nil, fmt.Errorf("%s has unsupported value %q", fieldKey, v)
		}
		return json.Marshal(v)
	case factNumber:
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("%s must be a number", fieldKey)
		}
		if v <= 0 {
			return nil, fmt.Errorf("%s must be positive when provided", fieldKey)
		}
		return json.Marshal(v)
	case factBool:
		var v bool
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("%s must be a boolean", fieldKey)
		}
		return json.Marshal(v)
	case factStringList, factEnumList:
		var v []string
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("%s must be an array of strings", fieldKey)
		}
		clean := []string{}
		seen := map[string]bool{}
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if field.kind == factEnumList && !field.enum[item] {
				return nil, fmt.Errorf("%s has unsupported value %q", fieldKey, item)
			}
			if !seen[strings.ToLower(item)] {
				seen[strings.ToLower(item)] = true
				clean = append(clean, item)
			}
		}
		if len(clean) == 0 {
			return nil, fmt.Errorf("%s must contain at least one value", fieldKey)
		}
		return json.Marshal(clean)
	case factSkill:
		var v Skill
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("skills must be an object with key, name and level")
		}
		v.Key = strings.TrimSpace(v.Key)
		v.Name = strings.TrimSpace(v.Name)
		v.Level = strings.TrimSpace(v.Level)
		if v.Key == "" || v.Name == "" {
			return nil, fmt.Errorf("skills require key and name")
		}
		v.Level = canonicalSkillLevel(v.Level)
		if v.Level != "" && !validSkillLevels[v.Level] {
			return nil, fmt.Errorf("skills level must be Production, Familiar, Learning or empty (an unstated level is not invented)")
		}
		v.Provenance = nil
		return json.Marshal(v)
	case factLanguage:
		var v Language
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("languages must be an object with name and optional level")
		}
		v.Name = strings.TrimSpace(v.Name)
		v.Level = strings.TrimSpace(v.Level)
		if v.Name == "" {
			return nil, fmt.Errorf("languages require a name")
		}
		return json.Marshal(v)
	case factExperience:
		var v Experience
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("experience must be an object with company and title")
		}
		v.Company = strings.TrimSpace(v.Company)
		v.Title = strings.TrimSpace(v.Title)
		if v.Company == "" || v.Title == "" {
			return nil, fmt.Errorf("experience requires company and title")
		}
		if v.Skills == nil {
			v.Skills = []string{}
		}
		return json.Marshal(v)
	}
	return nil, fmt.Errorf("unsupported field %q", fieldKey)
}

// ApplyFact merges a validated fact value into a profile copy. It returns changed=false and a
// FactConflict when a known value differs, unless force is set (an explicit user resolution).
func ApplyFact(data *ProfileData, fieldKey string, value json.RawMessage, force bool) (bool, *FactConflict, error) {
	if _, ok := factFields[fieldKey]; !ok {
		return false, nil, fmt.Errorf("unknown profile field %q", fieldKey)
	}
	switch fieldKey {
	case "identity.displayName":
		return applyString(fieldKey, &data.Identity.DisplayName, value, force)
	case "identity.headline":
		return applyString(fieldKey, &data.Identity.Headline, value, force)
	case "compensation.currency":
		return applyString(fieldKey, &data.Compensation.Currency, value, force)
	case "compensation.grossNet":
		return applyString(fieldKey, &data.Compensation.GrossNet, value, force)
	case "compensation.period":
		return applyString(fieldKey, &data.Compensation.Period, value, force)
	case "compensation.dayRates.currency":
		return applyString(fieldKey, &data.Compensation.DayRates.Currency, value, force)
	case "compensation.dayRates.period":
		return applyString(fieldKey, &data.Compensation.DayRates.Period, value, force)
	case "hours.mode":
		return applyString(fieldKey, &data.Hours.Mode, value, force)
	case "geography.current":
		return applyString(fieldKey, &data.Geography.Current, value, force)
	case "geography.sponsorship":
		return applyString(fieldKey, &data.Geography.Sponsorship, value, force)
	case "availability.noticePeriod":
		return applyString(fieldKey, &data.Availability.NoticePeriod, value, force)
	case "availability.startDate":
		return applyString(fieldKey, &data.Availability.StartDate, value, force)
	case "availability.timezone":
		return applyString(fieldKey, &data.Availability.Timezone, value, force)
	case "compensation.cashMin":
		return applyNumber(fieldKey, &data.Compensation.CashMin, value, force)
	case "compensation.cashTarget":
		return applyNumber(fieldKey, &data.Compensation.CashTarget, value, force)
	case "compensation.dayRates.min":
		return applyNumber(fieldKey, &data.Compensation.DayRates.Min, value, force)
	case "compensation.dayRates.target":
		return applyNumber(fieldKey, &data.Compensation.DayRates.Target, value, force)
	case "hours.weekly":
		return applyNumber(fieldKey, &data.Hours.Weekly, value, force)
	case "compensation.equityOnTop":
		return applyBool(fieldKey, &data.Compensation.EquityOnTop, value, force)
	case "geography.willingToRelocate":
		return applyBool(fieldKey, &data.Geography.WillingToRelocate, value, force)
	case "targetRoles":
		return applyStringList(fieldKey, &data.TargetRoles, value)
	case "targetIndustries":
		return applyStringList(fieldKey, &data.TargetIndustries, value)
	case "workModes.limits":
		return applyStringList(fieldKey, &data.WorkModes.Limits, value)
	case "geography.desired":
		return applyStringList(fieldKey, &data.Geography.Desired, value)
	case "geography.eligibility":
		return applyStringList(fieldKey, &data.Geography.Eligibility, value)
	case "geography.relocation":
		return applyStringList(fieldKey, &data.Geography.Relocation, value)
	case "geography.timezones":
		return applyStringList(fieldKey, &data.Geography.Timezones, value)
	case "dealbreakers":
		return applyStringList(fieldKey, &data.Dealbreakers, value)
	case "engagement.types":
		return applyStringList(fieldKey, &data.Engagement.Types, value)
	case "workModes.modes":
		return applyStringList(fieldKey, &data.WorkModes.Modes, value)
	case "skills":
		return applySkill(fieldKey, &data.Skills, value, force)
	case "languages":
		return applyLanguage(fieldKey, &data.Languages, value, force)
	case "experience":
		return applyExperience(fieldKey, &data.Experience, value, force)
	}
	return false, nil, fmt.Errorf("unsupported field %q", fieldKey)
}

func applyString(fieldKey string, dst *string, raw json.RawMessage, force bool) (bool, *FactConflict, error) {
	var proposed string
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return false, nil, err
	}
	current := strings.TrimSpace(*dst)
	if current == proposed {
		return false, nil, nil
	}
	if current != "" && !force {
		return false, &FactConflict{
			FieldKey: fieldKey, Current: current, Proposed: proposed,
			Message: fmt.Sprintf("%s already has a different confirmed value", fieldKey),
		}, nil
	}
	*dst = proposed
	return true, nil, nil
}

func applyNumber(fieldKey string, dst **float64, raw json.RawMessage, force bool) (bool, *FactConflict, error) {
	var proposed float64
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return false, nil, err
	}
	if *dst != nil {
		if **dst == proposed {
			return false, nil, nil
		}
		if !force {
			return false, &FactConflict{
				FieldKey: fieldKey, Current: **dst, Proposed: proposed,
				Message: fmt.Sprintf("%s already has a different confirmed value", fieldKey),
			}, nil
		}
	}
	v := proposed
	*dst = &v
	return true, nil, nil
}

func applyBool(fieldKey string, dst **bool, raw json.RawMessage, force bool) (bool, *FactConflict, error) {
	var proposed bool
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return false, nil, err
	}
	if *dst != nil {
		if **dst == proposed {
			return false, nil, nil
		}
		if !force {
			return false, &FactConflict{
				FieldKey: fieldKey, Current: **dst, Proposed: proposed,
				Message: fmt.Sprintf("%s already has a different confirmed value", fieldKey),
			}, nil
		}
	}
	v := proposed
	*dst = &v
	return true, nil, nil
}

func applyStringList(fieldKey string, dst *[]string, raw json.RawMessage) (bool, *FactConflict, error) {
	var proposed []string
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return false, nil, err
	}
	changed := false
	for _, item := range proposed {
		if !containsFold(*dst, item) {
			*dst = append(*dst, item)
			changed = true
		}
	}
	return changed, nil, nil
}

func applySkill(fieldKey string, dst *[]Skill, raw json.RawMessage, force bool) (bool, *FactConflict, error) {
	var proposed Skill
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return false, nil, err
	}
	for i := range *dst {
		cur := (*dst)[i]
		if cur.Key == proposed.Key || strings.EqualFold(cur.Name, proposed.Name) {
			if skillEqual(cur, proposed) {
				return false, nil, nil
			}
			if !force {
				return false, &FactConflict{
					FieldKey: fieldKey, Current: cur, Proposed: proposed,
					Message: fmt.Sprintf("skill %q already has a different confirmed level or context", proposed.Name),
				}, nil
			}
			proposed.Provenance = cur.Provenance
			(*dst)[i] = proposed
			return true, nil, nil
		}
	}
	*dst = append(*dst, proposed)
	return true, nil, nil
}

func applyLanguage(fieldKey string, dst *[]Language, raw json.RawMessage, force bool) (bool, *FactConflict, error) {
	var proposed Language
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return false, nil, err
	}
	for i := range *dst {
		cur := (*dst)[i]
		if strings.EqualFold(cur.Name, proposed.Name) {
			if cur.Level == proposed.Level {
				return false, nil, nil
			}
			if !force {
				return false, &FactConflict{
					FieldKey: fieldKey, Current: cur, Proposed: proposed,
					Message: fmt.Sprintf("language %q already has a different confirmed level", proposed.Name),
				}, nil
			}
			(*dst)[i] = proposed
			return true, nil, nil
		}
	}
	*dst = append(*dst, proposed)
	return true, nil, nil
}

func applyExperience(fieldKey string, dst *[]Experience, raw json.RawMessage, force bool) (bool, *FactConflict, error) {
	var proposed Experience
	if err := json.Unmarshal(raw, &proposed); err != nil {
		return false, nil, err
	}
	for i := range *dst {
		cur := (*dst)[i]
		if strings.EqualFold(cur.Company, proposed.Company) && strings.EqualFold(cur.Title, proposed.Title) {
			if experienceEqual(cur, proposed) {
				return false, nil, nil
			}
			if !force {
				return false, &FactConflict{
					FieldKey: fieldKey, Current: cur, Proposed: proposed,
					Message: fmt.Sprintf("experience %q at %q already has different confirmed details", proposed.Title, proposed.Company),
				}, nil
			}
			(*dst)[i] = proposed
			return true, nil, nil
		}
	}
	*dst = append(*dst, proposed)
	return true, nil, nil
}

func skillEqual(a, b Skill) bool {
	return a.Key == b.Key && a.Name == b.Name && a.Level == b.Level && a.Context == b.Context
}

func experienceEqual(a, b Experience) bool {
	return a.Company == b.Company && a.Title == b.Title && a.Start == b.Start && a.End == b.End &&
		a.Summary == b.Summary && reflect.DeepEqual(a.Skills, b.Skills)
}

func containsFold(list []string, v string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}

func factFromRecord(rec *store.ProfileFactRecord, documentName string) *Fact {
	f := &Fact{
		ID:              rec.ID,
		ProfileID:       rec.ProfileID,
		DocumentID:      rec.DocumentID,
		DocumentName:    documentName,
		FieldKey:        rec.FieldKey,
		Value:           json.RawMessage(rec.ValueJSON),
		Quote:           rec.Quote,
		SourceReference: rec.SourceReference,
		Origin:          rec.Origin,
		State:           rec.State,
		SupersedesID:    rec.SupersedesID,
		CreatedAt:       rec.CreatedAt,
		UpdatedAt:       rec.UpdatedAt,
	}
	return f
}
