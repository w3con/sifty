package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/prompts"
	"github.com/w3con/sifty/internal/store"
)

// isPerItemField reports whether facts of this kind may arrive from the model as an array
// that must be expanded into individual items, each anchored with its own quote.
func isPerItemField(kind factKind) bool {
	switch kind {
	case factSkill, factLanguage, factAuthorization, factExperience:
		return true
	}
	return false
}

// itemName returns the searchable display name from one expanded item value. For skills it is
// the skill name; for languages the language name; for experience the company name. Returns ""
// when no name can be extracted or the field kind does not support name-based fallback.
func itemName(fieldKey string, raw json.RawMessage) string {
	switch fieldKey {
	case "skills":
		var sk Skill
		if json.Unmarshal(raw, &sk) == nil && strings.TrimSpace(sk.Name) != "" {
			return strings.TrimSpace(sk.Name)
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return strings.TrimSpace(s)
		}
	case "languages":
		var lang Language
		if json.Unmarshal(raw, &lang) == nil && strings.TrimSpace(lang.Name) != "" {
			return strings.TrimSpace(lang.Name)
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return strings.TrimSpace(s)
		}
	case "experience":
		var exp Experience
		if json.Unmarshal(raw, &exp) == nil && strings.TrimSpace(exp.Company) != "" {
			return strings.TrimSpace(exp.Company)
		}
	}
	return ""
}

// isWordRune reports whether r counts as a word character for boundary detection.
// Only ASCII is considered; skill and language names in CVs are almost always ASCII.
func isWordRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// itemNameQuote searches source for name as a complete, word-boundary–delimited token and
// returns the matched substring. A short name such as "Go" or "C" only matches when it is
// not immediately preceded or followed by another word character — "Go" does not match
// "Golang" and "C" does not match "CTO". Internal whitespace in name is treated as \s+ so
// a name from structured output is still found when the document uses different spacing.
func itemNameQuote(source, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	words := strings.Fields(name)
	if len(words) == 0 {
		return "", false
	}
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = regexp.QuoteMeta(w)
	}
	inner := strings.Join(parts, `\s+`)

	firstRune, _ := utf8.DecodeRuneInString(name)
	lastRune, _ := utf8.DecodeLastRuneInString(name)
	var pre, suf string
	if isWordRune(firstRune) {
		pre = `\b`
	}
	if isWordRune(lastRune) {
		suf = `\b`
	}
	re, err := regexp.Compile(pre + inner + suf)
	if err != nil {
		return "", false
	}
	if m := re.FindString(source); m != "" {
		return m, true
	}
	return "", false
}

// quoteForItem resolves the quote for one expanded fact item. It first tries the shared
// group quote from the model output; if that fails it falls back to a word-boundary search
// for the item's own name in the full source text. Returns the matched quote and true if any
// anchor is found; returns "", false if neither the group quote nor the name is in the source.
func quoteForItem(source, fieldKey, groupQuote string, raw json.RawMessage) (string, bool) {
	if q, ok := exactQuote(source, groupQuote); ok {
		return q, true
	}
	name := itemName(fieldKey, raw)
	if name == "" {
		return "", false
	}
	return itemNameQuote(source, name)
}

type extractedFact struct {
	FieldKey string          `json:"fieldKey"`
	Value    json.RawMessage `json:"value"`
	Quote    string          `json:"quote"`
}

type extractedFacts struct {
	Facts []extractedFact `json:"facts"`
}

var factSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"facts": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fieldKey": map[string]any{"type": "string", "enum": AllowedFactFieldKeys()},
					"value":    map[string]any{"type": []string{"string", "number", "boolean", "array", "object"}},
					"quote":    map[string]any{"type": "string"},
				},
				"required":             []string{"fieldKey", "value", "quote"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"facts"},
	"additionalProperties": false,
}

