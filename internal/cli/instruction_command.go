package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// RunInstructionSync is the explicit opt-in command for updating one provider's
// repository-local instruction file. Scanning never calls this operation.
func RunInstructionSync(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sync-instructions", flag.ContinueOnError)
	flags.SetOutput(stderr)
	provider := flags.String("provider", "", "instruction provider to update: claude or codex (required)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *provider == "" {
		fmt.Fprintln(stderr, "gg-lint: sync-instructions requires --provider claude or codex")
		return 2
	}
	if len(flags.Args()) != 0 {
		fmt.Fprintln(stderr, "gg-lint: sync-instructions does not accept paths; it updates the Git repository root")
		return 2
	}
	if _, err := instructionTarget(*provider); err != nil {
		fmt.Fprintf(stderr, "gg-lint: %v\n", err)
		return 2
	}
	rootOutput, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		fmt.Fprintln(stderr, "gg-lint: sync-instructions requires a Git worktree")
		return 2
	}
	root := strings.TrimSpace(string(rootOutput))
	changed, err := syncInstructions(root, *provider, os.WriteFile)
	if err != nil {
		fmt.Fprintf(stderr, "gg-lint: %v\n", err)
		return 2
	}
	if len(changed) == 0 {
		fmt.Fprintf(stdout, "Instructions already current (%s).\n", *provider)
		return 0
	}
	fmt.Fprintf(stdout, "Updated %s project instructions: %s\n", *provider, changed[0])
	return 0
}
