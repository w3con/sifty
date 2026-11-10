// Package config loads and validates the YAML configuration files from the data directory's
// config/ subdirectory: search, runtime, providers, candidate and network. The structures and
// validation mirror the current TypeScript application's Zod schemas, so a configuration valid
// there is valid here.
package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"

	"github.com/w3con/sifty/internal/dirs"
	"gopkg.in/yaml.v3"
)

// Source is one job-board source.
type Source struct {
	Name    string `yaml:"name"    json:"name"`
	URL     string `yaml:"url"     json:"url"`
	Enabled *bool  `yaml:"enabled" json:"enabled,omitempty"`
}

// IsEnabled reports whether the source is active; a source defaults to enabled.
func (s Source) IsEnabled() bool { return s.Enabled == nil || *s.Enabled }

// SearchExpression is one OR or AND term group within the search configuration.
// Set Or for a phrase-or query (any term must appear); set And to require all phrases to be
// present in the indexed text. When both Or and And are set, Or takes precedence.
// Groups are combined with OR at the top level by BuildFTSQueryExprs.
type SearchExpression struct {
	Or  []string `yaml:"or,omitempty"  json:"or,omitempty"`
	And []string `yaml:"and,omitempty" json:"and,omitempty"`
}

// Search is the search configuration (sites and query terms).
type Search struct {
	Version int      `yaml:"version" json:"version"`
	Sites   []Source `yaml:"sites"   json:"sites"`
	Search  struct {
		Titles      []string           `yaml:"titles"      json:"titles"`
		Keywords    []string           `yaml:"keywords"    json:"keywords"`
		Expressions []SearchExpression `yaml:"expressions" json:"expressions"`
	} `yaml:"search" json:"search"`
	Preferences map[string]any `yaml:"preferences" json:"preferences"`
}

// EffectiveExpressions returns the search expressions to use for collection and ranking.
// When Expressions is non-empty, it is returned as-is (the owner has opted into the new model).
// Otherwise the existing flat Titles and Keywords lists are synthesised into a single OR group,
// preserving backward compatibility with configs that pre-date the expression model.
func (s *Search) EffectiveExpressions() []SearchExpression {
	if len(s.Search.Expressions) > 0 {
		return s.Search.Expressions
	}
	flat := append(append([]string{}, s.Search.Titles...), s.Search.Keywords...)
	if len(flat) == 0 {
		return nil
	}
	return []SearchExpression{{Or: flat}}
}

// Runtime is the collection and assessment runtime configuration.
type Runtime struct {
	Version              int  `yaml:"version"`
	Port                 int  `yaml:"port"`
	MaxPages             int  `yaml:"max_pages"`
	MaxJobsPerSource     int  `yaml:"max_jobs_per_source"`
	MaxReviewedPerSrc    int  `yaml:"max_reviewed_per_source"`
	MaxDetailRequests    int  `yaml:"max_detail_requests"`
	DetailConcurrency    int  `yaml:"detail_concurrency"`
	MaxQueryGroups       int  `yaml:"max_query_groups"`
	RequestTimeoutMS     int  `yaml:"request_timeout_ms"`
	MaxAssessmentsRun    int  `yaml:"max_assessments_per_run"`
	ModelExtraction      bool `yaml:"model_extraction"`
	DailyHour            int  `yaml:"daily_hour"`
	DailyMinute          int  `yaml:"daily_minute"`
	BrowserIntervalMS    int  `yaml:"browser_interval_ms"`
	BrowserJitterMS      int  `yaml:"browser_jitter_ms"`
	BrowserConcurrency   int  `yaml:"browser_concurrency"`
	BrowserCooldownMS    int  `yaml:"browser_cooldown_ms"`
	CollectionCooldownMS int  `yaml:"collection_cooldown_ms"`

	// Profile document handling. Zero values select the documented defaults in the
	// Effective methods below; they are the single configuration location for the
	// extraction limits and accepted document types.
	ProfileMaxDocumentBytes    int      `yaml:"profile_max_document_bytes"`
	ProfileTextTimeoutMS       int      `yaml:"profile_text_timeout_ms"`
	ProfileExtractionTimeoutMS int      `yaml:"profile_extraction_timeout_ms"`
	ProfileMaxFacts            int      `yaml:"profile_max_facts"`
	ProfileDocumentTypes       []string `yaml:"profile_document_types"`
}

// EffectiveProfileMaxDocumentBytes is the upload ceiling for one profile document.
func (r Runtime) EffectiveProfileMaxDocumentBytes() int {
	if r.ProfileMaxDocumentBytes > 0 {
		return r.ProfileMaxDocumentBytes
	}
	return 2 << 20 // 2 MiB
}

// EffectiveProfileUploadBodyBytes is the JSON request-body allowance for one base64 document
// upload. It is derived from the same configured document ceiling plus fixed base64/JSON overhead;
// the upload handler enforces it before decoding.
func (r Runtime) EffectiveProfileUploadBodyBytes() int {
	max := r.EffectiveProfileMaxDocumentBytes()
	return max*4/3 + 64*1024
}

// EffectiveProfileTextTimeoutMS bounds local text extraction (pdftotext/DOCX/plain read).
func (r Runtime) EffectiveProfileTextTimeoutMS() int {
	if r.ProfileTextTimeoutMS > 0 {
		return r.ProfileTextTimeoutMS
	}
	return 30000
}

