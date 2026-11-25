package harness

// newRoo returns the Roo Code harness. Its user-level file is
// ~/.roo/rules/gg-lint.md, and its project-level file is
// ./.roo/rules/gg-lint.md.
func newRoo() Harness {
	return instructionFile{
		name:      "roo",
		userBase:  inHome(".roo"),
		userFile:  "rules/gg-lint.md",
		localFile: ".roo/rules/gg-lint.md",
	}
}
