# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`go-guidelines-lint` is a Python CLI tool (`gg-lint`) that statically analyzes Go source code for guideline violations. It parses Markdown guideline files and applies regex-based detectors to `.go` files, producing findings as a Rich table or JSON.

## Commands

```sh
# Install in development mode
pip install -e ".[dev]"

# Run all tests
pytest

# Run a single test file
pytest tests/test_runner.py

# Run a single test by name
pytest tests/test_runner.py::test_function_name

# Run the CLI
gg-lint ./...
gg-lint ./... --format json
python -m go_guidelines_lint ./...
```

## Architecture

The tool is structured in layers:

1. **CLI layer** (`cli.py`, `cli_app.py`) — Click entry point, loads `AppConfig`, builds CLI overrides, delegates to `runner.py`
2. **Runner facade** (`runner.py`) — thin facade over `ScanService` and `CatalogService`; exposes `run_scan`, `list_guidelines`, `has_blocking_findings`
3. **Services** (`services/`) — orchestration logic:
   - `scan_service.py` — coordinates file discovery, guideline loading, rule filtering, parallel analysis, optional external tooling
   - `file_analysis_service.py` — runs detectors in a `ThreadPoolExecutor`
   - `tooling_service.py` — optionally invokes `gofmt`, `goimports`, `go vet`, `go test -race` via subprocess
4. **Rules** (`rules/`) — detection logic:
   - `detectors.py` — 20 `_detect_rule_N` functions for GBP001–GBP021
   - `comments_detectors.py` — 9 `_detect_gcm_00N` functions for GCM001–GCM009
   - `registry.py` — registers "best" and "comments" rule sets
   - `utils.py` — brace-balanced `extract_go_functions()` for parsing Go function bodies
5. **Config** (`config.py`) — reads `[tool.go_guidelines]` from `pyproject.toml`; defaults include `max_line_length=120`, `max_workers=6`, several rules disabled by default (GBP010, GBP012, GBP015, GBP020)
6. **Models** (`models.py`) — core types: `RuleMeta`, `Finding`, `ScanResult`, `GoFile`, `ToolRun`; severity mapped from confidence (high→error, medium→warning, low→info)

## Rule Sets

- **GBP rules** (GBP001–GBP021): Go best practices — error handling, naming, concurrency, context, interfaces, imports, etc.
- **GCM rules** (GCM001–GCM009): Comment/documentation quality — exported symbols, doc comment format, tautological comments, etc.

Rules disabled by default: `GBP010`, `GBP012`, `GBP015`, `GBP020`.

## Tests

Tests live in `tests/` and use `pytest` with `tmp_path` + `monkeypatch.chdir()` to create isolated Go project structures on disk. The heaviest test file is `test_runner.py` (26 integration tests). Fixture Go files are in `tests/fixtures/basic/`.
