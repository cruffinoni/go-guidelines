package loader

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"
)

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
				fatal = append(fatal, fmt.Errorf("package %s: %w", p.PkgPath, e))
			default: // packages.ParseError, packages.UnknownError
				fatal = append(fatal, fmt.Errorf("package %s: %w", p.PkgPath, e))
			}
		}
	}
	return errors.Join(fatal...)
}
