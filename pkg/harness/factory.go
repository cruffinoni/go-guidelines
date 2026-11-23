package harness

// newFactory returns the Factory Droids harness. Its user-level file is
// ~/.factory/AGENTS.md, and its project-level file is ./AGENTS.md.
func newFactory() Harness {
	return instructionFile{
		name:      "factory",
		userBase:  inHome(".factory"),
		userFile:  "AGENTS.md",
		localFile: "AGENTS.md",
	}
}
