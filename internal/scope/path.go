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
