package harness

// newCopilot returns the GitHub Copilot CLI harness. Its user-level file is
// ~/.copilot/copilot-instructions.md, and its project-level file is
// ./.github/copilot-instructions.md.
func newCopilot() Harness {
	return instructionFile{
		name:      "copilot",
		userBase:  inHome(".copilot"),
		userFile:  "copilot-instructions.md",
		localFile: ".github/copilot-instructions.md",
	}
}
