package matchcontract

import (
	"fmt"
	"math"
	"strings"
)

// This file is the contract of the two-score model (docs/specs/quick-match.md section 14):
// vacancy-side requirement records produced by a model pass, the per-requirement verdicts
// produced by a second pass against the confirmed profile, and the evaluator's two-score
// result. Like the rest of the package it performs no I/O and knows no provider: the model
// passes persist these records, the pure evaluator consumes them.

// RequirementKind classifies one requirement extracted from the vacancy text.
type RequirementKind string

const (
	RequirementSkill         RequirementKind = "skill"
	RequirementExperience    RequirementKind = "experience"
	RequirementQualification RequirementKind = "qualification"
	RequirementLanguage      RequirementKind = "language"
	RequirementAuthorization RequirementKind = "authorization"
	RequirementOther         RequirementKind = "other"
)

// Valid reports whether k is a known requirement kind.
func (k RequirementKind) Valid() bool {
	switch k {
	case RequirementSkill, RequirementExperience, RequirementQualification,
		RequirementLanguage, RequirementAuthorization, RequirementOther:
		return true
	}
	return false
}

// ExtractedRequirement is one requirement the vacancy states. Text is a short normalized
// phrase; Quote is the exact substring of the retained source that supports it. Alternatives is
// set when the demand is a group of which any one member satisfies it ("Kotlin, Python or Go",
// "e.g. React or Vue"): Text then names the group and Alternatives lists its members. A group is
// one requirement, never one per member.
type ExtractedRequirement struct {
	Text         string          `json:"text"`
	Kind         RequirementKind `json:"kind"`
	Level        Importance      `json:"level"`
	Quote        string          `json:"quote"`
	Alternatives []string        `json:"alternatives,omitempty"`
}

// ExtractedCondition is one stated working condition with its supporting quote.
type ExtractedCondition struct {
	Value string `json:"value"`
	Quote string `json:"quote"`
}

// ExtractedConditions carries the condition facts the model found. Nil means not stated.
type ExtractedConditions struct {
	Compensation *ExtractedCondition `json:"compensation,omitempty"`
	WorkMode     *ExtractedCondition `json:"workMode,omitempty"`
	Engagement   *ExtractedCondition `json:"engagement,omitempty"`
	Location     *ExtractedCondition `json:"location,omitempty"`
	Timezone     *ExtractedCondition `json:"timezone,omitempty"`
	Hours        *ExtractedCondition `json:"hours,omitempty"`
}

// VacancyExtraction is the persisted output of the vacancy-only model pass for one retained
// source. An extraction with no requirements is a final answer for that source, not a pending
// one. SourceHash is the RawSourceHash of the retained source the text came from.
type VacancyExtraction struct {
	SchemaVersion    string                 `json:"schemaVersion"`
	ExtractorVersion string                 `json:"extractorVersion"`
	JobID            int64                  `json:"jobId"`
	SourceHash       string                 `json:"sourceHash"`
	Requirements     []ExtractedRequirement `json:"requirements"`
	Conditions       ExtractedConditions    `json:"conditions"`
}

// Validate refuses a record the evaluator must not trust. It checks shape and enums only; that
// every quote is an exact substring of the retained source is verified by the adapter that has
// the source text.
func (e VacancyExtraction) Validate() error {
	if err := CheckSchema(e.SchemaVersion); err != nil {
		return err
	}
	if strings.TrimSpace(e.ExtractorVersion) == "" {
		return fmt.Errorf("extraction needs an extractor version")
	}
	if strings.TrimSpace(e.SourceHash) == "" {
		return fmt.Errorf("extraction needs a source hash")
	}
	seen := make(map[string]bool, len(e.Requirements))
	for i, r := range e.Requirements {
		text := normalizeSpace(r.Text)
		if text == "" {
			return fmt.Errorf("requirement %d has no text", i)
		}
		if !r.Kind.Valid() {
			return fmt.Errorf("requirement %d has invalid kind %q", i, r.Kind)
		}
		if !r.Level.Valid() {
			return fmt.Errorf("requirement %d has invalid level %q", i, r.Level)
		}
		if strings.TrimSpace(r.Quote) == "" {
			return fmt.Errorf("requirement %d has no quote", i)
		}
		for _, alt := range r.Alternatives {
			if normalizeSpace(alt) == "" {
				return fmt.Errorf("requirement %d has an empty alternative", i)
			}
		}
		key := string(r.Kind) + "\x00" + strings.ToLower(text)
		if seen[key] {
			return fmt.Errorf("requirement %d duplicates an earlier requirement %q", i, r.Text)
		}
		seen[key] = true
	}
	for name, c := range map[string]*ExtractedCondition{
		"compensation": e.Conditions.Compensation, "workMode": e.Conditions.WorkMode,
		"engagement": e.Conditions.Engagement, "location": e.Conditions.Location,
		"timezone": e.Conditions.Timezone, "hours": e.Conditions.Hours,
	} {
		if c != nil && (strings.TrimSpace(c.Value) == "" || strings.TrimSpace(c.Quote) == "") {
			return fmt.Errorf("condition %s needs a value and a quote", name)
		}
	}
	return nil
}

