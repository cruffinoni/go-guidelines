"""Integration tests for --llm and --git CLI flags."""

from __future__ import annotations

import json
import subprocess
from pathlib import Path

import pytest
from click.testing import CliRunner

from go_guidelines_lint.cli import main
from go_guidelines_lint.llm_inject import _SENTINEL_START as _SENTINEL

_GUIDELINES_FIXTURE = (Path(__file__).parent / "fixtures" / "basic" / "GO_BEST_PRACTICES.md").resolve()


@pytest.fixture()
def runner() -> CliRunner:
    return CliRunner()


def _git(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=cwd,
        check=True,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )


def _init_repo(repo: Path) -> None:
    _git(repo, "init", "-q")
    _git(repo, "config", "user.name", "Test User")
    _git(repo, "config", "user.email", "test@example.com")


def _write_config(repo: Path) -> None:
    (repo / "pyproject.toml").write_text(
        "\n".join(
            [
                "[tool.go_guidelines]",
                f"guidelines_path = {json.dumps(str(_GUIDELINES_FIXTURE))}",
                'enable_comments_guidelines = false',
                'format = "json"',
                "max_workers = 1",
            ]
        )
        + "\n",
        encoding="utf-8",
    )


def test_llm_claude_creates_claude_md(runner: CliRunner, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with runner.isolated_filesystem():
        result = runner.invoke(main, ["--llm", "claude"])
        assert result.exit_code == 0, result.output
    target = tmp_path / ".claude" / "CLAUDE.md"
    assert target.exists()
    assert _SENTINEL in target.read_text(encoding="utf-8")


def test_llm_codex_creates_agents_md(runner: CliRunner, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with runner.isolated_filesystem():
        result = runner.invoke(main, ["--llm", "codex"])
        assert result.exit_code == 0, result.output
    target = tmp_path / ".codex" / "AGENTS.md"
    assert target.exists()
    assert _SENTINEL in target.read_text(encoding="utf-8")


def test_llm_is_idempotent(runner: CliRunner, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with runner.isolated_filesystem():
        runner.invoke(main, ["--llm", "claude"])
        runner.invoke(main, ["--llm", "claude"])
    content = (tmp_path / ".claude" / "CLAUDE.md").read_text(encoding="utf-8")
    assert content.count(_SENTINEL) == 1


def test_llm_exits_without_running_scan(runner: CliRunner, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with runner.isolated_filesystem():
        result = runner.invoke(main, ["--llm", "claude"])
    assert result.exit_code == 0
    assert "scanned_files" not in result.output
    assert "findings" not in result.output


def test_git_flag_exits_with_error_outside_repo(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        result = runner.invoke(main, ["--git", "./..."])
        assert result.exit_code == 2


def test_git_flag_allows_unborn_repo_with_only_untracked_files(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        repo = Path.cwd()
        _init_repo(repo)
        _write_config(repo)
        Path("untracked.go").write_text("package main\n", encoding="utf-8")
        result = runner.invoke(main, ["--config", "pyproject.toml", "--git", "--format", "json", "./..."])

    assert result.exit_code == 0, result.output
    output = json.loads(result.stdout)
    scanned = output.get("scanned_files", [])
    assert scanned == ["untracked.go"]


def test_git_flag_limits_scan_to_changed_files(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        repo = Path.cwd()
        _init_repo(repo)
        _write_config(repo)
        Path("changed.go").write_text("package main\n", encoding="utf-8")
        Path("unchanged.go").write_text("package main\n", encoding="utf-8")
        _git(repo, "add", ".")
        _git(repo, "commit", "-q", "-m", "init")

        Path("changed.go").write_text("package main\n// changed\n", encoding="utf-8")
        result = runner.invoke(main, ["--config", "pyproject.toml", "--git", "--format", "json", "./..."])

    assert result.exit_code in (0, 1), result.stdout
    output = json.loads(result.stdout)
    scanned = output.get("scanned_files", [])
    assert any("changed.go" in f for f in scanned)
    assert not any("unchanged.go" in f for f in scanned)


def test_git_flag_includes_tracked_and_untracked_go_files_only(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        repo = Path.cwd()
        _init_repo(repo)
        _write_config(repo)
        Path("changed.go").write_text("package main\n", encoding="utf-8")
        Path("unchanged.go").write_text("package main\n", encoding="utf-8")
        Path("ignored.go").write_text("package main\n", encoding="utf-8")
        Path("notes.txt").write_text("note\n", encoding="utf-8")
        Path(".gitignore").write_text("ignored.go\n", encoding="utf-8")
        _git(repo, "add", ".")
        _git(repo, "commit", "-q", "-m", "init")

        Path("changed.go").write_text("package main\n// changed\n", encoding="utf-8")
        Path("fresh.go").write_text("package main\n", encoding="utf-8")
        Path("ignored.go").write_text("package main\n// still ignored\n", encoding="utf-8")
        Path("notes.txt").write_text("note 2\n", encoding="utf-8")
        result = runner.invoke(main, ["--config", "pyproject.toml", "--git", "--format", "json", "./..."])

    assert result.exit_code in (0, 1), result.stdout
    output = json.loads(result.stdout)
    scanned = output.get("scanned_files", [])
    assert "changed.go" in scanned
    assert "fresh.go" in scanned
    assert "unchanged.go" not in scanned
    assert "ignored.go" not in scanned


def test_changed_lines_requires_git_flag(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        result = runner.invoke(main, ["--changed-lines", "./..."])

    assert result.exit_code == 2
    assert "changed_lines requires git_only" in result.output


def test_changed_lines_suppresses_findings_outside_changed_hunks(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        repo = Path.cwd()
        _init_repo(repo)
        _write_config(repo)
        Path("logging.go").write_text(
            """
package sample

import "log"

func Run() {
    log.Printf("existing")
}
""".strip()
            + "\n",
            encoding="utf-8",
        )
        _git(repo, "add", ".")
        _git(repo, "commit", "-q", "-m", "init")

        with Path("logging.go").open("a", encoding="utf-8") as fh:
            fh.write("// changed comment\n")
        result = runner.invoke(
            main,
            ["--config", "pyproject.toml", "--git", "--changed-lines", "--enable-rule", "GBP011", "./..."],
        )

    assert result.exit_code == 0, result.output
    output = json.loads(result.stdout)
    assert output["findings"] == []


def test_changed_lines_keeps_findings_on_changed_hunks(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        repo = Path.cwd()
        _init_repo(repo)
        _write_config(repo)
        Path("logging.go").write_text(
            """
package sample

import "log"

func Run() {
    log.Printf("existing")
}
""".strip()
            + "\n",
            encoding="utf-8",
        )
        _git(repo, "add", ".")
        _git(repo, "commit", "-q", "-m", "init")

        Path("logging.go").write_text(
            """
package sample

import "log"

func Run() {
    log.Printf("changed")
}
""".strip()
            + "\n",
            encoding="utf-8",
        )
        result = runner.invoke(
            main,
            ["--config", "pyproject.toml", "--git", "--changed-lines", "--enable-rule", "GBP011", "./..."],
        )

    assert result.exit_code == 0, result.output
    output = json.loads(result.stdout)
    assert [finding["rule_id"] for finding in output["findings"]] == ["GBP011"]


def test_write_baseline_exits_zero_and_baseline_suppresses_matches(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        repo = Path.cwd()
        _write_config(repo)
        Path("dot.go").write_text('package sample\n\nimport . "fmt"\n\nvar _ = fmt.Println\n', encoding="utf-8")

        write_result = runner.invoke(
            main,
            ["--config", "pyproject.toml", "--enable-rule", "GBP001", "--write-baseline", "baseline.json", "./..."],
        )
        baseline = json.loads(Path("baseline.json").read_text(encoding="utf-8"))
        suppress_result = runner.invoke(
            main,
            ["--config", "pyproject.toml", "--enable-rule", "GBP001", "--baseline", "baseline.json", "./..."],
        )

    assert write_result.exit_code == 0, write_result.output
    assert len(baseline["findings"]) == 1
    assert suppress_result.exit_code == 0, suppress_result.output
    output = json.loads(suppress_result.stdout)
    assert output["findings"] == []
    assert output["suppressed_findings"] == 1


def test_baseline_unmatched_findings_still_block(runner: CliRunner, tmp_path: Path) -> None:
    with runner.isolated_filesystem(temp_dir=tmp_path):
        repo = Path.cwd()
        _write_config(repo)
        Path("dot.go").write_text('package sample\n\nimport . "fmt"\n\nvar _ = fmt.Println\n', encoding="utf-8")
        runner.invoke(
            main,
            ["--config", "pyproject.toml", "--enable-rule", "GBP001", "--write-baseline", "baseline.json", "./..."],
        )
        Path("other.go").write_text('package sample\n\nimport . "log"\n\nvar _ = log.Println\n', encoding="utf-8")

        result = runner.invoke(
            main,
            ["--config", "pyproject.toml", "--enable-rule", "GBP001", "--baseline", "baseline.json", "./..."],
        )

    assert result.exit_code == 1, result.output
    output = json.loads(result.stdout)
    assert len(output["findings"]) == 1
    assert output["suppressed_findings"] == 1
