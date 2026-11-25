package harness

// newVibe returns the Mistral Vibe CLI harness. Its user-level file is
// ~/.vibe/AGENTS.md, and its project-level file is ./AGENTS.md.
func newVibe() Harness {
	return instructionFile{
		name:      "vibe",
		userBase:  inHome(".vibe"),
		userFile:  "AGENTS.md",
		localFile: "AGENTS.md",
	}
}
