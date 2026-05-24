"""Tests for git-scoped file filtering."""

from __future__ import annotations

import subprocess
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from go_guidelines_lint.git_filter import GitError, get_git_changed_files, get_git_changed_lines


def _proc(*, returncode: int = 0, stdout: str = "", stderr: str = "") -> MagicMock:
    proc = MagicMock()
    proc.returncode = returncode
    proc.stdout = stdout
    proc.stderr = stderr
    return proc


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


def _commit_all(repo: Path, message: str = "init") -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-q", "-m", message)


def test_raises_git_error_outside_repo(tmp_path: Path) -> None:
    with pytest.raises(GitError, match="not a git repository"):
        get_git_changed_files(tmp_path)


def test_returns_only_go_files_in_repo_with_head(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    tracked_go = tmp_path / "pkg" / "main.go"
    other_go = tmp_path / "cmd" / "server.go"
    tracked_go.parent.mkdir()
    other_go.parent.mkdir()
    tracked_go.write_text("package main\n", encoding="utf-8")
    other_go.write_text("package main\n", encoding="utf-8")
    (tmp_path / "README.md").write_text("hello\n", encoding="utf-8")
    _commit_all(tmp_path)

    tracked_go.write_text("package main\n// changed\n", encoding="utf-8")
    other_go.write_text("package main\n// changed\n", encoding="utf-8")
    (tmp_path / "README.md").write_text("updated\n", encoding="utf-8")

    result = get_git_changed_files(tmp_path)

    assert result == [other_go.resolve(), tracked_go.resolve()]


def test_raises_git_error_when_git_not_in_path(tmp_path: Path) -> None:
    with patch("go_guidelines_lint.git_filter.subprocess.run", side_effect=FileNotFoundError):
        with pytest.raises(GitError, match="git binary not found"):
            get_git_changed_files(tmp_path)


def test_returns_untracked_go_files_in_unborn_repo(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    untracked = tmp_path / "untracked.go"
    untracked.write_text("package main\n", encoding="utf-8")

    result = get_git_changed_files(tmp_path)

    assert result == [untracked.resolve()]


def test_returns_staged_files_in_unborn_repo(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    staged = tmp_path / "staged.go"
    staged.write_text("package main\n", encoding="utf-8")
    _git(tmp_path, "add", "staged.go")

    result = get_git_changed_files(tmp_path)

    assert result == [staged.resolve()]


def test_returns_staged_unstaged_and_untracked_files_once_in_unborn_repo(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    staged = tmp_path / "staged.go"
    untracked = tmp_path / "fresh.go"
    staged.write_text("package main\n", encoding="utf-8")
    untracked.write_text("package main\n", encoding="utf-8")
    _git(tmp_path, "add", "staged.go")
    staged.write_text("package main\n// modified after staging\n", encoding="utf-8")

    result = get_git_changed_files(tmp_path)

    assert result == [staged.resolve(), untracked.resolve()]


def test_resolves_paths_against_repo_root_from_nested_directory(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    target = tmp_path / "pkg" / "sub" / "main.go"
    target.parent.mkdir(parents=True)
    target.write_text("package main\n", encoding="utf-8")
    _commit_all(tmp_path)

    target.write_text("package main\n// changed\n", encoding="utf-8")

    result = get_git_changed_files(tmp_path / "pkg")

    assert result == [target.resolve()]


def test_excludes_ignored_untracked_go_files(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    (tmp_path / ".gitignore").write_text("ignored.go\n", encoding="utf-8")
    kept = tmp_path / "kept.go"
    ignored = tmp_path / "ignored.go"
    kept.write_text("package main\n", encoding="utf-8")
    ignored.write_text("package main\n", encoding="utf-8")

    result = get_git_changed_files(tmp_path)

    assert result == [kept.resolve()]


def test_changed_lines_returns_modified_lines_in_tracked_file(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    target = tmp_path / "main.go"
    target.write_text("package main\n\nfunc Run() {\n\tprintln(\"old\")\n}\n", encoding="utf-8")
    _commit_all(tmp_path)

    target.write_text("package main\n\nfunc Run() {\n\tprintln(\"new\")\n}\n", encoding="utf-8")

    result = get_git_changed_lines(tmp_path)

    assert result == {target.resolve(): {4}}


def test_changed_lines_treats_untracked_file_as_all_lines(tmp_path: Path) -> None:
    _init_repo(tmp_path)
    target = tmp_path / "fresh.go"
    target.write_text("package main\n\nfunc Run() {}\n", encoding="utf-8")

    result = get_git_changed_lines(tmp_path)

    assert result == {target.resolve(): {1, 2, 3}}


def test_skips_paths_outside_git_root(tmp_path: Path) -> None:
    with patch(
        "go_guidelines_lint.git_filter.subprocess.run",
        side_effect=[
            _proc(stdout=f"{tmp_path}\n"),
            _proc(stdout="deadbeef\n"),
            _proc(stdout="../outside/evil.go\npkg/safe.go\n"),
            _proc(stdout=""),
        ],
    ):
        result = get_git_changed_files(tmp_path)

    assert result == [(tmp_path / "pkg" / "safe.go").resolve()]


def test_raises_git_error_for_unexpected_exit_code(tmp_path: Path) -> None:
    with patch(
        "go_guidelines_lint.git_filter.subprocess.run",
        side_effect=[
            _proc(stdout=f"{tmp_path}\n"),
            _proc(stdout="deadbeef\n"),
            _proc(returncode=1, stderr="error: object file is empty"),
        ],
    ):
        with pytest.raises(GitError, match="git exited with code 1"):
            get_git_changed_files(tmp_path)
