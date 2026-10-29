// Package linttest checks a lint.Rule against annotated fixture files, in the
// style of golang.org/x/tools/go/analysis/analysistest.
//
// A fixture line expecting findings carries a comment of the form
//
//	// want "regexp" `regexp`
//
// Each finding reported on that line must match one expectation, and each
// expectation must be matched exactly once.
package linttest

import (
	"cmp"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

type position struct {
	file string
	line int
}

func (p position) compare(q position) int {
	return cmp.Or(strings.Compare(p.file, q.file), cmp.Compare(p.line, q.line))
}

type expectation struct {
	re      *regexp.Regexp
	matched bool
}

func consume(wants []*expectation, message string) bool {
	for _, w := range wants {
		if !w.matched && w.re.MatchString(message) {
			w.matched = true
			return true
		}
	}
	return false
}

// parseDir parses dir's .go files in name order and groups them by package
// clause, mirroring how the engine hands one package at a time to a rule.
func parseDir(fset *token.FileSet, dir string) ([][]*ast.File, map[string][]byte, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, nil, err
	}
	if len(names) == 0 {
		return nil, nil, fmt.Errorf("no .go files in %s", dir)
	}
	sources := make(map[string][]byte, len(names))
	byPackage := make(map[string]int)
	var pkgFiles [][]*ast.File
	for _, name := range names {
		src, err := os.ReadFile(name)
		if err != nil {
			return nil, nil, err
		}
		file, err := parser.ParseFile(fset, name, src, parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, nil, err
		}
		sources[name] = src
		i, ok := byPackage[file.Name.Name]
		if !ok {
			i = len(pkgFiles)
			byPackage[file.Name.Name] = i
			pkgFiles = append(pkgFiles, nil)
		}
		pkgFiles[i] = append(pkgFiles[i], file)
	}
	return pkgFiles, sources, nil
}

func parsePatterns(text string) ([]*regexp.Regexp, error) {
	var res []*regexp.Regexp
	for text = strings.TrimSpace(text); text != ""; text = strings.TrimSpace(text) {
		quoted, err := strconv.QuotedPrefix(text)
		if err != nil {
			return nil, fmt.Errorf("want: expected quoted regexp at %q", text)
		}
		text = text[len(quoted):]
		pattern, err := strconv.Unquote(quoted)
		if err != nil {
			return nil, fmt.Errorf("want: %w", err)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("want: %w", err)
		}
		res = append(res, re)
	}
	if len(res) == 0 {
		return nil, errors.New("want: no patterns")
	}
	return res, nil
}

func collectWants(fset *token.FileSet, pkgFiles [][]*ast.File) (map[position][]*expectation, error) {
	wants := make(map[position][]*expectation)
	for _, files := range pkgFiles {
		for _, file := range files {
			for _, group := range file.Comments {
				for _, c := range group.List {
					text, ok := strings.CutPrefix(c.Text, "// want ")
					if !ok {
						continue
					}
					p := fset.Position(c.Pos())
					res, err := parsePatterns(text)
					if err != nil {
						return nil, fmt.Errorf("%s:%d: %w", p.Filename, p.Line, err)
					}
					key := position{p.Filename, p.Line}
					for _, re := range res {
						wants[key] = append(wants[key], &expectation{re: re})
					}
				}
			}
		}
	}
	return wants, nil
}

// typeCheck fills the Pass type fields; errors are recorded, not fatal, as in
// the engine.
func typeCheck(fset *token.FileSet, files []*ast.File, pass *lint.Pass) {
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	conf := types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			if e, ok := errors.AsType[types.Error](err); ok {
				pass.TypeErrors = append(pass.TypeErrors, e)
			}
		},
	}
	pass.Pkg, _ = conf.Check(files[0].Name.Name, fset, files, info)
	pass.TypesInfo = info
	pass.TypesSizes = types.SizesFor("gc", runtime.GOARCH)
}

func check(rule lint.Rule, dir string) ([]string, error) {
	fset := token.NewFileSet()
	pkgFiles, sources, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}
	wants, err := collectWants(fset, pkgFiles)
	if err != nil {
		return nil, err
	}

	meta := rule.Meta()
	var problems []string
	for _, files := range pkgFiles {
		var diagnostics []lint.Diagnostic
		pass := lint.NewPass(fset, files, sources, func(d lint.Diagnostic) { diagnostics = append(diagnostics, d) })
		if meta.NeedsTypes {
			typeCheck(fset, files, pass)
		}
		if err := rule.Check(pass); err != nil {
			return nil, fmt.Errorf("rule %s: %w", meta.ID, err)
		}
		for _, d := range diagnostics {
			p := fset.Position(d.Pos)
			if !consume(wants[position{p.Filename, p.Line}], d.Message) {
				problems = append(problems, fmt.Sprintf("%s:%d:%d: unexpected finding %q", p.Filename, p.Line, p.Column, d.Message))
			}
		}
	}
	for _, pos := range slices.SortedFunc(maps.Keys(wants), position.compare) {
		for _, w := range wants[pos] {
			if !w.matched {
				problems = append(problems, fmt.Sprintf("%s:%d: no finding matching %q", pos.file, pos.line, w.re))
			}
		}
	}
	return problems, nil
}

// Run checks rule against every .go file in dir and reports each mismatch
// between findings and want comments as a test error.
func Run(t testing.TB, rule lint.Rule, dir string) {
	t.Helper()
	problems, err := check(rule, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}
