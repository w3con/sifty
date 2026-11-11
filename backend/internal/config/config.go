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
	// CollectionDisabled stops every vacancy collection run of this process, manual and daily.
	// Review copies of the data use it so that they never fetch from the internet.
	CollectionDisabled bool `yaml:"collection_disabled"`

	// Profile document handling. Zero values select the documented defaults in the
	// Effective methods below; they are the single configuration location for the
	// extraction limits and accepted document types.
	ProfileMaxDocumentBytes    int      `yaml:"profile_max_document_bytes"`
	ProfileTextTimeoutMS       int      `yaml:"profile_text_timeout_ms"`
	ProfileExtractionTimeoutMS int      `yaml:"profile_extraction_timeout_ms"`
	ProfileMaxFacts            int      `yaml:"profile_max_facts"`
	ProfileDocumentTypes       []string `yaml:"profile_document_types"`

	// Profile avatar handling. These are the single authority for the accepted
	// actual image types and the upload ceiling; zero/empty values select the
	// documented defaults below.
	ProfileAvatarMaxBytes int      `yaml:"profile_avatar_max_bytes"`
	ProfileAvatarTypes    []string `yaml:"profile_avatar_types"`

	// Quick Match. This is the one configuration extension for the v1 rule defaults and
	// source-field usability policy. It contains no user/profile values and creates no new
	// settings framework. An absent block selects DefaultQuickMatch.
	QuickMatch QuickMatch `yaml:"quick_match"`
}

// QuickMatchWeights are the goal-group weights of Fits you. Fits you divides by the weights of
// the groups it uses, so they need not sum to 1; they are versioned with the rule set. A
// leftover professional key in an older runtime.yaml is ignored.
type QuickMatchWeights struct {
	Role         float64 `yaml:"role"         json:"role"`
	Compensation float64 `yaml:"compensation" json:"compensation"`
	Geography    float64 `yaml:"geography"    json:"geography"`
	Conditions   float64 `yaml:"conditions"   json:"conditions"`
}

// Sum returns the total of the four goal-group weights.
func (w QuickMatchWeights) Sum() float64 {
	return w.Role + w.Compensation + w.Geography + w.Conditions
}

// QuickMatchSourceField declares the usability of one source's structured fields. Only
// fields whose trust genuinely differs by source belong here; this is not a site preference
// coefficient.
type QuickMatchSourceField struct {
	Location string `yaml:"location,omitempty" json:"location,omitempty"`
}

// QuickMatch is the extractor/evaluator configuration extension. SourceFields maps a saved
// source name to field usability declarations. Weights and requirement weights are the
// approved v1 defaults, still uncalibrated.
type QuickMatch struct {
	Version             int                              `yaml:"version"              json:"version"`
	RulesVersion        string                           `yaml:"rules_version"        json:"rulesVersion"`
	SourcePolicyVersion string                           `yaml:"source_policy_version" json:"sourcePolicyVersion"`
	Weights             QuickMatchWeights                `yaml:"weights"              json:"weights"`
	RequiredWeight      float64                          `yaml:"required_weight"      json:"requiredWeight"`
	PreferredWeight     float64                          `yaml:"preferred_weight"     json:"preferredWeight"`
	SourceFields        map[string]QuickMatchSourceField `yaml:"source_fields"        json:"sourceFields"`
	ModelPasses         QuickMatchModelPasses            `yaml:"model_passes"         json:"modelPasses"`
}