// RequirementsHash identifies the exact ordered requirement list that verdicts refer to by
// index. Quotes and conditions are excluded: they do not change what is matched.
func (e VacancyExtraction) RequirementsHash() (string, error) {
	type key struct {
		Text         string          `json:"text"`
		Kind         RequirementKind `json:"kind"`
		Level        Importance      `json:"level"`
		Alternatives []string        `json:"alternatives,omitempty"`
	}
	keys := make([]key, 0, len(e.Requirements))
	for _, r := range e.Requirements {
		k := key{Text: normalizeSpace(r.Text), Kind: r.Kind, Level: r.Level}
		for _, alt := range r.Alternatives {
			k.Alternatives = append(k.Alternatives, normalizeSpace(alt))
		}
		keys = append(keys, k)
	}
	return hashJSON(keys)
}

// ProfileFact is one confirmed profile fact offered to the requirement-matching pass: the profile
// field it comes from and a one-line value. Only what the owner confirmed and what describes
// capability appears here; goals, demographics, name and photo never do.
type ProfileFact struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// Verdict is the model's judgement of one requirement against the confirmed profile.
type Verdict string

const (
	VerdictMatch   Verdict = "match"
	VerdictPartial Verdict = "partial"
	VerdictNo      Verdict = "no"
	VerdictUnclear Verdict = "unclear"
)

// Valid reports whether v is a known verdict.
func (v Verdict) Valid() bool {
	switch v {
	case VerdictMatch, VerdictPartial, VerdictNo, VerdictUnclear:
		return true
	}
	return false
}

// ProfileEvidence names the profile field and value a verdict rests on.
type ProfileEvidence struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// RequirementVerdict is one requirement's verdict. Index refers to the extraction's
// Requirements slice. Reason is a short model-written phrase, never parsed.
type RequirementVerdict struct {
	Index    int              `json:"index"`
	Verdict  Verdict          `json:"verdict"`
	Evidence *ProfileEvidence `json:"evidence,omitempty"`
	Reason   string           `json:"reason,omitempty"`
}

// RequirementMatches is the persisted output of the profile-matching pass, keyed by the
// requirement list it judged, the profile revision it saw and the matcher version.
type RequirementMatches struct {
	SchemaVersion    string               `json:"schemaVersion"`
	MatcherVersion   string               `json:"matcherVersion"`
	JobID            int64                `json:"jobId"`
	RequirementsHash string               `json:"requirementsHash"`
	ProfileRevision  int                  `json:"profileRevision"`
	Verdicts         []RequirementVerdict `json:"verdicts"`
}

