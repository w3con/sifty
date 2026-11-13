package config

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/w3con/sifty/internal/dirs"
)

//go:embed skills.default.yaml
var defaultSkillsYAML []byte

// SkillGroup is a set of names that mean the same thing (Go and Golang, K8s and Kubernetes).
type SkillGroup struct {
	Name string `yaml:"name" json:"name"`
	// Terms are synonyms matched as whole words regardless of case.
	Terms []string `yaml:"terms" json:"terms"`
	// AmbiguousTerms are names that are also ordinary words or single letters (Go, C, R, TS). They
	// count only where a technology context shows what they are.
	AmbiguousTerms []string `yaml:"ambiguous_terms" json:"ambiguousTerms"`
}

// SkillCategory ties ordinary words to technologies of one kind ("cloud" to AWS, GCP and Azure), so
// a requirement for "cloud platform experience" finds a profile passage that says only "AWS".
// It is used when choosing the passages shown to the model, never to decide a verdict.
type SkillCategory struct {
	Words  []string `yaml:"words"  json:"words"`
	Groups []string `yaml:"groups" json:"groups"`
}

// Skills is the vocabulary of the requirement matcher: which names are the same technology, which
// words show that an ambiguous name is a technology, which words say nothing when looking for the
// profile passages closest to a requirement, and which requirements a CV usually cannot answer.
// docs/specs/quick-match.md section 14.12.
type Skills struct {
	Version           int             `yaml:"version"                json:"version"`
	TechnologyContext []string        `yaml:"technology_context"     json:"technologyContext"`
	GenericWords      []string        `yaml:"generic_words"          json:"genericWords"`
	NotAnswerable     []string        `yaml:"not_answerable_from_cv" json:"notAnswerableFromCV"`
	// MaxSkillNameWords is the maximum word count a phrase may have to be treated as a skill
	// name when added from a vacancy row. Phrases with more words go to "Confirmed by you". 0 uses 3.
	MaxSkillNameWords int      `yaml:"max_skill_name_words" json:"maxSkillNameWords"`
	// VerbEndings lists word-ending strings whose presence in any word of a phrase marks it as an
	// activity or description rather than a skill name (e.g. "ing"). Default: ["ing"].
	VerbEndings       []string `yaml:"verb_endings"         json:"verbEndings"`
	// ModifierWords lists -ing words that act as modifiers (adjectives) rather than activity verbs
	// when they appear inside a multi-word phrase and are followed by at least one more word.
	// "coding" in "AI coding tools" is a modifier; "coding" alone is an activity.
	// Default: [coding, testing, scripting, streaming, programming].
	ModifierWords     []string `yaml:"modifier_words"       json:"modifierWords"`
	Groups            []SkillGroup    `yaml:"groups"                 json:"groups"`
	Categories        []SkillCategory `yaml:"categories"          json:"categories"`
}

// DefaultSkills returns the built-in vocabulary shipped with the binary.
func DefaultSkills() (Skills, error) {
	var s Skills
	if err := yaml.Unmarshal(defaultSkillsYAML, &s); err != nil {
		return Skills{}, fmt.Errorf("parse built-in skills.default.yaml: %w", err)
	}
	if err := s.validate("skills.default.yaml"); err != nil {
		return Skills{}, err
	}
	return s, nil
}

// loadSkills reads config/skills.yaml from the data directory when it exists and otherwise returns
// the built-in vocabulary, so an installation that predates the file keeps working.
func loadSkills(d dirs.Dirs) (Skills, error) {
	path := d.ConfigFile("skills")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return DefaultSkills()
		}
		return Skills{}, fmt.Errorf("stat %s: %w", path, err)
	}
	var s Skills
	if err := loadYAML(path, &s); err != nil {
		return Skills{}, err
	}
	if err := s.validate("skills.yaml"); err != nil {
		return Skills{}, err
	}
	return s, nil
}

func (s Skills) validate(file string) error {
	if s.Version != 1 {
		return fmt.Errorf("%s: version must be 1", file)
	}
	owner := map[string]string{}
	claim := func(group, term string) error {
		key := strings.ToLower(strings.Join(strings.Fields(term), " "))
		if key == "" {
			return fmt.Errorf("%s: group %q has an empty term", file, group)
		}
		if prev, taken := owner[key]; taken && prev != group {
			return fmt.Errorf("%s: term %q belongs to both %q and %q", file, term, prev, group)
		}
		owner[key] = group
		return nil
	}
	for _, g := range s.Groups {
		if strings.TrimSpace(g.Name) == "" {
			return fmt.Errorf("%s: a group has no name", file)
		}
		for _, t := range g.Terms {
			if err := claim(g.Name, t); err != nil {
				return err
			}
		}
		for _, t := range g.AmbiguousTerms {
			if err := claim(g.Name, t); err != nil {
				return err
			}
		}
	}
	names := map[string]bool{}
	for _, g := range s.Groups {
		names[g.Name] = true
	}
	for _, c := range s.Categories {
		if len(c.Words) == 0 {
			return fmt.Errorf("%s: a category has no words", file)
		}
		for _, g := range c.Groups {
			if !names[g] {
				return fmt.Errorf("%s: category %v names the unknown group %q", file, c.Words, g)
			}
		}
	}
	return nil
}

// Identity is a short digest of the whole vocabulary. It enters the matcher version, so changing
// the file makes the matching pass run again and never the extraction.
func (s Skills) Identity() string {
	blob, _ := json.Marshal(s)
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])[:10]
}
