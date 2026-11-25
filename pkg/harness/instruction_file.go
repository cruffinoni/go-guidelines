package harness

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	startMarker = "<!-- gg-lint:start -->"
	endMarker   = "<!-- gg-lint:end -->"
)

// markerPattern matches a managed marker line: the current harness-agnostic
// form, and the per-harness form ("<!-- gg-lint:claude:start -->") written
// by gg-lint v0.0.1 and v0.0.2, which Sync migrates to the current form.
var markerPattern = regexp.MustCompile(`<!-- gg-lint:(?:([a-z0-9-]+):)?(start|end) -->`)

// instructionFile is a [Harness] that keeps one managed block, delimited by
// the startMarker and endMarker lines, inside a single instruction file. The
// block is the same for every harness, so harnesses that share a file (such
// as AGENTS.md) share the block too.
type instructionFile struct {
	name string
	// userBase returns the harness's user-level directory, which must
	// already exist; nil means the harness has no user-level instructions.
	userBase func() (string, error)
	// userFile is the slash-separated path of the user-level file under
	// userBase.
	userFile string
	// localFile is the slash-separated path of the project-level file under
	// the project directory; "" means the harness has no project-level
	// instructions.
	localFile string
	// header is written above the block when Sync creates the file, for
	// formats that need front matter.
	header string
}

// Name returns the harness name.
func (f instructionFile) Name() string { return f.name }

// location returns the directory and the slash-separated path under it of
// the instruction file for scope.
func (f instructionFile) location(scope Scope, projectDir string) (dir, file string, err error) {
	switch {
	case scope == User && f.userBase != nil:
		dir, err = f.userBase()
		if err != nil {
			return "", "", fmt.Errorf("resolve %s user directory: %w", f.name, err)
		}
		return dir, f.userFile, nil
	case scope == Local && f.localFile != "":
		return projectDir, f.localFile, nil
	default:
		return "", "", fmt.Errorf("%w: %s has no %s instructions", ErrUnsupportedScope, f.name, scope)
	}
}

// markerIsStandaloneLine reports whether source[start:end] occupies its own
// line: preceded by a newline or the start of source, and followed by a
// carriage return, a newline, or the end of source.
func markerIsStandaloneLine(source []byte, start, end int) bool {
	if start > 0 && source[start-1] != '\n' {
		return false
	}
	return end == len(source) || source[end] == '\r' || source[end] == '\n'
}

// blockSpan is the byte range of one managed block, from its start marker
// through its end marker.
type blockSpan struct {
	start int
	end   int
}

// parseBlocks returns the spans of the managed blocks in text, in document
// order. It rejects malformed markers: an unmatched one, a start and end of
// different forms, inline markers, or nesting.
func parseBlocks(original []byte, text string, matches [][]int) ([]blockSpan, error) {
	if len(matches)%2 != 0 {
		return nil, fmt.Errorf("%w: found %d markers; want complete start/end pairs", ErrMalformedMarkers, len(matches))
	}
	// Submatch 1 is the legacy harness name, submatch 2 is start or end.
	kind := func(m []int) string { return text[m[4]:m[5]] }
	legacyName := func(m []int) string {
		if m[2] < 0 {
			return ""
		}
		return text[m[2]:m[3]]
	}
	spans := make([]blockSpan, 0, len(matches)/2)
	for i := 0; i < len(matches); i += 2 {
		start, end := matches[i], matches[i+1]
		switch {
		case kind(start) != "start" || kind(end) != "end":
			return nil, fmt.Errorf("%w: %q must be followed by %q, found %q", ErrMalformedMarkers, text[start[0]:start[1]], endMarker, text[end[0]:end[1]])
		case legacyName(start) != legacyName(end):
			return nil, fmt.Errorf("%w: start %q and end %q do not match", ErrMalformedMarkers, text[start[0]:start[1]], text[end[0]:end[1]])
		case !markerIsStandaloneLine(original, start[0], start[1]) || !markerIsStandaloneLine(original, end[0], end[1]):
			return nil, fmt.Errorf("%w: each marker must occupy its own line", ErrMalformedMarkers)
		}
		spans = append(spans, blockSpan{start: start[0], end: end[1]})
	}
	return spans, nil
}

// render returns original with exactly one managed block holding
// instructions, replacing every existing block (current or legacy markers)
// with one at the first block's position, keeping the text between blocks,
// or appending a block when there is none. It rejects malformed markers; see
// parseBlocks.
func render(original []byte, instructions string) ([]byte, bool, error) {
	text := string(original)
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	block := []byte(startMarker + newline + instructions + newline + endMarker)

	matches := markerPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
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

	spans, err := parseBlocks(original, text, matches)
	if err != nil {
		return nil, false, err
	}

	updated := make([]byte, 0, len(original)+len(block))
	updated = append(updated, original[:spans[0].start]...)
	updated = append(updated, block...)
	for i := 1; i < len(spans); i++ {
		updated = append(updated, original[spans[i-1].end:spans[i].start]...)
	}
	updated = append(updated, original[spans[len(spans)-1].end:]...)
	if bytes.Equal(updated, original) {
		return original, false, nil
	}
	return updated, true, nil
}

