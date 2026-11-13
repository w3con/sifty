package vacancyai

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
)

// ReasonNamedSkill is the reason stored on a verdict that code decided without the model. The
// measurement counts these to say how many requirements never needed the model.
const ReasonNamedSkill = "Named skill found in the profile"

// ReasonMarkedAbsent is the reason of a "no" decided by code because the owner has said they do not
// have the skill the requirement names.
const ReasonMarkedAbsent = "You said you do not have this skill"

// ReasonSomeNamedSkills is the reason of a partial verdict decided by code: a requirement that
// names several technologies, of which the profile lists some but not all.
const ReasonSomeNamedSkills = "Only some of the named skills are in the profile"

// ReasonSkillInProfile is the reason of a "match" decided by code because the owner has explicitly
// added the skill by its exact offered name, regardless of whether the vocabulary knows it.
const ReasonSkillInProfile = "Skill in your profile"

// Vocabulary is the compiled form of config.Skills: which names are one technology, which words
// show that an ambiguous name is a technology, which words carry no meaning for choosing profile
// passages and which requirements a CV usually cannot answer. A nil Vocabulary is valid and
// switches code matching and the CV-cannot-answer guard off.
type Vocabulary struct {
	id            string
	terms         map[string]termInfo // key: lowercase tokens joined by one space
	maxWords      int
	groupTerms    map[string][]string // group -> every term spelling, for synonym expansion
	context       map[string]bool
	generic       map[string]bool
	notAnswerable []string
	// categoryByStem maps the stem of an ordinary word ("cloud") to the technology groups it stands for.
	categoryByStem map[string][]string
	// maxSkillNameWords is the maximum word count for a phrase to be a skill (not a confirmed fact).
	maxSkillNameWords int
	// verbEndings lists word-ending strings that mark a word as a verb form (activity, not a skill).
	verbEndings []string
	// modifierWords lists -ing words that work as adjective modifiers inside a multi-word phrase.
	// They are treated as activity indicators only when they appear as the sole word of the phrase.
	modifierWords map[string]bool
}

type termInfo struct {
	group     string
	ambiguous bool
	exact     string // the configured spelling, compared case-sensitively for an ambiguous term
}

// NewVocabulary compiles the configured vocabulary.
func NewVocabulary(s config.Skills) *Vocabulary {
	maxSkillWords := s.MaxSkillNameWords
	if maxSkillWords <= 0 {
		maxSkillWords = 3
	}
	verbEndings := s.VerbEndings
	if len(verbEndings) == 0 {
		verbEndings = []string{"ing"}
	}
	modifierWords := map[string]bool{}
	if len(s.ModifierWords) > 0 {
		for _, w := range s.ModifierWords {
			modifierWords[strings.ToLower(strings.TrimSpace(w))] = true
		}
	} else {
		for _, w := range []string{"coding", "testing", "scripting", "streaming", "programming"} {
			modifierWords[w] = true
		}
	}
	v := &Vocabulary{
		id: s.Identity(), terms: map[string]termInfo{}, groupTerms: map[string][]string{},
		context: map[string]bool{}, generic: map[string]bool{}, maxWords: 1,
		categoryByStem: map[string][]string{},
		maxSkillNameWords: maxSkillWords,
		verbEndings:       verbEndings,
		modifierWords:     modifierWords,
	}
	for _, c := range s.Categories {
		for _, w := range c.Words {
			// Only single words: a category word is compared with one word of a requirement.
			if toks := tokenize(w); len(toks) == 1 {
				key := stem(toks[0].lower)
				v.categoryByStem[key] = append(v.categoryByStem[key], c.Groups...)
			}
		}
	}
	for _, w := range s.TechnologyContext {
		v.context[strings.ToLower(strings.TrimSpace(w))] = true
	}
	for _, w := range s.GenericWords {
		v.generic[strings.ToLower(strings.TrimSpace(w))] = true
	}
	for _, p := range s.NotAnswerable {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			v.notAnswerable = append(v.notAnswerable, p)
		}
	}
	add := func(group, term string, ambiguous bool) {
		toks := tokenize(term)
		if len(toks) == 0 {
			return
		}
		lowers := make([]string, len(toks))
		for i, t := range toks {
			lowers[i] = t.lower
		}
		if len(toks) > v.maxWords {
			v.maxWords = len(toks)
		}
		v.terms[strings.Join(lowers, " ")] = termInfo{group: group, ambiguous: ambiguous, exact: strings.TrimSpace(term)}
		v.groupTerms[group] = append(v.groupTerms[group], strings.Join(lowers, " "))
	}
	for _, g := range s.Groups {
		ambiguous := map[string]bool{}
		for _, t := range g.AmbiguousTerms {
			ambiguous[strings.ToLower(strings.TrimSpace(t))] = true
		}
		// The group's name is a term unless the name itself is listed as ambiguous.
		if !ambiguous[strings.ToLower(strings.TrimSpace(g.Name))] {
			add(g.Name, g.Name, false)
		}
		for _, t := range g.Terms {
			add(g.Name, t, false)
		}
		for _, t := range g.AmbiguousTerms {
			add(g.Name, t, true)
		}
	}
	return v
}

