package vacancyai

import (
	"regexp"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// The model sometimes reads a condition out of a sentence that is about something else: a
// "50/50 hybrid role" splits duties between two kinds of work and says nothing about where the
// work is done. The checks below accept a work mode or hours condition only when its own quote
// speaks about a place or about working time; they are a floor under the prompt, not a parser.

var (
	// A place word: where the work is done.
	workModePlaceRe = regexp.MustCompile(`(?i)\b(remote|remotely|office|offices|on-?site|in[- ]person|at home|from home|wfh|work from|working from|distributed|hybrid (work|working|model|schedule|setup|set-up|environment|arrangement|policy|office))\b|\bon site\b|\bin the office\b|\bdays? (in|per|a|each)\b`)
	// Explicit working time: a named schedule or an amount of hours.
	// The type of agreement: a contract kind, or an employment kind named outright. A bare word
	// such as "employment" or "employees" is not enough: equal-opportunity text, benefit lists and
	// work-authorisation notices all use it without saying how this role is engaged.
	engagementTypeRe = regexp.MustCompile(`(?i)\bcontract(or|ors|ing|ual)?\b|\bfreelanc\w*|\bb2b\b|\b1099\b|\bself-employed\b|\bfixed[- ]term\b|\bconsultan(t|cy)\b|\binterim\b|\bfractional\b|\bpermanent\b|\bfull[- ]?time\b|\bpart[- ]?time\b|\bw-?2\b|\bjob type\b|\bemployment (type|status|agreement)\b`)
	hoursTimeRe      = regexp.MustCompile(`(?i)\bfull[- ]?time\b|\bpart[- ]?time\b|\b\d+(\.\d+)?\s*(-|to|–)?\s*(\d+(\.\d+)?)?\s*(hours?|hrs?|h)\b|\bhours?\s+(per|a|each|/)\s*(week|day|month)\b|\b(per|a|each)\s+week\b|/\s*week\b`)
)

// groundWorkMode reduces a model's work-mode answer to remote, hybrid or office, and refuses it
// unless the quote itself is about the place of work.
func groundWorkMode(value, quote string) (string, bool) {
	if !workModePlaceRe.MatchString(quote) {
		return "", false
	}
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "hybrid"):
		return matchcontract.WorkHybrid, true
	case strings.Contains(lower, "remote"):
		return matchcontract.WorkRemote, true
	case strings.Contains(lower, "office") || strings.Contains(lower, "on-site") ||
		strings.Contains(lower, "onsite") || strings.Contains(lower, "on site") || strings.Contains(lower, "in person") ||
		strings.Contains(lower, "in-person"):
		return matchcontract.WorkOffice, true
	}
	return "", false
}

// groundHours accepts an hours condition only when its quote states full-time, part-time or an
// amount of hours. Anything inferred from another word ("permanent", "a dedicated role") is
// refused.
func groundHours(quote string) bool { return hoursTimeRe.MatchString(quote) }

// GroundEngagement (exported so that the sensitivity diagnostic in matchsvc applies the same rule) accepts an engagement condition only when its quote names a kind of agreement
// (contract, freelance, permanent, full-time and the like). Anything inferred from a sentence that
// merely contains the word "employment" is refused.
func GroundEngagement(quote string) bool { return engagementTypeRe.MatchString(quote) }
