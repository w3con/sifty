package profile

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/w3con/sifty/internal/matchcontract"
)

// Facts computed by code from the experience entries, never by the model. A model reading "CTO,
// 2019 to present" cannot be trusted to subtract dates, so the arithmetic is done here and the
// matcher receives the result as ordinary facts it can cite.
const (
	MatchFieldExperienceTotal      = "experienceTotal"
	MatchFieldExperienceLeadership = "experienceLeadership"
	MatchFieldExperienceFounder    = "experienceFounder"
	// MatchFieldExperienceCurrent says there is a current role, or one that ended within the last
	// year, and so that the owner has worked in the last year.
	MatchFieldExperienceCurrent = "experienceCurrent"
	// MatchFieldExperienceEngineering counts the years in roles titled engineer, developer,
	// architect or programmer.
	MatchFieldExperienceEngineering = "experienceEngineering"
)

// IsComputedField reports whether a match fact field names a fact computed by code from the
// experience entries (as opposed to one the owner wrote in the profile).
func IsComputedField(field string) bool {
	switch field {
	case MatchFieldExperienceTotal, MatchFieldExperienceLeadership, MatchFieldExperienceFounder,
		MatchFieldExperienceCurrent, MatchFieldExperienceEngineering:
		return true
	}
	return false
}

var (
	leadershipTitleRe = regexp.MustCompile(`(?i)\b(cto|ceo|coo|cio|vp|vice[- ]president|head|director|lead|manager|chief|founder|co-?founder)\b`)
	founderTitleRe    = regexp.MustCompile(`(?i)\b(founder|co-?founder|founding)\b`)
	engineerTitleRe   = regexp.MustCompile(`(?i)\b(engineer|engineers|engineering|developer|architect|programmer)\b`)
	isoMonthRe        = regexp.MustCompile(`^(\d{4})[-/.](\d{1,2})(?:[-/.]\d{1,2})?$`)
	monthYearRe       = regexp.MustCompile(`^(\d{1,2})[-/.](\d{4})$`)
	yearRe            = regexp.MustCompile(`^(\d{4})$`)
	monthNameYearRe   = regexp.MustCompile(`(?i)^([a-z]{3,9})\.?\s+(\d{4})$`)
)

var monthNames = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// monthIndex reads a profile date as a month count (year*12 + month-1). A bare year stands for
// January when it starts a role and for December when it ends one. It reports false for a date it
// cannot read, so an unreadable entry adds no years instead of guessed ones.
func monthIndex(s string, isEnd bool) (int, bool) {
	s = strings.TrimSpace(s)
	year, month := 0, 0
	switch {
	case isoMonthRe.MatchString(s):
		m := isoMonthRe.FindStringSubmatch(s)
		year, _ = strconv.Atoi(m[1])
		month, _ = strconv.Atoi(m[2])
	case monthYearRe.MatchString(s):
		m := monthYearRe.FindStringSubmatch(s)
		month, _ = strconv.Atoi(m[1])
		year, _ = strconv.Atoi(m[2])
	case yearRe.MatchString(s):
		year, _ = strconv.Atoi(s)
		month = 1
		if isEnd {
			month = 12
		}
	case monthNameYearRe.MatchString(s):
		m := monthNameYearRe.FindStringSubmatch(s)
		name := strings.ToLower(m[1])
		if len(name) > 3 {
			name = name[:3]
		}
		month = monthNames[name]
		year, _ = strconv.Atoi(m[2])
	}
	if year < 1950 || month < 1 || month > 12 {
		return 0, false
	}
	return year*12 + month - 1, true
}

type monthSpan struct{ from, to int } // inclusive month indexes

func (e Experience) span(now time.Time) (monthSpan, bool) {
	from, ok := monthIndex(e.Start, false)
	if !ok {
		return monthSpan{}, false
	}
	var to int
	if e.Current || strings.TrimSpace(e.End) == "" {
		if !e.Current {
			return monthSpan{}, false
		}
		to = now.Year()*12 + int(now.Month()) - 1
	} else if to, ok = monthIndex(e.End, true); !ok {
		return monthSpan{}, false
	}
	if to < from {
		return monthSpan{}, false
	}
	return monthSpan{from, to}, true
}

