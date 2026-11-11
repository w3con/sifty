package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/dirs"
)

func TestBuiltInSkillsAreValid(t *testing.T) {
	s, err := DefaultSkills()
	if err != nil {
		t.Fatalf("DefaultSkills: %v", err)
	}
	if len(s.Groups) < 40 || len(s.TechnologyContext) == 0 || len(s.NotAnswerable) == 0 {
		t.Fatalf("the built-in vocabulary is empty: %d groups", len(s.Groups))
	}
}

func TestSkillsFileInTheDataDirectoryReplacesTheDefaultAndAnOldDirectoryKeepsWorking(t *testing.T) {
	root := t.TempDir()
	d := dirs.Dirs{Root: root}
	if err := os.MkdirAll(d.Config(), 0o700); err != nil {
		t.Fatal(err)
	}
	// No skills.yaml: an installation that predates the file gets the built-in vocabulary.
	s, err := loadSkills(d)
	if err != nil || len(s.Groups) < 40 {
		t.Fatalf("without a file: %v, %d groups", err, len(s.Groups))
	}
	custom := "version: 1\ngroups:\n  - name: Zig\n    terms: [Ziglang]\n"
	if err := os.WriteFile(filepath.Join(d.Config(), "skills.yaml"), []byte(custom), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadSkills(d)
	if err != nil || len(got.Groups) != 1 || got.Groups[0].Name != "Zig" {
		t.Fatalf("with a file: %v, %+v", err, got)
	}
	if got.Identity() == s.Identity() {
		t.Fatal("a different vocabulary must have a different identity")
	}
}

func TestSkillsRefuseATermThatBelongsToTwoGroups(t *testing.T) {
	s := Skills{Version: 1, Groups: []SkillGroup{
		{Name: "Go", Terms: []string{"Golang"}},
		{Name: "Other", Terms: []string{"golang"}},
	}}
	if err := s.validate("skills.yaml"); err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("one term in two groups would make the match ambiguous: %v", err)
	}
	if err := (Skills{Version: 2}).validate("skills.yaml"); err == nil {
		t.Fatal("an unknown version must be refused")
	}
}
