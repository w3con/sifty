package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed regions.default.yaml
var defaultRegionsYAML []byte

// DefaultRegions returns the built-in region vocabulary shipped with the binary.
// It is used when quick_match.regions is absent from runtime.yaml.
func DefaultRegions() ([]QuickMatchRegion, error) {
	var regions []QuickMatchRegion
	if err := yaml.Unmarshal(defaultRegionsYAML, &regions); err != nil {
		return nil, fmt.Errorf("parse built-in regions.default.yaml: %w", err)
	}
	return regions, nil
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
