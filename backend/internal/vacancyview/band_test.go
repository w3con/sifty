package vacancyview

import (
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
)

func ptr(v float64) *float64 { return &v }

var testRates = map[string]float64{
	"EUR": 1.0,
	"USD": 0.92,
	"GBP": 1.17,
}

// TestParseCashText verifies the salary range extractor on representative inputs.
func TestParseCashText(t *testing.T) {
	cases := []struct {
		text    string
		wantMin *float64
		wantMax *float64
		wantOK  bool
	}{
		{"base salary between $150,000 and $250,000", ptr(150000), ptr(250000), true},
		{"$150,000 - $250,000", ptr(150000), ptr(250000), true},
		{"150k – 200k USD", ptr(150000), ptr(200000), true},
		{"€80,000 – €100,000 gross per year", ptr(80000), ptr(100000), true},
		{"up to $300,000", nil, ptr(300000), true},
		// Real-world: salary range with equity mentioned as an add-on should still parse.
		{"This role offers a base salary between $150,000 and $250,000, plus equity and benefits.", ptr(150000), ptr(250000), true},
		// Noise should not parse.
		{"competitive salary", nil, nil, false},
		{"$10k signing bonus", nil, nil, false},
		{"equity of $50,000", nil, nil, false},
	}
	for _, c := range cases {
		r := parseCashText(c.text)
		if !c.wantOK {
			if r != nil {
				t.Errorf("parseCashText(%q) = non-nil, want nil", c.text)
			}
			continue
		}
		if r == nil {
			t.Errorf("parseCashText(%q) = nil, want non-nil", c.text)
			continue
		}
		if c.wantMin != nil && (r.min == nil || *r.min != *c.wantMin) {
			t.Errorf("parseCashText(%q).min = %v, want %v", c.text, r.min, c.wantMin)
		}
		if c.wantMax != nil && (r.max == nil || *r.max != *c.wantMax) {
			t.Errorf("parseCashText(%q).max = %v, want %v", c.text, r.max, c.wantMax)
		}
	}
}

// TestComputeMoneyResultThreeStates covers the three verdict states and boundary cases.
// Profile minimum: €90,000/year gross.
// USD → EUR: USD * 0.92 / 1.0 = EUR value.
//
//	Lower bound  Upper bound   Expected
//	€138k        €230k         ok     (lower ≥ 90k)
//	€73.6k       €110.4k       unclear (lower < 90k ≤ upper)
//	€73.6k       €73.6k        mismatch (single figure: upper < 90k)
//	€90k         €110.4k       ok     (boundary: lower = minimum exactly)
//	€73.6k       €90k          unclear (boundary: upper = minimum exactly → minimum at top of range)
func TestComputeMoneyResultThreeStates(t *testing.T) {
	profile := matchcontract.ProfileSnapshot{
		ProfileID: "test",
		Compensation: &matchcontract.ProfileCash{
			Minimum:  ptr(90000),
			Currency: "EUR",
			Period:   "year",
			TaxBasis: "gross",
		},
	}
	makeFacts := func(minUSD, maxUSD float64) matchcontract.VacancyFacts {
		f := matchcontract.VacancyFacts{
			Compensation: matchcontract.CompensationFacts{
				Cash: &matchcontract.CashFact{
					Max: ptr(maxUSD), Currency: "USD", Period: "year", TaxBasis: "gross",
					Basis: matchcontract.CashBasisRangeCeiling,
					Evidence: matchcontract.Evidence{
						Field: "test", Section: matchcontract.SectionUnknown,
						Reliability: matchcontract.ReliabilityNeedsCorroboration,
					},
				},
			},
		}
		if minUSD > 0 {
			f.Compensation.Cash.Min = ptr(minUSD)
		}
		return f
	}

	cases := []struct {
		name        string
		minUSD      float64
		maxUSD      float64
		wantVerdict string
	}{
		// Core three states.
		{"lower above min → ok", 150000, 250000, "ok"},
		{"range straddles min → unclear", 80000, 120000, "unclear"},
		{"single figure below min → mismatch", 0, 80000, "mismatch"},
		// Boundary: lower = minimum exactly (90000 / 0.92 ≈ 97826 USD).
		{"lower equals min → ok", 97826.09, 120000, "ok"},
		// Boundary: upper = minimum exactly → minimum at top of range → unclear.
		{"upper equals min → unclear", 80000, 97826.09, "unclear"},
	}
	for _, c := range cases {
		res := ComputeMoneyResult(profile, makeFacts(c.minUSD, c.maxUSD), nil, testRates)
		if res.Verdict != c.wantVerdict {
			t.Errorf("%s: verdict = %q, want %q", c.name, res.Verdict, c.wantVerdict)
		}
	}
}

