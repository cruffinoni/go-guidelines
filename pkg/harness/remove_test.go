package harness

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const managedBlock = startMarker + "\nmanaged instructions\n" + endMarker

func TestRemoveManagedBlocks(t *testing.T) {
	const legacyBlock = "<!-- gg-lint:codex:start -->\r\nold\r\n<!-- gg-lint:codex:end -->"
	const header = "---\r\nname: fixture\r\n---\r\n\r\n"
	tests := map[string]struct {
		before string
		after  string
	}{
		"preserve surrounding bytes": {
			before: "prefix\xff\n\n" + managedBlock + "\n\nsuffix\x00",
			after:  "prefix\xff\n\n\nsuffix\x00",
		},
		"preserve CRLF and front matter": {
			before: header + startMarker + "\r\nmanaged\r\n" + endMarker + "\r\n\r\nuser\rnotes\r\n",
			after:  header + "\r\nuser\rnotes\r\n",
		},
		"multiple forms, adjacent blocks, end of file": {
			before: managedBlock + "\n" + legacyBlock + "\r\nkeep this\r\n" + managedBlock,
			after:  "keep this\r\n",
		},
		"leave an existing empty file": {before: managedBlock + "\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			target := filepath.Join(dir, "NOTES.md")
			writeFile(t, target, []byte(tt.before))
			f := fixture("")
			f.header = header
			var h Harness = f

			got, changed, err := h.Remove(Local, dir)
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, target, got)
			assert.Equal(t, []byte(tt.after), readFile(t, target))

			got, changed, err = h.Remove(Local, dir)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, target, got)
			assert.Equal(t, []byte(tt.after), readFile(t, target))
		})
	}
}

func TestRemoveMissingPathsAreNoOps(t *testing.T) {
	for _, scope := range []Scope{Local, User} {
		t.Run(scope.String(), func(t *testing.T) {
			root := t.TempDir()
			base := filepath.Join(root, "base")
			parent := filepath.Join(base, "nested")
			f := fixture(base)
			f.localFile = "nested/USER.md"
			target := filepath.Join(parent, "USER.md")
			for _, missing := range []string{base, parent, target} {
				got, changed, err := f.Remove(scope, base)
				require.NoError(t, err)
				assert.False(t, changed)
				assert.Equal(t, target, got)
				_, err = os.Lstat(missing)
				require.ErrorIs(t, err, fs.ErrNotExist)
				if missing != target {
					require.NoError(t, os.Mkdir(missing, 0o755))
				}
			}
		})
	}
}

func TestRemoveDoesNotRewriteFilesWithoutBlocks(t *testing.T) {
	for name, original := range map[string]string{"empty": "", "notes": "notes\r\n\x00"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "NOTES.md")
			writeFile(t, target, []byte(original))
			before, err := os.Stat(target)
			require.NoError(t, err)
			f := fixture("")
			f.header = "---\nname: fixture\n---\n"

			got, changed, err := f.Remove(Local, dir)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, target, got)
			after, err := os.Stat(target)
			require.NoError(t, err)
			assert.True(t, os.SameFile(before, after))
			assert.Equal(t, before.ModTime(), after.ModTime())
			assert.Equal(t, []byte(original), readFile(t, target))
		})
	}
}

func TestRemoveRejectsMalformedMarkersWithoutWriting(t *testing.T) {
	for name, malformed := range map[string]string{
		"unmatched":           startMarker + "\n",
		"mixed":               startMarker + "\n<!-- gg-lint:codex:end -->\n",
		"nested":              startMarker + "\n" + managedBlock + "\n" + endMarker + "\n",
		"inline":              "prefix " + startMarker + "\n" + endMarker + " suffix\n",
		"bare CR after start": startMarker + "\rmanaged\n" + endMarker + "\n",
		"bare CR after end":   managedBlock + "\ruser notes\n",
		"bare CR at EOF":      managedBlock + "\r",
		"double CR before LF": managedBlock + "\r\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "NOTES.md")
			before := managedBlock + "\n" + malformed
			writeFile(t, target, []byte(before))

			got, changed, err := fixture("").Remove(Local, dir)
			require.ErrorIs(t, err, ErrMalformedMarkers)
			assert.False(t, changed)
			assert.Equal(t, target, got)
			fileErr, ok := errors.AsType[*FileError](err)
			require.True(t, ok)
			assert.Equal(t, target, fileErr.File)
			assert.Equal(t, before, string(readFile(t, target)))
		})
	}
}

func makeSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink in this environment: %v", err)
	}
}

