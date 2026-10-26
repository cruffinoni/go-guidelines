package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// target is one input path resolved to an absolute location.
type target struct {
	abs   string
	isDir bool
}

// dirGroup is every target sharing one packages.Load invocation: either a
// module root, or (for standalone files outside any module) the file's own
// directory.
type dirGroup struct {
	dir     string
	targets []target
}

// groupByDir resolves paths and groups them by the module root enclosing
// them (or, for .go files outside any module, by their own directory), so
// that Load issues one packages.Load call per group.
func groupByDir(paths []string) ([]dirGroup, error) {
	byDir := make(map[string]*dirGroup)
	var order []string

	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("resolve path %q: %w", p, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", p, err)
		}
		isDir := info.IsDir()
		if !isDir && !strings.HasSuffix(abs, ".go") {
			return nil, fmt.Errorf("path %q is neither a directory nor a .go file", p)
		}

		searchDir := abs
		if !isDir {
			searchDir = filepath.Dir(abs)
		}
		dir, ok := findModuleRoot(searchDir)
		if !ok {
			if isDir {
				return nil, fmt.Errorf("directory %q is outside any Go module", p)
			}
			dir = searchDir
		}

		g, exists := byDir[dir]
		if !exists {
			g = &dirGroup{dir: dir}
			byDir[dir] = g
			order = append(order, dir)
		}
		g.targets = append(g.targets, target{abs: abs, isDir: isDir})
	}

	groups := make([]dirGroup, 0, len(order))
	for _, d := range order {
		groups = append(groups, *byDir[d])
	}
	return groups, nil
}

// findModuleRoot walks up from dir looking for a go.mod file.
func findModuleRoot(dir string) (string, bool) {
	for {
		if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// buildPatterns turns g's targets into go/packages patterns relative to
// g.dir, deduplicated and in a stable order.
func buildPatterns(g dirGroup) []string {
	seen := make(map[string]struct{}, len(g.targets))
	var out []string
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, t := range g.targets {
		if !t.isDir {
			add("file=" + t.abs)
			continue
		}
		rel, err := filepath.Rel(g.dir, t.abs)
		if err != nil || rel == "." {
			add("./...")
			continue
		}
		add("./" + filepath.ToSlash(rel) + "/...")
	}
	return out
}