func (v *Vocabulary) identity() string {
	if v == nil {
		return config.Skills{}.Identity()
	}
	return v.id
}

// token is one word of a text with its place in it.
type token struct {
	text  string
	lower string
	start int
	end   int
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// tokenize splits a text into words. A word is a run of letters and digits. A "+" or "#" directly
// after one belongs to it (C++, C#), and so does a "." or "&" between two of them (Node.js, R&D),
// and a "." that starts a name (.NET). Everything else separates words.
func tokenize(s string) []token {
	runes := []rune(s)
	// byteAt[i] is the byte offset of rune i, so a token can be sliced out of the original text.
	byteAt := make([]int, len(runes)+1)
	off := 0
	for i, r := range runes {
		byteAt[i] = off
		off += len(string(r))
	}
	byteAt[len(runes)] = off

	var out []token
	i := 0
	for i < len(runes) {
		r := runes[i]
		startsName := r == '.' && i+1 < len(runes) && isWordRune(runes[i+1]) && (i == 0 || !isWordRune(runes[i-1]))
		if !isWordRune(r) && !startsName {
			i++
			continue
		}
		j := i + 1
		for j < len(runes) {
			c := runes[j]
			switch {
			case isWordRune(c):
				j++
				continue
			case (c == '+' || c == '#') && isWordRune(runes[j-1]) || c == '+' && runes[j-1] == '+':
				j++
				continue
			case (c == '.' || c == '&') && j+1 < len(runes) && isWordRune(runes[j+1]) && isWordRune(runes[j-1]):
				j++
				continue
			}
			break
		}
		text := string(runes[i:j])
		out = append(out, token{text: text, lower: strings.ToLower(text), start: byteAt[i], end: byteAt[j]})
		i = j
	}
	return out
}

// hit is a known technology found in a text: the group it belongs to and the tokens it covers.
type hit struct {
	group    string
	from, to int // token indexes, [from, to)
}

// scan finds the known technologies in a text. A term that is not ambiguous counts wherever it
// stands as a whole word. An ambiguous term counts only when it is spelled exactly as configured,
// is not joined to a neighbouring word by a hyphen, and has a technology context: it is the whole
// text, or a technology-context word or another technology stands within three words of it. An
// ambiguous term that fails is returned in rejected, so a caller knows part of the text could not
// be understood.
func (v *Vocabulary) scan(text string) (hits []hit, rejected []int, toks []token) {
	toks = tokenize(text)
	if v == nil {
		return nil, nil, toks
	}
	var candidates []hit
	covered := make([]bool, len(toks))
	for i := 0; i < len(toks); i++ {
		for n := v.maxWords; n >= 1; n-- {
			if i+n > len(toks) {
				continue
			}
			key := make([]string, n)
			for k := 0; k < n; k++ {
				key[k] = toks[i+k].lower
			}
			info, ok := v.terms[strings.Join(key, " ")]
			if !ok {
				continue
			}
			h := hit{group: info.group, from: i, to: i + n}
			if info.ambiguous {
				candidates = append(candidates, h)
			} else {
				hits = append(hits, h)
				for k := i; k < i+n; k++ {
					covered[k] = true
				}
			}
			i += n - 1
			break
		}
	}
	whole := len(toks) == 1
	for _, h := range candidates {
		info := v.terms[joinLower(toks[h.from:h.to])]
		if v.ambiguousAccepted(text, toks, covered, h, info, whole) {
			hits = append(hits, h)
		} else {
			rejected = append(rejected, h.from)
		}
	}
	return hits, rejected, toks
}

func joinLower(toks []token) string {
	parts := make([]string, len(toks))
	for i, t := range toks {
		parts[i] = t.lower
	}
	return strings.Join(parts, " ")
}

// isFullVocabMatch reports whether name is exactly a known vocabulary technology — the whole phrase
// must match a single terms entry, not just contain one. Used by IsSkillName and isAlternativeName
// to avoid false positives from substring matches (e.g. "Familiarity with Go" contains "Go" but is
// not a vocabulary technology).
func (v *Vocabulary) isFullVocabMatch(name string) bool {
	if v == nil {
		return false
	}
	name = strings.TrimSpace(name)
	toks := tokenize(name)
	if len(toks) == 0 {
		return false
	}
	_, ok := v.terms[joinLower(toks)]
	return ok
}

func (v *Vocabulary) ambiguousAccepted(text string, toks []token, covered []bool, h hit, info termInfo, whole bool) bool {
	first, last := toks[h.from], toks[h.to-1]
	if text[first.start:last.end] != info.exact {
		return false // "go to market" is not Go: the configured spelling is "Go"
	}
	if joinedByHyphen(text, first.start, last.end) {
		return false // Go-to-market, C-level
	}
	if whole {
		return true
	}
	for k := h.from - 3; k <= h.to+2; k++ {
		if k < 0 || k >= len(toks) || (k >= h.from && k < h.to) {
			continue
		}
		// A technology-context word within three words, or a technology within two, shows what the
		// ambiguous name is.
		near := abs(k-h.from) <= 2 || abs(k-(h.to-1)) <= 2
		if v.context[toks[k].lower] || (covered[k] && near) {
			return true
		}
	}
	return false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// joinedByHyphen reports whether the word at [start, end) is glued by a hyphen to a letter on
// either side.
func joinedByHyphen(text string, start, end int) bool {
	before := []rune(text[:start])
	if n := len(before); n >= 2 && before[n-1] == '-' && isWordRune(before[n-2]) {
		return true
	}
	after := []rune(text[end:])
	return len(after) >= 2 && after[0] == '-' && isWordRune(after[1])
}

var parenTail = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

// scanName reads a short name that stands alone: a profile skill ("Go (expert)") or one member of a
// list of alternatives ("Go"). Standing alone is a technology context, so an ambiguous name counts
// there whatever its case.
func (v *Vocabulary) scanName(name string) []hit {
	if v == nil {
		return nil
	}
	name = strings.TrimSpace(parenTail.ReplaceAllString(strings.TrimSpace(name), ""))
	toks := tokenize(name)
	if len(toks) == 0 {
		return nil
	}
	if info, ok := v.terms[joinLower(toks)]; ok {
		return []hit{{group: info.group, from: 0, to: len(toks)}}
	}
	hits, _, _ := v.scan(name)
	return hits
}

func groupsOf(hits []hit) []string {
	var out []string
	seen := map[string]bool{}
	for _, h := range hits {
		if !seen[h.group] {
			seen[h.group] = true
			out = append(out, h.group)
		}
	}
	return out
}

// profileIndex says, for each technology group, which profile fact names it. A fact in the skills
// list wins over a mention in prose.
type profileIndex struct {
	byGroup map[string]int // group -> index into the facts
	// absentByGroup and absentByName say which facts record a skill the owner does not have, by
	// technology group and by the normalised name as the owner saw it on a vacancy.
	absentByGroup map[string]int
	absentByName  map[string]int
	// haveByName maps the normalised skill name (as offered on a vacancy) to its index in facts,
	// for skills the owner added by name that are not in the vocabulary. The same normalisation as
	// absentByName is used (skillOfferName), so a name added from one vacancy is recognised on another.
	haveByName map[string]int
}

func (v *Vocabulary) indexProfile(facts []matchcontract.ProfileFact) profileIndex {
	idx := profileIndex{byGroup: map[string]int{}, absentByGroup: map[string]int{}, absentByName: map[string]int{}, haveByName: map[string]int{}}
	if v == nil {
		return idx
	}
	skills := map[string]int{}
	prose := map[string]int{}
	for i, f := range facts {
		switch f.Field {
		case profile.MatchFieldSkills:
			for _, g := range groupsOf(v.scanName(f.Value)) {
				if _, ok := skills[g]; !ok {
					skills[g] = i
				}
			}
			// Also index by offered name for skills not in the vocabulary (or vocabulary skills added
			// by name). haveByName uses the same normalisation as absentByName for cross-vacancy matching.
			// The fact value is "Name" or "Name (Level)"; strip the level suffix before normalising so
			// that "Open source contributions (Proficient)" matches the requirement "Open source contributions".
			factName := f.Value
			if j := strings.Index(factName, " ("); j > 0 {
				factName = factName[:j]
			}
			if name := skillOfferName(factName); name != "" {
				if _, ok := idx.haveByName[name]; !ok {
					idx.haveByName[name] = i
				}
			}
		case profile.MatchFieldSkillsAbsent:
			name := strings.TrimPrefix(f.Value, profile.AbsentSkillPrefix)
			idx.absentByName[skillOfferName(name)] = i
			for _, g := range groupsOf(v.scanName(name)) {
				if _, ok := idx.absentByGroup[g]; !ok {
					idx.absentByGroup[g] = i
				}
			}
		case profile.MatchFieldHeadline, profile.MatchFieldExperience:
			hits, _, _ := v.scan(f.Value)
			for _, g := range groupsOf(hits) {
				if _, ok := prose[g]; !ok {
					prose[g] = i
				}
			}
		}
	}
	for g, i := range prose {
		idx.byGroup[g] = i
	}
	for g, i := range skills {
		idx.byGroup[g] = i
	}
	return idx
}

var yearsRe = regexp.MustCompile(`(?i)\b\d+\s*\+?\s*(years?|yrs?)\b`)

// codeMatch decides a requirement that names technologies by looking for them in the profile. It
// says "match" when every named technology is in the profile (for a list of alternatives, or
// "X or Y", any one of them), and "partial" when the requirement names several and the profile lists
// some but not all. Anything it cannot fully read, such as an unknown word, an ambiguous name
// without a technology context, or a number of years, is left to the model; no technology found is
// never a verdict.
func (v *Vocabulary) codeMatch(r matchcontract.ExtractedRequirement, facts []matchcontract.ProfileFact, idx profileIndex) (matchcontract.ProfileEvidence, matchcontract.Verdict, string, bool) {
	none := matchcontract.ProfileEvidence{}
	if v == nil || (len(idx.byGroup) == 0 && len(idx.absentByGroup) == 0 && len(idx.absentByName) == 0 && len(idx.haveByName) == 0) {
		return none, "", "", false
	}
	if r.Kind != matchcontract.RequirementSkill && r.Kind != matchcontract.RequirementExperience {
		return none, "", "", false
	}
	if yearsRe.MatchString(r.Text) {
		return none, "", "", false // years of a technology cannot be read from a skills list
	}

	var groups []string
	anyOf := false
	if len(r.Alternatives) > 0 {
		anyOf = true
		for _, alt := range r.Alternatives {
			groups = append(groups, groupsOf(v.scanName(alt))...)
		}
	} else {
		hits, rejected, toks := v.scan(r.Text)
		groups = groupsOf(hits)
		covered := make([]bool, len(toks))
		for _, h := range hits {
			for k := h.from; k < h.to; k++ {
				covered[k] = true
			}
		}
		for _, k := range rejected {
			covered[k] = false
		}
		orPresent := false
		leftover := len(rejected) > 0
		for k, t := range toks {
			if covered[k] {
				continue
			}
			if t.lower == "or" {
				orPresent = true
				continue
			}
			if len([]rune(t.lower)) < 3 || v.generic[t.lower] || v.context[t.lower] {
				continue
			}
			leftover = true
		}
		anyOf = orPresent
		if leftover && !anyOf {
			// Part of the requirement is not a technology we know. Code can still decide when the
			// owner explicitly named this exact wording as a skill they have or do not have.
			name := skillOfferName(r.Text)
			if i, ok := idx.haveByName[name]; ok {
				f := facts[i]
				return matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}, matchcontract.VerdictMatch, ReasonSkillInProfile, true
			}
			if i, ok := idx.absentByName[name]; ok {
				return absentEvidence(facts, i), matchcontract.VerdictNo, ReasonMarkedAbsent, true
			}
			// Try splitting the requirement text as a comma-separated list and check each part in
			// haveByName. indexProfile indexes every MatchFieldSkills fact (vocabulary and non-vocab)
			// into haveByName, so this covers profile skills added both via vocabulary and by name.
			// "Python, Postgres, AWS, distributed systems" → check each part → all found → match.
			parts := splitSkillParts(r.Text)
			if len(parts) > 1 {
				matchCount, firstIdx := 0, -1
				for _, p := range parts {
					pName := skillOfferName(p)
					if i, ok := idx.haveByName[pName]; ok {
						matchCount++
						if firstIdx < 0 {
							firstIdx = i
						}
					}
				}
				if firstIdx >= 0 {
					f := facts[firstIdx]
					ev := matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}
					if matchCount == len(parts) {
						return ev, matchcontract.VerdictMatch, ReasonSkillInProfile, true
					}
					return ev, matchcontract.VerdictPartial, ReasonSomeNamedSkills, true
				}
				// No part found in haveByName. Check if every comma-part is individually absent:
				// pressing ✕ on "Python, Postgres" now records each part separately.
				allPartsAbsent, firstAbsentIdx := true, -1
				for _, p := range parts {
					i, ok := v.absentFact(p, idx)
					if !ok {
						allPartsAbsent = false
						break
					}
					if firstAbsentIdx < 0 {
						firstAbsentIdx = i
					}
				}
				if allPartsAbsent && firstAbsentIdx >= 0 {
					return absentEvidence(facts, firstAbsentIdx), matchcontract.VerdictNo, ReasonMarkedAbsent, true
				}
			}
			// Also try splitting by "/" for slash-separated alternatives (e.g. "Datadog / OpenSearch").
			// splitSkillParts above splits only on "," and ";"; slash is a separate separator used for
			// alternatives. For alternatives, any one part in haveByName → match (one is enough).
			// Guard: every part must pass isAlternativeName (vocabulary-known OR single-word
			// IsSkillName), consistent with ClassifySkillAdd, to avoid false positives from "/" in
			// phrases like "research/analysis capabilities".
			if strings.Contains(r.Text, "/") {
				slashParts := splitBySlash(r.Text)
				if len(slashParts) > 1 {
					allValidSlashAlts := true
					for _, p := range slashParts {
						if !v.isAlternativeName(p) {
							allValidSlashAlts = false
							break
						}
					}
					if allValidSlashAlts {
						for _, p := range slashParts {
							if i, ok := idx.haveByName[skillOfferName(p)]; ok {
								f := facts[i]
								return matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}, matchcontract.VerdictMatch, ReasonSkillInProfile, true
							}
						}
						// No slash-part found in haveByName. Check if every part is individually absent:
						// pressing ✕ on "Datadog / OpenSearch" now records each part separately.
						allSlashAbsent, firstSlashAbsentIdx := true, -1
						for _, p := range slashParts {
							i, ok := v.absentFact(p, idx)
							if !ok {
								allSlashAbsent = false
								break
							}
							if firstSlashAbsentIdx < 0 {
								firstSlashAbsentIdx = i
							}
						}
						if allSlashAbsent && firstSlashAbsentIdx >= 0 {
							return absentEvidence(facts, firstSlashAbsentIdx), matchcontract.VerdictNo, ReasonMarkedAbsent, true
						}
					}
				}
			}
			return none, "", "", false
		}
	}
	groups = dedupe(groups)
	var first *int
	found := 0
	for _, g := range groups {
		if i, ok := idx.byGroup[g]; ok {
			found++
			if first == nil {
				first = &i
			}
		}
	}
	if found > 0 {
		verdict, reason := matchcontract.VerdictMatch, ReasonNamedSkill
		if !anyOf && found != len(groups) {
			verdict, reason = matchcontract.VerdictPartial, ReasonSomeNamedSkills // some named, not all
		}
		f := facts[*first]
		return matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}, verdict, reason, true
	}
	// Nothing named is among the skills the owner has. When every named skill is one they said they
	// do not have, the answer is theirs: "no". Also check haveByName before giving up: the owner may
	// have added a skill by name that the vocabulary does not know (groups is empty) or that is in the
	// vocabulary but not in idx.byGroup (vocabulary skill added by name, groups non-empty but found==0).
	if len(groups) == 0 {
		if i, ok := idx.haveByName[skillOfferName(r.Text)]; ok {
			f := facts[i]
			return matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}, matchcontract.VerdictMatch, ReasonSkillInProfile, true
		}
	}
	// For "X or Y" alternatives where no vocabulary group was found in the profile (found==0),
	// check each or-part in haveByName. Any one part in the profile → match (for alternatives,
	// one is enough). This handles the case where one alternative is vocabulary-known (e.g.
	// "Elixir") but the owner chose the non-vocabulary one (e.g. "Erlang"), so groups is
	// non-empty but found==0 and the len(groups)==0 block above was skipped.
	//
	// Guard: every part must pass isAlternativeName (vocabulary-known OR single-word IsSkillName),
	// matching the check ClassifySkillAdd uses. This prevents "Go or no go decisions" from matching
	// via haveByName["go"] when the vocabulary's ambiguous-name rule correctly rejected "Go" as a
	// technology name in that phrase.
	//
	// Also check the full phrase in absentByName for the ✕ case: allAbsent may return early when
	// vocabulary groups prevent it from reaching its own absentByName fallback.
	if anyOf && found == 0 {
		orParts := splitByOr(r.Text)
		allValidAlts := len(orParts) >= 2
		for _, p := range orParts {
			if !v.isAlternativeName(p) {
				allValidAlts = false
				break
			}
		}
		if allValidAlts {
			for _, p := range orParts {
				if i, ok := idx.haveByName[skillOfferName(p)]; ok {
					f := facts[i]
					return matchcontract.ProfileEvidence{Field: f.Field, Value: f.Value}, matchcontract.VerdictMatch, ReasonSkillInProfile, true
				}
			}
			// Check full phrase in absentByName (backward compat: ✕ used to store the whole phrase).
			if i, ok := idx.absentByName[skillOfferName(r.Text)]; ok {
				return absentEvidence(facts, i), matchcontract.VerdictNo, ReasonMarkedAbsent, true
			}
			// Check if every individual or-part is absent (new behaviour: ✕ now stores each part).
			allOrAbsent, firstOrAbsentIdx := true, -1
			for _, p := range orParts {
				i, ok := v.absentFact(p, idx)
				if !ok {
					allOrAbsent = false
					break
				}
				if firstOrAbsentIdx < 0 {
					firstOrAbsentIdx = i
				}
			}
			if allOrAbsent && firstOrAbsentIdx >= 0 {
				return absentEvidence(facts, firstOrAbsentIdx), matchcontract.VerdictNo, ReasonMarkedAbsent, true
			}
		}
	}
	if i, ok := v.allAbsent(r, groups, idx); ok {
		return absentEvidence(facts, i), matchcontract.VerdictNo, ReasonMarkedAbsent, true
	}
	return none, "", "", false
}

