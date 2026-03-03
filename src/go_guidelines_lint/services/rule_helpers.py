"""Shared helpers for scan and catalog services."""

from __future__ import annotations

from pathlib import Path
from typing import Any

from go_guidelines_lint.models import RuleMeta


def is_rule_enabled(rule_id: str, enable: list[str], disable: list[str]) -> bool:
    """Compute final rule enablement from explicit allow/deny lists."""

    enabled = set(enable)
    if enabled and rule_id not in enabled:
        return False
    if rule_id in disable and rule_id not in enabled:
        return False
    return True


def to_rel(path: Path, cwd: Path) -> str:
    """Render a path relative to cwd when possible."""

    try:
        return str(path.resolve().relative_to(cwd.resolve()))
    except ValueError:
        return str(path.resolve())


def build_rule12_meta(guideline_titles: dict[int, str]) -> RuleMeta:
    """Build metadata for special tooling/module rule GBP012."""

    return RuleMeta(
        rule_id="GBP012",
        guideline_number=12,
        title=guideline_titles.get(12, "Tooling and Modules"),
        severity="error",
        confidence="high",
    )


def rule_description(section: Any) -> str:
    """Extract a short description string from a parsed guideline section."""

    if section is None:
        return ""
    for candidate in [section.rationale, section.do, section.dont, section.content]:
        if candidate:
            normalized = " ".join(candidate.split())
            if normalized:
                return normalized
    return ""
