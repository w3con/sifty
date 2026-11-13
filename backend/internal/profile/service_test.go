package profile_test

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// TestBootstrapPreservesLegacyWithoutInventing catches two hidden bootstrap faults: losing
// unknown legacy preference keys, and turning free evidence text into confirmed skills or work
// authorization. Either would silently misrepresent the candidate.
func TestBootstrapPreservesLegacyWithoutInventing(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	if err := s.SaveContext(map[string]any{
		"version": "legacy-context-v1",
		"preferences": map[string]any{
			"context_only": "kept",
		},
		"sources": []any{
			map[string]any{
				"id":   "cv",
				"text": "Ten years of Go. Citizen of France. Available immediately.",
			},
		},
	}); err != nil {
		t.Fatalf("save context: %v", err)
	}

	cfg := &config.Config{Search: config.Search{Preferences: map[string]any{
		"roles":        []any{"Backend Engineer"},
		"industries":   []any{"Software"},
		"locations":    []any{"Berlin"},
		"languages":    []any{"English"},
		"dealbreakers": []any{"night shifts"},
		"unknown_flag": true,
	}}}
	svc := profile.NewService(s, config.NewHolder(cfg))

	got, err := svc.Get()
	if err != nil {
		t.Fatalf("bootstrap profile: %v", err)
	}
	if got.ID != profile.LocalProfileID {
		t.Errorf("profile id = %q, want %q", got.ID, profile.LocalProfileID)
	}
	if got.Revision != 1 {
		t.Errorf("bootstrap revision = %d, want 1", got.Revision)
	}
	// A newly bootstrapped profile must start with empty confirmed typed fields even when
	// legacy configuration and context carry values; those values are provenance only.
	empty := profile.ProfileData{}
	empty.Normalize()
	typed := got.Data
	typed.LegacyPreferences = nil
	typed.LegacyContextVersion = ""
	if !reflect.DeepEqual(typed, empty) {
		t.Errorf("initial typed profile = %+v, want all typed fields empty", got.Data)
	}
	if got.Data.LegacyContextVersion != "legacy-context-v1" {
		t.Errorf("legacy context version = %q, want legacy-context-v1", got.Data.LegacyContextVersion)
	}
	for _, key := range []string{"roles", "industries", "locations", "languages", "dealbreakers", "unknown_flag", "context_only"} {
		if _, ok := got.Data.LegacyPreferences[key]; !ok {
			t.Errorf("legacyPreferences lost key %q", key)
		}
	}
	if got.Data.LegacyPreferences["unknown_flag"] != true {
		t.Errorf("legacyPreferences unknown_flag = %v, want true", got.Data.LegacyPreferences["unknown_flag"])
	}
	if got.Data.LegacyPreferences["context_only"] != "kept" {
		t.Errorf("legacyPreferences context_only = %v, want kept", got.Data.LegacyPreferences["context_only"])
	}
	if _, ok := cfg.Search.Preferences["roles"]; !ok {
		t.Error("bootstrap mutated configured search preferences")
	}
	ctx, err := s.Context()
	if err != nil {
		t.Fatalf("read preserved context: %v", err)
	}
	if prefs, ok := ctx["preferences"].(map[string]any); !ok || prefs["context_only"] != "kept" {
		t.Errorf("bootstrap mutated stored candidate context preferences: %v", ctx["preferences"])
	}
	if len(got.Data.Skills) != 0 {
		t.Errorf("skills = %v, want none invented from free text", got.Data.Skills)
	}
	if len(got.Data.Geography.Eligibility) != 0 {
		t.Errorf("eligibility = %v, want unknown rather than invented authorization", got.Data.Geography.Eligibility)
	}

	// The service must surface a stale save as a conflict carrying the current revision, never
	// overwrite the newer profile.
	_, err = svc.Update(0, profile.ProfileData{})
	var conflict *profile.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale update error = %v, want *profile.ConflictError", err)
	}
	if conflict.Current == nil || conflict.Current.Revision != 1 {
		t.Fatalf("conflict current = %+v, want revision 1", conflict.Current)
	}

	// An editor that omits the server-owned bootstrap provenance must not drop it.
	partial, err := svc.Update(1, profile.ProfileData{
		Identity: profile.Identity{DisplayName: "Owner"},
	})
	if err != nil {
		t.Fatalf("partial update: %v", err)
	}
	if partial.Revision != 2 {
		t.Fatalf("partial update revision = %d, want 2", partial.Revision)
	}
	if partial.Data.LegacyContextVersion != "legacy-context-v1" {
		t.Errorf("partial update dropped legacyContextVersion: %q", partial.Data.LegacyContextVersion)
	}
	if _, ok := partial.Data.LegacyPreferences["context_only"]; !ok {
		t.Error("partial update dropped legacyPreferences")
	}

	// A normal GET must return the stored content unchanged, not re-bootstrap an empty or
	// legacy-imported profile over it.
	current, err := svc.Get()
	if err != nil {
		t.Fatalf("get stored profile: %v", err)
	}
	if current.Revision != 2 {
		t.Errorf("stored profile revision after get = %d, want 2", current.Revision)
	}
	if current.Data.Identity.DisplayName != "Owner" {
		t.Errorf("stored profile display name = %q, want Owner", current.Data.Identity.DisplayName)
	}
	if current.Data.LegacyContextVersion != "legacy-context-v1" {
		t.Errorf("stored profile lost legacyContextVersion: %q", current.Data.LegacyContextVersion)
	}
	if len(current.Data.TargetRoles) != 0 {
		t.Errorf("stored profile target roles = %v, want still empty", current.Data.TargetRoles)
	}

	// A client cannot forge or replace the same fields.
	forged, err := svc.Update(2, profile.ProfileData{
		LegacyPreferences:    map[string]any{"forged": true},
		LegacyContextVersion: "forged",
	})
	if err != nil {
		t.Fatalf("update with forged provenance: %v", err)
	}
	if forged.Data.LegacyContextVersion != "legacy-context-v1" {
		t.Errorf("forged legacyContextVersion accepted: %q", forged.Data.LegacyContextVersion)
	}
	if _, ok := forged.Data.LegacyPreferences["forged"]; ok {
		t.Error("forged legacyPreferences key accepted")
	}
	if _, ok := forged.Data.LegacyPreferences["context_only"]; !ok {
		t.Error("server legacyPreferences lost during forged update")
	}
}
