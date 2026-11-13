package quickmatch

import (
	"math"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

func evaluateCompensation(profile matchcontract.ProfileSnapshot, facts matchcontract.VacancyFacts) *groupEval {
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

	if c.Evidence.Reliability != matchcontract.ReliabilityUsable {
		base.reason = matchcontract.ReasonCompensationUnusable
		g.members = append(g.members, base)
		return g
	}
	if p.Currency == "" || p.Period == "" || p.TaxBasis == "" ||
		c.Currency == "" || c.Period == "" || c.TaxBasis == "" ||
		!strings.EqualFold(p.Currency, c.Currency) ||
		!strings.EqualFold(p.Period, c.Period) ||
		!strings.EqualFold(p.TaxBasis, c.TaxBasis) {
		base.reason = matchcontract.ReasonCompensationIncomparable
		g.members = append(g.members, base)
		return g
	}

	utility := math.Min(1, *c.Max / *p.Minimum)
	if !isFinite(utility) || utility < 0 {
		base.reason = matchcontract.ReasonCompensationIncomparable
		g.members = append(g.members, base)
		return g
	}
	base.utility = &utility
	base.reason = matchcontract.ReasonCompensationAligned
	if utility < 1 {
		base.reason = matchcontract.ReasonCompensationBelowMin
	}
	g.members = append(g.members, base)
	return g
}

func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
