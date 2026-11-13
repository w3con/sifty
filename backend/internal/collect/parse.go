// Package collect ports the current application's deterministic collection parsers to Go:
// structured JSON-LD JobPostings, RSS/Atom feeds, and generic HTML link extraction. Network
// fetching and dynamic-site scraping run elsewhere (the jobspy worker); these functions turn an
// already-fetched document into postings.
package collect

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/w3con/sifty/internal/model"
)

func ptr(s string) *string { return &s }

func parseHTML(html string) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(html))
}

// ── JSON-LD structured postings ─────────────────────────────────────────────────────────────

// StructuredJobs extracts every schema.org JobPosting from a page's ld+json scripts.
func StructuredJobs(html, pageURL, source string) []model.JobInput {
	jobs := []model.JobInput{}
	doc, err := parseHTML(html)
	if err != nil {
		return jobs
	}
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
		var data any
		if json.Unmarshal([]byte(s.Text()), &data) == nil {
			walkLD(data, pageURL, source, &jobs)
		}
	})
	return jobs
}

func flatten(x any) []any {
	switch v := x.(type) {
	case nil:
		return nil
	case []any:
		return v
	default:
		return []any{v}
	}
}

func typeIncludes(x any, want string) bool {
	for _, t := range flatten(x) {
		if s, ok := t.(string); ok && s == want {
			return true
		}
	}
	return false
}

func getString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func walkLD(o any, pageURL, source string, jobs *[]model.JobInput) {
	switch v := o.(type) {
	case []any:
		for _, e := range v {
			walkLD(e, pageURL, source, jobs)
		}
	case map[string]any:
		if typeIncludes(v["@type"], "JobPosting") {
			if job, ok := buildStructured(v, pageURL, source); ok {
				*jobs = append(*jobs, job)
			}
		}
		for k, val := range v {
			if k != "description" {
				walkLD(val, pageURL, source, jobs)
			}
		}
	}
}

