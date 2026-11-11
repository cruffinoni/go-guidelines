# Installing gg-lint

`gg-lint` publishes prebuilt archives for Linux, macOS, and Windows on
`amd64` and `arm64`. The installers download the archive for the current OS
and CPU from the latest GitHub release, verify its SHA-256 checksum against
the release's `checksums.txt`, and install the binary.

## Quick install

### Linux and macOS

```sh
curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh | sh
```

The script uses `curl`, `tar`, and a SHA-256 tool (`sha256sum`, `shasum`, or
`openssl`). It installs to `$XDG_BIN_HOME` when set, otherwise to
`~/.local/bin`, and prints the `PATH` line to add when the directory is not
already on `PATH`.

### Windows

Run in PowerShell 5.1 or later:

```powershell
irm https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.ps1 | iex
```

The script installs to `%LOCALAPPDATA%\Programs\gg-lint` and adds that
directory to the user `PATH`. Open a new terminal afterward if the current
one does not see `gg-lint`.

Verify either installation:

```sh
gg-lint version
```

The version line should match the release you installed, for example
`gg-lint <version>`.

## Local build

The installers can build the current working tree and replace the installed
binary. This requires Go and a checkout of this repository.

Linux and macOS:

```sh
./install.sh --local
./install.sh --local --bin-dir "$HOME/.local/bin"
```

Windows:

```powershell
.\install.ps1 -Local
.\install.ps1 -Local -BinDir C:\tools\bin
```

Local mode builds `./cmd/gg-lint` with `CGO_ENABLED=0` and installs over the
`gg-lint` found on PATH, or the default directory when no binary is found.
`--bin-dir` and `-BinDir` choose another target. It does not download a
release, verify a checksum, or modify PATH. `GG_LINT_VERSION` is ignored in
local mode; passing `--version` or `-Version` is an error.

## Update an existing install

Once installed, run:

```sh
gg-lint update
```

The command resolves the latest stable release, downloads the archive for
the current OS and CPU, verifies it against the release's `checksums.txt`,
and installs `gg-lint` into the same directory as the install scripts:
`$XDG_BIN_HOME` or `~/.local/bin` on Linux and macOS, and
`%LOCALAPPDATA%\Programs\gg-lint` on Windows. Use `--bin-dir` or
`GG_LINT_BIN_DIR` to choose another directory. The install directory must be
writable by the current user; do not elevate the command unless you installed
the binary with elevated permissions.

The command installs into that directory even when the running binary lives
elsewhere; the old copy is left in place. Update package-manager installs
through their package manager instead.

The command honors `GG_LINT_VERSION`, `GG_LINT_BIN_DIR`, and
`GG_LINT_NO_PATH_UPDATE`, matching the installers. On Windows it adds the
install directory to the user `PATH` when missing unless `--no-path-update`
or `GG_LINT_NO_PATH_UPDATE=1` is set; restart your terminal afterward. On
Linux and macOS it prints the `PATH` line when the directory is not on
`PATH`.

The command prints a step-by-step progress log on stderr. Interactive terminals
get a spinner and a download progress bar; pipes and CI get plain lines.
`--quiet` prints only the final result and errors.

Pin a specific release, including a prerelease, with `--version`:

```sh
gg-lint update --version vX.Y.Z
gg-lint update --version vX.Y.Z-rc.1
gg-lint update --bin-dir "$HOME/.local/bin"
gg-lint update --quiet
```

Without `--version`, prereleases are never selected. An explicit tag always
installs, so it can also downgrade or repair an install.

When updating the currently running executable on Windows, the previous
binary may remain as `gg-lint.exe.old` until the process exits; the next
update removes it. If latest-release resolution fails, for example through a
proxy, pass `--version`.

## Pin a release

For reproducible installs, pass the release tag. The scripts accept the tag
with or without the leading `v`. Replace `vX.Y.Z` below with the tag you
want.

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh \
  | sh -s -- --version vX.Y.Z
```

Windows:

```powershell
$env:GG_LINT_VERSION = 'vX.Y.Z'
irm https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.ps1 | iex
```

Environment variables work for both scripts and are convenient when the
command is piped to a shell:

| Variable | Purpose |
| --- | --- |
| `GG_LINT_VERSION` | Release tag to install; defaults to the latest stable release. |
| `GG_LINT_BIN_DIR` | Install directory. Unix default: `$XDG_BIN_HOME` or `~/.local/bin`. Windows default: `%LOCALAPPDATA%\Programs\gg-lint`. |
| `GG_LINT_NO_PATH_UPDATE` | Windows only: set to `1` to skip updating the user `PATH`. |

### Custom directory

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh \
  | sh -s -- --bin-dir /usr/local/bin
```

Use `sudo sh -s -- ...` when the directory needs elevated permissions:

```sh
curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh \
  | sudo sh -s -- --bin-dir /usr/local/bin
```

Windows PowerShell:

```powershell
.\install.ps1 -BinDir C:\tools\bin -NoPathUpdate
```

Or, for a piped one-liner:

