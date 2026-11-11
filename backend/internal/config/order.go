package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/w3con/sifty/internal/dirs"
)

// SetModelPassOrder saves quick_match.model_passes.order in runtime.yaml. It edits the one line in
// place (or adds it as the first key of the block, creating the block under quick_match when an
// older file has none, or the whole default quick_match block when that is missing too), so comments and layout stay as they are, and
// goes through Save for validation and the timestamped backup. It does not reload the running
// configuration; the caller does.
func SetModelPassOrder(d dirs.Dirs, order string) error {
	if order != OrderFit && order != OrderNewest {
		return fmt.Errorf("order must be %q or %q", OrderFit, OrderNewest)
	}
	data, err := os.ReadFile(d.ConfigFile("runtime"))
	if err != nil {
		return fmt.Errorf("read runtime.yaml: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	indentOf := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	header := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "model_passes:" {
			header = i
			break
		}
	}
	if header < 0 {
		// An older file without the block: add it as the first child of quick_match.
		for i, l := range lines {
			if strings.TrimSpace(l) == "quick_match:" {
				in := strings.Repeat(" ", indentOf(l)+2)
				block := []string{in + "model_passes:", in + "  order: " + order}
				lines = append(lines[:i+1], append(block, lines[i+1:]...)...)
				return Save(d, "runtime", strings.Join(lines, "\n"))
			}
		}
		// No quick_match block at all: the built-in defaults apply today, so write those defaults out
		// explicitly with the chosen order; nothing else changes.
		qm := DefaultQuickMatch()
		qm.ModelPasses.Order = order
		block, err := yaml.Marshal(struct {
			QuickMatch QuickMatch `yaml:"quick_match"`
		}{qm})
		if err != nil {
			return fmt.Errorf("render quick_match: %w", err)
		}
		text := strings.TrimRight(string(data), "\n") + "\n" + string(block)
		return Save(d, "runtime", text)
	}
	childIndent := -1
	for i := header + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" || strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		if indentOf(l) <= indentOf(lines[header]) {
			break // the block ended
		}
		if childIndent < 0 {
			childIndent = indentOf(l)
		}
		if indentOf(l) == childIndent && strings.HasPrefix(strings.TrimSpace(l), "order:") {
			lines[i] = strings.Repeat(" ", childIndent) + "order: " + order
			return Save(d, "runtime", strings.Join(lines, "\n"))
		}
	}
	if childIndent < 0 {
		childIndent = indentOf(lines[header]) + 2
	}
	line := strings.Repeat(" ", childIndent) + "order: " + order
	lines = append(lines[:header+1], append([]string{line}, lines[header+1:]...)...)
	return Save(d, "runtime", strings.Join(lines, "\n"))
}
