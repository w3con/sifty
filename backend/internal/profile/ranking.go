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
	d := p.Data
	out := map[string]any{
		"profileId":        p.ID,
		"profileRevision":  p.Revision,
		"identity":         d.Identity,
		"skills":           d.Skills,
		"experience":       d.Experience,
		"targetRoles":      d.TargetRoles,
		"targetIndustries": d.TargetIndustries,
		"compensation":     d.Compensation,
		"engagement":       d.Engagement,
		"hours":            d.Hours,
		"workModes":        d.WorkModes,
		"geography":        d.Geography,
		"languages":        d.Languages,
		"availability":     d.Availability,
		"dealbreakers":     d.Dealbreakers,
	}
	if len(d.LegacyPreferences) > 0 {
		out["legacyPreferences"] = d.LegacyPreferences
	}
	return out
}

// ContextSource renders the confirmed profile as one derived evidence source. It is a snapshot
// of stored values with its profile revision in both the text and the metadata; it is not a
// second editable master. The text is deterministic so a revision change is visible in the
// context version even when other sources are unchanged.
func ContextSource(p *Profile) candidate.EvidenceSource {
	if p == nil {
		return candidate.EvidenceSource{}
	}
	body, _ := json.MarshalIndent(p.Data, "", "  ")
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
