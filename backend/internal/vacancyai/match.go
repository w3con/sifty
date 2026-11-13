package vacancyai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/w3con/sifty/internal/assess"
	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/prompts"
)

// matcherBase names the matching behaviour that is not in the prompt or the schema: complete-CV
// verdicts, code matching of named skills, focused evidence (docs/specs/quick-match.md 14.12).
const matcherBase = "vacancyai.match/3"

// MatcherVersion identifies the matching behaviour in force: the prompt, the output schema, the
// configured skill vocabulary and the number of passages shown per requirement. A change to any of
// them makes the stored verdicts stale and runs the matching pass again, never the extraction.
func MatcherVersion(cfg *config.Config) string {
	var skills config.Skills
	passages := config.DefaultEvidencePassages
	model := ""
	if cfg != nil {
		skills = cfg.Skills
		passages = cfg.Runtime.EffectiveQuickMatch().ModelPasses.EffectiveEvidencePassages()
		model = cfg.Providers.Providers[cfg.Providers.MatchingStage()].Model
	}
	return matcherVersion(skills.Identity(), passages, model)
}

// matcherVersion names everything that decides a verdict besides the profile and the vacancy: the
// prompt, schema, vocabulary, number of passages and the model, so a different model is a
// different matcher and its verdicts never pass for the old model's.
func matcherVersion(skillsIdentity string, passages int, model string) string {
	return matcherBase + "+" + digest(prompts.RequirementMatching, matchingSchema, skillsIdentity, passages, model)
}

// maxReasonWords bounds a verdict's explanation; the prompt asks for twelve and the adapter cuts
// anything longer, because the reason is shown as one short line.
const maxReasonWords = 12

// reasonMissing explains a "no" for which the model gave no reason of its own.
const reasonMissing = "Not found in the profile"

type matchingOutput struct {
	Verdicts []matchedItem `json:"verdicts"`
}

type matchedItem struct {
	Index      int    `json:"index"`
	Verdict    string `json:"verdict"`
	EvidenceID string `json:"evidenceId"`
	Reason     string `json:"reason"`
}

var matchingSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"verdicts": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"index":      map[string]any{"type": "integer"},
					"verdict":    map[string]any{"type": "string", "enum": []string{"match", "partial", "no", "unclear"}},
					"evidenceId": map[string]any{"type": "string"},
					"reason":     map[string]any{"type": "string"},
				},
				"required":             []string{"index", "verdict", "evidenceId", "reason"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"verdicts"},
	"additionalProperties": false,
}

// Matcher runs the pass that compares extracted requirements with the confirmed profile. The
// profile facts go to the provider configured for the matching stage.
type Matcher struct {
	ProviderName string
	Provider     assess.Provider
	Infer        InferFunc
	MaxAttempts  int
	Backoff      time.Duration
	// Vocabulary drives code matching of named skills and the guard for what a CV cannot answer;
	// nil switches both off.
	Vocabulary *Vocabulary
	// EvidencePassages is how many profile passages the model is shown for one requirement; zero
	// selects the default.
	EvidencePassages int
	// CodeOnly makes the pass decide by code alone: requirements that name skills the profile
	// lists (or that the owner marked as not theirs) get their verdict, every other requirement is
	// unclear, and the model is never called. Used while the matching pass is switched off.
	CodeOnly bool
}

func (m Matcher) passages() int {
	if m.EvidencePassages > 0 {
		return m.EvidencePassages
	}
	return config.DefaultEvidencePassages
}

// Matching is the validated outcome of one matching pass.
type Matching struct {
	Record  matchcontract.RequirementMatches
	Skipped []string
	Raw     json.RawMessage
	Call    Call
	// CodeMatched is how many requirements code decided without the model; Asked is how many the
	// model judged.
	CodeMatched int
	Asked       int
}