// allAbsent reports whether every skill a requirement names is one the owner marked as not theirs,
// and the fact that records the first of them. For a group of alternatives every member must be
// marked, so a member the owner has not ruled out leaves the question to the model. A skill is known
// by its technology group when the vocabulary has one and by its offered name otherwise.
func (v *Vocabulary) allAbsent(r matchcontract.ExtractedRequirement, groups []string, idx profileIndex) (int, bool) {
	first := -1
	if len(r.Alternatives) > 0 {
		for _, alt := range r.Alternatives {
			i, ok := v.absentFact(alt, idx)
			if !ok {
				return 0, false
			}
			if first < 0 {
				first = i
			}
		}
		return first, first >= 0
	}
	for _, g := range groups {
		i, ok := idx.absentByGroup[g]
		if !ok {
			return 0, false
		}
		if first < 0 {
			first = i
		}
	}
	if first >= 0 {
		return first, true
	}
	i, ok := idx.absentByName[skillOfferName(r.Text)]
	return i, ok
}

// absentFact finds the fact that records one named skill as absent.
func (v *Vocabulary) absentFact(name string, idx profileIndex) (int, bool) {
	for _, g := range groupsOf(v.scanName(name)) {
		if i, ok := idx.absentByGroup[g]; ok {
			return i, true
		}
	}
	i, ok := idx.absentByName[skillOfferName(name)]
	return i, ok
}

