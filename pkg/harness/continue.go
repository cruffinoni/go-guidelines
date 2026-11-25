package harness

// newContinue returns the Continue harness. Its user-level file is
// ~/.continue/rules/gg-lint.md, and its project-level file is
// ./.continue/rules/gg-lint.md. New files start with the front matter that
// makes the rule always apply.
func newContinue() Harness {
	return instructionFile{
		name:      "continue",
		userBase:  inHome(".continue"),
		userFile:  "rules/gg-lint.md",
		localFile: ".continue/rules/gg-lint.md",
		header:    "---\nname: gg-lint\nalwaysApply: true\n---\n",
	}
}
