package background

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/matchsvc"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

// fitFixture is a temp database with a profile, a rules configuration that routes extraction to
// a local provider, a real matchsvc.Service as the rescorer and a model stub that counts calls.
type fitFixture struct {
	t      *testing.T
	s      *store.Store
	holder *config.Holder
	svc    *matchsvc.Service
	w      *FitWorker

	mu          sync.Mutex
	extractions []string // job titles the extraction pass saw
	matchings   int
	failWith    error
	answerBad   bool
}

const profileJSON = `{"targetRoles":["Rust Engineer"],"headline":"Rust engineer","skills":[{"key":"rust","name":"Rust","level":"expert"}]}`

func localProvider() config.Provider {
	return config.Provider{Type: "ollama", Model: "m", Endpoint: "http://127.0.0.1:11434", TimeoutMS: 1000, ContextChars: 32000}
}

func newFit(t *testing.T, mutate func(*config.Config)) *fitFixture {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "fit.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if _, err := s.CreateProfile("local", profileJSON); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	cfg := &config.Config{}
	cfg.Providers.Stages.Extraction = "local"
	cfg.Providers.Providers = map[string]config.Provider{"local": localProvider()}
	cfg.Runtime.QuickMatch = config.DefaultQuickMatch()
	cfg.Runtime.QuickMatch.ModelPasses.Matching = true // the tests below describe the matching pass; the off state has its own
	if mutate != nil {
		mutate(cfg)
	}
	f := &fitFixture{t: t, s: s, holder: config.NewHolder(cfg)}
	f.svc = matchsvc.New(s, f.holder)
	f.w = NewFitWorker(s, f.holder, f.svc)
	f.w.IdleWait = time.Second
	f.w.MinBackoff, f.w.MaxBackoff = 10*time.Millisecond, 40*time.Millisecond
	f.w.FailureCooldown = time.Hour
	f.w.PassBackoff = time.Millisecond
	f.w.Infer = f.infer
	return f
}

// infer stands in for the provider. The extraction answer quotes the vacancy's own first
// sentence; the matching answer judges every requirement a match on profile fact P1.
func (f *fitFixture) infer(_ context.Context, _ assess.Provider, request string, _ any, out any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return f.failWith
	}
	raw := out.(*json.RawMessage)
	if strings.Contains(request, "REQUIREMENTS (DATA)") {
		f.matchings++
		var verdicts []string
		for i := 0; strings.Contains(request, fmt.Sprintf("\n%d. [", i)); i++ {
			verdicts = append(verdicts, fmt.Sprintf(`{"index":%d,"verdict":"match","evidenceId":"P1","reason":"fits"}`, i))
		}
		*raw = json.RawMessage(`{"verdicts":[` + strings.Join(verdicts, ",") + `]}`)
		return nil
	}
	_, desc, _ := strings.Cut(request, "DESCRIPTION:\n")
	f.extractions = append(f.extractions, strings.TrimSpace(desc))
	switch {
	case f.answerBad:
		*raw = json.RawMessage(`{"requirements":[{"text":"Invented","kind":"skill","level":"required","quote":"not in the text at all","alternatives":[]}],"conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`)
	case strings.Contains(desc, "WELCOME"):
		*raw = json.RawMessage(`{"requirements":[],"conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`)
	default:
		*raw = json.RawMessage(`{"requirements":[{"text":"Rust","kind":"skill","level":"required","quote":"You know Rust.","alternatives":[]}],"conditions":{"compensation":null,"workMode":null,"engagement":null,"location":null,"timezone":null,"hours":null}}`)
	}
	return nil
}

func (f *fitFixture) job(title, description, status string) int64 {
	f.t.Helper()
	id, err := f.s.Upsert(model.JobInput{
		Title: title, Company: "Acme", OriginalURL: "https://example.com/" + strings.ReplaceAll(title, " ", "-"),
		Description: description, DescriptionStatus: status, Source: "Wellfound", Location: "Berlin, Germany",
	}, false)
	if err != nil {
		f.t.Fatalf("Upsert %s: %v", title, err)
	}
	return id
}

// drain runs steps until the worker has nothing left, failing if it never settles.
func (f *fitFixture) drain() {
	f.t.Helper()
	for i := 0; i < 200; i++ {
		if wait := f.w.Step(context.Background()); wait > 0 {
			return
		}
	}
	f.t.Fatal("the worker did not settle")
}