func absentEvidence(facts []matchcontract.ProfileFact, i int) matchcontract.ProfileEvidence {
	return matchcontract.ProfileEvidence{Field: facts[i].Field, Value: facts[i].Value}
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// answerableFromCV reports whether a CV could state what a requirement asks. Soft skills, team and
// company size, work authorization and availability are not something a CV usually holds, so a
// "no" on them is only "unclear".
func (v *Vocabulary) answerableFromCV(r matchcontract.ExtractedRequirement) bool {
	if v == nil {
		return true
	}
	text := strings.ToLower(r.Text + " " + strings.Join(r.Alternatives, " "))
	for _, p := range v.notAnswerable {
		if strings.Contains(text, p) {
			return false
		}
	}
	return true
}

var (
	offerPrefix = regexp.MustCompile(`(?i)^(?:(?:strong|deep|solid|proven|good|excellent|advanced|expert|hands-on)\s+)?(?:(?:experience|knowledge|proficiency|familiarity|expertise)\s+(?:with|in|of)\s+)?(?:(?:strong|deep|solid|proven|good|excellent|advanced|expert|hands-on)\s+)?`)
	offerSuffix = regexp.MustCompile(`(?i)\s+(?:expertise|experience|skills?|proficiency|knowledge|familiarity|background)$`)
)

// skillOfferName reduces a skill name to the form the vacancy panel offers it in (lower case, one
// space, no leading "strong" or "experience with" and no trailing "skills"), so a skill the owner
// marked from one vacancy is recognised on another that words it a little differently.
func skillOfferName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	name = offerSuffix.ReplaceAllString(offerPrefix.ReplaceAllString(name, ""), "")
	return strings.ToLower(strings.TrimSpace(name))
}
