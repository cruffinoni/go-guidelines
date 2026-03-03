"""Rule package for go guidelines lint detectors."""

from go_guidelines_lint.rules.detectors import (
    FileContext,
    RuleDefinition,
    analyze_file,
    build_rules,
    run_file_rules,
)
from go_guidelines_lint.rules.comments_detectors import build_comment_rules

__all__ = [
    "FileContext",
    "RuleDefinition",
    "analyze_file",
    "build_rules",
    "build_comment_rules",
    "run_file_rules",
]
