package vacancyai

import (
	"sort"
	"strings"

	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/profile"
)

// stem reduces a word to a short stem so that "managing", "managed" and "management" meet. It is
// meant only for choosing which profile passages to show next to a requirement; the model, not
// this, judges whether the passage answers it.
func stem(w string) string {
	for _, suffix := range []string{"ership", "ing", "ed", "es", "s"} {
		if strings.HasSuffix(w, suffix) && len([]rune(w))-len(suffix) >= 4 {
			w = strings.TrimSuffix(w, suffix)
			break
		}
	}
	if r := []rune(w); len(r) > 5 {
		w = string(r[:5])
	}
	return w
}

// meaningStems returns the stems of the words of a text that say something: generic words
// ("experience", "strong") and one-letter words are dropped.
func (v *Vocabulary) meaningStems(text string) map[string]bool {
	out := map[string]bool{}
	for _, t := range tokenize(text) {
		if len([]rune(t.lower)) < 2 || (v != nil && v.generic[t.lower]) {
			continue
		}
		out[stem(t.lower)] = true
	}
	return out
}

// requirementStems are the stems a passage is compared with: the requirement's own words and those
// of its alternatives, plus every spelling of a technology it names, so a requirement for Golang
// finds the skill "Go".
func (v *Vocabulary) requirementStems(r matchcontract.ExtractedRequirement) map[string]bool {
	text := r.Text + " " + strings.Join(r.Alternatives, " ")
	out := v.meaningStems(text)
	if v == nil {
		return out
	}
	hits, _, _ := v.scan(r.Text)
	for _, alt := range r.Alternatives {
		hits = append(hits, v.scanName(alt)...)
	}
	groups := groupsOf(hits)
	// An ordinary word that stands for a kind of technology ("cloud") brings that kind's names.
	for _, t := range tokenize(text) {
		groups = append(groups, v.categoryByStem[stem(t.lower)]...)
	}
	for _, g := range dedupe(groups) {
		for _, term := range v.groupTerms[g] {
			for s := range v.meaningStems(term) {
				out[s] = true
			}
		}
	}
	return out
}

// closestPassages picks, for one requirement, the indexes of the k profile facts that share the
// most meaningful words with it (and with the synonyms of the technologies it names). Computed
// facts are never candidates here: they are offered with every requirement. A fact that shares
// nothing is not offered, so a requirement the profile does not touch is shown no passage at all.
func (v *Vocabulary) closestPassages(r matchcontract.ExtractedRequirement, facts []matchcontract.ProfileFact, k int) []int {
	want := v.requirementStems(r)
	if len(want) == 0 || k <= 0 {
		return nil
	}
	type scored struct{ index, score int }
	var all []scored
	for i, f := range facts {
		if profile.IsComputedField(f.Field) {
			continue
		}
		value := strings.TrimPrefix(f.Value, profile.AbsentSkillPrefix)
		have := v.meaningStems(parenTail.ReplaceAllString(value, ""))
		score := 0
		for s := range want {
			if have[s] {
				score++
			}
		}
		if score > 0 {
			all = append(all, scored{i, score})
		}
	}
	sort.SliceStable(all, func(a, b int) bool { return all[a].score > all[b].score })
	if len(all) > k {
		all = all[:k]
	}
	out := make([]int, len(all))
	for i, s := range all {
		out[i] = s.index
	}
	sort.Ints(out)
	return out
}
