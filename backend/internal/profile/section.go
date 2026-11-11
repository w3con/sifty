package profile

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Section identifies one independently saved part of the profile editor. The server merges
// exactly this part into the current stored profile, so an editor never sends a stale copy of
// another section over a newer value.
type Section string

const (
	SectionIdentity          Section = "identity"
	SectionDemographics      Section = "demographics"
	SectionSkills            Section = "skills"
	SectionAbsentSkills      Section = "absent-skills"
	SectionExperience        Section = "experience"
	SectionGoals             Section = "goals"
	SectionCompensation      Section = "compensation"
	SectionWork              Section = "work"
	SectionGeography         Section = "geography"
	SectionLanguages         Section = "languages"
	SectionAvailability      Section = "availability"
	SectionDealbreakers      Section = "dealbreakers"
	SectionOtherRequirements Section = "other-requirements"
)

// ParseSection validates a route/section name.
func ParseSection(value string) (Section, error) {
	switch Section(value) {
	case SectionIdentity, SectionDemographics, SectionSkills, SectionAbsentSkills, SectionExperience, SectionGoals,
		SectionCompensation, SectionWork, SectionGeography, SectionLanguages,
		SectionAvailability, SectionDealbreakers, SectionOtherRequirements:
		return Section(value), nil
	}
	return "", fmt.Errorf("unknown profile section %q", value)
}

// goalsSection is the editor shape for target roles and industries; it maps onto two top-level
// list fields so the client never needs a second full ProfileData shape.
type goalsSection struct {
	TargetRoles      []string `json:"targetRoles"`
	TargetIndustries []string `json:"targetIndustries"`
}

// workSection groups the three independently stored work-related sub-objects into one editor
// section. The server still writes them as their existing contract fields.
type workSection struct {
	Engagement Engagement `json:"engagement"`
	Hours      Hours      `json:"hours"`
	WorkModes  WorkModes  `json:"workModes"`
}

// applySection replaces exactly one section in data and reports whether the normalized section
// differs from the stored one. A value changed and then reverted before Save is therefore a
// no-op, not a new revision.
func applySection(data *ProfileData, section Section, value json.RawMessage) (bool, error) {
	before := sectionSnapshot(*data, section)
	if err := mergeSection(data, section, value); err != nil {
		return false, err
	}
	data.Normalize()
	if err := data.SyncPlainText(); err != nil {
		return false, err
	}
	after := sectionSnapshot(*data, section)
	changed := !reflect.DeepEqual(before, after)
	if !changed {
		// An unchanged section writes nothing, even when it contains rows from an older
		// contract that a new change would be required to correct.
		return false, nil
	}
	if section == SectionExperience {
		if err := validateExperiencePartials(data.Experience); err != nil {
			return false, err
		}
	}
	return true, nil
}

// validateExperiencePartials names the visible fields and a human row number. It runs only for a
// changed Experience section so an existing legacy partial row can still be saved unchanged.
func validateExperiencePartials(rows []Experience) error {
	for i, row := range rows {
		company := strings.TrimSpace(row.Company) == ""
		title := strings.TrimSpace(row.Title) == ""
		switch {
		case company && title:
			return fmt.Errorf("Experience %d: enter a company and title", i+1)
		case company:
			return fmt.Errorf("Experience %d: enter a company", i+1)
		case title:
			return fmt.Errorf("Experience %d: enter a title", i+1)
		}
	}
	return nil
}

