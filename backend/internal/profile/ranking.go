package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/w3con/sifty/internal/candidate"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/store"
)

// Current returns the stored profile without bootstrapping one. A nil profile means no profile
// exists yet; callers keep their previous behaviour in that case.
func Current(st *store.Store) (*Profile, error) {
	rec, err := st.CurrentProfile()
	if err != nil || rec == nil {
		return nil, err
	}
	return decodeProfile(rec)
}

// ContextPreferences returns the typed, owner-confirmed profile values used as the derived
// candidate context preferences. Configuration preferences are not merged here: once a profile
// exists it is the single authoritative master and the legacy config editor cannot override it.
func ContextPreferences(p *Profile) map[string]any {
	if p == nil {
		return nil
	}
	d := contextProfileData(p.Data)
	out := map[string]any{
		"profileId":         p.ID,
		"profileRevision":   p.Revision,
		"identity":          d.Identity,
		"skills":            d.Skills,
		"experience":        d.Experience,
		"targetRoles":       d.TargetRoles,
		"targetIndustries":  d.TargetIndustries,
		"compensation":      d.Compensation,
		"engagement":        d.Engagement,
		"hours":             d.Hours,
		"workModes":         d.WorkModes,
		"geography":         d.Geography,
		"languages":         d.Languages,
		"availability":      d.Availability,
		"dealbreakers":      d.Dealbreakers,
		"otherRequirements": d.OtherRequirements,
	}
	if len(d.LegacyPreferences) > 0 {
		out["legacyPreferences"] = d.LegacyPreferences
	}
	return out
}

// contextProfileData prepares the effective profile projection used by the existing fit,
// assessment and preparation consumers. It is deliberately not a second stored master:
//   - demographics are removed, so optional gender/race/ethnicity cannot reach a prompt;
//   - a current role has no acting end date;
//   - part-time desired hours and an optional range are effective only when working time is
//     explicitly part-time; full-time and an unset working time retain the stored values without
//     presenting them as an active restriction;
//   - new structured fields replace legacy duplicates only when the owner has supplied them;
//   - rich documents are replaced by their authoritative plain text.
//
// Legacy stored values stay in the database and remain available to the profile editor.
func contextProfileData(d ProfileData) ProfileData {
	d.Demographics = Demographics{Race: []string{}}
	// Slices share their backing array with the stored value; copy row slices before the
	// projection changes an acting end or removes server row identities.
	skills := make([]Skill, len(d.Skills))
	copy(skills, d.Skills)
	for i := range skills {
		skills[i].ID = ""
	}
	d.Skills = skills
	experience := make([]Experience, len(d.Experience))
	copy(experience, d.Experience)
	for i := range experience {
		experience[i].ID = ""
		if experience[i].Current {
			experience[i].End = ""
		}
		experience[i].SummaryDocument = nil
	}
	d.Experience = experience
	if d.Hours.Mode != HoursPartTime {
		// The current editor exposes Weekly as desired part-time hours. A full-time or unset
		// working time keeps the stored values but must not send them as an active restriction.
		d.Hours.Weekly = nil
		d.Hours.WeeklyMin = nil
		d.Hours.WeeklyMax = nil
	}
	d.Hours.WeeklyRangeSet = false
	useOtherRequirements := d.OtherRequirements.Selected ||
		strings.TrimSpace(d.OtherRequirements.Text) != "" || hasRichTextDocument(d.OtherRequirements.Document)
	if useOtherRequirements {
		if strings.TrimSpace(d.OtherRequirements.Text) == "" {
			if text, err := RichTextPlainText(d.OtherRequirements.Document); err == nil {
				d.OtherRequirements.Text = text
			}
		}
		d.Dealbreakers = []string{}
		d.WorkModes.Limits = []string{}
	}
	d.OtherRequirements.Document = nil
	d.OtherRequirements.Selected = false
	if d.Availability.NoticeSet || (d.Availability.NoticeValue != nil && d.Availability.NoticeUnit != "") {
		d.Availability.NoticePeriod = ""
	}
	d.Availability.NoticeSet = false
	if d.Geography.CurrentLocationSet || d.Geography.CurrentLocation != nil {
		d.Geography.Current = ""
	}
	if d.Geography.PreferredLocationsSet || len(d.Geography.PreferredLocations) > 0 {
		d.Geography.Desired = []string{}
	}
	if d.Geography.WorkAuthorizationsSet || len(d.Geography.WorkAuthorizations) > 0 {
		d.Geography.Eligibility = []string{}
		d.Geography.Sponsorship = ""
	}
	if d.Geography.TimezonesSet || len(d.Geography.TimezoneIDs) > 0 {
		d.Geography.Timezones = []string{}
		d.Availability.Timezone = ""
	}
	d.Geography.CurrentLocationSet = false
	d.Geography.PreferredLocationsSet = false
	d.Geography.WorkAuthorizationsSet = false
	d.Geography.TimezonesSet = false
	return d
}

