package vacancyai

import "strings"

// SkillClassification is the result of classifying a phrase from a vacancy requirement for the
// POST /api/profile/skills endpoint. Exactly one of SkillsToAdd, ConfirmedFact and HasAlternatives
// is non-zero.
type SkillClassification struct {
	// SkillsToAdd are the individual skill names to add to the profile. Each is a known technology
	// or a short non-verb phrase (≤ MaxSkillNameWords words, no word ending in a configured VerbEnding).
	SkillsToAdd []string
	// ConfirmedFact is the original text to record as a "You have: …" confirmed fact. Set when any
	// part of the input is an activity, a phrase longer than MaxSkillNameWords words, or contains a
	// verb form recognised by VerbEndings.
	ConfirmedFact string
	// HasAlternatives is true when the requirement line has discrete alternatives (e.g. "Go or Rust").
	// The UI is expected to offer each alternative as a separate button rather than adding silently.
	HasAlternatives bool
	// Alternatives holds the detected alternatives when HasAlternatives is true and the server
	// detected them from the name itself (e.g. "Go or Rust" → ["Go", "Rust"]). Empty when the
	// alternatives came from the caller's body (pre-parsed from the job extraction result).
	Alternatives []string
}

// ClassifySkillAdd classifies input arriving at POST /api/profile/skills. The name is the raw text
// from the requirement row; alternatives are pre-parsed alternatives from the job extraction result,
// when any. When alternatives are present the classification delegates to the UI; each alternative is
// a separate click that arrives here without alternatives set.
// The server also detects "X or Y" and "X/Y" patterns in the name itself: when each part is a
// valid skill name (single-word or vocabulary-known), it returns HasAlternatives with Alternatives
// populated from the parsed parts.
func (v *Vocabulary) ClassifySkillAdd(name string, alternatives []string) SkillClassification {
	if len(alternatives) > 0 {
		return SkillClassification{HasAlternatives: true}
	}

	// Detect "X or Y" alternatives: split on " or " and check each part is a valid alternative name.
	// "Go or Rust" → ["Go", "Rust"]; "Familiarity with Go or Rust" → "Familiarity with Go" is not
	// a single-word or vocab-known name → ConfirmedFact (phrase, not a skill). A phrase that
	// contains " or " but whose parts are not valid alternatives is never added as a single skill.
	if idx := strings.Index(strings.ToLower(name), " or "); idx >= 0 {
		parts := splitByOr(name)
		if len(parts) >= 2 {
			allValid := true
			for _, p := range parts {
				if !v.isAlternativeName(p) {
					allValid = false
					break
				}
			}
			if allValid {
				return SkillClassification{HasAlternatives: true, Alternatives: parts}
			}
			// Parts not all valid: treat the whole phrase as a confirmed fact, not a skill.
			return SkillClassification{ConfirmedFact: strings.TrimSpace(name)}
		}
	}

	// Detect "X/Y" alternatives. "/" is treated as an alternative separator when every part passes
	// isAlternativeName (vocabulary-known OR single-word IsSkillName), consistent with the "or"
	// check above and with codeMatch. "Datadog / OpenSearch" → alternatives (Datadog single-word
	// skill, OpenSearch vocabulary). A phrase where any part fails (e.g. "reporting/planning",
	// where "reporting" is an -ing word) is returned as a confirmed fact, not a skill.
	if strings.Contains(name, "/") {
		parts := splitBySlash(name)
		if len(parts) >= 2 {
			allValid := true
			for _, p := range parts {
				if !v.isAlternativeName(p) {
					allValid = false
					break
				}
			}
			if allValid {
				return SkillClassification{HasAlternatives: true, Alternatives: parts}
			}
			// Parts not all valid: treat as confirmed fact, not a skill.
			return SkillClassification{ConfirmedFact: strings.TrimSpace(name)}
		}
	}

	// Split a comma-separated list; "TypeScript, Node" → ["TypeScript", "Node"].
	parts := splitSkillParts(name)
	var skills []string
	for _, part := range parts {
		if v.IsSkillName(part) {
			skills = append(skills, part)
		} else {
			// Any part that is not a skill makes the whole original phrase a confirmed fact.
			return SkillClassification{ConfirmedFact: strings.TrimSpace(name)}
		}
	}
	if len(skills) == 0 {
		return SkillClassification{ConfirmedFact: strings.TrimSpace(name)}
	}
	return SkillClassification{SkillsToAdd: skills}
}

