package quickmatch

import (
	"math"

	"github.com/w3con/sifty/internal/matchcontract"
)

// evaluateCompensation scores the compensation group. It converts the vacancy's stated
// currency to the profile currency using the rules' exchange rates when the currencies differ.
// Period inference: if period is empty and Max >= 10 000, year is assumed.
// Tax-basis inference: if tax basis is empty it is treated as gross.
// NeedsCorroboration facts proceed through comparison; only Unusable facts are refused.
func evaluateCompensation(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts, rates map[string]float64) *groupEval {
	g := &groupEval{group: matchcontract.GroupCompensation, applicability: matchcontract.NotApplicable}
	p := profile.Compensation
	if p == nil || p.Minimum == nil || *p.Minimum <= 0 {
		return g
	}
	g.applicability = matchcontract.Applicable
	base := member{
		group: matchcontract.GroupCompensation, criterion: matchcontract.CriterionCompensationCash,
		profileField: "compensation.minimum", profileNumber: p.Minimum,
		profileValue: formatCash(*p.Minimum, p.Currency, p.Period, p.TaxBasis, ""),
		jobFact:      "cash", applicability: matchcontract.Applicable, weight: 1,
	}

	c := facts.Compensation.Cash
	if c == nil || c.Max == nil || *c.Max <= 0 {
		base.reason = matchcontract.ReasonCompensationMissing
		g.members = append(g.members, base)
		return g
	}
	base.jobValue = formatCash(*c.Max, c.Currency, c.Period, c.TaxBasis, c.Basis)
	base.jobNumber = c.Max
	base.section = c.Evidence.Section
	base.evidenceField = c.Evidence.Field
	base.evidenceQuote = c.Evidence.Quote
	base.reliability = c.Evidence.Reliability

	if c.Evidence.Reliability == matchcontract.ReliabilityUnusable {
		base.reason = matchcontract.ReasonCompensationUnusable
		g.members = append(g.members, base)
		return g
	}

	// Delegate conversion, tax-basis comparison and three-state verdict to EvaluateMoney.
	// EvaluateMoney handles all of: tax-basis check, period inference, currency conversion.
	out := EvaluateMoney(MoneyInput{
		Min:      c.Min,
		Max:      c.Max,
		Currency: c.Currency,
		Period:   c.Period,
		TaxBasis: c.TaxBasis,
	}, p, rates)

	switch out.Verdict {
	case "not_comparable":
		base.reason = matchcontract.ReasonCompensationIncomparable
		g.members = append(g.members, base)
		return g
	case "ok":
		utility := 1.0
		base.utility = &utility
		base.reason = matchcontract.ReasonCompensationAligned
	case "mismatch":
		utility := math.Min(1, out.ConvertedMax / *p.Minimum)
		if !isFinite(utility) || utility < 0 {
			base.reason = matchcontract.ReasonCompensationIncomparable
			g.members = append(g.members, base)
			return g
		}
		base.utility = &utility
		base.reason = matchcontract.ReasonCompensationBelowMin
	default: // "unclear"
		// Minimum lies inside the range: not decided, treat as no data.
		base.reason = matchcontract.ReasonCompensationRangeCovers
	}
	g.members = append(g.members, base)
	return g
}

func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
