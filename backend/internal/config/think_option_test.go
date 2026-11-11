package config_test

import (
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
)

// A provider may set a reasoning effort for a model that cannot switch reasoning off; anything but
// low, medium or high is refused, and the matching stage may name its own provider.
func TestProviderThinkLevelAndMatchingStage(t *testing.T) {
	providers := func(think, matching string) string {
		stage := ""
		if matching != "" {
			stage = "  matching: " + matching + "\n"
		}
		line := ""
		if think != "" {
			line = "    think: " + think + "\n"
		}
		return "version: 1\nstages:\n  assessment: a\n  extraction: a\n" + stage + "providers:\n  a:\n    type: ollama\n    model: m\n    endpoint: http://127.0.0.1:11434\n" + line + "  b:\n    type: ollama\n    model: n\n    endpoint: http://127.0.0.1:11434\n"
	}
	for _, c := range []struct {
		think, matching, wantErr string
	}{
		{"low", "b", ""},
		{"", "", ""},
		{"extreme", "", "think must be low, medium or high"},
		{"", "nope", `stage "matching" names unknown provider`},
	} {
		d := tempDirs(t)
		writeFixture(t, d, "search", validSearch)
		writeFixture(t, d, "runtime", validRuntime)
		writeFixture(t, d, "candidate", validCandidate)
		writeFixture(t, d, "providers", providers(c.think, c.matching))
		cfg, err := config.Load(d)
		if c.wantErr == "" {
			if err != nil {
				t.Errorf("think=%q matching=%q: %v", c.think, c.matching, err)
			} else if c.matching == "b" && cfg.Providers.MatchingStage() != "b" {
				t.Errorf("matching stage = %q, want b", cfg.Providers.MatchingStage())
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("think=%q matching=%q: error %v, want it to contain %q", c.think, c.matching, err, c.wantErr)
		}
	}
}
