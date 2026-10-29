package gbp_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/rules/gbp"
)

var kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestRulesMeta(t *testing.T) {
	rules := gbp.Rules()
	require.NotEmpty(t, rules)

	ids := make(map[string]bool, len(rules))
	names := make(map[string]bool, len(rules))
	for _, rule := range rules {
		meta := rule.Meta()
		t.Run(meta.ID, func(t *testing.T) {
			assert.True(t, strings.HasPrefix(meta.ID, "GBP"), "ID %q lacks the GBP prefix", meta.ID)
			assert.Regexp(t, kebabCase, meta.Name)
			assert.NotEmpty(t, meta.Summary)
			assert.NotEmpty(t, meta.Description)
			assert.GreaterOrEqual(t, meta.Severity, lint.SeverityInfo)
			assert.LessOrEqual(t, meta.Severity, lint.SeverityError)
		})
		assert.False(t, ids[meta.ID], "duplicate ID %q", meta.ID)
		assert.False(t, names[meta.Name], "duplicate name %q", meta.Name)
		ids[meta.ID], names[meta.Name] = true, true
	}
}