// normalizeFactValue pre-processes a model-emitted fact value before schema validation.
// It converts a bare-string skill to a Skill object, a bare-string language to a Language
// object, and normalises the case of enum values to their canonical form.
func normalizeFactValue(fieldKey string, raw json.RawMessage) json.RawMessage {
	switch fieldKey {
	case "skills":
		// A bare name string → {key, name, level: "unstated"}.
		var s string
		if json.Unmarshal(raw, &s) == nil {
			s = strings.TrimSpace(s)
			if s != "" {
				sk := Skill{Key: skillKeyOf(s), Name: s, Level: SkillUnstated}
				if b, err := json.Marshal(sk); err == nil {
					return b
				}
			}
			return raw
		}
		// Object with missing key: derive from name.
		var sk Skill
		if json.Unmarshal(raw, &sk) == nil && strings.TrimSpace(sk.Key) == "" && strings.TrimSpace(sk.Name) != "" {
			sk.Key = skillKeyOf(sk.Name)
			if b, err := json.Marshal(sk); err == nil {
				return b
			}
		}

	case "languages":
		// A bare name string → {name, level: ""}.
		var s string
		if json.Unmarshal(raw, &s) == nil {
			s = strings.TrimSpace(s)
			if s != "" {
				lang := Language{Name: s}
				if b, err := json.Marshal(lang); err == nil {
					return b
				}
			}
		}
	}

	// For enum lists (workModes.modes, engagement.types): normalise each item to the canonical
	// case defined by the enum map so "Remote" is accepted the same as "remote".
	if f, ok := factFields[fieldKey]; ok {
		if f.kind == factEnumList && f.enum != nil {
			var items []string
			if json.Unmarshal(raw, &items) == nil {
				for i, item := range items {
					for valid := range f.enum {
						if strings.EqualFold(item, valid) {
							items[i] = valid
							break
						}
					}
				}
				if b, err := json.Marshal(items); err == nil {
					return b
				}
			}
		}
		// For single-value enum strings (hours.mode, etc.): normalise to canonical case.
		if f.kind == factString && f.enum != nil {
			var s string
			if json.Unmarshal(raw, &s) == nil {
				for valid := range f.enum {
					if strings.EqualFold(s, valid) {
						if b, err := json.Marshal(valid); err == nil {
							return b
						}
					}
				}
			}
		}
	}

	return raw
}

// splitTextIntoChunks splits a document text into parts that each fit within maxChars bytes.
// It tries to split at paragraph boundaries (double newline) or single newlines; when neither
// falls within the window it retreats to the nearest valid UTF-8 rune boundary so it never
// cuts inside a multi-byte character (Cyrillic, accented Latin, CJK, etc.).
// Both maxChars and len(text) are measured in bytes, consistent with the context-overflow check
// in assess.InferInto which also uses len() (bytes) to compare against ContextChars.
func splitTextIntoChunks(text string, maxChars int) []string {
	if maxChars <= 0 || len(text) <= maxChars {
		return []string{text}
	}
	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxChars {
			chunks = append(chunks, text)
			break
		}
		end := maxChars
		// Try to split at a paragraph break first, then a line break.
		if idx := strings.LastIndex(text[:end], "\n\n"); idx > maxChars/4 {
			end = idx
		} else if idx := strings.LastIndex(text[:end], "\n"); idx > maxChars/4 {
			end = idx
		} else {
			// No line boundary: retreat to a valid rune start so we do not cut inside a
			// multi-byte UTF-8 sequence (e.g. Cyrillic or accented characters).
			for end > 0 && !utf8.RuneStart(text[end]) {
				end--
			}
		}
		chunks = append(chunks, strings.TrimRight(text[:end], "\n "))
		text = strings.TrimLeft(text[end:], "\n ")
	}
	return chunks
}

