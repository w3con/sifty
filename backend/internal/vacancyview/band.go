package vacancyview

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
)

// sectionLike is the common interface for sections consumed by this package.
type sectionLike interface {
	Class() string
	TextContent() string
}

// wrappedSection adapts model.Section to sectionLike.
type wrappedSection struct{ s model.Section }

func (w wrappedSection) Class() string        { return w.s.Class }
func (w wrappedSection) TextContent() string   { return stripHTML(w.s.Content) }

// wrapSections adapts a slice of model.Section to []sectionLike.
func wrapSections(ss []model.Section) []sectionLike {
	out := make([]sectionLike, len(ss))
	for i, s := range ss {
		out[i] = wrappedSection{s}
	}
	return out
}

var tagRe = regexp.MustCompile(`<[^>]+>`)

// stripHTML removes HTML tags and decodes entities for plain-text search.
func stripHTML(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return s
}

// MoneyFromSections computes the MoneyResult from raw model.Section slices.
// It is the routes-facing wrapper so the caller does not need to import sectionLike.
func MoneyFromSections(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, sections []model.Section, rates map[string]float64) MoneyResult {
	return ComputeMoneyResult(profile, facts, wrapSections(sections), rates)
}

// ComputeBand derives the four goal tiles (Money, Location, Work mode, Contract) from the merged
// vacancy facts, sections and configuration.
//
// mergedFacts should be the output of jobextract.MergeModelConditions when an extraction
// is available, otherwise the raw stored facts.
func ComputeBand(
	profile matchcontract.ProfileSnapshot,
	mergedFacts matchcontract.VacancyFacts,
	sections []model.Section,
	jobLocation string,
	cfg config.VacancyViewConfig,
) *model.Band {
	wrapped := wrapSections(sections)
	tiles := []model.BandTile{
		moneyTile(profile, mergedFacts, wrapped, cfg.ExchangeRates),
		locationTile(mergedFacts, jobLocation),
		workModeTile(profile, mergedFacts, wrapped),
		contractTile(profile, mergedFacts, wrapped, cfg),
	}
	return &model.Band{Tiles: tiles}
}

// moneyTile builds the Money tile.
func moneyTile(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, sections []sectionLike, rates map[string]float64) model.BandTile {
	res := ComputeMoneyResult(profile, facts, sections, rates)
	// Use the short form ("$150–250k") as the displayed value; full range goes in approxValue row.
	displayValue := res.ShortRange
	if displayValue == "" {
		displayValue = res.OriginalRange
	}
	tile := model.BandTile{
		Label:       "Money",
		Value:       displayValue,
		ApproxValue: res.ApproxEUR,
		Verdict:     res.Verdict,
	}
	if displayValue != "" {
		tile.Origin = "cash"
	}
	switch res.Verdict {
	case "ok":
		tile.Hint = "above your minimum"
	case "unclear":
		tile.Hint = "covers your minimum"
	case "mismatch":
		tile.Hint = "below your minimum"
	}
	return tile
}

// locationTile builds the Location tile.
// Verdict is always "" — no verdict circle is shown; a location verdict is not yet implemented.
func locationTile(facts matchcontract.VacancyFacts, jobLocation string) model.BandTile {
	// Try the most reliable location first: a named hiring location from facts.
	for _, loc := range facts.Geography.HiringLocations {
		if strings.TrimSpace(loc.Label) != "" {
			return model.BandTile{Label: "Location", Value: loc.Label, Origin: "facts", Hint: "from listing", Verdict: ""}
		}
	}
	// Fall back to the job's raw location string.
	if loc := strings.TrimSpace(jobLocation); loc != "" {
		return model.BandTile{Label: "Location", Value: loc, Origin: "job", Hint: "from job record", Verdict: ""}
	}
	return model.BandTile{Label: "Location", Hint: "not stated", Verdict: ""}
}