// QuickMatchModelPasses tunes the two model passes (requirement extraction and profile matching).
// Zero values select the documented defaults through the Effective accessors, so an older block
// without this key stays valid.
type QuickMatchModelPasses struct {
	// MaxRequirements caps how many requirements one vacancy may keep after validation.
	MaxRequirements int `yaml:"max_requirements" json:"maxRequirements"`
	// MaxAttempts caps the calls made for one pass when the provider fails transiently.
	MaxAttempts int `yaml:"max_attempts" json:"maxAttempts"`
	// Temperature is the sampling temperature for both passes; nil selects the default.
	Temperature *float64 `yaml:"temperature" json:"temperature"`
	// AllowPaidBulkProvider lets bulk extraction use a provider that is neither local nor marked
	// free. It is false unless the owner turns it on explicitly.
	AllowPaidBulkProvider bool `yaml:"allow_paid_bulk_provider" json:"allowPaidBulkProvider"`
	// Paused stops the background worker that runs the passes; a pass already running finishes.
	Paused bool `yaml:"paused" json:"paused"`
	// DailyLimit caps the vacancies analysed per local calendar day; 0 means no limit.
	DailyLimit int `yaml:"daily_limit" json:"dailyLimit"`
	// Order is the order the background worker takes vacancies in. It follows the Inbox sort
	// toggle: "fit" (Fits you descending, the default) or "newest" (the Inbox Newest key). The
	// frontend saves it through PUT /api/quick-match/order; the worker reads it before each vacancy.
	Order string `yaml:"order" json:"order"`
	// EvidencePassages is how many profile passages closest to a requirement the matching model is
	// shown for it (the computed facts are always shown besides). Zero selects the default.
	EvidencePassages int `yaml:"evidence_passages" json:"evidencePassages"`
	// Matching switches the model's matching pass on. It is off unless set to true: the worker then
	// runs extraction only, and You fit is not shown (docs/specs/quick-match.md 14.14).
	Matching bool `yaml:"matching" json:"matching"`
}

// Defaults of the model passes.
const (
	DefaultMaxRequirements = 30
	DefaultMaxAttempts     = 3
	DefaultTemperature     = 0.1
	// DefaultEvidencePassages is the number of profile passages shown for one requirement.
	DefaultEvidencePassages = 3
)

// EffectiveMaxRequirements returns the configured cap, or the default when unset.
func (m QuickMatchModelPasses) EffectiveMaxRequirements() int {
	if m.MaxRequirements > 0 {
		return m.MaxRequirements
	}
	return DefaultMaxRequirements
}

// EffectiveMaxAttempts returns the configured attempt cap, or the default when unset.
func (m QuickMatchModelPasses) EffectiveMaxAttempts() int {
	if m.MaxAttempts > 0 {
		return m.MaxAttempts
	}
	return DefaultMaxAttempts
}

// EffectiveEvidencePassages returns the configured number of passages per requirement, or the
// default when unset.
func (m QuickMatchModelPasses) EffectiveEvidencePassages() int {
	if m.EvidencePassages > 0 {
		return m.EvidencePassages
	}
	return DefaultEvidencePassages
}

// Model work orders; they match the two values of the Inbox sort toggle.
const (
	OrderFit    = "fit"
	OrderNewest = "newest"
)

// EffectiveOrder returns the configured order, or Best fit when unset.
func (m QuickMatchModelPasses) EffectiveOrder() string {
	if m.Order == OrderNewest {
		return OrderNewest
	}
	return OrderFit
}

// EffectiveTemperature returns the configured temperature, or the default when unset.
func (m QuickMatchModelPasses) EffectiveTemperature() float64 {
	if m.Temperature != nil {
		return *m.Temperature
	}
	return DefaultTemperature
}

// DefaultSourcePolicyVersion identifies the default source-field policy data shape.
const DefaultSourcePolicyVersion = "source-policy/1"

// DefaultQuickMatch returns the approved v1 numerical defaults and the known ambiguous board
// location declaration. These values are configuration defaults, not calibrated preferences.
func DefaultQuickMatch() QuickMatch {
	return QuickMatch{
		Version:             1,
		RulesVersion:        "quickmatch.rules/2",
		SourcePolicyVersion: DefaultSourcePolicyVersion,
		Weights: QuickMatchWeights{
			Role: 0.25, Compensation: 0.20,
			Geography: 0.15, Conditions: 0.10,
		},
		RequiredWeight:  2,
		PreferredWeight: 1,
		SourceFields: map[string]QuickMatchSourceField{
			"Wellfound": {Location: "needs_corroboration"},
		},
	}
}

