package background

import (
	"sort"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/store"
)

// WorkKind is what one vacancy needs from the model passes.
type WorkKind string

const (
	// WorkExtract: no current extraction exists, so both passes run.
	WorkExtract WorkKind = "extract"
	// WorkMatch: the extraction is current but the matches are for another profile revision or
	// matcher version, so only the matching pass runs.
	WorkMatch WorkKind = "match"
)

// WorkItem is one vacancy the worker should process, in the order the worker takes them.
type WorkItem struct {
	Source matchcontract.RetainedSource
	Kind   WorkKind
}

// PlanParams are the inputs that decide what is current.
type PlanParams struct {
	ExtractorVersion string
	MatcherVersion   string
	ProfileRevision  int
	// Matching is quick_match.model_passes.matching. When false the model's matching pass is off:
	// an extracted vacancy is finished, nothing is queued for matching.
	Matching bool
	// Order is config.OrderFit (Fits you descending) or config.OrderNewest (the Inbox Newest key);
	// anything else is read as Best fit.
	Order string
	// Now is the time against which a publication date counts as in the future (Newest order only).
	Now time.Time
	// Skip lists vacancies that failed validation recently and must not be tried again yet, keyed by
	// job id to the description hash that failed. A changed description is tried again at once.
	Skip map[int64]string
}

// PlanStats counts the vacancies by what the model passes still owe them. Only vacancies with
// usable text are counted; text-less ones are never queued.
type PlanStats struct {
	Usable         int
	Extracted      int
	NoRequirements int
	WithRequires   int
	Matched        int
	PendingExtract int
	PendingMatch   int
	Cooling        int
}

// PlanWork decides which vacancies need which pass and in which order. There is one list for all
// model work, extraction and re-matching alike, in the order of the Inbox sort toggle: Best fit
// takes Fits you descending then job id (a vacancy without a Fits you yet counts as the midpoint);
// Newest takes the Inbox Newest key (NewestKey) then job id. The decision depends on the stored
// state and the configured order alone, so a run that was interrupted resumes where it stopped.
func PlanWork(rows []store.ModelWorkRow, p PlanParams) ([]WorkItem, PlanStats) {
	var stats PlanStats
	type ranked struct {
		item   WorkItem
		fits   float64
		newest NewestKey
	}
	var list []ranked
	newest := p.Order == config.OrderNewest
	now := p.Now
	if now.IsZero() {
		now = time.Now()
	}
	for _, r := range rows {
		if jobextract.SourceStateOf(r.Source) != matchcontract.SourceUsable {
			continue
		}
		stats.Usable++
		hash := matchcontract.HashDescription(r.Source.Description)
		fits := matchcontract.DefaultUnknownGoalUtility
		if r.FitsRaw != nil {
			fits = *r.FitsRaw
		}
		extractionCurrent := r.HasExtraction && r.ExtractionSource == hash && r.ExtractorVersion == p.ExtractorVersion

		var kind WorkKind
		switch {
		case !extractionCurrent:
			kind = WorkExtract
		case r.RequirementCount == 0:
			stats.Extracted++
			stats.NoRequirements++
			continue
		default:
			stats.Extracted++
			stats.WithRequires++
			if !p.Matching {
				continue
			}
			if r.HasMatches && r.MatcherVersion == p.MatcherVersion && r.ProfileRevision == p.ProfileRevision {
				stats.Matched++
				continue
			}
			kind = WorkMatch
		}
		if failed, ok := p.Skip[r.Source.JobID]; ok && failed == hash {
			stats.Cooling++
			continue
		}
		if kind == WorkExtract {
			stats.PendingExtract++
		} else {
			stats.PendingMatch++
		}
		it := ranked{item: WorkItem{Source: r.Source, Kind: kind}, fits: fits}
		if newest {
			it.newest = NewestKeyOf(r.PublishedAt, r.PublishedBasis, now)
		}
		list = append(list, it)
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if newest {
			if a.newest.Before(b.newest) {
				return true
			}
			if b.newest.Before(a.newest) {
				return false
			}
		} else if a.fits != b.fits {
			return a.fits > b.fits
		}
		return a.item.Source.JobID < b.item.Source.JobID
	})
	out := make([]WorkItem, 0, len(list))
	for _, r := range list {
		out = append(out, r.item)
	}
	return out, stats
}

// startOfLocalDay returns the store-format UTC timestamp of the start of the local calendar day
// containing t. The daily limit counts from it.
func startOfLocalDay(t time.Time) string {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).UTC().Format("2006-01-02T15:04:05.000Z")
}

// storeTime formats t the way the store writes timestamps.
func storeTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }
