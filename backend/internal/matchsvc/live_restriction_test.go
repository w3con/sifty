package matchsvc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/quickmatch"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)


// TestLiveRemoteRestriction compares Fits you scores for all vacancies evaluated with no region
// vocabulary (old behaviour, regions = nil) against scores evaluated with the region vocabulary
// loaded from the data-copy config. Stored model extraction conditions are merged in when present,
// matching the production scoring path. It prints every vacancy whose score changes and reports
// vacancy 1136 (Relevant Healthcare) in detail.
//
// SIFTY_PROBE_DATA_DIR  path to a copy of the data directory
//
// Run as:
//
//	SIFTY_PROBE_DATA_DIR=.../scratch/data go test ./internal/matchsvc/ -run TestLiveRemoteRestriction -v
func TestLiveRemoteRestriction(t *testing.T) {
	dataDir := os.Getenv("SIFTY_PROBE_DATA_DIR")
	if dataDir == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR to a copy of the data directory")
	}

	d, err := dirs.Resolve(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(d)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(d.Root, "store.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	svc := New(st, config.NewHolder(cfg))
	cur, err := profile.Current(st)
	if err != nil || cur == nil {
		t.Fatalf("profile: %v", err)
	}

	// Rules with regions loaded from the config (new behaviour).
	setNew, err := svc.activeSettings(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Rules without regions (old behaviour): copy rules but clear the region list.
	rulesOld := setNew.rules
	rulesOld.Regions = nil

	snap := profile.QuickMatchSnapshot(cur)

	t.Logf("profile: authorizations=%d currentLocation=%v",
		len(snap.Authorizations), snap.CurrentLocation)
	t.Logf("config: regions=%d", len(setNew.rules.Regions))

	scoreStr := func(s *int) string {
		if s == nil {
			return "nil"
		}
		return fmt.Sprintf("%d", *s)
	}

	// eval scores one vacancy under the given rules, merging stored model conditions when present.
	eval := func(j *model.Job, rules matchcontract.Rules) (matchcontract.FitResult, error) {
		facts, err := svc.factsFor(j.ID, setNew)
		if err != nil || facts == nil {
			return matchcontract.FitResult{}, fmt.Errorf("facts: %w", err)
		}
		hash := matchcontract.HashDescription(j.Description)
		in := matchcontract.FitInput{
			Profile: snap, Source: matchcontract.SourceUsable,
			Facts: facts, DescriptionHash: hash, Rules: rules,
		}
		ext, err := st.GetVacancyExtraction(j.ID, hash, vacancyai.ExtractorVersion())
		if err != nil {
			return matchcontract.FitResult{}, fmt.Errorf("extraction: %w", err)
		}
		if ext != nil {
			merged, conflicts, mergeErr := jobextract.MergeModelConditions(*facts, ext.Conditions)
			if mergeErr != nil {
				return matchcontract.FitResult{}, fmt.Errorf("merge: %w", mergeErr)
			}
			in.Facts, in.ConditionConflicts = &merged, conflicts
		}
		return quickmatch.EvaluateFit(in)
	}

	type scored struct {
		scoreOld  *int
		scoreNew  *int
		statusOld string
		statusNew string
		company   string
		title     string
	}

	allJobs, err := st.List("", "")
	if err != nil {
		t.Fatal(err)
	}

	results := make(map[int64]*scored)
	var changed []int64

	for _, j := range allJobs {
		src := matchcontract.RetainedSource{
			JobID: j.ID, Title: j.Title, Company: j.Company, Tags: j.Tags,
			Description: j.Description, DescriptionStatus: j.DescriptionStatus,
		}
		if jobextract.SourceStateOf(src) != matchcontract.SourceUsable {
			continue
		}

		resOld, err := eval(j, rulesOld)
		if err != nil {
			t.Logf("job %d old eval error: %v", j.ID, err)
			continue
		}
		resNew, err := eval(j, setNew.rules)
		if err != nil {
			t.Logf("job %d new eval error: %v", j.ID, err)
			continue
		}

		r := &scored{
			scoreOld:  resOld.FitsYou.Score,
			scoreNew:  resNew.FitsYou.Score,
			statusOld: string(resOld.FitsYou.Status),
			statusNew: string(resNew.FitsYou.Status),
			company:   j.Company,
			title:     j.Title,
		}
		results[j.ID] = r

		if scoreStr(r.scoreOld) != scoreStr(r.scoreNew) || r.statusOld != r.statusNew {
			changed = append(changed, j.ID)
		}
	}

	sort.Slice(changed, func(a, b int) bool { return changed[a] < changed[b] })

	t.Logf("=== vacancies evaluated: %d, changed: %d ===", len(results), len(changed))
	for _, id := range changed {
		r := results[id]
		t.Logf("  job %d %s / %s: score %s→%s status %s→%s",
			id, r.company, r.title, scoreStr(r.scoreOld), scoreStr(r.scoreNew), r.statusOld, r.statusNew)
	}

	// Detailed report for vacancy 1136 (Relevant Healthcare).
	const targetID = 1136
	if r, ok := results[targetID]; ok {
		t.Logf("=== vacancy %d: %s / %s ===", targetID, r.company, r.title)
		t.Logf("  score: %s → %s", scoreStr(r.scoreOld), scoreStr(r.scoreNew))
		t.Logf("  status: %s → %s", r.statusOld, r.statusNew)
	} else {
		t.Logf("vacancy %d not found in evaluated set (may be unavailable or not in store)", targetID)
	}
}

// TestLiveRemoteRestrictionCurrentLocationOnly runs the same comparison as
// TestLiveRemoteRestriction but with an overridden profile that has only a
// current location (NZ) and no work authorizations. Per spec §6.2, restricted
// vacancies must produce unknown verdicts (not covered / not not-covered), so
// the score impact should differ from the authorization-based profile.
//
// SIFTY_PROBE_DATA_DIR  path to a copy of the data directory (same as above)
//
// Run as:
//
//	SIFTY_PROBE_DATA_DIR=.../scratch/data go test ./internal/matchsvc/ -run TestLiveRemoteRestrictionCurrentLocationOnly -v
func TestLiveRemoteRestrictionCurrentLocationOnly(t *testing.T) {
	dataDir := os.Getenv("SIFTY_PROBE_DATA_DIR")
	if dataDir == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR to a copy of the data directory")
	}

	d, err := dirs.Resolve(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(d)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(d.Root, "store.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	svc := New(st, config.NewHolder(cfg))
	cur, err := profile.Current(st)
	if err != nil || cur == nil {
		t.Fatalf("profile: %v", err)
	}

	setNew, err := svc.activeSettings(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rulesOld := setNew.rules
	rulesOld.Regions = nil

	// Override: use the real profile's goals but replace authorizations with none and
	// set only current location NZ. This simulates an owner who has not configured work
	// authorizations; restricted vacancies must produce unknown verdicts.
	realSnap := profile.QuickMatchSnapshot(cur)
	locOnlySnap := realSnap
	locOnlySnap.Authorizations = nil
	locOnlySnap.CurrentLocation = &matchcontract.ProfileLocation{Country: "NZ"}

	t.Logf("profile override: authorizations=0 currentLocation=NZ")
	t.Logf("config: regions=%d", len(setNew.rules.Regions))

	scoreStr := func(s *int) string {
		if s == nil {
			return "nil"
		}
		return fmt.Sprintf("%d", *s)
	}

	eval := func(j *model.Job, snap matchcontract.ProfileSnapshot, rules matchcontract.Rules) (matchcontract.FitResult, error) {
		facts, err := svc.factsFor(j.ID, setNew)
		if err != nil || facts == nil {
			return matchcontract.FitResult{}, fmt.Errorf("facts: %w", err)
		}
		hash := matchcontract.HashDescription(j.Description)
		in := matchcontract.FitInput{
			Profile: snap, Source: matchcontract.SourceUsable,
			Facts: facts, DescriptionHash: hash, Rules: rules,
		}
		ext, err := st.GetVacancyExtraction(j.ID, hash, vacancyai.ExtractorVersion())
		if err != nil {
			return matchcontract.FitResult{}, fmt.Errorf("extraction: %w", err)
		}
		if ext != nil {
			merged, conflicts, mergeErr := jobextract.MergeModelConditions(*facts, ext.Conditions)
			if mergeErr != nil {
				return matchcontract.FitResult{}, fmt.Errorf("merge: %w", mergeErr)
			}
			in.Facts, in.ConditionConflicts = &merged, conflicts
		}
		return quickmatch.EvaluateFit(in)
	}

	type scored struct {
		scoreOld  *int
		scoreNew  *int
		statusOld string
		statusNew string
		company   string
		title     string
	}

	allJobs, err := st.List("", "")
	if err != nil {
		t.Fatal(err)
	}

	results := make(map[int64]*scored)
	var changed []int64

	for _, j := range allJobs {
		src := matchcontract.RetainedSource{
			JobID: j.ID, Title: j.Title, Company: j.Company, Tags: j.Tags,
			Description: j.Description, DescriptionStatus: j.DescriptionStatus,
		}
		if jobextract.SourceStateOf(src) != matchcontract.SourceUsable {
			continue
		}

		resOld, err := eval(j, locOnlySnap, rulesOld)
		if err != nil {
			t.Logf("job %d old eval error: %v", j.ID, err)
			continue
		}
		resNew, err := eval(j, locOnlySnap, setNew.rules)
		if err != nil {
			t.Logf("job %d new eval error: %v", j.ID, err)
			continue
		}

		r := &scored{
			scoreOld:  resOld.FitsYou.Score,
			scoreNew:  resNew.FitsYou.Score,
			statusOld: string(resOld.FitsYou.Status),
			statusNew: string(resNew.FitsYou.Status),
			company:   j.Company,
			title:     j.Title,
		}
		results[j.ID] = r

		if scoreStr(r.scoreOld) != scoreStr(r.scoreNew) || r.statusOld != r.statusNew {
			changed = append(changed, j.ID)
		}
	}

	sort.Slice(changed, func(a, b int) bool { return changed[a] < changed[b] })

	t.Logf("=== vacancies evaluated: %d, changed: %d (location-only profile) ===", len(results), len(changed))
	for _, id := range changed {
		r := results[id]
		t.Logf("  job %d %s / %s: score %s→%s status %s→%s",
			id, r.company, r.title, scoreStr(r.scoreOld), scoreStr(r.scoreNew), r.statusOld, r.statusNew)
	}

	// Detailed report for vacancy 1136 (Relevant Healthcare).
	// With location-only profile: restriction must be unknown, not not-covered.
	// Score should differ from the authorization-based probe.
	const targetID = 1136
	if r, ok := results[targetID]; ok {
		t.Logf("=== vacancy %d (location-only): %s / %s ===", targetID, r.company, r.title)
		t.Logf("  score: %s → %s", scoreStr(r.scoreOld), scoreStr(r.scoreNew))
		t.Logf("  status: %s → %s", r.statusOld, r.statusNew)
	} else {
		t.Logf("vacancy %d not found in evaluated set", targetID)
	}
}
