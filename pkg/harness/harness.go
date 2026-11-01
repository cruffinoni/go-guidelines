// Package harness keeps coding-agent project instructions current.
//
// Each coding-agent harness (Claude, Codex, ...) implements [Harness] to
// sync a block of gg-lint instructions into its own project file. Adding a
// harness means adding one file to this package.
package harness

// Harness is a coding-agent harness whose project instructions gg-lint
// keeps current.
type Harness interface {
	// Name identifies the harness, as accepted by [Lookup] and by the
	// gg-lint --harness flag.
	Name() string
	// Sync updates the harness's project instruction file under root so it
	// contains instructions, returning changed = true if the file was
	// written.
	Sync(root, instructions string) (changed bool, err error)
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
