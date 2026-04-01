"""Tests for git-scoped file filtering."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from go_guidelines_lint.git_filter import GitError, get_git_changed_files


def test_raises_git_error_outside_repo(tmp_path: Path) -> None:
    proc = MagicMock()
    proc.returncode = 128
    proc.stderr = "fatal: not a git repository (or any of the parent directories): .git"
    proc.stdout = ""
    with patch("go_guidelines_lint.git_filter.subprocess.run", return_value=proc):
        with pytest.raises(GitError, match="not a git repository"):
            get_git_changed_files(tmp_path)


def test_returns_only_go_files(tmp_path: Path) -> None:
    proc = MagicMock()
    proc.returncode = 0
    proc.stdout = "pkg/main.go\nREADME.md\ncmd/server.go\ngo.sum\n"
    proc.stderr = ""
    with patch("go_guidelines_lint.git_filter.subprocess.run", return_value=proc):
        result = get_git_changed_files(tmp_path)
    assert result == [
        (tmp_path / "pkg/main.go").resolve(),
        (tmp_path / "cmd/server.go").resolve(),
    ]


def test_raises_git_error_when_git_not_in_path(tmp_path: Path) -> None:
    with patch("go_guidelines_lint.git_filter.subprocess.run", side_effect=FileNotFoundError):
        with pytest.raises(GitError, match="git binary not found"):
            get_git_changed_files(tmp_path)


def test_returns_absolute_resolved_paths(tmp_path: Path) -> None:
    proc = MagicMock()
    proc.returncode = 0
    proc.stdout = "internal/foo.go\n"
    proc.stderr = ""
    with patch("go_guidelines_lint.git_filter.subprocess.run", return_value=proc):
        result = get_git_changed_files(tmp_path)
    assert len(result) == 1
    assert result[0].is_absolute()
    assert result[0] == (tmp_path / "internal/foo.go").resolve()


def test_returns_empty_list_when_no_go_files_changed(tmp_path: Path) -> None:
    proc = MagicMock()
    proc.returncode = 0
    proc.stdout = "README.md\ndocs/guide.txt\n"
    proc.stderr = ""
    with patch("go_guidelines_lint.git_filter.subprocess.run", return_value=proc):
        result = get_git_changed_files(tmp_path)
    assert result == []
