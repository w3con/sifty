package background

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/modellock"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)

// FitRescorer recomputes and stores the two-score result of one vacancy from what is already
// persisted. matchsvc.Service implements it; the worker calls it after every model pass.
type FitRescorer interface {
	RescoreFit(ctx context.Context, jobID int64) error
}

// Worker states reported by FitWorker.Status.
const (
	FitStateIdle    = "idle"    // nothing owed
	FitStateWorking = "working" // a pass is running or work is queued
	FitStatePaused  = "paused"  // paused in configuration
	FitStateBlocked = "blocked" // the configured provider may not be used for this work
	FitStateWaiting = "waiting" // the model could not be reached; retrying with backoff
	FitStateLimit   = "limit"   // today's configured limit is reached
	FitStateNoProf  = "no_profile"
)

// FitStatus is the worker's observable state: the counts the owner may see in Settings and the
// reason it is not working when it is not.
type FitStatus struct {
	State          string `json:"state"`
	Message        string `json:"message,omitempty"`
	ModelAvailable bool   `json:"modelAvailable"`
	Usable         int    `json:"usable"`
	Extracted      int    `json:"extracted"`
	NoRequirements int    `json:"noRequirements"`
	WithRequires   int    `json:"withRequirements"`
	Matched        int    `json:"matched"`
	PendingExtract int    `json:"pendingExtraction"`
	PendingMatch   int    `json:"pendingMatching"`
	Cooling        int    `json:"failedRecently"`
	LastError      string `json:"lastError,omitempty"`
}

// FitWorker runs the two model passes over the vacancy catalogue in the background: one worker,
// one vacancy at a time, each model call under the same cross-process inference lock the detailed
// assessment queue uses, so the two never use the provider together. It owns no scoring; after each
// pass it asks the rescorer to recompute the stored result. It never touches the assessment queue.
type FitWorker struct {
	store *store.Store
	cfg   *config.Holder
	fit   FitRescorer
	// Infer is the provider call boundary; nil selects the real provider. Tests inject a stub.
	Infer vacancyai.InferFunc
	// Now, FailureCooldown, IdleWait and MaxBackoff are injectable for tests.
	Now             func() time.Time
	FailureCooldown time.Duration
	IdleWait        time.Duration
	MinBackoff      time.Duration
	MaxBackoff      time.Duration
	// Ready, when set, delays the first step until it is closed. The server closes it once the
	// startup rescoring has given every vacancy a Fits you, so the first order is the real one.
	Ready <-chan struct{}
	// PassBackoff is the pause between the attempts one pass makes; zero keeps the pass default.
	PassBackoff time.Duration

	nudge chan struct{}

	mu     sync.Mutex
	status FitStatus
	// counts are the plan statistics of the last planning pass (Step). Status serves them from
	// memory so that a status request never reads the database; they lag the stored state by at most
	// one planning pass.
	counts PlanStats
	failed map[int64]string // job id -> description hash that failed validation
	// priority holds the vacancies the owner is looking at right now: their work is taken before
	// the configured order, once.
	priority map[int64]bool
	backoff  time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewFitWorker builds a worker. It does not start it.
func NewFitWorker(s *store.Store, cfg *config.Holder, fit FitRescorer) *FitWorker {
	return &FitWorker{
		store: s, cfg: cfg, fit: fit,
		Now:             time.Now,
		FailureCooldown: time.Hour,
		IdleWait:        30 * time.Second,
		MinBackoff:      15 * time.Second,
		MaxBackoff:      5 * time.Minute,
		nudge:           make(chan struct{}, 1),
		failed:          map[int64]string{},
		priority:        map[int64]bool{},
		status:          FitStatus{State: FitStateIdle, ModelAvailable: true},
	}
}

// Nudge wakes the worker so it looks for work now. It never blocks.
func (w *FitWorker) Nudge() {
	select {
	case w.nudge <- struct{}{}:
	default:
	}
}

// Prioritize asks for the work of one vacancy to be taken next, ahead of the configured order, and
// wakes the worker. It is for a vacancy the owner is looking at right after changing the profile; the
// rest of the work follows in its usual order.
func (w *FitWorker) Prioritize(jobID int64) {
	w.mu.Lock()
	w.priority[jobID] = true
	w.mu.Unlock()
	w.Nudge()
}

// withPriorityFirst moves the work of prioritized vacancies to the front of the plan, keeping the
// order inside each group.
func (w *FitWorker) withPriorityFirst(plan []WorkItem) []WorkItem {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.priority) == 0 {
		return plan
	}
	out := make([]WorkItem, 0, len(plan))
	for _, it := range plan {
		if w.priority[it.Source.JobID] {
			out = append(out, it)
		}
	}
	if len(out) == 0 {
		return plan
	}
	for _, it := range plan {
		if !w.priority[it.Source.JobID] {
			out = append(out, it)
		}
	}
	return out
}