// coveredMonths counts the distinct months the spans cover, so overlapping roles count once.
func coveredMonths(spans []monthSpan) int {
	sort.Slice(spans, func(i, j int) bool { return spans[i].from < spans[j].from })
	total, end := 0, -1
	for _, s := range spans {
		from := s.from
		if from <= end {
			from = end + 1
		}
		if s.to >= from {
			total += s.to - from + 1
		}
		if s.to > end {
			end = s.to
		}
	}
	return total
}

func yearsWord(years int) string {
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
}

// derivedExperienceFacts computes total years, years in leading roles and the founder flag.
// Years are whole years, rounded down. Nothing is emitted when the entries give no readable
// dates, and the founder fact is emitted only when a title says so: its absence means unknown,
// never "no".
func derivedExperienceFacts(experience []Experience, now time.Time) []matchcontract.ProfileFact {
	var all, lead, engineering []monthSpan
	var leadTitles, founderTitles, engineeringTitles, recentRoles []string
	nowIndex := now.Year()*12 + int(now.Month()) - 1
	describe := func(e Experience) string {
		title, company := strings.TrimSpace(e.Title), strings.TrimSpace(e.Company)
		if company == "" {
			return title
		}
		return title + " at " + company
	}
	for _, e := range experience {
		title := strings.TrimSpace(e.Title)
		span, dated := e.span(now)
		if dated {
			all = append(all, span)
		}
		if leadershipTitleRe.MatchString(title) {
			if dated {
				lead = append(lead, span)
			}
			leadTitles = append(leadTitles, describe(e))
		}
		if founderTitleRe.MatchString(title) {
			founderTitles = append(founderTitles, describe(e))
		}
		if engineerTitleRe.MatchString(title) {
			if dated {
				engineering = append(engineering, span)
			}
			engineeringTitles = append(engineeringTitles, describe(e))
		}
		// A role counts as recent when it is current or its span reaches into the last twelve
		// months: "worked in the last year" follows from the dates alone.
		if dated && span.to >= nowIndex-12 {
			recent := describe(e)
			if e.Current {
				recent += " (current)"
			}
			recentRoles = append(recentRoles, recent)
		}
	}
	var out []matchcontract.ProfileFact
	if years := coveredMonths(all) / 12; years >= 1 {
		out = append(out, matchcontract.ProfileFact{
			Field: MatchFieldExperienceTotal,
			Value: yearsWord(years) + " of work experience in total, computed from the dates of the experience entries",
		})
	}
	if years := coveredMonths(lead) / 12; years >= 1 {
		out = append(out, matchcontract.ProfileFact{
			Field: MatchFieldExperienceLeadership,
			Value: yearsWord(years) + " in leadership roles, computed from the dates of: " + strings.Join(leadTitles, "; "),
		})
	}
	if years := coveredMonths(engineering) / 12; years >= 1 {
		out = append(out, matchcontract.ProfileFact{
			Field: MatchFieldExperienceEngineering,
			Value: yearsWord(years) + " in engineering roles (titles with engineer, developer, architect or programmer), computed from the dates of: " + strings.Join(engineeringTitles, "; "),
		})
	}
	if len(recentRoles) > 0 {
		out = append(out, matchcontract.ProfileFact{
			Field: MatchFieldExperienceCurrent,
			Value: "Has worked within the last year (a current role or one that ended in the last 12 months), computed from the dates of: " + strings.Join(recentRoles, "; "),
		})
	}
	if len(founderTitles) > 0 {
		out = append(out, matchcontract.ProfileFact{
			Field: MatchFieldExperienceFounder,
			Value: "Founded or co-founded a company: " + strings.Join(founderTitles, "; "),
		})
	}
	return out
}