// Match gives one verdict per extracted requirement against the profile facts. A requirement that
// names technologies the profile lists is a match decided by code and never sent to the model. For
// each other requirement the model is shown the few profile passages closest to it and the computed
// facts, and a verdict that rests on a fact it was not shown becomes unclear. The profile is a
// complete CV: a specific skill or experience it does not show is a "no", while what a CV does not
// state (soft skills, team size, work authorization, availability) stays unclear. A requirement the
// model did not judge is unclear. An empty profile is all unclear without calling the model.
func (m Matcher) Match(ctx context.Context, e matchcontract.VacancyExtraction, facts []matchcontract.ProfileFact, profileRevision int) (Matching, error) {
	if err := e.Validate(); err != nil {
		return Matching{}, fmt.Errorf("extraction: %w", err)
	}
	if len(e.Requirements) == 0 {
		return Matching{}, fmt.Errorf("%w: the extraction has no requirements to match", ErrInvalidOutput)
	}
	reqHash, err := e.RequirementsHash()
	if err != nil {
		return Matching{}, err
	}
	record := matchcontract.RequirementMatches{
		SchemaVersion: matchcontract.SchemaVersion, MatcherVersion: m.version(), JobID: e.JobID,
		RequirementsHash: reqHash, ProfileRevision: profileRevision,
	}
	if len(facts) == 0 {
		for i := range e.Requirements {
			record.Verdicts = append(record.Verdicts, matchcontract.RequirementVerdict{Index: i, Verdict: matchcontract.VerdictUnclear})
		}
		return Matching{Record: record, Skipped: []string{"the profile has no facts to compare with; no model call"}}, nil
	}

	verdicts := make([]matchcontract.RequirementVerdict, len(e.Requirements))
	var pending []int
	codeMatched := 0
	profileTerms := m.Vocabulary.indexProfile(facts)
	for i, r := range e.Requirements {
		if evidence, verdict, reason, ok := m.Vocabulary.codeMatch(r, facts, profileTerms); ok {
			ev := evidence
			verdicts[i] = matchcontract.RequirementVerdict{Index: i, Verdict: verdict, Evidence: &ev, Reason: reason}
			codeMatched++
			continue
		}
		pending = append(pending, i)
	}
	if m.CodeOnly {
		for _, i := range pending {
			verdicts[i] = matchcontract.RequirementVerdict{Index: i, Verdict: matchcontract.VerdictUnclear}
		}
		record.Verdicts = verdicts
		if err := record.ValidateAgainst(e); err != nil {
			return Matching{}, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
		}
		return Matching{Record: record, CodeMatched: codeMatched, Skipped: []string{"matching by model is off; no model call"}}, nil
	}
	result := Matching{CodeMatched: codeMatched, Asked: len(pending)}
	result.Call = Call{ProviderName: m.ProviderName, ProviderType: m.Provider.Type, Model: m.Provider.Model}
	if len(pending) == 0 {
		record.Verdicts = verdicts
		if err := record.ValidateAgainst(e); err != nil {
			return Matching{}, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
		}
		result.Record = record
		result.Skipped = []string{"every requirement names skills the profile lists; no model call"}
		return result, nil
	}

	// What the model is shown: for each pending requirement the closest passages, besides the
	// computed facts that go with every requirement.
	computed := map[int]bool{}
	for i, f := range facts {
		if profile.IsComputedField(f.Field) {
			computed[i] = true
		}
	}
	offered := make(map[int]map[int]bool, len(pending))
	shown := map[int]bool{}
	var requirementLines strings.Builder
	for _, i := range pending {
		r := e.Requirements[i]
		picks := m.Vocabulary.closestPassages(r, facts, m.passages())
		offered[i] = make(map[int]bool, len(picks))
		ids := make([]string, 0, len(picks))
		for _, p := range picks {
			offered[i][p] = true
			shown[p] = true
			ids = append(ids, fmt.Sprintf("P%d", p+1))
		}
		fmt.Fprintf(&requirementLines, "%d. [%s] [%s] %s", i, r.Level, r.Kind, r.Text)
		if len(r.Alternatives) > 0 {
			fmt.Fprintf(&requirementLines, " (any of: %s)", strings.Join(r.Alternatives, " | "))
		}
		if len(ids) == 0 {
			requirementLines.WriteString(" | facts: none besides the computed facts")
		} else {
			fmt.Fprintf(&requirementLines, " | facts: %s", strings.Join(ids, ", "))
		}
		requirementLines.WriteString("\n")
	}
	var factLines, computedLines strings.Builder
	shownIdx := make([]int, 0, len(shown))
	for p := range shown {
		shownIdx = append(shownIdx, p)
	}
	sort.Ints(shownIdx)
	for _, p := range shownIdx {
		fmt.Fprintf(&factLines, "P%d | %s | %s\n", p+1, facts[p].Field, facts[p].Value)
	}
	for i, f := range facts {
		if computed[i] {
			fmt.Fprintf(&computedLines, "P%d | %s | %s\n", i+1, f.Field, f.Value)
		}
	}
	if factLines.Len() == 0 {
		factLines.WriteString("(none)\n")
	}
	if computedLines.Len() == 0 {
		computedLines.WriteString("(none)\n")
	}
	schemaJSON, _ := json.Marshal(matchingSchema)
	request := fmt.Sprintf(`%s
SCHEMA
%s
REQUIREMENTS (DATA)
%s
PROFILE FACTS (DATA)
%s
COMPUTED FACTS (DATA; they apply to every requirement)
%s`, prompts.RequirementMatching, schemaJSON, requirementLines.String(), factLines.String(), computedLines.String())

	p := pass{providerName: m.ProviderName, provider: m.Provider, infer: m.Infer, attempts: m.MaxAttempts, backoff: m.Backoff}
	raw, call, err := p.run(ctx, request, matchingSchema, func(raw json.RawMessage) error {
		var out matchingOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidOutput, err)
		}
		judged := append([]matchcontract.RequirementVerdict(nil), verdicts...)
		notes, err := m.validateMatching(e, facts, pending, offered, computed, judged, out)
		if err != nil {
			return err
		}
		record.Verdicts = judged
		if err := record.ValidateAgainst(e); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidOutput, err)
		}
		result.Record, result.Skipped = record, notes
		return nil
	})
	result.Raw, result.Call = raw, call
	return result, err
}

