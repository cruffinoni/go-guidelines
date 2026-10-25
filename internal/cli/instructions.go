package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	claudeStartMarker = "<!-- gg-lint:claude:start -->"
	claudeEndMarker   = "<!-- gg-lint:claude:end -->"
	codexStartMarker  = "<!-- gg-lint:codex:start -->"
	codexEndMarker    = "<!-- gg-lint:codex:end -->"
)

type instructionMarkers struct {
	filename string
	start    string
	end      string
	content  string
}

func instructionTarget(provider string) (instructionMarkers, error) {
	switch provider {
	case "claude":
		return instructionMarkers{
			filename: "CLAUDE.md",
			start:    claudeStartMarker,
			end:      claudeEndMarker,
			content:  "Run `gg-lint scan --changed-files` to check changed Go files before completing Go code changes.",
		}, nil
	case "codex":
		return instructionMarkers{
			filename: "AGENTS.md",
			start:    codexStartMarker,
			end:      codexEndMarker,
			content:  "Run `gg-lint scan --changed-files` to check changed Go files before completing Go code changes.",
		}, nil
	default:
		return instructionMarkers{}, fmt.Errorf("unknown instruction provider %q (want claude or codex)", provider)
	}
}

// syncInstructions updates only the selected provider's repository-root file.
// Marker lines are intentionally unique and form one bounded, replaceable block.
func syncInstructions(root, provider string, writeFile func(string, []byte, os.FileMode) error) ([]string, error) {
	target, err := instructionTarget(provider)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, target.filename)
	mode := fs.FileMode(0o644)
	original := []byte(nil)
	info, err := os.Lstat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("inspect project instruction file %s: %w", path, err)
	}
	if err == nil {
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing to update project instruction file %s: target is a symlink", path)
		}
		mode = info.Mode().Perm()
		original, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read project instruction file %s: %w", path, err)
		}
	}

	updated, changed, err := renderInstructionFile(original, target)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", target.filename, err)
	}
	if !changed {
		return nil, nil
	}
	if writeFile == nil {
		writeFile = os.WriteFile
	}
	if err := writeFile(path, updated, mode); err != nil {
		return nil, fmt.Errorf("write project instruction file %s: %w", path, err)
	}
	return []string{path}, nil
}

func markerIsStandaloneLine(source []byte, marker string, offset int) bool {
	if offset > 0 && source[offset-1] != '\n' {
		return false
	}
	after := offset + len(marker)
	return after == len(source) || source[after] == '\r' || source[after] == '\n'
}

func renderInstructionFile(original []byte, target instructionMarkers) ([]byte, bool, error) {
	all := []string{claudeStartMarker, claudeEndMarker, codexStartMarker, codexEndMarker}
	counts := make(map[string]int, len(all))
	positions := make(map[string]int, len(all))
	for _, marker := range all {
		counts[marker] = strings.Count(string(original), marker)
		positions[marker] = strings.Index(string(original), marker)
	}
	selectedStartCount, selectedEndCount := counts[target.start], counts[target.end]
	if selectedStartCount != selectedEndCount || selectedStartCount > 1 {
		return nil, false, fmt.Errorf("malformed managed instruction markers (expected either no markers or one %q and one %q); remove or repair the marker lines manually", target.start, target.end)
	}
	if selectedStartCount == 1 && (!markerIsStandaloneLine(original, target.start, positions[target.start]) || !markerIsStandaloneLine(original, target.end, positions[target.end])) {
		return nil, false, fmt.Errorf("malformed managed instruction markers: each marker must occupy its own line; move or repair the marker lines manually")
	}
	for _, marker := range all {
		if marker != target.start && marker != target.end && counts[marker] != 0 {
			return nil, false, fmt.Errorf("malformed managed instruction markers (found marker for another provider %q); remove or repair the marker lines manually", marker)
		}
	}

	newline := "\n"
	if strings.Contains(string(original), "\r\n") {
		newline = "\r\n"
	}
	block := []byte(target.start + newline + target.content + newline + target.end)
	if counts[target.start] == 0 {
		updated := append([]byte(nil), original...)
		if len(updated) > 0 && updated[len(updated)-1] != '\n' {
			updated = append(updated, newline...)
		}
		if len(updated) > 0 {
			updated = append(updated, newline...)
		}
		updated = append(updated, block...)
		updated = append(updated, newline...)
		return updated, true, nil
	}

	start := positions[target.start]
	end := positions[target.end]
	if start > end {
		return nil, false, fmt.Errorf("malformed managed instruction markers: end marker appears before start marker; reorder or remove the marker lines manually")
	}
	end += len(target.end)
	updated := make([]byte, 0, len(original)-(end-start)+len(block))
	updated = append(updated, original[:start]...)
	updated = append(updated, block...)
	updated = append(updated, original[end:]...)
	if string(updated) == string(original) {
		return original, false, nil
	}
	return updated, true, nil
}
