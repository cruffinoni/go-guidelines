package fixes

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyRejectsOverlappingFixes(t *testing.T) {
	source := []byte("abcdef")
	edits := []Edit{
		{RuleID: "GBP017", Start: 1, End: 4, Before: []byte("bcd"), After: []byte("x")},
		{RuleID: "GBP018", Start: 3, End: 5, Before: []byte("de"), After: []byte("y")},
	}
	if _, err := Apply(source, edits); err == nil {
		t.Fatal("Apply() accepted conflicting overlapping fixes")
	}
}

func TestApplyRejectsSourceChangedSincePlanning(t *testing.T) {
	original := []byte("package p\nvar a int\nvar b int\nvar c int\n")
	edits, err := Plan(original)
	if err != nil || len(edits) != 1 {
		t.Fatalf("Plan() = %d edits, %v; want one edit", len(edits), err)
	}
	changed := bytes.Replace(original, []byte("var b int"), []byte("var b bool"), 1)
	if _, err := Apply(changed, edits); err == nil {
		t.Fatal("Apply() accepted stale edit after source changed")
	}
}

func TestPlanLeavesCommentedDeclarationsUntouched(t *testing.T) {
	source := []byte("package p\n// documented\nvar a int\nvar b int\nvar c int\n")
	edits, err := Plan(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 0 {
		t.Fatalf("Plan() = %d edits, want no edit when declaration group has comments", len(edits))
	}
}

func TestWriteAtomicRefusesToReplaceSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.go")
	link := filepath.Join(root, "link.go")
	original := []byte("package target\n")
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if err := WriteAtomic(link, original, []byte("package changed\n")); err == nil {
		t.Fatal("WriteAtomic() replaced a symlink instead of refusing it")
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("path is no longer a symlink")
	}
}

func TestWriteAtomicRejectsStaleSourceWithoutOverwriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.go")
	current := []byte("newer contents")
	if err := os.WriteFile(path, current, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, []byte("stale contents"), []byte("replacement")); err == nil {
		t.Fatal("WriteAtomic() overwrote a file changed after planning")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, current) {
		t.Fatalf("file = %q, want unchanged %q", after, current)
	}
}

func TestWriteAtomicPreservesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.go")
	original := []byte("old")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, original, []byte("new")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
}