// ValidateAgainst checks the matches against the extraction they claim to judge: the same job,
// the same requirement list, exactly one verdict per requirement, and profile evidence on every
// verdict that asserts the profile supports a requirement. A verdict the adapter could not
// ground in the profile must already have been downgraded to unclear.
func (m RequirementMatches) ValidateAgainst(e VacancyExtraction) error {
	if err := CheckSchema(m.SchemaVersion); err != nil {
		return err
	}
	if strings.TrimSpace(m.MatcherVersion) == "" {
		return fmt.Errorf("matches need a matcher version")
	}
	if m.ProfileRevision < 0 {
		return fmt.Errorf("matches profile revision must not be negative")
	}
	if m.JobID != e.JobID {
		return fmt.Errorf("matches are for job %d, extraction is for job %d", m.JobID, e.JobID)
	}
	want, err := e.RequirementsHash()
	if err != nil {
		return err
	}
	if m.RequirementsHash != want {
		return fmt.Errorf("matches judge a different requirement list")
	}
	if len(m.Verdicts) != len(e.Requirements) {
		return fmt.Errorf("matches carry %d verdicts for %d requirements", len(m.Verdicts), len(e.Requirements))
	}
	seen := make([]bool, len(e.Requirements))
	for _, v := range m.Verdicts {
		if v.Index < 0 || v.Index >= len(e.Requirements) {
			return fmt.Errorf("verdict index %d is outside the requirement list", v.Index)
		}
		if seen[v.Index] {
			return fmt.Errorf("requirement %d has more than one verdict", v.Index)
		}
		seen[v.Index] = true
		if !v.Verdict.Valid() {
			return fmt.Errorf("requirement %d has invalid verdict %q", v.Index, v.Verdict)
		}
		if v.Verdict == VerdictMatch || v.Verdict == VerdictPartial {
			if v.Evidence == nil || strings.TrimSpace(v.Evidence.Field) == "" || strings.TrimSpace(v.Evidence.Value) == "" {
				return fmt.Errorf("requirement %d verdict %q needs profile evidence", v.Index, v.Verdict)
			}
		}
	}
	return nil
}

// VerdictUtilities is the utility each verdict contributes to You fit. It is policy data
// carried by Rules, never a constant inside the evaluator.
type VerdictUtilities struct {
	Match   float64 `json:"match"`
	Partial float64 `json:"partial"`
	Unclear float64 `json:"unclear"`
	No      float64 `json:"no"`
}

// Of returns the utility of a verdict. An unknown verdict has no utility.
func (u VerdictUtilities) Of(v Verdict) (float64, bool) {
	switch v {
	case VerdictMatch:
		return u.Match, true
	case VerdictPartial:
		return u.Partial, true
	case VerdictUnclear:
		return u.Unclear, true
	case VerdictNo:
		return u.No, true
	}
	return 0, false
}

func (u VerdictUtilities) validate() error {
	for name, v := range map[string]float64{"match": u.Match, "partial": u.Partial, "unclear": u.Unclear, "no": u.No} {
		if !isFinite(v) || v < 0 || v > 1 {
			return fmt.Errorf("verdict utility %q must be within 0..1", name)
		}
	}
	return nil
}

// Owner-approved starting policy (uncalibrated), kept in one place and injected into Rules by
// the configuration adapter. The evaluator reads Rules only.
const DefaultUnknownGoalUtility = 0.5

// DefaultVerdictUtilities returns the approved starting verdict utilities.
func DefaultVerdictUtilities() VerdictUtilities {
	return VerdictUtilities{Match: 1, Partial: 0.5, Unclear: 0.5, No: 0}
}

// SourceState says whether the retained vacancy text can be analysed at all. The caller decides
// it from the retained source; the evaluator never parses a free-text description status.
type SourceState string

const (
	SourceUsable      SourceState = "usable"
	SourceUnavailable SourceState = "unavailable"
)

// Valid reports whether s is a known source state.
func (s SourceState) Valid() bool { return s == SourceUsable || s == SourceUnavailable }

// ConditionConflict records a working condition the vacancy's deterministic parse and the model's
// extraction stated differently. The deterministic fact is the one used; the conflict is kept so
// it can be audited later and is never shown to the owner.
type ConditionConflict struct {
	Field         string `json:"field"`
	Deterministic string `json:"deterministic"`
	Model         string `json:"model"`
	Quote         string `json:"quote,omitempty"`
}

// FitInput is everything the pure two-score evaluator may read. A nil Extraction means the
// requirements have not been extracted yet; a nil Matches means they have not been matched to
// the current profile yet.
type FitInput struct {
	Profile ProfileSnapshot
	Source  SourceState
	Facts   *VacancyFacts
	// DescriptionHash is HashDescription of the retained description the facts came from. An
	// extraction is used only when its SourceHash equals it.
	DescriptionHash string
	Extraction      *VacancyExtraction
	Matches         *RequirementMatches
	// MatchingOff says the model's matching pass is switched off: You fit gets status off, and
	// Matches, when present, holds only what code decided; the requirements are listed all the same.
	MatchingOff bool
	Rules       Rules
	// ConditionConflicts are carried through to the result unchanged; the evaluator does not
	// interpret them.
	ConditionConflicts []ConditionConflict
}

