// Gg-lint lints Go code against the Go guidelines.
package main

import (
	"context"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, catalog.Builtin()))
}
