package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed regions.default.yaml
var defaultRegionsYAML []byte

// defaultVocabulary is the shape of regions.default.yaml: the same keys as the quick_match block.
type defaultVocabulary struct {
	RestrictionPhrases []string           `yaml:"restriction_phrases"`
	Regions            []QuickMatchRegion `yaml:"regions"`
}

func loadDefaultVocabulary() (defaultVocabulary, error) {
	var v defaultVocabulary
	if err := yaml.Unmarshal(defaultRegionsYAML, &v); err != nil {
		return defaultVocabulary{}, fmt.Errorf("parse built-in regions.default.yaml: %w", err)
	}
	return v, nil
}

// DefaultRegions returns the built-in region vocabulary shipped with the binary.
// It is used when quick_match.regions is absent from runtime.yaml.
func DefaultRegions() ([]QuickMatchRegion, error) {
	v, err := loadDefaultVocabulary()
	if err != nil {
		return nil, err
	}
	return v.Regions, nil
}

// DefaultRestrictionPhrases returns the built-in restriction phrases shipped with the binary.
// It is used when quick_match.restriction_phrases is absent from runtime.yaml.
func DefaultRestrictionPhrases() ([]string, error) {
	v, err := loadDefaultVocabulary()
	if err != nil {
		return nil, err
	}
	return v.RestrictionPhrases, nil
}

// EffectiveRegions returns the configured regions when quick_match.regions is non-empty,
// and the built-in default vocabulary otherwise. This ensures restricted-remote detection
// works without any manual configuration.
func (q QuickMatch) EffectiveRegions() ([]QuickMatchRegion, error) {
	if len(q.Regions) > 0 {
		return q.Regions, nil
	}
	return DefaultRegions()
}

// EffectiveRestrictionPhrases returns the configured restriction phrases when
// quick_match.restriction_phrases is non-empty, and the built-in default otherwise.
func (q QuickMatch) EffectiveRestrictionPhrases() ([]string, error) {
	if len(q.RestrictionPhrases) > 0 {
		return q.RestrictionPhrases, nil
	}
	return DefaultRestrictionPhrases()
}
