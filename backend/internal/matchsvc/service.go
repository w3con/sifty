// Package matchsvc owns the Quick Match application effect: it projects the confirmed profile,
// maps configuration, extracts or reuses persisted facts locally, evaluates and stores results
// with compare-and-swap. It performs no network, model, queue or collection work. The pure
// extractor and evaluator remain replaceable packages behind the shared contract.
package matchsvc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchconfig"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/quickmatch"
	"github.com/w3con/sifty/internal/store"
)

// ErrNoProfile means no confirmed profile exists; Quick Match has nothing authoritative to score.
var ErrNoProfile = errors.New("quick match: no confirmed profile")

// ErrConfigChanged means a computed result no longer belongs to the active configuration.
var ErrConfigChanged = errors.New("quick match: configuration changed before result write")

// Summary reports one bounded recomputation.
type Summary struct {
	Processed int
	Scored    int
	Unscored  int
	Stale     int
}

// Service is the application adapter around persistence, configuration and the pure packages.
type Service struct {
	store *store.Store
	cfg   *config.Holder
}

// New builds the Quick Match effect from the existing store and configuration holder.
func New(st *store.Store, cfg *config.Holder) *Service {
	if cfg == nil {
		cfg = config.NewHolder(&config.Config{})
	}
	return &Service{store: st, cfg: cfg}
}

type settings struct {
	rules              matchcontract.Rules
	rulesIdentity      string
	declaredPolicy     matchcontract.SourcePolicy
	policyIdentity     string
	effectivePolicyVer string
	extractionPolicy   matchcontract.SourcePolicy
	// matchingOn is quick_match.model_passes.matching. It is part of the identity a stored
	// two-score result is keyed by, so switching it recomputes the results.
	matchingOn  bool
	fitIdentity string
}

func (s *Service) activeSettings(cfg *config.Config) (settings, error) {
	if cfg == nil {
		cfg = &config.Config{}
	}
	q := cfg.Runtime.EffectiveQuickMatch()
	rules, err := matchconfig.Rules(q)
	if err != nil {
		return settings{}, err
	}
	rulesIdentity, err := matchcontract.RulesIdentity(rules)
	if err != nil {
		return settings{}, err
	}
	policy := matchconfig.SourcePolicy(q)
	policyIdentity, err := matchcontract.SourcePolicyIdentity(policy)
	if err != nil {
		return settings{}, err
	}
	effective := policy.Version + "+" + policyIdentity
	extraction := policy
	extraction.Version = effective
	matchingOn := q.ModelPasses.Matching
	fitIdentity := rulesIdentity
	if !matchingOn {
		fitIdentity += "+matching-off"
	}
	return settings{
		matchingOn: matchingOn, fitIdentity: fitIdentity,
		rules: rules, rulesIdentity: rulesIdentity,
		policyIdentity:     policyIdentity,
		effectivePolicyVer: effective, extractionPolicy: extraction,
	}, nil
}

// Recalculate implements the profile save, collection and rules-change effect, which needs only
// success/failure. It recomputes the single-score results, rescores the two-score results from
// what is already stored (no model call), and then wakes the model-pass worker so that stale
// profile matches and newly collected vacancies are picked up. A profile save therefore re-runs
// only the matching pass; extraction is reused because the description did not change.
func (s *Service) Recalculate(ctx context.Context) error {
	if _, err := s.Recompute(ctx); err != nil {
		return err
	}
	if _, err := s.RescoreAllFit(ctx); err != nil && !errors.Is(err, ErrNoProfile) {
		return err
	}
	notifyModelWork()
	return nil
}

// Recompute recomputes every saved role under one guarded configuration snapshot. A profile or
// rule change reuses facts without capture or extraction when source production is unchanged. A
// source or source-policy content change re-captures and re-extracts the already saved input
// only. Cancellation stops before the next role; each committed result is an independent row.
func (s *Service) Recompute(ctx context.Context) (Summary, error) {
	var out Summary
	if s == nil || s.store == nil {
		return out, nil
	}
	err := s.cfg.WithSnapshot(func(cfg *config.Config) error {
		var err error
		out, err = s.recomputeAll(ctx, cfg)
		return err
	})
	return out, err
}

