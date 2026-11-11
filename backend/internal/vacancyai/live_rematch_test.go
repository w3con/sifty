package vacancyai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)

// TestLiveRematchProbe is a gated diagnostic for the matching pass alone: it takes the stored
// extractions of some vacancies from a COPY of the application data and matches them again with
// the provider of the extraction stage, writing the request the model saw and its verdicts. It
// stores nothing. Set SIFTY_PROBE_DATA_DIR (a copy), SIFTY_PROBE_JOB_IDS and SIFTY_PROBE_OUT.
func TestLiveRematchProbe(t *testing.T) {
	dataDir, idList, out := os.Getenv("SIFTY_PROBE_DATA_DIR"), os.Getenv("SIFTY_PROBE_JOB_IDS"), os.Getenv("SIFTY_PROBE_OUT")
	if dataDir == "" || idList == "" || out == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR, SIFTY_PROBE_JOB_IDS and SIFTY_PROBE_OUT to run the live rematch probe")
	}
	d, err := dirs.Resolve(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(d)
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(filepath.Join(d.Root, "store.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	prof, err := profile.NewService(s, config.NewHolder(cfg)).Get()
	if err != nil {
		t.Fatal(err)
	}
	facts := profile.MatchFacts(prof)

	var md strings.Builder
	var lastRequest string
	infer := func(ctx context.Context, p assess.Provider, req string, schema any, o any) error {
		lastRequest = req
		return assess.InferInto(ctx, p, req, schema, o)
	}
	mt, err := vacancyai.NewMatcher(cfg, infer)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&md, "# Rematch probe, matcher %s, profile revision %d\n\n", vacancyai.MatcherVersion(cfg), prof.Revision)
	for _, part := range strings.Split(idList, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		var blob string
		if err := s.DB.QueryRow(`SELECT record FROM match_vacancy_extractions WHERE job_id=? ORDER BY created_at DESC LIMIT 1`, id).Scan(&blob); err != nil {
			t.Fatalf("job %d: %v", id, err)
		}
		var e matchcontract.VacancyExtraction
		if err := json.Unmarshal([]byte(blob), &e); err != nil {
			t.Fatal(err)
		}
		res, err := mt.Match(context.Background(), e, facts, prof.Revision)
		fmt.Fprintf(&md, "## job %d (code %d, asked %d, call %s)\n\n", id, res.CodeMatched, res.Asked, res.Call.Duration)
		if err != nil {
			fmt.Fprintf(&md, "ERROR: %v\n\n", err)
			continue
		}
		for i, v := range res.Record.Verdicts {
			ev := ""
			if v.Evidence != nil {
				ev = v.Evidence.Field + ": " + v.Evidence.Value
				if len(ev) > 90 {
					ev = ev[:90]
				}
			}
			fmt.Fprintf(&md, "- %d. **%s** %s %v | %s | %s\n", i, v.Verdict, e.Requirements[i].Text, e.Requirements[i].Alternatives, v.Reason, ev)
		}
		fmt.Fprintf(&md, "\nSkipped: %v\n\nLast request after the prompt:\n\n```\n%s\n```\n\n", res.Skipped,
			lastRequest[strings.Index(lastRequest, "REQUIREMENTS (DATA)"):])
	}
	if err := os.WriteFile(out, []byte(md.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
