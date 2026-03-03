"""Path discovery helpers for scan targets."""

from __future__ import annotations

from pathlib import Path
from typing import Iterable
import logging

import pathspec

logger = logging.getLogger(__name__)


def _iter_go_files(base: Path) -> Iterable[Path]:
    for p in base.rglob("*.go"):
        if p.is_file():
            yield p


def _normalize_target(target: str, cwd: Path) -> tuple[Path, bool]:
    if target in {"./...", "..."}:
        return cwd, True
    if target.endswith("/..."):
        return (cwd / target[:-4]).resolve(), True
    path = (cwd / target).resolve() if not Path(target).is_absolute() else Path(target).resolve()
    return path, False


def resolve_go_files(
    target: str,
    include_patterns: list[str],
    exclude_patterns: list[str],
    cwd: Path | None = None,
) -> list[Path]:
    """Resolve one target into matching Go files."""

    current = (cwd or Path.cwd()).resolve()
    target_path, recursive = _normalize_target(target, current)
    if not target_path.exists():
        raise FileNotFoundError(f"Target does not exist: {target}")

    include_spec = pathspec.PathSpec.from_lines("gitignore", include_patterns or ["**/*.go"])
    exclude_spec = pathspec.PathSpec.from_lines("gitignore", exclude_patterns or [])

    files: list[Path] = []
    if target_path.is_file():
        candidates = [target_path]
    elif recursive or target_path.is_dir():
        candidates = list(_iter_go_files(target_path))
    else:
        candidates = []

    for candidate in candidates:
        resolved = candidate.resolve()
        try:
            rel = resolved.relative_to(current).as_posix()
        except ValueError:
            rel = resolved.name
        if not include_spec.match_file(rel):
            continue
        if exclude_spec.match_file(rel):
            continue
        files.append(resolved)

    files.sort()
    logger.debug("Resolved %d go file(s)", len(files))
    return files
