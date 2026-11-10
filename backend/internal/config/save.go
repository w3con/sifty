package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/w3con/sifty/internal/dirs"
	"gopkg.in/yaml.v3"
)

// ValidNames is the set of configuration files the backend accepts by name.
var ValidNames = map[string]bool{
	"search": true, "candidate": true, "providers": true,
	"network": true, "runtime": true,
}

// secretRe matches patterns that suggest an inline secret value rather than an environment
// variable reference. Matches TypeScript's pattern:
// /\b(?:sk-[A-Za-z0-9]{12}|Bearer\s+\S+|https?:\/\/[^\s/]+:[^\s/]+@)/i
var secretRe = regexp.MustCompile(
	`(?i)\b(?:sk-[A-Za-z0-9]{12}|Bearer\s+\S+|https?://[^\s/]+:[^\s/]+@)`,
)

// Read returns the raw YAML text of the named configuration file.
func Read(d dirs.Dirs, name string) (string, error) {
	if !ValidNames[name] {
		return "", fmt.Errorf("unknown configuration %q", name)
	}
	data, err := os.ReadFile(d.ConfigFile(name))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return string(data), nil
}

// Save validates content as YAML for the named configuration file, rejects inline secrets,
// writes a timestamped backup of the existing file, and writes the new content.
func Save(d dirs.Dirs, name, content string) error {
	if !ValidNames[name] {
		return fmt.Errorf("unknown configuration %q", name)
	}

	// Parse the incoming YAML so we can validate structure.
	var raw any
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Schema validation: re-use the typed loaders' validate path by unmarshaling into the
	// target type and calling validate on a synthesised Config.
	switch name {
	case "search":
		var v Search
		if err := yaml.Unmarshal([]byte(content), &v); err != nil {
			return fmt.Errorf("parse search config: %w", err)
		}
		c := &Config{Search: v}
		if err := c.validateSearch(); err != nil {
			return err
		}
	case "runtime":
		var v Runtime
		if err := yaml.Unmarshal([]byte(content), &v); err != nil {
			return fmt.Errorf("parse runtime config: %w", err)
		}
		c := &Config{Runtime: v}
		if err := c.validateRuntime(); err != nil {
			return err
		}
	case "providers":
		var v Providers
		if err := yaml.Unmarshal([]byte(content), &v); err != nil {
			return fmt.Errorf("parse providers config: %w", err)
		}
		c := &Config{Providers: v}
		if err := c.validateProviders(); err != nil {
			return err
		}
	case "candidate":
		var v Candidate
		if err := yaml.Unmarshal([]byte(content), &v); err != nil {
			return fmt.Errorf("parse candidate config: %w", err)
		}
		c := &Config{Candidate: v}
		if err := c.validateCandidate(); err != nil {
			return err
		}
	case "network":
		// Network has no required fields; just check it parses.
		var v Network
		if err := yaml.Unmarshal([]byte(content), &v); err != nil {
			return fmt.Errorf("parse network config: %w", err)
		}
	}

	// Reject inline secrets.
	if secretRe.MatchString(content) {
		return fmt.Errorf("Use local environment references, not secret values")
	}

	// Write a timestamped backup of the existing file.
	existing := d.ConfigFile(name)
	if data, err := os.ReadFile(existing); err == nil {
		stamp := time.Now().UTC().Format("20060102-150405")
		backupDir := filepath.Join(d.Root, "config-backups")
		_ = os.MkdirAll(backupDir, 0o700)
		backupPath := filepath.Join(backupDir, fmt.Sprintf("%s-%s.yaml", name, stamp))
		_ = os.WriteFile(backupPath, data, 0o600)
	}

	// Write the new content.
	if err := os.WriteFile(existing, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// Split validate into per-field helpers so Save can call them without creating a full Config.
func (c *Config) validateSearch() error {
	if c.Search.Version != 1 {
		return fmt.Errorf("search.yaml: version must be 1")
	}
	if len(c.Search.Sites) == 0 {
		return fmt.Errorf("search.yaml: at least one site is required")
	}
	seen := map[string]bool{}
	for _, s := range c.Search.Sites {
		if s.Name == "" {
			return fmt.Errorf("search.yaml: a site is missing a name")
		}
		if err := httpURL(s.URL); err != nil {
			return fmt.Errorf("search.yaml: site %q: %w", s.Name, err)
		}
		if seen[s.Name] {
			return fmt.Errorf("search.yaml: source names must be unique (%q)", s.Name)
		}
		seen[s.Name] = true
	}
	return nil
}

func (c *Config) validateRuntime() error {
	if c.Runtime.Version != 1 {
		return fmt.Errorf("runtime.yaml: version must be 1")
	}
	if c.Runtime.Port < 1024 || c.Runtime.Port > 65535 {
		return fmt.Errorf("runtime.yaml: port must be between 1024 and 65535")
	}
	if err := validateQuickMatch(c.Runtime.QuickMatch); err != nil {
		return err
	}
	return nil
}

// validateQuickMatch refuses partial or contradictory Quick Match configuration. A zero
// version means the block is absent and the documented defaults apply.
func validateQuickMatch(q QuickMatch) error {
	if q.Version == 0 {
		return nil
	}
	if q.Version != 1 {
		return fmt.Errorf("runtime.yaml: quick_match.version must be 1")
	}
	if q.RulesVersion == "" {
		return fmt.Errorf("runtime.yaml: quick_match.rules_version is required")
	}
	if q.SourcePolicyVersion == "" {
		return fmt.Errorf("runtime.yaml: quick_match.source_policy_version is required")
	}
	if q.RequiredWeight <= 0 || q.PreferredWeight <= 0 {
		return fmt.Errorf("runtime.yaml: quick_match requirement weights must be positive")
	}
	if q.Weights.Role <= 0 || q.Weights.Professional <= 0 || q.Weights.Compensation <= 0 ||
		q.Weights.Geography <= 0 || q.Weights.Conditions <= 0 {
		return fmt.Errorf("runtime.yaml: quick_match weights must all be positive")
	}
	if sum := q.Weights.Sum(); sum < 1-1e-9 || sum > 1+1e-9 {
		return fmt.Errorf("runtime.yaml: quick_match weights sum to %v, want 1", sum)
	}
	for source, rule := range q.SourceFields {
		switch rule.Location {
		case "", "usable", "needs_corroboration", "unusable":
		default:
			return fmt.Errorf("runtime.yaml: quick_match source %q has invalid location reliability %q", source, rule.Location)
		}
	}
	return nil
}

func (c *Config) validateProviders() error {
	if c.Providers.Version != 1 {
		return fmt.Errorf("providers.yaml: version must be 1")
	}
	for name, p := range c.Providers.Providers {
		if !validProviderTypes[p.Type] {
			return fmt.Errorf("providers.yaml: provider %q has invalid type %q", name, p.Type)
		}
		if p.Model == "" {
			return fmt.Errorf("providers.yaml: provider %q is missing a model", name)
		}
		if p.Endpoint != "" {
			if err := httpURL(p.Endpoint); err != nil {
				return fmt.Errorf("providers.yaml: provider %q endpoint: %w", name, err)
			}
		}
	}
	for stage, pname := range map[string]string{"assessment": c.Providers.Stages.Assessment, "extraction": c.Providers.Stages.Extraction} {
		if pname == "" {
			return fmt.Errorf("providers.yaml: stage %q has no provider", stage)
		}
		if _, ok := c.Providers.Providers[pname]; !ok {
			return fmt.Errorf("providers.yaml: stage %q names unknown provider %q", stage, pname)
		}
	}
	// preparation is optional (empty means unset); if set, it must name a known provider.
	if p := c.Providers.Stages.Preparation; p != "" {
		if _, ok := c.Providers.Providers[p]; !ok {
			return fmt.Errorf("providers.yaml: stage %q names unknown provider %q", "preparation", p)
		}
	}
	return nil
}

func (c *Config) validateCandidate() error {
	if c.Candidate.Version != 1 {
		return fmt.Errorf("candidate.yaml: version must be 1")
	}
	if c.Candidate.CacheHours <= 0 {
		return fmt.Errorf("candidate.yaml: cache_hours must be positive")
	}
	return nil
}
