// Package catalog assembles lint.Rule implementations into a validated,
// immutable set that an engine selects rules from and that callers can
// render as a table or as Markdown.
package catalog

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/rules/gbp"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

// Catalog is a validated, immutable set of rules indexed by ID.
type Catalog struct {
	ordered []lint.Rule
	byID    map[string]lint.Rule
}

// New validates rules and returns a Catalog ordered by ID. Every rule must
// have a non-empty, unique ID; a non-empty, unique Name; a non-empty Summary
// and Description; and a Severity in [lint.SeverityInfo, lint.SeverityError].
func New(rules ...lint.Rule) (*Catalog, error) {
	byID := make(map[string]lint.Rule, len(rules))
	names := make(map[string]bool, len(rules))
	for _, r := range rules {
		m := r.Meta()
		if m.ID == "" {
			return nil, errors.New("catalog: rule has an empty ID")
		}
		if _, exists := byID[m.ID]; exists {
			return nil, fmt.Errorf("catalog: duplicate rule ID %q", m.ID)
		}
		if m.Name == "" {
			return nil, fmt.Errorf("catalog: rule %q has an empty Name", m.ID)
		}
		if names[m.Name] {
			return nil, fmt.Errorf("catalog: duplicate rule Name %q", m.Name)
		}
		if m.Summary == "" {
			return nil, fmt.Errorf("catalog: rule %q has an empty Summary", m.ID)
		}
		if m.Description == "" {
			return nil, fmt.Errorf("catalog: rule %q has an empty Description", m.ID)
		}
		if m.Severity < lint.SeverityInfo || m.Severity > lint.SeverityError {
			return nil, fmt.Errorf("catalog: rule %q has an invalid severity %d", m.ID, m.Severity)
		}
		byID[m.ID] = r
		names[m.Name] = true
	}

	ordered := slices.Clone(rules)
	slices.SortFunc(ordered, func(a, b lint.Rule) int {
		return cmp.Compare(a.Meta().ID, b.Meta().ID)
	})
	return &Catalog{ordered: ordered, byID: byID}, nil
}

// Builtin returns the catalog of every GBP and GCM rule. It panics if the
// built-in rule set is invalid, which is a programmer error guarded by a test.
func Builtin() *Catalog {
	c, err := New(slices.Concat(gbp.Rules(), gcm.Rules())...)
	if err != nil {
		panic(fmt.Sprintf("catalog: built-in rule set is invalid: %v", err))
	}
	return c
}

// List returns every rule in the catalog, ordered by ID. The returned slice
// is a copy; mutating it does not affect the catalog.
func (c *Catalog) List() []lint.Rule {
	return slices.Clone(c.ordered)
}

// Get returns the rule with the given ID, if the catalog has one.
func (c *Catalog) Get(id string) (lint.Rule, bool) {
	r, ok := c.byID[id]
	return r, ok
}
