package background

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/inspect"
	"github.com/w3con/sifty/internal/model"
)

// stubInspectFn returns an InspectFn that returns the given questionnaire.
func stubInspectFn(q inspect.Questionnaire, err error) InspectFn {
	return func(ctx context.Context, applyURL string) (inspect.Questionnaire, error) {
		return q, err
	}
}

// queueInspectOnly enqueues an inspect-only (no generate) preparation operation for a job.
func queueInspectOnly(t *testing.T, s interface{ QueuePreparation(int64, any) (map[string]any, error) }, jobID int64) {
	t.Helper()
	_, err := s.QueuePreparation(jobID, map[string]any{"inspect": true, "generate": false})
	if err != nil {
		t.Fatalf("QueuePreparation (inspect-only): %v", err)
	}
}

// seedPreselectedJob creates a job, pre-selects it, and returns its ID.
func seedPreselectedJob(t *testing.T, s interface {
	Upsert(model.JobInput, bool) (int64, error)
	Status(int64, string, bool) (*model.Job, error)
}, title, applyURL string) int64 {
	t.Helper()
	inp := model.JobInput{
		Title: title, Company: "TestCo",
		OriginalURL:       "https://example.com/jobs/" + title,
		Description:       "A role description.",
		DescriptionStatus: "full",
		Source:            "manual",
	}
	if applyURL != "" {
		inp.ApplyURL = &applyURL
	}
	id, err := s.Upsert(inp, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := s.Status(id, "saved", false); err != nil {
		t.Fatalf("Status → preselected: %v", err)
	}
	return id
}

// waitForOpStatus polls the operation list until the first operation reaches wantStatus
// or deadline passes.
func waitForOpStatus(t *testing.T, s interface {
	PreparationOperations(int64) ([]map[string]any, error)
}, jobID int64, wantStatus string, deadline time.Duration) string {
	t.Helper()
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		ops, err := s.PreparationOperations(jobID)
		if err != nil {
			t.Fatalf("PreparationOperations: %v", err)
		}
		if len(ops) > 0 {
			if status, _ := ops[0]["status"].(string); status == wantStatus {
				return status
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	ops, _ := s.PreparationOperations(jobID)
	if len(ops) > 0 {
		status, _ := ops[0]["status"].(string)
		return status
	}
	return ""
}

// --- Step 1.1 ---

// TestInspectOnlyOperationPopulatesDraft verifies that an inspect-only preparation operation
// calls the InspectFn, stores the returned questionnaire as the draft's fields, and marks the
// operation done.
func TestInspectOnlyOperationPopulatesDraft(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID := seedPreselectedJob(t, s, "Inspector", "https://apply.example.com/123")

	queueInspectOnly(t, s, jobID)

	questionnaire := inspect.Questionnaire{
		URL:       "https://apply.example.com/123",
		Status:    "partial",
		Language:  "en",
		CapturedAt: "2026-11-03T00:00:00.000Z",
		Fields: []inspect.Question{
			{ID: "q1", Label: "Cover letter", Type: "textarea", Origin: "site", SourceURL: "https://apply.example.com/123"},
			{ID: "q2", Label: "Years of experience", Type: "text", Origin: "site", SourceURL: "https://apply.example.com/123"},
		},
	}

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspectFn(questionnaire, nil))
	w.Start()
	defer w.Stop()

	status := waitForOpStatus(t, s, jobID, "done", 3*time.Second)
	if status != "done" {
		t.Fatalf("operation status: got %q, want %q", status, "done")
	}

	draft, err := s.GetDraft(jobID)
	if err != nil {
		t.Fatalf("GetDraft: %v", err)
	}
	if len(draft.Fields) != 2 {
		t.Fatalf("draft fields count: got %d, want 2", len(draft.Fields))
	}
	f0, ok := draft.Fields[0].(map[string]any)
	if !ok {
		t.Fatalf("draft.Fields[0] is not a map")
	}
	if f0["id"] != "q1" {
		t.Errorf("draft.Fields[0].id: got %v, want q1", f0["id"])
	}
	if f0["label"] != "Cover letter" {
		t.Errorf("draft.Fields[0].label: got %v, want Cover letter", f0["label"])
	}
	// Verify answer fields were initialized.
	if f0["value"] != "" {
		t.Errorf("draft.Fields[0].value: got %v, want empty string", f0["value"])
	}
	if _, ok := f0["evidence"].([]any); !ok {
		t.Errorf("draft.Fields[0].evidence: expected []any, got %T", f0["evidence"])
	}
}

// --- Step 1.2 ---

// TestInspectCompletesWithInferenceLockHeld verifies that an inspect-only operation runs to
// completion even when the inference lock is held. The lock is held by the test process itself,
// which is alive, so the worker's generate phase would block — but inspect never reaches it.
func TestInspectCompletesWithInferenceLockHeld(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	// Acquire the inference lock in this process to simulate a busy generate worker.
	thisPID := os.Getpid()
	if err := s.AcquireLock("inference", thisPID); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}
	defer func() { _ = s.ReleaseLock("inference", thisPID) }()
	// Confirm the lock is active: trying to acquire it again must fail.
	if err := s.AcquireLock("inference", thisPID); err == nil {
		t.Fatal("expected lock to be held; second AcquireLock should fail")
	}

	jobID := seedPreselectedJob(t, s, "InspectUnderLock", "https://apply.example.com/lock-test")
	queueInspectOnly(t, s, jobID)

	q := inspect.Questionnaire{
		URL: "https://apply.example.com/lock-test", Status: "partial",
		Fields: []inspect.Question{
			{ID: "ql1", Label: "Why us?", Type: "textarea", Origin: "site", SourceURL: "https://apply.example.com/lock-test"},
		},
	}

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspectFn(q, nil))
	w.Start()
	defer w.Stop()

	// The operation must complete while the inference lock stays held.
	status := waitForOpStatus(t, s, jobID, "done", 3*time.Second)
	if status != "done" {
		t.Fatalf("inspect-only operation did not complete while inference lock was held; status=%q", status)
	}

	// Confirm the lock is still held by this process (the inspect phase never touched it).
	if err := s.AcquireLock("inference", thisPID); err == nil {
		t.Error("inference lock should still be held; the inspect phase must not acquire it")
	}
}