// EffectiveProfileExtractionTimeoutMS bounds one configured extraction-provider call.
func (r Runtime) EffectiveProfileExtractionTimeoutMS() int {
	if r.ProfileExtractionTimeoutMS > 0 {
		return r.ProfileExtractionTimeoutMS
	}
	return 120000
}

// EffectiveProfileMaxFacts bounds how many attributed proposals one analysis may store.
func (r Runtime) EffectiveProfileMaxFacts() int {
	if r.ProfileMaxFacts > 0 {
		return r.ProfileMaxFacts
	}
	return 100
}

// EffectiveProfileDocumentTypes returns the accepted upload extensions without a leading dot.
func (r Runtime) EffectiveProfileDocumentTypes() []string {
	if len(r.ProfileDocumentTypes) > 0 {
		return r.ProfileDocumentTypes
	}
	return []string{"pdf", "docx", "md", "txt"}
}

// Provider is one inference provider definition.
type Provider struct {
	Type         string `yaml:"type"`
	Model        string `yaml:"model"`
	Endpoint     string `yaml:"endpoint"`
	APIKeyEnv    string `yaml:"api_key_env"`
	TimeoutMS    int    `yaml:"timeout_ms"`
	ContextChars int    `yaml:"context_chars"`
}

// Providers is the provider configuration, with the provider chosen for each stage.
type Providers struct {
	Version int `yaml:"version"`
	Stages  struct {
		Assessment  string `yaml:"assessment"`
		Extraction  string `yaml:"extraction"`
		Preparation string `yaml:"preparation"`
	} `yaml:"stages"`
	Providers map[string]Provider `yaml:"providers"`
}

// SourceRole describes the category and scope of a candidate evidence source.
type SourceRole struct {
	Category string `yaml:"category"`
	Scope    string `yaml:"scope"`
}

// ProfileRef is a remote candidate profile, with an optional local export file.
type ProfileRef struct {
	ID     string `yaml:"id"`
	URL    string `yaml:"url"`
	Export string `yaml:"export"`
}

// Candidate is the candidate-evidence configuration. Evidence file paths are resolved under the
// data directory's evidence/ subdirectory, never a git root.
type Candidate struct {
	Version     int                   `yaml:"version"`
	Files       []string              `yaml:"files"`
	Profiles    []ProfileRef          `yaml:"profiles"`
	CacheHours  float64               `yaml:"cache_hours"`
	SourceRoles map[string]SourceRole `yaml:"source_roles"`
	ReviewNotes []string              `yaml:"review_notes"`
}

// Network is the per-source network profile configuration.
type Network struct {
	Version  int                       `yaml:"version"`
	Profiles map[string]map[string]any `yaml:"profiles"`
	Sources  map[string]string         `yaml:"sources"`
}

// Config is the full loaded configuration.
type Config struct {
	Search    Search
	Runtime   Runtime
	Providers Providers
	Candidate Candidate
	Network   Network
}

var validProviderTypes = map[string]bool{"opencode": true, "claude": true, "ollama": true, "compatible": true}
var apiKeyEnvRe = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func loadYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func httpURL(s string) error {
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("not an HTTP(S) URL: %q", s)
	}
	return nil
}

// Load reads and validates every configuration file from the data directory.
func Load(d dirs.Dirs) (*Config, error) {
	c := &Config{}
	if err := loadYAML(d.ConfigFile("search"), &c.Search); err != nil {
		return nil, err
	}
	if err := loadYAML(d.ConfigFile("runtime"), &c.Runtime); err != nil {
		return nil, err
	}
	if err := loadYAML(d.ConfigFile("providers"), &c.Providers); err != nil {
		return nil, err
	}
	if err := loadYAML(d.ConfigFile("candidate"), &c.Candidate); err != nil {
		return nil, err
	}
	// network.yaml is optional.
	if _, err := os.Stat(d.ConfigFile("network")); err == nil {
		if err := loadYAML(d.ConfigFile("network"), &c.Network); err != nil {
			return nil, err
		}
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
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
	if c.Runtime.Version != 1 {
		return fmt.Errorf("runtime.yaml: version must be 1")
	}
	if c.Runtime.Port < 1024 || c.Runtime.Port > 65535 {
		return fmt.Errorf("runtime.yaml: port must be between 1024 and 65535")
	}
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
		if p.APIKeyEnv != "" && !apiKeyEnvRe.MatchString(p.APIKeyEnv) {
			return fmt.Errorf("providers.yaml: provider %q api_key_env must be an environment variable name", name)
		}
	}
	for stage, name := range map[string]string{"assessment": c.Providers.Stages.Assessment, "extraction": c.Providers.Stages.Extraction} {
		if name == "" {
			return fmt.Errorf("providers.yaml: stage %q has no provider", stage)
		}
		if _, ok := c.Providers.Providers[name]; !ok {
			return fmt.Errorf("providers.yaml: stage %q names unknown provider %q", stage, name)
		}
	}
	// preparation is optional (empty means unset); if set, it must name a known provider.
	if p := c.Providers.Stages.Preparation; p != "" {
		if _, ok := c.Providers.Providers[p]; !ok {
			return fmt.Errorf("providers.yaml: stage %q names unknown provider %q", "preparation", p)
		}
	}
	if c.Candidate.Version != 1 {
		return fmt.Errorf("candidate.yaml: version must be 1")
	}
	if c.Candidate.CacheHours <= 0 {
		return fmt.Errorf("candidate.yaml: cache_hours must be positive")
	}
	return nil
}
