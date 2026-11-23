package harness

// newPi returns the Pi harness. Its user-level file is
// ~/.pi/agent/AGENTS.md, and its project-level file is ./AGENTS.md.
func newPi() Harness {
	return instructionFile{
		name:      "pi",
		userBase:  inHome(".pi"),
		userFile:  "agent/AGENTS.md",
		localFile: "AGENTS.md",
	}
}
