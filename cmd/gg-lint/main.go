// Package main provides the gg-lint command-line application.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cruffinoni/go-guidelines/internal/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

func execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gg-lint <scan|sync-instructions|version> [OPTIONS]")
		return 2
	}
	switch args[0] {
	case "scan":
		return cli.Run(args[1:], stdout, stderr)
	case "sync-instructions":
		return cli.RunInstructionSync(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "gg-lint %s\ncommit: %s\nbuilt: %s\n", version, commit, builtAt)
		return 0
	default:
		fmt.Fprintln(stderr, "usage: gg-lint <scan|sync-instructions|version> [OPTIONS]")
		return 2
	}
}

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}
