package model

import (
	"strings"
	"time"
)

// publicationLayouts are the source date shapes the collectors receive in
// practice: ISO 8601 with or without a zone, date-only values, RFC/feed dates,
// and a space-separated timestamp followed by a zone. Update-only timestamps
// are not publication dates and are therefore never accepted here.
var publicationLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999999999 Z07:00",
	"2006-01-02 15:04:05 Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05Z0700",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05",
	"2006-01-02",
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC850,
	time.ANSIC,
}

// NormalizePublishedAt returns a canonical UTC publication timestamp when raw
// is a genuine, non-future source publication value. Invalid, empty and future
// values return "" so the caller stores an honest unknown instead of a false
// publication date. Callers must not pass source updated_at/fetch time here.
func NormalizePublishedAt(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	for _, layout := range publicationLayouts {
		t, err := time.Parse(layout, s)
		if err != nil {
			continue
		}
		if t.After(time.Now().UTC()) {
			return ""
		}
		// A date-only source value stays date-only: forcing it to UTC midnight
		// would display the previous calendar day in western time zones.
		if layout == "2006-01-02" {
			return t.Format("2006-01-02")
		}
		return t.UTC().Format(time.RFC3339)
	}
	return ""
}

// PublicationBasis returns the source field name only when a real date is
// present, so an empty date never carries a claimed origin.
func PublicationBasis(date, basis string) string {
	if date == "" {
		return ""
	}
	return basis
}
