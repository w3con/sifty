package records

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/dirs"
)

// literalChunks returns the template's fixed text between its {{...}} actions, dropping blanks.
func literalChunks(tmpl string) []string {
	var chunks []string
	for len(tmpl) > 0 {
		open := strings.Index(tmpl, "{{")
		if open < 0 {
			chunks = append(chunks, tmpl)
			break
		}
		chunks = append(chunks, tmpl[:open])
		close := strings.Index(tmpl, "}}")
		if close < 0 {
			break
		}
		tmpl = tmpl[close+2:]
	}
	var out []string
	for _, c := range chunks {
		if s := strings.TrimSpace(c); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// assertMatchesTemplate checks every fixed line of the template appears in the output in order,
// so a produced file is proven to follow the template skeleton rather than merely containing it.
func assertMatchesTemplate(t *testing.T, tmplPath, output string) {
	t.Helper()
	raw, err := templatesFS.ReadFile(tmplPath)
	if err != nil {
		t.Fatalf("read template %s: %v", tmplPath, err)
	}
	pos := 0
	for _, chunk := range literalChunks(string(raw)) {
		for _, line := range strings.Split(chunk, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			idx := strings.Index(output[pos:], line)
			if idx < 0 {
				t.Fatalf("template line %q not found in output after position %d", line, pos)
			}
			pos += idx + len(line)
		}
	}
}

func tempDirs(t *testing.T) dirs.Dirs {
	t.Helper()
	d := dirs.Dirs{Root: t.TempDir()}
	if err := d.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	return d
}

func TestAnswerFirstWriteMatchesTemplate(t *testing.T) {
	d := tempDirs(t)
	rec := &AnswerRecord{
		ID: "a1", Created: NowStamp(time.Now()), Scope: "company", JobID: 7,
		Company: "Acme", URL: "https://acme.test/job/7", Question: "Why us?",
		Answer: "Because of `backticks` in the answer.", Language: "en",
		FieldID: "q1", FieldRevision: 2, DraftRevision: 3, Source: "test",
	}
	path, err := WriteAnswer(d, rec)
	if err != nil {
		t.Fatalf("write answer: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	assertMatchesTemplate(t, "templates/answer.md.tmpl", string(data))

	got, err := ReadAnswers(d)
	if err != nil {
		t.Fatalf("read answers: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a1" || got[0].Answer != rec.Answer {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestFeedbackFirstWriteMatchesTemplate(t *testing.T) {
	d := tempDirs(t)
	rec := &FeedbackRecord{
		ID: "f1", Created: NowStamp(time.Now()), JobID: 7,
		JobURL: "https://acme.test/job/7", JobTitle: "Staff [Engineer]",
		Source: "test", Scope: "general", Text: "Statement with ```fence``` inside.",
		Facts: []Fact{{Subject: "docker", Value: "explicit experience", Quote: "Docker: experienced"}},
	}
	path, err := WriteFeedback(d, rec)
	if err != nil {
		t.Fatalf("write feedback: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	assertMatchesTemplate(t, "templates/feedback.md.tmpl", string(data))
	if strings.Contains(string(data), "[Engineer]") {
		t.Fatalf("job title brackets were not sanitized")
	}

	got, err := ReadFeedback(d)
	if err != nil {
		t.Fatalf("read feedback: %v", err)
	}
	if len(got) != 1 || got[0].ID != "f1" || len(got[0].Facts) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestWriteIsExclusive(t *testing.T) {
	d := tempDirs(t)
	rec := &AnswerRecord{ID: "dup", Created: NowStamp(time.Now()), Scope: "job", Question: "q", Answer: "a", Language: "en"}
	if _, err := WriteAnswer(d, rec); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if _, err := WriteAnswer(d, rec); err == nil {
		t.Fatalf("second write to the same path should fail, but succeeded")
	}
}
