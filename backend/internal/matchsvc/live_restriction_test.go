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
			merged, conflicts, mergeErr := jobextract.MergeModelConditions(*facts, ext.Conditions, rules.RestrictionPhrases)
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
			merged, conflicts, mergeErr := jobextract.MergeModelConditions(*facts, ext.Conditions, rules.RestrictionPhrases)
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

// TestLiveRestrictionPhrases compares every vacancy with usable text and a stored model extraction
// under the same region vocabulary twice: with no restriction phrases (every model-read location
// needs corroboration, the behaviour before ASSESSMENT-9) and with the configured phrases. It uses
// the data copy's real profile unchanged. It prints the restriction verdict transitions, every
// other line that changed, the model location quote of each changed vacancy, and vacancy 1136.
//
// Run as:
//
//	SIFTY_PROBE_DATA_DIR=.../scratch/data go test ./internal/matchsvc/ -run TestLiveRestrictionPhrases -count=1 -v
func TestLiveRestrictionPhrases(t *testing.T) {
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
	set, err := svc.activeSettings(cfg)
	if err != nil {
		t.Fatal(err)
	}
	snap := profile.QuickMatchSnapshot(cur)
	t.Logf("profile (unchanged copy): authorizations=%+v currentLocation=%v", snap.Authorizations, snap.CurrentLocation)
	t.Logf("rules: regions=%d restrictionPhrases=%q", len(set.rules.Regions), set.rules.RestrictionPhrases)

	type outcome struct {
		res     matchcontract.FitResult
		verdict string
		lines   map[string]bool
	}
	lists := func(r matchcontract.FitResult) map[string][]matchcontract.FitLine {
		return map[string][]matchcontract.FitLine{
			"Matches": r.Lists.Matches, "Mismatches": r.Lists.Mismatches,
			"Missing": r.Lists.Missing, "ToClarify": r.Lists.ToClarify,
		}
	}
	summarize := func(r matchcontract.FitResult) outcome {
		o := outcome{res: r, verdict: "absent", lines: map[string]bool{}}
		for name, ls := range lists(r) {
			for _, l := range ls {
				if l.Origin != matchcontract.OriginGoal {
					continue
				}
				o.lines[name+" "+string(l.Criterion)+" "+l.Reason] = true
				if l.Criterion == matchcontract.CriterionGeographyRemoteRestriction {
					o.verdict = l.Reason + " (" + name + ")"
				}
			}
		}
		return o
	}
	score := func(r matchcontract.FitResult) string {
		if r.FitsYou.Score == nil {
			return "nil/" + string(r.FitsYou.Status)
		}
		return fmt.Sprintf("%d", *r.FitsYou.Score)
	}

	jobs, err := st.List("", "")
	if err != nil {
		t.Fatal(err)
	}
	evaluated, withModelLocation, usableModelLocation := 0, 0, 0
	transitions := map[string]int{}
	otherChanges := map[string]int{}
	var changed, usableUnchanged []string
	for _, j := range jobs {
		src := matchcontract.RetainedSource{
			JobID: j.ID, Title: j.Title, Company: j.Company, Tags: j.Tags,
			Description: j.Description, DescriptionStatus: j.DescriptionStatus,
		}
		if jobextract.SourceStateOf(src) != matchcontract.SourceUsable {
			continue
		}
		hash := matchcontract.HashDescription(j.Description)
		ext, err := st.GetVacancyExtraction(j.ID, hash, vacancyai.ExtractorVersion())
		if err != nil || ext == nil {
			continue
		}
		facts, err := svc.factsFor(j.ID, set)
		if err != nil || facts == nil {
			t.Logf("job %d facts: %v", j.ID, err)
			continue
		}
		run := func(phrases []string) (outcome, matchcontract.VacancyFacts, error) {
			merged, conflicts, err := jobextract.MergeModelConditions(*facts, ext.Conditions, phrases)
			if err != nil {
				return outcome{}, merged, err
			}
			res, err := quickmatch.EvaluateFit(matchcontract.FitInput{
				Profile: snap, Source: matchcontract.SourceUsable, Facts: &merged,
				DescriptionHash: hash, Rules: set.rules, ConditionConflicts: conflicts,
				Extraction: ext, MatchingOff: !set.matchingOn,
			})
			return summarize(res), merged, err
		}
		old, _, err := run(nil)
		if err != nil {
			t.Logf("job %d old: %v", j.ID, err)
			continue
		}
		cur, merged, err := run(set.rules.RestrictionPhrases)
		if err != nil {
			t.Logf("job %d new: %v", j.ID, err)
			continue
		}
		evaluated++
		modelUsable := false
		if ext.Conditions.Location != nil {
			withModelLocation++
			for _, l := range merged.Geography.HiringLocations {
				if l.Evidence.Field == "model_extraction" && l.Evidence.Reliability == matchcontract.ReliabilityUsable {
					modelUsable = true
				}
			}
		}
		if modelUsable {
			usableModelLocation++
		}
		if old.verdict != cur.verdict {
			transitions[old.verdict+" -> "+cur.verdict]++
		}
		var diff []string
		for k := range old.lines {
			if !cur.lines[k] {
				diff = append(diff, "-"+k)
			}
		}
		for k := range cur.lines {
			if !old.lines[k] {
				diff = append(diff, "+"+k)
			}
		}
		sort.Strings(diff)
		if old.verdict == cur.verdict && len(diff) > 0 {
			otherChanges[fmt.Sprint(diff)]++
		}
		if modelUsable && len(diff) == 0 {
			var modes []string
			for _, m := range merged.Conditions.WorkModes {
				modes = append(modes, m.Mode)
			}
			usableUnchanged = append(usableUnchanged, fmt.Sprintf("job %d %s / %s: modes %v; restriction %s; model location value=%q quote=%q",
				j.ID, j.Company, j.Title, modes, cur.verdict, ext.Conditions.Location.Value, ext.Conditions.Location.Quote))
		}
		if len(diff) > 0 || score(old.res) != score(cur.res) || j.ID == 1136 {
			loc := ""
			if c := ext.Conditions.Location; c != nil {
				loc = fmt.Sprintf("value=%q quote=%q", c.Value, c.Quote)
			}
			changed = append(changed, fmt.Sprintf("job %d %s / %s: score %s -> %s; restriction %s -> %s; lines %v; model location %s",
				j.ID, j.Company, j.Title, score(old.res), score(cur.res), old.verdict, cur.verdict, diff, loc))
		}
	}
	t.Logf("=== evaluated (usable text + stored model extraction): %d; with a model location: %d; model location now usable: %d ===",
		evaluated, withModelLocation, usableModelLocation)
	var keys []string
	for k := range transitions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("transition %s: %d", k, transitions[k])
	}
	keys = keys[:0]
	for k := range otherChanges {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("other change (restriction verdict unchanged) %s: %d", k, otherChanges[k])
	}
	t.Logf("=== changed vacancies (and 1136): %d ===", len(changed))
	for _, line := range changed {
		t.Log(line)
	}
	t.Logf("=== model location now usable, no line changed: %d ===", len(usableUnchanged))
	for _, line := range usableUnchanged {
		t.Log(line)
	}
}