// isAlternativeName reports whether a name qualifies as one member of a server-detected "X or Y"
// alternatives list. A name qualifies when it is vocabulary-known (any word count) or a single
// word that IsSkillName accepts. Multi-word non-vocab names do not qualify, so "Familiarity with Go"
// does not qualify and "Familiarity with Go or Rust" falls through to the confirmed-fact path.
func (v *Vocabulary) isAlternativeName(name string) bool {
	name = strings.TrimSpace(name)
	toks := tokenize(name)
	if len(toks) == 0 {
		return false
	}
	// Vocabulary-known names (any length) qualify as alternatives.
	// Use isFullVocabMatch (exact match) rather than scanName (partial scan), so that a name like
	// "Familiarity with Go" does not qualify just because it contains the technology "Go".
	if v.isFullVocabMatch(name) {
		return true
	}
	// Single-word names: apply IsSkillName rules.
	if len(toks) == 1 {
		return v.IsSkillName(name)
	}
	return false
}

// splitByOr splits a phrase on " or " (case-insensitive) and trims whitespace from each part.
func splitByOr(name string) []string {
	lower := strings.ToLower(name)
	var parts []string
	for {
		idx := strings.Index(lower, " or ")
		if idx < 0 {
			break
		}
		parts = append(parts, strings.TrimSpace(name[:idx]))
		name = name[idx+4:]
		lower = lower[idx+4:]
	}
	if s := strings.TrimSpace(name); s != "" {
		parts = append(parts, s)
	}
	return parts
}

// splitBySlash splits a phrase on "/" (with optional surrounding spaces) and trims each part.
func splitBySlash(name string) []string {
	raw := strings.Split(name, "/")
	var parts []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// IsSkillName reports whether a single name (one part of a comma-separated list, or the whole
// subject when there is no list) should be recorded as a profile skill rather than a confirmed fact.
// A name qualifies when it is a known vocabulary technology or a short non-verb phrase: at most
// MaxSkillNameWords words and no word ending with a configured VerbEnding.
func (v *Vocabulary) IsSkillName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	// A technology the vocabulary knows exactly is always a skill. Use isFullVocabMatch (exact match)
	// rather than scanName (partial scan), so that a phrase like "Familiarity with Go or Rust" does
	// not pass just because it contains known technologies.
	if v.isFullVocabMatch(name) {
		return true
	}
	// Short-name rule: word count and verb-ending check.
	toks := tokenize(name)
	if len(toks) == 0 {
		return false
	}
	limit := 3
	if v != nil && v.maxSkillNameWords > 0 {
		limit = v.maxSkillNameWords
	}
	if len(toks) > limit {
		return false
	}
	endings := []string{"ing"}
	if v != nil && len(v.verbEndings) > 0 {
		endings = v.verbEndings
	}
	for i, t := range toks {
		lower := strings.ToLower(t.text)
		for _, e := range endings {
			if !strings.HasSuffix(lower, strings.ToLower(e)) {
				continue
			}
			// This word ends in a verb ending. Check whether it is a modifier word (from config).
			// A modifier word (e.g. "coding", "testing") acts as an adjective inside a multi-word
			// phrase and is NOT an activity indicator when the phrase has at least 2 words.
			// "AI coding tools" → "coding" is a modifier (phrase has 3 words) → continue.
			// "Coding" alone → "Coding" is standalone (phrase has 1 word) → activity → false.
			// (Owner decision 2026-11-13: "AI coding tools" must remain a skill.)
			isModifier := v != nil && v.modifierWords[lower]
			if isModifier {
				if len(toks) >= 2 {
					// Modifier word inside multi-word phrase: not an activity indicator.
					continue
				}
				// Standalone modifier word: treated as an activity.
				return false
			}
			// Non-modifier -ing word: activity indicator only at the first or last position.
			// A word in the middle of a phrase (e.g. "AI scripting language" if "scripting" were
			// not a modifier) is not treated as an activity when it is not leading or trailing.
			if i == 0 || i == len(toks)-1 {
				return false
			}
		}
	}
	return true
}

// splitSkillParts splits a comma-separated list of skill names. Commas and semicolons are
// separators; slashes, ampersands and other characters stay inside a name.
func splitSkillParts(name string) []string {
	raw := strings.FieldsFunc(name, func(r rune) bool { return r == ',' || r == ';' })
	var parts []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		if s := strings.TrimSpace(name); s != "" {
			return []string{s}
		}
	}
	return parts
}