```powershell
$env:GG_LINT_BIN_DIR = 'C:\tools\bin'
$env:GG_LINT_NO_PATH_UPDATE = '1'
irm https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.ps1 | iex
```

## Supported platforms

| OS | `amd64` | `arm64` |
| --- | --- | --- |
| Linux | yes | yes |
| macOS | yes | yes |
| Windows | yes | yes |

Other targets can be built from source; see the README.

## Manual install

Download the archive and `checksums.txt` from the
[latest release](https://github.com/cruffinoni/go-guidelines/releases/latest)
or choose a tag on the
[releases page](https://github.com/cruffinoni/go-guidelines/releases). Replace
`vX.Y.Z` below with the tag you want; include the leading `v`.

### Linux and macOS

```sh
set -eu
tag=vX.Y.Z
os=linux          # or darwin
arch=amd64        # or arm64
base="https://github.com/cruffinoni/go-guidelines/releases/download/${tag}"
archive="gg-lint_${tag#v}_${os}_${arch}.tar.gz"

curl -fLO "$base/$archive"
curl -fLO "$base/checksums.txt"
line=$(grep " $archive\$" checksums.txt)
test -n "$line"
printf '%s\n' "$line" | sha256sum -c -   # macOS: shasum -a 256 -c -
tar -xzf "$archive"
sudo install -m 0755 gg-lint /usr/local/bin/gg-lint
```

### Windows

In PowerShell:

```powershell
$tag = 'vX.Y.Z'
$arch = 'amd64'
$assetVersion = $tag.TrimStart('v')
$archive = "gg-lint_${assetVersion}_windows_${arch}.zip"
$base = "https://github.com/cruffinoni/go-guidelines/releases/download/$tag"

Invoke-WebRequest "$base/$archive" -OutFile $archive
Invoke-WebRequest "$base/checksums.txt" -OutFile checksums.txt
$pattern = ' ' + [regex]::Escape($archive) + '$'
$expected = (Get-Content checksums.txt | Where-Object { $_ -match $pattern }).Split()[0]
if ((Get-FileHash $archive -Algorithm SHA256).Hash -ne $expected) {
    throw 'checksum mismatch'
}
Expand-Archive $archive -DestinationPath .
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs\gg-lint" | Out-Null
Move-Item .\gg-lint.exe "$env:LOCALAPPDATA\Programs\gg-lint\gg-lint.exe" -Force
```

Add that directory to `PATH` if needed.

## Go users

With Go 1.27.3 or newer:

```sh
go install github.com/cruffinoni/go-guidelines/cmd/gg-lint@latest
```

`go install` builds from source and does not inject the release metadata, so
`gg-lint version` reports `dev`; use a release archive when the version
string matters.

## CI

Pin the version and install into a directory already on `PATH`, or append it
to `GITHUB_PATH`:

```yaml
- name: Install gg-lint
  run: |
    # Replace vX.Y.Z with the release tag you want to pin.
    curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh \
      | sh -s -- --version vX.Y.Z --bin-dir "$HOME/.local/bin"
    echo "$HOME/.local/bin" >> "$GITHUB_PATH"
```

## Uninstall

Linux and macOS:

```sh
rm -f ~/.local/bin/gg-lint
```

Windows:

```powershell
Remove-Item -Recurse -Force "$env:LOCALAPPDATA\Programs\gg-lint"
```

Remove the directory from `PATH` if you added it manually. The Windows
installer adds it to the user `PATH`; edit the user `Path` environment
variable to remove it.

## Troubleshooting

- **`404` while downloading**: check that the tag exists and that it
  publishes an `amd64` or `arm64` archive for your OS.
- **`gg-lint: command not found`**: the install directory is not on `PATH`;
  add the line the Unix installer prints, or open a new terminal on Windows.
- **Checksum mismatch**: the download was corrupted or an intermediate proxy
  changed it. Retry, and verify the tag and archive name.
- **`gg-lint update` permission denied**: the install directory is not
  writable by the current user; reinstall into a writable directory or rerun
  with the permissions you used to install.
- **`gg-lint update` cannot resolve the latest release**: pass
  `--version vX.Y.Z` to skip latest-release discovery.
- **`gg-lint update` installed somewhere else**: it installs into the
  installer directory even when the running binary is elsewhere. Run the
  installed path directly, or pass `--bin-dir` to choose another directory.
- **`gg-lint update` could not replace the binary**: close other `gg-lint`
  processes and retry; the update restores the old binary when replacement
  fails.
- **GitHub API rate limit**: the Windows installer prefers the release-page
  redirect and falls back to the API; pass `-Version` (or set
  `GG_LINT_VERSION`) to skip version discovery.
- **Proxy**: curl honors `HTTPS_PROXY`; PowerShell honors `HTTPS_PROXY` too.

## Security

The installers and `gg-lint update` fetch the archive and `checksums.txt`
over HTTPS and compare SHA-256 hashes before extracting or replacing the
binary. Checksums detect corruption and mismatched downloads; they do not
prove that a release itself is trustworthy. Pin a version and review
`install.sh`, `install.ps1`, or the updater code if your environment requires
it.
