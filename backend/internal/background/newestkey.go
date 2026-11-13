package background

import (
	"strings"
	"time"
)

// NewestKey is the Inbox "Newest" sort key of one vacancy. It is the server's copy of the key in
// frontend/filters.ts (sortCatalogue, sort "newest") and frontend/dates.ts (publicationDate); the
// two are checked against one shared fixture (testdata/newest-key.json) and defined once in
// docs/specs/quick-match.md section 14.3.
//
// Rank 0: a publication date with a recorded basis. Rank 1: a date without one. Rank 2: no usable
// date. Within ranks 0 and 1 the later date sorts first; equal keys fall back to the job id.
type NewestKey struct {
	Rank   int
	Millis int64
}

// NewestKeyOf computes the key from a stored published_at and published_basis. A date that is
// empty, invalid, impossible (2026-02-31) or later than now counts as no date.
func NewestKeyOf(publishedAt, basis string, now time.Time) NewestKey {
	t, ok := publicationDate(publishedAt, now)
	if !ok {
		return NewestKey{Rank: 2}
	}
	rank := 1
	if basis != "" {
		rank = 0
	}
	return NewestKey{Rank: rank, Millis: t.UnixMilli()}
}

// Before reports whether a vacancy with key k sorts ahead of one with key o in the Newest order;
// equal keys are left to the caller's job-id tie-breaker.
func (k NewestKey) Before(o NewestKey) bool {
	if k.Rank != o.Rank {
		return k.Rank < o.Rank
	}
	if k.Rank < 2 {
		return k.Millis > o.Millis
	}
	return false
}

// Layouts of the date strings the sources store today. A zone-less date-time is read in the
// machine's local zone, as the browser reads it; a zone name other than UTC or GMT is not read.
var publicationLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05 MST",
	time.RFC1123Z,
	time.RFC1123,
}

func publicationDate(raw string, now time.Time) (time.Time, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, !t.After(now)
	}
	for _, layout := range publicationLayouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err != nil {
			continue
		}
		if name, _ := t.Zone(); strings.HasSuffix(layout, "MST") && name != "UTC" && name != "GMT" {
			continue
		}
		return t, !t.After(now)
	}
	return time.Time{}, false
}
