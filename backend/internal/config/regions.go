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
	WrapperPrefixes    []string           `yaml:"wrapper_prefixes"`
	TrailingQualifiers []string           `yaml:"trailing_qualifiers"`
	Separators         []string           `yaml:"separators"`
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

// DefaultWrapperPrefixes returns the built-in wrapper prefixes shipped with the binary.
func DefaultWrapperPrefixes() ([]string, error) {
	v, err := loadDefaultVocabulary()
	if err != nil {
		return nil, err
	}
	return v.WrapperPrefixes, nil
}

// DefaultTrailingQualifiers returns the built-in trailing qualifiers shipped with the binary.
func DefaultTrailingQualifiers() ([]string, error) {
	v, err := loadDefaultVocabulary()
	if err != nil {
		return nil, err
	}
	return v.TrailingQualifiers, nil
}

// DefaultSeparators returns the built-in label separators shipped with the binary.
func DefaultSeparators() ([]string, error) {
	v, err := loadDefaultVocabulary()
	if err != nil {
		return nil, err
	}
	return v.Separators, nil
}

// EffectiveWrapperPrefixes returns the configured wrapper prefixes when
// quick_match.wrapper_prefixes is non-empty, and the built-in default otherwise.
func (q QuickMatch) EffectiveWrapperPrefixes() ([]string, error) {
	if len(q.WrapperPrefixes) > 0 {
		return q.WrapperPrefixes, nil
	}
	return DefaultWrapperPrefixes()
}

// EffectiveTrailingQualifiers returns the configured trailing qualifiers when
// quick_match.trailing_qualifiers is non-empty, and the built-in default otherwise.
func (q QuickMatch) EffectiveTrailingQualifiers() ([]string, error) {
	if len(q.TrailingQualifiers) > 0 {
		return q.TrailingQualifiers, nil
	}
	return DefaultTrailingQualifiers()
}

// EffectiveSeparators returns the configured separators when quick_match.separators is non-empty,
// and the built-in default otherwise.
func (q QuickMatch) EffectiveSeparators() ([]string, error) {
	if len(q.Separators) > 0 {
		return q.Separators, nil
	}
	return DefaultSeparators()
}