// --- Step 1.3 ---

// TestInspectNoFieldsReturnsHonestDone verifies that a form with no answerable fields
// (status "unsupported") is marked done with a descriptive message rather than failed.
func TestInspectNoFieldsReturnsHonestDone(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID := seedPreselectedJob(t, s, "NoFields", "https://apply.example.com/no-fields")
	queueInspectOnly(t, s, jobID)

	// A reachable form with no answerable fields.
	q := inspect.Questionnaire{
		URL:    "https://apply.example.com/no-fields",
		Status: "unsupported",
		Fields: []inspect.Question{},
	}

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspectFn(q, nil))
	w.Start()
	defer w.Stop()

	status := waitForOpStatus(t, s, jobID, "done", 3*time.Second)
	if status != "done" {
		t.Fatalf("no-fields operation: want status 'done', got %q", status)
	}

	ops, err := s.PreparationOperations(jobID)
	if err != nil || len(ops) == 0 {
		t.Fatalf("PreparationOperations: %v", err)
	}
	prog, _ := ops[0]["progress"].(string)
	if !strings.Contains(prog, "No readable questions") {
		t.Errorf("no-fields progress message: got %q, want to contain 'No readable questions'", prog)
	}
	if _, hasErr := ops[0]["error"]; hasErr {
		t.Errorf("no-fields operation should not have an error field, got: %v", ops[0]["error"])
	}
}

// --- Step 2.1 ---

// TestInspectDownloadDestinationIsDone verifies that when the Playwright worker returns a
// questionnaire with status "no-applicable-form" (the status it emits when navigation hits a
// download or non-HTML destination), the operation ends done with the documented message rather
// than failed.
func TestInspectDownloadDestinationIsDone(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID := seedPreselectedJob(t, s, "DownloadDest", "https://files.example.com/job.pdf")
	queueInspectOnly(t, s, jobID)

	// Worker returns no-applicable-form (the status emitted for download/non-HTML destinations).
	q := inspect.Questionnaire{
		URL:    "https://files.example.com/job.pdf",
		Status: "no-applicable-form",
		Fields: []inspect.Question{},
	}

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspectFn(q, nil))
	w.Start()
	defer w.Stop()

	status := waitForOpStatus(t, s, jobID, "done", 3*time.Second)
	if status != "done" {
		t.Fatalf("download-destination operation: want status 'done', got %q", status)
	}

	ops, err := s.PreparationOperations(jobID)
	if err != nil || len(ops) == 0 {
		t.Fatalf("PreparationOperations: %v", err)
	}
	prog, _ := ops[0]["progress"].(string)
	const wantMsg = "no resolved application form; apply via the original listing"
	if prog != wantMsg {
		t.Errorf("download-destination progress message: got %q, want %q", prog, wantMsg)
	}
}