// mergeSection decodes the editor value and assigns only the named section. Object sections are
// merged by field presence, not by zero values, so a client that does not know a retained field
// cannot erase it by omitting it. An explicitly sent empty value still clears that field.
// Skills keep the server-owned provenance for an unchanged row; a changed or new skill becomes
// manual. Experience rows merge by role identity so removed per-row fields (skills, rich summary)
// survive a save from the current editor.
func mergeSection(data *ProfileData, section Section, value json.RawMessage) error {
	switch section {
	case SectionIdentity:
		merged, err := mergeObject(data.Identity, value)
		if err != nil {
			return fmt.Errorf("identity: %w", err)
		}
		data.Identity = merged
	case SectionDemographics:
		merged, err := mergeObject(data.Demographics, value)
		if err != nil {
			return fmt.Errorf("demographics: %w", err)
		}
		data.Demographics = merged
	case SectionSkills:
		incoming, err := mergeSkills(data.Skills, value)
		if err != nil {
			return fmt.Errorf("skills: %w", err)
		}
		data.Skills = markManualSkillProvenance(incoming, data.Skills)
		// A skill the owner now has is no longer one they do not have.
		data.AbsentSkills = withoutSkills(data.AbsentSkills, data.Skills)
	case SectionAbsentSkills:
		rows, err := mergeAbsentSkills(value)
		if err != nil {
			return fmt.Errorf("skills you do not have: %w", err)
		}
		data.AbsentSkills = rows
		// The two lists never overlap: naming a skill here takes it out of the skills they have.
		kept := make([]Skill, 0, len(data.Skills))
		gone := map[string]bool{}
		for _, a := range rows {
			gone[a.Key] = true
		}
		for _, sk := range data.Skills {
			if !gone[sk.Key] {
				kept = append(kept, sk)
			}
		}
		data.Skills = kept
	case SectionExperience:
		incoming, err := mergeExperience(data.Experience, value)
		if err != nil {
			return fmt.Errorf("experience: %w", err)
		}
		data.Experience = incoming
	case SectionGoals:
		merged, err := mergeObject(goalsSection{TargetRoles: data.TargetRoles, TargetIndustries: data.TargetIndustries}, value)
		if err != nil {
			return fmt.Errorf("goals: %w", err)
		}
		data.TargetRoles = merged.TargetRoles
		data.TargetIndustries = merged.TargetIndustries
	case SectionCompensation:
		merged, err := mergeObject(data.Compensation, value)
		if err != nil {
			return fmt.Errorf("compensation: %w", err)
		}
		data.Compensation = merged
	case SectionWork:
		merged, err := mergeObject(workSection{Engagement: data.Engagement, Hours: data.Hours, WorkModes: data.WorkModes}, value)
		if err != nil {
			return fmt.Errorf("work: %w", err)
		}
		if hours, ok := objectKeys(value)["hours"]; ok {
			keys := objectKeys(hours)
			if _, set := keys["weeklyMin"]; set {
				merged.Hours.WeeklyRangeSet = true
			}
			if _, set := keys["weeklyMax"]; set {
				merged.Hours.WeeklyRangeSet = true
			}
		}
		data.Engagement = merged.Engagement
		data.Hours = merged.Hours
		data.WorkModes = merged.WorkModes
	case SectionGeography:
		merged, err := mergeObject(data.Geography, value)
		if err != nil {
			return fmt.Errorf("geography: %w", err)
		}
		keys := objectKeys(value)
		if _, set := keys["currentLocation"]; set {
			merged.CurrentLocationSet = true
		}
		if _, set := keys["preferredLocations"]; set {
			merged.PreferredLocationsSet = true
		}
		if _, set := keys["workAuthorizations"]; set {
			merged.WorkAuthorizationsSet = true
		}
		if _, set := keys["timezoneIds"]; set {
			merged.TimezonesSet = true
		}
		data.Geography = merged
	case SectionLanguages:
		var incoming []Language
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("languages: %w", err)
		}
		data.Languages = incoming
	case SectionAvailability:
		merged, err := mergeObject(data.Availability, value)
		if err != nil {
			return fmt.Errorf("availability: %w", err)
		}
		keys := objectKeys(value)
		if _, set := keys["noticeValue"]; set {
			merged.NoticeSet = true
		}
		if _, set := keys["noticeUnit"]; set {
			merged.NoticeSet = true
		}
		data.Availability = merged
	case SectionDealbreakers:
		var incoming []string
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("dealbreakers: %w", err)
		}
		data.Dealbreakers = incoming
	case SectionOtherRequirements:
		merged, err := mergeObject(data.OtherRequirements, value)
		if err != nil {
			return fmt.Errorf("other requirements: %w", err)
		}
		keys := objectKeys(value)
		if _, set := keys["text"]; set {
			merged.Selected = true
		}
		if _, set := keys["document"]; set {
			merged.Selected = true
		}
		data.OtherRequirements = merged
	default:
		return fmt.Errorf("unknown profile section %q", section)
	}
	return nil
}

