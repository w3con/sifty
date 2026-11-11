package vacancyai_test

import (
	"context"
	"encoding/json"
	"errors"
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

// TestLiveExportMatchInputs is a gated diagnostic that writes, for vacancies of a COPY of the
// application data, exactly what the matching pass would show the model (the requirements that
// code did not decide, the profile facts chosen for each and the computed facts), without calling
// any model. It exists to test other kinds of model on the same inputs. Set SIFTY_PROBE_DATA_DIR
// (a copy), SIFTY_PROBE_JOB_IDS (comma-separated ids, or "all" for every vacancy that has an
// extraction) and SIFTY_EXPORT_OUT (the JSON file to write).
func TestLiveExportMatchInputs(t *testing.T) {
	dataDir, idList, out := os.Getenv("SIFTY_PROBE_DATA_DIR"), os.Getenv("SIFTY_PROBE_JOB_IDS"), os.Getenv("SIFTY_EXPORT_OUT")
	if dataDir == "" || idList == "" || out == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR, SIFTY_PROBE_JOB_IDS and SIFTY_EXPORT_OUT to export the matching inputs")
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

	var ids []int64
	if idList == "all" {
		rows, err := s.DB.Query(`SELECT DISTINCT job_id FROM match_vacancy_extractions ORDER BY job_id`)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		rows.Close()
	} else {
		for _, part := range strings.Split(idList, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
	}

	var request string
	errStop := errors.New("stop after the request is built")
	infer := func(ctx context.Context, p assess.Provider, req string, schema any, o any) error {
		request = req
		return errStop
	}
	mt, err := vacancyai.NewMatcher(cfg, infer)
	if err != nil {
		t.Fatal(err)
	}
	mt.MaxAttempts, mt.Backoff = 1, 0

	type item struct {
		JobID       int64                           `json:"jobId"`
		CodeMatched int                             `json:"codeMatched"`
		Asked       int                             `json:"asked"`
		Request     string                          `json:"request"`
		Extraction  matchcontract.VacancyExtraction `json:"extraction"`
	}
	var all []item
	for _, id := range ids {
		var blob string
		if err := s.DB.QueryRow(`SELECT record FROM match_vacancy_extractions WHERE job_id=? ORDER BY created_at DESC LIMIT 1`, id).Scan(&blob); err != nil {
			t.Fatalf("job %d: %v", id, err)
		}
		var e matchcontract.VacancyExtraction
		if err := json.Unmarshal([]byte(blob), &e); err != nil {
			t.Fatal(err)
		}
		request = ""
		res, _ := mt.Match(context.Background(), e, facts, prof.Revision)
		it := item{JobID: id, CodeMatched: res.CodeMatched, Asked: res.Asked, Extraction: e}
		if i := strings.Index(request, "REQUIREMENTS (DATA)"); i >= 0 {
			it.Request = request[i:]
		}
		all = append(all, it)
	}
	blob, _ := json.MarshalIndent(all, "", " ")
	if err := os.WriteFile(out, blob, 0o644); err != nil {
		t.Fatal(err)
	}
}
