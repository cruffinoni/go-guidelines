package catalog

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

type fakeRule struct {
	meta lint.Meta
}

func (r fakeRule) Meta() lint.Meta { return r.meta }

func (fakeRule) Check(*lint.Pass) error { return nil }

func rule(id string, mutate ...func(*lint.Meta)) fakeRule {
	m := lint.Meta{
		ID:          id,
		Name:        "name-" + id,
		Summary:     "summary " + id,
		Description: "description " + id,
		Severity:    lint.SeverityWarning,
	}
	for _, f := range mutate {
		f(&m)
	}
	return fakeRule{meta: m}
}

func ids(rules []lint.Rule) []string {
	var out []string
	for _, r := range rules {
		out = append(out, r.Meta().ID)
	}
	return out
}

func TestNewRejectsInvalidRules(t *testing.T) {
	tests := map[string][]lint.Rule{
		"empty ID":          {rule("R1", func(m *lint.Meta) { m.ID = "" })},
		"empty Name":        {rule("R1", func(m *lint.Meta) { m.Name = "" })},
		"empty Summary":     {rule("R1", func(m *lint.Meta) { m.Summary = "" })},
		"empty Description": {rule("R1", func(m *lint.Meta) { m.Description = "" })},
		"zero Severity":     {rule("R1", func(m *lint.Meta) { m.Severity = 0 })},
		"duplicate ID":      {rule("R1"), rule("R1", func(m *lint.Meta) { m.Name = "other" })},
		"duplicate Name":    {rule("R1"), rule("R2", func(m *lint.Meta) { m.Name = "name-R1" })},
	}
	for name, rules := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := New(rules...)
			assert.Error(t, err)
		})
	}
}

func TestNewOrdersRulesByID(t *testing.T) {
	c, err := New(rule("R3"), rule("R1"), rule("R2"))
	require.NoError(t, err)
	assert.Equal(t, []string{"R1", "R2", "R3"}, ids(c.List()))
}

func TestGet(t *testing.T) {
	c, err := New(rule("R1"))
	require.NoError(t, err)

	got, ok := c.Get("R1")
	assert.True(t, ok)
	assert.Equal(t, "R1", got.Meta().ID)

	_, ok = c.Get("R404")
	assert.False(t, ok)
}

func TestListReturnsACopy(t *testing.T) {
	c, err := New(rule("R1"), rule("R2"))
	require.NoError(t, err)

	got := c.List()
	got[0] = rule("MUTATED")
	assert.Equal(t, []string{"R1", "R2"}, ids(c.List()))
}

func TestBuiltinIsValidAndNonEmpty(t *testing.T) {
	assert.NotEmpty(t, Builtin().List())
}

func TestCatalogIsSafeForConcurrentReads(t *testing.T) {
	c, err := New(rule("R1"), rule("R2"))
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			c.Get("R1")
			c.Get("R404")
			_ = c.List()
		})
	}
	wg.Wait()
}
