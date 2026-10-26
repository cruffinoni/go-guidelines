package loader

import "golang.org/x/tools/go/packages"

// Package is one Go package loaded by [Load]. It embeds the native
// go/packages result (Syntax, Fset, GoFiles, Types, TypesInfo, TypesSizes,
// TypeErrors, ...) rather than re-wrapping its fields.
type Package struct {
	*packages.Package
	// Sources holds the exact bytes each file was parsed from, keyed by
	// absolute filename. It is read-only once Load returns.
	Sources map[string][]byte
}
