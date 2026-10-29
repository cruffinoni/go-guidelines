package engine_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
	"github.com/cruffinoni/go-guidelines/internal/engine"
	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/scope"
)

// fakeRule adapts a Check function to lint.Rule for tests.
type fakeRule struct {
	meta  lint.Meta
	check func(*lint.Pass) error
}

func (r fakeRule) Meta() lint.Meta          { return r.meta }
func (r fakeRule) Check(p *lint.Pass) error { return r.check(p) }

func newFakeRule(id string, needsTypes bool, check func(*lint.Pass) error) fakeRule {
	return fakeRule{
		meta: lint.Meta{
			ID:          id,
			Name:        strings.ToLower(id),
			Summary:     id,
			Description: id,
			Severity:    lint.SeverityWarning,
			NeedsTypes:  needsTypes,
		},
		check: check,
	}
}

// fileCountRule reports one diagnostic per file in the package.
func fileCountRule(id string) fakeRule {
	return newFakeRule(id, false, func(p *lint.Pass) error {
		for _, f := range p.Files {
			p.Reportf(f.Pos(), "%s saw %s", id, p.Filename(f))
		}
		return nil
	})
}

func panicRule(id string) fakeRule {
	return newFakeRule(id, false, func(p *lint.Pass) error {
		panic("boom: " + id)
	})
}

func erroringRule(id string, wrapped error) fakeRule {
	return newFakeRule(id, false, func(p *lint.Pass) error {
		return wrapped
	})
}

func invalidPosRule(id string) fakeRule {
	return newFakeRule(id, false, func(p *lint.Pass) error {
		p.Report(lint.Diagnostic{Pos: token.NoPos, Message: "invalid"})
		return nil
	})
}

// typesPresenceRule fails via a Check error unless type information is
// present exactly as expectPresent requires.
func typesPresenceRule(id string, needsTypes, expectPresent bool) fakeRule {
	return newFakeRule(id, needsTypes, func(p *lint.Pass) error {
		present := p.Pkg != nil && p.TypesInfo != nil
		if present != expectPresent {
			return fmt.Errorf("type information present=%v, want %v", present, expectPresent)
		}
		return nil
	})
}

// concurrentReadRule exercises every shared read-only field of Pass from
// several goroutines, for use under -race.
func concurrentReadRule(id string) fakeRule {
	return newFakeRule(id, true, func(p *lint.Pass) error {
		var wg sync.WaitGroup
		wg.Go(func() {
			for _, f := range p.Files {
				_ = p.Filename(f)
				_ = p.Source(f)
				_ = p.IsTestFile(f)
			}
		})
		wg.Go(func() {
			p.Inspector.Preorder(nil, func(ast.Node) {})
		})
		wg.Go(func() {
			if p.TypesInfo != nil {
				for range p.TypesInfo.Types {
				}
			}
		})
		wg.Wait()
		p.Reportf(p.Files[0].Pos(), "%s ok", id)
		return nil
	})
}

func newCatalog(t *testing.T, rules ...lint.Rule) *catalog.Catalog {
	t.Helper()
	c, err := catalog.New(rules...)
	require.NoError(t, err)
	return c
}

func newScope(t *testing.T, paths, exclude []string) *scope.Set {
	t.Helper()
	s, err := scope.New(paths, exclude)
	require.NoError(t, err)
	return s
}

// abs returns path in the canonical form the engine reports: absolute and
// symlink-free, so tests pass when the repo lives under a symlink.
func abs(t *testing.T, path string) string {
	t.Helper()
	a, err := filepath.Abs(path)
	require.NoError(t, err)
	resolved, err := filepath.EvalSymlinks(a)
	require.NoError(t, err)
	return resolved
}

func TestNew_SelectsRulesByCondition(t *testing.T) {
	cat := newCatalog(t, fileCountRule("R1"), fileCountRule("R2"))

	e := engine.New(cat, engine.Options{Rules: func(m lint.Meta) bool { return m.ID == "R1" }})
	ids := make([]string, 0)
	for _, r := range e.Rules() {
		ids = append(ids, r.Meta().ID)
	}
	assert.Equal(t, []string{"R1"}, ids)
}

func TestNew_NilRulesSelectsEverything(t *testing.T) {
	cat := newCatalog(t, fileCountRule("R1"), fileCountRule("R2"))
	e := engine.New(cat, engine.Options{})
	assert.Len(t, e.Rules(), 2)
}

func TestCheck_NoRulesSelectedStillReturnsFiles(t *testing.T) {
	cat := newCatalog(t, fileCountRule("R1"))
	e := engine.New(cat, engine.Options{Rules: func(lint.Meta) bool { return false }})

	s := newScope(t, []string{"testdata/mod"}, nil)
	res, err := e.Check(t.Context(), s)
	require.NoError(t, err)
	assert.Empty(t, res.Findings)
	assert.NotEmpty(t, res.Files)
}

