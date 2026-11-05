package scope

import (
	"errors"
	"fmt"
)

// Sentinel errors identify scope input failures.
var (
	ErrNoPaths     = errors.New("scope: no paths given")
	ErrNotGoSource = errors.New("not a Go source file or directory")
)

// PathError attributes a scope failure to one input path. Op preserves the
// original user-facing verb, such as "inspect".
type PathError struct {
	Path string
	Op   string
	Err  error
}

// Error returns a string representation of the scope path error.
func (e *PathError) Error() string {
	if errors.Is(e.Err, ErrNotGoSource) {
		return fmt.Sprintf("%s is not a Go source file or directory", e.Path)
	}
	if e.Op != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Path, e.Err)
}

// Unwrap returns the underlying scope path error.
func (e *PathError) Unwrap() error { return e.Err }
