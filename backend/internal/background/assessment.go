// Package background contains the long-lived goroutines that run assessment, preparation and the
// daily collection scheduler. Each worker is a struct with Start/Stop methods. Callers own the
// lifecycle; the http server starts them and stops them on shutdown.
package background

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/candidate"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/prompts"
	"github.com/w3con/sifty/internal/store"
)

// AssessJob is the injectable function that performs one assessment. The real implementation
// calls the provider; tests inject a stub.
type AssessJob func(s *store.Store, d dirs.Dirs, cfg *config.Config, jobID int64, providerName string, inferenceFn assess.Inference) error

// AssessmentWorker processes the assessment queue. One job runs at a time, serialised by the
// database locks table. The ticker fires every second; the first tick runs immediately.
type AssessmentWorker struct {
	store    *store.Store
	dirs     dirs.Dirs
	cfg      *config.Holder
	assessFn AssessJob
	inferFn  assess.Inference

	mu      sync.Mutex
	busy    bool
	stopped bool
	cancel  context.CancelFunc

	lastEnqueueAll int64 // unix ms
}

// NewAssessmentWorker creates a worker. assessFn and inferFn are injectable; nil uses defaults.
func NewAssessmentWorker(s *store.Store, d dirs.Dirs, cfg *config.Holder, assessFn AssessJob, inferFn assess.Inference) *AssessmentWorker {
	if assessFn == nil {
		assessFn = defaultAssessJob
	}
	if inferFn == nil {
		inferFn = assess.Infer
	}
	w := &AssessmentWorker{
		store: s, dirs: d, cfg: cfg,
		assessFn: assessFn, inferFn: inferFn,
	}
	// Recover any rows stuck in 'running' status from a previous crashed process.
	if err := s.RecoverInterruptedJobs(nowMs()); err != nil {
		log.Printf("assessment worker: recovery error: %v", err)
	}
	return w
}

// Start launches the background goroutine and returns immediately.
func (w *AssessmentWorker) Start() {
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

// Stop signals the goroutine to stop and does not wait for an in-flight assessment to complete.
func (w *AssessmentWorker) Stop() {
	w.mu.Lock()
	w.stopped = true
	w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *AssessmentWorker) tick() {
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
		log.Printf("assessment worker: %v", err)
	}
}

func (w *AssessmentWorker) runOnce() error {
	// Check if queue is enabled.
	state, err := w.store.GetQueueState()
	if err != nil || !state.Enabled {
		return nil
	}

	ms := nowMs()

	// Periodically re-enqueue all jobs to pick up description/context changes.
	if ms-w.lastEnqueueAll > 30000 {
		_ = w.store.EnqueueAll()
		w.lastEnqueueAll = ms
	}

	// Acquire the inference lock to prevent parallel runs (across processes/goroutines).
	pid := os.Getpid()
	if err := w.store.AcquireLock("inference", pid); err != nil {
		return nil // another process holds the lock
	}
	defer func() { _ = w.store.ReleaseLock("inference", pid) }()

	// Check queue enabled and find the next row while holding the lock.
	state, err = w.store.GetQueueState()
	if err != nil || !state.Enabled {
		return nil
	}

	row, err := w.store.NextPendingQueueRow(ms)
	if err != nil || row == nil {
		return err
	}

	// Check if the job still exists.
	if _, err := w.store.Get(row.JobID); err != nil {
		// Job deleted; remove from queue silently.
		_ = w.store.MarkQueueDone(row.JobID, ms)
		return nil
	}

	// Mark running.
	if err := w.store.MarkQueueRunning(row.JobID, pid, ms); err != nil {
		return fmt.Errorf("mark running: %w", err)
	}

	// Run assessment.
	providerName := row.Provider
	if row.OverrideProvider != nil && *row.OverrideProvider != "" {
		providerName = *row.OverrideProvider
	}

	assessErr := w.assessFn(w.store, w.dirs, w.cfg.Get(), row.JobID, providerName, w.inferFn)
	if assessErr != nil {
		_ = w.store.MarkQueueFailed(row.JobID, assessErr.Error(), row.Attempts, nowMs())
		return nil // assessment failure is handled; don't propagate
	}

	_ = w.store.MarkQueueDone(row.JobID, nowMs())
	// EnqueueAll (called every 30 s) will re-enqueue this job if its description changed
	// while inference was running. No need to force a re-enqueue here.
	return nil
}

