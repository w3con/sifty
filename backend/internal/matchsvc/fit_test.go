package matchsvc_test

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/matchsvc"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

func fitStore(t *testing.T) (*store.Store, *matchsvc.Service) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "fit.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if _, err := s.CreateProfile("local", `{"targetRoles":["Rust Engineer"],"skills":[{"key":"rust","name":"Rust"}]}`); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	return s, matchsvc.New(s, config.NewHolder(&config.Config{}))
}

func addJob(t *testing.T, s *store.Store, title, desc, status string) int64 {
	t.Helper()
	id, err := s.Upsert(model.JobInput{
		Title: title, Company: "Acme", OriginalURL: "https://example.com/" + title,
		Description: desc, DescriptionStatus: status, Source: "Wellfound", Location: "Berlin, Germany",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	return id
}

func TestRecalculateRescoresBothScoresAndWakesTheWorker(t *testing.T) {
	s, svc := fitStore(t)
	var woken int32
	matchsvc.SetModelWorkNotifier(func() { atomic.AddInt32(&woken, 1) })
	t.Cleanup(func() { matchsvc.SetModelWorkNotifier(nil) })

	usable := addJob(t, s, "Rust Engineer", "You know Rust. You will build services.", "available structured description")
	noText := addJob(t, s, "Rust Developer", "Rust, Go", "skills metadata only; full description pending")
	if err := svc.Recalculate(context.Background()); err != nil {
		t.Fatalf("Recalculate: %v", err)
	}
	if atomic.LoadInt32(&woken) != 1 {
		t.Fatalf("the worker was woken %d times, want once", woken)
	}
	got, err := s.GetFitResult(usable)
	if err != nil || got == nil {
		t.Fatalf("no stored fit for a usable vacancy: %+v err=%v", got, err)
	}
	if got.Result.FitsYou.Status != matchcontract.ScoreScored || got.Result.YouFit.Status != matchcontract.ScorePending {
		t.Errorf("usable vacancy: fits=%s you=%s, want scored and pending (requirements not extracted yet)", got.Result.FitsYou.Status, got.Result.YouFit.Status)
	}
	if nt, _ := s.GetFitResult(noText); nt == nil || nt.Result.Source != matchcontract.SourceUnavailable {
		t.Errorf("a vacancy with no text must be stored as unavailable: %+v", nt)
	}
}

func TestDecorateCarriesTheFitAndMarksAStaleOneStale(t *testing.T) {
	s, svc := fitStore(t)
	id := addJob(t, s, "Rust Engineer", "You know Rust. You will build services.", "available structured description")
	if err := svc.Recalculate(context.Background()); err != nil {
		t.Fatalf("Recalculate: %v", err)
	}
	jobs := []*model.Job{mustGet(t, s, id)}
	svc.Decorate(jobs)
	if jobs[0].Fit == nil || jobs[0].Fit.Stale {
		t.Fatalf("fit = %+v, want a current view", jobs[0].Fit)
	}
	// A profile change makes the stored result stale until the rescoring runs.
	if _, err := s.SaveProfile("local", 1, `{"targetRoles":["Rust Engineer"],"skills":[{"key":"rust","name":"Rust"},{"key":"go","name":"Go"}]}`, "update", "manual"); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	jobs = []*model.Job{mustGet(t, s, id)}
	svc.Decorate(jobs)
	if jobs[0].Fit == nil || !jobs[0].Fit.Stale || jobs[0].Fit.StaleReasons[0] != "profile_changed" {
		t.Fatalf("fit after the profile save = %+v, want stale for profile_changed", jobs[0].Fit)
	}
	if err := svc.Recalculate(context.Background()); err != nil {
		t.Fatal(err)
	}
	jobs = []*model.Job{mustGet(t, s, id)}
	svc.Decorate(jobs)
	if jobs[0].Fit == nil || jobs[0].Fit.Stale {
		t.Fatalf("fit after rescoring = %+v, want current", jobs[0].Fit)
	}
}

func mustGet(t *testing.T, s *store.Store, id int64) *model.Job {
	t.Helper()
	j, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestRescoreUnscoredFitOnlyScoresNewVacancies(t *testing.T) {
	s, svc := fitStore(t)

	// Add two vacancies: one that already has a fit result, one that does not.
	scored := addJob(t, s, "Rust Engineer", "You know Rust. You will build services.", "available structured description")
	unscored := addJob(t, s, "Go Developer", "You know Go. You will build backends.", "available structured description")

	// Score only the first vacancy manually so it has a stored result.
	if err := svc.RescoreFit(context.Background(), scored); err != nil {
		t.Fatalf("RescoreFit: %v", err)
	}
	// Confirm the second vacancy has no result.
	if got, err := s.GetFitResult(unscored); err != nil || got != nil {
		t.Fatalf("want no fit result for unscored vacancy before RescoreUnscoredFit: got=%+v err=%v", got, err)
	}

	// RescoreUnscoredFit must score only the unscored vacancy.
	sum, err := svc.RescoreUnscoredFit(context.Background())
	if err != nil {
		t.Fatalf("RescoreUnscoredFit: %v", err)
	}
	if sum.Processed != 1 {
		t.Errorf("Processed=%d, want 1 (only the vacancy without a stored result)", sum.Processed)
	}
	if sum.Failed != 0 {
		t.Errorf("Failed=%d, want 0", sum.Failed)
	}

	// The unscored vacancy must now have a result.
	got, err := s.GetFitResult(unscored)
	if err != nil || got == nil {
		t.Fatalf("want stored fit for unscored vacancy after RescoreUnscoredFit: got=%+v err=%v", got, err)
	}

	// The already-scored vacancy must still have its result (unchanged).
	before, err := s.GetFitResult(scored)
	if err != nil || before == nil {
		t.Fatalf("scored vacancy lost its fit result: %v", err)
	}
}

func TestSortByFitFollowsTheTriagePolicy(t *testing.T) {
	score := func(fits, you *float64, youStatus matchcontract.ScoreStatus) *matchcontract.FitView {
		v := &matchcontract.FitView{}
		if fits != nil {
			v.Result.FitsYou = matchcontract.FitsYou{Status: matchcontract.ScoreScored, RawScore: fits}
		} else {
			v.Result.FitsYou = matchcontract.FitsYou{Status: matchcontract.ScoreUnavailable}
		}
		v.Result.YouFit = matchcontract.YouFit{Status: youStatus, RawScore: you}
		return v
	}
	f := func(x float64) *float64 { return &x }
	jobs := []*model.Job{
		{ID: 1, Fit: score(f(0.60), nil, matchcontract.ScorePending)},   // pending You fit counts as 0.5
		{ID: 2, Fit: score(f(0.80), f(0.2), matchcontract.ScoreScored)}, // best Fits you
		{ID: 3, Fit: score(f(0.60), f(0.9), matchcontract.ScoreScored)}, // same Fits you as 1, better You fit
		{ID: 4, Fit: score(nil, nil, matchcontract.ScoreUnavailable)},   // no numbers: last
		{ID: 5}, // never scored: last, after 4 by id
		{ID: 6, Fit: score(f(0.60), f(0.5), matchcontract.ScoreScored)}, // ties with the pending one: id decides
	}
	matchsvc.SortByFit(jobs)
	var got []int64
	for _, j := range jobs {
		got = append(got, j.ID)
	}
	want := []int64{2, 3, 1, 6, 4, 5}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}
