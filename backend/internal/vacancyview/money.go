// Package vacancyview computes the Fits you band and What they offer tiles from stored facts,
// classified sections and model extraction. It performs no I/O and calls no model.
package vacancyview

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/quickmatch"
)

// cashResult is the parsed cash value from a text source.
type cashResult struct {
	min      *float64
	max      *float64
	currency string
	period   string
	taxBasis string
}

var amountRe = regexp.MustCompile(`(\d{1,3}(?:[,\s]\d{3})*|\d+(?:\.\d+)?)\s*([kK])?`)
var currencyCodeRe = regexp.MustCompile(`(?i)\b(USD|EUR|GBP|CAD|AUD|CHF|JPY)\b`)
var currencySymbolRe = regexp.MustCompile(`[$€£]`)

// parseCashSkipRe matches whole-word equity/bonus markers to avoid treating equity grants as salary.
// Word boundaries prevent "ote" matching "remote", etc.
var parseCashSkipRe = regexp.MustCompile(`(?i)\b(bonus|equity|stock[- ]option|commission|ote|estimate|survey|allowance|relocation)\b`)

// parseCashText attempts to extract a salary range from a text string.
// Returns nil when no valid amount is found.
func parseCashText(text string) *cashResult {
	// Skip texts whose primary topic is an equity/bonus grant, signing bonus etc., not a salary range.
	// Exception: when the text contains an explicit salary indicator ("salary", "base", "wage"), it
	// describes cash compensation even if equity is mentioned as an add-on (e.g. "base salary …, plus equity").
	lower := strings.ToLower(text)
	hasSalaryCtx := strings.Contains(lower, "salary") || strings.Contains(lower, " base ") || strings.Contains(lower, "wage")
	if !hasSalaryCtx && parseCashSkipRe.MatchString(lower) {
		return nil
	}

	// Extract currency.
	currency := ""
	if m := currencyCodeRe.FindString(text); m != "" {
		currency = strings.ToUpper(m)
	} else if strings.ContainsRune(text, '$') {
		currency = "USD"
	} else if strings.ContainsRune(text, '€') {
		currency = "EUR"
	} else if strings.ContainsRune(text, '£') {
		currency = "GBP"
	}

	matches := amountRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	type parsed struct {
		value float64
		start int
		end   int
	}
	var amounts []parsed
	for _, m := range matches {
		raw := strings.ReplaceAll(strings.ReplaceAll(text[m[2]:m[3]], ",", ""), " ", "")
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil
		}
		if m[4] >= 0 && m[5] >= 0 && strings.EqualFold(text[m[4]:m[5]], "k") {
			n *= 1000
		}
		amounts = append(amounts, parsed{value: n, start: m[0], end: m[1]})
	}
	if len(amounts) == 0 {
		return nil
	}
	// Reject values that look like years (< 1 000) or unrealistically large.
	for _, a := range amounts {
		if a.value < 1000 || a.value > 10_000_000 {
			return nil
		}
	}

	var min, max *float64
	if len(amounts) > 2 {
		// Multiple salary ranges (e.g. by city/location): take the global min and max across all.
		lo, hi := amounts[0].value, amounts[0].value
		for _, a := range amounts[1:] {
			if a.value < lo {
				lo = a.value
			}
			if a.value > hi {
				hi = a.value
			}
		}
		min = &lo
		max = &hi
	} else if len(amounts) == 2 {
		raw := text[amounts[0].end:amounts[1].start]
		// Strip currency symbols, spaces and commas from the separator region to get just
		// the range marker ("-", "–", "—", "to", "and").
		between := strings.ToLower(strings.TrimFunc(raw, func(r rune) bool {
			return r == '$' || r == '€' || r == '£' || r == ' ' || r == '\t' || r == ','
		}))
		switch between {
		case "-", "–", "—", "to", "and":
			lo, hi := amounts[0].value, amounts[1].value
			if lo > hi {
				lo, hi = hi, lo
			}
			min = &lo
			max = &hi
		default:
			// Not a recognized range separator; take the second amount as ceiling only.
			v := amounts[1].value
			max = &v
		}
	} else {
		v := amounts[0].value
		max = &v
	}

	period := cashPeriod(lower)
	tax := ""
	if strings.Contains(lower, "gross") {
		tax = matchcontract.TaxGross
	} else if strings.Contains(lower, "net") {
		tax = matchcontract.TaxNet
	}

	return &cashResult{min: min, max: max, currency: currency, period: period, taxBasis: tax}
}

// cashPeriod reads an explicit period keyword from a lowercased string.
func cashPeriod(lower string) string {
	switch {
	case strings.Contains(lower, "/yr") || strings.Contains(lower, "/year") ||
		strings.Contains(lower, " per year") || strings.Contains(lower, " annually") ||
		strings.Contains(lower, " annual"):
		return matchcontract.PeriodYear
	case strings.Contains(lower, "/mo") || strings.Contains(lower, "/month") ||
		strings.Contains(lower, " per month") || strings.Contains(lower, " monthly"):
		return matchcontract.PeriodMonth
	case strings.Contains(lower, "/hr") || strings.Contains(lower, "/hour") ||
		strings.Contains(lower, " per hour") || strings.Contains(lower, " hourly"):
		return matchcontract.PeriodHour
	}
	return ""
}

