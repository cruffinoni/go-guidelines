//go:build !linux

package harness

import (
	"io/fs"
	"os"
)

// Other platforms retain the mode-only behavior; their ACLs are not exposed
// through Linux's POSIX ACL extended attribute.
func preserveFileAccess(_ *os.File, tmp *os.File, mode fs.FileMode) error {
	return tmp.Chmod(mode)
}
