// Package matchconfig maps the existing runtime configuration extension into the versioned
// Quick Match domain contract. It is an adapter only: the pure evaluator and extractor never
// import configuration, and no value here originates in a view or a user profile.
package matchconfig

import (
	"fmt"
	"strings"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
)

// Rules converts the effective configuration block into validated evaluator rules.
func Rules(q config.QuickMatch) (matchcontract.Rules, error) {
	r := matchcontract.Rules{
		SchemaVersion:   matchcontract.SchemaVersion,
		RulesVersion:    strings.TrimSpace(q.RulesVersion),
		RequiredWeight:  q.RequiredWeight,
		PreferredWeight: q.PreferredWeight,
		// The midpoint and verdict utilities are versioned policy owned by the contract, not
		// yet a runtime.yaml key; they enter the rules identity, so changing them re-evaluates
		// stored results without any model call.
		UnknownGoalUtility: matchcontract.DefaultUnknownGoalUtility,
		VerdictUtility:     matchcontract.DefaultVerdictUtilities(),
		Weights: map[matchcontract.Group]float64{
			matchcontract.GroupRole:         q.Weights.Role,
			matchcontract.GroupCompensation: q.Weights.Compensation,
			matchcontract.GroupGeography:    q.Weights.Geography,
			matchcontract.GroupConditions:   q.Weights.Conditions,
		},
	}
	if err := r.Validate(); err != nil {
		return matchcontract.Rules{}, fmt.Errorf("quick match configuration: %w", err)
	}
	return r, nil
}

// SourcePolicy converts the effective source-field usability declarations.
func SourcePolicy(q config.QuickMatch) matchcontract.SourcePolicy {
	p := matchcontract.SourcePolicy{
		Version: strings.TrimSpace(q.SourcePolicyVersion),
		Sources: make(map[string]matchcontract.SourcePolicyRule, len(q.SourceFields)),
	}
	for name, rule := range q.SourceFields {
		reliability := matchcontract.Reliability(strings.TrimSpace(rule.Location))
		if reliability != "" && !reliability.Valid() {
			reliability = matchcontract.ReliabilityUnusable
		}
		p.Sources[name] = matchcontract.SourcePolicyRule{Location: reliability}
	}
	return p
}
