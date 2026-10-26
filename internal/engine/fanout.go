package engine

import (
	"context"
	"errors"
	"slices"

	"golang.org/x/sync/errgroup"

	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/loader"
)

// unit is one (package, rule) pair to check.
type unit struct {
	pkgIdx  int
	ruleIdx int
}

// fanOut checks every package in pkgs against every rule, bounded to
// workers concurrent units, and fans the results back in deterministically.
// The returned error joins every operational failure, including context
// cancellation; a unit's failure never stops the others.
func fanOut(ctx context.Context, pkgs []*loader.Package, rules []lint.Rule, inScope map[string]struct{}, workers int) ([]lint.Finding, error) {
	if len(pkgs) == 0 || len(rules) == 0 {
		return nil, nil
	}

	targets := make([]target, len(pkgs))
	for i, pkg := range pkgs {
		targets[i] = newTarget(pkg, inScope)
	}

	units := make([]unit, 0, len(pkgs)*len(rules))
	for pi := range pkgs {
		for ri := range rules {
			units = append(units, unit{pkgIdx: pi, ruleIdx: ri})
		}
	}

	results := make([][]lint.Finding, len(units))
	unitErrs := make([]error, len(units))

	var g errgroup.Group
	g.SetLimit(min(workers, len(units)))
	for i := range units {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			u := units[i]
			results[i], unitErrs[i] = runUnit(targets[u.pkgIdx], rules[u.ruleIdx])
			return nil
		})
	}
	// g.Wait always returns nil here: every unit records its own error in
	// unitErrs instead of returning it to the group, so that one unit's
	// failure never cancels the others.
	_ = g.Wait()

	var all []lint.Finding
	for _, f := range results {
		all = append(all, f...)
	}

	errs := append(unitErrs, ctx.Err())
	slices.SortFunc(all, lint.CompareFindings)
	all = slices.Compact(all)

	return all, errors.Join(errs...)
}