func TestCheck_Determinism(t *testing.T) {
	cat := newCatalog(t, fileCountRule("R1"), fileCountRule("R2"))
	s := newScope(t, []string{"testdata/mod"}, nil)

	var baseline engine.Result
	for i := 0; i < 20; i++ {
		workers := 1
		if i%2 == 0 {
			workers = 8
		}
		e := engine.New(cat, engine.Options{Workers: workers})
		res, err := e.Check(t.Context(), s)
		require.NoError(t, err)
		if i == 0 {
			baseline = res
			continue
		}
		assert.Equal(t, baseline.Findings, res.Findings, "iteration %d", i)
		assert.Equal(t, baseline.Files, res.Files, "iteration %d", i)
	}
}

func TestCheck_PanicInOneRuleDoesNotStopOthers(t *testing.T) {
	cat := newCatalog(t, panicRule("PANIC"), fileCountRule("OK"))
	e := engine.New(cat, engine.Options{})
	s := newScope(t, []string{"testdata/mod"}, nil)

	res, err := e.Check(t.Context(), s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic:")
	assert.Contains(t, err.Error(), "PANIC")

	found := false
	for _, f := range res.Findings {
		if f.RuleID == "OK" {
			found = true
		}
	}
	assert.True(t, found, "expected rule OK to still report, findings=%v", res.Findings)
}

func TestCheck_ErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	cat := newCatalog(t, erroringRule("ERR", sentinel), fileCountRule("OK"))
	e := engine.New(cat, engine.Options{})
	s := newScope(t, []string{"testdata/mod"}, nil)

	res, err := e.Check(t.Context(), s)
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "ERR")

	found := false
	for _, f := range res.Findings {
		if f.RuleID == "OK" {
			found = true
		}
	}
	assert.True(t, found, "expected rule OK to still report")
}

func TestCheck_Cancellation(t *testing.T) {
	cat := newCatalog(t, fileCountRule("R1"))
	e := engine.New(cat, engine.Options{Workers: 1})
	s := newScope(t, []string{"testdata/mod"}, nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := e.Check(ctx, s)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestCheck_InvalidPositionIsAnError(t *testing.T) {
	cat := newCatalog(t, invalidPosRule("BAD"), fileCountRule("OK"))
	e := engine.New(cat, engine.Options{})
	s := newScope(t, []string{"testdata/mod"}, nil)

	res, err := e.Check(t.Context(), s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid position")
	assert.Contains(t, err.Error(), "BAD")

	for _, f := range res.Findings {
		assert.NotEqual(t, "BAD", f.RuleID, "expected no finding from the invalid-position rule")
	}
}

func TestCheck_NeedsTypesDrivesTypeLoading(t *testing.T) {
	cat := newCatalog(t, typesPresenceRule("TYPED", true, true))
	e := engine.New(cat, engine.Options{})
	s := newScope(t, []string{"testdata/mod"}, nil)

	_, err := e.Check(t.Context(), s)
	require.NoError(t, err)
}

func TestCheck_NoRuleNeedsTypesSkipsTypeLoading(t *testing.T) {
	cat := newCatalog(t, typesPresenceRule("UNTYPED", false, false))
	e := engine.New(cat, engine.Options{})
	s := newScope(t, []string{"testdata/mod"}, nil)

	_, err := e.Check(t.Context(), s)
	require.NoError(t, err)
}

func TestCheck_ScopeFilteringExcludesSiblingFile(t *testing.T) {
	cat := newCatalog(t, fileCountRule("R1"))
	e := engine.New(cat, engine.Options{})
	s := newScope(t, []string{"testdata/mod/a.go"}, nil)

	res, err := e.Check(t.Context(), s)
	require.NoError(t, err)

	want := abs(t, "testdata/mod/a.go")
	assert.Equal(t, []string{want}, res.Files)
	for _, f := range res.Findings {
		assert.Equal(t, want, f.File)
	}
}

func TestCheck_ExcludeDropsMatchingFiles(t *testing.T) {
	cat := newCatalog(t, fileCountRule("R1"))
	e := engine.New(cat, engine.Options{})
	s := newScope(t, []string{"testdata/mod"}, []string{"testdata/mod/sub"})

	res, err := e.Check(t.Context(), s)
	require.NoError(t, err)

	excluded := abs(t, "testdata/mod/sub/c.go")
	assert.NotContains(t, res.Files, excluded)
}

func TestCheck_ConcurrencyStress(t *testing.T) {
	var rules []lint.Rule
	for i := 0; i < 6; i++ {
		rules = append(rules, concurrentReadRule(fmt.Sprintf("C%d", i)))
	}
	cat := newCatalog(t, rules...)
	e := engine.New(cat, engine.Options{Workers: 8})
	s := newScope(t, []string{"testdata/mod"}, nil)

	res, err := e.Check(t.Context(), s)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Findings)
}
