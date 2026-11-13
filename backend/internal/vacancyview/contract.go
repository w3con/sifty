package vacancyview

import (
	"regexp"
	"strings"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
)

// contractResult holds the computed contract tile values.
type contractResult struct {
	value   string // display value, e.g. "Employment" or "Contract"
	origin  string // "structured", "model", or "text"
	hint    string // description line shown in the tile
	verdict string // always "not_stated" (no profile goal for contract yet)
}

// ReadContractKind derives the contract type from merged facts, sections and configuration.
// Order: (1) facts.Conditions.Engagement (from structured + model merge), (2) explicit words
// in Compensation/Benefits/other sections.
func ReadContractKind(facts matchcontract.VacancyFacts, sections []sectionLike, cfg config.VacancyViewConfig) contractResult {
	// 1. Structured / model-merged engagement fact.
	if e := facts.Conditions.Engagement; e != nil {
		switch e.Type {
		case matchcontract.EngagementEmployment:
			return contractResult{value: "Employment", origin: originLabel(e.Evidence.Field), hint: "from listing", verdict: "not_stated"}
		case matchcontract.EngagementContract:
			return contractResult{value: "Contract", origin: originLabel(e.Evidence.Field), hint: "from listing", verdict: "not_stated"}
		}
	}

	// 2. Explicit word search in classified sections only (skip "other" and unclassified).
	// This avoids false positives from boilerplate text in legal/EEO or generic sections.
	lists := cfg.ContractWords
	for _, s := range sections {
		cls := s.Class()
		if cls == "other" || cls == "" {
			continue
		}
		lower := strings.ToLower(s.TextContent())
		for _, w := range lists.Contract {
			if contractPhraseMatch(lower, w) {
				return contractResult{value: "Contract", origin: "text", hint: "from listing text", verdict: "not_stated"}
			}
		}
		for _, w := range lists.Permanent {
			if contractPhraseMatch(lower, w) {
				return contractResult{value: "Employment", origin: "text", hint: "from listing text", verdict: "not_stated"}
			}
		}
		for _, w := range lists.FixedTerm {
			if contractPhraseMatch(lower, w) {
				return contractResult{value: "Fixed-term", origin: "text", hint: "from listing text", verdict: "not_stated"}
			}
		}
	}

	return contractResult{hint: "not stated", verdict: "not_stated"}
}

// contractPhraseMatch reports whether text contains phrase w as a whole-word match.
// Multi-word phrases use a simple substring check; single tokens require word boundaries.
func contractPhraseMatch(text, w string) bool {
	lower := strings.ToLower(w)
	if !strings.Contains(text, lower) {
		return false
	}
	if strings.ContainsRune(lower, ' ') || strings.ContainsRune(lower, '-') {
		return true
	}
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(lower) + `\b`)
	return re.MatchString(text)
}

func originLabel(field string) string {
	switch {
	case strings.Contains(field, "structured"):
		return "structured"
	case strings.Contains(field, "model"):
		return "model"
	}
	return "facts"
}
