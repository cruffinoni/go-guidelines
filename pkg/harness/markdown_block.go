package harness

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

// anyMarkerPattern matches a managed marker line for any harness, so that a
// marker belonging to another harness can be detected without keeping a
// global list of every harness's markers.
var anyMarkerPattern = regexp.MustCompile(`<!-- gg-lint:([a-z0-9-]+):(start|end) -->`)

// markdownBlock is a [Harness] that keeps one managed block, delimited by
// "<!-- gg-lint:<name>:start -->" and "<!-- gg-lint:<name>:end -->" marker
// lines, inside a single Markdown file at the project root.
type markdownBlock struct {
	name string
	file string
}

// Name returns the harness name embedded in the block markers.
func (m markdownBlock) Name() string { return m.name }

func (m markdownBlock) startMarker() string { return fmt.Sprintf("<!-- gg-lint:%s:start -->", m.name) }
func (m markdownBlock) endMarker() string   { return fmt.Sprintf("<!-- gg-lint:%s:end -->", m.name) }

// markerIsStandaloneLine reports whether the marker at offset in source
// occupies its own line: preceded by a newline or the start of source, and
// followed by a carriage return, a newline, or the end of source.
func markerIsStandaloneLine(source []byte, marker string, offset int) bool {
	if offset > 0 && source[offset-1] != '\n' {
		return false
	}
	after := offset + len(marker)
	return after == len(source) || source[after] == '\r' || source[after] == '\n'
}

// render returns original with exactly one managed block holding
// instructions, replacing an existing block or appending one. It rejects
// malformed markers: duplicated, unmatched, reversed, nested, inline, or
// belonging to another harness.
func (m markdownBlock) render(original []byte, instructions string) ([]byte, bool, error) {
	start, end := m.startMarker(), m.endMarker()
	text := string(original)
	startCount, endCount := strings.Count(text, start), strings.Count(text, end)
	if startCount != endCount || startCount > 1 {
		return nil, false, fmt.Errorf("malformed managed instruction markers: want no markers or one %q and one %q", start, end)
	}

	var startPos, endPos int
	if startCount == 1 {
		startPos, endPos = strings.Index(text, start), strings.Index(text, end)
		if !markerIsStandaloneLine(original, start, startPos) || !markerIsStandaloneLine(original, end, endPos) {
			return nil, false, errors.New("malformed managed instruction markers: each marker must occupy its own line")
		}
	}
	for _, match := range anyMarkerPattern.FindAllString(text, -1) {
		if match != start && match != end {
			return nil, false, fmt.Errorf("malformed managed instruction markers: found marker for another harness %q", match)
		}
	}

	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	block := []byte(start + newline + instructions + newline + end)

	if startCount == 0 {
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

	if startPos > endPos {
		return nil, false, errors.New("malformed managed instruction markers: end marker appears before start marker")
	}
	endPos += len(end)
	updated := make([]byte, 0, len(original)-(endPos-startPos)+len(block))
	updated = append(updated, original[:startPos]...)
	updated = append(updated, block...)
	updated = append(updated, original[endPos:]...)
	if bytes.Equal(updated, original) {
		return original, false, nil
	}
	return updated, true, nil
}

// writeAtomic replaces m.file with data using a temporary file and a
// rename, preserving mode. Renaming replaces the directory entry rather
// than truncating the existing inode, so hard links to the old file are
// left untouched and a crash leaves either the old or the new content.
func (m markdownBlock) writeAtomic(rootDir *os.Root, data []byte, mode fs.FileMode) (err error) {
	var (
		tmp     *os.File
		tmpName string
	)
	for range 100 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return fmt.Errorf("generate temporary name: %w", err)
		}
		name := "." + m.file + ".tmp-" + hex.EncodeToString(suffix[:])
		f, openErr := rootDir.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
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
		_ = rootDir.Remove(tmpName)
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
	if err := rootDir.Rename(tmpName, m.file); err != nil {
		return fmt.Errorf("replace target: %w", err)
	}
	// Best effort: make the rename durable where directory syncing is
	// supported. A failure here does not invalidate the atomic rename.
	if dir, openErr := rootDir.Open("."); openErr == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

// Sync rewrites root/m.file so it contains exactly one managed block
// holding instructions, replacing an existing block in place or appending
// one. Every byte outside the block, and any CRLF line endings, are
// preserved. A no-op sync performs no write. Writes replace the directory
// entry atomically and never escape root; a symlink or other non-regular
// target is refused.
func (m markdownBlock) Sync(root, instructions string) (changed bool, err error) {
	rootDir, err := os.OpenRoot(root)
	if err != nil {
		return false, fmt.Errorf("open project root %s: %w", root, err)
	}
	defer rootDir.Close()

	mode := fs.FileMode(0o644)
	var original []byte
	info, statErr := rootDir.Lstat(m.file)
	switch {
	case statErr == nil:
		if info.Mode()&fs.ModeSymlink != 0 {
			return false, fmt.Errorf("refusing to update %s: target is a symlink", m.file)
		}
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("refusing to update %s: target is not a regular file", m.file)
		}
		mode = info.Mode().Perm()
		if original, err = rootDir.ReadFile(m.file); err != nil {
			return false, fmt.Errorf("read %s: %w", m.file, err)
		}
	case errors.Is(statErr, fs.ErrNotExist):
		// No existing file: original stays nil, and the block is appended
		// to an otherwise empty file.
	default:
		return false, fmt.Errorf("inspect %s: %w", m.file, statErr)
	}

	updated, changed, err := m.render(original, instructions)
	if err != nil {
		return false, fmt.Errorf("%s: %w", m.file, err)
	}
	if !changed {
		return false, nil
	}
	if err := m.writeAtomic(rootDir, updated, mode); err != nil {
		return false, fmt.Errorf("write %s: %w", m.file, err)
	}
	return true, nil
}
