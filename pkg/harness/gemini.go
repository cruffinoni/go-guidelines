package harness

// newGemini returns the Gemini CLI harness. Its user-level file is
// ~/.gemini/GEMINI.md, and its project-level file is ./GEMINI.md.
func newGemini() Harness {
	return instructionFile{
		name:      "gemini",
		userBase:  inHome(".gemini"),
		userFile:  "GEMINI.md",
		localFile: "GEMINI.md",
	}
}
