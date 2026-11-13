package quickmatch

import (
	"fmt"
	"math"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// MoneyInput describes the vacancy's stated cash compensation passed to EvaluateMoney.
type MoneyInput struct {
	Min      *float64
	Max      *float64
	Currency string
	Period   string
	TaxBasis string
}

// MoneyOutput is the result of EvaluateMoney.
type MoneyOutput struct {
	ConvertedMin float64
	ConvertedMax float64
	ApproxEUR    string // "≈ €N" or empty when not convertible
	Verdict      string // "ok", "unclear", "mismatch", "not_comparable"
	Reason       string // human-readable; for debugging, not shown in UI directly
}

// EvaluateMoney converts the job's stated cash to the profile currency and derives the
// three-state verdict. It performs all comparisons including tax-basis: when input.TaxBasis and
// profile.TaxBasis differ (after defaulting empty to "gross"), the verdict is "not_comparable".
// When the profile is nil or has no Minimum the verdict is "not_comparable"
// (vacancyview maps this to "not_stated" for display purposes).
func EvaluateMoney(input MoneyInput, profile *matchcontract.ProfileCash, rates map[string]float64) MoneyOutput {
	if input.Max == nil || *input.Max <= 0 {
		return MoneyOutput{Verdict: "not_comparable", Reason: "no salary stated"}
	}

	jobCurrency := strings.ToUpper(strings.TrimSpace(input.Currency))
	jobPeriod := strings.ToLower(strings.TrimSpace(input.Period))
	if jobPeriod == "" && *input.Max >= 10000 {
		jobPeriod = matchcontract.PeriodYear
	}

	approx := approxEURStr(*input.Max, jobCurrency, jobPeriod, rates)

	if profile == nil || profile.Minimum == nil || *profile.Minimum <= 0 {
		return MoneyOutput{ApproxEUR: approx, Verdict: "not_comparable", Reason: "no profile goal"}
	}

	// Tax-basis comparison: empty → gross for both. If they differ the values are not comparable.
	jobTax := strings.ToLower(strings.TrimSpace(input.TaxBasis))
	if jobTax == "" {
		jobTax = matchcontract.TaxGross
	}
	profTax := strings.ToLower(strings.TrimSpace(profile.TaxBasis))
	if profTax == "" {
		profTax = matchcontract.TaxGross
	}
	if !strings.EqualFold(jobTax, profTax) {
		return MoneyOutput{ApproxEUR: approx, Verdict: "not_comparable", Reason: "tax basis differs"}
	}

	profCurrency := strings.ToUpper(strings.TrimSpace(profile.Currency))
	profPeriod := strings.ToLower(strings.TrimSpace(profile.Period))
	if profPeriod == "" {
		profPeriod = matchcontract.PeriodYear
	}

	convertedMax, comparable := ConvertToProfile(*input.Max, jobCurrency, jobPeriod, profCurrency, profPeriod, rates)
	if !comparable {
		return MoneyOutput{ApproxEUR: approx, Verdict: "not_comparable", Reason: "incomparable currencies or periods"}
	}

	convertedMin := convertedMax
	if input.Min != nil && *input.Min > 0 {
		if cm, ok := ConvertToProfile(*input.Min, jobCurrency, jobPeriod, profCurrency, profPeriod, rates); ok {
			convertedMin = cm
		}
	}

	profMin := *profile.Minimum
	var verdict, reason string
	switch {
	case convertedMin >= profMin:
		verdict = "ok"
		reason = "lower bound meets minimum"
	case convertedMax < profMin:
		verdict = "mismatch"
		reason = "upper bound below minimum"
	default:
		verdict = "unclear"
		reason = "minimum lies inside range"
	}

	return MoneyOutput{
		ConvertedMin: convertedMin,
		ConvertedMax: convertedMax,
		ApproxEUR:    approx,
		Verdict:      verdict,
		Reason:       reason,
	}
}

// ConvertToProfile converts jobMax (in jobCurrency per jobPeriod) to the profile's currency
// and period using rates (EUR-relative). An empty jobCurrency is not comparable.
func ConvertToProfile(jobMax float64, jobCurrency, jobPeriod, profCurrency, profPeriod string, rates map[string]float64) (converted float64, ok bool) {
	if jobCurrency == "" {
		return 0, false
	}
	annual := ToAnnual(jobMax, jobPeriod)
	if !isFinite(annual) || annual <= 0 {
		return 0, false
	}
	if jobCurrency != "" && profCurrency != "" && !strings.EqualFold(jobCurrency, profCurrency) {
		if len(rates) == 0 {
			return 0, false
		}
		jobRate, okJ := rates[jobCurrency]
		profRate, okP := rates[profCurrency]
		if !okJ || !okP || jobRate <= 0 || profRate <= 0 {
			return 0, false
		}
		annualEUR := annual * jobRate
		annual = annualEUR / profRate
	}
	result := FromAnnual(annual, profPeriod)
	if !isFinite(result) || result <= 0 {
		return 0, false
	}
	return result, true
}

// ToAnnual converts an amount in the given period to an annual value.
// Empty period is treated as annual. Unknown periods return 0.
func ToAnnual(v float64, period string) float64 {
	switch period {
	case matchcontract.PeriodYear, "":
		return v
	case matchcontract.PeriodMonth:
		return v * 12
	case matchcontract.PeriodHour:
		return v * 2080
	case matchcontract.PeriodDay:
		return v * 260
	}
	return 0
}

// FromAnnual converts an annual amount to the given period.
// Empty or year period returns the annual value unchanged. Unknown periods return 0.
func FromAnnual(annual float64, period string) float64 {
	switch period {
	case matchcontract.PeriodYear, "":
		return annual
	case matchcontract.PeriodMonth:
		return annual / 12
	case matchcontract.PeriodHour:
		return annual / 2080
	case matchcontract.PeriodDay:
		return annual / 260
	}
	return 0
}

// approxEURStr formats a "≈ €N" string. Empty jobCurrency or unknown rate returns "".
func approxEURStr(jobMax float64, jobCurrency, jobPeriod string, rates map[string]float64) string {
	annual := ToAnnual(jobMax, jobPeriod)
	if annual <= 0 {
		return ""
	}
	if jobCurrency == "" {
		return ""
	}
	if strings.EqualFold(jobCurrency, "EUR") {
		return fmt.Sprintf("≈ €%s", moneyFormatAmount(annual))
	}
	if len(rates) == 0 {
		return ""
	}
	jobRate, okJ := rates[jobCurrency]
	eurRate, okE := rates["EUR"]
	if !okJ || !okE || jobRate <= 0 || eurRate <= 0 {
		return ""
	}
	return fmt.Sprintf("≈ €%s", moneyFormatAmount(annual*jobRate/eurRate))
}

// moneyFormatAmount formats a float as a rounded comma-separated integer string.
func moneyFormatAmount(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "?"
	}
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