// expandValueItems handles the case where the model returns a JSON array for a field that expects
// individual objects (skills, languages, geography.workAuthorizations, experience). The model
// often groups all values into one array fact instead of emitting one fact per value. Each array
// element becomes a separate value to process independently through normalizeFactValue and
// ValidateFactValue. For fields that legitimately accept arrays as their value (factEnumList,
// factString, etc.) the value is returned unchanged.
func expandValueItems(fieldKey string, raw json.RawMessage) []json.RawMessage {
	def, ok := factFields[fieldKey]
	if !ok {
		return []json.RawMessage{raw}
	}
	switch def.kind {
	case factSkill, factLanguage, factAuthorization, factExperience:
		// These fields expect exactly one value object per fact. If the model wrapped several in
		// an array, unpack them. A single non-array value is left as-is.
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) == nil && len(items) > 0 {
			return items
		}
	}
	return []json.RawMessage{raw}
}

// deduplicateFactsByValue removes exact duplicates (same fieldKey and value JSON) keeping the
// first occurrence. This prevents the same skill or language from appearing twice when it appears
// in multiple chunks of a split document.
func deduplicateFactsByValue(facts []*store.ProfileFactRecord) []*store.ProfileFactRecord {
	seen := map[string]bool{}
	out := make([]*store.ProfileFactRecord, 0, len(facts))
	for _, f := range facts {
		key := f.FieldKey + "\x00" + f.ValueJSON
		if !seen[key] {
			seen[key] = true
			out = append(out, f)
		}
	}
	return out
}

// matchFactToProfile reports whether the extracted fact value is already present in the confirmed
// profile. For experience entries it uses company name plus period overlap (more permissive than
// exact title match) to avoid marking a CV entry as new when the same role is already stored
// under a slightly different title. For all other fields it calls ApplyFact on a detached copy
// and checks whether applying the fact would leave the profile unchanged.
func matchFactToProfile(data ProfileData, fact *store.ProfileFactRecord) bool {
	if fact.FieldKey == "experience" {
		return experienceAlreadyInProfile(data, json.RawMessage(fact.ValueJSON))
	}
	// Work with a shallow copy so ApplyFact slice mutations never reach the caller's data.
	cp := data
	cp.TargetRoles = append([]string(nil), data.TargetRoles...)
	cp.TargetIndustries = append([]string(nil), data.TargetIndustries...)
	cp.WorkModes.Modes = append([]string(nil), data.WorkModes.Modes...)
	cp.WorkModes.Limits = append([]string(nil), data.WorkModes.Limits...)
	cp.Geography.Desired = append([]string(nil), data.Geography.Desired...)
	cp.Geography.Eligibility = append([]string(nil), data.Geography.Eligibility...)
	cp.Geography.Relocation = append([]string(nil), data.Geography.Relocation...)
	cp.Geography.Timezones = append([]string(nil), data.Geography.Timezones...)
	cp.Dealbreakers = append([]string(nil), data.Dealbreakers...)
	cp.Engagement.Types = append([]string(nil), data.Engagement.Types...)
	cp.Skills = append([]Skill(nil), data.Skills...)
	cp.Languages = append([]Language(nil), data.Languages...)
	cp.Geography.WorkAuthorizations = append([]WorkAuthorization(nil), data.Geography.WorkAuthorizations...)

	changed, conflict, err := ApplyFact(&cp, fact.FieldKey, json.RawMessage(fact.ValueJSON), false)
	// changed=false && conflict==nil && err==nil means the value is already in the profile.
	return err == nil && !changed && conflict == nil
}

// experienceAlreadyInProfile checks whether the proposed experience entry matches one already in
// the profile by company name and either: (a) the same title, or (b) an overlapping date span.
// Same company + overlapping period is treated as the same role even when the title differs
// slightly, so a CV that lists a promotion as a separate row does not cause a duplicate.
func experienceAlreadyInProfile(data ProfileData, raw json.RawMessage) bool {
	var proposed Experience
	if json.Unmarshal(raw, &proposed) != nil {
		return false
	}
	proposed.Company = strings.TrimSpace(proposed.Company)
	if proposed.Company == "" {
		return false
	}
	now := time.Now()
	for _, e := range data.Experience {
		if !strings.EqualFold(strings.TrimSpace(e.Company), proposed.Company) {
			continue
		}
		// Same company: exact title match is sufficient.
		if strings.EqualFold(strings.TrimSpace(e.Title), strings.TrimSpace(proposed.Title)) {
			return true
		}
		// Also match on overlapping date spans.
		existingSpan, existingOK := e.span(now)
		proposedSpan, proposedOK := proposed.span(now)
		if existingOK && proposedOK &&
			existingSpan.from <= proposedSpan.to && proposedSpan.from <= existingSpan.to {
			return true
		}
	}
	return false
}