// MoneyResult holds the computed money tile values.
type MoneyResult struct {
	OriginalRange string  // e.g. "$150,000 – $250,000"
	ShortRange    string  // e.g. "$150–250k" for the tile value
	ApproxEUR     string  // e.g. "≈ €230,000" (empty when not convertible)
	Verdict       string  // "ok", "unclear", "mismatch", "not_stated"
	Currency      string
	Period        string  // e.g. "year"
	Max           float64
}

// ComputeMoneyResult derives the money tile from merged facts and configuration.
// It reads only from mergedFacts.Compensation.Cash (the same source the number uses)
// so that tile verdict and Fits you group verdict are always consistent.
// The sections parameter is retained for API compatibility but no longer used here.
// The profile minimum is compared after converting to the profile currency via EvaluateMoney.
// "not_comparable" from EvaluateMoney is mapped to "not_stated" for display purposes.
func ComputeMoneyResult(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, _ []sectionLike, rates map[string]float64) MoneyResult {
	cash := facts.Compensation.Cash
	if cash == nil || cash.Max == nil || *cash.Max <= 0 {
		return MoneyResult{Verdict: "not_stated"}
	}

	// Build display ranges.
	displayRange := formatCashRange(cash)
	shortRange := formatShortRange(cash)
	currency := strings.ToUpper(strings.TrimSpace(cash.Currency))
	period := strings.ToLower(strings.TrimSpace(cash.Period))

	// Delegate conversion and three-state verdict to the shared EvaluateMoney.
	// Tax-basis comparison is not done here (vacancyview shows the tile neutrally when bases differ).
	out := quickmatch.EvaluateMoney(quickmatch.MoneyInput{
		Min:      cash.Min,
		Max:      cash.Max,
		Currency: cash.Currency,
		Period:   cash.Period,
		TaxBasis: cash.TaxBasis,
	}, profile.Compensation, rates)

	// Map "not_comparable" (no profile goal, incomparable currency/period) to "not_stated".
	verdict := out.Verdict
	if verdict == "not_comparable" {
		verdict = "not_stated"
	}

	return MoneyResult{
		OriginalRange: displayRange,
		ShortRange:    shortRange,
		ApproxEUR:     out.ApproxEUR,
		Verdict:       verdict,
		Currency:      currency,
		Period:        period,
		Max:           *cash.Max,
	}
}

// formatShortRange builds a compact display string like "$150–250k".
// Amounts are rounded to the nearest thousand and expressed in thousands (k).
func formatShortRange(c *matchcontract.CashFact) string {
	if c == nil {
		return ""
	}
	sym := currencySymbol(c.Currency)
	round := func(v float64) float64 { return math.Round(v/1000) * 1 } // result in thousands
	if c.Min != nil && c.Max != nil {
		lo := round(*c.Min)
		hi := round(*c.Max)
		if lo == hi {
			return fmt.Sprintf("%s%gk", sym, lo)
		}
		return fmt.Sprintf("%s%g–%gk", sym, lo, hi)
	}
	if c.Max != nil {
		return fmt.Sprintf("%s%gk", sym, round(*c.Max))
	}
	if c.Min != nil {
		return fmt.Sprintf("from %s%gk", sym, round(*c.Min))
	}
	return ""
}

// formatCashRange builds a display string for the cash range.
func formatCashRange(c *matchcontract.CashFact) string {
	sym := currencySymbol(c.Currency)
	if c.Min != nil && c.Max != nil {
		return fmt.Sprintf("%s%s – %s%s", sym, formatAmount(*c.Min), sym, formatAmount(*c.Max))
	}
	if c.Max != nil {
		return fmt.Sprintf("%s%s", sym, formatAmount(*c.Max))
	}
	if c.Min != nil {
		return fmt.Sprintf("from %s%s", sym, formatAmount(*c.Min))
	}
	return ""
}

func currencySymbol(code string) string {
	switch strings.ToUpper(code) {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	default:
		if code != "" {
			return code + " "
		}
		return ""
	}
}

func formatAmount(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "?"
	}
	// Round to nearest thousand and format with comma separators.
	rounded := math.Round(v/1000) * 1000
	if rounded >= 1000 {
		millions := int(rounded) / 1_000_000
		thousands := (int(rounded) % 1_000_000) / 1000
		hundreds := int(rounded) % 1000
		if millions > 0 {
			return fmt.Sprintf("%d,%03d,%03d", millions, thousands, hundreds)
		}
		if thousands > 0 {
			return fmt.Sprintf("%d,%03d", thousands, hundreds)
		}
	}
	return fmt.Sprintf("%d", int(rounded))
}

