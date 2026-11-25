package harness

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, content, 0o600))
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}

// isolateHome points every base resolver at a fresh, empty home directory
// and returns it.
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir uses USERPROFILE on Windows
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	for _, env := range []string{"XDG_CONFIG_HOME", "CLAUDE_CONFIG_DIR", "CODEX_HOME"} {
		t.Setenv(env, "")
	}
	return home
}

// fixture returns a harness whose local file is NOTES.md and whose user file
// is nested/USER.md under userBase.
func fixture(userBase string) instructionFile {
	return instructionFile{
		name:      "fixture",
		userBase:  func() (string, error) { return userBase, nil },
		userFile:  "nested/USER.md",
		localFile: "NOTES.md",
	}
}

// syncLocal syncs the fixture's local file in dir and returns its path.
func syncLocal(t *testing.T, dir, instructions string) (string, bool, error) {
	t.Helper()
	return fixture("").Sync(Local, dir, instructions)
}

var kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestAllNamesAreUniqueSortedAndKebabCase(t *testing.T) {
	var names []string
	for _, h := range All() {
		assert.Regexp(t, kebabCase, h.Name())
		names = append(names, h.Name())
	}
	assert.True(t, slices.IsSorted(names), "All() not in name order: %v", names)
	assert.Len(t, slices.Compact(slices.Clone(names)), len(names), "duplicate names: %v", names)
}

func TestLookup(t *testing.T) {
	for _, want := range All() {
		got, ok := Lookup(want.Name())
		require.True(t, ok, want.Name())
		assert.Equal(t, want.Name(), got.Name())
	}
	_, ok := Lookup("no-such-harness")
	assert.False(t, ok)
}

// assertSyncs checks that syncing f for scope creates want, starting with
// f's header, and that a repeat sync is a no-op.
func assertSyncs(t *testing.T, f instructionFile, scope Scope, project, want string) {
	t.Helper()
	got, changed, err := f.Sync(scope, project, "run the linter")
	require.NoError(t, err, scope)
	assert.Equal(t, want, got, scope)
	assert.True(t, changed, scope)
	content := string(readFile(t, want))
	assert.True(t, strings.HasPrefix(content, f.header), "%s: header missing from %q", scope, content)
	assert.Contains(t, content, startMarker+"\nrun the linter\n"+endMarker+"\n", scope)

	_, changed, err = f.Sync(scope, project, "run the linter")
	require.NoError(t, err, scope)
	assert.False(t, changed, "%s: repeat sync reported a change", scope)
}

func TestSyncEveryHarnessAndScope(t *testing.T) {
	for _, h := range All() {
		f, ok := h.(instructionFile)
		require.True(t, ok, "%s is not an instructionFile", h.Name())
		t.Run(f.name, func(t *testing.T) {
			isolateHome(t)
			project := t.TempDir()

			if f.userBase == nil {
				_, _, err := f.Sync(User, project, "run the linter")
				assert.ErrorIs(t, err, ErrUnsupportedScope)
			} else {
				base, err := f.userBase()
				require.NoError(t, err)
				_, _, err = f.Sync(User, project, "run the linter")
				require.ErrorIs(t, err, fs.ErrNotExist, "user sync must require an existing base")
				assert.NoDirExists(t, base)

				require.NoError(t, os.MkdirAll(base, 0o755))
				assertSyncs(t, f, User, project, filepath.Join(base, filepath.FromSlash(f.userFile)))
			}

			if f.localFile == "" {
				_, _, err := f.Sync(Local, project, "run the linter")
				assert.ErrorIs(t, err, ErrUnsupportedScope)
			} else {
				assertSyncs(t, f, Local, project, filepath.Join(project, filepath.FromSlash(f.localFile)))
			}
		})
	}
}

func TestSyncSharedFileHoldsOneBlock(t *testing.T) {
	byFile := make(map[string][]Harness)
	for _, h := range All() {
		if f := h.(instructionFile); f.localFile != "" {
			byFile[f.localFile] = append(byFile[f.localFile], h)
		}
	}
	shared := 0
	for file, harnesses := range byFile {
		if len(harnesses) < 2 {
			continue
		}
		shared++
		project := t.TempDir()
		for _, h := range harnesses {
			_, _, err := h.Sync(Local, project, "run the linter")
			require.NoError(t, err, h.Name())
		}
		content := string(readFile(t, filepath.Join(project, filepath.FromSlash(file))))
		assert.Equal(t, 1, strings.Count(content, startMarker), "%s: %q", file, content)
	}
	assert.NotZero(t, shared, "no harnesses share a local file")
}

