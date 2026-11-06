// Package background — preparation worker component.
// See assessment.go for package-level documentation.
package background

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/inspect"
	"github.com/w3con/sifty/internal/prepare"
	"github.com/w3con/sifty/internal/prompts"
	"github.com/w3con/sifty/internal/store"
)

// InspectFn is the injectable function that inspects an application form at the given URL
// and returns the questionnaire it found. The real implementation drives the Playwright
// worker; tests inject a stub.
type InspectFn func(ctx context.Context, applyURL string) (inspect.Questionnaire, error)

// PrepareJob is the injectable function that generates answers for one preparation operation.
type PrepareJob func(
	s *store.Store, d dirs.Dirs, cfg *config.Config,
	jobID, opID int64,
	request map[string]any,
	progressFn func(string),
	inferFn prepare.Inference,
) error

// PreparationWorker processes queued application-preparation operations one at a time.
// Inspect operations run without the inference lock; generate operations acquire it.
type PreparationWorker struct {
	store     *store.Store
	dirs      dirs.Dirs
	cfg       *config.Holder
	prepareFn PrepareJob
	inferFn   prepare.Inference
	inspectFn InspectFn

	mu      sync.Mutex
	busy    bool
	stopped bool
	cancel  context.CancelFunc
}

// NewPreparationWorker creates a worker and recovers any operations that were left in
// 'running' state by a previous process that crashed. prepareFn, inferFn, and inspectFn
// are injectable; nil uses defaults (nil inspectFn silently skips the inspect phase).
func NewPreparationWorker(s *store.Store, d dirs.Dirs, cfg *config.Holder, prepareFn PrepareJob, inferFn prepare.Inference, inspectFn InspectFn) *PreparationWorker {
	if prepareFn == nil {
		prepareFn = defaultPrepareJob
	}
	// No default for inferFn: the real implementation lives in the main package which wires
	// up the full provider routing. Tests inject a stub.
	w := &PreparationWorker{
		store: s, dirs: d, cfg: cfg,
		prepareFn: prepareFn, inferFn: inferFn, inspectFn: inspectFn,
	}
	w.recoverInterrupted()
	return w
}

// Start launches the background goroutine and returns immediately.
func (w *PreparationWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go func() {
		w.tick()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.tick()
			}
		}
	}()
}

// Stop signals the goroutine to stop; it does not wait for an in-flight operation to finish.
func (w *PreparationWorker) Stop() {
	w.mu.Lock()
	w.stopped = true
	w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *PreparationWorker) tick() {
	w.mu.Lock()
	if w.busy || w.stopped {
		w.mu.Unlock()
		return
	}
	w.busy = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.busy = false
		w.mu.Unlock()
	}()

	if err := w.runOnce(); err != nil {
		log.Printf("preparation worker: %v", err)
	}
}