func buildStructured(o map[string]any, pageURL, source string) (model.JobInput, bool) {
	original := abs(getString(o, "url"), pageURL)
	if original == "" {
		original = pageURL
	}
	var company string
	switch h := o["hiringOrganization"].(type) {
	case string:
		company = h
	case map[string]any:
		company = getString(h, "name")
	}
	var locs []string
	if t := getString(o, "jobLocationType"); t != "" {
		locs = append(locs, t)
	}
	for _, r := range flatten(o["applicantLocationRequirements"]) {
		if m, ok := r.(map[string]any); ok {
			if n := getString(m, "name"); n != "" {
				locs = append(locs, n)
			}
		}
	}
	for _, jl := range flatten(o["jobLocation"]) {
		m, ok := jl.(map[string]any)
		if !ok {
			continue
		}
		addr, ok := m["address"].(map[string]any)
		if !ok {
			continue
		}
		if s := getString(addr, "addressLocality"); s != "" {
			locs = append(locs, s)
		}
		if s := getString(addr, "addressRegion"); s != "" {
			locs = append(locs, s)
		}
		switch c := addr["addressCountry"].(type) {
		case string:
			if c != "" {
				locs = append(locs, c)
			}
		case map[string]any:
			if n := getString(c, "name"); n != "" {
				locs = append(locs, n)
			}
		}
	}
	title := getString(o, "title")
	if title == "" || company == "" {
		return model.JobInput{}, false
	}
	descRaw := getString(o, "description")
	status := "missing"
	if descRaw != "" {
		status = "available structured description"
	}
	published := model.NormalizePublishedAt(getString(o, "datePosted"))
	return model.JobInput{
		Title: text(title), Company: text(company), OriginalURL: original, ApplyURL: nil,
		Description: text(descRaw), DescriptionStatus: status,
		Location: strings.Join(dedupe(locs), " · "), PublishedAt: published,
		PublishedBasis: model.PublicationBasis(published, "datePosted"),
		ExpiresAt:      getString(o, "validThrough"), Source: source,
	}, true
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ── RSS/Atom feeds ──────────────────────────────────────────────────────────────────────────

// ParseFeed extracts postings from an RSS or Atom feed.
func ParseFeed(xmlText, pageURL, source string) []model.JobInput {
	result := []model.JobInput{}
	for _, it := range parseFeedItems(xmlText) {
		title := text(it.fields["title"])
		linkRaw := it.linkHref
		if linkRaw == "" {
			linkRaw = it.fields["link"]
		}
		link := abs(linkRaw, pageURL)
		if title == "" || link == "" {
			continue
		}
		var company, role string
		if strings.Contains(title, ":") {
			parts := strings.Split(title, ":")
			if len(parts) > 1 {
				company = strings.TrimSpace(parts[0])
				role = strings.TrimSpace(strings.Join(parts[1:], ":"))
			} else {
				company, role = "Unknown", title
			}
		} else if parts := strings.Split(title, " at "); len(parts) > 1 {
			company = strings.TrimSpace(strings.Join(parts[1:], " at "))
			role = strings.TrimSpace(parts[0])
		} else {
			company, role = "Unknown", title
		}
		desc := ""
		for _, k := range []string{"description", "content", "encoded", "summary"} {
			if len(it.fields[k]) > len(desc) {
				desc = it.fields[k]
			}
		}
		published, basis := it.pubDate, "pubDate"
		if published == "" {
			published, basis = it.published, "published"
		}
		published = model.NormalizePublishedAt(published)
		result = append(result, model.JobInput{
			Title: role, Company: company, OriginalURL: link, ApplyURL: nil,
			Description: text(desc), DescriptionStatus: "feed description; may be excerpt",
			PublishedAt: published, PublishedBasis: model.PublicationBasis(published, basis), Source: source,
		})
	}
	return result
}

// ── Generic HTML links ──────────────────────────────────────────────────────────────────────

var (
	labelSkipRe  = regexp.MustCompile(`(?i)^(apply|view|learn|see all|jobs|careers|next|previous)`)
	relPathRe    = regexp.MustCompile(`/jobs?/(?:[^/]+/)?[^/?]+`)
	relJobsRe    = regexp.MustCompile(`-jobs/[^/]+`)
	atsHostRe    = regexp.MustCompile(`jobs\.(ashbyhq|lever)\.(com|co)`)
	greenhouseRe = regexp.MustCompile(`greenhouse\.io`)
	ccjHostRe    = regexp.MustCompile(`cryptocurrencyjobs\.co$`)
	ccjPathRe    = regexp.MustCompile(`^/(engineering|operations|product|design|sales|marketing|finance|customer-support)/[^/]+/?$`)
	searchPathRe = regexp.MustCompile(`/jobs/(?:search|categories|new|remote|rss|feed)$`)
	ariaDashRe   = regexp.MustCompile(`^.*? – `)
	wsRe         = regexp.MustCompile(`\s+`)
)

// HTMLLinks extracts postings from a board's listing page: the Getro hydration payload when
// present, otherwise the page's job anchors.
func HTMLLinks(html, base, source string) []model.JobInput {
	doc, err := parseHTML(html)
	if err != nil {
		return []model.JobInput{}
	}
	if jobs, ok := getroJobs(doc, base, source); ok {
		return jobs
	}

	index := map[string]int{}
	order := []model.JobInput{}
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		u := abs(href, base)
		if u == "" {
			return
		}
		parsed, err := parseHref(u)
		if err != nil {
			return
		}
		label := strings.TrimSpace(wsRe.ReplaceAllString(a.Text(), " "))
		if label == "" || len(label) > 400 || labelSkipRe.MatchString(label) {
			return
		}
		relevant := relPathRe.MatchString(parsed.path) ||
			relJobsRe.MatchString(parsed.path) ||
			atsHostRe.MatchString(parsed.host) ||
			greenhouseRe.MatchString(parsed.host) ||
			(ccjHostRe.MatchString(parsed.host) && ccjPathRe.MatchString(parsed.path))
		if !relevant || searchPathRe.MatchString(parsed.path) {
			return
		}
		container := a.Closest(`[itemtype*="JobPosting"], article,li,.job-list-item,.job-listing,[data-testid="job-card"],[data-testid="job-list-item"],.job`).First()
		title := ""
		if aria, ok := a.Attr("aria-label"); ok {
			title = ariaDashRe.ReplaceAllString(aria, "")
		}
		if title == "" {
			if t := strings.TrimSpace(a.Find(`h2,h3,[itemprop="title"],.job-title`).First().Text()); t != "" {
				title = t
			}
		}
		if title == "" {
			title = label
		}
		company := strings.TrimSpace(container.Find(`[itemprop="hiringOrganization"],.company,.company-name,[data-testid="company-name"]`).First().Text())
		if company == "" {
			company = "Unknown"
		}
		job := model.JobInput{
			Title: title, Company: company, OriginalURL: u, ApplyURL: nil,
			Description: "", DescriptionStatus: "pending detail fetch", Source: source,
		}
		key, err := canonical(u)
		if err != nil {
			return
		}
		if pos, ok := index[key]; ok {
			order[pos] = job
		} else {
			index[key] = len(order)
			order = append(order, job)
		}
	})
	return order
}

func getroJobs(doc *goquery.Document, base, source string) ([]model.JobInput, bool) {
	raw := doc.Find("#__NEXT_DATA__").Text()
	if raw == "" {
		return nil, false
	}
	var data map[string]any
	if json.Unmarshal([]byte(raw), &data) != nil {
		return nil, false
	}
	found := digArray(data, "props", "pageProps", "initialState", "jobs", "found")
	if len(found) == 0 {
		return nil, false
	}
	jobs := []model.JobInput{}
	for _, f := range found {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		title := getString(m, "title")
		org, _ := m["organization"].(map[string]any)
		slug := getString(m, "slug")
		if title == "" || org == nil || getString(org, "name") == "" || slug == "" {
			continue
		}
		original := abs("/companies/"+getString(org, "slug")+"/jobs/"+slug, base)
		var applyURL *string
		if au := abs(getString(m, "url"), base); au != "" {
			applyURL = ptr(au)
		}
		jobs = append(jobs, model.JobInput{
			Title: title, Company: getString(org, "name"), OriginalURL: original, ApplyURL: applyURL,
			Description:       strings.Join(stringList(m["skills"]), ", "),
			DescriptionStatus: "skills metadata only; full description pending",
			Location:          strings.Join(stringList(m["locations"]), " · "),
			PublishedAt:       model.NormalizePublishedAt(epochToISO(m["createdAt"])),
			PublishedBasis:    model.PublicationBasis(model.NormalizePublishedAt(epochToISO(m["createdAt"])), "created_at"), Source: source,
		})
	}
	return jobs, true
}

func digArray(m map[string]any, path ...string) []any {
	cur := any(m)
	for _, k := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[k]
	}
	if arr, ok := cur.([]any); ok {
		return arr
	}
	return nil
}

func stringList(x any) []string {
	arr, ok := x.([]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