// Start launches the loop and returns immediately.
func (w *FitWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		if w.Ready != nil {
			select {
			case <-ctx.Done():
				return
			case <-w.Ready:
			}
		}
		for {
			wait := w.Step(ctx)
			if ctx.Err() != nil {
				return
			}
			if wait <= 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-w.nudge:
			case <-time.After(wait):
			}
		}
	}()
}

// Stop cancels the loop and waits for it to end. A model call in flight is cancelled with it.
func (w *FitWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
		<-w.done
	}
}

// Status returns the worker's state with the counts of the last planning pass. It never touches
// the database: the owner's page polls it every few seconds, and a database read per poll made a
// stuck read stop every other request. Before the first planning the counts are zero.
func (w *FitWorker) Status() FitStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	st := w.status
	c := w.counts
	st.Usable, st.Extracted, st.NoRequirements = c.Usable, c.Extracted, c.NoRequirements
	st.WithRequires, st.Matched = c.WithRequires, c.Matched
	st.PendingExtract, st.PendingMatch, st.Cooling = c.PendingExtract, c.PendingMatch, c.Cooling
	return st
}

// setCounts caches the statistics of a planning pass for Status.
func (w *FitWorker) setCounts(c PlanStats) {
	w.mu.Lock()
	w.counts = c
	w.mu.Unlock()
}

// refreshCounts plans without working, for the states in which Step does no work (paused) but the
// counts must still follow the catalogue. A failed read leaves the previous counts in place.
func (w *FitWorker) refreshCounts() {
	prof, err := profile.Current(w.store)
	if err != nil {
		return
	}
	if prof == nil {
		w.setCounts(PlanStats{})
		return
	}
	rows, err := w.store.ModelWorkRows()
	if err != nil {
		return
	}
	w.mu.Lock()
	skip := copySkip(w.failed)
	w.mu.Unlock()
	_, stats := PlanWork(rows, w.planParams(prof.Revision, skip))
	w.setCounts(stats)
}