// TestParseCashTextSkipWords covers D: cashSkipRe word-boundary false positives from the
// coordinator review (ids 1296, 265, 2796 and "remote" containing "ote").
func TestParseCashTextSkipWords(t *testing.T) {
	cases := []struct {
		desc   string
		text   string
		wantOK bool
	}{
		// "ote" substring used to match "remote", "note", etc.
		{"remote contains ote", "This role is remote and pays $150,000", true},
		{"note contains ote", "Please note: salary is $120,000 – $180,000", true},
		// Plain OTE should still block when no salary context.
		{"bare OTE line", "OTE $200,000", false},
		// "equity" should block without salary context.
		{"equity without base", "equity of $500,000", false},
		// "equity" with explicit "base salary" should NOT block.
		{"equity with base salary", "base salary $200,000 – $300,000, plus equity", true},
		// Multi-city range (> 2 amounts): take global min/max.
		{"multi-city range", "salary $120,000 – $150,000 (NYC) or $100,000 – $130,000 (SF)", true},
	}
	for _, c := range cases {
		r := parseCashText(c.text)
		got := r != nil
		if got != c.wantOK {
			t.Errorf("[%s] parseCashText(%q): got non-nil=%v, want %v", c.desc, c.text, got, c.wantOK)
		}
	}
}

// TestReadContractKindFalsePositives covers C: bare-word contract vocabulary was triggering on
// phrases like "government contractor", "permanent resident", "Medical, Dental" and
// "government contractor". The fixed vocabulary requires explicit phrases.
func TestReadContractKindFalsePositives(t *testing.T) {
	cfg, err := config.DefaultVacancyViewConfig()
	if err != nil {
		t.Fatal(err)
	}
	// All sections are classified as "Requirements" so section-class filter does not hide them.
	makeSection := func(text string) []sectionLike {
		return []sectionLike{wrappedSection{model.Section{Class: "Requirements", Content: text}}}
	}

	cases := []struct {
		desc        string
		text        string
		wantVerdictEmpty bool // when true: contractResult.value must be ""
	}{
		// "government contractor" used to trigger Contract.
		{"government contractor", "Must not be a government contractor or federal employee.", true},
		// "permanent resident" used to trigger Employment.
		{"permanent resident", "U.S. lawful, permanent resident or citizen required.", true},
		// "Medical, Dental, and Vision plans" used to trigger Employment.
		{"medical dental vision", "Benefits include Medical, Dental, and Vision plans.", true},
		// Explicit "contract role" should still work.
		{"contract role", "This is a contract role with potential for extension.", false},
		// Explicit "permanent position" should work.
		{"permanent position", "This is a permanent position with full benefits.", false},
		// Explicit "fixed-term contract" should work.
		{"fixed-term contract", "We are offering a fixed-term contract for 12 months.", false},
	}
	for _, c := range cases {
		sections := makeSection(c.text)
		res := ReadContractKind(matchcontract.VacancyFacts{}, sections, cfg)
		if c.wantVerdictEmpty {
			if res.value != "" {
				t.Errorf("[%s] ReadContractKind(%q).value = %q, want empty", c.desc, c.text, res.value)
			}
		} else {
			if res.value == "" {
				t.Errorf("[%s] ReadContractKind(%q).value is empty, want non-empty", c.desc, c.text)
			}
		}
	}
}

