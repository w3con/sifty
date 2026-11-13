package quickmatch_test

import (
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/quickmatch"
	"github.com/w3con/sifty/internal/vacancyview"
)

func pf(v float64) *float64 { return &v }

var moneyRates = map[string]float64{
	"EUR": 1.0,
	"USD": 0.92,
	"GBP": 1.17,
	"CAD": 0.68,
}

func euroProfile(min float64) *matchcontract.ProfileCash {
	return &matchcontract.ProfileCash{Minimum: pf(min), Currency: "EUR", Period: "year", TaxBasis: "gross"}
}

// TestEvaluateMoneyThreeStates covers the core ok/unclear/mismatch verdicts.
// Profile minimum €90,000. USD → EUR: * 0.92.
//
//	$150k–$250k → €138k–€230k → ok   (lower ≥ 90k)
//	$80k–$120k  → €73.6k–€110.4k → unclear (lower < 90k ≤ upper)
//	$80k only   → €73.6k–€73.6k → mismatch (single figure; upper < 90k)
func TestEvaluateMoneyThreeStates(t *testing.T) {
	cases := []struct {
		name    string
		minUSD  *float64
		maxUSD  float64
		want    string
	}{
		{"lower meets min → ok", pf(150000), 250000, "ok"},
		{"range straddles min → unclear", pf(80000), 120000, "unclear"},
		{"single figure below min → mismatch", nil, 80000, "mismatch"},
		// Boundary: lower = minimum exactly (90000 / 0.92 ≈ 97826 USD).
		{"lower equals min → ok", pf(97826.09), 120000, "ok"},
	}
	for _, c := range cases {
		out := quickmatch.EvaluateMoney(quickmatch.MoneyInput{
			Min: c.minUSD, Max: pf(c.maxUSD), Currency: "USD", Period: "year",
		}, euroProfile(90000), moneyRates)
		if out.Verdict != c.want {
			t.Errorf("[%s] verdict = %q, want %q", c.name, out.Verdict, c.want)
		}
	}
}

// TestEvaluateMoneyPeriodConversion verifies monthly job salary is annualised correctly.
// $8,333/month → $99,996/year → €91,996 → ok against €90k minimum.
func TestEvaluateMoneyPeriodConversion(t *testing.T) {
	out := quickmatch.EvaluateMoney(quickmatch.MoneyInput{
		Min: pf(8333), Max: pf(8333), Currency: "USD", Period: matchcontract.PeriodMonth,
	}, euroProfile(90000), moneyRates)
	if out.Verdict != "ok" {
		t.Errorf("monthly conversion verdict = %q, want \"ok\"; convertedMax=%g", out.Verdict, out.ConvertedMax)
	}
}

// TestEvaluateMoneyUnknownCurrency verifies that an empty currency returns "not_comparable"
// and that ApproxEUR is empty.
func TestEvaluateMoneyUnknownCurrency(t *testing.T) {
	out := quickmatch.EvaluateMoney(quickmatch.MoneyInput{
		Min: pf(230000), Max: pf(280000), Currency: "", Period: "year",
	}, euroProfile(90000), moneyRates)
	if out.Verdict != "not_comparable" {
		t.Errorf("unknown currency verdict = %q, want \"not_comparable\"", out.Verdict)
	}
	if out.ApproxEUR != "" {
		t.Errorf("unknown currency ApproxEUR = %q, want empty", out.ApproxEUR)
	}
}

// TestEvaluateMoneyUSDByRate verifies USD→EUR conversion: $200k * 0.92 = €184k → ok vs €90k.
func TestEvaluateMoneyUSDByRate(t *testing.T) {
	out := quickmatch.EvaluateMoney(quickmatch.MoneyInput{
		Max: pf(200000), Currency: "USD", Period: "year",
	}, euroProfile(90000), moneyRates)
	if out.Verdict != "ok" {
		t.Errorf("USD by rate verdict = %q, want \"ok\"; convertedMax=%g", out.Verdict, out.ConvertedMax)
	}
	if !strings.HasPrefix(out.ApproxEUR, "≈ €") {
		t.Errorf("ApproxEUR = %q, want \"≈ €…\"", out.ApproxEUR)
	}
}

