// Package loader loads Go packages with go/packages: resolving input paths,
// grouping them by their enclosing module, parsing (and optionally
// type-checking) each group, and normalizing test variants and load errors
// into one deterministic package list.
package loader

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"golang.org/x/tools/go/packages"
)

// Config controls how Load discovers, parses, and optionally type-checks
// packages.
type Config struct {
	// Tests includes _test.go files and test-only package variants.
	Tests bool
	// NeedsTypes requests full type-checking; otherwise only syntax is loaded.
	NeedsTypes bool
	// BuildFlags is passed through to the underlying go list invocation,
	// e.g. []string{"-tags", "integration"}.
	BuildFlags []string
	// Env is the environment for the go list invocation. nil uses os.Environ().
	Env []string
}

// loadGroup runs one packages.Load call for g and classifies the result's
// errors: list, parse, and unknown errors are fatal; type errors stay in
// each package's native TypeErrors field.
func loadGroup(ctx context.Context, cfg Config, g dirGroup) ([]*Package, error) {
	patterns := g.patterns()
	store := newSourceStore()

	mode := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
		packages.NeedSyntax | packages.NeedModule
	if cfg.NeedsTypes {
		mode |= packages.NeedTypes | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedImports
	}

	pcfg := &packages.Config{
		Context:    ctx,
		Mode:       mode,
		Dir:        g.dir,
		Env:        cfg.Env,
		BuildFlags: cfg.BuildFlags,
		Tests:      cfg.Tests,
		ParseFile:  store.parseFile,
	}

	pkgs, err := packages.Load(pcfg, patterns...)
	if err != nil {
		// go/packages flattens a cancellation into text; surface ctx.Err so
		// callers can match it with errors.Is.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("load %v in %s: %w", patterns, g.dir, ctxErr)
		}
		return nil, fmt.Errorf("load %v in %s: %w", patterns, g.dir, err)
	}
	if err := classifyErrors(pkgs); err != nil {
		return nil, err
	}

	out := make([]*Package, 0, len(pkgs))
	for _, p := range pkgs {
		out = append(out, &Package{Package: p, Sources: store.m})
	}
	return out, nil
}

// Load parses, and if cfg.NeedsTypes is set type-checks, the packages named
// by paths. A directory means that directory recursively, following go's
// ./... semantics (vendor, testdata, and dot/underscore-prefixed directories
// are skipped); a .go file means that single file. Directories must be
// inside a Go module; standalone .go files may live outside one.
//
// The result is sorted deterministically by package ID, then by GoFiles.
func Load(ctx context.Context, cfg Config, paths ...string) ([]*Package, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	groups, err := groupByDir(paths)
	if err != nil {
		return nil, err
	}

	var all []*Package
	for _, g := range groups {
		pkgs, err := loadGroup(ctx, cfg, g)
		if err != nil {
			return nil, err
		}
		// Groups never share files, but IDs can repeat across them (every
		// outside-module load is "command-line-arguments"), so dedupe per group.
		all = append(all, dedupeTestVariants(pkgs)...)
	}

	slices.SortFunc(all, func(a, b *Package) int {
		return cmp.Or(cmp.Compare(a.ID, b.ID), slices.Compare(a.GoFiles, b.GoFiles))
	})

	return all, nil
}
