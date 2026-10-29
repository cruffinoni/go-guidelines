package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ast/inspector"
)

// Pass gives a rule read-only access to one parsed package. The engine builds
// one Pass per (rule, package); the syntax trees, type information, and source
// bytes it points to are shared between concurrently running rules, so rules
// must never modify them.
type Pass struct {
	Fset *token.FileSet
	// Files holds the package syntax, parsed with comments and without
	// legacy object resolution.
	Files []*ast.File
	// Pkg, TypesInfo, and TypesSizes are nil unless the rule sets NeedsTypes.
	Pkg        *types.Package
	TypesInfo  *types.Info
	TypesSizes types.Sizes
	// TypeErrors lists type-checking errors; type information may be partial
	// when it is non-empty.
	TypeErrors []types.Error
	// Inspector traverses Files and is built once per package.
	Inspector *inspector.Inspector
	// Report records a diagnostic for the running rule.
	Report func(Diagnostic)

	sources map[string][]byte
}

// NewPass returns a Pass over files whose source bytes are keyed by filename
// in sources. Type fields are left for the caller to populate.
func NewPass(fset *token.FileSet, files []*ast.File, sources map[string][]byte, report func(Diagnostic)) *Pass {
	return &Pass{
		Fset:      fset,
		Files:     files,
		Inspector: inspector.New(files),
		Report:    report,
		sources:   sources,
	}
}

// Reportf reports a diagnostic at pos with a formatted message.
func (p *Pass) Reportf(pos token.Pos, format string, args ...any) {
	p.Report(Diagnostic{Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// Filename returns the absolute path of f as recorded in the file set.
func (p *Pass) Filename(f *ast.File) string {
	return p.Fset.File(f.FileStart).Name()
}

// Source returns the exact bytes f was parsed from. Callers must not modify them.
func (p *Pass) Source(f *ast.File) []byte {
	return p.sources[p.Filename(f)]
}

// IsTestFile reports whether f is a _test.go file.
func (p *Pass) IsTestFile(f *ast.File) bool {
	return strings.HasSuffix(p.Filename(f), "_test.go")
}
