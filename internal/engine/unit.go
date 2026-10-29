package engine

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/loader"
)

// target is a loaded package prepared once for every rule that checks it.
type target struct {
	id      string
	base    *lint.Pass
	inScope map[string]struct{} // this package's GoFiles that are in scope
}

// newTarget builds a target for pkg, keeping only the GoFiles present in
// inScope so that findings outside the requested scope are never produced.
func newTarget(pkg *loader.Package, inScope map[string]struct{}) target {
	base := lint.NewPass(pkg.Fset, pkg.Syntax, pkg.Sources, nil)
	if pkg.Types != nil {
		base.Pkg = pkg.Types
		base.TypesInfo = pkg.TypesInfo
		base.TypesSizes = pkg.TypesSizes
		base.TypeErrors = pkg.TypeErrors
	}
	kept := make(map[string]struct{}, len(pkg.GoFiles))
	for _, f := range pkg.GoFiles {
		if _, ok := inScope[f]; ok {
			kept[f] = struct{}{}
		}
	}
	return target{id: pkg.ID, base: base, inScope: kept}
}

// check runs rule.Check, turning a panic into an error carrying its stack.
func check(rule lint.Rule, pass *lint.Pass) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	return rule.Check(pass)
}

// run checks t against one rule, isolating panics and filtering
// diagnostics down to in-scope findings.
func (t target) run(rule lint.Rule) ([]lint.Finding, error) {
	meta := rule.Meta()

	pass := *t.base
	if !meta.NeedsTypes {
		pass.Pkg = nil
		pass.TypesInfo = nil
		pass.TypesSizes = nil
		pass.TypeErrors = nil
	}
	var diags []lint.Diagnostic
	pass.Report = func(d lint.Diagnostic) { diags = append(diags, d) }

	var errs []error
	if err := check(rule, &pass); err != nil {
		errs = append(errs, fmt.Errorf("rule %s on package %s: %w", meta.ID, t.id, err))
	}

	fset := t.base.Fset
	findings := make([]lint.Finding, 0, len(diags))
	for _, d := range diags {
		if !d.Pos.IsValid() || fset.File(d.Pos) == nil {
			errs = append(errs, fmt.Errorf("rule %s on package %s: diagnostic has invalid position", meta.ID, t.id))
			continue
		}
		filename := fset.Position(d.Pos).Filename
		if _, ok := t.inScope[filename]; !ok {
			continue
		}
		findings = append(findings, lint.NewFinding(meta, fset, d))
	}
	return findings, errors.Join(errs...)
}
