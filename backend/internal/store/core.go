package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// stageAliases maps every stored status value and stage name to its canonical stage.
var stageAliases = map[string]string{
	"new":         "triage",
	"saved":       "preselected",
	"rejected":    "no",
	"shortlist":   "preselected",
	"triage":      "triage",
	"preselected": "preselected",
	"no":          "no",
	"applied":     "applied",
}

// stageFor returns the canonical stage for a stored status or stage name.
func stageFor(status string) (string, error) {
	s, ok := stageAliases[status]
	if !ok {
		return "", fmt.Errorf("invalid workflow stage %q", status)
	}
	return s, nil
}

// statusFor returns the stored status column value for a canonical stage.
var statusForStage = map[string]string{
	"triage":      "new",
	"preselected": "saved",
	"no":          "rejected",
	"applied":     "applied",
}

func statusFor(stage string) string { return statusForStage[stage] }

// web3Re is the regular expression used by isWeb3, matching the TypeScript implementation exactly.
var web3Re = regexp.MustCompile(
	`(?i)\b(web\s?3|blockchains?|crypto(?:currency|currencies)?|defi|solidity|ethereum|bitcoin|smart\s+contracts?|nfts?|evm|solana|polkadot|on[ -]chain|decentralized\s+finance)\b`,
)

// isWeb3 returns true when the role's title, tags or description mention web3-related terms.
func isWeb3(title string, tags []string, description string) bool {
	return web3Re.MatchString(title + " " + strings.Join(tags, " ") + " " + description)
}

// archiveRetentionMs is 30 days, matching the TypeScript constant.
const archiveRetentionMs = 30 * 24 * 60 * 60 * 1000

// now returns an ISO-8601 millisecond-precision UTC timestamp matching TypeScript's toISOString().
func now() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

// hashStr computes the SHA-256 hex digest of s, matching TypeScript's hash().
func hashStr(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

var (
	trackingParamRe = regexp.MustCompile(`(?i)^(utm_|ref$|source$|gh_src$)`)
	nodejsRe        = regexp.MustCompile(`node\.?js`)
	web3jsRe        = regexp.MustCompile(`web3\.?js`)
	layer2Re        = regexp.MustCompile(`\blayer\s*2\b`)
	ctoRe           = regexp.MustCompile(`\bchief technology officer\b`)
	techLeadRe      = regexp.MustCompile(`\btechnical lead\b`)
	aiRe            = regexp.MustCompile(`\bartificial intelligence\b`)
	nonAlnumRe      = regexp.MustCompile(`[^\p{L}\p{N}]+`)
)

// canonicalURL normalises an HTTP(S) URL for identity comparison, matching TypeScript canonical().
func canonicalURL(raw string) (string, error) {
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
	u.RawQuery = q.Encode()
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}

// normalizeText mirrors TypeScript normalize(): lowercase, canonicalise a few technology terms,
// reduce to space-separated alphanumeric tokens.
func normalizeText(s string) string {
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

// listingKeys returns the SHA-256 hashes of the canonical listing URLs used for deleted_listings.
func listingKeys(originalURL string, applyURL *string) ([]string, error) {
	canon, err := canonicalURL(originalURL)
	if err != nil {
		return nil, fmt.Errorf("canonical original URL: %w", err)
	}
	keys := []string{hashStr("url:" + canon)}
	if applyURL != nil && *applyURL != "" {
		aCanon, err := canonicalURL(*applyURL)
		if err == nil {
			keys = append(keys, hashStr("url:"+aCanon))
		}
	}
	return keys, nil
}

// jobVersionHash computes the staleness fingerprint for an assessment: the SHA-256 of the
// JSON-encoded {title, company, description, descriptionStatus, location} object, matching
// the TypeScript implementation exactly.
func jobVersionHash(title, company, description, descriptionStatus string, location *string) string {
	loc := ""
	if location != nil {
		loc = *location
	}
	data, _ := json.Marshal(map[string]any{
		"title":             title,
		"company":           company,
		"description":       description,
		"descriptionStatus": descriptionStatus,
		"location":          loc,
	})
	return hashStr(string(data))
}

// JobVersionHash exposes the same vacancy fingerprint to the assessment worker so a stored
// result carries the version that Get/List compare against.
func JobVersionHash(title, company, description, descriptionStatus string, location *string) string {
	return jobVersionHash(title, company, description, descriptionStatus, location)
}

// intValue reads a decoded JSON number without inventing a revision from an absent field.
func intValue(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}
