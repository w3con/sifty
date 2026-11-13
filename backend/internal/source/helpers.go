package source

import "github.com/w3con/sifty/internal/collect"

// plainOrFallback derives plain text from the structured HTML produced by
// collect.StructuredHTML. When the structured form is empty (e.g. the page
// returned no recognisable content), it falls back to collect.CleanHTML so
// the caller always gets something usable.
func plainOrFallback(structuredHTML, rawInput string) string {
	if p := collect.PlainFromStructured(structuredHTML); p != "" {
		return p
	}
	return collect.CleanHTML(rawInput)
}
