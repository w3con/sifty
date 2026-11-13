package vacancyai_test

import (
	"testing"
)

// TestClassifySkillAdd covers the cases from plan.md 2026-11-13 and owner decisions.
//
// Rules (from config defaults: max_skill_name_words=3, verb_endings=[ing],
//   modifier_words=[coding,testing,scripting,streaming,programming]):
//   - "X or Y" where each part is a single-word or vocab-known name → offer_alternatives.
//   - "X/Y" where each part is vocab-known → offer_alternatives.
//   - Every comma-separated part is a known vocabulary technology, OR
//   - Each part is a short name: ≤ 3 words AND no word (at first or last position) ending with
//     "ing" that is not a modifier word with a following word.
//   - If any part fails → the whole original phrase goes to ConfirmedFact.
//   - Alternatives present in request → HasAlternatives=true, no auto-add.
func TestClassifySkillAdd(t *testing.T) {
	v := defaultVocabulary(t)

	cases := []struct {
		name         string
		input        string
		alternatives []string
		// expected
		wantSkills  []string
		wantFact    string
		wantAltFlag bool
		wantAlts    []string // expected Alternatives when wantAltFlag=true (nil = not checked)
	}{
		{
			name:     "phrase with ing verbs → confirmed fact",
			input:    "Coding, debugging, reviewing",
			wantFact: "Coding, debugging, reviewing",
		},
		{
			name:       "two vocabulary names → two skills",
			input:      "TypeScript, Node",
			wantSkills: []string{"TypeScript", "Node"},
		},
		{
			// Server detects "or" and returns alternatives; no request-body alternatives needed.
			name:        "Go or Rust: server detects alternatives",
			input:       "Go or Rust",
			wantAltFlag: true,
			wantAlts:    []string{"Go", "Rust"},
		},
		{
			// When alternatives are already in the request body, the flag is set without parsing the name.
			name:         "alternatives in request → offer alternatives flag",
			input:        "Go or Rust",
			alternatives: []string{"Go", "Rust"},
			wantAltFlag:  true,
		},
		{
			name:       "vocabulary skill → skill",
			input:      "Python",
			wantSkills: []string{"Python"},
		},
		// "AI coding tools": owner decision 2026-11-13: should be a skill.
		// "coding" is a modifier_word; inside a multi-word phrase it is not an activity indicator.
		{
			name:       "AI coding tools: coding is a modifier → skill",
			input:      "AI coding tools",
			wantSkills: []string{"AI coding tools"},
		},
		{
			name:       "single short non-verb name → skill",
			input:      "GraphQL",
			wantSkills: []string{"GraphQL"},
		},
		{
			name:       "two-word non-verb name → skill",
			input:      "Machine Learning",
			wantSkills: []string{"Machine Learning"},
		},
		{
			name:     "four-word name exceeds word limit → confirmed fact",
			input:    "Real time data processing",
			wantFact: "Real time data processing",
		},
		{
			name:     "mixed list: one part has -ing → whole phrase is confirmed fact",
			input:    "TypeScript, debugging",
			wantFact: "TypeScript, debugging",
		},
		// "Coding" alone is an activity (modifier_word as sole word → activity).
		{
			name:     "Coding alone → confirmed fact",
			input:    "Coding",
			wantFact: "Coding",
		},
		// "Coding tools" → "Coding" is first but has a following word → modifier → skill.
		{
			name:       "Coding tools → modifier with following word → skill",
			input:      "Coding tools",
			wantSkills: []string{"Coding tools"},
		},
		// "Code reviewing" → "reviewing" is last, not in modifierWords → activity indicator → fact.
		{
			name:     "Code reviewing → reviewing at last position → confirmed fact",
			input:    "Code reviewing",
			wantFact: "Code reviewing",
		},
		// "Familiarity with Go or Rust" → "Familiarity with Go" is not isAlternativeName →
		// or-detected-but-invalid path → ConfirmedFact immediately.
		{
			name:     "Familiarity with Go or Rust: prefix phrase → confirmed fact",
			input:    "Familiarity with Go or Rust",
			wantFact: "Familiarity with Go or Rust",
		},
		// "Datadog / OpenSearch": Datadog not in vocabulary but single-word IsSkillName; OpenSearch
		// is vocabulary (Elasticsearch group). Both pass isAlternativeName → alternatives.
		{
			name:        "Datadog / OpenSearch: slash alternatives via isAlternativeName",
			input:       "Datadog / OpenSearch",
			wantAltFlag: true,
			wantAlts:    []string{"Datadog", "OpenSearch"},
		},
		// "C++ or C": both in vocabulary → alternatives (pressing ✕ should dismiss each separately).
		{
			name:        "C++ or C: vocabulary or-alternatives",
			input:       "C++ or C",
			wantAltFlag: true,
			wantAlts:    []string{"C++", "C"},
		},
		// "reporting/planning": both -ing words → both fail isAlternativeName → ConfirmedFact.
		{
			name:     "reporting/planning: -ing parts fail isAlternativeName → confirmed fact",
			input:    "reporting/planning",
			wantFact: "reporting/planning",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cls := v.ClassifySkillAdd(c.input, c.alternatives)
			if c.wantAltFlag {
				if !cls.HasAlternatives {
					t.Errorf("want HasAlternatives=true, got SkillsToAdd=%v ConfirmedFact=%q", cls.SkillsToAdd, cls.ConfirmedFact)
				}
				if c.wantAlts != nil {
					if len(cls.Alternatives) != len(c.wantAlts) {
						t.Errorf("Alternatives = %v, want %v", cls.Alternatives, c.wantAlts)
					} else {
						for i, a := range cls.Alternatives {
							if a != c.wantAlts[i] {
								t.Errorf("Alternatives[%d] = %q, want %q", i, a, c.wantAlts[i])
							}
						}
					}
				}
				return
			}
			if c.wantFact != "" {
				if cls.ConfirmedFact != c.wantFact {
					t.Errorf("ConfirmedFact = %q, want %q (SkillsToAdd=%v)", cls.ConfirmedFact, c.wantFact, cls.SkillsToAdd)
				}
				return
			}
			if len(cls.SkillsToAdd) != len(c.wantSkills) {
				t.Errorf("SkillsToAdd = %v, want %v (ConfirmedFact=%q)", cls.SkillsToAdd, c.wantSkills, cls.ConfirmedFact)
				return
			}
			for i, s := range cls.SkillsToAdd {
				if s != c.wantSkills[i] {
					t.Errorf("SkillsToAdd[%d] = %q, want %q", i, s, c.wantSkills[i])
				}
			}
		})
	}
}