// objectKeys decodes an object's top-level keys without changing the value. It is used only to
// record which current groups the editor explicitly handled, so a cleared choice does not fall
// back to retained legacy text.
func objectKeys(value json.RawMessage) map[string]json.RawMessage {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(value, &keys); err != nil {
		return nil
	}
	return keys
}

// mergeObject overlays only the fields present in value onto current. Nested objects are merged
// recursively; arrays are replaced when present. This is what keeps a hidden legacy field when a
// newer editor omits it, while still allowing an explicit empty value to clear it.
func mergeObject[T any](current T, value json.RawMessage) (T, error) {
	var out T
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return out, err
	}
	currentMap := map[string]json.RawMessage{}
	if err := json.Unmarshal(currentJSON, &currentMap); err != nil {
		return out, err
	}
	var incomingMap map[string]json.RawMessage
	if err := json.Unmarshal(value, &incomingMap); err != nil {
		return out, err
	}
	mergeRawObject(currentMap, incomingMap)
	mergedJSON, err := json.Marshal(currentMap)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(mergedJSON, &out); err != nil {
		return out, err
	}
	return out, nil
}

func mergeRawObject(dst, src map[string]json.RawMessage) {
	for key, incoming := range src {
		if isJSONObject(incoming) {
			if existing, ok := dst[key]; ok && isJSONObject(existing) {
				existingMap := map[string]json.RawMessage{}
				incomingChild := map[string]json.RawMessage{}
				if json.Unmarshal(existing, &existingMap) == nil && json.Unmarshal(incoming, &incomingChild) == nil {
					mergeRawObject(existingMap, incomingChild)
					merged, err := json.Marshal(existingMap)
					if err == nil {
						dst[key] = merged
						continue
					}
				}
			}
		}
		dst[key] = append(json.RawMessage(nil), incoming...)
	}
}