func (w *PreparationWorker) runOnce() error {
	// Find the next queued operation in a transaction so no other process claims it.
	op, err := w.claimNextQueued()
	if err != nil || op == nil {
		return err
	}

	opID := op["id"].(int64)
	jobID := op["jobId"].(int64)
	pid := os.Getpid()
	request := requestMap(op)

	progress := func(msg string) {
		_, _ = w.store.DB.Exec(
			"UPDATE preparation_operations SET progress=?,updated=? WHERE id=? AND status='running'",
			msg, nowMs(), opID)
	}

	// Phase 1 — inspect (no inference lock needed).
	// The inspect phase reads the application form via the Playwright worker. It does not
	// acquire the inference lock so it can run even while model work is in progress.
	doInspect, _ := request["inspect"].(bool)
	if doInspect && w.inspectFn != nil {
		job, err := w.store.Get(jobID)
		if err != nil {
			_, _ = w.store.DB.Exec(
				"UPDATE preparation_operations SET status='failed',error=?,progress='Preparation failed; saved answers preserved',pid=NULL,updated=? WHERE id=? AND status='running'",
				err.Error(), nowMs(), opID)
			return nil
		}
		applyURL := externalApplyURL(job.ApplyURL, job.OriginalURL)
		if applyURL == "" {
			_, _ = w.store.DB.Exec(
				"UPDATE preparation_operations SET status='done',progress=?,pid=NULL,updated=? WHERE id=? AND status='running'",
				"no resolved application form; apply via the original listing", nowMs(), opID)
			return nil
		}

		progress("Reading questionnaire; nothing is filled or submitted")
		ctx := context.Background()
		q, err := w.inspectFn(ctx, applyURL)
		if err != nil {
			_, _ = w.store.DB.Exec(
				"UPDATE preparation_operations SET status='failed',error=?,progress='Preparation failed; saved answers preserved',pid=NULL,updated=? WHERE id=? AND status='running'",
				err.Error(), nowMs(), opID)
			return nil
		}
		if err := inspectIntoDraft(w.store, jobID, q); err != nil {
			_, _ = w.store.DB.Exec(
				"UPDATE preparation_operations SET status='failed',error=?,progress='Preparation failed; saved answers preserved',pid=NULL,updated=? WHERE id=? AND status='running'",
				err.Error(), nowMs(), opID)
			return nil
		}

		// Mark inspect done in the request so a generate-only retry skips inspection.
		request["inspect"] = false
		reqJSON, _ := json.Marshal(request)
		_, _ = w.store.DB.Exec(
			"UPDATE preparation_operations SET request=?,updated=? WHERE id=?",
			string(reqJSON), nowMs(), opID)

		// Honest non-question outcomes.
		// A reachable form with no answerable fields, one that requires sign-in, or one
		// whose destination is a download or non-HTML page, is a legitimate result — mark
		// it done with a descriptive message rather than failed.
		if len(q.Fields) == 0 {
			msg := "No readable questions found. Retry inspection or add questions manually."
			switch q.Status {
			case "login-required":
				msg = "Sign in to continue, then retry inspection. Saved answers are preserved."
			case "no-applicable-form":
				msg = "no resolved application form; apply via the original listing"
			case "backed-off":
				// The site's anti-bot layer blocked the request; BrowserDiscipline has registered
				// the cooldown so the next inspection attempt will wait until it lapses.
				if len(q.Gaps) > 0 && q.Gaps[0] != "" {
					msg = q.Gaps[0]
				} else {
					msg = "The site asked to slow down; inspection is paused. Retry later."
				}
			}
			_, _ = w.store.DB.Exec(
				"UPDATE preparation_operations SET status='done',progress=?,pid=NULL,updated=? WHERE id=? AND status='running'",
				msg, nowMs(), opID)
			return nil
		}
	}

	// Phase 2 — generate (needs inference lock).
	doGenerate, _ := request["generate"].(bool)
	if !doGenerate {
		// Inspect-only or no work left: mark done.
		_, _ = w.store.DB.Exec(
			"UPDATE preparation_operations SET status='done',progress='Questionnaire ready for review; nothing submitted',pid=NULL,updated=? WHERE id=? AND status='running'",
			nowMs(), opID)
		return nil
	}

	// Try to acquire the inference lock. If another process holds it, re-queue this
	// operation (with inspect=false already saved) to be retried on the next tick.
	if err := w.store.AcquireLock("inference", pid); err != nil {
		_, _ = w.store.DB.Exec(
			"UPDATE preparation_operations SET status='queued',pid=NULL,progress='Waiting for shared inference slot',updated=? WHERE id=?",
			nowMs(), opID)
		return nil
	}
	defer func() { _ = w.store.ReleaseLock("inference", pid) }()

	// Run the preparation. Errors are recorded on the operation row; they do not propagate.
	prepErr := w.prepareFn(w.store, w.dirs, w.cfg.Get(), jobID, opID, request, progress, w.inferFn)
	if prepErr != nil {
		_, _ = w.store.DB.Exec(
			"UPDATE preparation_operations SET status='failed',error=?,progress='Preparation failed; saved answers preserved',pid=NULL,updated=? WHERE id=? AND status='running'",
			prepErr.Error(), nowMs(), opID)
		return nil
	}

	_, _ = w.store.DB.Exec(
		"UPDATE preparation_operations SET status='done',progress='Questionnaire ready for review; nothing submitted',pid=NULL,updated=? WHERE id=? AND status='running'",
		nowMs(), opID)
	return nil
}

