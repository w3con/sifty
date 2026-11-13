package vacancyview

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/model"
)

// ComputeOfferItems builds the What they offer tile list.
//
// Order: (1) salary tile from the money result when available, (2) vocabulary items found in
// Compensation or Benefits sections (unfound items are omitted).
//
// When money is nil and no Compensation or Benefits section exists, nil is returned and the
// caller shows "No Compensation or Benefits section recognised in this listing."
func ComputeOfferItems(sections []model.Section, cfg config.VacancyViewConfig, money *MoneyResult) []model.OfferItem {
	var items []model.OfferItem

	// Salary tile first, taken from the band money result.
	if money != nil && money.OriginalRange != "" {
		label := "Base salary"
		switch {
		case money.Currency != "" && money.Period != "":
			label = fmt.Sprintf("Base salary, %s / %s", money.Currency, money.Period)
		case money.Currency != "":
			label = fmt.Sprintf("Base salary, %s", money.Currency)
		case money.Period != "":
			label = fmt.Sprintf("Base salary / %s", money.Period)
		}
		items = append(items, model.OfferItem{Key: "salary", Label: label, Value: money.ShortRange})
	}

	// Vocabulary items: search Compensation and Benefits sections; include only found items.
	var offerText strings.Builder
	hasOfferSection := false
	for _, s := range sections {
		if s.Class == "Compensation" || s.Class == "Benefits" {
			offerText.WriteString(s.Content)
			offerText.WriteString("\n")
			hasOfferSection = true
		}
	}

	if hasOfferSection {
		text := strings.ToLower(offerText.String())
		for _, def := range cfg.OfferVocabulary {
			found := false
			for _, kw := range def.Keywords {
				if containsWord(text, strings.ToLower(kw)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			notStated := def.NotStatedLabel
			if notStated == "" {
				notStated = "Amount not stated"
			}
			items = append(items, model.OfferItem{Key: def.Key, Label: def.Label, Value: notStated})
		}
	}

	if len(items) == 0 {
		return nil
	}
	return items
}

// containsWord reports whether text contains the keyword as a word or phrase (not inside another word).
func containsWord(text, keyword string) bool {
	if !strings.Contains(text, keyword) {
		return false
	}
	// For multi-word phrases, substring match is fine.
	if strings.ContainsRune(keyword, ' ') {
		return true
	}
	// For single words, check word boundaries.
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
	return re.MatchString(text)
}
