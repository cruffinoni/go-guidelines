# gg-lint

`gg-lint` is a Go-native analyzer for the selected Go Guidelines rule set. It scans Go files, reports stable rule IDs and locations, supports project configuration and changed-file scans, and manages opt-in Claude/Codex instruction blocks. Scans never modify source files.

## Install

Download the archive for your operating system and CPU from the [GitHub Releases page](https://github.com/cruffinoni/go-guidelines/releases). Verify the archive against `checksums.txt`, extract `gg-lint` (or `gg-lint.exe` on Windows), and place it on your `PATH`.

Check the installed build:

```sh
gg-lint version
```

## Use

```sh
gg-lint scan .
gg-lint scan --format=json --changed-files
gg-lint rules
gg-lint rules --format=markdown
gg-lint sync-instructions --harness claude   # or codex
gg-lint version
```

Every rule is enabled by default. Disable one with `--disable-rule GBP001`
(repeatable), or in `gg-lint.toml`:

```toml
format = "text"   # text or json
fail_on = "error" # none, info, warning, or error

[rules]
GBP001 = false
```

`--include` and `--exclude` (repeatable) narrow every scan, not only
`--changed-files` scans.

`gg-lint` looks for `gg-lint.toml` at the Git repository root unless
`--config` names a file explicitly. Flags always win over the config file,
which wins over the built-in defaults.

Finding locations are printed relative to the current directory, like
`go vet`.

`gg-lint rules` lists the built-in rule catalog (ID, name, severity, and
summary); pass `--format=markdown` for a documentation-ready rendering with
one section per rule.

Scanning a directory requires it to be inside a Go module (a `go.mod` in that
directory or an ancestor); a single `.go` file can be scanned without one.
Directory scans follow `go build ./...` semantics: files excluded by build
tags, `testdata/` directories, and `.`/`_`-prefixed directories are skipped.

Run the Go checks with `go test ./...`, `go vet ./...`, and `go build -buildvcs=false ./...`. GitHub Actions also runs native Linux, macOS, and Windows tests.

## Build from source

Go 1.27.3 is required (see `go.mod`):

```sh
go test ./...
go build -buildvcs=false -o gg-lint ./cmd/gg-lint
```
