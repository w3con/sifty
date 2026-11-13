package profile_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// Adding a skill from a vacancy must neither lose an existing skill, nor lose one of two additions
// made at the same moment (both would silently vanish from the owner's profile), nor write a new
// revision (and so re-run every match) when the skill is already there.
func TestAddSkillKeepsEverySkillAndIgnoresADuplicate(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := profile.NewService(s, config.NewHolder(&config.Config{}))
	base, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	seeded := base.Data
	seeded.Skills = []profile.Skill{{Key: "go", Name: "Go", Level: "Expert"}}
	if _, err := svc.Update(base.Revision, seeded); err != nil {
		t.Fatalf("seed: %v", err)
	}

	added, changed, err := svc.AddSkill("  Kubernetes  ", "")
	if err != nil || !changed {
		t.Fatalf("AddSkill: changed=%v err=%v", changed, err)
	}
	names := map[string]string{}
	for _, k := range added.Data.Skills {
		names[k.Name] = k.Level
	}
	if names["Go"] != "Expert" || names["Kubernetes"] != profile.SkillProficient || len(names) != 2 {
		t.Fatalf("skills after adding: %v", names)
	}

	again, changed, err := svc.AddSkill("kubernetes", "Beginner")
	if err != nil || changed || again.Revision != added.Revision {
		t.Fatalf("a skill that is already there must change nothing: changed=%v err=%v", changed, err)
	}
	for _, k := range again.Data.Skills {
		if k.Name == "Kubernetes" && k.Level != profile.SkillProficient {
			t.Fatalf("an existing skill keeps its level: %+v", k)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, _, err := svc.AddSkill(fmt.Sprintf("Skill %d", i), ""); err != nil {
				t.Errorf("concurrent AddSkill %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	final, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(final.Data.Skills); got != 2+6 {
		t.Fatalf("every concurrent addition must survive: %d skills", got)
	}

	for _, bad := range []struct{ name, level string }{{"   ", ""}, {strings.Repeat("x", 200), ""}, {"Rust", "Wizard"}} {
		if _, _, err := svc.AddSkill(bad.name, bad.level); err == nil {
			t.Errorf("AddSkill(%q, %q) must be refused", bad.name[:min(len(bad.name), 10)], bad.level)
		}
	}
}

// A skill the owner marks as one they do not have must stay out of the skills they have, must not
// be recorded twice, and must disappear from the list when they add it after all; and it reaches the
// matcher as a fact. Otherwise the profile contradicts itself and a requirement gets two answers.
func TestAbsentSkillsStaySeparateFromSkillsTheOwnerHas(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := profile.NewService(s, config.NewHolder(&config.Config{}))
	if _, err := svc.Get(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.AddSkill("Go", ""); err != nil {
		t.Fatal(err)
	}

	got, changed, err := svc.AddAbsentSkill("  Kubernetes ")
	if err != nil || !changed || len(got.Data.AbsentSkills) != 1 || got.Data.AbsentSkills[0].Name != "Kubernetes" {
		t.Fatalf("AddAbsentSkill: changed=%v err=%v absent=%+v", changed, err, got.Data.AbsentSkills)
	}
	if _, changed, err := svc.AddAbsentSkill("kubernetes"); err != nil || changed {
		t.Fatalf("a skill already recorded changes nothing: changed=%v err=%v", changed, err)
	}
	if _, _, err := svc.AddAbsentSkill("go"); err == nil {
		t.Fatal("a skill the owner has cannot also be one they do not have")
	}
	var found bool
	for _, f := range profile.MatchFacts(got) {
		if f.Field == profile.MatchFieldSkillsAbsent && f.Value == "Does not have: Kubernetes" {
			found = true
		}
	}
	if !found {
		t.Fatal("the matcher must be given the absence as a fact")
	}

	added, changed, err := svc.AddSkill("Kubernetes", "")
	if err != nil || !changed {
		t.Fatalf("adding the skill after all: changed=%v err=%v", changed, err)
	}
	if len(added.Data.AbsentSkills) != 0 {
		t.Fatalf("a skill the owner now has leaves the list of absences: %+v", added.Data.AbsentSkills)
	}
	back, _, err := svc.UpdateSection("absent-skills", []byte(`[{"name":"Go"}]`))
	if err != nil {
		t.Fatalf("naming a skill the owner has among the absences: %v", err)
	}
	for _, k := range back.Data.Skills {
		if k.Name == "Go" {
			t.Fatal("naming a skill among the absences takes it out of the skills the owner has")
		}
	}
}
