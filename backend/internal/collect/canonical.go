package collect

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var trackingParamRe = regexp.MustCompile(`(?i)^(utm_|ref$|source$|gh_src$)`)

// Canonical normalises an HTTP(S) URL for identity comparison. It is the exported
// form of the internal canonical helper used by HTMLLinks.
func Canonical(raw string) (string, error) { return canonical(raw) }

// canonical mirrors the TypeScript canonical(): it normalises an HTTP(S) URL for identity
// comparison by dropping the fragment, a leading www., tracking parameters, and a trailing slash,
// and sorting the query.
func canonical(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("not an HTTP URL: %q", raw)
	}
	u.Fragment = ""
	u.Host = strings.TrimPrefix(u.Host, "www.")
	q := u.Query()
	for k := range q {
		if trackingParamRe.MatchString(k) {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode() // Encode sorts by key.
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}

var (
	nodejsRe   = regexp.MustCompile(`node\.?js`)
	web3jsRe   = regexp.MustCompile(`web3\.?js`)
	layer2Re   = regexp.MustCompile(`\blayer\s*2\b`)
	ctoRe      = regexp.MustCompile(`\bchief technology officer\b`)
	techLeadRe = regexp.MustCompile(`\btechnical lead\b`)
	aiRe       = regexp.MustCompile(`\bartificial intelligence\b`)
	nonAlnumRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)
)

// normalize mirrors the TypeScript normalize(): lowercase, canonicalise a few technology terms,
// and reduce to space-separated alphanumeric tokens.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = nodejsRe.ReplaceAllString(s, "nodejs")
	s = web3jsRe.ReplaceAllString(s, "web3js")
	s = layer2Re.ReplaceAllString(s, "l2")
	s = ctoRe.ReplaceAllString(s, "cto")
	s = techLeadRe.ReplaceAllString(s, "tech lead")
	s = aiRe.ReplaceAllString(s, "ai")
	s = nonAlnumRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
