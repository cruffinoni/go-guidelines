"""Shared helpers for building finding objects from source indexes."""

from __future__ import annotations

from go_guidelines_lint.models import Finding, GoFile, RuleMeta
from go_guidelines_lint.rules.utils import index_to_line_col


def make_finding(
    meta: RuleMeta,
    go_file: GoFile,
    idx: int,
    message: str,
    *,
    suggestion: str | None = None,
    evidence: str | None = None,
) -> Finding:
    """Create a finding located from a byte-index in file content."""

    line, col = index_to_line_col(go_file.content, idx)
    return Finding(
        rule_id=meta.rule_id,
        guideline_number=meta.guideline_number,
        title=meta.title,
        severity=meta.severity,
        confidence=meta.confidence,
        message=message,
        file=str(go_file.path),
        line=line,
        column=col,
        suggestion=suggestion,
        evidence=evidence,
    )
