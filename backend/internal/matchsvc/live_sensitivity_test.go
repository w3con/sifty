package matchsvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/quickmatch"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)

// TestLiveFitSensitivity is a gated diagnostic: it measures how much the first score ("Fits you")
// depends on the conditions a model reads from the vacancy text. For each vacancy of a COPY of the
// application data it scores the vacancy from the deterministic facts alone, then again after
// merging the conditions of each named source, and writes the scores, the fields the source filled
// and the conflicts to a JSON file. No model is called and nothing is written to the store.
//
// SIFTY_PROBE_DATA_DIR  a copy of the data directory
// SIFTY_PROBE_JOB_IDS   comma-separated job ids
// SIFTY_SENS_SOURCES    comma-separated list of "label=path"; path is an extraction run file written by
//
//	TestLiveExtractionRegression, or "stored" to use the extractions held in the store
//
// SIFTY_SENS_GROUND_ENGAGEMENT  when set, drops each source's engagement whose quote names no kind of agreement
// SIFTY_SENS_OUT        the JSON file to write
func TestLiveFitSensitivity(t *testing.T) {
	dataDir, idList, sources, out := os.Getenv("SIFTY_PROBE_DATA_DIR"), os.Getenv("SIFTY_PROBE_JOB_IDS"),
		os.Getenv("SIFTY_SENS_SOURCES"), os.Getenv("SIFTY_SENS_OUT")
	if dataDir == "" || idList == "" || sources == "" || out == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR, SIFTY_PROBE_JOB_IDS, SIFTY_SENS_SOURCES and SIFTY_SENS_OUT")
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

	type runFile struct {
		Jobs []struct {
			ID         int64                             `json:"id"`
			Conditions matchcontract.ExtractedConditions `json:"conditions"`
		} `json:"jobs"`
	}
	type source struct {
		label string
		conds map[int64]matchcontract.ExtractedConditions
	}
	var srcs []source
	for _, part := range strings.Split(sources, ",") {
		label, path, _ := strings.Cut(part, "=")
		s := source{label: label, conds: map[int64]matchcontract.ExtractedConditions{}}
		if path == "stored" {
			rows, err := st.DB.Query(`SELECT DISTINCT job_id FROM match_vacancy_extractions`)
			if err != nil {
				t.Fatal(err)
			}
			var ids []int64
			for rows.Next() {
				var id int64
				_ = rows.Scan(&id)
				ids = append(ids, id)
			}
			rows.Close()
			for _, id := range ids {
				job, err := st.Get(id)
				if err != nil {
					continue
				}
				ext, err := st.GetVacancyExtraction(id, matchcontract.HashDescription(job.Description), vacancyai.ExtractorVersion())
				if err == nil && ext != nil {
					s.conds[id] = ext.Conditions
				}
			}
		} else {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var rf runFile
			if err := json.Unmarshal(raw, &rf); err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			for _, j := range rf.Jobs {
				s.conds[j.ID] = j.Conditions
			}
		}
		if os.Getenv("SIFTY_SENS_GROUND_ENGAGEMENT") != "" {
			for id, c := range s.conds {
				if c.Engagement != nil && !vacancyai.GroundEngagement(c.Engagement.Quote) {
					c.Engagement = nil
					s.conds[id] = c
				}
			}
		}
		srcs = append(srcs, s)
	}

	type result struct {
		Score     *int     `json:"score"`
		Raw       *float64 `json:"raw"`
		Status    string   `json:"status"`
		Filled    []string `json:"filled,omitempty"`
		Conflicts []string `json:"conflicts,omitempty"`
		Present   bool     `json:"present"`
		// OnlyField is the score after merging a single condition field of the source.
		OnlyField map[string]*int `json:"onlyField,omitempty"`
	}
	type row struct {
		ID            int64             `json:"id"`
		Company       string            `json:"company"`
		Title         string            `json:"title"`
		Deterministic result            `json:"deterministic"`
		Sources       map[string]result `json:"sources"`
		FactFields    []string          `json:"factFields"`
	}
	var rows []row
	score := func(job int64, facts *matchcontract.VacancyFacts, conflicts []matchcontract.ConditionConflict, hash string) result {
		in := matchcontract.FitInput{
			Profile: profile.QuickMatchSnapshot(cur), Source: matchcontract.SourceUsable,
			Facts: facts, DescriptionHash: hash, Rules: set.rules, ConditionConflicts: conflicts,
		}
		res, err := quickmatch.EvaluateFit(in)
		if err != nil {
			t.Fatalf("job %d: %v", job, err)
		}
		return result{Score: res.FitsYou.Score, Raw: res.FitsYou.RawScore, Status: string(res.FitsYou.Status), Present: true}
	}
	fields := func(f matchcontract.VacancyFacts) map[string]bool {
		return map[string]bool{
			"work_mode": len(f.Conditions.WorkModes) > 0, "engagement": f.Conditions.Engagement != nil,
			"hours": f.Conditions.Hours != nil, "timezone": f.Conditions.Timezone != nil,
			"compensation": f.Compensation.Cash != nil,
			"location":     len(f.Geography.HiringLocations) > 0,
		}
	}
	for _, part := range strings.Split(idList, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		job, err := st.Get(id)
		if err != nil {
			t.Fatalf("job %d: %v", id, err)
		}
		src := matchcontract.RetainedSource{
			JobID: id, Title: job.Title, Company: job.Company, Tags: job.Tags,
			Description: job.Description, DescriptionStatus: job.DescriptionStatus,
		}
		if jobextract.SourceStateOf(src) != matchcontract.SourceUsable {
			continue
		}
		facts, err := svc.factsFor(id, set)
		if err != nil || facts == nil {
			t.Fatalf("facts %d: %v", id, err)
		}
		hash := matchcontract.HashDescription(job.Description)
		r := row{ID: id, Company: job.Company, Title: job.Title, Sources: map[string]result{}}
		for k, v := range fields(*facts) {
			if v {
				r.FactFields = append(r.FactFields, k)
			}
		}
		r.Deterministic = score(id, facts, nil, hash)
		base := fields(*facts)
		for _, s := range srcs {
			c, ok := s.conds[id]
			if !ok {
				r.Sources[s.label] = result{}
				continue
			}
			merged, conflicts, err := jobextract.MergeModelConditions(*facts, c)
			if err != nil {
				t.Fatalf("merge %d: %v", id, err)
			}
			res := score(id, &merged, conflicts, hash)
			after := fields(merged)
			for k, v := range after {
				if v && !base[k] {
					res.Filled = append(res.Filled, k)
				}
			}
			for _, cf := range conflicts {
				res.Conflicts = append(res.Conflicts, cf.Field+": "+cf.Deterministic+" vs "+cf.Model)
			}
			res.OnlyField = map[string]*int{}
			one := map[string]func(matchcontract.ExtractedConditions) matchcontract.ExtractedConditions{
				"compensation": func(x matchcontract.ExtractedConditions) matchcontract.ExtractedConditions {
					return matchcontract.ExtractedConditions{Compensation: x.Compensation}
				},
				"work_mode": func(x matchcontract.ExtractedConditions) matchcontract.ExtractedConditions {
					return matchcontract.ExtractedConditions{WorkMode: x.WorkMode}
				},
				"engagement": func(x matchcontract.ExtractedConditions) matchcontract.ExtractedConditions {
					return matchcontract.ExtractedConditions{Engagement: x.Engagement}
				},
				"location": func(x matchcontract.ExtractedConditions) matchcontract.ExtractedConditions {
					return matchcontract.ExtractedConditions{Location: x.Location}
				},
				"timezone": func(x matchcontract.ExtractedConditions) matchcontract.ExtractedConditions {
					return matchcontract.ExtractedConditions{Timezone: x.Timezone}
				},
				"hours": func(x matchcontract.ExtractedConditions) matchcontract.ExtractedConditions {
					return matchcontract.ExtractedConditions{Hours: x.Hours}
				},
			}
			for name, pick := range one {
				m1, c1, err := jobextract.MergeModelConditions(*facts, pick(c))
				if err != nil {
					t.Fatalf("merge %d %s: %v", id, name, err)
				}
				res.OnlyField[name] = score(id, &m1, c1, hash).Score
			}
			r.Sources[s.label] = res
		}
		rows = append(rows, r)
	}
	raw, _ := json.MarshalIndent(rows, "", " ")
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d vacancies to %s", len(rows), out)
}