// ScoreStatus is the state of one of the two scores.
type ScoreStatus string

const (
	// ScoreScored: Score and RawScore are present.
	ScoreScored ScoreStatus = "scored"
	// ScoreUnscored (Fits you only): the profile states no goal this vacancy could be compared with.
	ScoreUnscored ScoreStatus = "unscored"
	// ScoreNoRequirements (You fit only): the text was analysed and states no requirements.
	ScoreNoRequirements ScoreStatus = "no_requirements"
	// ScorePending (You fit only): the requirements are not extracted or matched yet.
	ScorePending ScoreStatus = "pending"
	// ScoreUnavailable: the vacancy text is unavailable, so neither score has a number.
	ScoreUnavailable ScoreStatus = "unavailable"
	// ScoreOff (You fit only): the model's matching pass is switched off, so there is no You fit.
	ScoreOff ScoreStatus = "off"
)

func (s ScoreStatus) valid() bool {
	switch s {
	case ScoreScored, ScoreUnscored, ScoreNoRequirements, ScorePending, ScoreUnavailable, ScoreOff:
		return true
	}
	return false
}

// GroupFit is one goal group's part in Fits you. Known is false when the group had no
// comparable fact and the midpoint stood in for it.
type GroupFit struct {
	Group   Group   `json:"group"`
	Weight  float64 `json:"weight"`
	Utility float64 `json:"utility"`
	Known   bool    `json:"known"`
}

// FitsYou is the goals score: does the job match what the owner wants.
type FitsYou struct {
	Status   ScoreStatus `json:"status"`
	Score    *int        `json:"score"`
	RawScore *float64    `json:"rawScore"`
	Groups   []GroupFit  `json:"groups"`
}

// FitCounts counts requirement verdicts for display next to You fit.
type FitCounts struct {
	Match   int `json:"match"`
	Partial int `json:"partial"`
	No      int `json:"no"`
	Unclear int `json:"unclear"`
}

// YouFit is the requirements score: do the owner's confirmed facts meet what the job asks.
type YouFit struct {
	Status   ScoreStatus `json:"status"`
	Score    *int        `json:"score"`
	RawScore *float64    `json:"rawScore"`
	Counts   FitCounts   `json:"counts"`
}

// FitOrigin says whether a line comes from a goal comparison or from a requirement verdict.
type FitOrigin string

const (
	OriginGoal        FitOrigin = "goal"
	OriginRequirement FitOrigin = "requirement"
)

// FitLine is one factual line of a list. Goal lines carry the criterion, the compared values and
// a stable reason code; requirement lines carry the requirement, its verdict and the profile
// evidence. Views format these values and never recompute them.
type FitLine struct {
	Origin       FitOrigin       `json:"origin"`
	Group        Group           `json:"group,omitempty"`
	Criterion    Criterion       `json:"criterion,omitempty"`
	Utility      *float64        `json:"utility,omitempty"`
	Kind         RequirementKind `json:"kind,omitempty"`
	Level        Importance      `json:"level,omitempty"`
	Verdict      Verdict         `json:"verdict,omitempty"`
	Subject      string          `json:"subject,omitempty"`
	Alternatives []string        `json:"alternatives,omitempty"`
	ProfileField string          `json:"profileField,omitempty"`
	ProfileValue string          `json:"profileValue,omitempty"`
	JobValue     string          `json:"jobValue,omitempty"`
	Reason       string          `json:"reason,omitempty"`
	Quote        string          `json:"quote,omitempty"`
}

// FitLists are the four short lists shown to the owner. Every list is non-nil.
type FitLists struct {
	Matches    []FitLine `json:"matches"`
	Mismatches []FitLine `json:"mismatches"`
	Missing    []FitLine `json:"missing"`
	ToClarify  []FitLine `json:"toClarify"`
}