func TestSyncReplacesTheBlockWithoutChangingSurroundingBytes(t *testing.T) {
	for name, block := range map[string]string{
		"current": startMarker + "\r\nold\r\n" + endMarker,
		"legacy":  "<!-- gg-lint:claude:start -->\r\nold\r\n<!-- gg-lint:claude:end -->",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "NOTES.md")
			prefix := []byte("user prefix\r\n")
			suffix := []byte("\r\nuser suffix\x00\n")
			writeFile(t, path, slices.Concat(prefix, []byte(block), suffix))

			_, changed, err := syncLocal(t, dir, "run the linter")
			require.NoError(t, err)
			assert.True(t, changed)

			first := readFile(t, path)
			assert.Equal(t, slices.Concat(prefix, []byte(startMarker+"\r\nrun the linter\r\n"+endMarker), suffix), first)

			before, err := os.Stat(path)
			require.NoError(t, err)
			_, changed, err = syncLocal(t, dir, "run the linter")
			require.NoError(t, err)
			assert.False(t, changed, "no-op sync reported a change")
			after, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, before.ModTime(), after.ModTime(), "no-op sync wrote the file")
			assert.Equal(t, first, readFile(t, path), "repeat sync was not byte-idempotent")
		})
	}
}

func TestSyncAppendsBelowExistingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "NOTES.md")
	writeFile(t, path, []byte("# My notes"))

	_, changed, err := syncLocal(t, dir, "run the linter")
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, "# My notes\n\n"+startMarker+"\nrun the linter\n"+endMarker+"\n", string(readFile(t, path)))
}

func TestSyncCreatesMissingParentsUnderTheUserBase(t *testing.T) {
	base := t.TempDir()
	path, changed, err := fixture(base).Sync(User, "", "run the linter")
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, filepath.Join(base, "nested", "USER.md"), path)
	assert.FileExists(t, path)
}

func TestSyncRefusesASymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "global-instructions.md")
	writeFile(t, outside, []byte("keep these bytes\n"))
	link := filepath.Join(dir, "NOTES.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create symlink in this environment: %v", err)
	}

	_, _, err := syncLocal(t, dir, "run the linter")
	require.ErrorIs(t, err, ErrSymlinkTarget)
	fileErr, ok := errors.AsType[*FileError](err)
	require.True(t, ok)
	assert.Equal(t, link, fileErr.File)
	assert.Equal(t, "keep these bytes\n", string(readFile(t, outside)), "external target changed")
}

func TestSyncFollowsAUserLevelSymlink(t *testing.T) {
	base := t.TempDir()
	dotfile := filepath.Join(t.TempDir(), "USER.md")
	writeFile(t, dotfile, []byte("# Mine\n"))
	require.NoError(t, os.Mkdir(filepath.Join(base, "nested"), 0o755))
	link := filepath.Join(base, "nested", "USER.md")
	if err := os.Symlink(dotfile, link); err != nil {
		t.Skipf("cannot create symlink in this environment: %v", err)
	}

	path, changed, err := fixture(base).Sync(User, "", "run the linter")
	require.NoError(t, err)
	assert.True(t, changed)
	wantPath, err := filepath.EvalSymlinks(dotfile)
	require.NoError(t, err)
	assert.Equal(t, wantPath, path)
	assert.Equal(t, "# Mine\n\n"+startMarker+"\nrun the linter\n"+endMarker+"\n", string(readFile(t, dotfile)))
	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&fs.ModeSymlink, "symlink was replaced by a regular file")
}

func TestSyncRejectsADanglingUserLevelSymlink(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(base, "nested"), 0o755))
	link := filepath.Join(base, "nested", "USER.md")
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing.md"), link); err != nil {
		t.Skipf("cannot create symlink in this environment: %v", err)
	}

	_, _, err := fixture(base).Sync(User, "", "run the linter")
	fileErr, ok := errors.AsType[*FileError](err)
	require.True(t, ok, "got %v", err)
	assert.Equal(t, link, fileErr.File)
}

