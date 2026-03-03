"""Guideline catalog listing service."""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

from go_guidelines_lint.config import DEFAULT_DISABLED_RULES, AppConfig, expand_user_path
from go_guidelines_lint.guidelines_parser import parse_guidelines
from go_guidelines_lint.rules.contracts import RuleCatalogEntry, RuleSetSpec
from go_guidelines_lint.rules.registry import get_default_rule_sets
from go_guidelines_lint.services.rule_helpers import build_rule12_meta, is_rule_enabled, rule_description

logger = logging.getLogger(__name__)


class CatalogService:
    """Produce effective guideline catalogs across configured rule-sets."""

    def __init__(self, rule_sets: list[RuleSetSpec] | None = None) -> None:
        self._rule_sets = rule_sets or get_default_rule_sets()

    def _load_sections(self, *, set_name: str, path: Path, enabled: bool, required_when_enabled: bool) -> dict[int, Any]:
        if path.exists():
            return parse_guidelines(path)

        if enabled and required_when_enabled:
            raise FileNotFoundError(f"Guidelines file not found for set '{set_name}': {path}")
        if enabled:
            logger.warning("Guidelines file not found for set '%s': %s. Listing fallback titles.", set_name, path)
        return {}

    def _catalog_entries_for_set(
        self,
        *,
        set_name: str,
        metas,
        sections: dict[int, Any],
        config: AppConfig,
        set_enabled: bool,
    ) -> list[RuleCatalogEntry]:
        entries: list[RuleCatalogEntry] = []
        for meta in sorted(metas, key=lambda item: (item.guideline_number, item.rule_id)):
            section = sections.get(meta.guideline_number)
            entries.append(
                RuleCatalogEntry.from_meta(
                    set_name=set_name,
                    meta=meta,
                    section=section,
                    description=rule_description(section),
                    default_enabled=meta.rule_id not in DEFAULT_DISABLED_RULES,
                    enabled=set_enabled and is_rule_enabled(meta.rule_id, config.rules.enable, config.rules.disable),
                )
            )
        return entries

    def list_guidelines(self, config: AppConfig) -> list[RuleCatalogEntry]:
        """Return guideline catalog entries in source order with effective state."""

        best_guidelines_path = expand_user_path(config.guidelines_path)
        entries: list[RuleCatalogEntry] = []

        for spec in self._rule_sets:
            set_enabled = spec.is_enabled(config)
            path = spec.resolve_guidelines_path(config, best_guidelines_path)
            sections = self._load_sections(
                set_name=spec.name,
                path=path,
                enabled=set_enabled,
                required_when_enabled=spec.required_when_enabled,
            )
            titles = {number: section.title for number, section in sections.items()}
            metas = [rule.meta for rule in spec.build_rules(titles)]
            if spec.name == "best":
                metas.append(build_rule12_meta(titles))
            entries.extend(
                self._catalog_entries_for_set(
                    set_name=spec.name,
                    metas=metas,
                    sections=sections,
                    config=config,
                    set_enabled=set_enabled,
                )
            )

        return entries
