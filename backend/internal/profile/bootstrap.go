package profile

import "fmt"

// bootstrapData builds an initial profile without importing typed claims.
//
// The configured search preferences and the stored candidate context are retained verbatim in
// LegacyPreferences for provenance; LegacyContextVersion records the context snapshot. Neither
// source is applied as confirmed typed fields: a new profile starts empty and the owner confirms
// values explicitly. Candidate source text is never parsed here, so skills and legal eligibility
// are not invented.
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
	return data, nil
}
