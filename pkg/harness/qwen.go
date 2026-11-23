package harness

// newQwen returns the Qwen Code harness. Its user-level file is
// ~/.qwen/QWEN.md, and its project-level file is ./QWEN.md.
func newQwen() Harness {
	return instructionFile{
		name:      "qwen",
		userBase:  inHome(".qwen"),
		userFile:  "QWEN.md",
		localFile: "QWEN.md",
	}
}