// extractFacts calls the configured extraction stage through the existing assess provider
// adapters and returns canonical, quote-anchored proposals. It never writes the profile.
// When the document text exceeds the provider's context budget it is split into parts and
// the proposals from all parts are merged and de-duplicated.
func (s *Service) extractFacts(ctx context.Context, rec *store.ProfileDocumentRecord, runID, providerName string, provider providerConfig) ([]*store.ProfileFactRecord, map[string]any, error) {
	cfg := s.cfg.Get()
	schemaJSON, err := json.Marshal(factSchema)
	if err != nil {
		return nil, nil, err
	}

	// Calculate the per-document budget: total context minus the fixed overhead of the prompt,
	// schema header and document header.
	const headers = "\nSCHEMA\n\nDOCUMENT\n"
	overhead := len(prompts.ProfileFacts) + len(headers) + len(schemaJSON)
	documentBudget := provider.ContextChars - overhead

	chunks := splitTextIntoChunks(rec.ExtractedText, documentBudget)

	p := assess.Provider{
		Type: provider.Type, Model: provider.Model, Endpoint: provider.Endpoint,
		APIKeyEnv: provider.APIKeyEnv, TimeoutMS: provider.TimeoutMS, ContextChars: provider.ContextChars,
		Think: provider.Think,
	}

	maxFacts := cfg.Runtime.EffectiveProfileMaxFacts()
	var allFacts []*store.ProfileFactRecord
	var allSkipped []map[string]string

	for _, chunk := range chunks {
		requestText := fmt.Sprintf("%s\nSCHEMA\n%s\nDOCUMENT\n%s",
			prompts.ProfileFacts, schemaJSON, chunk)

		var output extractedFacts
		if err := assess.InferInto(ctx, p, requestText, factSchema, &output); err != nil {
			return nil, nil, fmt.Errorf("extraction provider %q failed: %w", providerName, err)
		}

		for _, item := range output.Facts {
			fieldKey := strings.TrimSpace(item.FieldKey)
			docID := rec.ID
			sourceRef := rec.ID
			if rec.ContentHash != "" {
				sourceRef += "@" + rec.ContentHash
			}

			// Fields that the model may group into one array fact (skills, languages,
			// authorizations, experience) are anchored per-item: if the shared group quote
			// fails exactQuote, each item's own name is searched as a whole word in the
			// document, so a near-literal group quote no longer drops the whole batch.
			if def, known := factFields[fieldKey]; known && isPerItemField(def.kind) {
				values := expandValueItems(fieldKey, item.Value)
				for _, val := range values {
					q, qok := quoteForItem(rec.ExtractedText, fieldKey, item.Quote, val)
					if !qok {
						name := itemName(fieldKey, val)
						reason := "quote is not an exact excerpt of the document"
						if name != "" {
							reason = fmt.Sprintf("%q: name not found in document as a whole word", name)
						}
						allSkipped = append(allSkipped, map[string]string{"fieldKey": fieldKey, "reason": reason})
						continue
					}
					normalised := normalizeFactValue(fieldKey, val)
					canonical, err := ValidateFactValue(fieldKey, normalised)
					if err != nil {
						allSkipped = append(allSkipped, map[string]string{"fieldKey": fieldKey, "reason": err.Error()})
						continue
					}
					allFacts = append(allFacts, &store.ProfileFactRecord{
						ID:              uuid.NewString(),
						ProfileID:       rec.ProfileID,
						DocumentID:      &docID,
						FieldKey:        fieldKey,
						ValueJSON:       string(canonical),
						Quote:           q,
						SourceReference: sourceRef,
						Origin:          "document",
						State:           FactProposed,
					})
				}
				continue
			}

			// All other fields require an exact group quote from the model.
			quote, ok := exactQuote(rec.ExtractedText, item.Quote)
			if !ok {
				allSkipped = append(allSkipped, map[string]string{"fieldKey": fieldKey, "reason": "quote is not an exact excerpt of the document"})
				continue
			}
			values := expandValueItems(fieldKey, item.Value)
			for _, val := range values {
				normalised := normalizeFactValue(fieldKey, val)
				canonical, err := ValidateFactValue(fieldKey, normalised)
				if err != nil {
					allSkipped = append(allSkipped, map[string]string{"fieldKey": fieldKey, "reason": err.Error()})
					continue
				}
				allFacts = append(allFacts, &store.ProfileFactRecord{
					ID:              uuid.NewString(),
					ProfileID:       rec.ProfileID,
					DocumentID:      &docID,
					FieldKey:        fieldKey,
					ValueJSON:       string(canonical),
					Quote:           quote,
					SourceReference: sourceRef,
					Origin:          "document",
					State:           FactProposed,
				})
			}
		}
	}

	// Remove exact duplicates that arise when the same item appears in several chunks.
	allFacts = deduplicateFactsByValue(allFacts)

	if len(allFacts) > maxFacts {
		return nil, nil, fmt.Errorf("analysis produced %d facts; configured limit is %d; no facts were stored", len(allFacts), maxFacts)
	}

	metadata := map[string]any{
		"runId": runID, "finishedAt": nowISO(), "phase": "analysis",
		"provider": providerName, "model": provider.Model, "contentHash": rec.ContentHash,
		"factCount": len(allFacts), "documentChars": len(rec.ExtractedText),
	}
	if len(chunks) > 1 {
		metadata["chunkCount"] = len(chunks)
	}
	if len(allSkipped) > 0 {
		metadata["skipped"] = allSkipped
	}
	return allFacts, metadata, nil
}

