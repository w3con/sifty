// Package assess ports the current application's fit-assessment pipeline to Go: it selects
// candidate evidence within the model's context budget, builds the evidence bank the model
// references, calls a provider-configurable model, and post-processes the model's output into a
// grounded assessment. The model call is injectable (see Inference); the deterministic selection,
// grounding and mismatch disambiguation around it match the TypeScript application.
package assess

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// EvidenceSource is one candidate evidence source in the context.
type EvidenceSource struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Category string `json:"category,omitempty"`
	Scope    string `json:"scope,omitempty"`
	Text     string `json:"-"`
}

// CandidateContext is the candidate evidence the assessment draws on.
type CandidateContext struct {
	Preferences any
	ReviewNotes []string
	Sources     []EvidenceSource
}

// Job is the posting being assessed.
type Job struct {
	ID                int
	Title             string
	Company           string
	Description       string
	DescriptionStatus string
	Location          string
}

// Provider is the inference provider configuration relevant to the assessment call.
type Provider struct {
	Type         string
	Model        string
	Endpoint     string
	APIKeyEnv    string
	TimeoutMS    int
	ContextChars int
	// Think controls the ollama thinking/reasoning mode for this call. Nil keeps the provider's
	// default. Structured extraction sets false: a reasoning trace is useless for a
	// schema-constrained JSON answer and, measured on qwen3.5:9b, alone consumed the whole call
	// budget before any JSON was emitted.
	Think *bool
	// ThinkLevel is the reasoning effort ("low", "medium" or "high") for models that cannot switch
	// reasoning off, such as gpt-oss. When set it replaces Think in the ollama request.
	ThinkLevel string
	// Temperature is the sampling temperature for the ollama and compatible adapters. Nil keeps
	// the provider's default. The command-line adapters ignore it.
	Temperature *float64
}

// RawEvidence, RawMismatch and RawModel are the model's structured output, referencing evidence
// by the bank identifiers (E1, E2, …) the request supplies.
type RawEvidence struct {
	Claim      string `json:"claim"`
	EvidenceID string `json:"evidenceId"`
}

type RawMismatch struct {
	Reason         string `json:"reason"`
	EvidenceID     string `json:"evidenceId"`
	JobRequirement string `json:"jobRequirement"`
	Kind           string `json:"kind"`
}

type RawModel struct {
	Score               float64       `json:"score"`
	Summary             string        `json:"summary"`
	Evidence            []RawEvidence `json:"evidence"`
	Gaps                []string      `json:"gaps"`
	ConfirmedMismatches []RawMismatch `json:"confirmedMismatches"`
	CandidateUnknowns   []string      `json:"candidateUnknowns"`
	EmployerUnknowns    []string      `json:"employerUnknowns"`
}

// Evidence is a grounded evidence item in the final result.
type Evidence struct {
	Claim    string `json:"claim"`
	SourceID string `json:"sourceId"`
	Quote    string `json:"quote"`
}

// Result is the grounded assessment (the schema-level fields, excluding volatile metadata).
type Result struct {
	Score               float64    `json:"score"`
	Summary             string     `json:"summary"`
	Evidence            []Evidence `json:"evidence"`
	Gaps                []string   `json:"gaps"`
	ConfirmedMismatches []string   `json:"confirmedMismatches"`
	CandidateUnknowns   []string   `json:"candidateUnknowns"`
	EmployerUnknowns    []string   `json:"employerUnknowns"`
}

// Inference is the provider-configurable model call. requestText is the assembled prompt and
// schema is the JSON schema the model must satisfy; a real implementation lives in infer.go.
type Inference func(p Provider, requestText string, schema any) (RawModel, error)

type bankEntry struct {
	EvidenceID string
	SourceID   string
	Quote      string
}

var (
	termRe          = regexp.MustCompile(`[a-z][a-z0-9+.#-]{2,}`)
	paragraphSplit  = regexp.MustCompile(`\n\s*\n`)
	reasonUnknownRe = regexp.MustCompile(`(?i)not mentioned|not documented|no evidence|unknown|not listed|unconfirmed|not established`)
	absenceRe       = regexp.MustCompile(`(?i)\black|does not (?:know|have)|no .{0,30}experience|not know|не зна|нет опыта`)
	feedbackQuoteRe = regexp.MustCompile(`(?i)not known|no experience|don't know|do not know|does not know|не знаю|нет опыта`)
	quoteWSRe       = regexp.MustCompile(`\s+`)
)

type selSource struct {
	ID   string
	Text string
}

