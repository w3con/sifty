// Package candidate assembles the candidate evidence context from local files, remote profiles
// and owner-confirmed feedback records. The assembled snapshot is stored in the database by
// calling store.SaveContext and read back by store.Context, so the HTTP layer can serve it
// without re-reading files on every request.
package candidate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	gocfg "github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/extract"
	"github.com/w3con/sifty/internal/records"
)

// EvidenceSource is one piece of candidate evidence: a local file, a remote profile snapshot, or
// an owner-confirmed feedback record. The JSON field names match the TypeScript type exactly.
type EvidenceSource struct {
	ID          string `json:"id"`
	Reference   string `json:"reference"`
	RetrievedAt string `json:"retrievedAt"`
	Hash        string `json:"hash"`
	Status      string `json:"status"`
	Text        string `json:"text"`
	Error       string `json:"error,omitempty"`
	Category    string `json:"category,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

// CandidateContext is the assembled evidence snapshot. The JSON field names match the TypeScript
// CandidateContext type.
type CandidateContext struct {
	Version         string           `json:"version"`
	CreatedAt       string           `json:"createdAt"`
	ProfileID       string           `json:"profileId,omitempty"`
	ProfileRevision int              `json:"profileRevision,omitempty"`
	Preferences     any              `json:"preferences"`
	ReviewNotes     []string         `json:"reviewNotes"`
	Sources         []EvidenceSource `json:"sources"`
}

// ContextStore is the subset of *store.Store used here, allowing tests to inject a stub.
type ContextStore interface {
	// Context returns the stored context, or nil if none.
	Context() (map[string]any, error)
	// SaveContext persists a new context snapshot.
	SaveContext(map[string]any) error
}

// hashStr returns the SHA-256 hex digest of s.
func hashStr(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:])
}

// nowISO returns the current UTC time in the ISO 8601 millisecond format.
func nowISO() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

// ContextVersion returns a short fingerprint over the context content that changes whenever
// preferences, review notes, or source content changes. It matches the TypeScript contextVersion.
func ContextVersion(preferences any, reviewNotes []string, sources []EvidenceSource) string {
	return ContextVersionForProfile("", 0, preferences, reviewNotes, sources)
}

// ContextVersionForProfile is ContextVersion with the authoritative profile identity included.
// A confirmed-profile source and its revision therefore change the derived context version even
// when the remaining evidence text happens to be unchanged. The profile fields are omitted for
// an empty profile id so contexts assembled before profiles existed keep their old fingerprint.
func ContextVersionForProfile(profileID string, profileRevision int, preferences any, reviewNotes []string, sources []EvidenceSource) string {
	type sourceEntry struct {
		ID        string `json:"id"`
		Reference string `json:"reference"`
		Hash      string `json:"hash"`
		Status    string `json:"status"`
		Error     string `json:"error,omitempty"`
		Category  string `json:"category,omitempty"`
		Scope     string `json:"scope,omitempty"`
		TextHash  string `json:"textHash"`
	}
	entries := make([]sourceEntry, len(sources))
	for i, s := range sources {
		entries[i] = sourceEntry{
			ID: s.ID, Reference: s.Reference, Hash: s.Hash,
			Status: s.Status, Error: s.Error, Category: s.Category, Scope: s.Scope,
			TextHash: hashStr(s.Text),
		}
	}
	payload := map[string]any{
		"preferences": preferences, "reviewNotes": reviewNotes, "sources": entries,
	}
	if profileID != "" {
		payload["profileId"] = profileID
		payload["profileRevision"] = profileRevision
	}
	data, _ := json.Marshal(payload)
	return hashStr(string(data))
}

// FeedbackSources builds EvidenceSource entries from the owner-confirmed feedback files.
// This matches the TypeScript feedbackSources() function.
func FeedbackSources(d dirs.Dirs) ([]EvidenceSource, error) {
	recs, err := records.ReadFeedback(d)
	if err != nil {
		return nil, err
	}
	// Build a lookup of which records supersede each earlier record.
	type revision struct{ id, supersedes string }
	var all []revision
	for _, r := range recs {
		s := ""
		if r.Supersedes != nil {
			s = *r.Supersedes
		}
		all = append(all, revision{r.ID, s})
	}

	var out []EvidenceSource
	for _, r := range recs {
		var revisions []string
		for _, a := range all {
			if a.supersedes == r.ID {
				revisions = append(revisions, a.id)
			}
		}
		scopeStr := "general candidate statement; conflicts unresolved"
		if r.Scope == "job" {
			scopeStr = fmt.Sprintf("job:%d", r.JobID)
		}
		supersedes := "none"
		if r.Supersedes != nil {
			supersedes = *r.Supersedes
		}
		revisionNote := ""
		if len(revisions) > 0 {
			revisionNote = fmt.Sprintf("Historical statement explicitly revised by %s; do not treat the old wording as a current fact.", strings.Join(revisions, ", "))
		}
		factsJSON, _ := json.Marshal(r.Facts)
		conflictsJSON, _ := json.Marshal(r.Conflicts)
		text := fmt.Sprintf(
			"Owner-confirmed candidate feedback. Record ID: %s. Date: %s. Scope: %s. Source job: %s. %s\nOriginal user statement (verbatim):\n%s\n\nGrounded explicit facts: %s\nConflicts/review: %s. Supersedes: %s. Earlier records are retained. Do not silently resolve conflicting facts or generalise role-scoped statements.",
			r.ID, r.Created, r.Scope, r.JobURL, revisionNote, r.Text,
			string(factsJSON), string(conflictsJSON), supersedes,
		)
		out = append(out, EvidenceSource{
			ID:          r.Path,
			Reference:   r.Path,
			RetrievedAt: r.Created,
			Hash:        hashStr(text),
			Status:      "available",
			Category:    "owner-confirmed-feedback",
			Scope:       scopeStr,
			Text:        text,
		})
	}
	return out, nil
}

// cleanHTML strips script/style/navigation elements and extracts human-readable text from HTML,
// matching the TypeScript cleanHtml function.
func cleanHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}
	doc.Find("script,style,nav,footer,header,noscript,svg").Remove()
	// Prepend newlines to block-level elements so that text runs are separated.
	doc.Find("p,br,div,li,h1,h2,h3,section").Each(func(_ int, s *goquery.Selection) {
		s.PrependHtml("\n")
	})
	text := doc.Find("body").Text()
	// Collapse horizontal whitespace on each line, then collapse multiple blank lines.
	spaceRe := regexp.MustCompile(`[ \t]+`)
	blankRe := regexp.MustCompile(`\n\s*\n+`)
	text = spaceRe.ReplaceAllString(text, " ")
	text = blankRe.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text), nil
}

// fileText reads extractable text from a file. PDFs are extracted with pdftotext; text files are
// read directly. The small shared reader lives in internal/extract so the profile document path
// uses the same PDF/plain-text behaviour.
func fileText(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	text, err := extract.PlainText(ctx, path)
	if err == nil {
		return text, nil
	}
	// Preserve the legacy candidate-context behaviour for other configured formats: they are
	// read as raw text. PDF failures stay errors, and profile documents use the real DOCX reader.
	if extract.Kind(path) == "pdf" {
		return "", err
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return "", err
	}
	return string(data), nil
}

// fetchURL performs an HTTP GET and returns the response body as a string. It uses a 30-second
// timeout and a browser-like User-Agent to avoid trivial bot blocks.
func fetchURL(rawURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return buf.String(), nil
}

// contextToMap serialises a CandidateContext to the map[string]any shape the store expects.
func contextToMap(c *CandidateContext) (map[string]any, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// MapToContext deserialises a stored map[string]any back to a CandidateContext.
func MapToContext(m map[string]any) (*CandidateContext, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var c CandidateContext
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// RefreshContext assembles a fresh CandidateContext from local files, remote profiles and
// feedback records, saves it to the store, and returns it. It matches the TypeScript
// refreshContext function.
//
// refreshRemote forces re-fetching of remote profiles even if the cache is current.
// localOnly skips remote fetches and marks profiles without an export as stale/missing.
func RefreshContext(store ContextStore, d dirs.Dirs, cfg *gocfg.Config, preferences any, refreshRemote, localOnly bool) (*CandidateContext, error) {
	return refreshContext(store, d, cfg, preferences, "", 0, refreshRemote, localOnly)
}

// RefreshContextForProfile is RefreshContext for a derived context whose authoritative source is
// the stored confirmed profile. The profile identity is recorded on the snapshot (not inferred
// from source text), and any extra sources (the confirmed-profile evidence) are appended after
// the configured files and owner feedback.
func RefreshContextForProfile(store ContextStore, d dirs.Dirs, cfg *gocfg.Config, preferences any, profileID string, profileRevision int, refreshRemote, localOnly bool, extra ...EvidenceSource) (*CandidateContext, error) {
	return refreshContext(store, d, cfg, preferences, profileID, profileRevision, refreshRemote, localOnly, extra...)
}

func refreshContext(store ContextStore, d dirs.Dirs, cfg *gocfg.Config, preferences any, profileID string, profileRevision int, refreshRemote, localOnly bool, extra ...EvidenceSource) (*CandidateContext, error) {
	var cc gocfg.Candidate
	if cfg != nil {
		cc = cfg.Candidate
	}

	// Load previous context from the store (for profile text reuse within cache window).
	var previous *CandidateContext
	if prev, err := store.Context(); err == nil && prev != nil {
		if p, err := MapToContext(prev); err == nil {
			previous = p
		}
	}

	var sources []EvidenceSource

	// --- Local files ---
	for _, relPath := range cc.Files {
		path := relPath
		if !strings.HasPrefix(path, "/") {
			// Resolve relative to the data directory.
			var err error
			path, err = d.Under(relPath)
			if err != nil {
				sources = append(sources, EvidenceSource{
					ID: relPath, Reference: relPath, RetrievedAt: nowISO(),
					Hash: "", Status: "missing", Text: "",
					Error: fmt.Sprintf("Path %q escapes data directory", relPath),
				})
				continue
			}
		}
		text, err := fileText(path)
		if err != nil || strings.TrimSpace(text) == "" {
			errMsg := fmt.Sprintf("No extractable text; provide text/OCR export")
			if err != nil {
				errMsg = err.Error()
			}
			sources = append(sources, EvidenceSource{
				ID: relPath, Reference: relPath, RetrievedAt: nowISO(),
				Hash: "", Status: "missing", Text: "", Error: errMsg,
			})
			continue
		}
		raw, _ := os.ReadFile(path)
		sources = append(sources, EvidenceSource{
			ID: relPath, Reference: relPath, RetrievedAt: nowISO(),
			Hash: hashStr(string(raw)), Status: "available", Text: text,
		})
	}

	// --- Remote profiles ---
	for _, p := range cc.Profiles {
		var old *EvidenceSource
		if previous != nil {
			for i, s := range previous.Sources {
				if s.ID == p.ID {
					old = &previous.Sources[i]
					break
				}
			}
		}

		if localOnly && p.Export == "" {
			if old != nil {
				src := *old
				src.Reference = p.URL
				if src.Reference != p.URL {
					src.Status = "stale"
					src.Error = fmt.Sprintf("Configured profile URL changed; retained text is from %s. Refresh this profile explicitly.", old.Reference)
				}
				sources = append(sources, src)
			} else {
				sources = append(sources, EvidenceSource{
					ID: p.ID, Reference: p.URL, RetrievedAt: nowISO(),
					Hash: "", Status: "missing", Text: "",
					Error: "Remote profile not collected; refresh explicitly",
				})
			}
			continue
		}

		// Check cache validity.
		if !refreshRemote && p.Export == "" && old != nil && old.Text != "" {
			if age := time.Since(parseISO(old.RetrievedAt)); age < time.Duration(cc.CacheHours*float64(time.Hour)) {
				sources = append(sources, *old)
				continue
			}
		}

		// Resolve export file path if configured.
		exportPath := ""
		if p.Export != "" {
			ep, err := d.Under(p.Export)
			if err == nil {
				if _, err := os.Stat(ep); err == nil {
					exportPath = ep
				}
			}
			if exportPath == "" {
				sources = append(sources, EvidenceSource{
					ID: p.ID, Reference: p.URL,
					RetrievedAt: func() string {
						if old != nil {
							return old.RetrievedAt
						}
						return nowISO()
					}(),
					Hash: func() string {
						if old != nil {
							return old.Hash
						}
						return ""
					}(),
					Text: func() string {
						if old != nil {
							return old.Text
						}
						return ""
					}(),
					Status: func() string {
						if old != nil && old.Text != "" {
							return "stale"
						}
						return "missing"
					}(),
					Error: fmt.Sprintf("Configured export missing: %s", p.Export),
				})
				continue
			}
		}

		var text string
		var fetchErr error
		if exportPath != "" {
			text, fetchErr = fileText(exportPath)
		} else {
			body, err := fetchURL(p.URL)
			if err != nil {
				fetchErr = err
			} else {
				text, fetchErr = cleanHTML(body)
			}
		}

		if fetchErr != nil || len(text) < 100 || isLinkedInGate(p.ID, text) {
			errMsg := "Public profile incomplete; configure an owner-provided export"
			if fetchErr != nil {
				errMsg = fetchErr.Error()
			}
			retrievedAt := nowISO()
			h := ""
			t := ""
			status := "missing"
			if old != nil {
				retrievedAt = old.RetrievedAt
				h = old.Hash
				t = old.Text
				if old.Text != "" {
					status = "stale"
				}
			}
			sources = append(sources, EvidenceSource{
				ID: p.ID, Reference: p.URL, RetrievedAt: retrievedAt,
				Hash: h, Status: status, Text: t, Error: errMsg,
			})
			continue
		}

		ref := p.URL
		if exportPath != "" {
			ref = p.Export
		}
		sources = append(sources, EvidenceSource{
			ID: p.ID, Reference: ref, RetrievedAt: nowISO(),
			Hash: hashStr(text), Status: "available", Text: text,
		})
	}

	// --- Feedback sources ---
	feedback, err := FeedbackSources(d)
	if err != nil {
		return nil, fmt.Errorf("load feedback: %w", err)
	}

	// Apply source_roles to non-feedback sources.
	for i, s := range sources {
		if role, ok := cc.SourceRoles[s.ID]; ok {
			sources[i].Category = role.Category
			sources[i].Scope = role.Scope
		} else {
			if sources[i].Category == "" {
				sources[i].Category = "public-profile"
			}
			if sources[i].Scope == "" {
				sources[i].Scope = "Public source snapshot; unverified statements remain attributed"
			}
		}
	}

	allSources := append(sources, feedback...)
	allSources = append(allSources, extra...)

	reviewNotes := cc.ReviewNotes
	if reviewNotes == nil {
		reviewNotes = []string{}
	}

	version := ContextVersionForProfile(profileID, profileRevision, preferences, reviewNotes, allSources)
	c := &CandidateContext{
		Version:         version,
		CreatedAt:       nowISO(),
		ProfileID:       profileID,
		ProfileRevision: profileRevision,
		Preferences:     preferences,
		ReviewNotes:     reviewNotes,
		Sources:         allSources,
	}

	m, err := contextToMap(c)
	if err != nil {
		return nil, err
	}
	if err := store.SaveContext(m); err != nil {
		return nil, fmt.Errorf("save context: %w", err)
	}
	return c, nil
}

// isLinkedInGate returns true if this looks like a LinkedIn gate page rather than the profile.
func isLinkedInGate(id, text string) bool {
	if id != "linkedin" {
		return false
	}
	lower := strings.ToLower(text)
	return (strings.Contains(lower, "sign in") || strings.Contains(lower, "join linkedin")) &&
		!strings.Contains(lower, "experience")
}

// parseISO parses an ISO 8601 millisecond timestamp, returning zero time on failure.
func parseISO(s string) time.Time {
	for _, layout := range []string{"2006-01-02T15:04:05.000Z", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ContextForJobResult is the per-job evidence selection returned by ContextForJob.
type ContextForJobResult struct {
	Overview map[string]any `json:"overview"`
	Sources  []struct {
		ID           string `json:"id"`
		Text         string `json:"text"`
		TotalChars   int    `json:"totalChars"`
		OmittedChars int    `json:"omittedChars"`
	} `json:"sources"`
}

// ContextForJob selects the most relevant paragraphs from the candidate context for a given
// job description, staying within the provider's context budget. It matches the TypeScript
// contextForJob function.
func ContextForJob(c *CandidateContext, jobText string, budget int) (*ContextForJobResult, error) {
	// Build a set of term tokens from the job text.
	termRe := regexp.MustCompile(`[a-z][a-z0-9+.#\-]{2,}`)
	terms := make(map[string]bool)
	for _, m := range termRe.FindAllString(strings.ToLower(jobText), -1) {
		terms[m] = true
	}

	// Build the overview header.
	type sourceStatus struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Error    string `json:"error,omitempty"`
		Category string `json:"category,omitempty"`
		Scope    string `json:"scope,omitempty"`
	}
	statuses := make([]sourceStatus, len(c.Sources))
	for i, s := range c.Sources {
		statuses[i] = sourceStatus{ID: s.ID, Status: s.Status, Error: s.Error, Category: s.Category, Scope: s.Scope}
	}
	headers, err := json.Marshal(map[string]any{
		"preferences": c.Preferences, "reviewNotes": c.ReviewNotes, "sourceStatus": statuses,
	})
	if err != nil {
		return nil, err
	}

	available := make([]EvidenceSource, 0, len(c.Sources))
	for _, s := range c.Sources {
		if s.Text != "" {
			available = append(available, s)
		}
	}

	if budget-len(headers) < len(available)*400 {
		return nil, fmt.Errorf("context budget too small for source overview; increase context_chars")
	}

	left := budget - len(headers)

	type selectedSource struct {
		ID           string `json:"id"`
		Text         string `json:"text"`
		TotalChars   int    `json:"totalChars"`
		OmittedChars int    `json:"omittedChars"`
	}
	var selected []selectedSource

	blankRe := regexp.MustCompile(`\n\s*\n`)

	for i, s := range available {
		allowance := int(math.Floor(float64(left) / float64(len(available)-i)))
		paragraphs := blankRe.Split(s.Text, -1)
		type scored struct {
			text  string
			index int
			score int
		}
		ranked := make([]scored, 0, len(paragraphs))
		for idx, p := range paragraphs {
			p = strings.TrimFunc(p, unicode.IsSpace)
			if p == "" {
				continue
			}
			score := 0
			lower := strings.ToLower(p)
			for term := range terms {
				if strings.Contains(lower, term) {
					score++
				}
			}
			ranked = append(ranked, scored{p, idx, score})
		}
		sort.Slice(ranked, func(a, b int) bool { return ranked[a].score > ranked[b].score })

		var picked []scored
		used := 0
		for _, p := range ranked {
			if used+len(p.text)+2 <= allowance {
				picked = append(picked, p)
				used += len(p.text) + 2
			}
		}

		if len(picked) == 0 && s.Text != "" && len(s.Text) > allowance {
			return nil, fmt.Errorf("unsegmentable source %s; budget too small, no silent truncation", s.ID)
		}

		// Restore original paragraph order.
		sort.Slice(picked, func(a, b int) bool { return picked[a].index < picked[b].index })
		texts := make([]string, len(picked))
		for j, p := range picked {
			texts[j] = p.text
		}
		text := strings.Join(texts, "\n\n")
		omitted := 0
		if len(s.Text) > len(text) {
			omitted = len(s.Text) - len(text)
		}
		selected = append(selected, selectedSource{
			ID: s.ID, Text: text, TotalChars: len(s.Text), OmittedChars: omitted,
		})
		left -= len(text)
	}

	result := &ContextForJobResult{}
	result.Overview = make(map[string]any)
	_ = json.Unmarshal(headers, &result.Overview)
	for _, s := range selected {
		result.Sources = append(result.Sources, struct {
			ID           string `json:"id"`
			Text         string `json:"text"`
			TotalChars   int    `json:"totalChars"`
			OmittedChars int    `json:"omittedChars"`
		}{s.ID, s.Text, s.TotalChars, s.OmittedChars})
	}
	return result, nil
}