// analysisErrorKind classifies one failed extraction call into a stable, presentation-facing
// cause. The UI maps the kind to a plain explanation and next action; the original error remains
// available as technical detail. Unknown failures stay provider_error rather than being guessed.
func analysisErrorKind(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "context deadline exceeded"),
		strings.Contains(message, "timed out"),
		strings.Contains(message, "timeout"):
		return "timeout"
	case strings.Contains(message, "context overflow"):
		return "context_overflow"
	case strings.Contains(message, "connection refused"),
		strings.Contains(message, "no such host"),
		strings.Contains(message, "dial tcp"),
		strings.Contains(message, "network is unreachable"):
		return "unavailable"
	case strings.Contains(message, "not found") && strings.Contains(message, "model"):
		return "model_unavailable"
	case strings.Contains(message, "not valid json"),
		strings.Contains(message, "model output"):
		return "invalid_output"
	case strings.Contains(message, "publication failed"),
		strings.Contains(message, "could not be stored"):
		return "publication"
	default:
		return "provider_error"
	}
}

var quoteSpaceRe = regexp.MustCompile(`\s+`)

// exactQuote resolves a model quote to an exact original substring, allowing only whitespace
// differences. It returns false rather than inventing a source anchor.
func exactQuote(source, quote string) (string, bool) {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return "", false
	}
	if strings.Contains(source, quote) {
		return quote, true
	}
	words := quoteSpaceRe.Split(quote, -1)
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = regexp.QuoteMeta(w)
	}
	re, err := regexp.Compile(strings.Join(parts, `\s+`))
	if err != nil {
		return "", false
	}
	if match := re.FindString(source); match != "" {
		return match, true
	}
	return "", false
}
