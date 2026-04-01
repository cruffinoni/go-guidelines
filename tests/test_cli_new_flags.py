"""Integration tests for --llm and --git CLI flags."""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest
from click.testing import CliRunner

from go_guidelines_lint.cli import main
from go_guidelines_lint.llm_inject import _SENTINEL


@pytest.fixture()
def runner() -> CliRunner:
    return CliRunner()


def test_llm_claude_creates_claude_md(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        result = runner.invoke(main, ["--llm", "claude", "--list-guidelines"])
        assert result.exit_code == 0, result.output
        target = Path("CLAUDE.md")
        assert target.exists()
        assert _SENTINEL in target.read_text(encoding="utf-8")


def test_llm_codex_creates_agents_md(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        result = runner.invoke(main, ["--llm", "codex", "--list-guidelines"])
        assert result.exit_code == 0, result.output
        target = Path("AGENTS.md")
        assert target.exists()
        assert _SENTINEL in target.read_text(encoding="utf-8")


def test_llm_is_idempotent(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        runner.invoke(main, ["--llm", "claude", "--list-guidelines"])
        runner.invoke(main, ["--llm", "claude", "--list-guidelines"])
        content = Path("CLAUDE.md").read_text(encoding="utf-8")
        assert content.count(_SENTINEL) == 1


def test_git_flag_exits_with_error_outside_repo(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        result = runner.invoke(main, ["--git", "./..."])
        assert result.exit_code == 2


def test_git_flag_limits_scan_to_changed_files(runner: CliRunner, tmp_path: Path) -> None:
    changed = tmp_path / "changed.go"
    unchanged = tmp_path / "unchanged.go"
    changed.write_text("package main\n", encoding="utf-8")
    unchanged.write_text("package main\n", encoding="utf-8")

    mock_proc = MagicMock()
    mock_proc.returncode = 0
    mock_proc.stdout = "changed.go\n"
    mock_proc.stderr = ""

    with patch("go_guidelines_lint.git_filter.subprocess.run", return_value=mock_proc):
        with runner.isolated_filesystem(temp_dir=tmp_path):
            result = runner.invoke(main, ["--git", "--format", "json", str(tmp_path)])

    assert result.exit_code in (0, 1), result.output
    output = json.loads(result.output)
    scanned = output.get("scanned_files", [])
    assert any("changed.go" in f for f in scanned)
    assert not any("unchanged.go" in f for f in scanned)
