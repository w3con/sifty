package profile_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// TestSectionUpdateIsAtomicAndNoOp catches the hidden failures section editing can cause without
// being visible in the form: a concurrent save overwriting another section, losing server-owned
// legacy/provenance data, a no-op Save writing a new revision and rescoring, or a section merge
// clobbering a value that an accepted fact wrote. It also proves a fact can still be accepted
// after section writes, so the existing CAS/late-analysis protection remains intact.
func TestSectionUpdateIsAtomicAndNoOp(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	if err := s.SaveContext(map[string]any{
		"version": "legacy-context-v1",
		"preferences": map[string]any{
			"context_only": "kept",
		},
	}); err != nil {
		t.Fatalf("save context: %v", err)
	}
	cfg := &config.Config{Search: config.Search{Preferences: map[string]any{
		"roles":        []any{"Legacy Role"},
		"unknown_flag": true,
	}}}
	svc := profile.NewService(s, config.NewHolder(cfg))

	got, err := svc.Get()
	if err != nil {
		t.Fatalf("bootstrap profile: %v", err)
	}

	// Seed a document-sourced skill directly so the test starts with server-owned provenance;
	// the public full-profile Update deliberately marks client-supplied skills as manual.
	seeded := got.Data
	seeded.Skills = []profile.Skill{{
		Key: "go", Name: "Go", Level: profile.SkillProduction,
		Provenance: &profile.Provenance{Kind: "document", Reference: "doc-1", Quote: "Go"},
	}}
	seeded.TargetRoles = []string{"Backend Engineer"}
	encoded, err := json.Marshal(seeded)
	if err != nil {
		t.Fatalf("marshal seed profile: %v", err)
	}
	seededRecord, err := s.SaveProfile(profile.LocalProfileID, got.Revision, string(encoded), "seed", "test")
	if err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	identityValue := json.RawMessage(`{"displayName":"Owner","headline":"Engineer"}`)
	skillsValue := json.RawMessage(`[
		{"key":"go","name":"Go","level":"Production"},
		{"key":"rust","name":"Rust","level":"Learning","context":"systems"}
	]`)

	// Two different sections save concurrently. Both transactions compose on the latest row;
	// neither may replace the other or the seeded target roles/legacy provenance.
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, item := range []struct {
		section string
		value   json.RawMessage
	}{
		{"identity", identityValue},
		{"skills", skillsValue},
	} {
		wg.Add(1)
		go func(section string, value json.RawMessage) {
			defer wg.Done()
			_, changed, err := svc.UpdateSection(section, value)
			if err != nil {
				errs <- err
				return
			}
			if !changed {
				errs <- fmt.Errorf("section %s reported no change", section)
			}
		}(item.section, item.value)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent section update: %v", err)
	}

	current, err := svc.Get()
	if err != nil {
		t.Fatalf("get after concurrent updates: %v", err)
	}
	if current.Revision != seededRecord.Revision+2 {
		t.Errorf("revision after two section saves = %d, want %d", current.Revision, seededRecord.Revision+2)
	}
	if current.Data.Identity.DisplayName != "Owner" {
		t.Errorf("identity display name = %q, want Owner", current.Data.Identity.DisplayName)
	}
	if len(current.Data.TargetRoles) != 1 || current.Data.TargetRoles[0] != "Backend Engineer" {
		t.Errorf("target roles = %v, want the seeded value preserved", current.Data.TargetRoles)
	}
	byName := map[string]profile.Skill{}
	for _, skill := range current.Data.Skills {
		byName[skill.Name] = skill
	}
	if goSkill, ok := byName["Go"]; !ok || goSkill.Provenance == nil || goSkill.Provenance.Kind != "document" {
		t.Errorf("unchanged Go skill lost its document provenance: %+v", byName["Go"])
	}
	if rust, ok := byName["Rust"]; !ok || rust.Provenance == nil || rust.Provenance.Kind != "manual" {
		t.Errorf("new Rust skill provenance = %+v, want manual", byName["Rust"])
	}
	if current.Data.LegacyContextVersion != "legacy-context-v1" {
		t.Errorf("legacyContextVersion = %q, want preserved", current.Data.LegacyContextVersion)
	}
	if current.Data.LegacyPreferences["unknown_flag"] != true {
		t.Errorf("legacyPreferences lost server-owned value: %v", current.Data.LegacyPreferences)
	}

	// The exact normalized section is a no-op: no revision, no history row, no derived rescore.
	noOpSkills := json.RawMessage(`[
		{"key":"go","name":"Go","level":"Production"},
		{"key":"rust","name":"Rust","level":"Learning","context":"systems"}
	]`)
	beforeRevision := current.Revision
	beforeContext := current.ContextVersion
	var beforeRows int
	if err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM profile_revisions WHERE profile_id=?`, profile.LocalProfileID).Scan(&beforeRows); err != nil {
		t.Fatalf("count revisions before no-op: %v", err)
	}
	noOpProfile, changed, err := svc.UpdateSection("skills", noOpSkills)
	if err != nil {
		t.Fatalf("no-op section update: %v", err)
	}
	if changed {
		t.Error("unchanged normalized section reported a change")
	}
	if noOpProfile.Revision != beforeRevision {
		t.Errorf("no-op revision = %d, want %d", noOpProfile.Revision, beforeRevision)
	}
	if noOpProfile.ContextVersion != beforeContext {
		t.Errorf("no-op changed derived context version from %q to %q", beforeContext, noOpProfile.ContextVersion)
	}
	var afterRows int
	if err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM profile_revisions WHERE profile_id=?`, profile.LocalProfileID).Scan(&afterRows); err != nil {
		t.Fatalf("count revisions after no-op: %v", err)
	}
	if afterRows != beforeRows {
		t.Errorf("no-op wrote %d new revision rows", afterRows-beforeRows)
	}

	// A fact acceptance still works after the section writes and its value survives a later
	// section merge. This is the existing CAS-protected fact path, not a second write path.
	docID := "doc-2"
	if err := s.CreateProfileDocument(&store.ProfileDocumentRecord{
		ID: docID, ProfileID: profile.LocalProfileID, Kind: "md", OriginalName: "cv.md",
		ProcessingState: "analyzing", ExtractedText: "Berlin",
	}); err != nil {
		t.Fatalf("create document: %v", err)
	}
	if err := s.StartProfileDocumentAnalysis(docID, `{"runId":"run-2"}`); err != nil {
		t.Fatalf("start analysis: %v", err)
	}
	fact := &store.ProfileFactRecord{
		ID: "fact-2", ProfileID: profile.LocalProfileID, DocumentID: &docID,
		FieldKey: "availability.timezone", ValueJSON: `"Europe/Berlin"`,
		Quote: "Berlin", Origin: "document", State: profile.FactProposed,
	}
	published, err := s.CompleteProfileDocumentAnalysis(
		docID, "run-2", "analyzed", `{"runId":"run-2"}`, "", []*store.ProfileFactRecord{fact})
	if err != nil || !published {
		t.Fatalf("complete analysis: published=%v err=%v", published, err)
	}
	facts, err := s.ListProfileFacts(profile.LocalProfileID, "")
	if err != nil || len(facts) != 1 {
		t.Fatalf("facts = %d, err=%v", len(facts), err)
	}
	decision, err := svc.AcceptFact(facts[0].ID, noOpProfile.Revision)
	if err != nil {
		t.Fatalf("accept fact after section writes: %v", err)
	}
	if !decision.ProfileChanged || decision.Profile == nil {
		t.Fatalf("fact acceptance did not change the profile: %+v", decision)
	}
	if decision.Profile.Data.Availability.Timezone != "Europe/Berlin" {
		t.Errorf("accepted fact timezone = %q, want Europe/Berlin", decision.Profile.Data.Availability.Timezone)
	}

	afterFact, _, err := svc.UpdateSection("identity", json.RawMessage(`{"displayName":"Owner 2","headline":"Engineer"}`))
	if err != nil {
		t.Fatalf("identity update after fact: %v", err)
	}
	if afterFact.Data.Availability.Timezone != "Europe/Berlin" {
		t.Errorf("section merge dropped the accepted fact value: %q", afterFact.Data.Availability.Timezone)
	}
	acceptedFacts, err := s.ListProfileFacts(profile.LocalProfileID, "")
	if err != nil || len(acceptedFacts) != 1 || acceptedFacts[0].State != profile.FactAccepted {
		t.Errorf("accepted fact state after section merge = %+v, err=%v", acceptedFacts, err)
	}
}