// writeAtomic replaces file in root with data using a temporary file and a
// rename, preserving mode. Renaming replaces the directory entry rather
// than truncating the existing inode, so hard links to the old file are
// left untouched and a crash leaves either the old or the new content.
func writeAtomic(root *os.Root, file string, data []byte, mode fs.FileMode) error {
	var (
		tmp     *os.File
		tmpName string
	)
	for range 100 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return fmt.Errorf("generate temporary name: %w", err)
		}
		name := path.Join(path.Dir(file), "."+path.Base(file)+".tmp-"+hex.EncodeToString(suffix[:]))
		f, openErr := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if errors.Is(openErr, fs.ErrExist) {
			continue
		}
		if openErr != nil {
			return fmt.Errorf("create temporary file: %w", openErr)
		}
		tmp, tmpName = f, name
		break
	}
	if tmp == nil {
		return errors.New("could not create a unique temporary file")
	}
	defer func() {
		_ = tmp.Close()
		_ = root.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("preserve mode: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := root.Rename(tmpName, file); err != nil {
		return fmt.Errorf("replace target: %w", err)
	}
	// Best effort: make the rename durable where directory syncing is
	// supported. A failure here does not invalidate the atomic rename.
	if d, openErr := root.Open(path.Dir(file)); openErr == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// syncFile implements [instructionFile.Sync] for the slash-separated file
// under dir.
func syncFile(dir, file, header, instructions string) (bool, error) {
	target := filepath.Join(dir, filepath.FromSlash(file))
	root, err := os.OpenRoot(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, &FileError{File: dir, Op: "open", Err: ErrMissingDirectory}
		}
		return false, fmt.Errorf("open instruction directory: %w", err)
	}
	defer root.Close()
	if parent := path.Dir(file); parent != "." {
		if err := root.MkdirAll(parent, 0o755); err != nil {
			return false, fmt.Errorf("create %s: %w", filepath.Join(dir, filepath.FromSlash(parent)), err)
		}
	}

	mode := fs.FileMode(0o644)
	var original []byte
	info, statErr := root.Lstat(file)
	switch {
	case statErr == nil:
		if info.Mode()&fs.ModeSymlink != 0 {
			return false, &FileError{File: target, Op: "refusing to update", Err: ErrSymlinkTarget}
		}
		if !info.Mode().IsRegular() {
			return false, &FileError{File: target, Op: "refusing to update", Err: ErrNotRegularFile}
		}
		mode = info.Mode().Perm()
		if original, err = root.ReadFile(file); err != nil {
			return false, fmt.Errorf("read %s: %w", target, err)
		}
	case errors.Is(statErr, fs.ErrNotExist):
		// No existing file: the block goes below the header, if any.
		original = []byte(header)
	default:
		return false, fmt.Errorf("inspect %s: %w", target, statErr)
	}

	updated, changed, err := render(original, instructions)
	if err != nil {
		return false, &FileError{File: target, Err: err}
	}
	if !changed {
		return false, nil
	}
	if err := writeAtomic(root, file, updated, mode); err != nil {
		return false, &FileError{File: target, Op: "write", Err: err}
	}
	return true, nil
}

// resolveUserTarget returns the existing directory and the slash-separated
// file below it that a user-level write should use, following a symlinked
// directory below the base or a symlinked final file. Missing directories
// below the base stay in the file path so syncFile creates them. The base
// must already exist and is not itself resolved, so the configured path is
// the one reported.
func resolveUserTarget(dir, file string) (string, string, error) {
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", "", &FileError{File: dir, Op: "open", Err: ErrMissingDirectory}
	case err != nil:
		return "", "", &FileError{File: dir, Op: "open", Err: err}
	case !info.IsDir():
		return "", "", &FileError{File: dir, Op: "open", Err: fmt.Errorf("not a directory")}
	}
	target := filepath.Join(dir, filepath.FromSlash(file))

	// Follow a symlinked final file, as dotfile managers create.
	if linkInfo, statErr := os.Lstat(target); statErr == nil && linkInfo.Mode()&fs.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(target)
		if err != nil {
			return "", "", &FileError{File: target, Op: "resolve symlink", Err: err}
		}
		return filepath.Dir(resolved), filepath.Base(resolved), nil
	}

	// Follow symlinked directories between the base and the file. The
	// returned directory exists; missing components stay in the file path so
	// syncFile creates them below it.
	current := dir
	if parent := path.Dir(file); parent != "." {
		parts := strings.Split(parent, "/")
		for i, part := range parts {
			next := filepath.Join(current, part)
			info, err := os.Lstat(next)
			switch {
			case err == nil && info.Mode()&fs.ModeSymlink != 0:
				resolved, err := filepath.EvalSymlinks(next)
				if err != nil {
					return "", "", &FileError{File: next, Op: "resolve symlink", Err: err}
				}
				current = resolved
			case err == nil:
				current = next
			case errors.Is(err, fs.ErrNotExist):
				rel := append(parts[i:], path.Base(file))
				return current, filepath.ToSlash(filepath.Join(rel...)), nil
			default:
				return "", "", &FileError{File: next, Op: "inspect", Err: err}
			}
		}
	}
	return current, path.Base(file), nil
}

// Sync rewrites the instruction file for scope so it contains exactly one
// managed block holding instructions, replacing an existing block in place
// or appending one, and returns the file's path. Every byte outside the
// block, and any CRLF line endings, are preserved. A no-op sync performs no
// write. The scope's directory must exist; missing subdirectories below it
// are created. Writes replace the directory entry atomically and never
// escape that directory. A user-level file or directory that is a symlink,
// as dotfile managers create, is followed and its target is updated
// instead; a project-level symlink or any non-regular target is refused.
func (f instructionFile) Sync(scope Scope, projectDir, instructions string) (string, bool, error) {
	dir, file, err := f.location(scope, projectDir)
	if err != nil {
		return "", false, err
	}
	if scope == User {
		if dir, file, err = resolveUserTarget(dir, file); err != nil {
			return "", false, err
		}
	}
	target := filepath.Join(dir, filepath.FromSlash(file))
	changed, err := syncFile(dir, file, f.header, instructions)
	return target, changed, err
}
