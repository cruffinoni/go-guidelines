package harness

// newBob returns the IBM Bob harness. Its user-level file is
// ~/.bob/rules/gg-lint.md, and its project-level file is ./.bob/rules/gg-
// lint.md.
func newBob() Harness {
	return instructionFile{
		name:      "bob",
		userBase:  inHome(".bob"),
		userFile:  "rules/gg-lint.md",
		localFile: ".bob/rules/gg-lint.md",
	}
}