// EffectiveQuickMatch returns the configured block, or the documented defaults when the block
// is absent. It does not merge partial blocks: a present block must be complete and valid.
func (r Runtime) EffectiveQuickMatch() QuickMatch {
	if r.QuickMatch.Version == 0 {
		return DefaultQuickMatch()
	}
	return r.QuickMatch
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

// EffectiveProfileAvatarMaxBytes is the upload ceiling for the profile photo.
func (r Runtime) EffectiveProfileAvatarMaxBytes() int {
	if r.ProfileAvatarMaxBytes > 0 {
		return r.ProfileAvatarMaxBytes
	}
	return 2 << 20 // 2 MiB
}

// EffectiveProfileAvatarUploadBodyBytes is the JSON request-body allowance for one base64
// avatar. It is derived from the same configured ceiling plus fixed base64/JSON overhead.
func (r Runtime) EffectiveProfileAvatarUploadBodyBytes() int {
	return r.EffectiveProfileAvatarMaxBytes()*4/3 + 64*1024
}

// EffectiveProfileAvatarTypes returns the accepted actual image types without a leading dot.
func (r Runtime) EffectiveProfileAvatarTypes() []string {
	if len(r.ProfileAvatarTypes) > 0 {
		return r.ProfileAvatarTypes
	}
	return []string{"png", "jpg", "webp"}
}

// Provider is one inference provider definition.
type Provider struct {
	Type         string `yaml:"type"`
	Model        string `yaml:"model"`
	Endpoint     string `yaml:"endpoint"`
	APIKeyEnv    string `yaml:"api_key_env"`
	TimeoutMS    int    `yaml:"timeout_ms"`
	ContextChars int    `yaml:"context_chars"`
	// Free marks a provider that costs nothing per call (for example a flat subscription the
	// owner accepts for bulk work). Local providers (ollama, compatible over loopback) are free
	// by construction and need no mark.
	Free bool `yaml:"free,omitempty"`
	// Think sets the reasoning effort ("low", "medium" or "high") of an ollama model that cannot
	// switch reasoning off (gpt-oss). Empty leaves the model's own default, and the quick-match
	// passes then ask for no reasoning.
	Think string `yaml:"think,omitempty"`
}

// Providers is the provider configuration, with the provider chosen for each stage.
type Providers struct {
	Version int `yaml:"version"`
	Stages  struct {
		Assessment  string `yaml:"assessment"`
		Extraction  string `yaml:"extraction"`
		Preparation string `yaml:"preparation"`
		// Matching is the provider that compares requirements with the profile. Optional: empty
		// means the extraction provider, so one model does both passes unless this names another.
		Matching string `yaml:"matching,omitempty"`
	} `yaml:"stages"`
	Providers map[string]Provider `yaml:"providers"`
}

// MatchingStage is the name of the provider the matching pass uses.
func (p Providers) MatchingStage() string {
	if p.Stages.Matching != "" {
		return p.Stages.Matching
	}
	return p.Stages.Extraction
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
	// Skills is the vocabulary of the requirement matcher (skills.yaml, or the built-in default).
	Skills Skills
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
	skills, err := loadSkills(d)
	if err != nil {
		return nil, err
	}
	c.Skills = skills
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
	if err := validateQuickMatch(c.Runtime.QuickMatch); err != nil {
		return err
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
		switch p.Think {
		case "", "low", "medium", "high":
		default:
			return fmt.Errorf("providers.yaml: provider %q think must be low, medium or high", name)
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
	// preparation and matching are optional (empty means unset); if set, each must name a known provider.
	for stage, p := range map[string]string{"preparation": c.Providers.Stages.Preparation, "matching": c.Providers.Stages.Matching} {
		if p == "" {
			continue
		}
		if _, ok := c.Providers.Providers[p]; !ok {
			return fmt.Errorf("providers.yaml: stage %q names unknown provider %q", stage, p)
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
