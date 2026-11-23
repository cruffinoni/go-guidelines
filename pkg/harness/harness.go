// Package harness keeps coding-agent instructions current.
//
// Each coding-agent harness (Claude, Codex, ...) implements [Harness] to
// sync a block of gg-lint instructions into its own instruction file, either
// in a project directory or in the harness's user-level directory. Adding a
// harness means adding one file to this package.
package harness

import (
	"fmt"
	"os"
	"path/filepath"
)

// Harness is a coding-agent harness whose instructions gg-lint keeps
// current.
type Harness interface {
	// Name identifies the harness, as accepted by [Lookup] and by the
	// gg-lint sync-instructions command.
	Name() string
	// UserDir returns the directory holding the harness's user-level
	// instruction file, which applies to every project.
	UserDir() (string, error)
	// Sync updates the harness's instruction file in dir so it contains
	// instructions, returning changed = true if the file was written.
	Sync(dir, instructions string) (changed bool, err error)
}

// All returns every known harness, in name order.
func All() []Harness {
	return []Harness{Claude(), Codex()}
}

// Lookup returns the harness registered under name, if any.
func Lookup(name string) (Harness, bool) {
	for _, h := range All() {
		if h.Name() == name {
			return h, true
		}
	}
	return nil, false
}

// homeSubdir returns the directory named by the env variable when it holds
// an absolute path, and ~/fallback otherwise.
func homeSubdir(env, fallback string) (string, error) {
	if dir := os.Getenv(env); filepath.IsAbs(dir) {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, fallback), nil
}
