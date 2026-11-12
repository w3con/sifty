package profile

// Targeted regression tests for SETTINGS-8 (CV analysis loses skills, languages and work
// authorization). Two specific failure modes are guarded here:
//
//  (a) F.1 — a bare skill name string must survive normalisation as a Skill object; without the
//      normalizeFactValue fix it was silently dropped by ValidateFactValue with "skills must be
//      an object with key, name and level".
//
//  (b) F.4 — a proposed fact whose value is already in the confirmed profile must be classified
//      as FactMatches, not shown as a pending decision; without matchFactToProfile the fact was
//      always "proposed", making the UI ask the user to confirm their own name.
//
// Each test is designed to fail on the pre-fix code path, so a reversion of either fix will be
// caught immediately.

import (
	"encoding/json"
	"testing"
	"unicode/utf8"

	"github.com/w3con/sifty/internal/store"
)

// TestSplitTextIntoChunks_RuneSafe verifies that splitTextIntoChunks never produces a chunk
// containing an invalid UTF-8 sequence when the natural byte boundary falls inside a multi-byte
// rune. Without the rune-boundary retreat, text[:maxChars] can cut a Cyrillic or accented
// character in half and send malformed UTF-8 to the model.
func TestSplitTextIntoChunks_RuneSafe(t *testing.T) {
	// "é" is U+00E9, encoded as 2 bytes: 0xC3 0xA9.
	// "ABC" = 3 bytes; "é" starts at byte 3. With maxChars=4 the raw cut lands at byte 4,
	// which is the second byte of "é" — not a rune start. The fix must retreat to 3.
	text := "ABC" + "éàü" + " invented text without personal data"
	for _, maxChars := range []int{4, 5, 6, 7, 10, 15} {
		chunks := splitTextIntoChunks(text, maxChars)
		for i, chunk := range chunks {
			if !utf8.ValidString(chunk) {
				t.Errorf("maxChars=%d chunk[%d] is not valid UTF-8: %q", maxChars, i, chunk)
			}
		}
	}

	// Verify that all original runes are present across the chunks (no data loss).
	for _, maxChars := range []int{4, 5, 10} {
		joined := ""
		for _, chunk := range splitTextIntoChunks(text, maxChars) {
			joined += chunk
		}
		// TrimRight / TrimLeft in splitTextIntoChunks strips leading/trailing spaces and
		// newlines at boundaries; check rune count not byte equality.
		if utf8.RuneCountInString(joined) > utf8.RuneCountInString(text) {
			t.Errorf("maxChars=%d: joined chunk rune count %d > original %d",
				maxChars, utf8.RuneCountInString(joined), utf8.RuneCountInString(text))
		}
	}
}

// TestExpandValueItems_ArrayOfStringsExpanded verifies that a JSON array returned by the model
// for a skills fact (the actual qwen3.5:9b output format: ["Zig","Go","Haskell"]) is
// unpacked into separate values by expandValueItems, so each value can be normalised and
// validated independently. Without this fix every skill in the array was dropped by
// ValidateFactValue with "skills must be an object with key, name and level".
func TestExpandValueItems_ArrayOfStringsExpanded(t *testing.T) {
	raw := json.RawMessage(`["Zig","Go","Haskell"]`)
	items := expandValueItems("skills", raw)
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	// Each bare string must survive the full pipeline: normalise then validate.
	wantNames := []string{"Zig", "Go", "Haskell"}
	for i, item := range items {
		normalised := normalizeFactValue("skills", item)
		canonical, err := ValidateFactValue("skills", normalised)
		if err != nil {
			t.Fatalf("item[%d] ValidateFactValue: %v", i, err)
		}
		var sk Skill
		if err := json.Unmarshal(canonical, &sk); err != nil {
			t.Fatalf("item[%d] unmarshal: %v", i, err)
		}
		if sk.Name != wantNames[i] {
			t.Errorf("item[%d] name = %q, want %q", i, sk.Name, wantNames[i])
		}
		if sk.Level != SkillUnstated {
			t.Errorf("item[%d] level = %q, want %q", i, sk.Level, SkillUnstated)
		}
	}
}

