package quickmatch

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	countryCodeRe = regexp.MustCompile(`^[A-Za-z]{2}$`)
)

// normalizedTokens lower-cases a title and splits it into unique alphanumeric tokens. Case and
// punctuation are separators; '+' '#' and '.' stay inside a token so C++, C# and Node.js are not
// silently reduced to C, C or Node. Seniority words are retained, never removed.
func normalizedTokens(s string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '#' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	seen := make(map[string]bool)
	var out []string
	for _, token := range strings.Fields(b.String()) {
		token = strings.Trim(token, ".")
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

// tokenF1 is the maximum-free F1 of two unique token sets. It never consults another vacancy.
func tokenF1(a, b string) float64 {
	as := normalizedTokens(a)
	bs := normalizedTokens(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}
	set := make(map[string]bool, len(as))
	for _, t := range as {
		set[t] = true
	}
	inter := 0
	for _, t := range bs {
		if set[t] {
			inter++
		}
	}
	if inter == 0 {
		return 0
	}
	return 2 * float64(inter) / float64(len(as)+len(bs))
}

func normalizeWorkMode(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// normalizeEngagement keeps the known legacy profile spellings inside the shared contract's
// two engagement values. It is not an open alias registry.
func normalizeEngagement(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "employment", "employee":
		return "employment"
	case "contract", "contractor":
		return "contract"
	}
	return ""
}

func normalizeCountry(s string) (value string, isCode bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	return s, countryCodeRe.MatchString(s)
}

func normalizeCity(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func formatCash(value float64, currency, period, tax, basis string) string {
	out := strconv.FormatFloat(value, 'f', -1, 64)
	if currency != "" {
		out += " " + currency
	}
	if period != "" {
		out += "/" + period
	}
	if tax != "" {
		out += " " + tax
	}
	if basis != "" {
		out += " (" + strings.ReplaceAll(basis, "_", " ") + ")"
	}
	return out
}

func joinNonEmpty(values []string) string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func sortedUniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func fmtFloat(v float64) string { return fmt.Sprintf("%g", v) }
