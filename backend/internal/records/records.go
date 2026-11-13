// Package records reads and writes the two durable candidate-knowledge file kinds in the data
// directory: remembered application answers (answers/) and owner-confirmed feedback (feedback/).
// Each file carries a base64-encoded JSON record in an HTML comment marker, so the file is both
// human-readable prose and a machine-recoverable record. The markers are SIFT_ANSWER and
// SIFT_FEEDBACK; no project-specific naming is used.
package records

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/w3con/sifty/internal/dirs"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var (
	answerTmpl   = template.Must(template.ParseFS(templatesFS, "templates/answer.md.tmpl"))
	feedbackTmpl = template.Must(template.ParseFS(templatesFS, "templates/feedback.md.tmpl"))

	answerMarkerRe   = regexp.MustCompile(`<!-- SIFT_ANSWER (.+) -->`)
	feedbackMarkerRe = regexp.MustCompile(`<!-- SIFT_FEEDBACK (.+) -->`)
)

// Fact is a conservatively extracted explicit statement from feedback text.
type Fact struct {
	Subject string `json:"subject"`
	Value   string `json:"value"`
	Quote   string `json:"quote"`
}

// AnswerConflict records an unresolved disagreement with a prior answer.
type AnswerConflict struct {
	Record         string `json:"record"`
	PreviousAnswer string `json:"previousAnswer"`
	Resolution     string `json:"resolution"`
}

// AnswerRecord is a remembered application answer. The JSON field names match the current
// TypeScript application so a record is interchangeable between the two.
type AnswerRecord struct {
	ID            string           `json:"id"`
	Created       string           `json:"created"`
	Scope         string           `json:"scope"`
	JobID         int              `json:"jobId"`
	Company       string           `json:"company"`
	URL           string           `json:"url"`
	Question      string           `json:"question"`
	Answer        string           `json:"answer"`
	Language      string           `json:"language"`
	FieldID       string           `json:"fieldId"`
	FieldRevision int              `json:"fieldRevision"`
	DraftRevision int              `json:"draftRevision"`
	Evidence      any              `json:"evidence"`
	Source        string           `json:"source"`
	Conflicts     []AnswerConflict `json:"conflicts"`
	Path          string           `json:"path"`
}

// FeedbackConflict records an unresolved disagreement between feedback statements.
type FeedbackConflict struct {
	Record     string `json:"record"`
	Subject    string `json:"subject"`
	Previous   string `json:"previous"`
	Resolution string `json:"resolution"`
}

// FeedbackRecord is an owner-confirmed candidate feedback statement.
type FeedbackRecord struct {
	ID         string             `json:"id"`
	Created    string             `json:"created"`
	Path       string             `json:"path"`
	JobID      int                `json:"jobId"`
	JobURL     string             `json:"jobUrl"`
	JobTitle   string             `json:"jobTitle"`
	Source     string             `json:"source"`
	Scope      string             `json:"scope"`
	Text       string             `json:"text"`
	Facts      []Fact             `json:"facts"`
	Conflicts  []FeedbackConflict `json:"conflicts"`
	Supersedes *string            `json:"supersedes"`
}

// fence returns a backtick fence long enough to wrap content that may itself contain backticks,
// matching the TypeScript rule: at least three, and one more than the longest run inside.
func fence(content string) string {
	longest := 0
	for _, run := range regexp.MustCompile("`+").FindAllString(content, -1) {
		if len(run) > longest {
			longest = len(run)
		}
	}
	n := longest + 1
	if n < 3 {
		n = 3
	}
	return strings.Repeat("`", n)
}