func TestRemoveFollowsValidUserSymlinks(t *testing.T) {
	for _, kind := range []string{"base", "parent", "file"} {
		t.Run(kind, func(t *testing.T) {
			base := filepath.Join(t.TempDir(), "base")
			parent := filepath.Join(base, "nested")
			target := filepath.Join(parent, "USER.md")
			outside := t.TempDir()
			var link, actual, wantPath string
			switch kind {
			case "base":
				link, actual, wantPath = base, filepath.Join(outside, "nested", "USER.md"), target
				makeSymlink(t, outside, link)
				require.NoError(t, os.MkdirAll(filepath.Dir(actual), 0o755))
			case "parent":
				require.NoError(t, os.Mkdir(base, 0o755))
				link, actual = parent, filepath.Join(outside, "USER.md")
				makeSymlink(t, outside, link)
			case "file":
				require.NoError(t, os.MkdirAll(parent, 0o755))
				link, actual = target, filepath.Join(outside, "USER.md")
			}
			writeFile(t, actual, []byte(managedBlock+"\nkeep\n"))
			if kind == "file" {
				makeSymlink(t, actual, link)
			}
			if wantPath == "" {
				var err error
				wantPath, err = filepath.EvalSymlinks(actual)
				require.NoError(t, err)
			}

			got, changed, err := fixture(base).Remove(User, "ignored")
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, wantPath, got)
			assert.Equal(t, "keep\n", string(readFile(t, actual)))
			info, err := os.Lstat(link)
			require.NoError(t, err)
			assert.NotZero(t, info.Mode()&fs.ModeSymlink)
		})
	}
}

func TestRemoveRejectsDanglingUserSymlinks(t *testing.T) {
	for _, kind := range []string{"base", "base ancestor", "parent", "file"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			base := filepath.Join(root, "base")
			parent := filepath.Join(base, "nested")
			missing := filepath.Join(root, "missing")
			var link string
			switch kind {
			case "base":
				link = base
			case "base ancestor":
				link = base
				base = filepath.Join(base, "child")
			case "parent":
				require.NoError(t, os.Mkdir(base, 0o755))
				link = parent
			case "file":
				require.NoError(t, os.MkdirAll(parent, 0o755))
				link = filepath.Join(parent, "USER.md")
			}
			makeSymlink(t, missing, link)

			_, changed, err := fixture(base).Remove(User, "")
			require.ErrorIs(t, err, fs.ErrNotExist)
			assert.False(t, changed)
			fileErr, ok := errors.AsType[*FileError](err)
			require.True(t, ok)
			assert.Equal(t, link, fileErr.File)
			got, err := os.Readlink(link)
			require.NoError(t, err)
			assert.Equal(t, missing, got)
			_, err = os.Lstat(missing)
			require.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestRemoveRefusesProjectSymlinks(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		dir := t.TempDir()
		outside := filepath.Join(t.TempDir(), "notes.md")
		writeFile(t, outside, []byte(managedBlock))
		link := filepath.Join(dir, "NOTES.md")
		makeSymlink(t, outside, link)

		got, changed, err := fixture("").Remove(Local, dir)
		require.ErrorIs(t, err, ErrSymlinkTarget)
		assert.False(t, changed)
		assert.Equal(t, link, got)
		assert.Equal(t, managedBlock, string(readFile(t, outside)))
	})
	t.Run("parent escaping project", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "project")
		outside := filepath.Join(root, "outside")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.Mkdir(outside, 0o755))
		target := filepath.Join(outside, "NOTES.md")
		writeFile(t, target, []byte(managedBlock))
		makeSymlink(t, filepath.Join("..", "outside"), filepath.Join(dir, "nested"))
		f := fixture("")
		f.localFile = "nested/NOTES.md"

		_, changed, err := f.Remove(Local, dir)
		require.Error(t, err)
		assert.False(t, changed)
		assert.Equal(t, managedBlock, string(readFile(t, target)))
	})
}

func TestRemoveReportsOtherErrors(t *testing.T) {
	dir := t.TempDir()
	f := fixture("")
	f.userBase = nil
	_, changed, err := f.Remove(User, dir)
	require.ErrorIs(t, err, ErrUnsupportedScope)
	assert.False(t, changed)

	target := filepath.Join(dir, "NOTES.md")
	require.NoError(t, os.Mkdir(target, 0o755))
	_, changed, err = f.Remove(Local, dir)
	require.ErrorIs(t, err, ErrNotRegularFile)
	assert.False(t, changed)
	assert.DirExists(t, target)

	fileAsDir := filepath.Join(dir, "file")
	writeFile(t, fileAsDir, []byte(managedBlock))
	_, changed, err = f.Remove(Local, fileAsDir)
	require.Error(t, err)
	assert.False(t, changed)
	assert.Equal(t, managedBlock, string(readFile(t, fileAsDir)))
}

func TestRemovePreservesModeAndUsesAtomicReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are emulated on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "NOTES.md")
	other := filepath.Join(dir, "other.md")
	writeFile(t, target, []byte(managedBlock))
	require.NoError(t, os.Chmod(target, 0o440))
	if err := os.Link(target, other); err != nil {
		t.Skipf("cannot create hard link in this environment: %v", err)
	}

	_, changed, err := fixture("").Remove(Local, dir)
	require.NoError(t, err)
	assert.True(t, changed)
	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o440), info.Mode().Perm())
	assert.Zero(t, info.Size())
	assert.Equal(t, managedBlock, string(readFile(t, other)))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestRemoveReportsWriteFailureWithoutChangingFile(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permission bits do not restrict this process")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "NOTES.md")
	writeFile(t, target, []byte(managedBlock))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	got, changed, err := fixture("").Remove(Local, dir)
	require.ErrorIs(t, err, fs.ErrPermission)
	assert.False(t, changed)
	assert.Equal(t, target, got)
	assert.Equal(t, managedBlock, string(readFile(t, target)))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}
