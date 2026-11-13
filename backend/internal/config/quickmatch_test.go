package config

import "testing"

func TestQuickMatchWeightsNeedNotSumToOne(t *testing.T) {
	q := DefaultQuickMatch()
	if sum := q.Weights.Sum(); sum < 0.69 || sum > 0.71 {
		t.Fatalf("default goal weights sum to %v, want 0.70 (role 25, pay 20, location 15, conditions 10)", sum)
	}
	if err := validateQuickMatch(q); err != nil {
		t.Fatalf("default block refused: %v", err)
	}
	q.Weights.Role = 0.5
	if err := validateQuickMatch(q); err != nil {
		t.Fatalf("a block whose weights do not sum to 1 must be valid: %v", err)
	}
	q.Weights.Conditions = 0
	if err := validateQuickMatch(q); err == nil {
		t.Fatal("a zero goal weight must be refused")
	}
}

func TestQuickMatchModelPassesDefaultsAndLimits(t *testing.T) {
	q := DefaultQuickMatch()
	m := q.ModelPasses
	if m.EffectiveMaxRequirements() != DefaultMaxRequirements || m.EffectiveMaxAttempts() != DefaultMaxAttempts ||
		m.EffectiveTemperature() != DefaultTemperature || m.AllowPaidBulkProvider {
		t.Fatalf("unset model passes must select the documented defaults and never allow a paid provider: %+v", m)
	}
	hot := 3.0
	for name, mp := range map[string]QuickMatchModelPasses{
		"negative cap":     {MaxRequirements: -1},
		"too many retries": {MaxAttempts: 11},
		"hot temperature":  {Temperature: &hot},
	} {
		q := DefaultQuickMatch()
		q.ModelPasses = mp
		if err := validateQuickMatch(q); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
