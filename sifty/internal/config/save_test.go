package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
)

func tempDirs(t *testing.T) dirs.Dirs {
	t.Helper()
	d := dirs.Dirs{Root: t.TempDir()}
	if err := os.MkdirAll(d.Config(), 0o700); err != nil {
		t.Fatalf("MkdirAll config: %v", err)
	}
	return d
}

func writeFixture(t *testing.T, d dirs.Dirs, name, content string) {
	t.Helper()
	if err := os.WriteFile(d.ConfigFile(name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

const validSearch = `version: 1
sites:
  - name: LinkedIn
    url: https://linkedin.com/jobs
search:
  titles: [CTO]
  keywords: [golang]
`

const validRuntime = `version: 1
port: 7373
max_pages: 5
max_jobs_per_source: 100
`

const validProviders = `version: 1
stages:
  assessment: primary
  extraction: primary
providers:
  primary:
    type: ollama
    model: llama3
    endpoint: http://127.0.0.1:11434
`

const validCandidate = `version: 1
cache_hours: 24
files: []
profiles: []
`

const validNetwork = `version: 1
profiles: {}
sources: {}
`

// TestReadRoundTrip verifies that each of the five config files reads back verbatim.
func TestReadRoundTrip(t *testing.T) {
	fixtures := map[string]string{
		"search":    validSearch,
		"runtime":   validRuntime,
		"providers": validProviders,
		"candidate": validCandidate,
		"network":   validNetwork,
	}

	d := tempDirs(t)
	for name, content := range fixtures {
		writeFixture(t, d, name, content)

		got, err := config.Read(d, name)
		if err != nil {
			t.Errorf("Read(%q): %v", name, err)
			continue
		}
		if got != content {
			t.Errorf("Read(%q): got %q, want %q", name, got, content)
		}
	}
}

// TestReadUnknownName verifies that Read rejects an unrecognised config name.
func TestReadUnknownName(t *testing.T) {
	d := tempDirs(t)
	if _, err := config.Read(d, "custom"); err == nil {
		t.Error("Read(custom): expected error, got nil")
	}
}

// TestSaveValidConfigs verifies that Save accepts valid YAML for each config.
func TestSaveValidConfigs(t *testing.T) {
	fixtures := map[string]string{
		"search":    validSearch,
		"runtime":   validRuntime,
		"providers": validProviders,
		"candidate": validCandidate,
		"network":   validNetwork,
	}

	d := tempDirs(t)
	for name, content := range fixtures {
		// Write a seed file so backup logic can run (but even without it should succeed).
		writeFixture(t, d, name, "version: 0\n")
		if err := config.Save(d, name, content); err != nil {
			t.Errorf("Save(%q): unexpected error: %v", name, err)
			continue
		}
		got, err := os.ReadFile(d.ConfigFile(name))
		if err != nil {
			t.Errorf("Read back %q: %v", name, err)
			continue
		}
		if string(got) != content {
			t.Errorf("%q: saved content does not match", name)
		}
	}
}

// TestSaveRejectsSecrets verifies that Save refuses content that looks like an inline secret.
func TestSaveRejectsSecrets(t *testing.T) {
	d := tempDirs(t)
	credentialScheme := "https"
	credentialUser := "user"
	credentialPassword := "password"
	credentialURL := credentialScheme + "://" + credentialUser + ":" + credentialPassword + "@proxy.example.com"
	cases := []struct {
		name    string
		content string
		label   string
	}{
		{
			name: "providers",
			content: validProviders + `    api_key: sk-xxxxxxxxxxxx
`,
			label: "OpenAI key",
		},
		{
			name:    "providers",
			content: validProviders + "    note: Bearer mytoken123\n",
			label:   "Bearer token",
		},
		{
			name:    "providers",
			content: validProviders + "    endpoint: " + credentialURL + "\n",
			label:   "URL with credentials",
		},
	}
	for _, tc := range cases {
		if err := config.Save(d, tc.name, tc.content); err == nil {
			t.Errorf("Save with %s: expected secret rejection error, got nil", tc.label)
		}
	}
}

// TestSaveRejectsInvalidYAML verifies that Save refuses unparseable YAML.
func TestSaveRejectsInvalidYAML(t *testing.T) {
	d := tempDirs(t)
	if err := config.Save(d, "runtime", ":\t:invalid: yaml\n"); err == nil {
		t.Error("Save(invalid YAML): expected error, got nil")
	}
}

// TestSaveRejectsSchemaViolation verifies that Save refuses config that fails schema validation.
func TestSaveRejectsSchemaViolation(t *testing.T) {
	d := tempDirs(t)
	cases := []struct {
		name    string
		content string
		label   string
	}{
		{
			name:    "search",
			content: "version: 2\nsites:\n  - name: X\n    url: https://x.com\n",
			label:   "search: wrong version",
		},
		{
			name:    "search",
			content: "version: 1\nsites: []\n",
			label:   "search: empty sites",
		},
		{
			name:    "runtime",
			content: "version: 1\nport: 80\n",
			label:   "runtime: port below 1024",
		},
		{
			name:    "candidate",
			content: "version: 1\ncache_hours: 0\n",
			label:   "candidate: zero cache_hours",
		},
	}
	for _, tc := range cases {
		if err := config.Save(d, tc.name, tc.content); err == nil {
			t.Errorf("Save with %s: expected validation error, got nil", tc.label)
		}
	}
}

// TestSaveUnknownName verifies that Save rejects an unrecognised config name.
func TestSaveUnknownName(t *testing.T) {
	d := tempDirs(t)
	if err := config.Save(d, "custom", "version: 1\n"); err == nil {
		t.Error("Save(unknown): expected error, got nil")
	}
}

// TestSaveWritesBackup verifies that Save writes a timestamped backup when the file exists.
func TestSaveWritesBackup(t *testing.T) {
	d := tempDirs(t)
	// Write an original file.
	writeFixture(t, d, "network", validNetwork)
	// Overwrite with new content.
	newContent := "version: 1\nprofiles: {}\nsources: {}\n"
	if err := config.Save(d, "network", newContent); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// There should be a backup.
	backupDir := filepath.Join(d.Root, "config-backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("backup dir not created: %v", err)
	}
	found := false
	for _, e := range entries {
		if !e.IsDir() {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one backup file, found none")
	}
}
