# gg-lint

`gg-lint` is a Go-native analyzer for the selected Go Guidelines rule set. It scans Go files, reports stable rule IDs and locations, supports project configuration and changed-file scans, and manages opt-in Claude/Codex instruction blocks. A narrowly scoped GBP017 in-process fix is available only through explicit fix flags; normal scans do not modify source files. Executable hooks are not part of V1.

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
gg-lint scan --fix-dry-run .
```

Run the Go checks with `go test ./...`, `go vet ./...`, and `go build -buildvcs=false ./...`. GitHub Actions also runs native Linux, macOS, and Windows tests.

## Build from source

Go 1.27.3 is required (see `go.mod`):

```sh
go test ./...
go build -buildvcs=false -o gg-lint ./cmd/gg-lint
```