// workModeTile builds the Work mode tile.
// When the profile states a work-mode goal and the vacancy fact is usable, the verdict is
// "ok" (match) or "mismatch" (disjoint), mirroring evaluateWorkModes in quickmatch.
func workModeTile(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, sections []sectionLike) model.BandTile {
	// Structured / model-merged work mode (usable reliability only).
	for _, wm := range facts.Conditions.WorkModes {
		if wm.Evidence.Reliability != matchcontract.ReliabilityUsable {
			continue
		}
		label := workModeLabel(wm.Mode)
		origin := originLabel(wm.Evidence.Field)
		verdict := workModeVerdict(profile, wm.Mode)
		hint := workModeHint(profile, verdict)
		return model.BandTile{Label: "Work mode", Value: label, Origin: origin, Hint: hint, Verdict: verdict}
	}
	// Unreliable fact (NeedsCorroboration): show value but mark not_stated.
	for _, wm := range facts.Conditions.WorkModes {
		label := workModeLabel(wm.Mode)
		origin := originLabel(wm.Evidence.Field)
		return model.BandTile{Label: "Work mode", Value: label, Origin: origin, Hint: "from listing", Verdict: "not_stated"}
	}

	// Weak hint from section text — shown as a hint, never as a confident value.
	for _, s := range sections {
		lower := strings.ToLower(s.TextContent())
		switch {
		case strings.Contains(lower, "hybrid"):
			return model.BandTile{Label: "Work mode", Value: "Hybrid", Origin: "text", Hint: "hint from text", Verdict: "not_stated"}
		case strings.Contains(lower, "remote"):
			return model.BandTile{Label: "Work mode", Value: "Remote", Origin: "text", Hint: "hint from text", Verdict: "not_stated"}
		case strings.Contains(lower, "on-site") || strings.Contains(lower, "in office") || strings.Contains(lower, "in-office"):
			return model.BandTile{Label: "Work mode", Value: "On-site", Origin: "text", Hint: "hint from text", Verdict: "not_stated"}
		}
	}

	return model.BandTile{Label: "Work mode", Hint: "not stated", Verdict: "not_stated"}
}

// workModeVerdict compares the job work mode against the profile goal.
// Returns "ok", "mismatch", or "not_stated" (no profile goal or empty profile modes).
func workModeVerdict(profile matchcontract.ProfileSnapshot, jobMode string) string {
	if len(profile.WorkModes) == 0 {
		return "not_stated"
	}
	jobNorm := strings.ToLower(jobMode)
	for _, m := range profile.WorkModes {
		if strings.ToLower(m) == jobNorm {
			return "ok"
		}
	}
	return "mismatch"
}

// workModeHint returns the tile hint line for a work-mode verdict.
func workModeHint(profile matchcontract.ProfileSnapshot, verdict string) string {
	if len(profile.WorkModes) == 0 || verdict == "not_stated" {
		return "from listing"
	}
	want := workModeLabel(profile.WorkModes[0])
	switch verdict {
	case "ok":
		return fmt.Sprintf("Work mode · you want %s", strings.ToLower(want))
	case "mismatch":
		return fmt.Sprintf("Work mode · you want %s", strings.ToLower(want))
	}
	return "from listing"
}

// contractTile builds the Contract tile.
func contractTile(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, sections []sectionLike, cfg config.VacancyViewConfig) model.BandTile {
	res := ReadContractKind(facts, sections, cfg)
	verdict := engagementVerdict(profile, facts, res)
	hint := contractHint(profile, verdict, res.hint)
	return model.BandTile{Label: "Contract", Value: res.value, Origin: res.origin, Hint: hint, Verdict: verdict}
}

// engagementVerdict compares the job engagement type against the profile goal.
// When the profile has no engagement goal, falls back to the contractResult verdict.
func engagementVerdict(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, res contractResult) string {
	if len(profile.EngagementTypes) == 0 {
		return res.verdict // "not_stated" when no profile goal
	}
	// Use the structured/model fact when available.
	if e := facts.Conditions.Engagement; e != nil && e.Evidence.Reliability == matchcontract.ReliabilityUsable {
		jobType := strings.ToLower(e.Type)
		for _, t := range profile.EngagementTypes {
			if strings.ToLower(t) == jobType {
				return "ok"
			}
		}
		return "mismatch"
	}
	// Fall back to text-parsed value.
	if res.value != "" {
		jobType := strings.ToLower(res.value)
		profileMap := map[string]bool{
			"employment": false,
			"contract":   false,
		}
		for _, t := range profile.EngagementTypes {
			switch strings.ToLower(t) {
			case "employment", "fulltime", "full-time", "full_time":
				profileMap["employment"] = true
			case "contract", "freelance", "contractor":
				profileMap["contract"] = true
			}
		}
		if (jobType == "employment" && profileMap["employment"]) ||
			(jobType == "contract" && profileMap["contract"]) ||
			(jobType == "fixed-term" && profileMap["contract"]) {
			return "ok"
		}
		return "mismatch"
	}
	return "not_stated"
}

// contractHint returns the tile hint line for a contract verdict.
func contractHint(profile matchcontract.ProfileSnapshot, verdict, fallback string) string {
	if len(profile.EngagementTypes) == 0 || verdict == "not_stated" {
		return fallback
	}
	want := profile.EngagementTypes[0]
	return fmt.Sprintf("Contract · you want %s", strings.ToLower(want))
}

func workModeLabel(mode string) string {
	switch mode {
	case matchcontract.WorkRemote:
		return "Remote"
	case matchcontract.WorkHybrid:
		return "Hybrid"
	case matchcontract.WorkOffice:
		return "On-site"
	}
	return mode
}
