package config

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/w3con/sifty/internal/dirs"
)

func runtimeFixture(t *testing.T, modelPasses string) dirs.Dirs {
	t.Helper()
	d := dirs.Dirs{Root: t.TempDir()}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	body := "version: 1\nport: 8081\n# a comment that must survive\nquick_match:\n  version: 1\n  rules_version: quickmatch.rules/2\n  source_policy_version: source-policy/1\n" +
		"  weights:\n    role: 0.25\n    compensation: 0.20\n    geography: 0.15\n    conditions: 0.10\n  required_weight: 2\n  preferred_weight: 1\n" + modelPasses + "  source_fields:\n    Wellfound:\n      location: needs_corroboration\n"
	if err := os.WriteFile(d.ConfigFile("runtime"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSetModelPassOrder(t *testing.T) {
	t.Run("replaces an existing line and keeps the rest", func(t *testing.T) {
		d := runtimeFixture(t, "  model_passes:\n    max_attempts: 3\n    order: fit\n    daily_limit: 0\n")
		if err := SetModelPassOrder(d, OrderNewest); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(d.ConfigFile("runtime"))
		if !strings.Contains(string(got), "    order: newest\n") || strings.Contains(string(got), "order: fit") ||
			!strings.Contains(string(got), "# a comment that must survive") || !strings.Contains(string(got), "    daily_limit: 0\n") {
			t.Fatalf("file after edit:\n%s", got)
		}
		var rt Runtime
		if err := yaml.Unmarshal(got, &rt); err != nil {
			t.Fatalf("saved file does not parse: %v", err)
		}
		if o := rt.EffectiveQuickMatch().ModelPasses.EffectiveOrder(); o != OrderNewest {
			t.Fatalf("loaded order = %q", o)
		}
	})
	t.Run("adds the line when absent", func(t *testing.T) {
		d := runtimeFixture(t, "  model_passes:\n    max_attempts: 3\n")
		if err := SetModelPassOrder(d, OrderFit); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(d.ConfigFile("runtime"))
		if !strings.Contains(string(got), "  model_passes:\n    order: fit\n    max_attempts: 3\n") {
			t.Fatalf("file after edit:\n%s", got)
		}
	})
	t.Run("creates the block in an older file", func(t *testing.T) {
		d := runtimeFixture(t, "")
		if err := SetModelPassOrder(d, OrderNewest); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(d.ConfigFile("runtime"))
		if !strings.Contains(string(got), "quick_match:\n  model_passes:\n    order: newest\n  version: 1\n") {
			t.Fatalf("file after edit:\n%s", got)
		}
	})
	t.Run("refuses an unknown order", func(t *testing.T) {
		d := runtimeFixture(t, "  model_passes:\n    max_attempts: 3\n")
		if err := SetModelPassOrder(d, "random"); err == nil {
			t.Error("an unknown order was accepted")
		}
	})
	t.Run("writes the default quick_match block into a file that has none", func(t *testing.T) {
		d := runtimeFixture(t, "")
		if err := os.WriteFile(d.ConfigFile("runtime"), []byte("version: 1\nport: 8081\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := SetModelPassOrder(d, OrderNewest); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(d.ConfigFile("runtime"))
		var rt Runtime
		if err := yaml.Unmarshal(got, &rt); err != nil {
			t.Fatalf("saved file does not parse: %v\n%s", err, got)
		}
		def, qm := DefaultQuickMatch(), rt.EffectiveQuickMatch()
		if qm.ModelPasses.EffectiveOrder() != OrderNewest || qm.Weights != def.Weights || qm.RequiredWeight != def.RequiredWeight {
			t.Fatalf("quick_match after edit = %+v, want the defaults with order newest\n%s", qm, got)
		}
	})
	t.Run("an unset or unknown stored order reads as best fit", func(t *testing.T) {
		if (QuickMatchModelPasses{}).EffectiveOrder() != OrderFit || (QuickMatchModelPasses{Order: "x"}).EffectiveOrder() != OrderFit {
			t.Error("EffectiveOrder must default to fit")
		}
	})
}
