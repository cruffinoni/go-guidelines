package harness

// newOpenCode returns the OpenCode harness. Its user-level file is AGENTS.md
// in $XDG_CONFIG_HOME/opencode (default ~/.config/opencode), and its
// project-level file is ./AGENTS.md.
func newOpenCode() Harness {
	return instructionFile{
		name:      "opencode",
		userBase:  inXDGConfig("opencode"),
		userFile:  "AGENTS.md",
		localFile: "AGENTS.md",
	}
}