func (s *Service) recomputeAll(ctx context.Context, cfg *config.Config) (Summary, error) {
	var out Summary
	current, err := profile.Current(s.store)
	if err != nil {
		return out, fmt.Errorf("quick match: read profile: %w", err)
	}
	if current == nil {
		return out, ErrNoProfile
	}
	set, err := s.activeSettings(cfg)
	if err != nil {
		return out, err
	}
	ids, err := s.store.AllJobIDs()
	if err != nil {
		return out, err
	}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		status, err := s.recomputeJob(*current, set, id)
		if err != nil {
			return out, err
		}
		out.Processed++
		switch status {
		case matchcontract.StatusScored:
			out.Scored++
		case matchcontract.StatusUnscored:
			out.Unscored++
		default:
			out.Stale++
		}
	}
	return out, nil
}

// RecalculateJob recomputes one saved role under the active configuration.
func (s *Service) RecalculateJob(ctx context.Context, jobID int64) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.cfg.WithSnapshot(func(cfg *config.Config) error {
		current, err := profile.Current(s.store)
		if err != nil {
			return fmt.Errorf("quick match: read profile: %w", err)
		}
		if current == nil {
			return ErrNoProfile
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		set, err := s.activeSettings(cfg)
		if err != nil {
			return err
		}
		if _, err = s.recomputeJob(*current, set, jobID); err != nil {
			return err
		}
		if err = s.fitJob(*current, set, jobID); err != nil {
			return err
		}
		notifyModelWork()
		return nil
	})
}

// factsFor returns the deterministic facts of one saved role under the active source policy,
// reusing the persisted facts when the retained source and the policy are unchanged and
// re-capturing and re-extracting the already saved input otherwise.
func (s *Service) factsFor(jobID int64, set settings) (*matchcontract.VacancyFacts, error) {
	facts, err := s.store.GetMatchFacts(jobID)
	if err != nil || facts == nil {
		facts = nil
	}
	sourceCurrent := false
	if facts != nil && facts.SchemaVersion == matchcontract.SchemaVersion &&
		facts.ExtractorVersion == matchcontract.ExtractorVersion &&
		facts.SourcePolicyVersion == set.effectivePolicyVer {
		sourceCurrent, err = s.store.MatchSourceCurrent(jobID)
		if err != nil {
			return nil, err
		}
		// The retained input can be newer than the facts when a previous capture succeeded but
		// extraction or fact save failed. Reusing those older facts would fail the result CAS
		// forever, so require the facts' source hash to match the current retained input too.
		if sourceCurrent {
			input, ierr := s.store.GetMatchInput(jobID)
			if ierr != nil {
				return nil, ierr
			}
			if input == nil {
				sourceCurrent = false
			} else {
				inputHash, herr := matchcontract.HashRetainedSource(*input)
				if herr != nil {
					return nil, herr
				}
				sourceCurrent = facts.RawSourceHash == inputHash
			}
		}
	}
	if facts == nil || !sourceCurrent {
		src, err := s.store.CaptureMatchInput(jobID)
		if err != nil {
			return nil, err
		}
		extracted, err := jobextract.Extract(src, jobextract.Policy{Source: set.extractionPolicy})
		if err != nil {
			return nil, fmt.Errorf("quick match: extract job %d: %w", jobID, err)
		}
		if err := s.store.SaveMatchFacts(extracted); err != nil {
			return nil, fmt.Errorf("quick match: save facts for job %d: %w", jobID, err)
		}
		facts = &extracted
	}

	return facts, nil
}

func (s *Service) recomputeJob(current profile.Profile, set settings, jobID int64) (matchcontract.ResultStatus, error) {
	facts, err := s.factsFor(jobID, set)
	if err != nil {
		return "", err
	}

	snapshot := profile.QuickMatchSnapshot(&current)
	result, err := quickmatch.Evaluate(snapshot, *facts, set.rules)
	if err != nil {
		return "", fmt.Errorf("quick match: evaluate job %d: %w", jobID, err)
	}
	result.Provenance.RulesVersion = set.rules.RulesVersion
	result.Provenance.RulesIdentity = set.rulesIdentity
	result.Provenance.SourcePolicyVersion = set.effectivePolicyVer
	result.Provenance.SourcePolicyIdentity = set.policyIdentity
	if err := s.verifyActive(set, result); err != nil {
		return "", err
	}
	if err := s.store.SaveMatchResult(jobID, result); err != nil {
		return "", fmt.Errorf("quick match: save result for job %d: %w", jobID, err)
	}
	return result.Status, nil
}

