package harness

// newCline returns the Cline harness. Its user-level file is
// ~/Documents/Cline/Rules/gg-lint.md, and its project-level file is
// ./.clinerules/gg-lint.md.
func newCline() Harness {
	return instructionFile{
		name:      "cline",
		userBase:  inHome("Documents/Cline"),
		userFile:  "Rules/gg-lint.md",
		localFile: ".clinerules/gg-lint.md",
	}
}
