"""Shared service-layer data types."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path


@dataclass(slots=True)
class ScanInputs:
    """Expanded scan inputs derived from CLI/config."""

    cwd: Path
    target: str
    best_guidelines_path: Path
    rule_set_paths: dict[str, Path]
    files: list[Path]