func (m Matcher) version() string {
	return matcherVersion(m.Vocabulary.identity(), m.passages(), m.Provider.Model)
}

// validateMatching turns the model's answer into verdicts for the pending requirements, written
// into verdicts in place. A claim that the profile meets a requirement rests on a fact the model
// was shown for it; a "no" on a requirement a CV cannot answer is unclear; a "no" on authorization
// or other needs a shown fact that contradicts it, while a "no" on a specific skill, experience,
// qualification or language needs none, because the profile is a complete CV.
func (m Matcher) validateMatching(e matchcontract.VacancyExtraction, facts []matchcontract.ProfileFact, pending []int,
	offered map[int]map[int]bool, computed map[int]bool, verdicts []matchcontract.RequirementVerdict, out matchingOutput) ([]string, error) {
	var skipped []string
	skip := func(format string, args ...any) { skipped = append(skipped, fmt.Sprintf(format, args...)) }

	isPending := make(map[int]bool, len(pending))
	for _, i := range pending {
		isPending[i] = true
	}
	byIndex := make(map[int]matchedItem, len(pending))
	for _, item := range out.Verdicts {
		if !isPending[item.Index] {
			skip("verdict for index %d: not a requirement that was asked, ignored", item.Index)
			continue
		}
		if _, dup := byIndex[item.Index]; dup {
			skip("verdict for index %d: duplicate ignored", item.Index)
			continue
		}
		byIndex[item.Index] = item
	}
	if len(byIndex) == 0 {
		return skipped, fmt.Errorf("%w: no verdict for any requirement", ErrInvalidOutput)
	}

	for _, i := range pending {
		req := e.Requirements[i]
		unclear := matchcontract.RequirementVerdict{Index: i, Verdict: matchcontract.VerdictUnclear}
		item, judged := byIndex[i]
		if !judged {
			skip("requirement %d %q: not judged, unclear", i, req.Text)
			verdicts[i] = unclear
			continue
		}
		verdict := matchcontract.Verdict(strings.ToLower(strings.TrimSpace(item.Verdict)))
		if !verdict.Valid() {
			skip("requirement %d %q: unknown verdict %q, unclear", i, req.Text, item.Verdict)
			verdicts[i] = unclear
			continue
		}
		evidence := resolveEvidence(facts, item.EvidenceID, func(n int) bool { return offered[i][n] || computed[n] })
		reason := shortReason(item.Reason)
		switch verdict {
		case matchcontract.VerdictMatch, matchcontract.VerdictPartial:
			if evidence == nil {
				// A claim that the profile supports a requirement, resting on no fact the model was
				// shown for it, is unconfirmed.
				skip("requirement %d %q: verdict %q names no profile fact offered for it (%q), unclear", i, req.Text, verdict, item.EvidenceID)
				verdicts[i] = unclear
				continue
			}
		case matchcontract.VerdictNo:
			switch {
			case !m.Vocabulary.answerableFromCV(req):
				skip("requirement %d %q: a CV does not usually state this, so \"no\" is unclear", i, req.Text)
				verdicts[i] = unclear
				continue
			case (req.Kind == matchcontract.RequirementAuthorization || req.Kind == matchcontract.RequirementOther) && evidence == nil:
				skip("requirement %d %q: \"no\" on %s names no contradicting fact offered for it, unclear", i, req.Text, req.Kind)
				verdicts[i] = unclear
				continue
			}
			if reason == "" {
				reason = reasonMissing
			}
			if req.Kind != matchcontract.RequirementAuthorization && req.Kind != matchcontract.RequirementOther {
				// For a skill, experience, qualification or language the absence is the whole
				// finding; a fact the model attached to it would only be shown as if it mattered.
				evidence = nil
			}
		default: // unclear
			verdicts[i] = unclear
			verdicts[i].Reason = reason
			continue
		}
		verdicts[i] = matchcontract.RequirementVerdict{Index: i, Verdict: verdict, Reason: reason, Evidence: evidence}
	}
	return skipped, nil
}

// resolveEvidence maps a model's "P3" to the profile fact it names, provided that fact was offered
// for the requirement. Anything else is nil.
func resolveEvidence(facts []matchcontract.ProfileFact, id string, offered func(index int) bool) *matchcontract.ProfileEvidence {
	id = strings.ToUpper(strings.TrimSpace(id))
	if !strings.HasPrefix(id, "P") {
		return nil
	}
	n, err := strconv.Atoi(id[1:])
	if err != nil || n < 1 || n > len(facts) || !offered(n-1) {
		return nil
	}
	return &matchcontract.ProfileEvidence{Field: facts[n-1].Field, Value: facts[n-1].Value}
}

func shortReason(s string) string {
	words := strings.Fields(s)
	if len(words) > maxReasonWords {
		words = words[:maxReasonWords]
	}
	return strings.Join(words, " ")
}
