// Package prepare ports the current application's Stage 2 application-answer preparation (the JOB2
// flow) to Go: it selects the unanswered, readable questionnaire fields, assembles the candidate
// evidence bank within the model's context budget, calls a provider-configurable model, and
// post-processes the model's draft into grounded answers. The model call is injectable (see
// Inference); the deterministic selection, grounding and validation around it match the TypeScript
// generateApplication. Browser automation and questionnaire inspection are not part of this package.
//
// The candidate context this package receives is already assembled (feedback and remembered-answer
// sources bound in) by the store subsystem, exactly as the fit-assessment port receives it.
package prepare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// EvidenceSource is one candidate evidence source in the assembled context.
type EvidenceSource struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Category string `json:"category,omitempty"`
	Scope    string `json:"scope,omitempty"`
	Text     string `json:"-"`
}

// CandidateContext is the candidate evidence the preparation draws on.
type CandidateContext struct {
	Preferences any
	ReviewNotes []string
	Sources     []EvidenceSource
}

// Job is the posting whose questionnaire is being answered.
type Job struct {
	ID          int
	Title       string
	Company     string
	Description string
	Location    string
}

// Provider is the inference provider configuration relevant to the preparation call.
type Provider struct {
	Model        string
	ContextChars int
}

// AnswerValue is a questionnaire answer value, which is either a single string (text, textarea,
// radio, select) or a list of strings (checkbox), mirroring the TypeScript string|string[] union.
type AnswerValue struct {
	IsList bool
	Str    string
	List   []string
}

// MarshalJSON renders a list value as a JSON array and a scalar value as a JSON string.
func (v AnswerValue) MarshalJSON() ([]byte, error) {
	if v.IsList {
		l := v.List
		if l == nil {
			l = []string{}
		}
		return json.Marshal(l)
	}
	return json.Marshal(v.Str)
}

// UnmarshalJSON accepts either a JSON array (a list value) or a JSON string (a scalar value).
func (v *AnswerValue) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		v.IsList = true
		v.Str = ""
		list := []string{}
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return err
		}
		v.List = list
		return nil
	}
	v.IsList = false
	v.List = nil
	return json.Unmarshal(trimmed, &v.Str)
}

// nonEmpty reports whether the value carries content: a non-empty list, or a string that is not
// blank after trimming, mirroring the TypeScript (Array.isArray(v)?v.length:v.trim()) truthiness.
func (v AnswerValue) nonEmpty() bool {
	if v.IsList {
		return len(v.List) > 0
	}
	return strings.TrimSpace(v.Str) != ""
}

// Field is one questionnaire field with its current answer state, mirroring the TypeScript
// AnswerField (question metadata plus value, evidence, unknowns and revision bookkeeping).
type Field struct {
	ID           string      `json:"id"`
	Label        string      `json:"label"`
	Instructions *string     `json:"instructions,omitempty"`
	LabelStatus  *string     `json:"labelStatus,omitempty"`
	Type         string      `json:"type"`
	Options      []string    `json:"options"`
	Required     bool        `json:"required"`
	MaxLength    *int        `json:"maxLength"`
	Accept       string      `json:"accept"`
	SourceURL    string      `json:"sourceUrl"`
	Origin       string      `json:"origin"`
	Language     string      `json:"language"`
	Signature    string      `json:"signature"`
	ControlName  string      `json:"controlName"`
	Value        AnswerValue `json:"value"`
	Revision     int         `json:"revision"`
	Edited       bool        `json:"edited"`
	Removed      bool        `json:"removed"`
}

// questionView is a field stripped of its answer state, serialised into the request exactly as the
// TypeScript map(({value,evidence,unknowns,edited,revision,...q})=>q) does. Its JSON feeds term
// extraction and budget accounting; the key order matches the TypeScript question object.
type questionView struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Instructions *string  `json:"instructions,omitempty"`
	LabelStatus  *string  `json:"labelStatus,omitempty"`
	Type         string   `json:"type"`
	Options      []string `json:"options"`
	Required     bool     `json:"required"`
	MaxLength    *int     `json:"maxLength"`
	Accept       string   `json:"accept"`
	SourceURL    string   `json:"sourceUrl"`
	Origin       string   `json:"origin"`
	Language     string   `json:"language"`
	Signature    string   `json:"signature"`
	ControlName  string   `json:"controlName"`
}

// Evidence is one resolved supporting quotation in a prepared answer.
type Evidence struct {
	SourceID string `json:"sourceId"`
	Quote    string `json:"quote"`
}

// Answer is one prepared answer: the grounded value, its supporting evidence and any unknowns.
type Answer struct {
	ID       string      `json:"id"`
	Value    AnswerValue `json:"value"`
	Unknowns []string    `json:"unknowns"`
	Evidence []Evidence  `json:"evidence"`
}

