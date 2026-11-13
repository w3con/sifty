package config

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/w3con/sifty/internal/dirs"
)

//go:embed sections.default.yaml
var defaultSectionsYAML []byte

// SectionEntry maps one class label to the heading patterns that belong to it.
// Prefix, when non-empty, matches any heading whose normalised text starts with
// that word or phrase followed by a space (e.g. "about" matches "about chainlink").
// Contains, when non-empty, lists words any one of which causes the heading to be
// classified into this class when the normalised text contains it as a word.
type SectionEntry struct {
	Class    string   `yaml:"class"`
	Headings []string `yaml:"headings"`
	Prefix   []string `yaml:"prefix,omitempty"`
	Contains []string `yaml:"contains,omitempty"`
}

// SectionsDef is the section vocabulary loaded from sections.yaml (or the built-in default).
// It is used by the section splitter to classify heading-delimited blocks in the listing view.
type SectionsDef struct {
	Version  int            `yaml:"version"`
	Sections []SectionEntry `yaml:"sections"`
}

// DefaultSectionsDef returns the built-in section vocabulary shipped with the binary.
func DefaultSectionsDef() (SectionsDef, error) {
	var s SectionsDef
	if err := yaml.Unmarshal(defaultSectionsYAML, &s); err != nil {
		return SectionsDef{}, fmt.Errorf("parse built-in sections.default.yaml: %w", err)
	}
	if s.Version != 1 {
		return SectionsDef{}, fmt.Errorf("sections.default.yaml: version must be 1")
	}
	return s, nil
}

// loadSectionsDef reads config/sections.yaml from the data directory when it exists and
// otherwise returns the built-in vocabulary, so an installation that predates the file
// keeps working.
func loadSectionsDef(d dirs.Dirs) (SectionsDef, error) {
	path := d.ConfigFile("sections")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return DefaultSectionsDef()
		}
		return SectionsDef{}, fmt.Errorf("stat %s: %w", path, err)
	}
	var s SectionsDef
	if err := loadYAML(path, &s); err != nil {
		return SectionsDef{}, err
	}
	if s.Version != 1 {
		return SectionsDef{}, fmt.Errorf("sections.yaml: version must be 1")
	}
	return s, nil
}
