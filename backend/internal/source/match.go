package source

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/w3con/sifty/internal/model"
)

// trackingParamRe strips only known campaign/analytics parameters. Identity
// parameters such as id=1 and id=2 stay distinct.
var trackingParamRe = regexp.MustCompile(`(?i)^(utm_.*|ref|source|gh_src|fbclid|gclid)$`)

// matchStructuredJob returns the single reliable posting that belongs to ref.
// A multi-posting page with no unique URL or unique title/company match returns
// ok=false, so callers keep existing partial fields instead of guessing.
func matchStructuredJob(jobs []model.JobInput, ref Ref) (model.JobInput, bool) {
	if len(jobs) == 0 {
		return model.JobInput{}, false
	}
	var matches []model.JobInput
	for _, j := range jobs {
		if samePostingURL(j.OriginalURL, ref.URL) {
			matches = append(matches, j)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}

	title := strings.TrimSpace(ref.Partial.Title)
	company := strings.TrimSpace(ref.Partial.Company)
	if title != "" {
		matches = matches[:0]
		for _, j := range jobs {
			if strings.TrimSpace(j.Title) != title {
				continue
			}
			if company != "" && strings.TrimSpace(j.Company) != company {
				continue
			}
			matches = append(matches, j)
		}
		if len(matches) == 1 {
			return matches[0], true
		}
	}
	return model.JobInput{}, false
}

func samePostingURL(a, b string) bool {
	ca, cb := canonicalPostingURL(a), canonicalPostingURL(b)
	return ca != "" && ca == cb
}

func canonicalPostingURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	u.Fragment = ""
	q := u.Query()
	for k := range q {
		if trackingParamRe.MatchString(k) {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	out := host + strings.TrimRight(u.Path, "/")
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}
