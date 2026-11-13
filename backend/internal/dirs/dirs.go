// Package dirs resolves the local data directory and the paths beneath it. The application is
// self-contained: it never resolves a git root, writes outside the data directory only to the one
// machine-wide model lock file (ModelLockPath), and carries no project-specific directory names. The data directory is configured by the
// --data-dir flag, then the SIFTY_DATA_DIR environment variable, then a default of ./data.
package dirs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvVar is the environment variable that configures the data directory.
const EnvVar = "SIFTY_DATA_DIR"

// Dirs is a resolved data directory and the subpaths beneath it.
type Dirs struct {
	Root string
}

// Resolve picks the data directory from, in order, the flag value, the SIFTY_DATA_DIR
// environment variable, and the default ./data. The returned path is absolute.
func Resolve(flagValue string) (Dirs, error) {
	root := flagValue
	source := "--data-dir"
	if root == "" {
		root = os.Getenv(EnvVar)
		source = EnvVar
	}
	if root == "" {
		root = "data"
		source = "default"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Dirs{}, fmt.Errorf("resolve data directory from %s: %w", source, err)
	}
	return Dirs{Root: abs}, nil
}

// Ensure creates the data directory and its standard subdirectories with private permissions.
func (d Dirs) Ensure() error {
	for _, p := range []string{d.Root, d.Config(), d.Answers(), d.Feedback(), d.Evidence()} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", p, err)
		}
	}
	return nil
}

// Config is the directory holding the YAML configuration files.
func (d Dirs) Config() string { return filepath.Join(d.Root, "config") }

// ConfigFile is the path to a named YAML config (without extension).
func (d Dirs) ConfigFile(name string) string {
	return filepath.Join(d.Config(), name+".yaml")
}

// Answers is the directory holding remembered application answers.
func (d Dirs) Answers() string { return filepath.Join(d.Root, "answers") }

// Feedback is the directory holding owner-confirmed candidate feedback.
func (d Dirs) Feedback() string { return filepath.Join(d.Root, "feedback") }

// Evidence is the directory holding candidate evidence files (CV, profile exports).
func (d Dirs) Evidence() string { return filepath.Join(d.Root, "evidence") }

// Store is the path to the local SQLite store.
func (d Dirs) Store() string { return filepath.Join(d.Root, "store.sqlite") }

// Under joins one or more elements beneath the data root and refuses any result that escapes
// the root, so a crafted relative segment cannot write outside the data directory.
func (d Dirs) Under(elem ...string) (string, error) {
	joined := filepath.Join(append([]string{d.Root}, elem...)...)
	rel, err := filepath.Rel(d.Root, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the data directory", filepath.Join(elem...))
	}
	return joined, nil
}

// ModelLockEnvVar is the environment variable that overrides the location of the machine-wide
// model lock file.
const ModelLockEnvVar = "SIFTY_MODEL_LOCK"

// ModelLockPath is the file every Sifty process on this machine locks around a model call. It is
// the one path the application keeps outside the data directory, on purpose: production, staging
// and review servers each have their own data directory but share one local model, so a lock
// inside a data directory could not stop them overlapping. The location is the SIFTY_MODEL_LOCK
// environment variable, then sifty/model.lock under the user's cache directory, then the same
// beneath the system temporary directory.
func ModelLockPath() string {
	if p := os.Getenv(ModelLockEnvVar); p != "" {
		return p
	}
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "sifty", "model.lock")
}