// FitProvenance identifies the exact inputs that produced a FitResult.
type FitProvenance struct {
	SchemaVersion    string `json:"schemaVersion"`
	ProfileID        string `json:"profileId"`
	ProfileRevision  int    `json:"profileRevision"`
	RawSourceHash    string `json:"rawSourceHash,omitempty"`
	FactsHash        string `json:"factsHash,omitempty"`
	ExtractorVersion string `json:"extractorVersion,omitempty"`
	RequirementsHash string `json:"requirementsHash,omitempty"`
	MatcherVersion   string `json:"matcherVersion,omitempty"`
	RulesVersion     string `json:"rulesVersion"`
	RulesIdentity    string `json:"rulesIdentity,omitempty"`
}

// FitResult is the evaluator output and the renderer input for both scores.
type FitResult struct {
	Source  SourceState `json:"source"`
	FitsYou FitsYou     `json:"fitsYou"`
	YouFit  YouFit      `json:"youFit"`
	// MatchingOff is true while the model's matching pass is switched off: a view then draws no
	// You fit at all, whatever that score's status is (docs/specs/quick-match.md 14.14).
	MatchingOff bool          `json:"matchingOff"`
	Lists       FitLists      `json:"lists"`
	Provenance  FitProvenance `json:"provenance"`
	// ConditionConflicts are kept for audit and never rendered.
	ConditionConflicts []ConditionConflict `json:"conditionConflicts"`
}

// FitView is the API envelope of one stored two-score result. A stale view keeps the numbers of the
// inputs it was computed from and says so; the worker recomputes it, and the owner is never shown
// a stale number as current.
type FitView struct {
	Result       FitResult `json:"result"`
	Stale        bool      `json:"stale"`
	StaleReasons []string  `json:"staleReasons,omitempty"`
}

// Validate checks the result's status and score invariants.
func (r FitResult) Validate() error {
	if !r.Source.Valid() {
		return fmt.Errorf("invalid source state %q", r.Source)
	}
	if err := validateScore("fits you", r.FitsYou.Status, r.FitsYou.Score, r.FitsYou.RawScore); err != nil {
		return err
	}
	if err := validateScore("you fit", r.YouFit.Status, r.YouFit.Score, r.YouFit.RawScore); err != nil {
		return err
	}
	if r.Source == SourceUnavailable {
		if r.FitsYou.Status != ScoreUnavailable || r.YouFit.Status != ScoreUnavailable {
			return fmt.Errorf("unavailable source must leave both scores unavailable")
		}
	}
	switch r.FitsYou.Status {
	case ScoreScored, ScoreUnscored, ScoreUnavailable:
	default:
		return fmt.Errorf("fits you cannot have status %q", r.FitsYou.Status)
	}
	switch r.YouFit.Status {
	case ScoreScored, ScoreNoRequirements, ScorePending, ScoreUnavailable, ScoreOff:
	default:
		return fmt.Errorf("you fit cannot have status %q", r.YouFit.Status)
	}
	for _, g := range r.FitsYou.Groups {
		if !g.Group.Valid() || !isFinite(g.Weight) || g.Weight < 0 || !isFinite(g.Utility) || g.Utility < 0 || g.Utility > 1 {
			return fmt.Errorf("fits you group %q carries an invalid number", g.Group)
		}
	}
	return nil
}

func validateScore(name string, status ScoreStatus, score *int, raw *float64) error {
	if !status.valid() {
		return fmt.Errorf("%s has invalid status %q", name, status)
	}
	if status == ScoreScored {
		if score == nil || raw == nil {
			return fmt.Errorf("scored %s needs score and raw score", name)
		}
		if *score < 0 || *score > 100 || !isFinite(*raw) || *raw < 0 || *raw > 1 {
			return fmt.Errorf("%s is outside its range", name)
		}
		return nil
	}
	if score != nil || raw != nil {
		return fmt.Errorf("%s with status %q must not carry a number", name, status)
	}
	return nil
}

// ScoreFromRaw converts a 0..1 mean into the 0..100 integer shown to the owner.
func ScoreFromRaw(raw float64) int {
	score := int(math.Round(raw * 100))
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func normalizeSpace(s string) string { return strings.Join(strings.Fields(s), " ") }