// TestEvaluateMoneyNoProfile verifies that a nil profile returns "not_comparable".
func TestEvaluateMoneyNoProfile(t *testing.T) {
	out := quickmatch.EvaluateMoney(quickmatch.MoneyInput{
		Max: pf(200000), Currency: "USD", Period: "year",
	}, nil, moneyRates)
	if out.Verdict != "not_comparable" {
		t.Errorf("nil profile verdict = %q, want \"not_comparable\"", out.Verdict)
	}
	// ApproxEUR should still be computed even without a profile.
	if !strings.HasPrefix(out.ApproxEUR, "≈ €") {
		t.Errorf("nil profile ApproxEUR = %q, want \"≈ €…\"", out.ApproxEUR)
	}
}

// TestEvaluateMoneyEmptyInput verifies that nil Max returns "not_comparable".
func TestEvaluateMoneyEmptyInput(t *testing.T) {
	out := quickmatch.EvaluateMoney(quickmatch.MoneyInput{}, euroProfile(90000), moneyRates)
	if out.Verdict != "not_comparable" {
		t.Errorf("nil Max verdict = %q, want \"not_comparable\"", out.Verdict)
	}
}

// TestToAnnual verifies period conversions to annual (moved from vacancyview; calls exported function).
func TestToAnnual(t *testing.T) {
	cases := []struct {
		value  float64
		period string
		want   float64
	}{
		{100000, matchcontract.PeriodYear, 100000},
		{100000, "", 100000},
		{8333, matchcontract.PeriodMonth, 99996},
		{50, matchcontract.PeriodHour, 104000},
		{400, matchcontract.PeriodDay, 104000},
		{100000, "unknown", 0},
	}
	for _, c := range cases {
		got := quickmatch.ToAnnual(c.value, c.period)
		if got != c.want {
			t.Errorf("ToAnnual(%g, %q) = %g, want %g", c.value, c.period, got, c.want)
		}
	}
}

// TestConvertToProfile verifies cross-currency conversion (moved from vacancyview; calls exported function).
func TestConvertToProfile(t *testing.T) {
	cases := []struct {
		jobMax       float64
		jobCurrency  string
		jobPeriod    string
		profCurrency string
		profPeriod   string
		wantOK       bool
		wantApprox   float64 // approximate; checked within 1%
	}{
		// Same currency: no conversion.
		{100000, "EUR", "year", "EUR", "year", true, 100000},
		// USD to EUR: 100000 * 0.92 / 1.0 = 92000.
		{100000, "USD", "year", "EUR", "year", true, 92000},
		// USD to EUR, monthly profile: 92000 / 12 ≈ 7666.
		{100000, "USD", "year", "EUR", "month", true, 7666.67},
		// Missing rate for unknown currency.
		{100000, "XYZ", "year", "EUR", "year", false, 0},
		// Empty rates map: incomparable.
		{100000, "USD", "year", "EUR", "year", false, 0},
	}
	for i, c := range cases {
		rates := moneyRates
		if i == len(cases)-1 {
			rates = nil
		}
		got, ok := quickmatch.ConvertToProfile(c.jobMax, c.jobCurrency, c.jobPeriod, c.profCurrency, c.profPeriod, rates)
		if ok != c.wantOK {
			t.Errorf("case %d: ConvertToProfile ok=%v, want %v", i, ok, c.wantOK)
			continue
		}
		if !c.wantOK {
			continue
		}
		diff := got - c.wantApprox
		if diff < 0 {
			diff = -diff
		}
		if diff > c.wantApprox*0.01 {
			t.Errorf("case %d: ConvertToProfile = %g, want ≈ %g", i, got, c.wantApprox)
		}
	}
}