func (f *fitFixture) counts() (extractions, matchings int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.extractions), f.matchings
}

func tableHash(t *testing.T, s *store.Store, query string) string {
	t.Helper()
	rows, err := s.DB.Query(query)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	h := sha256.New()
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(h, vals...)
	}
	return hex.EncodeToString(h.Sum(nil))
}

const (
	ruleDesc = "You know Rust. You will build services."
	otherDoc = "You know Rust. You will build Rust tools."
)

func TestWorkerScoresBothPassesEndToEndAndLeavesTheAssessmentQueueAlone(t *testing.T) {
	f := newFit(t, nil)
	a := f.job("Rust Engineer", ruleDesc, "available structured description")
	b := f.job("Rust Developer", otherDoc, "available structured description")
	none := f.job("Greeter", "WELCOME to our page, a nice greeting for every visitor.", "available structured description")
	meta := f.job("Metadata Only", "Rust, Go", "skills metadata only; full description pending")
	// An existing assessment queue row and an assessment, which the worker must never touch.
	if _, err := f.s.DB.Exec(`INSERT INTO assessment_queue(job_id, fingerprint, provider, status) VALUES(?, 'fp', 'luna', 'pending')`, a); err != nil {
		t.Fatalf("queue row: %v", err)
	}
	queueBefore := tableHash(t, f.s, "SELECT * FROM assessment_queue ORDER BY job_id")
	assessBefore := tableHash(t, f.s, "SELECT id, assessment, assessment_error FROM jobs ORDER BY id")
	profileBefore := tableHash(t, f.s, "SELECT * FROM profile_revisions ORDER BY 1, 2")

	if _, err := f.svc.RescoreAllFit(context.Background()); err != nil {
		t.Fatalf("initial rescore: %v", err)
	}
	for _, id := range []int64{a, b, none} {
		fit, err := f.s.GetFitResult(id)
		if err != nil || fit == nil || fit.Result.YouFit.Status != matchcontract.ScorePending {
			t.Fatalf("job %d before the passes: %+v err=%v; You fit must be pending", id, fit, err)
		}
	}
	if fit, _ := f.s.GetFitResult(meta); fit == nil || fit.Result.Source != matchcontract.SourceUnavailable {
		t.Fatalf("a text-less vacancy must be stored as unavailable: %+v", fit)
	}

	f.drain()

	if n, m := f.counts(); n != 3 || m != 2 {
		t.Fatalf("model calls: %d extractions, %d matchings; want 3 and 2 (metadata-only is never sent, no-requirements is not matched)", n, m)
	}
	for _, id := range []int64{a, b} {
		fit, _ := f.s.GetFitResult(id)
		if fit.Result.YouFit.Status != matchcontract.ScoreScored || fit.Result.YouFit.Counts.Match != 1 {
			t.Errorf("job %d You fit = %+v, want scored with one match", id, fit.Result.YouFit)
		}
	}
	if fit, _ := f.s.GetFitResult(none); fit.Result.YouFit.Status != matchcontract.ScoreNoRequirements {
		t.Errorf("a vacancy without requirements = %+v, want no_requirements", fit.Result.YouFit)
	}
	if fit, _ := f.s.GetFitResult(meta); fit.Result.YouFit.Status != matchcontract.ScoreUnavailable {
		t.Errorf("text-less vacancy You fit = %+v", fit.Result.YouFit)
	}
	if got := tableHash(t, f.s, "SELECT * FROM assessment_queue ORDER BY job_id"); got != queueBefore {
		t.Error("the detailed assessment queue changed")
	}
	if got := tableHash(t, f.s, "SELECT id, assessment, assessment_error FROM jobs ORDER BY id"); got != assessBefore {
		t.Error("assessments changed")
	}
	if got := tableHash(t, f.s, "SELECT * FROM profile_revisions ORDER BY 1, 2"); got != profileBefore {
		t.Error("the profile changed")
	}
	if st := f.w.Status(); st.State != FitStateIdle || st.PendingExtract != 0 || st.PendingMatch != 0 || st.Usable != 3 || st.Matched != 2 || st.NoRequirements != 1 {
		t.Errorf("status = %+v", st)
	}
}

