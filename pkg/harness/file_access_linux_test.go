package harness

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// testACL encodes Linux's POSIX ACL xattr format. The group mode bits are
// the mask (read), even though the owning group entry grants nothing.
func testACL() []byte {
	const noID = ^uint32(0)
	acl := binary.LittleEndian.AppendUint32(nil, 2) // POSIX ACL xattr version
	for _, e := range []struct {
		tag, perm uint16
		id        uint32
	}{
		{0x01, 6, noID}, // ACL_USER_OBJ
		{0x04, 0, noID}, // ACL_GROUP_OBJ
		{0x10, 4, noID}, // ACL_MASK
		{0x20, 0, noID}, // ACL_OTHER
	} {
		acl = binary.LittleEndian.AppendUint16(acl, e.tag)
		acl = binary.LittleEndian.AppendUint16(acl, e.perm)
		acl = binary.LittleEndian.AppendUint32(acl, e.id)
	}
	return acl
}

func setTestACL(t *testing.T, path, name string, acl []byte) {
	t.Helper()
	err := unix.Setxattr(path, name, acl, 0)
	if errors.Is(err, unix.ENOTSUP) {
		t.Skipf("filesystem does not support %s: %v", name, err)
	}
	require.NoError(t, err)
}

func assertAccessACL(t *testing.T, path string, want []byte) {
	t.Helper()
	acl := make([]byte, 128)
	n, err := unix.Getxattr(path, accessACL, acl)
	if want == nil {
		require.ErrorIs(t, err, unix.ENODATA)
		return
	}
	require.NoError(t, err)
	assert.Equal(t, want, acl[:n])
}

func TestRemovePreservesLinuxPOSIXACL(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "NOTES.md")
	writeFile(t, target, []byte(managedBlock+"\nkeep\n"))
	acl := testACL()
	setTestACL(t, target, accessACL, acl)
	before, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), before.Mode().Perm())

	_, changed, err := fixture("").Remove(Local, dir)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, "keep\n", string(readFile(t, target)))
	assertAccessACL(t, target, acl)
	after, err := os.Stat(target)
	require.NoError(t, err)
	assert.Equal(t, before.Mode().Perm(), after.Mode().Perm())
	assert.False(t, os.SameFile(before, after), "expected an atomic replacement")
}

func TestRemoveDoesNotInheritNewLinuxPOSIXACL(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "NOTES.md")
	writeFile(t, target, []byte(managedBlock))
	require.NoError(t, os.Chmod(target, 0o640))
	setTestACL(t, dir, "system.posix_acl_default", testACL())
	assertAccessACL(t, target, nil)

	_, changed, err := fixture("").Remove(Local, dir)
	require.NoError(t, err)
	assert.True(t, changed)
	assertAccessACL(t, target, nil)
	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}