// TestMoneyTileAndNumberCoherence verifies that for every case the tile verdict
// (vacancyview.ComputeMoneyResult) and the EvaluateMoney verdict (which evaluateCompensation
// also uses) produce consistent outcomes.
// Consistency rule: EvaluateMoney.Verdict maps tile verdict as follows:
//
//	"ok"             → tile "ok"             / number utility = 1
//	"mismatch"       → tile "mismatch"       / number utility < 1
//	"unclear"        → tile "unclear"        / number group unknown
//	"not_comparable" → tile "not_stated"     / number group unknown
//
// Profile: €90,000/year/gross. Rates: USD*0.92 = EUR.
func TestMoneyTileAndNumberCoherence(t *testing.T) {
	prof := matchcontract.ProfileSnapshot{
		ProfileID: "coh-test",
		Compensation: &matchcontract.ProfileCash{
			Minimum: pf(90000), Currency: "EUR", Period: "year", TaxBasis: "gross",
		},
	}

	type cas struct {
		name       string
		cash       matchcontract.CashFact
		wantEM     string // EvaluateMoney verdict
		wantTile   string // ComputeMoneyResult verdict (not_comparable → not_stated)
	}

	mkCash := func(minV, maxV float64, currency, period, taxBasis string) matchcontract.CashFact {
		c := matchcontract.CashFact{
			Max: pf(maxV), Currency: currency, Period: period, TaxBasis: taxBasis,
			Basis: matchcontract.CashBasisRangeCeiling,
			Evidence: matchcontract.Evidence{
				Field:       "test",
				Section:     matchcontract.SectionUnknown,
				Reliability: matchcontract.ReliabilityNeedsCorroboration,
			},
		}
		if minV > 0 {
			c.Min = pf(minV)
		}
		return c
	}

	cases := []cas{
		// EUR annual, above minimum → ok.
		{"EUR annual ok", mkCash(150000, 200000, "EUR", "year", "gross"), "ok", "ok"},
		// USD annual, above minimum (200000*0.92=184000 > 90000) → ok.
		{"USD annual ok", mkCash(0, 200000, "USD", "year", "gross"), "ok", "ok"},
		// USD annual, single figure below minimum (70000*0.92=64400 < 90000) → mismatch.
		{"USD mismatch", mkCash(0, 70000, "USD", "year", "gross"), "mismatch", "mismatch"},
		// Monthly, annualises to above minimum (8500*12=102000 > 90000) → ok.
		{"EUR monthly ok", mkCash(0, 8500, "EUR", matchcontract.PeriodMonth, "gross"), "ok", "ok"},
		// Hourly, annualises to below minimum (40*2080=83200 < 90000) → mismatch.
		{"EUR hourly mismatch", mkCash(0, 40, "EUR", matchcontract.PeriodHour, "gross"), "mismatch", "mismatch"},
		// Range where lower < min ≤ upper → unclear.
		{"unclear range", mkCash(70000, 110000, "EUR", "year", "gross"), "unclear", "unclear"},
		// Lower bound equals minimum exactly → ok.
		{"boundary equal min", mkCash(90000, 110000, "EUR", "year", "gross"), "ok", "ok"},
		// Unknown currency (not in rates) → not_comparable.
		{"unknown currency", mkCash(0, 150000, "XYZ", "year", "gross"), "not_comparable", "not_stated"},
		// Empty currency → not_comparable.
		{"empty currency", mkCash(0, 150000, "", "year", "gross"), "not_comparable", "not_stated"},
		// Net vs gross (profile is gross, job is net) → not_comparable.
		{"net vs gross", mkCash(0, 150000, "EUR", "year", "net"), "not_comparable", "not_stated"},
	}

	for _, c := range cases {
		// EvaluateMoney verdict.
		em := quickmatch.EvaluateMoney(quickmatch.MoneyInput{
			Min: c.cash.Min, Max: c.cash.Max,
			Currency: c.cash.Currency, Period: c.cash.Period, TaxBasis: c.cash.TaxBasis,
		}, prof.Compensation, moneyRates)
		if em.Verdict != c.wantEM {
			t.Errorf("[%s] EvaluateMoney verdict = %q, want %q", c.name, em.Verdict, c.wantEM)
		}

		// Tile verdict via ComputeMoneyResult (no sections = no fallback).
		facts := matchcontract.VacancyFacts{
			Compensation: matchcontract.CompensationFacts{Cash: &c.cash},
		}
		tile := vacancyview.ComputeMoneyResult(prof, facts, nil, moneyRates)
		if tile.Verdict != c.wantTile {
			t.Errorf("[%s] tile verdict = %q, want %q", c.name, tile.Verdict, c.wantTile)
		}

		// Coherence: EvaluateMoney drives both; not_comparable maps to not_stated for tile.
		expectedTile := em.Verdict
		if expectedTile == "not_comparable" {
			expectedTile = "not_stated"
		}
		if tile.Verdict != expectedTile {
			t.Errorf("[%s] incoherent: tile=%q, EvaluateMoney=%q (mapped=%q)",
				c.name, tile.Verdict, em.Verdict, expectedTile)
		}
	}
}
