package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

const accessACL = "system.posix_acl_access"

func readAccessACL(fd int) ([]byte, error) {
	size, err := unix.Fgetxattr(fd, accessACL, nil)
	if errors.Is(err, unix.ENODATA) || errors.Is(err, unix.ENOTSUP) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	acl := make([]byte, size)
	n, err := unix.Fgetxattr(fd, accessACL, acl)
	if err != nil {
		return nil, err
	}
	return acl[:n], nil
}

func preserveFileAccess(source, tmp *os.File, mode fs.FileMode) error {
	srcFD, dstFD := int(source.Fd()), int(tmp.Fd())
	acl, err := readAccessACL(srcFD)
	if err != nil {
		return fmt.Errorf("read POSIX ACL: %w", err)
	}

	// ACL owner and group entries apply to the inode's owner and group.
	var src, dst unix.Stat_t
	if err := unix.Fstat(srcFD, &src); err != nil {
		return fmt.Errorf("stat original owner: %w", err)
	}
	if err := unix.Fstat(dstFD, &dst); err != nil {
		return fmt.Errorf("stat replacement owner: %w", err)
	}
	if src.Uid != dst.Uid || src.Gid != dst.Gid {
		if err := tmp.Chown(int(src.Uid), int(src.Gid)); err != nil {
			return fmt.Errorf("preserve owner and group: %w", err)
		}
	}

	if acl != nil {
		// On Linux, installing a POSIX ACL also sets the corresponding mode bits.
		if err := unix.Fsetxattr(dstFD, accessACL, acl, 0); err != nil {
			return fmt.Errorf("copy POSIX ACL: %w", err)
		}
		return nil
	}

	// A default directory ACL may have been inherited at creation. Clear
	// it before chmod so any inherited entries cannot gain access.
	err = unix.Fremovexattr(dstFD, accessACL)
	if err != nil && !errors.Is(err, unix.ENODATA) && !errors.Is(err, unix.ENOTSUP) {
		return fmt.Errorf("clear inherited POSIX ACL: %w", err)
	}
	return tmp.Chmod(mode)
}
