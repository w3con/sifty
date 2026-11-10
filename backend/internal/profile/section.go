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
	SectionIdentity     Section = "identity"
	SectionSkills       Section = "skills"
	SectionExperience   Section = "experience"
	SectionGoals        Section = "goals"
	SectionCompensation Section = "compensation"
	SectionWork         Section = "work"
	SectionGeography    Section = "geography"
	SectionLanguages    Section = "languages"
	SectionAvailability Section = "availability"
	SectionDealbreakers Section = "dealbreakers"
)

// ParseSection validates a route/section name.
func ParseSection(value string) (Section, error) {
	switch Section(value) {
	case SectionIdentity, SectionSkills, SectionExperience, SectionGoals,
		SectionCompensation, SectionWork, SectionGeography, SectionLanguages,
		SectionAvailability, SectionDealbreakers:
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

// mergeSection decodes the editor value and assigns only the named section. Skills keep the
// server-owned provenance for an unchanged row; a changed or new skill becomes manual.
func mergeSection(data *ProfileData, section Section, value json.RawMessage) error {
	switch section {
	case SectionIdentity:
		var incoming Identity
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("identity: %w", err)
		}
		data.Identity = incoming
	case SectionSkills:
		var incoming []Skill
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("skills: %w", err)
		}
		data.Skills = markManualSkillProvenance(incoming, data.Skills)
	case SectionExperience:
		var incoming []Experience
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("experience: %w", err)
		}
		data.Experience = incoming
	case SectionGoals:
		var incoming goalsSection
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("goals: %w", err)
		}
		data.TargetRoles = incoming.TargetRoles
		data.TargetIndustries = incoming.TargetIndustries
	case SectionCompensation:
		var incoming Compensation
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("compensation: %w", err)
		}
		data.Compensation = incoming
	case SectionWork:
		var incoming workSection
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("work: %w", err)
		}
		data.Engagement = incoming.Engagement
		data.Hours = incoming.Hours
		data.WorkModes = incoming.WorkModes
	case SectionGeography:
		var incoming Geography
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("geography: %w", err)
		}
		data.Geography = incoming
	case SectionLanguages:
		var incoming []Language
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("languages: %w", err)
		}
		data.Languages = incoming
	case SectionAvailability:
		var incoming Availability
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("availability: %w", err)
		}
		data.Availability = incoming
	case SectionDealbreakers:
		var incoming []string
		if err := json.Unmarshal(value, &incoming); err != nil {
			return fmt.Errorf("dealbreakers: %w", err)
		}
		data.Dealbreakers = incoming
	default:
		return fmt.Errorf("unknown profile section %q", section)
	}
	return nil
}

// sectionSnapshot returns the part of the profile the section owns. It is used only to compare
// normalized before/after values, so no whole-profile overwrite can hide a section no-op.
func sectionSnapshot(data ProfileData, section Section) any {
	switch section {
	case SectionIdentity:
		return data.Identity
	case SectionSkills:
		return data.Skills
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
	}
	return nil
}
