// Package git gives read-only access to a Git worktree: its root directory
// and the files changed within it.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// ErrNotRepository is returned when a directory is not inside a Git
// worktree.
var ErrNotRepository = errors.New("not a git repository")

// Root returns the top-level directory of the Git worktree containing dir.
// It wraps ErrNotRepository, with the underlying Git error as cause, when
// dir is not inside a worktree.
func Root(ctx context.Context, dir string) (string, error) {
	out, err := gitOutput(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotRepository, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ChangedFiles returns the absolute paths of files changed in the Git
// worktree rooted at root: tracked changes against HEAD (or against the
// index, when HEAD does not exist yet, as in a freshly initialized
// repository), plus untracked files. Only existing regular files are
// returned, sorted and deduplicated. Any extension is included; callers
// that care about Go files filter the result themselves.
func ChangedFiles(ctx context.Context, root string) ([]string, error) {
	trackedArgs := []string{"diff", "--name-only", "--diff-filter=ACMRTUXB", "-z", "HEAD"}
	if _, err := gitOutput(ctx, root, "rev-parse", "--verify", "HEAD"); err != nil {
		trackedArgs = []string{"diff", "--cached", "--name-only", "--diff-filter=ACMRTUXB", "-z"}
	}
	tracked, err := gitOutput(ctx, root, trackedArgs...)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	untracked, err := gitOutput(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	set := make(map[string]struct{})
	for _, raw := range slices.Concat(bytes.Split(tracked, []byte{0}), bytes.Split(untracked, []byte{0})) {
		name := filepath.ToSlash(string(raw))
		if name == "" {
			continue
		}
		absolute := filepath.Join(root, filepath.FromSlash(name))
		info, statErr := os.Stat(absolute)
		switch {
		case statErr == nil && info.Mode().IsRegular():
			set[absolute] = struct{}{}
		case statErr != nil && !errors.Is(statErr, os.ErrNotExist):
			return nil, fmt.Errorf("inspect changed file %s: %w", absolute, statErr)
		}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	slices.Sort(result)
	return result, nil
}

// gitOutput runs a Git subcommand with dir as its working directory and
// returns its standard output.
func gitOutput(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return cmd.Output()
}
