package profile

import "fmt"

// legacyPreferenceKeys are the typed keys copied from the legacy search preferences on the
// one-time bootstrap. The value must be a list of strings; anything else stays only in
// LegacyPreferences. This list is intentionally conservative: it never derives skills, work
// authorization or other claims from free text.
var legacyPreferenceKeys = []string{
	"roles",
	"target_roles",
	"industries",
	"locations",
	"languages",
	"dealbreakers",
}

// bootstrapData builds an initial profile from legacy sources without changing them.
//
// The configured search preferences are the typed-import authority. The stored candidate
// context contributes its preferences only when configuration has none. Every key from both
// maps, mapped or not, is retained verbatim in LegacyPreferences so nothing is silently lost.
// Candidate source text is never parsed here: skills and legal eligibility are not invented.
func (s *Service) bootstrapData() (ProfileData, error) {
	var data ProfileData

	var configured map[string]any
	if s.cfg != nil {
		if cfg := s.cfg.Get(); cfg != nil {
			configured = cfg.Search.Preferences
		}
	}

	var fromContext map[string]any
	ctx, err := s.store.Context()
	if err != nil {
		return ProfileData{}, fmt.Errorf("read candidate context for profile bootstrap: %w", err)
	}
	if ctx != nil {
		if version, ok := ctx["version"].(string); ok {
			data.LegacyContextVersion = version
		}
		if prefs, ok := ctx["preferences"].(map[string]any); ok {
			fromContext = prefs
		}
	}

	legacy := map[string]any{}
	for k, v := range fromContext {
		legacy[k] = v
	}
	for k, v := range configured {
		legacy[k] = v
	}
	if len(legacy) > 0 {
		data.LegacyPreferences = legacy
	}

	source := configured
	if len(source) == 0 {
		source = fromContext
	}
	applyLegacyPreferences(&data, source)
	return data, nil
}

// applyLegacyPreferences copies the conservative typed subset of a legacy preference map.
// Recognised keys with a non-string-list value are left untouched (they remain in
// LegacyPreferences) rather than being coerced into a false typed value.
func applyLegacyPreferences(data *ProfileData, prefs map[string]any) {
	if values, ok := stringList(prefs["roles"]); ok {
		data.TargetRoles = appendUnique(data.TargetRoles, values...)
	}
	if values, ok := stringList(prefs["target_roles"]); ok {
		data.TargetRoles = appendUnique(data.TargetRoles, values...)
	}
	if values, ok := stringList(prefs["industries"]); ok {
		data.TargetIndustries = appendUnique(data.TargetIndustries, values...)
	}
	if values, ok := stringList(prefs["locations"]); ok {
		data.Geography.Desired = appendUnique(data.Geography.Desired, values...)
	}
	if values, ok := stringList(prefs["languages"]); ok {
		for _, name := range values {
			data.Languages = appendLanguage(data.Languages, Language{Name: name})
		}
	}
	if values, ok := stringList(prefs["dealbreakers"]); ok {
		data.Dealbreakers = appendUnique(data.Dealbreakers, values...)
	}
}

// stringList accepts the two shapes a decoded YAML/JSON list can take and rejects anything
// that would require guessing. A non-list or a list containing a non-string yields false.
func stringList(value any) ([]string, bool) {
	switch list := value.(type) {
	case []string:
		out := make([]string, 0, len(list))
		for _, s := range list {
			if s == "" {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			s, ok := item.(string)
			if !ok || s == "" {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

func appendUnique(dst []string, values ...string) []string {
	seen := make(map[string]bool, len(dst))
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range values {
		if !seen[v] {
			dst = append(dst, v)
			seen[v] = true
		}
	}
	return dst
}

func appendLanguage(dst []Language, language Language) []Language {
	for _, existing := range dst {
		if existing.Name == language.Name {
			return dst
		}
	}
	return append(dst, language)
}
