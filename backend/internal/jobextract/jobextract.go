// Package jobextract turns one retained vacancy snapshot into validated Quick Match facts.
// It is deterministic, performs no I/O, calls no model, and imports only the shared
// matchcontract package plus the Go standard library. It does not import the evaluator,
// store, provider, site-specific parsers or presentation code.
package jobextract

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
)

// DefaultSourcePolicyVersion identifies the built-in policy when a caller supplies no version.
const DefaultSourcePolicyVersion = "source-policy/1"

// Policy is the injected extractor policy. It is data, not a preference coefficient, and the
// evaluator never receives it.
type Policy struct {
	Source matchcontract.SourcePolicy
}

var bulletRe = regexp.MustCompile(`^(?:[-*•◦▪]|\d+[.)])\s+(.+)$`)
var amountRe = regexp.MustCompile(`(\d{1,3}(?:[,\s]\d{3})+|\d+(?:\.\d+)?)\s*([kK])?`)
var currencyCodeRe = regexp.MustCompile(`(?i)\b(USD|EUR|GBP|CAD|AUD|CHF)\b`)
var weeklyHoursRe = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:h|hr|hrs|hours)\s*(?:/|per)?\s*week`)

var requiredHeadings = map[string]bool{
	"requirements": true, "required": true, "required skills": true,
	"must have": true, "must-have": true, "minimum requirements": true,
	"minimum qualifications": true, "what you need": true, "what you'll need": true,
	"qualifications": true,
}
var preferredHeadings = map[string]bool{
	"preferred": true, "preferred skills": true, "nice to have": true,
	"nice-to-have": true, "bonus": true, "bonus points": true, "desired": true,
	"good to have": true, "plus": true,
}
var companyHeadings = map[string]bool{
	"about us": true, "about the company": true, "the company": true, "company": true,
	"our stack": true, "our tech stack": true, "tech stack": true, "why us": true,
}

var salaryLabels = map[string]bool{
	"salary": true, "base salary": true, "base pay": true, "cash": true,
	"cash compensation": true, "compensation": true, "annual salary": true,
	"salary range": true, "pay range": true, "total cash": true,
}

var hiringLocationLabels = []string{
	"hiring location", "hiring locations", "permitted location", "permitted locations",
	"permitted hiring location", "permitted hiring locations",
	"eligible location", "eligible locations", "eligible hiring locations",
	"location eligibility", "locations we hire in", "location requirements", "work eligibility",
}

var genericLocationLabels = []string{"location", "locations", "work location"}

// Extract prepares facts from retained source fields and explicit retained headings/labels.
// It never reconstructs structure from flattened prose: without an explicit section heading a
// bullet stays unknown and is recorded as an ambiguity, not as a requirement.
func Extract(src matchcontract.RetainedSource, policy Policy) (matchcontract.VacancyFacts, error) {
	if err := src.Validate(); err != nil {
		return matchcontract.VacancyFacts{}, fmt.Errorf("retained source: %w", err)
	}
	rawHash, err := matchcontract.HashRetainedSource(src)
	if err != nil {
		return matchcontract.VacancyFacts{}, err
	}
	policyVersion := strings.TrimSpace(policy.Source.Version)
	if policyVersion == "" {
		policyVersion = DefaultSourcePolicyVersion
	}

	f := matchcontract.VacancyFacts{
		SchemaVersion:       matchcontract.SchemaVersion,
		ExtractorVersion:    matchcontract.ExtractorVersion,
		SourcePolicyVersion: policyVersion,
		JobID:               src.JobID,
		RawSourceHash:       rawHash,
	}
	f.Role.Title = &matchcontract.TitleFact{
		Value: src.Title,
		Evidence: matchcontract.Evidence{
			Field: "title", Section: matchcontract.SectionUnknown,
			Reliability: matchcontract.ReliabilityUsable,
		},
	}

	applyStructured(&f, src)
	parseText(&f, src, policy)

	sortFacts(&f)
	if err := f.SetFactsHash(); err != nil {
		return matchcontract.VacancyFacts{}, fmt.Errorf("extracted facts: %w", err)
	}
	return f, nil
}

func applyStructured(f *matchcontract.VacancyFacts, src matchcontract.RetainedSource) {
	s := src.Structured
	if s == nil {
		return
	}
	if s.Cash != nil && s.Cash.Max != nil {
		rel := matchcontract.ReliabilityUsable
		if s.Cash.Currency == "" || s.Cash.Period == "" || s.Cash.TaxBasis == "" {
			rel = matchcontract.ReliabilityNeedsCorroboration
		}
		f.Compensation.Cash = &matchcontract.CashFact{
			Min: s.Cash.Min, Max: s.Cash.Max, Currency: s.Cash.Currency,
			Period: s.Cash.Period, TaxBasis: s.Cash.TaxBasis,
			Basis: cashBasis(s.Cash.Min, s.Cash.Max),
			Evidence: matchcontract.Evidence{
				Field: "structured.cash", Section: matchcontract.SectionUnknown, Reliability: rel,
			},
		}
	}
	for _, m := range s.WorkModes {
		f.Conditions.WorkModes = append(f.Conditions.WorkModes, matchcontract.WorkModeFact{
			Mode: m, Evidence: fieldEvidence("structured.workModes", matchcontract.ReliabilityUsable),
		})
	}
	if len(s.EngagementTypes) > 0 {
		f.Conditions.Engagement = &matchcontract.EngagementFact{
			Type: s.EngagementTypes[0], Evidence: fieldEvidence("structured.engagementTypes", matchcontract.ReliabilityUsable),
		}
	}
	if s.Hours != nil {
		f.Conditions.Hours = &matchcontract.HoursFact{
			Mode: s.Hours.Mode, Weekly: s.Hours.Weekly, WeekMin: s.Hours.WeekMin, WeekMax: s.Hours.WeekMax,
			Evidence: fieldEvidence("structured.hours", matchcontract.ReliabilityUsable),
		}
	}
	if strings.TrimSpace(s.WorkAuthorization) != "" {
		f.Geography.WorkAuthorization = &matchcontract.AuthorizationFact{
			Requirement: s.WorkAuthorization,
			Evidence:    fieldEvidence("structured.workAuthorization", matchcontract.ReliabilityUsable),
		}
	}
	if strings.TrimSpace(s.Timezone) != "" {
		f.Conditions.Timezone = &matchcontract.TimezoneFact{
			Value:    strings.TrimSpace(s.Timezone),
			Evidence: fieldEvidence("structured.timezone", matchcontract.ReliabilityUsable),
		}
	}
	for _, label := range s.HiringLocations {
		if strings.TrimSpace(label) == "" {
			continue
		}
		f.Geography.HiringLocations = append(f.Geography.HiringLocations, matchcontract.LocationFact{
			Label: label,
			Evidence: matchcontract.Evidence{
				Field: "structured.hiringLocations", Section: matchcontract.SectionUnknown,
				Reliability: matchcontract.ReliabilityUsable,
			},
		})
	}
}

func parseText(f *matchcontract.VacancyFacts, src matchcontract.RetainedSource, policy Policy) {
	section := matchcontract.SectionUnknown
	hadRequirementHeading := false
	unknownBullets := false
	var boardLocations []matchcontract.LocationFact
	var textLocations []matchcontract.LocationFact
	var textLocationQuote string
	var ambiguity *matchcontract.Ambiguity

	addBoardLocations(f, src, policy, &boardLocations)

	lines := strings.Split(strings.ReplaceAll(src.Description, "\r\n", "\n"), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if bulletRe.MatchString(line) {
			item := strings.TrimSpace(bulletRe.FindStringSubmatch(line)[1])
			switch section {
			case matchcontract.SectionRequired, matchcontract.SectionPreferred:
				f.Professional.Requirements = append(f.Professional.Requirements, requirement(item, line, section))
			case matchcontract.SectionCompanyContext:
				f.Professional.ContextMentions = append(f.Professional.ContextMentions, matchcontract.ContextMention{
					Label: item, Evidence: textEvidence(line, section, matchcontract.ReliabilityUsable),
				})
			default:
				unknownBullets = true
			}
			continue
		}
		if s, rest, ok := heading(line); ok {
			section = s
			if s == matchcontract.SectionRequired || s == matchcontract.SectionPreferred {
				hadRequirementHeading = true
			}
			if strings.TrimSpace(rest) != "" {
				addInlineItems(f, rest, line, s)
			}
			continue
		}
		if looksLikeHeading(line) {
			// An explicit but unrecognised heading ends any known requirement/company
			// section. Its content stays unknown; flattened prose is never re-sectioned.
			section = matchcontract.SectionUnknown
		}
		if value, ok := salaryLabelValue(line); ok {
			if f.Compensation.Cash == nil {
				if cash, ok := cashText(value, line); ok {
					f.Compensation.Cash = cash
				}
			}
			continue
		}
		if value := labelValue(line, hiringLocationLabels...); value != "" {
			textLocationQuote = line
			for _, loc := range splitLocations(value) {
				if mode, ok := workModeToken(loc); ok {
					addWorkMode(f, mode, textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable))
					continue
				}
				textLocations = append(textLocations, matchcontract.LocationFact{
					Label: loc, Evidence: textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
				})
			}
			continue
		}
		if value := labelValue(line, genericLocationLabels...); value != "" {
			textLocationQuote = line
			for _, loc := range splitLocations(value) {
				if mode, ok := workModeToken(loc); ok {
					addWorkMode(f, mode, textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable))
					continue
				}
				textLocations = append(textLocations, matchcontract.LocationFact{
					Label: loc, Evidence: textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityNeedsCorroboration),
				})
			}
			continue
		}
		if value := labelValue(line, "work mode", "workplace", "working mode"); value != "" {
			if mode, ok := workModeToken(value); ok {
				addWorkMode(f, mode, textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable))
			}
			continue
		}
		if value := labelValue(line, "employment type", "engagement", "contract type"); value != "" {
			if typ, ok := engagementToken(value); ok && f.Conditions.Engagement == nil {
				f.Conditions.Engagement = &matchcontract.EngagementFact{
					Type: typ, Evidence: textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
				}
			}
			continue
		}
		if value := labelValue(line, "hours", "working hours", "time commitment"); value != "" {
			addHours(f, value, line)
			continue
		}
		if value := labelValue(line, "timezone", "time zone"); value != "" {
			if f.Conditions.Timezone == nil {
				f.Conditions.Timezone = &matchcontract.TimezoneFact{
					Value: value, Evidence: textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
				}
			}
			continue
		}
		if value := labelValue(line, "work authorization", "authorization", "right to work", "visa"); value != "" {
			if auth, ok := authorizationText(value, line); ok && f.Geography.WorkAuthorization == nil {
				f.Geography.WorkAuthorization = auth
			}
			continue
		}
		if isWorkAuthorizationStatement(line) && f.Geography.WorkAuthorization == nil {
			f.Geography.WorkAuthorization = authorizationFromStatement(line)
			continue
		}
		if value := labelValue(line, "our stack", "tech stack", "stack"); value != "" {
			for _, item := range splitItems(value) {
				f.Professional.ContextMentions = append(f.Professional.ContextMentions, matchcontract.ContextMention{
					Label: item, Evidence: textEvidence(line, matchcontract.SectionCompanyContext, matchcontract.ReliabilityUsable),
				})
			}
		}
	}

	if !hadRequirementHeading && strings.TrimSpace(src.Description) != "" {
		detail := "no retained required/preferred heading; requirements were not inferred"
		if unknownBullets {
			detail = "bullets have no retained required/preferred heading; requirements were not inferred"
		}
		ambiguity = &matchcontract.Ambiguity{
			Group: matchcontract.GroupProfessional, Field: "description", Kind: "missing_structure", Detail: detail,
			Evidence: matchcontract.Evidence{
				Field: "description", Section: matchcontract.SectionUnknown,
				Reliability: matchcontract.ReliabilityNeedsCorroboration,
			},
		}
		f.Ambiguities = append(f.Ambiguities, *ambiguity)
	}

	mergeLocations(f, boardLocations, textLocations, textLocationQuote)
}

func addBoardLocations(f *matchcontract.VacancyFacts, src matchcontract.RetainedSource, policy Policy, out *[]matchcontract.LocationFact) {
	reliability := policy.Source.LocationReliability(src.Sources)
	if strings.TrimSpace(src.Location) == "" {
		return
	}
	for _, loc := range splitLocations(src.Location) {
		if mode, ok := workModeToken(loc); ok {
			addWorkMode(f, mode, fieldEvidence("location", reliability))
			continue
		}
		if reliability == matchcontract.ReliabilityUnusable {
			continue
		}
		*out = append(*out, matchcontract.LocationFact{
			Label: loc,
			Evidence: matchcontract.Evidence{
				Field: "location", Section: matchcontract.SectionUnknown, Reliability: reliability,
			},
		})
	}
}

func mergeLocations(f *matchcontract.VacancyFacts, board, text []matchcontract.LocationFact, textQuote string) {
	if len(text) == 0 {
		f.Geography.HiringLocations = append(f.Geography.HiringLocations, board...)
		return
	}
	if len(board) == 0 {
		f.Geography.HiringLocations = append(f.Geography.HiringLocations, text...)
		return
	}
	for _, b := range board {
		matched := false
		for _, t := range text {
			if locationsAgree(b.Label, t.Label) {
				matched = true
				if t.Evidence.Reliability == matchcontract.ReliabilityUsable {
					b.Evidence.Reliability = matchcontract.ReliabilityUsable
				}
				break
			}
		}
		if matched {
			f.Geography.HiringLocations = append(f.Geography.HiringLocations, b)
			continue
		}
		// Board and text disagree. A generic location statement never upgrades the board,
		// and neither statement is silently treated as a confirmed hiring restriction.
		if b.Evidence.Reliability == matchcontract.ReliabilityUsable {
			b.Evidence.Reliability = matchcontract.ReliabilityNeedsCorroboration
		}
		f.Geography.HiringLocations = append(f.Geography.HiringLocations, b)
		f.Ambiguities = append(f.Ambiguities, matchcontract.Ambiguity{
			Group: matchcontract.GroupGeography, Field: "location", Kind: "source_conflict",
			Detail: "board field and job text disagree; needs corroboration",
			Evidence: matchcontract.Evidence{
				Quote: textQuote, Section: matchcontract.SectionUnknown,
				Reliability: matchcontract.ReliabilityNeedsCorroboration,
			},
		})
	}
	for _, t := range text {
		agreed := false
		for _, b := range board {
			if locationsAgree(b.Label, t.Label) {
				agreed = true
				break
			}
		}
		if agreed {
			continue
		}
		// A board/text disagreement makes both sides at most needs-corroboration.
		t.Evidence.Reliability = matchcontract.ReliabilityNeedsCorroboration
		f.Geography.HiringLocations = append(f.Geography.HiringLocations, t)
	}
}

func heading(line string) (matchcontract.Section, string, bool) {
	trimmed := strings.TrimSpace(line)
	if bulletRe.MatchString(trimmed) || len(trimmed) > 80 {
		return "", "", false
	}
	head, rest := trimmed, ""
	if i := strings.Index(trimmed, ":"); i >= 0 {
		head, rest = strings.TrimSpace(trimmed[:i]), strings.TrimSpace(trimmed[i+1:])
	}
	key := strings.ToLower(strings.Join(strings.Fields(head), " "))
	switch {
	case requiredHeadings[key]:
		return matchcontract.SectionRequired, rest, true
	case preferredHeadings[key]:
		return matchcontract.SectionPreferred, rest, true
	case companyHeadings[key]:
		return matchcontract.SectionCompanyContext, rest, true
	}
	return "", "", false
}

func looksLikeHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if bulletRe.MatchString(trimmed) {
		return false
	}
	if strings.HasPrefix(trimmed, "#") {
		return true
	}
	if i := strings.Index(trimmed, ":"); i > 0 && i <= 40 {
		return true
	}
	if len(trimmed) > 60 || strings.ContainsAny(trimmed, ".!?") {
		return false
	}
	words := strings.Fields(trimmed)
	return len(words) >= 1 && len(words) <= 6 && strings.ToUpper(trimmed[:1]) == trimmed[:1]
}

func salaryLabelValue(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	i := strings.Index(trimmed, ":")
	if i < 0 {
		return "", false
	}
	key := strings.ToLower(strings.Join(strings.Fields(trimmed[:i]), " "))
	if !salaryLabels[key] {
		return "", false
	}
	return strings.TrimSpace(trimmed[i+1:]), true
}

func addInlineItems(f *matchcontract.VacancyFacts, value, line string, section matchcontract.Section) {
	switch section {
	case matchcontract.SectionRequired, matchcontract.SectionPreferred:
		for _, item := range splitItems(value) {
			f.Professional.Requirements = append(f.Professional.Requirements, requirement(item, line, section))
		}
	case matchcontract.SectionCompanyContext:
		for _, item := range splitItems(value) {
			f.Professional.ContextMentions = append(f.Professional.ContextMentions, matchcontract.ContextMention{
				Label: item, Evidence: textEvidence(line, section, matchcontract.ReliabilityUsable),
			})
		}
	}
}

func requirement(label, quote string, section matchcontract.Section) matchcontract.RequirementFact {
	importance := matchcontract.ImportancePreferred
	if section == matchcontract.SectionRequired {
		importance = matchcontract.ImportanceRequired
	}
	return matchcontract.RequirementFact{
		Label: label, Text: quote, Importance: importance,
		Evidence: textEvidence(quote, section, matchcontract.ReliabilityUsable),
	}
}

// cashSkipRe matches whole-word equity/bonus markers in a compensation quote. Word boundaries
// prevent "ote" from matching "remote", "stock" alone from blocking "stock grants", etc.
var cashSkipRe = regexp.MustCompile(`(?i)\b(bonus|equity|stock[- ]option|commission|estimate|survey|ote|allowance|relocation)\b`)

func cashText(value, line string) (*matchcontract.CashFact, bool) {
	lower := strings.ToLower(line)
	// Only skip if the quote is NOT about a salary/base/wage (equity-grant context check).
	hasSalaryCtx := strings.Contains(lower, "salary") || strings.Contains(lower, " base ") || strings.Contains(lower, "wage")
	if !hasSalaryCtx && cashSkipRe.MatchString(lower) {
		return nil, false
	}
	// Detect currency from explicit ISO code first, then from symbol in value, then in line.
	// Mirrors parseCashText in vacancyview/money.go — both must use the same detection order.
	code := ""
	if m := currencyCodeRe.FindString(value); m != "" {
		code = strings.ToUpper(m)
	} else if strings.ContainsRune(value, '$') {
		code = "USD"
	} else if strings.ContainsRune(value, '€') {
		code = "EUR"
	} else if strings.ContainsRune(value, '£') {
		code = "GBP"
	} else if m := currencyCodeRe.FindString(line); m != "" {
		code = strings.ToUpper(m)
	} else if strings.ContainsRune(line, '$') {
		code = "USD"
	} else if strings.ContainsRune(line, '€') {
		code = "EUR"
	} else if strings.ContainsRune(line, '£') {
		code = "GBP"
	}
	matches := amountRe.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return nil, false
	}
	type amount struct {
		value float64
		start int
		end   int
	}
	var amounts []amount
	for _, m := range matches {
		text := strings.ReplaceAll(strings.ReplaceAll(value[m[2]:m[3]], ",", ""), " ", "")
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, false
		}
		if m[4] >= 0 && m[5] >= 0 && strings.EqualFold(value[m[4]:m[5]], "k") {
			n *= 1000
		}
		amounts = append(amounts, amount{value: n, start: m[0], end: m[1]})
	}
	// When more than two amounts are present (multiple city/location ranges), take the global
	// min across all lowers and the global max across all uppers as the overall range.
	if len(amounts) > 2 {
		if len(amounts)%2 != 0 {
			return nil, false // odd count — unclear structure
		}
		globalMin := amounts[0].value
		globalMax := amounts[0].value
		for _, a := range amounts[1:] {
			if a.value < globalMin {
				globalMin = a.value
			}
			if a.value > globalMax {
				globalMax = a.value
			}
		}
		if globalMin >= 1000 && globalMax <= 10_000_000 {
			period := cashPeriod(lower)
			tax := ""
			if strings.Contains(lower, "gross") {
				tax = matchcontract.TaxGross
			}
			if strings.Contains(lower, "net") && !strings.Contains(lower, "gross") {
				tax = matchcontract.TaxNet
			}
			usable := code != "" && period != "" && tax != ""
			reliability := matchcontract.ReliabilityNeedsCorroboration
			if usable {
				reliability = matchcontract.ReliabilityUsable
			}
			cash := &matchcontract.CashFact{
				Min: &globalMin, Max: &globalMax,
				Currency: code, Period: period, TaxBasis: tax,
				Basis: matchcontract.CashBasisRangeCeiling,
				Evidence: matchcontract.Evidence{
					Field: "description", Quote: line,
					Section: matchcontract.SectionUnknown, Reliability: reliability,
				},
			}
			return cash, true
		}
		return nil, false
	}
	rangePair := false
	if len(amounts) == 2 {
		// Strip currency symbols and whitespace so "$230,000-$280,000" parses the same as
		// "$230,000 - $280,000" or "USD 230,000 to 280,000" (mirrors parseCashText in money.go).
		rawSep := value[amounts[0].end:amounts[1].start]
		between := strings.ToLower(strings.TrimFunc(rawSep, func(r rune) bool {
			return r == '$' || r == '€' || r == '£' || r == ' ' || r == '\t' || r == ','
		}))
		switch between {
		case "-", "–", "—", "to":
			rangePair = true
		default:
			return nil, false
		}
	}
	period := cashPeriod(lower)
	tax := ""
	if strings.Contains(lower, "gross") {
		tax = matchcontract.TaxGross
	}
	if strings.Contains(lower, "net") && !strings.Contains(lower, "gross") {
		tax = matchcontract.TaxNet
	}
	floorOnly := strings.Contains(lower, " from ") || strings.HasPrefix(lower, "from ") ||
		strings.Contains(lower, "starting at") || strings.Contains(lower, "starting from") ||
		strings.Contains(lower, "minimum") || strings.Contains(lower, "at least") || strings.Contains(lower, ">=")
	ceilingOnly := strings.Contains(lower, "up to") || strings.Contains(lower, "maximum") ||
		strings.Contains(lower, "no more than") || strings.Contains(lower, "not exceeding") ||
		strings.Contains(lower, "ceiling")
	cash := &matchcontract.CashFact{Currency: code, Period: period, TaxBasis: tax}
	switch {
	case rangePair:
		min, max := amounts[0].value, amounts[1].value
		if min > max {
			min, max = max, min
		}
		cash.Min, cash.Max = &min, &max
		cash.Basis = matchcontract.CashBasisRangeCeiling
	case floorOnly && len(amounts) == 1:
		// A floor alone is not a ceiling: never invent one.
		min := amounts[0].value
		cash.Min = &min
		cash.Basis = matchcontract.CashBasisStated
	case ceilingOnly && len(amounts) == 1:
		max := amounts[0].value
		cash.Max = &max
		cash.Basis = matchcontract.CashBasisCeiling
	case len(amounts) == 1:
		max := amounts[0].value
		cash.Max = &max
		cash.Basis = matchcontract.CashBasisStated
	default:
		return nil, false
	}
	usable := cash.Basis == matchcontract.CashBasisRangeCeiling || cash.Basis == matchcontract.CashBasisCeiling || cash.Basis == matchcontract.CashBasisStated
	usable = usable && cash.Max != nil && code != "" && period != "" && tax != ""
	reliability := matchcontract.ReliabilityNeedsCorroboration
	if usable {
		reliability = matchcontract.ReliabilityUsable
	}
	cash.Evidence = matchcontract.Evidence{
		Field: "description", Quote: line, Section: matchcontract.SectionUnknown, Reliability: reliability,
	}
	return cash, true
}

func cashPeriod(lower string) string {
	switch {
	case strings.Contains(lower, "per year") || strings.Contains(lower, "/year") ||
		strings.Contains(lower, "annual") || strings.Contains(lower, "p.a."):
		return matchcontract.PeriodYear
	case strings.Contains(lower, "per month") || strings.Contains(lower, "/month") || strings.Contains(lower, "monthly"):
		return matchcontract.PeriodMonth
	case strings.Contains(lower, "per day") || strings.Contains(lower, "/day") || strings.Contains(lower, "daily"):
		return matchcontract.PeriodDay
	case strings.Contains(lower, "per hour") || strings.Contains(lower, "/hour") || strings.Contains(lower, "hourly"):
		return matchcontract.PeriodHour
	}
	return ""
}

func cashBasis(min, max *float64) string {
	if min != nil && max != nil {
		return matchcontract.CashBasisRangeCeiling
	}
	if max != nil {
		return matchcontract.CashBasisCeiling
	}
	return matchcontract.CashBasisStated
}

func addWorkMode(f *matchcontract.VacancyFacts, mode string, evidence matchcontract.Evidence) {
	if evidence.Reliability == matchcontract.ReliabilityUnusable {
		return
	}
	for i, existing := range f.Conditions.WorkModes {
		if existing.Mode != mode {
			continue
		}
		if existing.Evidence.Reliability != matchcontract.ReliabilityUsable &&
			evidence.Reliability == matchcontract.ReliabilityUsable {
			f.Conditions.WorkModes[i].Evidence = evidence
		}
		return
	}
	f.Conditions.WorkModes = append(f.Conditions.WorkModes, matchcontract.WorkModeFact{Mode: mode, Evidence: evidence})
}

func addHours(f *matchcontract.VacancyFacts, value, line string) {
	if f.Conditions.Hours != nil {
		return
	}
	lower := strings.ToLower(value)
	h := &matchcontract.HoursFact{Evidence: textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable)}
	switch {
	case strings.Contains(lower, "full-time") || strings.Contains(lower, "full time"):
		h.Mode = matchcontract.HoursFullTime
	case strings.Contains(lower, "part-time") || strings.Contains(lower, "part time"):
		h.Mode = matchcontract.HoursPartTime
	}
	if m := weeklyHoursRe.FindStringSubmatch(value); m != nil {
		if n, err := strconv.ParseFloat(m[1], 64); err == nil {
			h.WeekMax = &n
		}
	}
	if h.Mode != "" || h.WeekMax != nil {
		f.Conditions.Hours = h
	}
}

func authorizationText(value, line string) (*matchcontract.AuthorizationFact, bool) {
	lower := strings.TrimSpace(strings.ToLower(value))
	req := ""
	switch {
	case lower == "required" || strings.HasPrefix(lower, "required in ") ||
		strings.HasPrefix(lower, "required for ") ||
		strings.HasPrefix(lower, "must be authorized to work") ||
		strings.HasPrefix(lower, "you must be authorized to work"):
		req = matchcontract.AuthorizationRequired
	case lower == "not required" || lower == "no authorization required" ||
		lower == "authorization not required":
		req = matchcontract.AuthorizationNotRequired
	default:
		// Sponsorship statements and conditional wording stay unknown.
		return nil, false
	}
	return &matchcontract.AuthorizationFact{
		Country:     authorizationCountry(value),
		Requirement: req,
		Evidence:    textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
	}, true
}

func isWorkAuthorizationStatement(line string) bool {
	lower := strings.TrimSpace(strings.ToLower(line))
	return strings.HasPrefix(lower, "must be authorized to work") ||
		strings.HasPrefix(lower, "you must be authorized to work") ||
		strings.HasPrefix(lower, "must have the right to work")
}

func authorizationFromStatement(line string) *matchcontract.AuthorizationFact {
	return &matchcontract.AuthorizationFact{
		Country:     authorizationCountry(line),
		Requirement: matchcontract.AuthorizationRequired,
		Evidence:    textEvidence(line, matchcontract.SectionUnknown, matchcontract.ReliabilityUsable),
	}
}

func authorizationCountry(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{"work in ", "right to work in ", "authorization in "} {
		if i := strings.Index(lower, marker); i >= 0 {
			country := strings.TrimSpace(text[i+len(marker):])
			country = strings.Trim(country, " .,;:()")
			if country != "" {
				return country
			}
		}
	}
	return ""
}

func workModeToken(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(strings.Trim(value, ".,:;"))) {
	case "remote", "remote only", "fully remote", "remote-first", "remote first":
		return matchcontract.WorkRemote, true
	case "hybrid":
		return matchcontract.WorkHybrid, true
	case "office", "on-site", "onsite", "on site", "in office", "in-office":
		return matchcontract.WorkOffice, true
	}
	return "", false
}

func engagementToken(value string) (string, bool) {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "contract") || strings.Contains(lower, "freelance"):
		return matchcontract.EngagementContract, true
	case strings.Contains(lower, "employment") || strings.Contains(lower, "permanent") ||
		strings.Contains(lower, "full-time") || strings.Contains(lower, "full time") ||
		strings.Contains(lower, "employee"):
		return matchcontract.EngagementEmployment, true
	}
	return "", false
}

func splitItems(value string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
		part = strings.TrimSpace(strings.Trim(part, ".-"))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitLocations(value string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool {
		return r == '·' || r == ';' || r == '|'
	}) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func labelValue(line string, labels ...string) string {
	trimmed := strings.TrimSpace(line)
	i := strings.Index(trimmed, ":")
	if i < 0 {
		return ""
	}
	key := strings.ToLower(strings.TrimSpace(trimmed[:i]))
	for _, label := range labels {
		if key == label {
			return strings.TrimSpace(trimmed[i+1:])
		}
	}
	return ""
}

func lowerHasAny(line string, needles ...string) bool {
	lower := strings.ToLower(line)
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

func textEvidence(quote string, section matchcontract.Section, reliability matchcontract.Reliability) matchcontract.Evidence {
	return matchcontract.Evidence{Quote: quote, Section: section, Reliability: reliability}
}

func fieldEvidence(field string, reliability matchcontract.Reliability) matchcontract.Evidence {
	return matchcontract.Evidence{Field: field, Section: matchcontract.SectionUnknown, Reliability: reliability}
}

func locationsAgree(a, b string) bool {
	x := strings.ToLower(strings.Join(strings.Fields(a), " "))
	y := strings.ToLower(strings.Join(strings.Fields(b), " "))
	return x != "" && x == y
}

func ptrIf(ok bool, v float64) *float64 {
	if !ok {
		return nil
	}
	return &v
}

func sectionOrder(s matchcontract.Section) int {
	switch s {
	case matchcontract.SectionRequired:
		return 0
	case matchcontract.SectionPreferred:
		return 1
	case matchcontract.SectionCompanyContext:
		return 2
	default:
		return 3
	}
}

func sortFacts(f *matchcontract.VacancyFacts) {
	reqs := f.Professional.Requirements
	sort.SliceStable(reqs, func(i, j int) bool {
		if sectionOrder(reqs[i].Evidence.Section) != sectionOrder(reqs[j].Evidence.Section) {
			return sectionOrder(reqs[i].Evidence.Section) < sectionOrder(reqs[j].Evidence.Section)
		}
		if reqs[i].Importance != reqs[j].Importance {
			return reqs[i].Importance < reqs[j].Importance
		}
		return reqs[i].Label < reqs[j].Label
	})
	locs := f.Geography.HiringLocations
	sort.SliceStable(locs, func(i, j int) bool { return locs[i].Label < locs[j].Label })
	modes := f.Conditions.WorkModes
	sort.SliceStable(modes, func(i, j int) bool { return modes[i].Mode < modes[j].Mode })
	mentions := f.Professional.ContextMentions
	sort.SliceStable(mentions, func(i, j int) bool { return mentions[i].Label < mentions[j].Label })
	amb := f.Ambiguities
	sort.SliceStable(amb, func(i, j int) bool {
		if amb[i].Group != amb[j].Group {
			return amb[i].Group < amb[j].Group
		}
		return amb[i].Detail < amb[j].Detail
	})
}
