package matchsvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/quickmatch"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)

// FitSummary reports one bounded rescoring of every vacancy.
type FitSummary struct {
	Processed int
	Failed    int
}

// RescoreFit recomputes and stores the two-score result of one vacancy from what is already
// persisted: the deterministic facts, the extraction and the matches. It makes no model call, so
// it is cheap and is run after every model pass and after every profile or rules change.
func (s *Service) RescoreFit(ctx context.Context, jobID int64) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.cfg.WithSnapshot(func(cfg *config.Config) error {
		current, err := profile.Current(s.store)
		if err != nil {
			return fmt.Errorf("fit: read profile: %w", err)
		}
		if current == nil {
			return ErrNoProfile
		}
		set, err := s.activeSettings(cfg)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return s.fitJob(*current, set, jobID)
	})
}

// RescoreAllFit recomputes the two-score result of every saved vacancy. One vacancy's failure is
// counted and does not stop the others; cancellation stops before the next vacancy.
func (s *Service) RescoreAllFit(ctx context.Context) (FitSummary, error) {
	var out FitSummary
	if s == nil || s.store == nil {
		return out, nil
	}
	err := s.cfg.WithSnapshot(func(cfg *config.Config) error {
		current, err := profile.Current(s.store)
		if err != nil {
			return fmt.Errorf("fit: read profile: %w", err)
		}
		if current == nil {
			return ErrNoProfile
		}
		set, err := s.activeSettings(cfg)
		if err != nil {
			return err
		}
		ids, err := s.store.AllJobIDs()
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := ctx.Err(); err != nil {
				return err
			}
			out.Processed++
			if err := s.fitJob(*current, set, id); err != nil {
				out.Failed++
			}
		}
		return nil
	})
	return out, err
}

func (s *Service) fitJob(current profile.Profile, set settings, jobID int64) error {
	job, err := s.store.Get(jobID)
	if err != nil {
		return fmt.Errorf("fit: read job %d: %w", jobID, err)
	}
	src := matchcontract.RetainedSource{
		JobID: jobID, Title: job.Title, Company: job.Company, Tags: job.Tags,
		Description: job.Description, DescriptionStatus: job.DescriptionStatus,
	}
	descHash := matchcontract.HashDescription(job.Description)
	in := matchcontract.FitInput{
		Profile:         profile.QuickMatchSnapshot(&current),
		Source:          jobextract.SourceStateOf(src),
		DescriptionHash: descHash,
		Rules:           set.rules,
		MatchingOff:     !set.matchingOn,
	}
	if in.Source == matchcontract.SourceUsable {
		facts, err := s.factsFor(jobID, set)
		if err != nil {
			return err
		}
		in.Facts = facts
		ext, err := s.store.GetVacancyExtraction(jobID, descHash, vacancyai.ExtractorVersion())
		if err != nil {
			return err
		}
		if ext != nil {
			// The conditions the model read fill only what the ordinary parse did not find; where
			// they disagree the deterministic fact stands and the conflict is kept.
			merged, conflicts, err := jobextract.MergeModelConditions(*facts, ext.Conditions, set.rules.RestrictionPhrases)
			if err != nil {
				return err
			}
			in.Facts, in.ConditionConflicts, in.Extraction = &merged, conflicts, ext
			if len(ext.Requirements) > 0 {
				hash, err := ext.RequirementsHash()
				if err != nil {
					return err
				}
				if set.matchingOn {
					in.Matches, err = s.store.GetRequirementMatches(jobID, hash, current.Revision, vacancyai.MatcherVersion(s.cfg.Get()))
					if err != nil {
						return err
					}
				} else {
					// The model's matching is off: only what code decides is shown, computed
					// here from the current profile and stored nowhere.
					m, err := vacancyai.NewCodeMatcher(s.cfg.Get()).Match(context.Background(), *ext, profile.MatchFacts(&current), current.Revision)
					if err != nil {
						return err
					}
					in.Matches = &m.Record
				}
			}
		}
	}
	result, err := quickmatch.EvaluateFit(in)
	if err != nil {
		return fmt.Errorf("fit: evaluate job %d: %w", jobID, err)
	}
	if err := s.store.SaveFitResult(jobID, result, store.FitKey{RulesIdentity: set.fitIdentity, DescriptionHash: descHash}); err != nil {
		return err
	}
	return nil
}

// fitView projects one stored result and says whether it still belongs to the current inputs.
func (s *Service) fitView(job *model.Job, profileRevision int, set settings) (*matchcontract.FitView, error) {
	stored, err := s.store.GetFitResult(job.ID)
	if err != nil || stored == nil {
		return nil, err
	}
	view := &matchcontract.FitView{Result: stored.Result}
	if stored.ProfileRevision != profileRevision {
		view.StaleReasons = append(view.StaleReasons, "profile_changed")
	}
	if stored.Key.RulesIdentity != set.fitIdentity {
		view.StaleReasons = append(view.StaleReasons, "rules_changed")
	}
	if stored.Key.DescriptionHash != matchcontract.HashDescription(job.Description) {
		view.StaleReasons = append(view.StaleReasons, "text_changed")
	}
	view.Stale = len(view.StaleReasons) > 0
	return view, nil
}

// attachFit sets Job.Fit for every job. It reads only; a failure to read one job leaves its Fit
// nil and is reported through the job's quickMatchError so the list still renders.
func (s *Service) attachFit(jobs []*model.Job, profileRevision int, set settings) {
	for _, job := range jobs {
		view, err := s.fitView(job, profileRevision, set)
		if err != nil {
			job.Fit = nil
			if job.QuickMatchError == "" {
				job.QuickMatchError = err.Error()
			}
			continue
		}
		job.Fit = view
	}
}

// ErrFitUnavailable is returned by FitView when no confirmed profile exists.
var ErrFitUnavailable = errors.New("fit: no confirmed profile")

// FitOf returns the stored two-score view of one vacancy, or nil when it has none.
func (s *Service) FitOf(jobID int64) (*matchcontract.FitView, error) {
	if s == nil || s.store == nil {
		return nil, nil
	}
	var view *matchcontract.FitView
	err := s.cfg.WithSnapshot(func(cfg *config.Config) error {
		current, err := profile.Current(s.store)
		if err != nil {
			return err
		}
		if current == nil {
			return ErrFitUnavailable
		}
		set, err := s.activeSettings(cfg)
		if err != nil {
			return err
		}
		job, err := s.store.Get(jobID)
		if err != nil {
			return err
		}
		view, err = s.fitView(job, current.Revision, set)
		return err
	})
	return view, err
}