// TestInspectEmptyApplyURLIsDone verifies the case where a job has no dedicated apply URL
// (applyURL is nil) and the worker returns "no-applicable-form" when navigating to the
// original URL (e.g. the original listing URL returns a download or non-HTML page). The
// operation must end done with the documented message rather than failed. This matches job 758,
// whose apply URL is unresolved so the inspection falls back to the original listing URL.
func TestInspectEmptyApplyURLIsDone(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	// Seed a job with no dedicated apply URL; externalApplyURL will fall back to originalURL.
	jobID := seedPreselectedJob(t, s, "NoApplyURL", "")
	queueInspectOnly(t, s, jobID)

	// Worker returns no-applicable-form when navigating to the original listing URL returns
	// a download or non-HTML page.
	q := inspect.Questionnaire{
		Status: "no-applicable-form",
		Fields: []inspect.Question{},
	}

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspectFn(q, nil))
	w.Start()
	defer w.Stop()

	status := waitForOpStatus(t, s, jobID, "done", 3*time.Second)
	if status != "done" {
		t.Fatalf("no-apply-URL operation: want status 'done', got %q", status)
	}

	ops, err := s.PreparationOperations(jobID)
	if err != nil || len(ops) == 0 {
		t.Fatalf("PreparationOperations: %v", err)
	}
	prog, _ := ops[0]["progress"].(string)
	const wantMsg = "no resolved application form; apply via the original listing"
	if prog != wantMsg {
		t.Errorf("no-apply-URL progress message: got %q, want %q", prog, wantMsg)
	}
}

// TestInspectBackedOffReturnsHonestDone verifies that when the browser discipline returns
// "backed-off" (the site rate-limited the request), the operation ends done with the gap
// message rather than "No readable questions found".
func TestInspectBackedOffReturnsHonestDone(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID := seedPreselectedJob(t, s, "BackedOff", "https://ats.example.com/apply/123")
	queueInspectOnly(t, s, jobID)

	// Worker returns backed-off: the site's anti-bot layer blocked the request.
	const gapMsg = "Host ats.example.com is on cooldown after a previous block; inspection paused."
	q := inspect.Questionnaire{
		URL:    "https://ats.example.com/apply/123",
		Status: "backed-off",
		Fields: []inspect.Question{},
		Gaps:   []string{gapMsg},
	}

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspectFn(q, nil))
	w.Start()
	defer w.Stop()

	status := waitForOpStatus(t, s, jobID, "done", 3*time.Second)
	if status != "done" {
		t.Fatalf("backed-off operation: want status 'done', got %q", status)
	}

	ops, err := s.PreparationOperations(jobID)
	if err != nil || len(ops) == 0 {
		t.Fatalf("PreparationOperations: %v", err)
	}
	prog, _ := ops[0]["progress"].(string)
	if strings.Contains(prog, "No readable questions") {
		t.Errorf("backed-off progress must not say 'No readable questions', got: %q", prog)
	}
	if prog != gapMsg {
		t.Errorf("backed-off progress: got %q, want %q", prog, gapMsg)
	}
	if _, hasErr := ops[0]["error"]; hasErr {
		t.Errorf("backed-off operation must not have an error field, got: %v", ops[0]["error"])
	}
}

// TestInspectLoginRequiredReturnsHonestDone verifies that a form requiring owner sign-in
// is marked done with a descriptive message rather than failed.
func TestInspectLoginRequiredReturnsHonestDone(t *testing.T) {
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()

	jobID := seedPreselectedJob(t, s, "LoginRequired", "https://apply.example.com/login-required")
	queueInspectOnly(t, s, jobID)

	// A form that requires owner sign-in.
	q := inspect.Questionnaire{
		URL:    "https://apply.example.com/login-required",
		Status: "login-required",
		Fields: []inspect.Question{},
	}

	w := NewPreparationWorker(s, d, config.NewHolder(cfg), nil, nil, stubInspectFn(q, nil))
	w.Start()
	defer w.Stop()

	status := waitForOpStatus(t, s, jobID, "done", 3*time.Second)
	if status != "done" {
		t.Fatalf("login-required operation: want status 'done', got %q", status)
	}

	ops, err := s.PreparationOperations(jobID)
	if err != nil || len(ops) == 0 {
		t.Fatalf("PreparationOperations: %v", err)
	}
	prog, _ := ops[0]["progress"].(string)
	if !strings.Contains(prog, "Sign in") {
		t.Errorf("login-required progress message: got %q, want to contain 'Sign in'", prog)
	}
	if _, hasErr := ops[0]["error"]; hasErr {
		t.Errorf("login-required operation should not have an error field, got: %v", ops[0]["error"])
	}
}
