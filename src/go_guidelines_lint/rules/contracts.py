"""Contracts and typed models for rule-set composition."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Callable

from go_guidelines_lint.guidelines_parser import GuidelineSection
from go_guidelines_lint.models import RuleMeta
from go_guidelines_lint.rules.detectors import RuleDefinition

if False:  # pragma: no cover
    from go_guidelines_lint.config import AppConfig


@dataclass(slots=True, frozen=True)
class RuleSetSpec:
    """Description of one logical guideline rule-set."""

    name: str
    default_guidelines_filename: str
    enabled_by_default: bool
    required_when_enabled: bool
    is_enabled: Callable[["AppConfig"], bool]
    resolve_guidelines_path: Callable[["AppConfig", Path], Path]
    build_rules: Callable[[dict[int, str]], list[RuleDefinition]]


@dataclass(slots=True)
class RuleCatalogEntry:
    """Typed catalog entry used for guideline listing output."""

    set: str
    rule_id: str
    title: str
    description: str
    severity: str
    confidence: str
    default_enabled: bool
    enabled: bool

    @classmethod
    def from_meta(
        cls,
        *,
        set_name: str,
        meta: RuleMeta,
        section: GuidelineSection | None,
        description: str,
        default_enabled: bool,
        enabled: bool,
    ) -> "RuleCatalogEntry":
        return cls(
            set=set_name,
            rule_id=meta.rule_id,
            title=section.title if section else meta.title,
            description=description,
            severity=meta.severity,
            confidence=meta.confidence,
            default_enabled=default_enabled,
            enabled=enabled,
        )

    def to_dict(self) -> dict[str, str | bool]:
        """Serialize entry to dict for reporter compatibility."""

        return {
            "set": self.set,
            "rule_id": self.rule_id,
            "title": self.title,
            "description": self.description,
            "severity": self.severity,
            "confidence": self.confidence,
            "default_enabled": self.default_enabled,
            "enabled": self.enabled,
        }
