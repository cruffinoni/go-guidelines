package scope

import (
	"path/filepath"
	"strings"
)

// canonicalize resolves symlinks in an already-absolute path, falling back
// to the cleaned path if that fails (e.g. the path does not exist).
func canonicalize(abs string) string {
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return filepath.Clean(abs)
}

// canonicalizeAllowMissing is like canonicalize, but it resolves the deepest
// existing ancestor and appends the missing remainder. This lets an exclude
// selector that does not exist yet still compare equal to a real file when
// the project is reached through a symlink.
func canonicalizeAllowMissing(abs string) string {
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	parent := filepath.Dir(abs)
	if parent == abs {
		return filepath.Clean(abs)
	}
	return filepath.Join(canonicalizeAllowMissing(parent), filepath.Base(abs))
}

// canonicalAbs resolves path to an absolute, symlink-free form, so that
// inputs, selectors, and the filenames Contains checks all compare equal
// regardless of how each was spelled (e.g. macOS /var vs /private/var).
func canonicalAbs(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return canonicalize(abs), nil
}

// canonicalAbsAllowMissing is [canonicalAbs] for paths that may not exist.
func canonicalAbsAllowMissing(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return canonicalizeAllowMissing(abs), nil
}

// under reports whether target names base itself or a descendant of it,
// comparing whole path segments so that "/a/bc" is not under "/a/b".
func under(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}

// matchesAny reports whether target lies under any of selectors.
func matchesAny(selectors []string, target string) bool {
	for _, sel := range selectors {
		if under(sel, target) {
			return true
		}
	}
	return false
}