func TestSyncReportsAWriteFailureWithoutLeavingAFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not block writes on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permission checks")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, _, err := syncLocal(t, dir, "run the linter")
	assert.Error(t, err)
	assert.NoFileExists(t, filepath.Join(dir, "NOTES.md"))
}

func TestSyncRejectsMalformedMarkersWithoutWriting(t *testing.T) {
	const legacyStart = "<!-- gg-lint:claude:start -->"
	const legacyEnd = "<!-- gg-lint:claude:end -->"
	tests := map[string]string{
		"duplicate start":        startMarker + "\n" + startMarker + "\n" + endMarker,
		"unmatched start":        startMarker + "\nuser content",
		"unmatched end":          endMarker + "\n",
		"reversed":               endMarker + "\n" + startMarker,
		"two blocks":             startMarker + "\n" + endMarker + "\n" + legacyStart + "\n" + legacyEnd,
		"current and legacy mix": startMarker + "\nold\n" + legacyEnd,
		"two legacy harnesses":   legacyStart + "\nold\n<!-- gg-lint:codex:end -->",
		"inline markers":         "prefix " + startMarker + "\nold\n" + endMarker + " suffix",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "NOTES.md")
			before := []byte("head\n" + content + "\ntail\n")
			writeFile(t, path, before)

			_, _, err := syncLocal(t, dir, "run the linter")
			require.ErrorIs(t, err, ErrMalformedMarkers)
			fileErr, ok := errors.AsType[*FileError](err)
			require.True(t, ok)
			assert.Equal(t, path, fileErr.File)
			assert.Equal(t, before, readFile(t, path), "malformed file changed")
		})
	}
}

func TestSyncDoesNotModifyHardLinkedFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "go.mod")
	link := filepath.Join(dir, "NOTES.md")
	original := []byte("module example.test/project\n")
	writeFile(t, target, original)
	if err := os.Link(target, link); err != nil {
		t.Skipf("cannot create hard link in this environment: %v", err)
	}

	_, changed, err := syncLocal(t, dir, "run the linter")
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, original, readFile(t, target), "hard-linked file was modified")
	assert.Contains(t, string(readFile(t, link)), startMarker)
}

func TestSyncRejectsNonRegularTarget(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "NOTES.md"), 0o755))

	_, _, err := syncLocal(t, dir, "run the linter")
	require.ErrorIs(t, err, ErrNotRegularFile)
}

func TestSyncPreservesModeAndLeavesNoTemporaryFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are emulated on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "NOTES.md")
	writeFile(t, path, []byte("# Notes\n"))
	require.NoError(t, os.Chmod(path, 0o600))

	_, changed, err := syncLocal(t, dir, "run the linter")
	require.NoError(t, err)
	require.True(t, changed)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "NOTES.md", entries[0].Name())
}

func TestBaseResolvers(t *testing.T) {
	home := isolateHome(t)
	override := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	configDefault := filepath.Join(home, ".config", "tool")
	if runtime.GOOS == "windows" {
		configDefault = filepath.Join(home, "AppData", "tool")
	}
	tests := map[string]struct {
		resolve  func() (string, error)
		env, val string
		want     string
	}{
		"home":                 {resolve: inHome(".tool"), want: filepath.Join(home, ".tool")},
		"env default":          {resolve: inEnvOrHome("TOOL_HOME", ".tool"), env: "TOOL_HOME", want: filepath.Join(home, ".tool")},
		"env absolute":         {resolve: inEnvOrHome("TOOL_HOME", ".tool"), env: "TOOL_HOME", val: override, want: override},
		"env relative":         {resolve: inEnvOrHome("TOOL_HOME", ".tool"), env: "TOOL_HOME", val: "rel", want: filepath.Join(cwd, "rel")},
		"xdg default":          {resolve: inXDGConfig("tool"), env: "XDG_CONFIG_HOME", want: filepath.Join(home, ".config", "tool")},
		"xdg absolute":         {resolve: inXDGConfig("tool"), env: "XDG_CONFIG_HOME", val: override, want: filepath.Join(override, "tool")},
		"xdg relative ignored": {resolve: inXDGConfig("tool"), env: "XDG_CONFIG_HOME", val: "rel", want: filepath.Join(home, ".config", "tool")},
		"config dir default":   {resolve: inConfigDir("tool"), want: configDefault},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv(tc.env, tc.val)
			}
			got, err := tc.resolve()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
