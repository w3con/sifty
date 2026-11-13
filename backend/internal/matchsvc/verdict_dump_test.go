package matchsvc

// TestDumpRestrictionVerdicts writes one line per usable vacancy to the file at
// SIFTY_VERDICT_OUT: "id<TAB>verdict" where verdict is one of:
//
//   covered        – remote_restricted_covered (Matches)
//   not_covered    – remote_restricted_not_covered (Mismatches)
//   unknown        – remote_restricted_unknown (ToClarify)
//   not_applicable – criterion absent; vacancy has no applicable remote restriction
//
// Identical in structure to the post-N.1 version; the behavioural difference comes from
// the code and Rules loaded by activeSettings at each commit.
//
// Run as:
//
//	SIFTY_PROBE_DATA_DIR=.../scratch/data \
//	  SIFTY_VERDICT_OUT=/tmp/before.tsv \
//	  go test ./internal/matchsvc/ -run TestDumpRestrictionVerdicts -count=1 -v
import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

func TestDumpRestrictionVerdicts(t *testing.T) {
	dataDir := os.Getenv("SIFTY_PROBE_DATA_DIR")
	if dataDir == "" {
		t.Skip("set SIFTY_PROBE_DATA_DIR to an absolute path to a copy of the data directory")
	}
	outPath := os.Getenv("SIFTY_VERDICT_OUT")
	if outPath == "" {
		t.Skip("set SIFTY_VERDICT_OUT to the absolute path of the output file")
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

	t.Logf("profile: authorizations=%d", len(snap.Authorizations))
	t.Logf("rules: regions=%d restrictionPhrases=%d",
		len(set.rules.Regions), len(set.rules.RestrictionPhrases))

	restrictionVerdict := func(r matchcontract.FitResult) string {
		ls := map[string][]matchcontract.FitLine{
			"Matches":    r.Lists.Matches,
			"Mismatches": r.Lists.Mismatches,
			"Missing":    r.Lists.Missing,
			"ToClarify":  r.Lists.ToClarify,
		}
		for name, lines := range ls {
			for _, l := range lines {
				if l.Criterion == matchcontract.CriterionGeographyRemoteRestriction {
					if l.Reason == matchcontract.ReasonRemoteUnrestricted {
						return "unrestricted"
					}
					switch name {
					case "Matches":
						return "covered"
					case "Mismatches":
						return "not_covered"
					case "ToClarify":
						return "unknown"
					default:
						return "missing"
					}
				}
			}
		}
		return "not_applicable"
	}

	allJobs, err := st.List("", "")
	if err != nil {
		t.Fatal(err)
	}

	var rows []string
	for _, j := range allJobs {
		src := matchcontract.RetainedSource{
			JobID: j.ID, Title: j.Title, Company: j.Company, Tags: j.Tags,
			Description: j.Description, DescriptionStatus: j.DescriptionStatus,
		}
		if jobextract.SourceStateOf(src) != matchcontract.SourceUsable {
			continue
		}
		hash := matchcontract.HashDescription(j.Description)
		facts, factsErr := svc.factsFor(j.ID, set)
		if factsErr != nil || facts == nil {
			continue
		}
		in := matchcontract.FitInput{
			Profile: snap, Source: matchcontract.SourceUsable,
			Facts: facts, DescriptionHash: hash, Rules: set.rules,
		}
		ext, extErr := st.GetVacancyExtraction(j.ID, hash, vacancyai.ExtractorVersion())
		if extErr != nil {
			t.Logf("job %d extraction error: %v", j.ID, extErr)
			continue
		}
		if ext != nil {
			merged, conflicts, mergeErr := jobextract.MergeModelConditions(*facts, ext.Conditions, set.rules.RestrictionPhrases)
			if mergeErr != nil {
				t.Logf("job %d merge error: %v", j.ID, mergeErr)
				continue
			}
			in.Facts = &merged
			in.ConditionConflicts = conflicts
		}
		res, evalErr := quickmatch.EvaluateFit(in)
		if evalErr != nil {
			t.Logf("job %d eval error: %v", j.ID, evalErr)
			continue
		}
		scoreStr := "nil"
		if res.FitsYou.Score != nil {
			scoreStr = fmt.Sprintf("%d", *res.FitsYou.Score)
		}
		rows = append(rows, fmt.Sprintf("%d\t%s\t%s\t%s\t%s", j.ID, restrictionVerdict(res), scoreStr, j.Company, j.Title))
	}

	sort.Slice(rows, func(i, j int) bool {
		return rowID(rows[i]) < rowID(rows[j])
	})

	content := strings.Join(rows, "\n") + "\n"
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d rows to %s", len(rows), outPath)
}

// rowID extracts the numeric job ID from a "id\tverdict" row for numeric sorting.
func rowID(row string) int64 {
	var id int64
	fmt.Sscanf(row, "%d", &id)
	return id
}

