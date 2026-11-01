package gbp

import (
	"go/ast"
	"go/token"
	"slices"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// GroupVarDeclarations flags three or more consecutive single-variable var
// declarations.
type GroupVarDeclarations struct{}

// Meta returns the GBP017 identity, severity, and documentation.
func (GroupVarDeclarations) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP017",
		Name:    "group-var-declarations",
		Summary: "Flags three or more consecutive single-variable var declarations.",
		Description: "Reports a run of three or more consecutive single-line `var` declarations, " +
			"excluding parenthesized `var (...)` blocks. Scattering related declarations one per line is " +
			"harder to scan than one grouped block; consolidate them into a single `var (...)` block.",
		Severity: lint.SeverityWarning,
	}
}

// varDecl is one single-line, single-variable var declaration.
type varDecl struct {
	line int
	pos  token.Pos
}

// appendVarDecl appends g when it is a single-line, single-variable,
// non-parenthesized var declaration.
func appendVarDecl(tf *token.File, g *ast.GenDecl, out *[]varDecl) {
	if g.Tok != token.VAR || g.Lparen.IsValid() || len(g.Specs) != 1 {
		return
	}
	spec, ok := g.Specs[0].(*ast.ValueSpec)
	if !ok || len(spec.Names) != 1 {
		return
	}
	start, end := tf.Position(g.Pos()), tf.Position(g.End())
	if start.Line != end.Line {
		return
	}
	*out = append(*out, varDecl{line: start.Line, pos: token.Pos(tf.LineStart(start.Line))})
}

// collectDecls returns the single-line, single-variable var declarations in
// node.
func collectDecls(tf *token.File, node ast.Node) []varDecl {
	var decls []varDecl
	ast.Inspect(node, func(n ast.Node) bool {
		g, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		appendVarDecl(tf, g, &decls)
		return true
	})
	return decls
}

// collectDirectDecls returns the package-level var declarations in decls.
func collectDirectDecls(tf *token.File, decls []ast.Decl) []varDecl {
	var out []varDecl
	for _, d := range decls {
		if g, ok := d.(*ast.GenDecl); ok {
			appendVarDecl(tf, g, &out)
		}
	}
	return out
}

// reportRuns reports runs of three or more adjacent declarations in decls.
func reportRuns(pass *lint.Pass, decls []varDecl) {
	slices.SortFunc(decls, func(a, b varDecl) int { return a.line - b.line })
	start := 0
	for i := range decls {
		if i == 0 || decls[i].line != decls[i-1].line+1 {
			start = i
		}
		if i+1 < len(decls) && decls[i+1].line == decls[i].line+1 {
			continue
		}
		if count := i - start + 1; count >= 3 {
			pass.Reportf(decls[start].pos, "%d consecutive var declarations can be grouped", count)
		}
	}
}

// Check reports runs of ungrouped var declarations in pass.
func (GroupVarDeclarations) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		tf := pass.Fset.File(file.Pos())
		reportRuns(pass, collectDirectDecls(tf, file.Decls))
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Body != nil {
				reportRuns(pass, collectDecls(tf, f.Body))
			}
		}
	}
	return nil
}