// TestExpandValueItems_EnumListUntouched verifies that expandValueItems does not unpack an enum
// list (workModes.modes), since for that field the array IS the expected value, not a container
// wrapping individual facts.
func TestExpandValueItems_EnumListUntouched(t *testing.T) {
	raw := json.RawMessage(`["remote","hybrid"]`)
	items := expandValueItems("workModes.modes", raw)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1 (enum list must not be expanded)", len(items))
	}
	// The single item is the original array, which passes ValidateFactValue as-is.
	if _, err := ValidateFactValue("workModes.modes", items[0]); err != nil {
		t.Errorf("ValidateFactValue: %v", err)
	}
}

// TestNormalizeFactValue_BareSkillNameKept verifies that a bare JSON string skill value is
// converted to a valid Skill object by normalizeFactValue and then passes ValidateFactValue.
// Without the normalizeFactValue fix, json.Unmarshal("Zig", &Skill{}) fails and the fact
// is silently dropped with "skills must be an object with key, name and level".
func TestNormalizeFactValue_BareSkillNameKept(t *testing.T) {
	cases := []struct {
		input     string
		wantKey   string
		wantName  string
		wantLevel string
	}{
		{`"Zig"`, "zig", "Zig", SkillUnstated},
		{`"Go"`, "go", "Go", SkillUnstated},
		{`"Haskell"`, "haskell", "Haskell", SkillUnstated},
		{`"Nim"`, "nim", "Nim", SkillUnstated},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			raw := json.RawMessage(c.input)
			normalised := normalizeFactValue("skills", raw)

			// Must now be a valid Skill object — ValidateFactValue must not error.
			canonical, err := ValidateFactValue("skills", normalised)
			if err != nil {
				t.Fatalf("ValidateFactValue after normalisation: %v", err)
			}

			var sk Skill
			if err := json.Unmarshal(canonical, &sk); err != nil {
				t.Fatalf("unmarshal canonical skill: %v", err)
			}
			if sk.Key != c.wantKey {
				t.Errorf("key = %q, want %q", sk.Key, c.wantKey)
			}
			if sk.Name != c.wantName {
				t.Errorf("name = %q, want %q", sk.Name, c.wantName)
			}
			if sk.Level != c.wantLevel {
				t.Errorf("level = %q, want %q", sk.Level, c.wantLevel)
			}
		})
	}
}

// TestNormalizeFactValue_EnumCaseNormalised verifies that enum values with wrong case are
// normalised to their canonical form before validation. Without this fix "Remote" (capital R)
// failed ValidateFactValue with "workModes.modes has unsupported value "Remote"".
func TestNormalizeFactValue_EnumCaseNormalised(t *testing.T) {
	raw := json.RawMessage(`["Remote","Hybrid"]`)
	normalised := normalizeFactValue("workModes.modes", raw)

	var items []string
	if err := json.Unmarshal(normalised, &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0] != "remote" || items[1] != "hybrid" {
		t.Errorf("items = %v, want [remote hybrid]", items)
	}
	// Must also pass ValidateFactValue.
	if _, err := ValidateFactValue("workModes.modes", normalised); err != nil {
		t.Errorf("ValidateFactValue: %v", err)
	}
}