// SaveComputedResult writes one already-computed result only when its provenance still matches
// the active configuration. It is the explicit active-config guard for any in-flight result.
func (s *Service) SaveComputedResult(jobID int64, result matchcontract.Result) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.cfg.WithSnapshot(func(cfg *config.Config) error {
		set, err := s.activeSettings(cfg)
		if err != nil {
			return err
		}
		if err := s.verifyActive(set, result); err != nil {
			return err
		}
		return s.store.SaveMatchResult(jobID, result)
	})
}

func (s *Service) verifyActive(set settings, result matchcontract.Result) error {
	p := result.Provenance
	if p.RulesIdentity != set.rulesIdentity ||
		p.RulesVersion != set.rules.RulesVersion ||
		p.SourcePolicyIdentity != set.policyIdentity ||
		p.SourcePolicyVersion != set.effectivePolicyVer {
		return ErrConfigChanged
	}
	return nil
}

// View returns the API envelope for one persisted result. It never mutates and never recomputes.
func (s *Service) View(jobID int64) (*matchcontract.View, error) {
	if s == nil || s.store == nil {
		return nil, nil
	}
	var view *matchcontract.View
	err := s.cfg.WithSnapshot(func(cfg *config.Config) error {
		set, err := s.activeSettings(cfg)
		if err != nil {
			return err
		}
		v, err := s.viewWithSettings(jobID, set)
		view = v
		return err
	})
	return view, err
}

func (s *Service) viewWithSettings(jobID int64, set settings) (*matchcontract.View, error) {
	stored, err := s.store.GetMatchResult(jobID)
	if err != nil || stored == nil {
		return nil, err
	}
	current, reasons, err := s.store.MatchResultCurrent(jobID, stored.Result, set.rulesIdentity, set.effectivePolicyVer, set.policyIdentity)
	if err != nil {
		return nil, err
	}
	view := &matchcontract.View{Result: stored.Result, Stale: !current, StaleReasons: reasons}
	if !current {
		view.Result.Status = matchcontract.StatusStale
	} else {
		facts, ferr := s.store.GetMatchFacts(jobID)
		if ferr != nil {
			return nil, ferr
		}
		if facts != nil && facts.FactsHash == stored.Result.Provenance.FactsHash && facts.RawSourceHash == stored.Result.Provenance.RawSourceHash {
			view.Facts = facts
		}
	}
	view.Summary = summaryOf(view.Result)
	return view, nil
}