// contextProfileView is the explicit serialized evidence shape. It mirrors the effective profile
// but deliberately has no demographics field at all, not merely an empty one, so a prompt or a
// stored context cannot carry the optional form answers.
type contextProfileView struct {
	Identity             Identity          `json:"identity"`
	Skills               []Skill           `json:"skills"`
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

func contextView(d ProfileData) contextProfileView {
	d = contextProfileData(d)
	return contextProfileView{
		Identity:             d.Identity,
		Skills:               d.Skills,
		Experience:           d.Experience,
		TargetRoles:          d.TargetRoles,
		TargetIndustries:     d.TargetIndustries,
		Compensation:         d.Compensation,
		Engagement:           d.Engagement,
		Hours:                d.Hours,
		WorkModes:            d.WorkModes,
		Geography:            d.Geography,
		Languages:            d.Languages,
		Availability:         d.Availability,
		Dealbreakers:         d.Dealbreakers,
		OtherRequirements:    d.OtherRequirements,
		LegacyPreferences:    d.LegacyPreferences,
		LegacyContextVersion: d.LegacyContextVersion,
	}
}

// ContextSource renders the confirmed profile as one derived evidence source. It is a snapshot
// of stored values with its profile revision in both the text and the metadata; it is not a
// second editable master. The text is deterministic so a revision change is visible in the
// context version even when other sources are unchanged.
func ContextSource(p *Profile) candidate.EvidenceSource {
	if p == nil {
		return candidate.EvidenceSource{}
	}
	body, _ := json.MarshalIndent(contextView(p.Data), "", "  ")
	text := fmt.Sprintf(
		"Owner-confirmed current profile (authoritative; not an editable duplicate).\nProfile ID: %s\nRevision: %d\nUpdated: %s\nConfirmed data:\n%s",
		p.ID, p.Revision, p.UpdatedAt, string(body))
	sum := sha256.Sum256([]byte(text))
	return candidate.EvidenceSource{
		ID:          "confirmed-profile",
		Reference:   "profile:" + p.ID,
		RetrievedAt: p.UpdatedAt,
		Hash:        hex.EncodeToString(sum[:]),
		Status:      "available",
		Category:    "confirmed-profile",
		Scope:       "Authoritative owner-confirmed profile values; derived context, not a second editable profile",
		Text:        text,
	}
}

// FastRankingQuery composes the existing configured FTS search with the confirmed profile's
// target roles and skill names. Each source keeps its own semantics: the configured expression
// groups retain their AND/OR structure, while profile terms are one OR group. The two queries
// are ORed rather than ANDed, so profile skills are ranking signals and never silently filter
// the catalogue. There is no new weighting engine and no automatic search collection.
func FastRankingQuery(cfg *config.Config, data *ProfileData) string {
	var configQuery string
	if cfg != nil {
		exprs := cfg.Search.EffectiveExpressions()
		groups := make([]store.SearchGroup, len(exprs))
		for i, e := range exprs {
			groups[i] = store.SearchGroup{Or: e.Or, And: e.And}
		}
		configQuery = store.BuildFTSQueryExprs(groups)
	}

	var terms []string
	if data != nil {
		terms = append(terms, data.TargetRoles...)
		for _, skill := range data.Skills {
			if strings.TrimSpace(skill.Name) != "" {
				terms = append(terms, skill.Name)
			}
		}
	}
	profileQuery := store.BuildFTSQuery(terms)

	switch {
	case configQuery == "":
		return profileQuery
	case profileQuery == "":
		return configQuery
	default:
		return "(" + configQuery + ") OR (" + profileQuery + ")"
	}
}