// Result is the prepared draft at the schema level: the model's language and selected file, and the
// grounded answers for the selected fields.
type Result struct {
	Language     string   `json:"language"`
	SelectedFile string   `json:"selectedFile"`
	Answers      []Answer `json:"answers"`
}

// ModelAnswer is one answer in the model's structured output, referencing evidence by bank IDs.
type ModelAnswer struct {
	ID          string      `json:"id"`
	Value       AnswerValue `json:"value"`
	EvidenceIDs []string    `json:"evidenceIds"`
	Unknowns    []string    `json:"unknowns"`
}

// ModelOutput is the model's structured output for the whole questionnaire.
type ModelOutput struct {
	Language     string        `json:"language"`
	SelectedFile string        `json:"selectedFile"`
	Answers      []ModelAnswer `json:"answers"`
}

// Inference is the provider-configurable model call. prompt is the assembled request and schema is
// the JSON schema the model must satisfy.
type Inference func(p Provider, prompt string, schema any) (ModelOutput, error)

type bankEntry struct {
	ID       string
	SourceID string
	Quote    string
}

type selSource struct {
	ID   string
	Text string
}

var (
	termRe         = regexp.MustCompile(`[a-z][a-z0-9+.#-]{2,}`)
	paragraphSplit = regexp.MustCompile(`\n\s*\n`)
)

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
// restoring original order, mirroring the TypeScript contextForJob. It returns the selected sources
// and the overview headers (used for the prompt body).
func contextForJob(c CandidateContext, jobText string, budget int) ([]selSource, json.RawMessage, error) {
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
	overview, _ := json.Marshal(map[string]any{
		"preferences": c.Preferences, "reviewNotes": c.ReviewNotes, "sourceStatus": status,
	})
	headers := len(overview)

	available := []EvidenceSource{}
	for _, s := range c.Sources {
		if s.Text != "" {
			available = append(available, s)
		}
	}
	if budget-headers < len(available)*400 {
		return nil, nil, fmt.Errorf("context budget too small for source overview; increase context_chars")
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
			return nil, nil, fmt.Errorf("unsegmentable source %s; budget too small, no silent truncation", s.ID)
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
	return selected, overview, nil
}

// buildBank flattens the selected sources into one evidence entry per paragraph, numbered E1, E2, …
// in order, mirroring the TypeScript bank construction (no fixed-size chunking).
func buildBank(selected []selSource) []bankEntry {
	bank := []bankEntry{}
	for _, s := range selected {
		for _, quote := range splitParagraphs(s.Text) {
			bank = append(bank, bankEntry{
				ID: fmt.Sprintf("E%d", len(bank)+1), SourceID: s.ID, Quote: quote,
			})
		}
	}
	return bank
}

func findBank(bank []bankEntry, id string) (bankEntry, bool) {
	for _, b := range bank {
		if b.ID == id {
			return b, true
		}
	}
	return bankEntry{}, false
}

func findField(fields []Field, id string) (Field, bool) {
	for _, f := range fields {
		if f.ID == id {
			return f, true
		}
	}
	return Field{}, false
}

func optionAllowed(options []string, v string) bool {
	for _, o := range options {
		if o == v {
			return true
		}
	}
	return false
}

var optionTypes = map[string]bool{"radio": true, "select": true, "checkbox": true}

// Prepare runs application-answer preparation for one job's questionnaire. fieldID selects a single
// field to prepare; when empty, every unanswered readable field is prepared. It returns the grounded
// answers for the selected fields together with the model's language and selected file.
func Prepare(c CandidateContext, job Job, p Provider, prompt string, questionnaireLanguage string,
	baseFields []Field, fieldID string, infer Inference) (Result, error) {
	// Select the unanswered, readable fields, mirroring the TypeScript filter.
	var fields []Field
	for _, f := range baseFields {
		if f.Removed {
			continue
		}
		if f.LabelStatus != nil && *f.LabelStatus == "uncertain" {
			continue
		}
		if f.Type == "file" {
			continue
		}
		if fieldID != "" {
			if f.ID == fieldID {
				fields = append(fields, f)
			}
			continue
		}
		if !f.Edited && !f.Value.nonEmpty() {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		return Result{}, fmt.Errorf("no unanswered readable questions; inspect, add questions or regenerate an individual answer")
	}

	views := make([]questionView, len(fields))
	for i, f := range fields {
		opts := f.Options
		if opts == nil {
			opts = []string{}
		}
		views[i] = questionView{
			ID: f.ID, Label: f.Label, Instructions: f.Instructions, LabelStatus: f.LabelStatus,
			Type: f.Type, Options: opts, Required: f.Required, MaxLength: f.MaxLength, Accept: f.Accept,
			SourceURL: f.SourceURL, Origin: f.Origin, Language: f.Language, Signature: f.Signature,
			ControlName: f.ControlName,
		}
	}
	questions, _ := json.Marshal(views)

	schemaJSON, _ := json.Marshal(schemaPlaceholder)
	overhead := len(prompt) + len(questions) + len(job.Description) + len(schemaJSON) + 3000
	budget := p.ContextChars - overhead
	if budget > 45000 {
		budget = 45000
	}

	lang := questionnaireLanguage
	if lang == "" {
		lang = "Infer from each question"
	}

	build := func(b int) ([]selSource, []bankEntry, string, error) {
		selected, overview, err := contextForJob(c, job.Description+" "+string(questions), b)
		if err != nil {
			return nil, nil, "", err
		}
		bank := buildBank(selected)
		bankJSON, _ := json.Marshal(bankView(bank))
		requestText := fmt.Sprintf("%s\nJSON schema: %s\nQuestionnaire: %s\nLanguage: %s\nJob (untrusted): %s\nCandidate overview: %s\nEvidence bank: %s",
			prompt, schemaJSON, questions, lang, job.Description, overview, bankJSON)
		return selected, bank, requestText, nil
	}

	_, bank, requestText, err := build(budget)
	if err != nil {
		return Result{}, err
	}
	for len(requestText) > p.ContextChars {
		budget -= len(requestText) - p.ContextChars + 1000
		if budget < 3000 {
			return Result{}, fmt.Errorf("questionnaire exceeds provider budget; prepare one field at a time")
		}
		_, bank, requestText, err = build(budget)
		if err != nil {
			return Result{}, err
		}
	}

	out, err := infer(p, requestText, schemaPlaceholder)
	if err != nil {
		return Result{}, err
	}

	seen := map[string]bool{}
	answers := []Answer{}
	for _, a := range out.Answers {
		field, ok := findField(fields, a.ID)
		if !ok || seen[a.ID] {
			return Result{}, fmt.Errorf("model returned unknown or duplicate question")
		}
		seen[a.ID] = true

		evidence := []Evidence{}
		for _, eid := range a.EvidenceIDs {
			b, ok := findBank(bank, eid)
			if !ok {
				return Result{}, fmt.Errorf("model returned unsupported evidence")
			}
			evidence = append(evidence, Evidence{SourceID: b.SourceID, Quote: b.Quote})
		}

		value := a.Value
		unknowns := a.Unknowns
		if unknowns == nil {
			unknowns = []string{}
		}

		if field.Type == "checkbox" {
			if !value.IsList {
				return Result{}, fmt.Errorf("model answer type does not match field")
			}
		} else if value.IsList {
			return Result{}, fmt.Errorf("model answer type does not match field")
		}

		if optionTypes[field.Type] {
			vals := value.List
			if !value.IsList {
				vals = []string{value.Str}
			}
			for _, v := range vals {
				if v != "" && !optionAllowed(field.Options, v) {
					return Result{}, fmt.Errorf("model invented an unavailable option")
				}
			}
		}

		if field.MaxLength != nil && !value.IsList && len(value.Str) > *field.MaxLength {
			return Result{}, fmt.Errorf("model exceeded field length; retry this field")
		}

		if value.nonEmpty() && len(evidence) == 0 {
			if field.Type == "checkbox" {
				value = AnswerValue{IsList: true, List: []string{}}
			} else {
				value = AnswerValue{Str: ""}
			}
			unknowns = append(unknowns, "No supporting candidate evidence; owner clarification required.")
		}

		answers = append(answers, Answer{ID: a.ID, Value: value, Unknowns: unknowns, Evidence: evidence})
	}

	for _, f := range fields {
		if seen[f.ID] {
			continue
		}
		value := AnswerValue{Str: ""}
		if f.Type == "checkbox" {
			value = AnswerValue{IsList: true, List: []string{}}
		}
		answers = append(answers, Answer{
			ID: f.ID, Value: value,
			Unknowns: []string{"No draft returned for this field; owner review required."},
			Evidence: []Evidence{},
		})
	}

	selectedFile := out.SelectedFile
	if selectedFile != "cv.pdf" && selectedFile != "profile.pdf" {
		selectedFile = ""
	}
	return Result{Language: out.Language, SelectedFile: selectedFile, Answers: answers}, nil
}

type bankViewEntry struct {
	SourceID string `json:"sourceId"`
	Quote    string `json:"quote"`
	ID       string `json:"id"`
}

func bankView(bank []bankEntry) []bankViewEntry {
	out := make([]bankViewEntry, len(bank))
	for i, b := range bank {
		out[i] = bankViewEntry{SourceID: b.SourceID, Quote: b.Quote, ID: b.ID}
	}
	return out
}

// schemaPlaceholder stands in for the model JSON schema in overhead accounting; the exact schema
// bytes do not affect the result, only the budget arithmetic, which the fixture keeps clear of the
// shrink boundary.
var schemaPlaceholder = map[string]any{"type": "object"}