func copySkip(in map[int64]string) map[int64]string {
	out := make(map[int64]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (w *FitWorker) planParams(revision int, skip map[int64]string) PlanParams {
	return PlanParams{
		ExtractorVersion: vacancyai.ExtractorVersion(), MatcherVersion: vacancyai.MatcherVersion(w.cfg.Get()),
		ProfileRevision: revision, Skip: skip,
		Matching: w.cfg.Get().Runtime.EffectiveQuickMatch().ModelPasses.Matching,
		Order:    w.cfg.Get().Runtime.EffectiveQuickMatch().ModelPasses.EffectiveOrder(), Now: w.Now(),
	}
}

func (w *FitWorker) set(state, message string, modelAvailable bool) {
	w.mu.Lock()
	w.status.State, w.status.Message, w.status.ModelAvailable = state, message, modelAvailable
	w.mu.Unlock()
}

// Step does at most one unit of work (one vacancy, both passes if both are owed) and returns how
// long the loop should wait before the next step: zero when more work may be waiting.
func (w *FitWorker) Step(ctx context.Context) time.Duration {
	cfg := w.cfg.Get()
	mp := cfg.Runtime.EffectiveQuickMatch().ModelPasses
	if mp.Paused {
		w.refreshCounts()
		w.set(FitStatePaused, "Analysis is paused in settings", true)
		return w.IdleWait
	}
	prof, err := profile.Current(w.store)
	if err != nil {
		w.set(FitStateWaiting, "profile unreadable: "+err.Error(), true)
		return w.IdleWait
	}
	if prof == nil {
		w.setCounts(PlanStats{})
		w.set(FitStateNoProf, "No confirmed profile", true)
		return w.IdleWait
	}
	rows, err := w.store.ModelWorkRows()
	if err != nil {
		log.Printf("fit worker: %v", err)
		return w.IdleWait
	}
	w.mu.Lock()
	skip := copySkip(w.failed)
	w.mu.Unlock()
	plan, stats := PlanWork(rows, w.planParams(prof.Revision, skip))
	w.setCounts(stats)
	if len(plan) == 0 {
		w.mu.Lock()
		w.priority = map[int64]bool{}
		w.mu.Unlock()
		w.set(FitStateIdle, "", true)
		return w.IdleWait
	}
	plan = w.withPriorityFirst(plan)

	limitHit := false
	if mp.DailyLimit > 0 {
		n, err := w.store.CountExtractionsSince(startOfLocalDay(w.Now()))
		limitHit = err == nil && n >= mp.DailyLimit
	}
	var item *WorkItem
	for i := range plan {
		if plan[i].Kind == WorkExtract && limitHit {
			continue
		}
		item = &plan[i]
		break
	}
	if item == nil {
		w.set(FitStateLimit, fmt.Sprintf("Today's limit of %d vacancies is reached", mp.DailyLimit), true)
		return w.IdleWait
	}

	// Both passes are built up front: a configuration that forbids one on this provider must stop
	// the work before any model call is spent on it. With matching off there is no matching pass.
	extractor, err := vacancyai.NewExtractor(cfg, w.Infer)
	var matcher vacancyai.Matcher
	if err == nil && mp.Matching {
		matcher, err = vacancyai.NewMatcher(cfg, w.Infer)
	}
	if err != nil {
		w.set(FitStateBlocked, err.Error(), true)
		return w.IdleWait
	}

	if w.PassBackoff > 0 {
		extractor.Backoff, matcher.Backoff = w.PassBackoff, w.PassBackoff
	}
	w.set(FitStateWorking, "", true)
	w.mu.Lock()
	delete(w.priority, item.Source.JobID) // taken now; a failure is retried in the ordinary order
	w.mu.Unlock()
	if err := w.process(ctx, *item, prof, extractor, matcher, mp.Matching); err != nil {
		return w.handleFailure(*item, err)
	}
	w.mu.Lock()
	w.backoff = 0
	w.status.LastError = ""
	w.mu.Unlock()
	return 0
}

// handleFailure classifies a failed unit. A response the model gave that cannot be used (or a
// vacancy too large for the provider) is that vacancy's own problem: it is set aside until its
// text changes or the cooldown passes. Anything else (provider down, timeout, lock) says nothing
// about the vacancy, so the worker backs off and tries the same work again.
func (w *FitWorker) handleFailure(item WorkItem, err error) time.Duration {
	if errors.Is(err, context.Canceled) {
		return time.Second
	}
	w.mu.Lock()
	w.status.LastError = err.Error()
	w.mu.Unlock()
	hash := matchcontract.HashDescription(item.Source.Description)
	switch {
	case errors.Is(err, errInferenceBusy):
		return 2 * time.Second
	case errors.Is(err, vacancyai.ErrInvalidOutput), errors.Is(err, vacancyai.ErrContextOverflow), errors.Is(err, vacancyai.ErrNoText),
		errors.Is(err, store.ErrMatchSourceChanged):
		if errors.Is(err, store.ErrMatchSourceChanged) {
			return 0 // the text changed under us; the next plan picks it up afresh
		}
		w.mu.Lock()
		w.failed[item.Source.JobID] = hash
		w.mu.Unlock()
		log.Printf("fit worker: job %d set aside: %v", item.Source.JobID, err)
		time.AfterFunc(w.FailureCooldown, func() {
			w.mu.Lock()
			if w.failed[item.Source.JobID] == hash {
				delete(w.failed, item.Source.JobID)
			}
			w.mu.Unlock()
		})
		return 0
	}
	w.mu.Lock()
	if w.backoff == 0 {
		w.backoff = w.MinBackoff
	} else if w.backoff *= 2; w.backoff > w.MaxBackoff {
		w.backoff = w.MaxBackoff
	}
	wait := w.backoff
	w.mu.Unlock()
	w.set(FitStateWaiting, "The model is not reachable; trying again later", false)
	log.Printf("fit worker: provider problem, retrying in %s: %v", wait, err)
	return wait
}

var errInferenceBusy = modellock.ErrBusy

// withInferenceLock runs fn while holding the machine-wide model lock that the detailed
// assessment queue and application preparation also take, in every Sifty process.
func (w *FitWorker) withInferenceLock(fn func() error) error {
	lock, err := modellock.TryAcquire()
	if err != nil {
		return err
	}
	defer lock.Release()
	return fn()
}

func (w *FitWorker) process(ctx context.Context, item WorkItem, prof *profile.Profile, ex vacancyai.Extractor, mt vacancyai.Matcher, matchingOn bool) error {
	id := item.Source.JobID
	var extraction matchcontract.VacancyExtraction
	if item.Kind == WorkExtract {
		var res vacancyai.Extraction
		err := w.withInferenceLock(func() error {
			var err error
			res, err = ex.Extract(ctx, item.Source)
			return err
		})
		if err != nil {
			return err
		}
		if err := w.store.SaveVacancyExtraction(res.Record); err != nil {
			return err
		}
		extraction = res.Record
		w.rescore(ctx, id)
	} else {
		stored, err := w.store.GetVacancyExtraction(id, matchcontract.HashDescription(item.Source.Description), vacancyai.ExtractorVersion())
		if err != nil {
			return err
		}
		if stored == nil {
			return nil // superseded since planning; the next plan decides afresh
		}
		extraction = *stored
	}
	if len(extraction.Requirements) == 0 || !matchingOn {
		return nil // no requirements is final; with matching off the vacancy is finished after extraction
	}
	var matching vacancyai.Matching
	err := w.withInferenceLock(func() error {
		var err error
		matching, err = mt.Match(ctx, extraction, profile.MatchFacts(prof), prof.Revision)
		return err
	})
	if err != nil {
		return err
	}
	if err := w.store.SaveRequirementMatches(matching.Record); err != nil {
		return err
	}
	w.rescore(ctx, id)
	return nil
}

func (w *FitWorker) rescore(ctx context.Context, jobID int64) {
	if w.fit == nil {
		return
	}
	if err := w.fit.RescoreFit(ctx, jobID); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("fit worker: rescore job %d: %v", jobID, err)
	}
}