func splitParagraphs(s string) []string {
	out := []string{}
	for _, p := range paragraphSplit.Split(s, -1) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// contextForJob selects evidence paragraphs within the budget, ranking by job-term overlap and
// restoring original order, mirroring the TypeScript contextForJob.
func contextForJob(c CandidateContext, jobText string, budget int) ([]selSource, error) {
	terms := map[string]bool{}
	for _, t := range termRe.FindAllString(strings.ToLower(jobText), -1) {
		terms[t] = true
	}
	type statusItem struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Error    string `json:"error,omitempty"`
		Category string `json:"category,omitempty"`
		Scope    string `json:"scope,omitempty"`
	}
	var status []statusItem
	for _, s := range c.Sources {
		status = append(status, statusItem{s.ID, s.Status, s.Error, s.Category, s.Scope})
	}
	headersBytes, _ := json.Marshal(map[string]any{
		"preferences": c.Preferences, "reviewNotes": c.ReviewNotes, "sourceStatus": status,
	})
	headers := len(headersBytes)

	available := []EvidenceSource{}
	for _, s := range c.Sources {
		if s.Text != "" {
			available = append(available, s)
		}
	}
	if budget-headers < len(available)*400 {
		return nil, fmt.Errorf("context budget too small for source overview; increase context_chars")
	}
	left := budget - headers
	selected := []selSource{}
	for i, s := range available {
		allowance := left / (len(available) - i)
		paragraphs := splitParagraphs(s.Text)
		type ranked struct {
			text  string
			index int
			score int
		}
		rs := make([]ranked, len(paragraphs))
		for idx, para := range paragraphs {
			score := 0
			lower := strings.ToLower(para)
			for t := range terms {
				if strings.Contains(lower, t) {
					score++
				}
			}
			rs[idx] = ranked{para, idx, score}
		}
		sort.SliceStable(rs, func(a, b int) bool { return rs[a].score > rs[b].score })
		picked := []ranked{}
		used := 0
		for _, p := range rs {
			if used+len(p.text)+2 <= allowance {
				picked = append(picked, p)
				used += len(p.text) + 2
			}
		}
		if len(picked) == 0 && len(s.Text) > allowance {
			return nil, fmt.Errorf("unsegmentable source %s; budget too small, no silent truncation", s.ID)
		}
		sort.SliceStable(picked, func(a, b int) bool { return picked[a].index < picked[b].index })
		parts := make([]string, len(picked))
		for j, p := range picked {
			parts[j] = p.text
		}
		txt := strings.Join(parts, "\n\n")
		selected = append(selected, selSource{ID: s.ID, Text: txt})
		left -= len(txt)
	}
	return selected, nil
}

func buildBank(selected []selSource) []bankEntry {
	bank := []bankEntry{}
	for _, s := range selected {
		for _, para := range splitParagraphs(s.Text) {
			for offset := 0; offset < len(para); offset += 650 {
				end := offset + 650
				if end > len(para) {
					end = len(para)
				}
				bank = append(bank, bankEntry{
					EvidenceID: fmt.Sprintf("E%d", len(bank)+1),
					SourceID:   s.ID,
					Quote:      para[offset:end],
				})
			}
		}
	}
	return bank
}

// sourceQuote resolves a model-supplied quote to the exact original substring, permitting only
// whitespace differences, mirroring the TypeScript sourceQuote.
func sourceQuote(source, quote string) (string, bool) {
	if strings.Contains(source, quote) {
		return quote, true
	}
	words := quoteWSRe.Split(strings.TrimSpace(quote), -1)
	escaped := make([]string, len(words))
	for i, w := range words {
		escaped[i] = regexp.QuoteMeta(w)
	}
	re, err := regexp.Compile(strings.Join(escaped, `\s+`))
	if err != nil {
		return "", false
	}
	if m := re.FindString(source); m != "" {
		return m, true
	}
	return "", false
}

func findBank(bank []bankEntry, id string) (bankEntry, bool) {
	for _, b := range bank {
		if b.EvidenceID == id {
			return b, true
		}
	}
	return bankEntry{}, false
}

