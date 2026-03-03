"""Registry of built-in guideline rule-sets."""

from __future__ import annotations

from pathlib import Path

from go_guidelines_lint.config import expand_user_path
from go_guidelines_lint.rules.comments_detectors import build_comment_rules
from go_guidelines_lint.rules.contracts import RuleSetSpec
from go_guidelines_lint.rules.detectors import build_rules as build_best_rules


def _is_best_enabled(config) -> bool:
    del config
    return True


def _is_comments_enabled(config) -> bool:
    return config.enable_comments_guidelines


def _resolve_best_path(config, best_guidelines_path: Path) -> Path:
    del best_guidelines_path
    return expand_user_path(config.guidelines_path)


def _resolve_comments_path(config, best_guidelines_path: Path) -> Path:
    if config.comments_guidelines_path:
        return expand_user_path(config.comments_guidelines_path)
    return (best_guidelines_path.parent / "COMMENTS.md").resolve()


def get_default_rule_sets() -> list[RuleSetSpec]:
    """Return the built-in rule-set specs in display/evaluation order."""

    return [
        RuleSetSpec(
            name="best",
            default_guidelines_filename="GO_BEST_PRACTICES.md",
            enabled_by_default=True,
            required_when_enabled=True,
            is_enabled=_is_best_enabled,
            resolve_guidelines_path=_resolve_best_path,
            build_rules=build_best_rules,
        ),
        RuleSetSpec(
            name="comments",
            default_guidelines_filename="COMMENTS.md",
            enabled_by_default=True,
            required_when_enabled=False,
            is_enabled=_is_comments_enabled,
            resolve_guidelines_path=_resolve_comments_path,
            build_rules=build_comment_rules,
        ),
    ]
