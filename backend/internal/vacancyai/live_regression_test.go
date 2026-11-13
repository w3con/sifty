package vacancyai_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)

// TestLiveExtractionRegression is a gated diagnostic for the regression set of the extraction
// prompt: it runs pass 1 only, on a COPY of the application data, and writes one JSON record per
// vacancy (kept requirements and conditions, what validation dropped, the model's raw answer and
// the time). The expectations live next to the card and are compared by compare.py. Set
// SIFTY_PROBE_DATA_DIR, SIFTY_PROBE_JOB_IDS and SIFTY_REGRESSION_OUT; otherwise it is skipped.
func TestLiveExtractionRegression(t *testing.T) {
	dataDir, idList, out := os.Getenv("SIFTY_PROBE_DATA_DIR"), os.Getenv("SIFTY_PROBE_JOB_IDS"), os.Getenv("SIFTY_REGRESSION_OUT")
	if dataDir == "" || idList == "" || out == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR, SIFTY_PROBE_JOB_IDS and SIFTY_REGRESSION_OUT to run the live regression")
	}
	d, err := dirs.Resolve(dataDir)
	if err != nil {
		t.Fatalf("resolve data dir: %v", err)
	}
	cfg, err := config.Load(d)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	ex, err := vacancyai.NewExtractor(cfg, nil)
	if err != nil {
		t.Fatalf("NewExtractor: %v", err)
	}
	s, err := store.Open(filepath.Join(d.Root, "store.sqlite"))
	if err != nil {
		t.Fatalf("open store copy: %v", err)
	}
	defer s.Close()

	type record struct {
		ID           int64           `json:"id"`
		Title        string          `json:"title"`
		Seconds      float64         `json:"seconds"`
		Attempts     int             `json:"attempts"`
		Error        string          `json:"error,omitempty"`
		Requirements json.RawMessage `json:"requirements"`
		Conditions   json.RawMessage `json:"conditions"`
		Skipped      []string        `json:"skipped"`
		Raw          json.RawMessage `json:"raw"`
	}
	var all []record
	for _, idText := range strings.Split(idList, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(idText), 10, 64)
		if err != nil {
			t.Fatalf("job id %q: %v", idText, err)
		}
		src, err := s.CaptureMatchInput(id)
		if err != nil {
			t.Fatalf("job %d: %v", id, err)
		}
		res, err := ex.Extract(context.Background(), src)
		r := record{ID: id, Title: src.Title + " | " + src.Company, Seconds: res.Call.Duration.Seconds(), Attempts: res.Call.Attempts,
			Skipped: res.Skipped, Raw: res.Raw}
		if err != nil {
			r.Error = err.Error()
		}
		r.Requirements, _ = json.Marshal(res.Record.Requirements)
		r.Conditions, _ = json.Marshal(res.Record.Conditions)
		all = append(all, r)
	}
	blob, _ := json.MarshalIndent(map[string]any{"extractorVersion": vacancyai.ExtractorVersion(), "model": ex.Provider.Model, "jobs": all}, "", "  ")
	if err := os.WriteFile(out, blob, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