// TestDismissSkillClassification verifies that IsSkillName returns the right value for the
// same cases used for ✕: a known technology or short non-verb name goes to absent-skills,
// anything else goes to a confirmed "do not have" fact.
func TestDismissSkillClassification(t *testing.T) {
	v := defaultVocabulary(t)
	cases := []struct {
		name    string
		input   string
		isSkill bool
	}{
		{"vocabulary skill: absent-skills path", "Python", true},
		{"short non-verb: absent-skills path", "GraphQL", true},
		{"phrase with ing verbs: confirmed-fact path", "Coding, debugging, reviewing", false},
		// "AI coding tools": owner decision 2026-11-13 — should be a skill.
		// "coding" is a modifier_word inside a multi-word phrase → not an activity indicator.
		{"AI coding tools: coding is a modifier → skill path", "AI coding tools", true},
		{"Coding alone → activity indicator → confirmed-fact path", "Coding", false},
		{"Code reviewing → reviewing at last position → confirmed-fact path", "Code reviewing", false},
		{"unit testing framework: testing is a modifier → skill path", "unit testing framework", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := v.IsSkillName(c.input)
			if got != c.isSkill {
				t.Errorf("IsSkillName(%q) = %v, want %v", c.input, got, c.isSkill)
			}
		})
	}
}
