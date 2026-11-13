package jobextract

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/w3con/sifty/internal/matchcontract"
)

// MergeModelConditions fills the working-condition facts the deterministic parse did not find
// with the conditions a model extracted from the same vacancy text (docs/specs/quick-match.md
// section 14). It follows three rules:
//
//   - A condition is used only when the ordinary parse found no fact for that field. The caller
//     has already verified that the condition's quote is an exact excerpt of the retained source.
//   - When both state the field and disagree, the deterministic fact stands and the disagreement
//     is returned as a conflict to be kept for audit, never shown.
//   - A model value that cannot be read as a typed value (an unknown work mode, a salary with no
//     amount) is ignored, never guessed.
//   - A model location is usable only when its quote states the restriction explicitly in one of
//     restrictionPhrases (matchcontract.Rules.RestrictionPhrases); a bare place name needs
//     corroboration.
//
// It returns a copy of the facts with the facts hash recomputed; the input is not modified.
func MergeModelConditions(f matchcontract.VacancyFacts, c matchcontract.ExtractedConditions, restrictionPhrases []string) (matchcontract.VacancyFacts, []matchcontract.ConditionConflict, error) {
	out := f
	out.Compensation = f.Compensation
	out.Geography.HiringLocations = append([]matchcontract.LocationFact(nil), f.Geography.HiringLocations...)
	out.Conditions.WorkModes = append([]matchcontract.WorkModeFact(nil), f.Conditions.WorkModes...)
	var conflicts []matchcontract.ConditionConflict
	conflict := func(field, deterministic, model, quote string) {
		conflicts = append(conflicts, matchcontract.ConditionConflict{
			Field: field, Deterministic: deterministic, Model: model, Quote: quote,
		})
	}
	evidence := func(quote string, reliability matchcontract.Reliability) matchcontract.Evidence {
		return matchcontract.Evidence{
			Field: "model_extraction", Quote: quote,
			Section: matchcontract.SectionUnknown, Reliability: reliability,
		}
	}

	if v := c.WorkMode; v != nil {
		if mode, ok := modeFromValue(v.Value); ok {
			if len(out.Conditions.WorkModes) == 0 {
				out.Conditions.WorkModes = append(out.Conditions.WorkModes, matchcontract.WorkModeFact{
					Mode: mode, Evidence: evidence(v.Quote, matchcontract.ReliabilityUsable),
				})
			} else if !hasMode(out.Conditions.WorkModes, mode) {
				conflict("work_mode", joinModes(out.Conditions.WorkModes), mode, v.Quote)
			}
		}
	}
	if v := c.Engagement; v != nil {
		if typ, ok := engagementToken(v.Value); ok {
			if out.Conditions.Engagement == nil {
				out.Conditions.Engagement = &matchcontract.EngagementFact{
					Type: typ, Evidence: evidence(v.Quote, matchcontract.ReliabilityUsable),
				}
			} else if out.Conditions.Engagement.Type != typ {
				conflict("engagement", out.Conditions.Engagement.Type, typ, v.Quote)
			}
		}
	}
	if v := c.Hours; v != nil {
		var probe matchcontract.VacancyFacts
		addHours(&probe, v.Value, v.Quote)
		if h := probe.Conditions.Hours; h != nil {
			h.Evidence = evidence(v.Quote, matchcontract.ReliabilityUsable)
			switch {
			case out.Conditions.Hours == nil:
				out.Conditions.Hours = h
			case out.Conditions.Hours.Mode != "" && h.Mode != "" && out.Conditions.Hours.Mode != h.Mode:
				conflict("hours", out.Conditions.Hours.Mode, h.Mode, v.Quote)
			}
		}
	}
	if v := c.Timezone; v != nil && strings.TrimSpace(v.Value) != "" {
		switch {
		case out.Conditions.Timezone == nil:
			out.Conditions.Timezone = &matchcontract.TimezoneFact{
				Value: strings.TrimSpace(v.Value), Evidence: evidence(v.Quote, matchcontract.ReliabilityUsable),
			}
		case !locationsAgree(out.Conditions.Timezone.Value, v.Value):
			conflict("timezone", out.Conditions.Timezone.Value, v.Value, v.Quote)
		}
	}
	if v := c.Compensation; v != nil {
		if cash, ok := cashText(v.Value, v.Quote); ok {
			cash.Evidence.Field = "model_extraction"
			switch {
			case out.Compensation.Cash == nil:
				out.Compensation.Cash = cash
			case !sameCash(*out.Compensation.Cash, *cash):
				conflict("compensation", describeCash(*out.Compensation.Cash), describeCash(*cash), v.Quote)
			}
		}
	}
	if v := c.Location; v != nil {
		var labels []string
		for _, loc := range splitLocations(v.Value) {
			if _, isMode := workModeToken(loc); !isMode {
				labels = append(labels, loc)
			}
		}
		switch {
		case len(labels) == 0:
		case len(out.Geography.HiringLocations) == 0:
			// A bare place the model read from running text is one statement, not a verified
			// hiring restriction, so it needs corroboration like any generic location line. A quote
			// that states the restriction itself ("Remote (US/Canada only)") is usable.
			reliability := matchcontract.ReliabilityNeedsCorroboration
			if statesRestriction(v.Quote, restrictionPhrases) {
				reliability = matchcontract.ReliabilityUsable
			}
			for _, label := range labels {
				out.Geography.HiringLocations = append(out.Geography.HiringLocations, matchcontract.LocationFact{
					Label: label, Evidence: evidence(v.Quote, reliability),
				})
			}
		default:
			agreed := false
			for _, label := range labels {
				for _, existing := range out.Geography.HiringLocations {
					if locationsAgree(existing.Label, label) {
						agreed = true
					}
				}
			}
			if !agreed {
				var have []string
				for _, existing := range out.Geography.HiringLocations {
					have = append(have, existing.Label)
				}
				conflict("location", strings.Join(have, " · "), strings.Join(labels, " · "), v.Quote)
			}
		}
	}

	sortFacts(&out)
	if err := out.SetFactsHash(); err != nil {
		return matchcontract.VacancyFacts{}, nil, fmt.Errorf("merge model conditions: %w", err)
	}
	if err := out.Validate(); err != nil {
		return matchcontract.VacancyFacts{}, nil, fmt.Errorf("merge model conditions: %w", err)
	}
	return out, conflicts, nil
}

