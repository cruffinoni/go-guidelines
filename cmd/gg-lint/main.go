// Gg-lint lints Go code against the Go guidelines.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
)

// version, commit, and builtAt are set at build time via -ldflags by
// .goreleaser.yaml.
var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

func main() {
	cat, err := catalog.Builtin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gg-lint: %v\n", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, cat))
}
