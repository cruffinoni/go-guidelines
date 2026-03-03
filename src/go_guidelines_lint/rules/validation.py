"""Validation helpers for rule ID parsing and configuration constraints."""

from __future__ import annotations

import re

from go_guidelines_lint.config import AppConfig, ConfigError
from go_guidelines_lint.rules.registry import get_default_rule_sets
from go_guidelines_lint.services.rule_helpers import build_rule12_meta

_RULE_ID_RE = re.compile(r"^[A-Z]{3}\d{3}$")


def collect_known_rule_ids() -> set[str]:
    """Collect all known built-in rule IDs across registered rule-sets."""

    known_ids: set[str] = set()
    for spec in get_default_rule_sets():
        for rule in spec.build_rules({}):
            known_ids.add(rule.meta.rule_id.upper())
    known_ids.add(build_rule12_meta({}).rule_id.upper())
    return known_ids


def _split_csv_strict(raw: str, source: str) -> list[str]:
    parts = raw.split(",")
    parsed: list[str] = []
    for part in parts:
        token = part.strip()
        if not token:
            raise ConfigError(f"Malformed rule list in {source}: empty token in {raw!r}")
        parsed.append(token)
    return parsed


def parse_rule_csv_values(values: tuple[str, ...], source: str) -> list[str]:
    """Parse comma-separated CLI values and reject malformed input."""

    parsed: list[str] = []
    for raw in values:
        parsed.extend(_split_csv_strict(raw, source))
    return parsed


def normalize_and_validate_rule_ids(ids: list[str], source: str, known_ids: set[str]) -> list[str]:
    """Normalize and validate rule IDs from one source."""

    normalized: list[str] = []
    for rule_id in ids:
        rid = rule_id.strip().upper()
        if not _RULE_ID_RE.match(rid):
            raise ConfigError(f"Malformed rule id {rule_id!r} in {source}; expected pattern XXX999")
        normalized.append(rid)

    unknown = sorted({rid for rid in normalized if rid not in known_ids})
    if unknown:
        raise ConfigError(f"Unknown rule id(s) in {source}: {', '.join(unknown)}")

    # Preserve first-seen order while removing duplicates.
    seen: set[str] = set()
    out: list[str] = []
    for rid in normalized:
        if rid in seen:
            continue
        seen.add(rid)
        out.append(rid)
    return out


def validate_config_rule_ids(config: AppConfig, known_ids: set[str]) -> None:
    """Validate and normalize rule IDs coming from TOML configuration."""

    config.rules.enable = normalize_and_validate_rule_ids(
        list(config.rules.enable),
        "[tool.go_guidelines.rules.enable]",
        known_ids,
    )
    config.rules.disable = normalize_and_validate_rule_ids(
        list(config.rules.disable),
        "[tool.go_guidelines.rules.disable]",
        known_ids,
    )
