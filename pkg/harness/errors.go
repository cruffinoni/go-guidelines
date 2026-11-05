package harness

import (
	"errors"
	"fmt"
)

// Sentinel errors identify managed-block failures without depending on
// message text.
var (
	ErrMalformedMarkers = errors.New("malformed managed instruction markers")
	ErrSymlinkTarget    = errors.New("target is a symlink")
	ErrNotRegularFile   = errors.New("target is not a regular file")
)

// FileError attributes a harness failure to one project instruction file.
// Op preserves the original user-facing verb, such as "refusing to update".
type FileError struct {
	File string
	Op   string
	Err  error
}

// Error returns a string representation of the managed file error.
func (e *FileError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.File, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.File, e.Err)
}

// Unwrap returns the underlying harness failure.
func (e *FileError) Unwrap() error { return e.Err }
