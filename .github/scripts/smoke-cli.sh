#!/usr/bin/env bash
# Exercise the built CLI against a temporary Go module and Git repository.
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cd "$repo_root"
go build -buildvcs=false -o "$tmp/gg-lint" ./cmd/gg-lint

mkdir -p "$tmp/project/pkg"
printf 'module example.test/project\n\ngo 1.27\n' > "$tmp/project/go.mod"
printf 'package pkg\nimport . "fmt"\n\nvar _ = Sprintf\n' > "$tmp/project/pkg/bad.go"
printf 'package pkg\n' > "$tmp/project/pkg/ok.go"

(
	cd "$tmp/project"

	set +e
	"$tmp/gg-lint" scan . > "$tmp/full.txt"
	status=$?
	set -e
	test "$status" -eq 1
	grep -q 'GBP001' "$tmp/full.txt"

	set +e
	"$tmp/gg-lint" scan --format=json . > "$tmp/full.json"
	status=$?
	set -e
	test "$status" -eq 1
	grep -q '"scanned_files":2' "$tmp/full.json"

	git init -q
	git config user.email validate@example.invalid
	git config user.name validate
	printf '%s\n' 'pkg/ok.go' > .gitignore
	git add .gitignore pkg/bad.go

	set +e
	"$tmp/gg-lint" scan --git-changes --format=json . > "$tmp/changed.json"
	status=$?
	set -e
	test "$status" -eq 1
	grep -q '"scanned_files":1' "$tmp/changed.json"

	"$tmp/gg-lint" sync-instructions --harness claude > /dev/null
	test -f CLAUDE.md
	test ! -e AGENTS.md

	set +e
	"$tmp/gg-lint" scan . > /dev/null
	status=$?
	set -e
	test "$status" -eq 1
	test ! -e AGENTS.md
)
