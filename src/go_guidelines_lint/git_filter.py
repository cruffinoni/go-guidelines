"""Git-scoped file filtering for incremental linting."""

from __future__ import annotations

import os
import re
import subprocess
from pathlib import Path
from typing import Iterable


class GitError(RuntimeError):
    """Raised when git is unavailable or the working directory is not a git repository."""


_GIT_ENV = {**os.environ, "LANG": "C", "GIT_TERMINAL_PROMPT": "0"}
_DIFF_FILE_RE = re.compile(r"^\+\+\+ b/(.+)$")
_DIFF_HUNK_RE = re.compile(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@")


def _run_git(command: list[str], cwd: Path) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(
            ["git", *command],
            cwd=cwd,
            check=False,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            env=_GIT_ENV,
        )
    except FileNotFoundError as exc:
        raise GitError("git binary not found in PATH") from exc


def _raise_git_error(proc: subprocess.CompletedProcess[str]) -> None:
    stderr = proc.stderr.strip()
    raise GitError(f"git exited with code {proc.returncode}: {stderr}")


def _resolve_git_root(anchor: Path) -> Path:
    search_cwd = anchor if anchor.is_dir() else anchor.parent
    proc = _run_git(["rev-parse", "--show-toplevel"], search_cwd)
    if proc.returncode != 0:
        if "not a git repository" in proc.stderr.lower():
            raise GitError(f"not a git repository: {search_cwd.resolve()}")
        _raise_git_error(proc)

    root = proc.stdout.strip()
    if not root:
        raise GitError("git did not return a repository root")
    return Path(root).resolve()


def _head_exists(git_root: Path) -> bool:
    proc = _run_git(["rev-parse", "--verify", "HEAD"], git_root)
    if proc.returncode == 0:
        return True

    stderr = proc.stderr.lower()
    if (
        "needed a single revision" in stderr
        or "bad revision" in stderr
        or "unknown revision or path not in the working tree" in stderr
        or "ambiguous argument 'head'" in stderr
    ):
        return False

    _raise_git_error(proc)
    return False


def _git_name_only(git_root: Path, command: list[str]) -> list[str]:
    proc = _run_git(command, git_root)
    if proc.returncode != 0:
        _raise_git_error(proc)
    return [line.strip() for line in proc.stdout.splitlines() if line.strip()]


def _resolve_go_paths(git_root: Path, relative_paths: Iterable[str]) -> list[Path]:
    paths: list[Path] = []
    seen: set[Path] = set()
    for relative_path in relative_paths:
        if not relative_path.endswith(".go"):
            continue

        resolved = (git_root / relative_path).resolve()
        if not resolved.is_relative_to(git_root) or resolved in seen:
            continue

        seen.add(resolved)
        paths.append(resolved)
    return paths


def _resolve_go_path(git_root: Path, relative_path: str) -> Path | None:
    if not relative_path.endswith(".go"):
        return None
    resolved = (git_root / relative_path).resolve()
    if not resolved.is_relative_to(git_root):
        return None
    return resolved


def _tracked_go_paths(git_root: Path) -> list[str]:
    if _head_exists(git_root):
        return _git_name_only(git_root, ["diff", "HEAD", "--name-only", "--diff-filter=ACM"])

    staged = _git_name_only(git_root, ["diff", "--cached", "--name-only", "--diff-filter=ACM", "--root"])
    unstaged = _git_name_only(git_root, ["diff", "--name-only", "--diff-filter=ACM"])
    return [*staged, *unstaged]


def _untracked_go_paths(git_root: Path) -> list[str]:
    return _git_name_only(git_root, ["ls-files", "--others", "--exclude-standard", "--", "*.go"])


def _git_diff(git_root: Path, command: list[str]) -> str:
    proc = _run_git(command, git_root)
    if proc.returncode != 0:
        _raise_git_error(proc)
    return proc.stdout


def _parse_changed_lines(git_root: Path, diff_output: str) -> dict[Path, set[int]]:
    changed: dict[Path, set[int]] = {}
    current_path: Path | None = None

    for line in diff_output.splitlines():
        file_match = _DIFF_FILE_RE.match(line)
        if file_match:
            current_path = _resolve_go_path(git_root, file_match.group(1))
            if current_path is not None:
                changed.setdefault(current_path, set())
            continue

        hunk_match = _DIFF_HUNK_RE.match(line)
        if hunk_match is None or current_path is None:
            continue

        start = int(hunk_match.group(1))
        count = int(hunk_match.group(2) or "1")
        if count == 0:
            continue
        changed.setdefault(current_path, set()).update(range(start, start + count))

    return changed


def _merge_changed_lines(target: dict[Path, set[int]], source: dict[Path, set[int]]) -> None:
    for path, lines in source.items():
        target.setdefault(path, set()).update(lines)


def _all_existing_lines(path: Path) -> set[int]:
    line_count = len(path.read_text(encoding="utf-8", errors="replace").splitlines())
    return set(range(1, max(line_count, 1) + 1))


def get_git_changed_files(anchor: Path) -> list[Path]:
    """Return absolute paths of changed .go files in the current git worktree.

    When ``HEAD`` exists, this uses ``git diff HEAD --name-only --diff-filter=ACM``.
    In repositories without a first commit yet, it combines staged root diff
    output with unstaged tracked changes. Untracked ``.go`` files are included
    via ``git ls-files --others --exclude-standard``. Raises ``GitError`` if
    git is unavailable or *anchor* is not inside a git repository.
    """
    git_root = _resolve_git_root(anchor)
    return _resolve_go_paths(git_root, [*_tracked_go_paths(git_root), *_untracked_go_paths(git_root)])


def get_git_changed_lines(anchor: Path) -> dict[Path, set[int]]:
    """Return changed line numbers for changed .go files in the worktree."""

    git_root = _resolve_git_root(anchor)
    changed: dict[Path, set[int]] = {}

    if _head_exists(git_root):
        diff = _git_diff(git_root, ["diff", "HEAD", "--unified=0", "--diff-filter=ACM", "--", "*.go"])
        _merge_changed_lines(changed, _parse_changed_lines(git_root, diff))
    else:
        cached = _git_diff(git_root, ["diff", "--cached", "--root", "--unified=0", "--diff-filter=ACM", "--", "*.go"])
        unstaged = _git_diff(git_root, ["diff", "--unified=0", "--diff-filter=ACM", "--", "*.go"])
        _merge_changed_lines(changed, _parse_changed_lines(git_root, cached))
        _merge_changed_lines(changed, _parse_changed_lines(git_root, unstaged))

    for relative_path in _untracked_go_paths(git_root):
        resolved = _resolve_go_path(git_root, relative_path)
        if resolved is None or not resolved.exists():
            continue
        changed[resolved] = _all_existing_lines(resolved)

    return changed
