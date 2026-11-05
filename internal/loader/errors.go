package loader

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Sentinel errors identify loader input failures.
var (
	ErrOutsideModule = errors.New("directory is outside any Go module")
	ErrNotGoFile     = errors.New("path is neither a directory nor a .go file")
)

// PathError attributes a loader input failure to one path. Op names the
// failed operation, such as "stat".
type PathError struct {
	Path string
	Op   string
	Err  error
}

// Error returns a string representation of the loader path error.
func (e *PathError) Error() string {
	switch {
	case errors.Is(e.Err, ErrNotGoFile):
		return fmt.Sprintf("path %q is neither a directory nor a .go file", e.Path)
	case errors.Is(e.Err, ErrOutsideModule):
		return fmt.Sprintf("directory %q is outside any Go module", e.Path)
	case e.Op != "":
		return fmt.Sprintf("%s %q: %v", e.Op, e.Path, e.Err)
	default:
		return fmt.Sprintf("%s: %v", e.Path, e.Err)
	}
}

// Unwrap returns the underlying loader path error.
func (e *PathError) Unwrap() error { return e.Err }

// PackageError attributes a fatal package diagnostic to one package.
type PackageError struct {
	Package string
	Pos     string
	Err     error
}

// Error returns a string representation of the package error.
func (e *PackageError) Error() string { return fmt.Sprintf("package %s: %v", e.Package, e.Err) }

// Unwrap returns the underlying package error.
func (e *PackageError) Unwrap() error { return e.Err }

// classifyErrors joins every fatal error reported against pkgs. List, parse,
// and unknown errors are fatal; type errors are left where go/packages put
// them, in each package's TypeErrors field, and are not fatal. The result is
// nil when no package has a fatal error.
func classifyErrors(pkgs []*packages.Package) error {
	var fatal []error
	for _, p := range pkgs {
		for _, e := range p.Errors {
			switch e.Kind {
			case packages.TypeError:
				// Already captured in p.TypeErrors; avoid double-reporting.
			case packages.ListError:
				if len(p.TypeErrors) > 0 && strings.HasPrefix(e.Msg, "# "+p.PkgPath+"\n") {
					// go list -export echoes the compiler's output for a
					// package that failed type-checking; p.TypeErrors
					// already carries it.
					continue
				}
				fatal = append(fatal, &PackageError{Package: p.PkgPath, Pos: e.Pos, Err: e})
			default: // packages.ParseError, packages.UnknownError
				fatal = append(fatal, &PackageError{Package: p.PkgPath, Pos: e.Pos, Err: e})
			}
		}
	}
	return errors.Join(fatal...)
}
