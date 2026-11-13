package config

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/w3con/sifty/internal/dirs"
)

//go:embed vacancy-view.default.yaml
var defaultVacancyViewYAML []byte

// VacancyViewConfig holds the runtime configuration for the vacancy band and offer tiles.
type VacancyViewConfig struct {
	// ExchangeRates maps ISO 4217 upper-case currency codes to their value relative to EUR (the
	// base; "EUR" maps to 1.0). These are PLACEHOLDER values: the owner must update them manually.
	ExchangeRates map[string]float64 `yaml:"exchangeRates"`
	// ContractWords lists the explicit terms that identify an engagement type in vacancy text.
	ContractWords ContractWordLists `yaml:"contractWords"`
	// OfferVocabulary lists the offer items to look for in Compensation and Benefits sections.
	OfferVocabulary []OfferItemDef `yaml:"offerVocabulary"`
}

// ContractWordLists holds word lists for each recognised contract kind, keyed by engagement type.
type ContractWordLists struct {
	Contract  []string `yaml:"contract"`
	Permanent []string `yaml:"permanent"`
	FixedTerm []string `yaml:"fixedTerm"`
}

// OfferItemDef describes one item in the offer vocabulary.
type OfferItemDef struct {
	Key           string   `yaml:"key"`
	Label         string   `yaml:"label"`
	Keywords      []string `yaml:"keywords"`
	// NotStatedLabel is the description shown when a keyword is found in the section but no
	// specific value is extracted. Defaults to "Amount not stated" when empty.
	NotStatedLabel string `yaml:"notStatedLabel"`
}

// DefaultVacancyViewConfig returns the built-in vacancy-view configuration.
func DefaultVacancyViewConfig() (VacancyViewConfig, error) {
	var c VacancyViewConfig
	if err := yaml.Unmarshal(defaultVacancyViewYAML, &c); err != nil {
		return VacancyViewConfig{}, fmt.Errorf("parse built-in vacancy-view.default.yaml: %w", err)
	}
	return c, nil
}

// loadVacancyViewConfig reads config/vacancy-view.yaml from the data directory when it exists
// and otherwise returns the built-in defaults.
func loadVacancyViewConfig(d dirs.Dirs) (VacancyViewConfig, error) {
	path := d.ConfigFile("vacancy-view")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return DefaultVacancyViewConfig()
		}
		return VacancyViewConfig{}, fmt.Errorf("stat %s: %w", path, err)
	}
	var c VacancyViewConfig
	if err := loadYAML(path, &c); err != nil {
		return VacancyViewConfig{}, err
	}
	return c, nil
}