func sourceByID(c CandidateContext, id string) (EvidenceSource, bool) {
	for _, s := range c.Sources {
		if s.ID == id {
			return s, true
		}
	}
	return EvidenceSource{}, false
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// Assess runs the fit assessment for one job against the candidate context, using the provider's
// context budget and the injected model call, and returns the grounded result.
func Assess(c CandidateContext, job Job, p Provider, prompt string, infer Inference) (Result, error) {
	jobData := map[string]any{
		"title": job.Title, "company": job.Company, "description": job.Description,
		"descriptionStatus": job.DescriptionStatus, "location": job.Location,
	}
	jobDataJSON, _ := json.Marshal(jobData)
	schemaJSON, _ := json.Marshal(jsonSchemaPlaceholder)
	overhead := len(prompt) + len(schemaJSON) + len(jobDataJSON) + 1200
	if overhead > p.ContextChars-5000 {
		return Result{}, fmt.Errorf("job exceeds context budget; select larger model budget (no silent truncation)")
	}
	budget := int(math.Floor(float64(p.ContextChars-overhead) * 0.78))
	selected, err := contextForJob(c, job.Description, budget)
	if err != nil {
		return Result{}, err
	}
	bank := buildBank(selected)
	if len(bank) == 0 {
		return Result{}, fmt.Errorf("no candidate evidence available")
	}
	requestText := fmt.Sprintf("%s\nSCHEMA\n%s\nDATA\n%s", prompt, schemaJSON, jobDataJSON)

	raw, err := infer(p, requestText, jsonSchemaPlaceholder)
	if err != nil {
		return Result{}, err
	}
	for _, e := range raw.Evidence {
		if _, ok := findBank(bank, e.EvidenceID); !ok {
			return Result{}, fmt.Errorf("unsupported evidence reference: %s", e.EvidenceID)
		}
	}

	// Resolve each mismatch to its bank entry and require its requirement be an exact job quote.
	jobQuoteText := job.Title + "\n" + job.Description + "\n" + job.Location
	type mismatch struct {
		reason   string
		sourceID string
		quote    string
		kind     string
	}
	var mismatches []mismatch
	for _, m := range raw.ConfirmedMismatches {
		b, ok := findBank(bank, m.EvidenceID)
		if !ok {
			return Result{}, fmt.Errorf("unsupported evidence reference: %s", m.EvidenceID)
		}
		if _, ok := sourceQuote(jobQuoteText, m.JobRequirement); !ok {
			return Result{}, fmt.Errorf("mismatch requirement is not an exact vacancy quote")
		}
		mismatches = append(mismatches, mismatch{m.Reason, b.SourceID, b.Quote, m.Kind})
	}

	var confirmed, unconfirmed []mismatch
	for _, m := range mismatches {
		if reasonUnknownRe.MatchString(m.reason) {
			unconfirmed = append(unconfirmed, m)
			continue
		}
		absence := m.kind == "explicit-absence" || absenceRe.MatchString(m.reason)
		if !absence {
			confirmed = append(confirmed, m)
			continue
		}
		src, _ := sourceByID(c, m.sourceID)
		if src.Category != "owner-confirmed-feedback" || !feedbackQuoteRe.MatchString(m.quote) {
			unconfirmed = append(unconfirmed, m)
		} else {
			confirmed = append(confirmed, m)
		}
	}

	gaps := raw.Gaps
	if gaps == nil {
		gaps = []string{}
	}
	candidateUnknowns := append([]string{}, raw.CandidateUnknowns...)
	for _, m := range unconfirmed {
		candidateUnknowns = append(candidateUnknowns, "Unconfirmed: "+m.reason)
	}
	evidence := []Evidence{}
	for _, e := range raw.Evidence {
		b, _ := findBank(bank, e.EvidenceID)
		evidence = append(evidence, Evidence{Claim: e.Claim, SourceID: b.SourceID, Quote: b.Quote})
	}
	confirmedReasons := []string{}
	for _, m := range confirmed {
		confirmedReasons = append(confirmedReasons, m.reason)
	}

	for _, g := range gaps {
		if !contains(candidateUnknowns, g) {
			candidateUnknowns = append(candidateUnknowns, g)
		}
	}
	keptConfirmed := []string{}
	for _, r := range confirmedReasons {
		if reasonUnknownRe.MatchString(r) {
			candidateUnknowns = append(candidateUnknowns, r)
		} else {
			keptConfirmed = append(keptConfirmed, r)
		}
	}
	confirmedReasons = keptConfirmed

	// Resolve evidence (and confirmed mismatch) quotes to exact original substrings.
	for i, e := range evidence {
		src, _ := sourceByID(c, e.SourceID)
		original, ok := sourceQuote(src.Text, e.Quote)
		if !ok {
			return Result{}, fmt.Errorf("unsupported evidence quote in %s: %q", e.SourceID, e.Quote)
		}
		evidence[i].Quote = original
	}

	employer := raw.EmployerUnknowns
	if employer == nil {
		employer = []string{}
	}
	return Result{
		Score: raw.Score, Summary: raw.Summary, Evidence: evidence, Gaps: gaps,
		ConfirmedMismatches: confirmedReasons, CandidateUnknowns: candidateUnknowns,
		EmployerUnknowns: employer,
	}, nil
}

// jsonSchemaPlaceholder stands in for the model JSON schema in overhead accounting; the exact
// schema bytes do not affect the result, only the budget arithmetic.
var jsonSchemaPlaceholder = map[string]any{"type": "object"}