// With the matching pass switched off (the default) the worker extracts and never matches: the
// model is asked for requirements only, You fit has the status off, and the requirements are listed.
func TestMatchingOffMakesNoMatchingCallAndShowsNoYouFit(t *testing.T) {
	f := newFit(t, func(c *config.Config) { c.Runtime.QuickMatch.ModelPasses.Matching = false })
	a := f.job("Rust Engineer", ruleDesc, "available structured description")
	f.job("Rust Developer", otherDoc, "available structured description")
	f.drain()
	if n, m := f.counts(); n != 2 || m != 0 {
		t.Fatalf("model calls: %d extractions, %d matchings; want 2 and 0", n, m)
	}
	if st := f.w.Status(); st.State != FitStateIdle || st.PendingExtract != 0 || st.PendingMatch != 0 {
		t.Errorf("status = %+v; extraction is done and nothing is owed for matching", st)
	}
	fit, err := f.s.GetFitResult(a)
	if err != nil || fit == nil {
		t.Fatalf("fit result: %+v err=%v", fit, err)
	}
	if !fit.Result.MatchingOff || fit.Result.YouFit.Status != matchcontract.ScoreOff || fit.Result.YouFit.Score != nil {
		t.Errorf("You fit = %+v, want status off and no number", fit.Result.YouFit)
	}
	lines := 0
	for _, list := range [][]matchcontract.FitLine{fit.Result.Lists.Matches, fit.Result.Lists.Mismatches, fit.Result.Lists.Missing} {
		for _, l := range list {
			if l.Origin == matchcontract.OriginRequirement {
				lines++
			}
		}
	}
	if lines == 0 {
		t.Error("the requirements of the vacancy must still be listed")
	}
}

func TestProfileChangeRunsOnlyTheMatchingPass(t *testing.T) {
	f := newFit(t, nil)
	f.job("Rust Engineer", ruleDesc, "available structured description")
	f.job("Rust Developer", otherDoc, "available structured description")
	f.drain()
	ext, mat := f.counts()
	if ext != 2 || mat != 2 {
		t.Fatalf("first run: %d extractions, %d matchings", ext, mat)
	}
	if _, err := f.s.SaveProfile("local", 1, strings.Replace(profileJSON, `"level":"expert"`, `"level":"proficient"`, 1), "update", "manual"); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	f.w.Step(context.Background()) // Status serves the counts of the last planning pass; plan again
	if st := f.w.Status(); st.PendingMatch != 2 || st.PendingExtract != 0 {
		t.Fatalf("after the profile save: %+v; want two matchings owed and no extraction", st)
	}
	f.drain()
	ext2, mat2 := f.counts()
	if ext2 != ext || mat2 != mat+2 {
		t.Fatalf("after the profile change: %d extractions (was %d), %d matchings (was %d); extraction must be reused", ext2, ext, mat2, mat)
	}
	var rev int
	if err := f.s.DB.QueryRow(`SELECT profile_revision FROM match_fit_results ORDER BY job_id LIMIT 1`).Scan(&rev); err != nil || rev != 2 {
		t.Fatalf("stored fit is for profile revision %d err=%v, want 2", rev, err)
	}
}

func TestChangedDescriptionIsExtractedAgainButANewSightingIsNot(t *testing.T) {
	f := newFit(t, nil)
	a := f.job("Rust Engineer", ruleDesc, "available structured description")
	f.job("Rust Developer", otherDoc, "available structured description")
	f.drain()
	if n, _ := f.counts(); n != 2 {
		t.Fatalf("extractions = %d", n)
	}
	// The same text seen again on another board with other tags: the model is not asked again.
	if _, err := f.s.Upsert(model.JobInput{
		Title: "Rust Engineer", Company: "Acme", OriginalURL: "https://example.com/Rust-Engineer",
		Description: ruleDesc, DescriptionStatus: "available structured description", Source: "Remotive", Location: "Paris", Tags: []string{"rust"},
	}, false); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	f.drain()
	if n, _ := f.counts(); n != 2 {
		t.Fatalf("a new sighting of the same text triggered %d extractions in total, want 2", n)
	}
	if _, err := f.s.DB.Exec(`UPDATE jobs SET description=? WHERE id=?`, "You know Rust. You also know Go.", a); err != nil {
		t.Fatal(err)
	}
	f.drain()
	if n, _ := f.counts(); n != 3 {
		t.Fatalf("extractions = %d, want 3: only the changed vacancy is read again", n)
	}
}

