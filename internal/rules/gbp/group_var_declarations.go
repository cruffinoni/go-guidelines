package gbp

import (
	"go/token"
	"strings"

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
		Description: "Reports a run of three or more consecutive lines starting with `var ` (excluding " +
			"`var (` blocks). Scattering related declarations one per line is harder to scan than one " +
			"grouped block; consolidate them into a single `var (...)` block.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports runs of ungrouped var declarations in pass.
func (GroupVarDeclarations) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		source := pass.Source(file)
		lines := strings.Split(string(source), "\n")
		run := 0
		start := 0
		tf := pass.Fset.File(file.Pos())
		for i, l := range lines {
			trim := strings.TrimSpace(l)
			if strings.HasPrefix(trim, "var ") && !strings.HasPrefix(trim, "var (") {
				if run == 0 {
					start = i + 1
				}
				run++
			} else {
				if run >= 3 && tf != nil {
					pass.Reportf(token.Pos(tf.LineStart(start)), "%d consecutive var declarations can be grouped", run)
				}
				run = 0
			}
		}
		if run >= 3 && tf != nil {
			pass.Reportf(token.Pos(tf.LineStart(start)), "%d consecutive var declarations can be grouped", run)
		}
	}
	return nil
}
