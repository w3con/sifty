package vacancyai

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/config"
)

var (
	// ErrPaidProvider means a bulk model pass was refused because its stage names a provider that
	// is neither local nor marked free and the configuration does not allow it. The guard is about
	// cost only: where the data goes is the owner's choice of provider.
	ErrPaidProvider = errors.New("bulk model pass refused: the stage provider is not local or free")
)

// local reports whether a provider is a model on this machine: an ollama or OpenAI-compatible
// endpoint on a loopback address. Local providers cost nothing per call and keep data on the
// machine.
func local(cp config.Provider) bool {
	if cp.Type != "ollama" && cp.Type != "compatible" {
		return false
	}
	u, err := url.Parse(cp.Endpoint)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1", "[::1]":
		return true
	}
	return false
}

// stageProvider resolves the provider the extraction stage names and builds the call options both
// passes share (the matching pass uses the extraction provider unless stages.matching names another): reasoning off, the configured low temperature, and the provider's own timeout and
// context budget. Nothing about the model, endpoint or limits is decided here.
func stageProvider(cfg *config.Config, stage, name string) (string, config.Provider, assess.Provider, config.QuickMatchModelPasses, error) {
	if name == "" {
		return "", config.Provider{}, assess.Provider{}, config.QuickMatchModelPasses{},
			fmt.Errorf("%s provider is not configured; set stages.%s in providers.yaml", stage, stage)
	}
	cp, ok := cfg.Providers.Providers[name]
	if !ok {
		return "", config.Provider{}, assess.Provider{}, config.QuickMatchModelPasses{},
			fmt.Errorf("%s provider %q is not defined in providers.yaml", stage, name)
	}
	passes := cfg.Runtime.EffectiveQuickMatch().ModelPasses
	noThinking := false
	temperature := passes.EffectiveTemperature()
	return name, cp, assess.Provider{
		Type: cp.Type, Model: cp.Model, Endpoint: cp.Endpoint, APIKeyEnv: cp.APIKeyEnv,
		TimeoutMS: cp.TimeoutMS, ContextChars: cp.ContextChars,
		Think: &noThinking, ThinkLevel: cp.Think, Temperature: &temperature,
	}, passes, nil
}

// NewExtractor builds the extraction pass on the provider of the extraction stage. It refuses a
// provider that is neither local nor marked free unless quick_match.model_passes
// allow_paid_bulk_provider is true, because extraction runs for every vacancy and would otherwise
// spend a paid provider on a backlog.
func NewExtractor(cfg *config.Config, infer InferFunc) (Extractor, error) {
	name, cp, provider, passes, err := stageProvider(cfg, "extraction", cfg.Providers.Stages.Extraction)
	if err != nil {
		return Extractor{}, err
	}
	if !local(cp) && !cp.Free && !passes.AllowPaidBulkProvider {
		return Extractor{}, fmt.Errorf("%w: stage extraction uses %q (type %s); route it to a local provider, mark that provider free: true, or set quick_match.model_passes.allow_paid_bulk_provider",
			ErrPaidProvider, name, cp.Type)
	}
	return Extractor{
		ProviderName: name, Provider: provider, Infer: infer,
		MaxRequirements: passes.EffectiveMaxRequirements(), MaxAttempts: passes.EffectiveMaxAttempts(),
		Backoff: 2 * time.Second,
	}, nil
}

// NewMatcher builds the matching pass on the provider of the matching stage (the extraction
// provider when none is named). Like extraction it runs for every vacancy, so the same cost guard
// applies. Which provider receives the profile facts is the owner's configuration choice.
func NewMatcher(cfg *config.Config, infer InferFunc) (Matcher, error) {
	name, cp, provider, passes, err := stageProvider(cfg, "matching", cfg.Providers.MatchingStage())
	if err != nil {
		return Matcher{}, err
	}
	if !local(cp) && !cp.Free && !passes.AllowPaidBulkProvider {
		return Matcher{}, fmt.Errorf("%w: stage matching uses %q (type %s); route it to a local provider, mark that provider free: true, or set quick_match.model_passes.allow_paid_bulk_provider",
			ErrPaidProvider, name, cp.Type)
	}
	return Matcher{
		ProviderName: name, Provider: provider, Infer: infer,
		MaxAttempts: passes.EffectiveMaxAttempts(), Backoff: 2 * time.Second,
		Vocabulary: NewVocabulary(cfg.Skills), EvidencePassages: passes.EffectiveEvidencePassages(),
	}, nil
}

// NewCodeMatcher builds a matcher that decides by code alone and calls no model, for the time the
// matching pass is switched off (quick_match.model_passes.matching). It needs no provider.
func NewCodeMatcher(cfg *config.Config) Matcher {
	return Matcher{Vocabulary: NewVocabulary(cfg.Skills), CodeOnly: true}
}
