// Package scope decides which files a scan covers, from explicit path
// inputs narrowed by exclude selectors.
package scope

import (
	"os"
	"path/filepath"
	"slices"
)

// Set is the set of files a scan covers: the Go files under its path
// inputs, minus those under its exclude selectors.
type Set struct {
	paths    []string            // canonical absolute inputs, input order
	files    map[string]struct{} // canonical file inputs
	dirs     []string            // canonical directory inputs
	excludes []string            // canonical exclude selectors
}

// New validates every path as an existing directory or an existing .go
// file, and builds a Set scoped to them. exclude narrows which files
// Contains accepts; its selectors may name files or directories and need
// not exist. paths must not be empty; callers wanting every file default it to
// ["."].
func New(paths, exclude []string) (*Set, error) {
	if len(paths) == 0 {
		return nil, ErrNoPaths
	}

	s := &Set{files: make(map[string]struct{}, len(paths))}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, &PathError{Path: p, Op: "inspect", Err: err}
		}
		if !info.IsDir() && filepath.Ext(p) != ".go" {
			return nil, &PathError{Path: p, Err: ErrNotGoSource}
		}
		abs, err := canonicalAbs(p)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			s.dirs = append(s.dirs, abs)
		} else {
			s.files[abs] = struct{}{}
		}
		s.paths = append(s.paths, abs)
	}

	for _, sel := range exclude {
		abs, err := canonicalAbsAllowMissing(sel)
		if err != nil {
			return nil, err
		}
		s.excludes = append(s.excludes, abs)
	}
	return s, nil
}

// NewEmpty returns an empty Set, used when a narrowed scan has no files.
func NewEmpty() *Set {
	return &Set{files: make(map[string]struct{})}
}

// Empty reports whether the Set has no path inputs.
func (s *Set) Empty() bool {
	return len(s.paths) == 0
}

// HasDirectoryInput reports whether the Set was built from at least one
// directory path. Directory scans include test files by default.
func (s *Set) HasDirectoryInput() bool {
	return len(s.dirs) > 0
}

// Paths returns the Set's absolute path inputs, in input order, for the
// loader to read.
func (s *Set) Paths() []string {
	return slices.Clone(s.paths)
}

// DirectoryInputs returns the Set's canonical directory inputs.
func (s *Set) DirectoryInputs() []string {
	return slices.Clone(s.dirs)
}

// IsFileInput reports whether filename names an explicit file input rather
// than a file merely covered by a directory input.
func (s *Set) IsFileInput(filename string) bool {
	f, err := canonicalAbs(filename)
	if err != nil {
		return false
	}
	_, ok := s.files[f]
	return ok
}

// matchesInputs reports whether f equals a file input or lies under a
// directory input.
func (s *Set) matchesInputs(f string) bool {
	if _, ok := s.files[f]; ok {
		return true
	}
	for _, dir := range s.dirs {
		if under(dir, f) {
			return true
		}
	}
	return false
}

// Contains reports whether filename is in scope: it must equal a file
// input or lie under a directory input, and lie under no exclude
// selector.
func (s *Set) Contains(filename string) bool {
	f, err := canonicalAbs(filename)
	if err != nil {
		return false
	}
	if !s.matchesInputs(f) {
		return false
	}
	return !matchesAny(s.excludes, f)
}
