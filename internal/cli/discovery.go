package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func discover(inputs []string) ([]string, map[string]string, error) {
	files := make([]string, 0)
	display := make(map[string]string)
	seen := make(map[string]bool)
	for _, input := range inputs {
		info, err := os.Stat(input)
		if err != nil {
			return nil, nil, fmt.Errorf("inspect %s: %w", input, err)
		}
		if !info.IsDir() {
			if filepath.Ext(input) != ".go" {
				return nil, nil, fmt.Errorf("%s is not a Go source file or directory", input)
			}
			absolute, err := filepath.Abs(input)
			if err != nil {
				return nil, nil, err
			}
			if !seen[absolute] {
				seen[absolute] = true
				files = append(files, absolute)
				display[absolute] = filepath.Base(input)
			}
			continue
		}
		root, err := filepath.Abs(input)
		if err != nil {
			return nil, nil, err
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && path != root && (entry.Name() == "vendor" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
				rel, e := filepath.Rel(root, path)
				if e != nil {
					return e
				}
				display[path] = filepath.ToSlash(rel)
			}
			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("walk %s: %w", input, err)
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, nil, errors.New("no Go source files found")
	}
	return files, display, nil
}

func gitChangedFiles() ([]string, string, error) {
	rootOutput, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, "", errors.New("--changed-files requires a Git worktree")
	}
	root := strings.TrimSpace(string(rootOutput))
	trackedArgs := []string{"diff", "--name-only", "--diff-filter=ACMRTUXB", "-z", "HEAD"}
	if _, err := exec.Command("git", "rev-parse", "--verify", "HEAD").Output(); err != nil {
		trackedArgs = []string{"diff", "--cached", "--name-only", "--diff-filter=ACMRTUXB", "-z"}
	}
	tracked, err := exec.Command("git", trackedArgs...).Output()
	if err != nil {
		return nil, "", fmt.Errorf("git diff failed: %w", err)
	}
	untracked, err := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return nil, "", fmt.Errorf("git list untracked files failed: %w", err)
	}
	set := make(map[string]struct{})
	for _, raw := range append(bytes.Split(tracked, []byte{0}), bytes.Split(untracked, []byte{0})...) {
		name := filepath.ToSlash(string(raw))
		if name == "" || filepath.Ext(name) != ".go" {
			continue
		}
		absolute := filepath.Join(root, filepath.FromSlash(name))
		info, statErr := os.Stat(absolute)
		if statErr == nil && !info.IsDir() {
			set[name] = struct{}{}
		} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return nil, "", fmt.Errorf("inspect changed file %s: %w", name, statErr)
		}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, root, nil
}

func selectChangedPaths(changed []string, root string, targets, includes, excludes []string) ([]string, error) {
	rootPath := root
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Git worktree root: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}
	match := func(file, selector string) bool {
		absolute := selector
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(cwd, selector)
		}
		rel, relErr := filepath.Rel(root, absolute)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "." {
			return true
		}
		return file == rel || strings.HasPrefix(file, strings.TrimSuffix(rel, "/")+"/")
	}
	selected := make([]string, 0, len(changed))
	for _, file := range changed {
		inTarget := false
		for _, target := range targets {
			if match(file, target) {
				inTarget = true
				break
			}
		}
		if !inTarget {
			continue
		}
		if len(includes) > 0 {
			included := false
			for _, selector := range includes {
				if match(file, selector) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}
		excluded := false
		for _, selector := range excludes {
			if match(file, selector) {
				excluded = true
				break
			}
		}
		if !excluded {
			selected = append(selected, filepath.Join(rootPath, filepath.FromSlash(file)))
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no changed Go source files found")
	}
	return selected, nil
}