// TestSectionUpdateRejectsPartialExperience catches a silently persisted half-role: the browser
// keeps a partial row visible after the server refuses it, instead of dropping the row or storing
// it as a claimed experience fact.
func TestSectionUpdateRejectsPartialExperience(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()
	svc := profile.NewService(s, config.NewHolder(&config.Config{}))

	before, err := svc.Get()
	if err != nil {
		t.Fatalf("bootstrap profile: %v", err)
	}
	_, changed, err := svc.UpdateSection("experience", json.RawMessage(`[{"company":"","title":"","start":"2020","summary":"Only summary"}]`))
	if err == nil || !strings.Contains(err.Error(), "Experience 1: enter a company and title") {
		t.Fatalf("partial experience update error = %v, want a human field message", err)
	}
	if changed {
		t.Fatal("partial experience reported a change")
	}
	after, err := svc.Get()
	if err != nil {
		t.Fatalf("get after rejected update: %v", err)
	}
	if after.Revision != before.Revision {
		t.Errorf("rejected partial experience wrote revision %d, want %d", after.Revision, before.Revision)
	}
	if len(after.Data.Experience) != 0 {
		t.Errorf("partial experience was persisted: %+v", after.Data.Experience)
	}

	// A legacy partial row already stored under the older contract must not block an unchanged
	// Save or another section: normalized no-op still writes nothing and the row is preserved.
	seeded := after.Data
	seeded.Experience = []profile.Experience{{Summary: "Legacy summary", Skills: []string{}}}
	encoded, err := json.Marshal(seeded)
	if err != nil {
		t.Fatalf("marshal legacy partial profile: %v", err)
	}
	seedRecord, err := s.SaveProfile(profile.LocalProfileID, after.Revision, string(encoded), "seed legacy partial", "test")
	if err != nil {
		t.Fatalf("seed legacy partial experience: %v", err)
	}
	legacy, err := svc.Get()
	if err != nil {
		t.Fatalf("get seeded legacy partial: %v", err)
	}
	if len(legacy.Data.Experience) != 1 {
		t.Fatalf("seeded legacy partial experience = %+v, want one row", legacy.Data.Experience)
	}
	unchanged, changed, err := svc.UpdateSection("experience", json.RawMessage(`[{"company":"","title":"","summary":"Legacy summary","skills":[]}]`))
	if err != nil {
		t.Fatalf("unchanged legacy partial Save must be a no-op, got error: %v", err)
	}
	if changed || unchanged.Revision != seedRecord.Revision {
		t.Errorf("unchanged legacy partial changed=%v revision=%d, want no-op at %d", changed, unchanged.Revision, seedRecord.Revision)
	}
	if len(unchanged.Data.Experience) != 1 || unchanged.Data.Experience[0].Summary != "Legacy summary" {
		t.Errorf("unchanged legacy partial row was not preserved: %+v", unchanged.Data.Experience)
	}
	other, otherChanged, err := svc.UpdateSection("identity", json.RawMessage(`{"displayName":"Owner","headline":""}`))
	if err != nil {
		t.Fatalf("another-section Save must not be blocked by the legacy partial row: %v", err)
	}
	if !otherChanged {
		t.Fatal("another-section Save reported no change")
	}
	if other.Revision != seedRecord.Revision+1 || len(other.Data.Experience) != 1 {
		t.Errorf("another-section Save revision=%d experience=%+v, want revision %d with the legacy row", other.Revision, other.Data.Experience, seedRecord.Revision+1)
	}
}
