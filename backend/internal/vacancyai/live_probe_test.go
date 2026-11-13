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
	"time"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
	"github.com/w3con/sifty/internal/vacancyai"
)

// TestLiveTwoPassProbe is a gated diagnostic: it runs both model passes against the provider the
// extraction stage names, on a COPY of the application data, and records what the model said. It
// never touches the live data directories. Set:
//
//	SIFTY_PROBE_DATA_DIR  a copy of a data directory (config/ and store.sqlite)
//	SIFTY_PROBE_JOB_IDS   comma-separated job ids
//	SIFTY_PROBE_OUT       file the markdown report is written to
//
// Every ordinary test run skips it.
func TestLiveTwoPassProbe(t *testing.T) {
	dataDir, idList, out := os.Getenv("SIFTY_PROBE_DATA_DIR"), os.Getenv("SIFTY_PROBE_JOB_IDS"), os.Getenv("SIFTY_PROBE_OUT")
	if dataDir == "" || idList == "" || out == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR, SIFTY_PROBE_JOB_IDS and SIFTY_PROBE_OUT to run the live two-pass probe")
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
	mt, err := vacancyai.NewMatcher(cfg, nil)
	if err != nil {
		t.Fatalf("NewMatcher: %v", err)
	}
	s, err := store.Open(filepath.Join(d.Root, "store.sqlite"))
	if err != nil {
		t.Fatalf("open store copy: %v", err)
	}
	defer s.Close()
	prof, err := profile.NewService(s, config.NewHolder(cfg)).Get()
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	facts := profile.MatchFacts(prof)

	var md strings.Builder
	fmt.Fprintf(&md, "# Q.3 live run of both model passes\n\n")
	fmt.Fprintf(&md, "Run: %s. Provider from configuration: stage extraction = `%s`, type `%s`, model `%s`, endpoint `%s`, context %d chars, timeout %d ms, thinking off, temperature %v, at most %d attempts.\n\n",
		time.Now().Format(time.RFC3339), ex.ProviderName, ex.Provider.Type, ex.Provider.Model, ex.Provider.Endpoint,
		ex.Provider.ContextChars, ex.Provider.TimeoutMS, *ex.Provider.Temperature, ex.MaxAttempts)
	fmt.Fprintf(&md, "Data: a copy of the staging store in `.local/assessment-3/data` (profile revision %d, %d profile facts offered to the matcher). Extractor version `%s`; matcher version `%s`.\n\n",
		prof.Revision, len(facts), vacancyai.ExtractorVersion(), vacancyai.MatcherVersion(cfg))

	ctx := context.Background()
	for _, idText := range strings.Split(idList, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(idText), 10, 64)
		if err != nil {
			t.Fatalf("job id %q: %v", idText, err)
		}
		src, err := s.CaptureMatchInput(id)
		if err != nil {
			t.Fatalf("job %d: %v", id, err)
		}
		fmt.Fprintf(&md, "## Job %d — %s (%s)\n\n", id, src.Title, src.Company)
		fmt.Fprintf(&md, "Description: %d characters, status `%s`.\n\n", len(src.Description), src.DescriptionStatus)

		// The raw answer is captured by wrapping the real transport.
		var rawCalls []string
		wrap := func(ctx context.Context, p assess.Provider, req string, schema any, out any) error {
			err := assess.InferInto(ctx, p, req, schema, out)
			if r, ok := out.(*json.RawMessage); ok && err == nil {
				rawCalls = append(rawCalls, string(*r))
			}
			return err
		}
		ex.Infer, mt.Infer = wrap, wrap

		res, err := ex.Extract(ctx, src)
		fmt.Fprintf(&md, "### Pass 1 — vacancy extraction\n\n")
		fmt.Fprintf(&md, "- Time: %s total, %d attempt(s), last call %s, request %d characters.\n", res.Call.Duration.Round(time.Millisecond), res.Call.Attempts, res.Call.LastDuration.Round(time.Millisecond), res.Call.RequestChars)
		if err != nil {
			fmt.Fprintf(&md, "- **Failed:** %v\n\n", err)
			appendRaw(&md, "Raw model answer (last attempt)", res.Raw)
			continue
		}
		var rawItems struct {
			Requirements []json.RawMessage `json:"requirements"`
		}
		_ = json.Unmarshal(res.Raw, &rawItems)
		fmt.Fprintf(&md, "- Quote check: model returned %d requirement(s); %d kept with an exact quote; %d dropped.\n", len(rawItems.Requirements), len(res.Record.Requirements), len(res.Skipped))
		for _, sk := range res.Skipped {
			fmt.Fprintf(&md, "  - dropped: %s\n", sk)
		}
		fmt.Fprintf(&md, "- Conditions kept: %s.\n\n", describeConditions(res.Record.Conditions))
		appendRaw(&md, "Raw model answer", res.Raw)
		fmt.Fprintf(&md, "Kept requirements (quote verified as an exact excerpt of the description):\n\n")
		for i, r := range res.Record.Requirements {
			ok := strings.Contains(strings.ReplaceAll(src.Description, "\r\n", "\n"), r.Quote)
			fmt.Fprintf(&md, "%d. **%s** — %s, %s. Quote ✔=%v: “%s”\n", i, r.Text, r.Kind, r.Level, ok, r.Quote)
		}
		fmt.Fprintln(&md)
		if err := s.SaveVacancyExtraction(res.Record); err != nil {
			t.Fatalf("job %d: save extraction into the copy: %v", id, err)
		}
		if len(res.Record.Requirements) == 0 {
			fmt.Fprintf(&md, "No requirements found, so pass 2 is not run.\n\n")
			continue
		}

		mres, err := mt.Match(ctx, res.Record, facts, prof.Revision)
		fmt.Fprintf(&md, "### Pass 2 — matching against the profile\n\n")
		fmt.Fprintf(&md, "- Time: %s total, %d attempt(s), last call %s, request %d characters.\n", mres.Call.Duration.Round(time.Millisecond), mres.Call.Attempts, mres.Call.LastDuration.Round(time.Millisecond), mres.Call.RequestChars)
		if err != nil {
			fmt.Fprintf(&md, "- **Failed:** %v\n\n", err)
			appendRaw(&md, "Raw model answer (last attempt)", mres.Raw)
			continue
		}
		fmt.Fprintf(&md, "- Profile-evidence check: %d downgrade(s) to unclear.\n", len(mres.Skipped))
		for _, sk := range mres.Skipped {
			fmt.Fprintf(&md, "  - %s\n", sk)
		}
		fmt.Fprintln(&md)
		appendRaw(&md, "Raw model answer", mres.Raw)
		fmt.Fprintf(&md, "Verdicts after validation:\n\n")
		for _, v := range mres.Record.Verdicts {
			ev := "—"
			if v.Evidence != nil {
				ev = v.Evidence.Field + ": " + v.Evidence.Value
			}
			fmt.Fprintf(&md, "- %s → **%s** (%s) %s\n", res.Record.Requirements[v.Index].Text, v.Verdict, ev, v.Reason)
		}
		fmt.Fprintln(&md)
		if err := s.SaveRequirementMatches(mres.Record); err != nil {
			t.Fatalf("job %d: save matches into the copy: %v", id, err)
		}
	}
	if err := os.WriteFile(out, []byte(md.String()), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	t.Logf("report written to %s", out)
}

func appendRaw(md *strings.Builder, title string, raw json.RawMessage) {
	pretty := string(raw)
	var v any
	if json.Unmarshal(raw, &v) == nil {
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
			pretty = string(b)
		}
	}
	fmt.Fprintf(md, "<details><summary>%s</summary>\n\n```json\n%s\n```\n\n</details>\n\n", title, pretty)
}

func describeConditions(c matchcontract.ExtractedConditions) string {
	var parts []string
	for name, v := range map[string]*matchcontract.ExtractedCondition{
		"compensation": c.Compensation, "workMode": c.WorkMode, "engagement": c.Engagement,
		"location": c.Location, "timezone": c.Timezone, "hours": c.Hours,
	} {
		if v != nil {
			parts = append(parts, fmt.Sprintf("%s=%q", name, v.Value))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}
