package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/prompts"
	"github.com/w3con/sifty/internal/store"
)

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

// extractFacts calls the configured extraction stage through the existing assess provider
// adapters and returns canonical, quote-anchored proposals. It never writes the profile.
func (s *Service) extractFacts(ctx context.Context, rec *store.ProfileDocumentRecord, runID, providerName string, provider providerConfig) ([]*store.ProfileFactRecord, map[string]any, error) {
	cfg := s.cfg.Get()
	schemaJSON, err := json.Marshal(factSchema)
	if err != nil {
		return nil, nil, err
	}
	requestText := fmt.Sprintf("%s\nSCHEMA\n%s\nDOCUMENT\n%s",
		prompts.ProfileFacts, schemaJSON, rec.ExtractedText)
	var output extractedFacts
	p := assess.Provider{
		Type: provider.Type, Model: provider.Model, Endpoint: provider.Endpoint,
		APIKeyEnv: provider.APIKeyEnv, TimeoutMS: provider.TimeoutMS, ContextChars: provider.ContextChars,
		Think: provider.Think,
	}
	if err := assess.InferInto(ctx, p, requestText, factSchema, &output); err != nil {
		return nil, nil, fmt.Errorf("extraction provider %q failed: %w", providerName, err)
	}
	maxFacts := cfg.Runtime.EffectiveProfileMaxFacts()
	var facts []*store.ProfileFactRecord
	var skipped []map[string]string
	for _, item := range output.Facts {
		fieldKey := strings.TrimSpace(item.FieldKey)
		canonical, err := ValidateFactValue(fieldKey, item.Value)
		if err != nil {
			skipped = append(skipped, map[string]string{"fieldKey": fieldKey, "reason": err.Error()})
			continue
		}
		quote, ok := exactQuote(rec.ExtractedText, item.Quote)
		if !ok {
			skipped = append(skipped, map[string]string{"fieldKey": fieldKey, "reason": "quote is not an exact excerpt of the document"})
			continue
		}
		docID := rec.ID
		sourceRef := rec.ID
		if rec.ContentHash != "" {
			sourceRef += "@" + rec.ContentHash
		}
		facts = append(facts, &store.ProfileFactRecord{
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
	if len(facts) > maxFacts {
		return nil, nil, fmt.Errorf("analysis produced %d facts; configured limit is %d; no facts were stored", len(facts), maxFacts)
	}
	metadata := map[string]any{
		"runId": runID, "finishedAt": nowISO(), "phase": "analysis",
		"provider": providerName, "model": provider.Model, "contentHash": rec.ContentHash,
		"factCount": len(facts), "documentChars": len(rec.ExtractedText),
	}
	if len(skipped) > 0 {
		metadata["skipped"] = skipped
	}
	return facts, metadata, nil
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