func isJSONObject(raw json.RawMessage) bool {
	for _, b := range raw {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// mergeSkills keeps a retained context value when an incoming skill row omits it, then lets the
// regular provenance rule decide whether the row is unchanged or manual.
func mergeSkills(current []Skill, value json.RawMessage) ([]Skill, error) {
	rows, err := decodeRawRows(value)
	if err != nil {
		return nil, err
	}
	used := make([]bool, len(current))
	out := make([]Skill, 0, len(rows))
	for _, row := range rows {
		var incoming Skill
		rowJSON, _ := json.Marshal(row)
		if err := json.Unmarshal(rowJSON, &incoming); err != nil {
			return nil, err
		}
		match := -1
		if incoming.ID != "" {
			for i, old := range current {
				if !used[i] && old.ID == incoming.ID {
					match = i
					break
				}
			}
		}
		if match < 0 {
			for i, old := range current {
				if used[i] {
					continue
				}
				if old.Key != "" && old.Key == incoming.Key || (old.Name != "" && strings.EqualFold(old.Name, incoming.Name)) {
					match = i
					break
				}
			}
		}
		if match >= 0 {
			used[match] = true
			old := current[match]
			oldJSON, _ := json.Marshal(old)
			oldMap := map[string]json.RawMessage{}
			_ = json.Unmarshal(oldJSON, &oldMap)
			mergeRawObject(oldMap, row)
			mergedJSON, _ := json.Marshal(oldMap)
			_ = json.Unmarshal(mergedJSON, &incoming)
			// The server owns row identity. A client may echo it, but cannot replace it.
			incoming.ID = old.ID
		} else {
			// An unknown client id must not let a different row inherit hidden data. The
			// normalizer will assign a fresh deterministic id after the merge.
			incoming.ID = ""
		}
		out = append(out, incoming)
	}
	return out, nil
}

// mergeExperience keeps removed per-row fields for a row the owner did not delete. Rows are
// matched by company/title, then by position when the arrays have the same length, so a hidden
// skills list or date field is not silently lost on an ordinary edit.
func mergeExperience(current []Experience, value json.RawMessage) ([]Experience, error) {
	rows, err := decodeRawRows(value)
	if err != nil {
		return nil, err
	}
	used := make([]bool, len(current))
	out := make([]Experience, 0, len(rows))
	for _, row := range rows {
		var incoming Experience
		rowJSON, _ := json.Marshal(row)
		if err := json.Unmarshal(rowJSON, &incoming); err != nil {
			return nil, err
		}
		match := -1
		if incoming.ID != "" {
			for i, old := range current {
				if !used[i] && old.ID == incoming.ID {
					match = i
					break
				}
			}
		}
		if match < 0 {
			for i, old := range current {
				if used[i] {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(old.Company), strings.TrimSpace(incoming.Company)) &&
					strings.EqualFold(strings.TrimSpace(old.Title), strings.TrimSpace(incoming.Title)) {
					match = i
					break
				}
			}
		}
		if match >= 0 {
			used[match] = true
			old := current[match]
			oldJSON, _ := json.Marshal(old)
			oldMap := map[string]json.RawMessage{}
			_ = json.Unmarshal(oldJSON, &oldMap)
			mergeRawObject(oldMap, row)
			mergedJSON, _ := json.Marshal(oldMap)
			_ = json.Unmarshal(mergedJSON, &incoming)
			// The server owns row identity. A client may echo it, but cannot replace it.
			incoming.ID = old.ID
		} else {
			incoming.ID = ""
		}
		out = append(out, incoming)
	}
	return out, nil
}

func decodeRawRows(value json.RawMessage) ([]map[string]json.RawMessage, error) {
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(value, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// sectionSnapshot returns the part of the profile the section owns. It is used only to compare
// normalized before/after values, so no whole-profile overwrite can hide a section no-op.
func sectionSnapshot(data ProfileData, section Section) any {
	switch section {
	case SectionIdentity:
		return data.Identity
	case SectionDemographics:
		return data.Demographics
	case SectionSkills:
		return struct {
			Skills []Skill
			Absent []AbsentSkill
		}{data.Skills, data.AbsentSkills}
	case SectionAbsentSkills:
		return struct {
			Skills []Skill
			Absent []AbsentSkill
		}{data.Skills, data.AbsentSkills}
	case SectionExperience:
		return data.Experience
	case SectionGoals:
		return goalsSection{TargetRoles: data.TargetRoles, TargetIndustries: data.TargetIndustries}
	case SectionCompensation:
		return data.Compensation
	case SectionWork:
		return workSection{Engagement: data.Engagement, Hours: data.Hours, WorkModes: data.WorkModes}
	case SectionGeography:
		return data.Geography
	case SectionLanguages:
		return data.Languages
	case SectionAvailability:
		return data.Availability
	case SectionDealbreakers:
		return data.Dealbreakers
	case SectionOtherRequirements:
		return data.OtherRequirements
	}
	return nil
}

// mergeAbsentSkills decodes the list of skills the owner does not have. The list is replaced as a
// whole; a row without a key gets one from its name, and a repeated name is kept once.
func mergeAbsentSkills(value json.RawMessage) ([]AbsentSkill, error) {
	var rows []AbsentSkill
	if err := json.Unmarshal(value, &rows); err != nil {
		return nil, err
	}
	out := make([]AbsentSkill, 0, len(rows))
	seen := map[string]bool{}
	for _, r := range rows {
		r.Name = strings.Join(strings.Fields(r.Name), " ")
		if r.Name == "" {
			continue
		}
		if strings.TrimSpace(r.Key) == "" {
			r.Key = skillKeyOf(r.Name)
		}
		if r.Key == "" || seen[r.Key] {
			continue
		}
		seen[r.Key] = true
		out = append(out, r)
	}
	return out, nil
}

// withoutSkills drops from the absences every skill that is now among the skills the owner has.
func withoutSkills(absent []AbsentSkill, skills []Skill) []AbsentSkill {
	have := map[string]bool{}
	for _, s := range skills {
		have[s.Key] = true
	}
	out := make([]AbsentSkill, 0, len(absent))
	for _, a := range absent {
		if !have[a.Key] {
			out = append(out, a)
		}
	}
	return out
}