// statesRestriction reports whether quote contains one of phrases as whole words; case,
// punctuation and spacing are ignored, so "US-only" contains "only".
func statesRestriction(quote string, phrases []string) bool {
	hay := " " + plainWords(quote) + " "
	for _, p := range phrases {
		if w := plainWords(p); w != "" && strings.Contains(hay, " "+w+" ") {
			return true
		}
	}
	return false
}

func plainWords(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

// modeFromValue reads a work mode from a model's free wording. The hybrid cue wins over remote,
// because "hybrid, remote two days" is a hybrid role.
func modeFromValue(value string) (string, bool) {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "hybrid"):
		return matchcontract.WorkHybrid, true
	case strings.Contains(lower, "remote"):
		return matchcontract.WorkRemote, true
	case strings.Contains(lower, "on-site") || strings.Contains(lower, "onsite") ||
		strings.Contains(lower, "on site") || strings.Contains(lower, "in office") ||
		strings.Contains(lower, "in-office") || lower == "office":
		return matchcontract.WorkOffice, true
	}
	return "", false
}

func hasMode(modes []matchcontract.WorkModeFact, mode string) bool {
	for _, m := range modes {
		if m.Mode == mode {
			return true
		}
	}
	return false
}

func joinModes(modes []matchcontract.WorkModeFact) string {
	parts := make([]string, len(modes))
	for i, m := range modes {
		parts[i] = m.Mode
	}
	return strings.Join(parts, ",")
}

func sameCash(a, b matchcontract.CashFact) bool {
	eq := func(x, y *float64) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) }
	return eq(a.Min, b.Min) && eq(a.Max, b.Max) && (a.Currency == "" || b.Currency == "" || a.Currency == b.Currency)
}

func describeCash(c matchcontract.CashFact) string {
	num := func(x *float64) string {
		if x == nil {
			return "?"
		}
		return fmt.Sprintf("%g", *x)
	}
	return fmt.Sprintf("%s-%s %s %s", num(c.Min), num(c.Max), c.Currency, c.Period)
}