// nowMs returns the current Unix timestamp in milliseconds.
func nowMs() int64 { return time.Now().UnixMilli() }

// defaultAssessJob is the real assessment implementation. It calls the provider-configured
// inference function with the candidate context and job data.
func defaultAssessJob(s *store.Store, d dirs.Dirs, cfg *config.Config, jobID int64, providerName string, inferFn assess.Inference) error {
	job, err := s.Get(jobID)
	if err != nil {
		return err
	}

	// Load candidate context.
	ctxRaw, err := s.Context()
	if err != nil || ctxRaw == nil {
		return fmt.Errorf("candidate context not available; refresh context first")
	}
	c, err := candidate.MapToContext(ctxRaw)
	if err != nil {
		return err
	}

	// Select provider.
	if providerName == "" {
		providerName = cfg.Providers.Stages.Assessment
	}
	cp, ok := cfg.Providers.Providers[providerName]
	if !ok {
		return fmt.Errorf("unknown provider %q", providerName)
	}
	p := assess.Provider{
		Type: cp.Type, Model: cp.Model, Endpoint: cp.Endpoint,
		APIKeyEnv: cp.APIKeyEnv, TimeoutMS: cp.TimeoutMS, ContextChars: cp.ContextChars,
	}

	// Build candidate context for this job.
	jobCtx, err := candidate.ContextForJob(c, job.Description, p.ContextChars-5000)
	if err != nil {
		return err
	}

	// Translate to assess.CandidateContext.
	var sources []assess.EvidenceSource
	for _, raw := range c.Sources {
		sources = append(sources, assess.EvidenceSource{
			ID: raw.ID, Status: raw.Status, Error: raw.Error, Category: raw.Category, Scope: raw.Scope,
		})
	}
	// Populate text from the job-scoped selection.
	textByID := make(map[string]string)
	for _, sel := range jobCtx.Sources {
		textByID[sel.ID] = sel.Text
	}
	for i := range sources {
		sources[i].Text = textByID[sources[i].ID]
	}

	ac := assess.CandidateContext{
		Preferences: c.Preferences,
		ReviewNotes: c.ReviewNotes,
		Sources:     sources,
	}

	assessJob := assess.Job{
		ID: int(job.ID), Title: job.Title, Company: job.Company,
		Description: job.Description, DescriptionStatus: job.DescriptionStatus,
	}
	if job.Location != nil {
		assessJob.Location = *job.Location
	}

	result, err := assess.Assess(ac, assessJob, p, prompts.Assessment, inferFn)
	if err != nil {
		_ = s.AssessmentFailure(jobID, err.Error(), map[string]any{
			"contextVersion": c.Version, "profileId": c.ProfileID,
			"profileRevision": c.ProfileRevision, "provider": providerName,
			"model": p.Model, "createdAt": assessmentNow(),
		})
		return err
	}

	// Record the exact inputs this result used. A later profile/context change must make this
	// result stale on read rather than letting a late assessment look current.
	var resultMap map[string]any
	encoded, _ := json.Marshal(result)
	_ = json.Unmarshal(encoded, &resultMap)
	resultMap["jobVersion"] = store.JobVersionHash(job.Title, job.Company, job.Description, job.DescriptionStatus, job.Location)
	resultMap["contextVersion"] = c.Version
	resultMap["profileId"] = c.ProfileID
	resultMap["profileRevision"] = c.ProfileRevision
	resultMap["provider"] = providerName
	resultMap["model"] = p.Model
	resultMap["promptVersion"] = promptVersion()
	resultMap["createdAt"] = assessmentNow()

	return s.Assessment(jobID, resultMap)
}

func assessmentNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func promptVersion() string {
	sum := sha256.Sum256([]byte(prompts.Assessment))
	return hex.EncodeToString(sum[:])
}
