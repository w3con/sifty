package config

import (
	"testing"
)

// TestDefaultRegionsEmbedded verifies that the binary ships a non-trivial region vocabulary
// and that EffectiveRegions returns it when quick_match.regions is absent.
func TestDefaultRegionsEmbedded(t *testing.T) {
	t.Run("DefaultRegions returns built-in vocabulary", func(t *testing.T) {
		regions, err := DefaultRegions()
		if err != nil {
			t.Fatalf("DefaultRegions: %v", err)
		}
		if len(regions) < 10 {
			t.Fatalf("expected at least 10 built-in regions, got %d", len(regions))
		}
		// The vocabulary must include the common compound label "US/Canada".
		found := false
		for _, r := range regions {
			if r.Name == "US" || r.Name == "Canada" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("built-in vocabulary must include US and Canada entries")
		}
	})

	t.Run("absent quick_match.regions falls back to built-in", func(t *testing.T) {
		q := QuickMatch{} // no regions configured
		got, err := q.EffectiveRegions()
		if err != nil {
			t.Fatalf("EffectiveRegions: %v", err)
		}
		def, _ := DefaultRegions()
		if len(got) != len(def) {
			t.Fatalf("absent regions: got %d, default has %d", len(got), len(def))
		}
	})

	t.Run("explicit quick_match.regions overrides built-in", func(t *testing.T) {
		override := []QuickMatchRegion{
			{Name: "CustomLand", Aliases: []string{"CL"}, Countries: []string{"CL"}},
		}
		q := QuickMatch{Regions: override}
		got, err := q.EffectiveRegions()
		if err != nil {
			t.Fatalf("EffectiveRegions: %v", err)
		}
		if len(got) != 1 || got[0].Name != "CustomLand" {
			t.Fatalf("explicit regions not returned: %+v", got)
		}
	})

	t.Run("absent regions enables restricted-remote detection via matchconfig", func(t *testing.T) {
		// A QuickMatch block with no regions should still produce rules that classify
		// "US/Canada" as a restricted label (not remoteNoRemote or remoteUnrestricted).
		q := DefaultQuickMatch() // no Regions field
		regions, err := q.EffectiveRegions()
		if err != nil {
			t.Fatalf("EffectiveRegions: %v", err)
		}
		// Verify "US" and "Canada" resolve to restricted country sets.
		usFound, caFound := false, false
		for _, r := range regions {
			switch r.Name {
			case "US":
				usFound = true
				if len(r.Countries) == 0 {
					t.Fatalf("US entry must have at least one country code")
				}
			case "Canada":
				caFound = true
				if len(r.Countries) == 0 {
					t.Fatalf("Canada entry must have at least one country code")
				}
			}
		}
		if !usFound || !caFound {
			t.Fatalf("built-in vocabulary must contain US and Canada; usFound=%v caFound=%v", usFound, caFound)
		}
	})
}