func encode(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func writeExclusive(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// WriteAnswer renders and writes a remembered answer file under answers/, setting rec.Path.
func WriteAnswer(d dirs.Dirs, rec *AnswerRecord) (string, error) {
	rel := filepath.Join("answers", rec.Created[:10]+"-"+rec.ID+".md")
	path, err := d.Under(rel)
	if err != nil {
		return "", err
	}
	rec.Path = rel
	if rec.Conflicts == nil {
		rec.Conflicts = []AnswerConflict{}
	}
	conflictsJSON, err := json.Marshal(rec.Conflicts)
	if err != nil {
		return "", err
	}
	encoded, err := encode(rec)
	if err != nil {
		return "", err
	}
	original := "Question: " + rec.Question + "\nAnswer: " + rec.Answer
	var buf strings.Builder
	if err := answerTmpl.Execute(&buf, map[string]any{
		"Created": rec.Created, "Scope": rec.Scope, "Company": rec.Company,
		"JobID": rec.JobID, "DraftRevision": rec.DraftRevision, "FieldRevision": rec.FieldRevision,
		"URL": rec.URL, "Language": rec.Language, "Fence": fence(original),
		"Question": rec.Question, "Answer": rec.Answer,
		"ConflictsJSON": string(conflictsJSON), "Encoded": encoded,
	}); err != nil {
		return "", err
	}
	return path, writeExclusive(path, buf.String())
}

// ReadAnswers reads every remembered answer record from answers/.
func ReadAnswers(d dirs.Dirs) ([]AnswerRecord, error) {
	return readRecords[AnswerRecord](d.Answers(), answerMarkerRe)
}

// WriteFeedback renders and writes an owner feedback file under feedback/, setting rec.Path.
func WriteFeedback(d dirs.Dirs, rec *FeedbackRecord) (string, error) {
	rel := filepath.Join("feedback", rec.Created[:10]+"-"+rec.ID+".md")
	path, err := d.Under(rel)
	if err != nil {
		return "", err
	}
	rec.Path = rel
	if rec.Facts == nil {
		rec.Facts = []Fact{}
	}
	if rec.Conflicts == nil {
		rec.Conflicts = []FeedbackConflict{}
	}
	encoded, err := encode(rec)
	if err != nil {
		return "", err
	}

	extraction := "No structured facts extracted. The original statement remains available as attributed evidence; no model is required to save it."
	if len(rec.Facts) > 0 {
		var lines []string
		for _, f := range rec.Facts {
			q, _ := json.Marshal(f.Quote)
			lines = append(lines, fmt.Sprintf("- %s: %s. Original evidence: %s.", f.Subject, f.Value, string(q)))
		}
		extraction = strings.Join(lines, "\n")
	}
	conflicts := "Earlier evidence remains intact. Missing mentions are unknown, not confirmed absence. Any disagreement with CV/KB evidence requires review."
	if len(rec.Conflicts) > 0 {
		cj, _ := json.Marshal(rec.Conflicts)
		conflicts = "Conflicting statements are unresolved: " + string(cj)
	}
	supersedes := "This record is additive; it does not overwrite earlier facts."
	if rec.Supersedes != nil {
		supersedes = "This is an explicit revision of record " + *rec.Supersedes + "; the original is retained."
	}
	scopeText := "general candidate knowledge"
	if rec.Scope != "general" {
		scopeText = "this role only"
	}

	var buf strings.Builder
	if err := feedbackTmpl.Execute(&buf, map[string]any{
		"Date": rec.Created[:10], "Created": rec.Created, "JobTitle": sanitizeTitle(rec.JobTitle),
		"JobURL": rec.JobURL, "ScopeText": scopeText, "Fence": fence(rec.Text), "Text": rec.Text,
		"ExtractionBlock": extraction, "ConflictsBlock": conflicts, "SupersedesBlock": supersedes,
		"Encoded": encoded,
	}); err != nil {
		return "", err
	}
	return path, writeExclusive(path, buf.String())
}

// ReadFeedback reads every owner feedback record from feedback/.
func ReadFeedback(d dirs.Dirs) ([]FeedbackRecord, error) {
	return readRecords[FeedbackRecord](d.Feedback(), feedbackMarkerRe)
}

func sanitizeTitle(title string) string {
	return strings.NewReplacer("[", "", "]", "").Replace(title)
}

func readRecords[T any](dir string, marker *regexp.Regexp) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []T
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		m := marker.FindSubmatch(data)
		if m == nil {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(string(m[1]))
		if err != nil {
			return nil, fmt.Errorf("decode record in %s: %w", e.Name(), err)
		}
		var rec T
		if err := json.Unmarshal(raw, &rec); err != nil {
			return nil, fmt.Errorf("parse record in %s: %w", e.Name(), err)
		}
		out = append(out, rec)
	}
	return out, nil
}

// NowStamp is the RFC3339 timestamp format the records use, matching the TypeScript ISO string.
func NowStamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z07:00") }
