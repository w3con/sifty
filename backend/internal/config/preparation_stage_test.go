package config_test

import (
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
)

// TestPreparationStageValidation verifies that:
//   - stages.preparation naming a defined provider validates (config.Load succeeds)
//   - stages.preparation naming an unknown provider is rejected (config.Load errors)
//   - stages.preparation absent or empty is accepted (config.Load succeeds)
func TestPreparationStageValidation(t *testing.T) {
	base := func(providerYAML string) func(t *testing.T) dirs.Dirs {
		return func(t *testing.T) dirs.Dirs {
			d := tempDirs(t)
			writeFixture(t, d, "search", validSearch)
			writeFixture(t, d, "runtime", validRuntime)
			writeFixture(t, d, "candidate", validCandidate)
			writeFixture(t, d, "providers", providerYAML)
			return d
		}
	}

	cases := []struct {
		label     string
		providers string
		wantErr   bool
		errContains string
	}{
		{
			label: "preparation names a defined provider — valid",
			providers: `version: 1
stages:
  assessment: primary
  extraction: primary
  preparation: primary
providers:
  primary:
    type: ollama
    model: llama3
    endpoint: http://127.0.0.1:11434
`,
			wantErr: false,
		},
		{
			label: "preparation names an unknown provider — rejected",
			providers: `version: 1
stages:
  assessment: primary
  extraction: primary
  preparation: nonexistent
providers:
  primary:
    type: ollama
    model: llama3
    endpoint: http://127.0.0.1:11434
`,
			wantErr:     true,
			errContains: "preparation",
		},
		{
			label: "preparation is absent (unset) — valid",
			providers: validProviders,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			d := base(tc.providers)(t)
			_, err := config.Load(d)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Load: expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("Load: error %q does not contain %q", err.Error(), tc.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("Load: unexpected error: %v", err)
				}
			}
		})
	}
}

// TestSavePreparationStageValidation verifies that config.Save enforces the same rules.
func TestSavePreparationStageValidation(t *testing.T) {
	cases := []struct {
		label     string
		providers string
		wantErr   bool
		errContains string
	}{
		{
			label: "preparation names a defined provider — valid",
			providers: `version: 1
stages:
  assessment: primary
  extraction: primary
  preparation: primary
providers:
  primary:
    type: ollama
    model: llama3
    endpoint: http://127.0.0.1:11434
`,
			wantErr: false,
		},
		{
			label: "preparation names an unknown provider — rejected",
			providers: `version: 1
stages:
  assessment: primary
  extraction: primary
  preparation: nonexistent
providers:
  primary:
    type: ollama
    model: llama3
    endpoint: http://127.0.0.1:11434
`,
			wantErr:     true,
			errContains: "preparation",
		},
		{
			label: "preparation absent — valid",
			providers: validProviders,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			d := tempDirs(t)
			err := config.Save(d, "providers", tc.providers)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Save: expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("Save: error %q does not contain %q", err.Error(), tc.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("Save: unexpected error: %v", err)
				}
			}
		})
	}
}
