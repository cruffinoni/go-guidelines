# gg-lint

`gg-lint` is a Go-native analyzer for the selected Go Guidelines rule set. It scans Go files, reports stable rule IDs and locations, supports project configuration and Git-changes scans, and manages opt-in Claude/Codex instruction blocks. Scans never modify source files.

## Install

One-line installers for Linux, macOS, and Windows download the release
archive, verify it against `checksums.txt`, and install the binary. The Unix
installer targets `~/.local/bin` (or `$XDG_BIN_HOME`) and prints the `PATH`
line when needed; the Windows installer updates the user `PATH`.

**Linux / macOS**

```sh
curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh | sh
```

**Windows (PowerShell 5.1+)**

```powershell
irm https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.ps1 | iex
```

Pin a release by passing the tag to the Unix installer, or by setting
`GG_LINT_VERSION` before the Windows installer runs:

```sh
curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh \
  | sh -s -- --version v0.0.1
```

```powershell
$env:GG_LINT_VERSION = 'v0.0.1'
irm https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.ps1 | iex
```

See [INSTALL.md](INSTALL.md) for manual installs, custom directories, CI
examples, and uninstall steps.

Check the installed build:

```sh
gg-lint version
```

## Use

```sh
gg-lint scan .
gg-lint scan --format=json --git-changes
gg-lint scan --no-config --fail-on=error .   # CI: ignore repository config
gg-lint rules
gg-lint rules --format=markdown
gg-lint sync-instructions --harness claude   # or codex
gg-lint version
```

Every rule is enabled by default. Disable one with `--disable-rule GBP001`
(repeatable), or in `gg-lint.yaml`:

```yaml
format: text    # text or json
fail_on: error  # none, info, warning, or error
rules:
  GBP001: false
```

`--exclude` (repeatable) narrows every scan, not only `--git-changes` scans.

Unless `--config` names a file explicitly, `gg-lint` loads the first config
file it finds among `gg-lint.yaml` at the **scanned target's** Git repository
root and `$XDG_CONFIG_HOME/gg-lint/config.yaml`
(`~/.config/gg-lint/config.yaml` when `XDG_CONFIG_HOME` is unset). Flags always
win over the config file, which wins over the built-in defaults. A scan whose
paths span multiple Git worktrees requires `--config` (or `--no-config`); the
caller's working directory is never used as an implicit fallback.
`--no-config` ignores both project and user configuration.

Repository config is convenient for local development, but it is not a
trustworthy enforcement boundary: a pull request can change it. For CI, use a
trusted config path or disable config entirely:

```sh
gg-lint scan --config=/ci/gg-lint.yaml .
gg-lint scan --no-config --fail-on=error .
```

`--verbose` prints the resolved config path and other scan diagnostics to
stderr without changing the report on stdout.

Finding locations are printed relative to the current directory, like
`go vet`. If the target is on a different Windows volume, the absolute
slash-separated path is used instead.

`gg-lint rules` lists the built-in rule catalog (ID, name, severity, and
summary); pass `--format=markdown` for a documentation-ready rendering with
one section per rule.

The catalog enforces the mechanically checkable project conventions, including
receiver consistency, parameter limits, initializer nil checks, top-down
declaration order, test isolation, error-string style, and `errors.AsType`.
A few conventions remain review criteria because their intent is not reliably
inferable: merging shared test setup into table-driven tests, sorting
map-derived output before assertions, and judging whether a function exists
solely to operate on one struct. GBP018 is a heuristic for the last case.

Scanning a directory requires it to be inside a Go module (a `go.mod` in that
directory or an ancestor); a single `.go` file can be scanned without one.
Directory scans follow `go build ./...` semantics: files excluded by build
tags, `testdata/` directories, and `.`/`_`-prefixed directories are skipped.
`--git-changes` applies the same exclusions to the changed-file set; if no
lintable Go files changed, it reports zero scanned files and exits
successfully. Files marked `// Code generated … DO NOT EDIT.` are loaded for
analysis (and type-checked when an active rule needs type information) but
are never reported on or counted as scanned.

Run the Go checks with `go test ./...`, `go vet ./...`, and `go build -buildvcs=false ./...`. GitHub Actions also runs native Linux, macOS, and Windows tests.

## Build from source

Go 1.27.3 is required (see `go.mod`):

```sh
go test ./...
go build -buildvcs=false -o gg-lint ./cmd/gg-lint
```
