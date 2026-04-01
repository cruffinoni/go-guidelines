"""Git-scoped file filtering for incremental linting."""

from __future__ import annotations

import subprocess
from pathlib import Path


class GitError(RuntimeError):
    """Raised when git is unavailable or the working directory is not a git repository."""


def get_git_changed_files(cwd: Path) -> list[Path]:
    """Return absolute paths of .go files modified relative to HEAD.

    Runs ``git diff HEAD --name-only --diff-filter=ACM`` so deleted files
    are excluded.  Raises ``GitError`` if git is not available or *cwd* is
    not inside a git repository.

    Note: paths are resolved relative to *cwd*, which should be the repository
    root (the standard invocation directory for gg-lint).
    """
    try:
        proc = subprocess.run(
            ["git", "diff", "HEAD", "--name-only", "--diff-filter=ACM"],
            cwd=cwd,
            check=False,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
        )
    except FileNotFoundError as exc:
        raise GitError("git binary not found in PATH") from exc

    if proc.returncode != 0:
        stderr = proc.stderr.strip()
        if "not a git repository" in stderr.lower():
            raise GitError(f"not a git repository: {cwd}")
        raise GitError(f"git exited with code {proc.returncode}: {stderr}")

    paths: list[Path] = []
    for line in proc.stdout.splitlines():
        stripped = line.strip()
        if stripped.endswith(".go"):
            paths.append((cwd / stripped).resolve())
    return paths