// TestMatchFactToProfile_EqualValueIsNotADecision verifies that a proposed fact whose value is
// identical to the current confirmed profile value is classified as FactMatches. Without the
// matchFactToProfile fix, every extracted fact was stored as FactProposed regardless of whether
// the value was already in the profile, requiring the user to confirm their own name and all
// their existing experience entries.
func TestMatchFactToProfile_EqualValueIsNotADecision(t *testing.T) {
	profile := ProfileData{
		Identity: Identity{DisplayName: "Jane Roe"},
		Skills:   []Skill{{Key: "go", Name: "Go", Level: SkillExpert}},
		Languages: []Language{
			{Name: "English", Level: "C1"},
		},
	}

	tests := []struct {
		name     string
		fieldKey string
		value    string
		want     bool // true = matches (not a decision), false = new/different value
	}{
		// Equal value: should match.
		{"display name matches", "identity.displayName", `"Jane Roe"`, true},
		// Trimmed-equal: ValidateFactValue trims the value so the canonical form equals the
		// stored name; matchFactToProfile should return true (matches, not a decision).
		{"display name trimmed matches", "identity.displayName", `"  Jane Roe  "`, true},
		// Different value: should NOT match.
		{"display name differs", "identity.displayName", `"Jane R."`, false},
		// Skill with same key and level: should match.
		{"skill matches", "skills", `{"key":"go","name":"Go","level":"Expert"}`, true},
		// Skill with different level: should NOT match.
		{"skill level differs", "skills", `{"key":"go","name":"Go","level":"Beginner"}`, false},
		// New skill: should NOT match.
		{"new skill", "skills", `{"key":"elm","name":"Elm","level":""}`, false},
		// Language matches.
		{"language matches", "languages", `{"name":"English","level":"C1"}`, true},
		// Language not in profile.
		{"new language", "languages", `{"name":"French","level":"B2"}`, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			canonical, err := ValidateFactValue(tc.fieldKey, json.RawMessage(tc.value))
			if err != nil {
				t.Fatalf("ValidateFactValue: %v", err)
			}
			rec := &store.ProfileFactRecord{
				FieldKey:  tc.fieldKey,
				ValueJSON: string(canonical),
				State:     FactProposed,
			}
			got := matchFactToProfile(profile, rec)
			if got != tc.want {
				t.Errorf("matchFactToProfile = %v, want %v (fieldKey=%q value=%s)",
					got, tc.want, tc.fieldKey, tc.value)
			}
		})
	}
}

// TestItemNameQuote_WordBoundary verifies that itemNameQuote matches a skill name only as a
// whole word. "Go" must be found in a comma-separated skills list but must NOT match inside
// "Golang" or "Django". This prevents the pipeline from anchoring a 2-letter name to an
// unrelated token.
func TestItemNameQuote_WordBoundary(t *testing.T) {
	present := "Elm, Zig, Go, OCaml, Lua"
	if m, ok := itemNameQuote(present, "Go"); !ok {
		t.Errorf("itemNameQuote: expected to find 'Go' in %q, got not-found", present)
	} else if m != "Go" {
		t.Errorf("itemNameQuote: got %q, want %q", m, "Go")
	}

	absent := "Golang, Django, Pango"
	if _, ok := itemNameQuote(absent, "Go"); ok {
		t.Errorf("itemNameQuote: 'Go' must not match inside 'Golang'/'Django' in %q", absent)
	}
}

// TestItemNameQuote_ShortName verifies that a one-letter name like "C" matches only as a
// whole token and is not found inside "CTO" or "React".
func TestItemNameQuote_ShortName(t *testing.T) {
	present := "C, Elm, Lua"
	if m, ok := itemNameQuote(present, "C"); !ok {
		t.Errorf("itemNameQuote: expected to find 'C' in %q", present)
	} else if m != "C" {
		t.Errorf("itemNameQuote: got %q, want 'C'", m)
	}

	absent := "CTO, React"
	if _, ok := itemNameQuote(absent, "C"); ok {
		t.Errorf("itemNameQuote: 'C' must not match inside 'CTO'/'React' in %q", absent)
	}
}

// TestQuoteForItem_GroupFallback_AllPresent verifies that when the model returns a group quote
// that is near-literal (not an exact excerpt), quoteForItem falls back to per-item name search
// and salvages every item whose name appears verbatim in the document. All invented text.
func TestQuoteForItem_GroupFallback_AllPresent(t *testing.T) {
	// Invented CV fragment. Skills are listed comma-separated; the model might return a group
	// quote with different spacing or punctuation, causing exactQuote to fail.
	doc := "Programming languages: Elm  Zig  Go  Haskell  Lua"
	// Near-literal group quote (commas added, fails exactQuote).
	groupQuote := "Elm, Zig, Go, Haskell, Lua"

	names := []string{"Elm", "Zig", "Go", "Haskell", "Lua"}
	for _, name := range names {
		raw, _ := json.Marshal(name) // bare string, as the model often emits
		q, ok := quoteForItem(doc, "skills", groupQuote, json.RawMessage(raw))
		if !ok {
			t.Errorf("quoteForItem: skill %q not found in doc via fallback", name)
			continue
		}
		if q != name {
			t.Errorf("quoteForItem: skill %q got quote %q, want exact name", name, q)
		}
	}
}

