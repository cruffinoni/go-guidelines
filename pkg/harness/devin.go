package harness

// newDevin returns the Devin harness. Its user-level file is AGENTS.md in the
// user config directory (~/.config/devin, or %APPDATA%\devin on Windows), and
// its project-level file is ./AGENTS.md.
func newDevin() Harness {
	return instructionFile{
		name:      "devin",
		userBase:  inConfigDir("devin"),
		userFile:  "AGENTS.md",
		localFile: "AGENTS.md",
	}
}
