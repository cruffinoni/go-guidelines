// Package engine runs catalog rules against loaded Go packages, sharing
// each package's base [lint.Pass] read-only across the rules that check
// it, and fanning the diagnostics back in deterministically.
package engine

import (
	"context"
	"fmt"
	"go/ast"
	"runtime"
	"slices"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/loader"
	"github.com/cruffinoni/go-guidelines/internal/scope"
)

// Options controls which catalog rules an Engine runs and how many units
// run concurrently.
type Options struct {
	// Workers caps how many (package, rule) units run concurrently.
	// <=0 means runtime.GOMAXPROCS(0).
	Workers int
	// Rules selects which catalog rules are active. nil selects every rule.
	Rules func(lint.Meta) bool
}

// Result is the deterministic outcome of a Check.
type Result struct {
	// Files lists every in-scope analyzed file, absolute and sorted.
	// Generated files are loaded for type checking but never listed or
	// reported on.
	Files []string
	// Findings is sorted with [lint.CompareFindings] and de-duplicated.
	Findings []lint.Finding
	// TypeErrors collects non-fatal type-check errors from loaded packages.
	TypeErrors []error
}

// Engine checks a [scope.Set] against a fixed set of rules selected from a
// catalog.
type Engine struct {
	rules   []lint.Rule
	workers int
}

// New builds an Engine over the rules of cat selected by opt.Rules.
func New(cat *catalog.Catalog, opt Options) *Engine {
	var selected []lint.Rule
	for _, r := range cat.List() {
		if opt.Rules == nil || opt.Rules(r.Meta()) {
			selected = append(selected, r)
		}
	}
	workers := opt.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	return &Engine{rules: selected, workers: workers}
}

// Rules returns the rules this Engine was built with.
func (e *Engine) Rules() []lint.Rule {
	return slices.Clone(e.rules)
}

// generatedFiles returns the names of pkg's files marked as generated, as
// defined by [ast.IsGenerated].
func generatedFiles(pkg *loader.Package) map[string]struct{} {
	generated := make(map[string]struct{})
	for _, f := range pkg.Syntax {
		if ast.IsGenerated(f) {
			generated[pkg.Fset.File(f.FileStart).Name()] = struct{}{}
		}
	}
	return generated
}

// Check loads the packages under s and runs the Engine's rules against
// them. The returned error joins every operational failure; Result is
// populated even when it is non-nil, and Files is populated even when no
// rules are selected.
func (e *Engine) Check(ctx context.Context, s *scope.Set) (Result, error) {
	needsTypes := slices.ContainsFunc(e.rules, func(r lint.Rule) bool { return r.Meta().NeedsTypes })

	pkgs, err := loader.Load(ctx, loader.Config{Tests: true, NeedsTypes: needsTypes}, s.Paths()...)
	if err != nil {
		return Result{}, fmt.Errorf("load packages: %w", err)
	}

	var typeErrs []error
	inScope := make(map[string]struct{})
	for _, pkg := range pkgs {
		for _, te := range pkg.TypeErrors {
			typeErrs = append(typeErrs, fmt.Errorf("%s: %w", pkg.ID, te))
		}
		generated := generatedFiles(pkg)
		for _, f := range pkg.GoFiles {
			if _, ok := generated[f]; ok {
				continue
			}
			if s.Contains(f) {
				inScope[f] = struct{}{}
			}
		}
	}

	files := make([]string, 0, len(inScope))
	for f := range inScope {
		files = append(files, f)
	}
	slices.Sort(files)

	findings, runErr := fanOut(ctx, pkgs, e.rules, inScope, e.workers)

	return Result{Files: files, Findings: findings, TypeErrors: typeErrs}, runErr
}