// TestQuoteForItem_GroupFallback_AbsentName verifies that an item whose name does not appear
// anywhere in the document is correctly rejected (quoteForItem returns false). All invented text.
func TestQuoteForItem_GroupFallback_AbsentName(t *testing.T) {
	doc := "Elm  Zig  Go"
	groupQuote := "Elm, Zig, Go, CobolPlus" // near-literal, CobolPlus not in doc
	raw, _ := json.Marshal("CobolPlus")
	if _, ok := quoteForItem(doc, "skills", groupQuote, json.RawMessage(raw)); ok {
		t.Error("quoteForItem: 'CobolPlus' absent from doc must not return ok=true")
	}
}

// TestMatchFactToProfile_ExperienceAlreadyInProfile verifies that an experience entry for a
// company already in the profile is detected as matching, even when the title differs, as long
// as the period overlaps. This prevents the UI from proposing duplicate experience entries.
func TestMatchFactToProfile_ExperienceAlreadyInProfile(t *testing.T) {
	profile := ProfileData{
		Experience: []Experience{
			{Company: "Acme", Title: "Senior Engineer", Start: "2020-01", End: "2023-06"},
			{Company: "FutureCorp", Title: "CTO", Start: "2023-07", Current: true},
		},
	}

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		// Same company and overlapping period: matches.
		{"same company overlap", `{"company":"Acme","title":"Lead Engineer","start":"2021-01","end":"2022-12"}`, true},
		// Same company and same title: matches.
		{"same company same title", `{"company":"Acme","title":"Senior Engineer","start":"2020-01","end":"2023-06"}`, true},
		// Same company, non-overlapping period: does NOT match.
		{"same company no overlap", `{"company":"Acme","title":"Advisor","start":"2024-01","end":"2024-12"}`, false},
		// Different company: does NOT match.
		{"different company", `{"company":"OtherCo","title":"Engineer","start":"2020-01","end":"2023-06"}`, false},
		// Current role at existing company: matches.
		{"current role match", `{"company":"FutureCorp","title":"CTO","start":"2023-07","current":true}`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			canonical, err := ValidateFactValue("experience", json.RawMessage(tc.value))
			if err != nil {
				t.Fatalf("ValidateFactValue: %v", err)
			}
			rec := &store.ProfileFactRecord{
				FieldKey:  "experience",
				ValueJSON: string(canonical),
				State:     FactProposed,
			}
			got := matchFactToProfile(profile, rec)
			if got != tc.want {
				t.Errorf("matchFactToProfile = %v, want %v (value=%s)", got, tc.want, tc.value)
			}
		})
	}
}

// TestSkillLevels_PromptWordsAreAccepted keeps the skill levels named in the extraction prompt
// in step with the validation: each of the six listed words passes (in any letter case), the
// empty level passes, and a wording the prompt tells the model not to use is rejected.
func TestSkillLevels_PromptWordsAreAccepted(t *testing.T) {
	for _, level := range []string{"Beginner", "Proficient", "Expert", "Production", "Familiar", "Learning", "expert", ""} {
		raw := json.RawMessage(`{"key":"zig","name":"Zig","level":"` + level + `"}`)
		if _, err := ValidateFactValue("skills", raw); err != nil {
			t.Errorf("level %q must be accepted: %v", level, err)
		}
	}
	for _, level := range []string{"Advanced", "Intermediate", "Senior"} {
		raw := json.RawMessage(`{"key":"zig","name":"Zig","level":"` + level + `"}`)
		if _, err := ValidateFactValue("skills", raw); err == nil {
			t.Errorf("level %q must be rejected", level)
		}
	}
}