// The worker takes vacancies in the order of the Inbox toggle, reads the configured order before
// every vacancy, and gives newly collected vacancies no priority of their own.
func TestModelWorkFollowsTheConfiguredOrder(t *testing.T) {
	setup := func(order string) (*fitFixture, [3]int64) {
		f := newFit(t, func(c *config.Config) { c.Runtime.QuickMatch.ModelPasses.Order = order })
		a := f.job("Rust One", "You know Rust. Text one.", "available structured description")
		b := f.job("Rust Two", "You know Rust. Text two.", "available structured description")
		c := f.job("Rust Three", "You know Rust. Text three.", "available structured description")
		if _, err := f.svc.RescoreAllFit(context.Background()); err != nil {
			t.Fatal(err)
		}
		// A: high Fits you, old; B: low Fits you, newest; C: middle Fits you, no date and just collected.
		for _, row := range []struct {
			id        int64
			fits      float64
			published string
		}{{a, 0.9, "2026-01-01"}, {b, 0.1, "2026-10-01"}, {c, 0.5, ""}} {
			if _, err := f.s.DB.Exec(`UPDATE match_fit_results SET fits_raw=? WHERE job_id=?`, row.fits, row.id); err != nil {
				t.Fatal(err)
			}
			if _, err := f.s.DB.Exec(`UPDATE jobs SET published_at=?, first_seen=? WHERE id=?`, row.published, storeTime(time.Now()), row.id); err != nil {
				t.Fatal(err)
			}
		}
		return f, [3]int64{a, b, c}
	}
	sequence := func(f *fitFixture) string {
		var out []string
		for range 3 {
			f.w.Step(context.Background())
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, d := range f.extractions {
			switch {
			case strings.Contains(d, "Text one"):
				out = append(out, "A")
			case strings.Contains(d, "Text two"):
				out = append(out, "B")
			case strings.Contains(d, "Text three"):
				out = append(out, "C")
			}
		}
		return strings.Join(out, "")
	}
	t.Run("best fit", func(t *testing.T) {
		f, _ := setup(config.OrderFit)
		if got := sequence(f); got != "ACB" {
			t.Fatalf("order = %s, want ACB (Fits you 0.9, 0.5, 0.1)", got)
		}
	})
	t.Run("newest", func(t *testing.T) {
		f, _ := setup(config.OrderNewest)
		if got := sequence(f); got != "BAC" {
			t.Fatalf("order = %s, want BAC (published 2026-10-01, 2026-01-01, no date)", got)
		}
	})
	t.Run("a switch takes effect at the next vacancy", func(t *testing.T) {
		f, _ := setup(config.OrderFit)
		f.w.Step(context.Background()) // A, the best fit
		f.holder.Get().Runtime.QuickMatch.ModelPasses.Order = config.OrderNewest
		f.w.Step(context.Background()) // now B, the newest
		f.w.Step(context.Background())
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(f.extractions) != 3 || !strings.Contains(f.extractions[0], "Text one") || !strings.Contains(f.extractions[1], "Text two") {
			t.Fatalf("extractions = %q, want text one, text two, text three", f.extractions)
		}
	})
}

// A vacancy the owner is looking at, after adding a skill from it, must be matched first; the
// others follow in the configured order, and the priority is used once.
func TestPrioritizedVacancyIsTakenFirstAndOnce(t *testing.T) {
	f := newFit(t, func(c *config.Config) { c.Runtime.QuickMatch.ModelPasses.Order = config.OrderFit })
	a := f.job("Rust One", "You know Rust. Text one.", "available structured description")
	b := f.job("Rust Two", "You know Rust. Text two.", "available structured description")
	c := f.job("Rust Three", "You know Rust. Text three.", "available structured description")
	if _, err := f.svc.RescoreAllFit(context.Background()); err != nil {
		t.Fatal(err)
	}
	for id, fits := range map[int64]float64{a: 0.9, b: 0.5, c: 0.1} {
		if _, err := f.s.DB.Exec(`UPDATE match_fit_results SET fits_raw=? WHERE job_id=?`, fits, id); err != nil {
			t.Fatal(err)
		}
	}
	f.w.Prioritize(c) // the lowest fit, but the one on the owner's screen
	for range 3 {
		f.w.Step(context.Background())
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var order []string
	for _, d := range f.extractions {
		switch {
		case strings.Contains(d, "Text one"):
			order = append(order, "A")
		case strings.Contains(d, "Text two"):
			order = append(order, "B")
		case strings.Contains(d, "Text three"):
			order = append(order, "C")
		}
	}
	if got := strings.Join(order, ""); got != "CAB" {
		t.Fatalf("order = %s, want CAB: the prioritized vacancy first, then Fits you 0.9, 0.5", got)
	}
}

func TestPlanWorkOrder(t *testing.T) {
	hashOf := func(s string) string { return matchcontract.HashDescription(s) }
	fits := func(v float64) *float64 { return &v }
	mk := func(id int64, published string, fit *float64) store.ModelWorkRow {
		return store.ModelWorkRow{
			Source:      matchcontract.RetainedSource{JobID: id, Title: "t", Description: fmt.Sprintf("text %d with enough words to be a description", id), DescriptionStatus: "available"},
			PublishedAt: published, FitsRaw: fit,
		}
	}
	rows := []store.ModelWorkRow{
		mk(1, "2026-03-01", fits(0.9)), // high fit
		mk(2, "2026-04-01", fits(0.2)), // low fit, cooling
		mk(3, "", nil),                 // no fit yet = 0.5, no date
		mk(4, "2026-10-01", fits(0.1)),
		mk(5, "2026-11-01", fits(0.1)), // same fit as 4, newer
	}
	// Job 6: extraction current, matches for an older profile revision.
	six := mk(6, "2026-02-01", fits(0.3))
	six.HasExtraction, six.ExtractionSource, six.ExtractorVersion, six.RequirementCount = true, hashOf(six.Source.Description), "x/1", 3
	six.HasMatches, six.MatcherVersion, six.ProfileRevision = true, "m/1", 1
	// Job 7: fully current; job 8: extraction current with no requirements.
	seven := mk(7, "2026-01-01", fits(0.9))
	seven.HasExtraction, seven.ExtractionSource, seven.ExtractorVersion, seven.RequirementCount = true, hashOf(seven.Source.Description), "x/1", 3
	seven.HasMatches, seven.MatcherVersion, seven.ProfileRevision = true, "m/1", 2
	eight := mk(8, "2026-01-01", fits(0.9))
	eight.HasExtraction, eight.ExtractionSource, eight.ExtractorVersion = true, hashOf(eight.Source.Description), "x/1"
	rows = append(rows, six, seven, eight)

	now := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	for _, c := range []struct{ name, order, want string }{
		{"best fit", config.OrderFit, "1e 3e 6m 4e 5e"},  // Fits you descending then id; one list, re-matching included
		{"newest", config.OrderNewest, "5e 4e 1e 6m 3e"}, // Inbox Newest key; the undated vacancy last
		{"unset reads as best fit", "", "1e 3e 6m 4e 5e"},
	} {
		plan, stats := PlanWork(rows, PlanParams{ExtractorVersion: "x/1", MatcherVersion: "m/1", ProfileRevision: 2, Matching: true, Order: c.order, Now: now,
			Skip: map[int64]string{2: hashOf(rows[1].Source.Description)}})
		var order []string
		for _, it := range plan {
			order = append(order, fmt.Sprintf("%d%s", it.Source.JobID, map[WorkKind]string{WorkExtract: "e", WorkMatch: "m"}[it.Kind]))
		}
		if got := strings.Join(order, " "); got != c.want {
			t.Errorf("%s: order = %s, want %s (job 2 cooling)", c.name, got, c.want)
		}
		if stats.Usable != 8 || stats.Cooling != 1 || stats.Matched != 1 || stats.NoRequirements != 1 || stats.PendingMatch != 1 || stats.PendingExtract != 4 {
			t.Errorf("%s: stats = %+v", c.name, stats)
		}
	}
}

// Status is polled by the owner's page every few seconds. It must answer from memory: when the
// database is stuck, a status that reads it stops every other request that waits for the same
// connection. Closing the store after a planning pass stands for a stuck database: a Status that
// still read it would return zero counts (the read fails), not the cached ones.
func TestStatusAnswersWithoutTheDatabase(t *testing.T) {
	f := newFit(t, nil)
	f.job("Rust Engineer", ruleDesc, "available structured description")
	f.job("Rust Developer", otherDoc, "available structured description")
	f.drain()
	want := f.w.Status()
	if want.Usable != 2 || want.Matched != 2 {
		t.Fatalf("status before closing the store = %+v", want)
	}
	f.s.Close()
	done := make(chan FitStatus, 1)
	go func() { done <- f.w.Status() }()
	select {
	case got := <-done:
		if got != want {
			t.Fatalf("status after the store was closed = %+v, want %+v", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Status did not answer")
	}
}

func TestProviderDownBacksOffAndKeepsTheWorkWithoutSettingItAside(t *testing.T) {
	f := newFit(t, nil)
	f.job("Rust Engineer", ruleDesc, "available structured description")
	f.failWith = errors.New("dial tcp 127.0.0.1:11434: connection refused")
	wait := f.w.Step(context.Background())
	if wait <= 0 {
		t.Fatalf("a provider failure must back off, got wait=%v", wait)
	}
	st := f.w.Status()
	if st.State != FitStateWaiting || st.ModelAvailable || st.PendingExtract != 1 || st.Cooling != 0 {
		t.Fatalf("status = %+v; the model is unavailable, the vacancy is still owed and not set aside", st)
	}
	if wait2 := f.w.Step(context.Background()); wait2 <= wait {
		t.Errorf("the backoff must grow: %v then %v", wait, wait2)
	}
	f.mu.Lock()
	f.failWith = nil
	f.mu.Unlock()
	f.drain()
	if st := f.w.Status(); st.State != FitStateIdle || !st.ModelAvailable || st.Matched != 1 || st.LastError != "" {
		t.Fatalf("after the model came back: %+v", st)
	}
}

func TestUnusableAnswerSetsOnlyThatVacancyAside(t *testing.T) {
	f := newFit(t, nil)
	f.answerBad = true
	f.w.Infer = f.infer
	f.job("Rust Engineer", ruleDesc, "available structured description")
	f.w.Step(context.Background())
	f.w.Step(context.Background()) // the second planning pass sees the vacancy set aside
	st := f.w.Status()
	if st.Cooling != 1 || st.PendingExtract != 0 {
		t.Fatalf("status = %+v; the vacancy whose answer could not be used must be set aside, not retried forever", st)
	}
	n1, _ := f.counts()
	f.drain()
	if n2, _ := f.counts(); n2 != n1 {
		t.Fatalf("a set-aside vacancy was sent again: %d then %d", n1, n2)
	}
}

func TestPauseBlockedProviderAndLimit(t *testing.T) {
	t.Run("paused", func(t *testing.T) {
		f := newFit(t, func(c *config.Config) { c.Runtime.QuickMatch.ModelPasses.Paused = true })
		f.job("Rust Engineer", ruleDesc, "available structured description")
		f.w.Step(context.Background())
		if n, _ := f.counts(); n != 0 || f.w.Status().State != FitStatePaused {
			t.Fatalf("paused worker made %d calls, state %s", n, f.w.Status().State)
		}
	})
	t.Run("paid provider is refused", func(t *testing.T) {
		f := newFit(t, func(c *config.Config) {
			c.Providers.Providers["local"] = config.Provider{Type: "opencode", Model: "luna", Endpoint: "https://api.example.com", TimeoutMS: 1000, ContextChars: 32000}
		})
		f.job("Rust Engineer", ruleDesc, "available structured description")
		f.w.Step(context.Background())
		st := f.w.Status()
		if n, _ := f.counts(); n != 0 || st.State != FitStateBlocked || !strings.Contains(st.Message, "refused") {
			t.Fatalf("a paid provider must stop the work: calls=%d status=%+v", n, st)
		}
	})
	t.Run("daily limit", func(t *testing.T) {
		f := newFit(t, func(c *config.Config) { c.Runtime.QuickMatch.ModelPasses.DailyLimit = 1 })
		f.job("Rust Engineer", ruleDesc, "available structured description")
		f.job("Rust Developer", otherDoc, "available structured description")
		f.drain()
		if n, _ := f.counts(); n != 1 {
			t.Fatalf("extractions = %d, want exactly the limit of 1", n)
		}
		if st := f.w.Status(); st.State != FitStateLimit || st.PendingExtract != 1 {
			t.Fatalf("status = %+v", st)
		}
	})
	t.Run("another run holds the inference lock", func(t *testing.T) {
		f := newFit(t, nil)
		f.job("Rust Engineer", ruleDesc, "available structured description")
		holdModelLock(t)
		f.w.Step(context.Background())
		if n, _ := f.counts(); n != 0 {
			t.Fatalf("the worker called the model while the inference lock was held elsewhere")
		}
	})
}
