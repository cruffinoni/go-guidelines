// Package harness keeps coding-agent instructions current.
//
// Each coding-agent harness (Claude, Codex, Gemini, ...) implements
// [Harness] to sync or remove managed gg-lint instructions in its own
// instruction file, either in a project directory ([Local]) or at user
// level ([User]). Adding a harness means adding one file to this package
// and listing its constructor in [All].
package harness

import "errors"

// ErrUnsupportedScope is returned when a harness has no instruction file
// for the requested [Scope].
var ErrUnsupportedScope = errors.New("unsupported scope")

// Scope selects which of a harness's instruction files to update.
type Scope int

const (
	// User selects the user-level file, which applies to every project.
	User Scope = iota
	// Local selects the file in a project directory.
	Local
)

// String returns "user" or "local".
func (s Scope) String() string {
	if s == Local {
		return "local"
	}
	return "user"
}

// Harness is a coding-agent harness whose instructions gg-lint keeps
// current.
type Harness interface {
	// Name identifies the harness, as accepted by [Lookup].
	Name() string
	// Sync updates the harness's instruction file for scope so it contains
	// instructions, and returns the file's path and whether it was written.
	// projectDir is the project directory for [Local] and is ignored for
	// [User]. It wraps [ErrUnsupportedScope] when the harness has no file
	// for scope.
	Sync(scope Scope, projectDir, instructions string) (path string, changed bool, err error)
	// Remove removes all managed blocks from the instruction file for scope
	// and returns the file's path and whether it was written. A missing
	// directory or file, or a file without managed blocks, is a no-op.
	// projectDir is used only for [Local]. Like Sync, Remove wraps
	// [ErrUnsupportedScope] when the harness has no file for scope.
	Remove(scope Scope, projectDir string) (path string, changed bool, err error)
}

// All returns every known harness, in name order.
func All() []Harness {
	return []Harness{
		newAmp(), newBob(), newClaude(), newCline(), newCodex(), newContinue(),
		newCopilot(), newDevin(), newDsh(), newFactory(), newGemini(), newGoose(),
		newOpenClaw(), newOpenCode(), newOpenHands(), newPi(), newQwen(), newRoo(),
		newVibe(),
	}
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