// TestFormatShortRange covers A2: compact money display format.
func TestFormatShortRange(t *testing.T) {
	cases := []struct {
		desc string
		fact *matchcontract.CashFact
		want string
	}{
		{"range", &matchcontract.CashFact{Min: ptr(150000), Max: ptr(250000), Currency: "USD"}, "$150–250k"},
		{"max only", &matchcontract.CashFact{Max: ptr(200000), Currency: "USD"}, "$200k"},
		{"min only", &matchcontract.CashFact{Min: ptr(100000), Currency: "EUR"}, "from €100k"},
		{"same value", &matchcontract.CashFact{Min: ptr(200000), Max: ptr(200000), Currency: "GBP"}, "£200k"},
		{"nil", nil, ""},
	}
	for _, c := range cases {
		got := formatShortRange(c.fact)
		if got != c.want {
			t.Errorf("[%s] formatShortRange = %q, want %q", c.desc, got, c.want)
		}
	}
}

// TestCashCurrencyUnknown covers the empty-currency guard: when currency cannot be detected
// (no ISO code, no symbol in value or quote), the tile must show the range neutrally
// (verdict "not_stated") with no approxValue, and convertToProfile must return not-comparable
// so the number is not counted as a match.
func TestCashCurrencyUnknown(t *testing.T) {
	profile := matchcontract.ProfileSnapshot{
		ProfileID: "test",
		Compensation: &matchcontract.ProfileCash{
			Minimum:  ptr(90000),
			Currency: "EUR",
			Period:   "year",
			TaxBasis: "gross",
		},
	}
	max := 280000.0
	min := 230000.0
	facts := matchcontract.VacancyFacts{
		Compensation: matchcontract.CompensationFacts{
			Cash: &matchcontract.CashFact{
				Min: &min, Max: &max,
				Currency: "", Period: "year", TaxBasis: "",
				Basis: matchcontract.CashBasisRangeCeiling,
				Evidence: matchcontract.Evidence{
					Field: "model_extraction", Quote: "230,000-280,000 per year",
					Section: matchcontract.SectionUnknown, Reliability: matchcontract.ReliabilityNeedsCorroboration,
				},
			},
		},
	}
	res := ComputeMoneyResult(profile, facts, nil, testRates)
	if res.Verdict != "not_stated" {
		t.Errorf("unknown currency verdict = %q, want \"not_stated\"", res.Verdict)
	}
	if res.ApproxEUR != "" {
		t.Errorf("unknown currency must have no approxEUR, got %q", res.ApproxEUR)
	}
	if res.ShortRange == "" {
		t.Errorf("unknown currency ShortRange must still display the range")
	}
}

// TestComputeMoneyResultJob950 is the named example from the brief: vacancy 950 with
// "$150,000 – $250,000 USD" and a profile minimum of €90,000. The tile should show ✓
// (ok) because $150,000 * 0.92 = €138,000 ≥ €90,000 (lower bound meets minimum).
func TestComputeMoneyResultJob950(t *testing.T) {
	maxV := 250000.0
	minV := 150000.0
	profile := matchcontract.ProfileSnapshot{
		ProfileID: "test",
		Compensation: &matchcontract.ProfileCash{
			Minimum:  ptr(90000),
			Currency: "EUR",
			Period:   "year",
			TaxBasis: "gross",
		},
	}
	facts := matchcontract.VacancyFacts{
		Compensation: matchcontract.CompensationFacts{
			Cash: &matchcontract.CashFact{
				Min: &minV, Max: &maxV,
				Currency: "USD", Period: "year", TaxBasis: "",
				Basis: matchcontract.CashBasisRangeCeiling,
				Evidence: matchcontract.Evidence{
					Field: "model_extraction", Quote: "base salary between $150,000 and $250,000",
					Section: matchcontract.SectionUnknown, Reliability: matchcontract.ReliabilityNeedsCorroboration,
				},
			},
		},
	}
	res := ComputeMoneyResult(profile, facts, nil, testRates)
	if res.Verdict != "ok" {
		t.Errorf("Job 950 money verdict = %q, want \"ok\"", res.Verdict)
	}
	if res.ApproxEUR == "" {
		t.Errorf("Job 950 money approxEUR is empty, want ≈ €230,000")
	}
	if res.OriginalRange == "" {
		t.Errorf("Job 950 originalRange is empty")
	}
}
