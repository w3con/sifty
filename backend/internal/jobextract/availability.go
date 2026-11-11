package jobextract

import (
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// SourceStateOf says whether a retained vacancy carries text the two-score model can analyse.
// It prefers the collector's own record of the description (metadata only, pending, unavailable,
// blocked or missing all mean there is no vacancy text), and it treats a description that merely
// repeats the title or the tags as no text either. A genuine short description is never refused
// for its length. It performs no I/O and never fetches anything.
func SourceStateOf(src matchcontract.RetainedSource) matchcontract.SourceState {
	status := strings.ToLower(strings.TrimSpace(src.DescriptionStatus))
	for _, word := range []string{"metadata", "pending", "unavailable", "error", "blocked", "missing"} {
		if strings.Contains(status, word) {
			return matchcontract.SourceUnavailable
		}
	}
	if strings.TrimSpace(src.Description) == "" || metadataOnlyText(src) {
		return matchcontract.SourceUnavailable
	}
	return matchcontract.SourceUsable
}

// metadataOnlyText reports the narrow case where the stored description is exactly the saved
// title, exactly the saved tag list, or both.
func metadataOnlyText(src matchcontract.RetainedSource) bool {
	desc := normalizeSourceText(src.Description)
	if desc == "" {
		return false
	}
	title := normalizeSourceText(src.Title)
	if title != "" && desc == title {
		return true
	}
	var tags []string
	for _, tag := range src.Tags {
		if normalized := normalizeSourceText(tag); normalized != "" {
			tags = append(tags, normalized)
		}
	}
	if len(tags) == 0 {
		return false
	}
	joined := strings.Join(tags, " ")
	if desc == joined {
		return true
	}
	return title != "" && (desc == title+" "+joined || desc == joined+" "+title)
}

func normalizeSourceText(value string) string {
	replacer := strings.NewReplacer(",", " ", ";", " ", ":", " ")
	return strings.ToLower(strings.Join(strings.Fields(replacer.Replace(value)), " "))
}