// Decorate attaches the stored Quick Match view to API jobs and mirrors the compatibility score
// only when the stored result is current. It never exposes a legacy BM25 value as current and
// never calls a provider. A corrupt/incompatible result becomes an explicit job error instead of
// a silent unscored value; the stored row is not modified.
func (s *Service) Decorate(jobs []*model.Job) {
	if s == nil || s.store == nil || len(jobs) == 0 {
		return
	}
	err := s.cfg.WithSnapshot(func(cfg *config.Config) error {
		current, err := profile.Current(s.store)
		if err != nil {
			return err
		}
		if current == nil {
			for _, job := range jobs {
				clearQuickMatch(job)
			}
			return ErrNoProfile
		}
		set, err := s.activeSettings(cfg)
		if err != nil {
			return err
		}
		s.attachFit(jobs, current.Revision, set)
		for _, job := range jobs {
			view, verr := s.viewWithSettings(job.ID, set)
			if verr != nil {
				clearQuickMatch(job)
				job.QuickMatchError = verr.Error()
				continue
			}
			if view == nil {
				clearQuickMatch(job)
				continue
			}
			job.QuickMatch = view
			job.MatchProfileRevision = view.Result.Provenance.ProfileRevision
			job.MatchStale = view.Stale
			if view.Stale {
				job.MatchScore = 0
				continue
			}
			if view.Result.Score != nil {
				job.MatchScore = *view.Result.Score
			} else {
				job.MatchScore = 0
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrNoProfile) {
		for _, job := range jobs {
			if job.QuickMatchError == "" {
				job.QuickMatchError = err.Error()
			}
		}
	}
}

// Sort orders decorated jobs by raw score descending, coverage descending, then stable job id
// ascending. Unscored and stale results follow current scored results.
func Sort(jobs []*model.Job) {
	sort.SliceStable(jobs, func(i, j int) bool {
		a, b := jobs[i], jobs[j]
		aScored, aRaw := sortableScore(a)
		bScored, bRaw := sortableScore(b)
		if aScored != bScored {
			return aScored
		}
		if aScored && aRaw != bRaw {
			return aRaw > bRaw
		}
		aCov, bCov := coverageOf(a), coverageOf(b)
		if aCov != bCov {
			return aCov > bCov
		}
		return a.ID < b.ID
	})
}

func sortableScore(j *model.Job) (bool, float64) {
	if j.QuickMatch == nil || j.QuickMatch.Stale || j.QuickMatch.Summary.Status != matchcontract.StatusScored {
		return false, 0
	}
	if j.QuickMatch.Summary.RawScore != nil {
		return true, *j.QuickMatch.Summary.RawScore
	}
	if j.QuickMatch.Summary.Score != nil {
		return true, float64(*j.QuickMatch.Summary.Score) / 100
	}
	return false, 0
}

func summaryOf(r matchcontract.Result) matchcontract.Summary {
	return matchcontract.Summary{
		Status: r.Status, Score: r.Score, RawScore: r.RawScore, Coverage: r.Coverage.Weighted,
		Unknowns: len(r.Unknowns), Conflicts: len(r.Conflicts),
	}
}

func coverageOf(j *model.Job) float64 {
	if j.QuickMatch == nil {
		return 0
	}
	return j.QuickMatch.Summary.Coverage
}

func clearQuickMatch(job *model.Job) {
	job.QuickMatch = nil
	job.QuickMatchError = ""
	job.MatchScore = 0
	job.MatchProfileRevision = 0
	job.MatchStale = true
}

var modelWorkNotifier atomic.Pointer[func()]

// SetModelWorkNotifier registers the function that wakes the model-pass worker. It is a
// package-level hook because request handlers build a fresh Service per request; passing nil
// removes it.
func SetModelWorkNotifier(fn func()) {
	if fn == nil {
		modelWorkNotifier.Store(nil)
		return
	}
	modelWorkNotifier.Store(&fn)
}

var modelWorkPriority atomic.Pointer[func(int64)]

// SetModelWorkPriority registers the function that asks the model-pass worker to take one vacancy
// next. Like the notifier it is a package-level hook, because request handlers build a fresh
// Service per request; passing nil removes it.
func SetModelWorkPriority(fn func(jobID int64)) {
	if fn == nil {
		modelWorkPriority.Store(nil)
		return
	}
	modelWorkPriority.Store(&fn)
}

// PrioritizeModelWork asks for the model work of one vacancy to be taken next. It does nothing when
// no worker is registered.
func PrioritizeModelWork(jobID int64) {
	if fn := modelWorkPriority.Load(); fn != nil {
		(*fn)(jobID)
	}
}

func notifyModelWork() {
	if fn := modelWorkNotifier.Load(); fn != nil {
		(*fn)()
	}
}

// SortByFit orders decorated jobs for triage: Fits you descending, then You fit descending, then
// job id ascending. A vacancy whose You fit is still pending, or states no requirements, sorts as
// if it were the midpoint; one with no Fits you number (unavailable text, no profile goal to
// compare, not scored yet) follows every scored one.
func SortByFit(jobs []*model.Job) {
	key := func(j *model.Job) (hasFits bool, fits, you float64) {
		you = matchcontract.DefaultUnknownGoalUtility
		if j.Fit == nil || j.Fit.Stale {
			return false, 0, you
		}
		r := j.Fit.Result
		if r.FitsYou.Status == matchcontract.ScoreScored && r.FitsYou.RawScore != nil {
			hasFits, fits = true, *r.FitsYou.RawScore
		}
		if r.YouFit.Status == matchcontract.ScoreScored && r.YouFit.RawScore != nil {
			you = *r.YouFit.RawScore
		}
		return hasFits, fits, you
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		ah, af, ay := key(jobs[i])
		bh, bf, by := key(jobs[j])
		switch {
		case ah != bh:
			return ah
		case ah && af != bf:
			return af > bf
		case ay != by:
			return ay > by
		}
		return jobs[i].ID < jobs[j].ID
	})
}
