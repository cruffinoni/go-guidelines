package main

import (
	"fmt"
	"os"

	"github.com/cruffinoni/go-guidelines/internal/cli"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gg-lint <scan|sync-instructions> [OPTIONS]")
		os.Exit(2)
	}
	switch args[0] {
	case "scan":
		os.Exit(cli.Run(args[1:], os.Stdout, os.Stderr))
	case "sync-instructions":
		os.Exit(cli.RunInstructionSync(args[1:], os.Stdout, os.Stderr))
	default:
		fmt.Fprintln(os.Stderr, "usage: gg-lint <scan|sync-instructions> [OPTIONS]")
		os.Exit(2)
	}
}