// claimNextQueued picks the oldest queued operation and transitions it to 'running' in one
// transaction, so parallel processes cannot double-claim the same row. Returns nil,nil when there
// is nothing to process.
func (w *PreparationWorker) claimNextQueued() (map[string]any, error) {
	tx, err := w.store.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	// If any operation is already running (owned by any process), wait.
	var running int
	if err := tx.QueryRow("SELECT COUNT(*) FROM preparation_operations WHERE status='running'").Scan(&running); err != nil {
		return nil, err
	}
	if running > 0 {
		_ = tx.Rollback()
		tx = nil
		return nil, nil
	}

	var opID, jobID int64
	var reqStr string
	err = tx.QueryRow(
		"SELECT id, job_id, request FROM preparation_operations WHERE status='queued' ORDER BY id LIMIT 1").
		Scan(&opID, &jobID, &reqStr)
	if err != nil {
		_ = tx.Rollback()
		tx = nil
		return nil, nil // nothing queued
	}

	pid := os.Getpid()
	if _, err := tx.Exec(
		"UPDATE preparation_operations SET status='running',pid=?,updated=? WHERE id=?",
		pid, nowMs(), opID); err != nil {
		return nil, fmt.Errorf("claim operation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil

	var req map[string]any
	_ = json.Unmarshal([]byte(reqStr), &req)
	return map[string]any{"id": opID, "jobId": jobID, "request": req}, nil
}

// requestMap extracts the request sub-map from an operation map.
func requestMap(op map[string]any) map[string]any {
	if r, ok := op["request"].(map[string]any); ok {
		return r
	}
	return map[string]any{}
}

// recoverInterrupted marks all 'running' preparation operations left by a dead previous process
// as 'interrupted'. Operations owned by living processes are left alone.
func (w *PreparationWorker) recoverInterrupted() {
	rows, err := w.store.DB.Query("SELECT id, pid FROM preparation_operations WHERE status='running'")
	if err != nil {
		return
	}
	var toRecover []int64
	for rows.Next() {
		var id int64
		var pid sql.NullInt64 //nolint:govet
		if err := rows.Scan(&id, &pid); err != nil {
			continue
		}
		alive := false
		if pid.Valid && pid.Int64 > 0 {
			if err := signalProcess(int(pid.Int64)); err == nil {
				alive = true
			}
		}
		if !alive {
			toRecover = append(toRecover, id)
		}
	}
	rows.Close()
	for _, id := range toRecover {
		_, _ = w.store.DB.Exec(
			"UPDATE preparation_operations SET status='interrupted',pid=NULL,updated=?,error='Preparation interrupted; existing draft preserved. Retry explicitly.' WHERE id=?",
			nowMs(), id)
	}
}

// signalProcess checks if a process is alive by sending signal 0.
func signalProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(os.Signal(nil))
}

// defaultPrepareJob is the real preparation implementation. It runs the generate step using the
// prepare core. The inspect step is handled directly in runOnce before this is called.
func defaultPrepareJob(
	s *store.Store, d dirs.Dirs, cfg *config.Config,
	jobID, opID int64,
	request map[string]any,
	progressFn func(string),
	inferFn prepare.Inference,
) error {
	generate, _ := request["generate"].(bool)
	if !generate {
		return nil // nothing to do
	}
	provider, _ := request["provider"].(string)
	if provider == "" {
		provider = cfg.Providers.Stages.Preparation
	}
	if provider == "" {
		return fmt.Errorf("no preparation provider: set stages.preparation in providers.yaml or pass a provider in the request")
	}
	fieldID, _ := request["fieldId"].(string)

	// Load job.
	job, err := s.Get(jobID)
	if err != nil {
		return err
	}

	// Load candidate context.
	ctxRaw, err := s.Context()
	if err != nil || ctxRaw == nil {
		return fmt.Errorf("build candidate context before preparing answers")
	}

	// Convert to prepare.CandidateContext via JSON round-trip.
	ctxJSON, _ := json.Marshal(ctxRaw)
	var pc prepare.CandidateContext
	if err := json.Unmarshal(ctxJSON, &pc); err != nil {
		return fmt.Errorf("parse candidate context: %w", err)
	}

	// Select provider.
	cp, ok := cfg.Providers.Providers[provider]
	if !ok {
		return fmt.Errorf("unknown preparation provider %q", provider)
	}
	p := prepare.Provider{
		Model: cp.Model, ContextChars: cp.ContextChars,
	}

	// Load current draft and convert fields.
	draft, err := s.GetDraft(jobID)
	if err != nil {
		return err
	}
	// Convert []any fields to []prepare.Field via JSON round-trip.
	fieldsJSON, _ := json.Marshal(draft.Fields)
	var fields []prepare.Field
	if err := json.Unmarshal(fieldsJSON, &fields); err != nil {
		return fmt.Errorf("parse draft fields: %w", err)
	}

	prepJob := prepare.Job{
		ID: int(job.ID), Title: job.Title, Company: job.Company,
		Description: job.Description,
	}
	if job.Location != nil {
		prepJob.Location = *job.Location
	}

	lang := ""
	if q, ok := draft.Questionnaire.(map[string]any); ok {
		lang, _ = q["language"].(string)
	}

	progressFn(fmt.Sprintf("Drafting answers with %s; saved edits are protected", provider))
	result, err := prepare.Prepare(pc, prepJob, p, prompts.ApplicationPreparation, lang, fields, fieldID, inferFn)
	if err != nil {
		return err
	}

	// Merge answers back into the draft using JSON round-trip to keep the []any shape.
	answersJSON, _ := json.Marshal(result.Answers)
	var answers []map[string]any
	_ = json.Unmarshal(answersJSON, &answers)

	// Build a lookup by field ID.
	byID := make(map[string]map[string]any, len(answers))
	for _, a := range answers {
		if id, ok := a["id"].(string); ok {
			byID[id] = a
		}
	}

	// Merge into draft fields.
	merged := make([]any, len(draft.Fields))
	for i, raw := range draft.Fields {
		f, ok := raw.(map[string]any)
		if !ok {
			merged[i] = raw
			continue
		}
		id, _ := f["id"].(string)
		if a, found := byID[id]; found {
			// Patch value, evidence and unknowns but preserve everything else.
			out := make(map[string]any, len(f))
			for k, v := range f {
				out[k] = v
			}
			out["value"] = a["value"]
			out["evidence"] = a["evidence"]
			out["unknowns"] = a["unknowns"]
			merged[i] = out
		} else {
			merged[i] = raw
		}
	}
	draft.Fields = merged
	if result.SelectedFile != "" {
		draft.SelectedFile = result.SelectedFile
	}
	_, err = s.WriteDraft(draft, fmt.Sprintf("Generated with %s", provider))
	return err
}
