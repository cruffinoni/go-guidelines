package harness

// newDsh returns the DeepSeek Harness harness. Its user-level file is
// ~/.dsh/AGENTS.md, and its project-level file is ./AGENTS.md.
func newDsh() Harness {
	return instructionFile{
		name:      "dsh",
		userBase:  inHome(".dsh"),
		userFile:  "AGENTS.md",
		localFile: "AGENTS.md",
	}
}
