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
`gg-lint 0.0.1`.

## Pin a release

For reproducible installs, pass the release tag. The scripts accept the tag
with or without the leading `v`.

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh \
  | sh -s -- --version v0.0.1
```

Windows:

```powershell
$env:GG_LINT_VERSION = 'v0.0.1'
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
[v0.0.1 release](https://github.com/cruffinoni/go-guidelines/releases/tag/v0.0.1)
(or replace `v0.0.1` with another tag), verify the checksum, and extract the
binary.

### Linux and macOS

```sh
set -eu
version=0.0.1
os=linux          # or darwin
arch=amd64        # or arm64
base="https://github.com/cruffinoni/go-guidelines/releases/download/v${version}"
archive="gg-lint_${version}_${os}_${arch}.tar.gz"

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
$version = '0.0.1'
$arch = 'amd64'
$archive = "gg-lint_${version}_windows_${arch}.zip"
$base = "https://github.com/cruffinoni/go-guidelines/releases/download/v$version"

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
go install github.com/cruffinoni/go-guidelines/cmd/gg-lint@v0.0.1
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
    curl -fsSL https://raw.githubusercontent.com/cruffinoni/go-guidelines/master/install.sh \
      | sh -s -- --version v0.0.1 --bin-dir "$HOME/.local/bin"
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
- **GitHub API rate limit**: the Windows installer prefers the release-page
  redirect and falls back to the API; pass `-Version` (or set
  `GG_LINT_VERSION`) to skip version discovery.
- **Proxy**: curl honors `HTTPS_PROXY`; PowerShell honors `HTTPS_PROXY` too.

## Security

The installers fetch the archive and `checksums.txt` over HTTPS and compare
SHA-256 hashes before extracting. Checksums detect corruption and mismatched
downloads; they do not prove that a release itself is trustworthy. Pin a
version and review `install.sh` or `install.ps1` if your environment requires
it.
