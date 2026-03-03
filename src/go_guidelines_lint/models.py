"""Data models for scan configuration and results."""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any

_CONFIDENCE_TO_SEVERITY = {
    "high": "error",
    "medium": "warning",
    "low": "info",
}


def severity_from_confidence(confidence: str, default: str = "warning") -> str:
    """Map confidence to severity using project policy."""

    return _CONFIDENCE_TO_SEVERITY.get(str(confidence).lower(), default)


@dataclass(slots=True)
class RuleMeta:
    """Static metadata for a guideline rule."""

    rule_id: str
    guideline_number: int
    title: str
    severity: str = "warning"
    confidence: str = "medium"

    def __post_init__(self) -> None:
        """Keep severity aligned with confidence-level policy."""

        self.severity = severity_from_confidence(self.confidence, self.severity)


@dataclass(slots=True)
class Finding:
    """A single rule violation or quality signal."""

    rule_id: str
    guideline_number: int
    title: str
    severity: str
    confidence: str
    message: str
    file: str
    line: int = 1
    column: int = 1
    suggestion: str | None = None
    evidence: str | None = None

    def __post_init__(self) -> None:
        """Keep severity aligned with confidence-level policy."""

        self.severity = severity_from_confidence(self.confidence, self.severity)

    def to_dict(self) -> dict[str, Any]:
        """Serialize a finding for JSON output."""
        return asdict(self)


@dataclass(slots=True)
class ToolRun:
    """Result of one optional external tooling check."""

    name: str
    command: list[str]
    return_code: int
    output: str
    findings: int

    def to_dict(self) -> dict[str, Any]:
        """Serialize a tool run for JSON output."""
        return asdict(self)


@dataclass(slots=True)
class ScanResult:
    """Complete output from a lint scan."""

    findings: list[Finding] = field(default_factory=list)
    scanned_files: list[str] = field(default_factory=list)
    elapsed_ms: int = 0
    tool_runs: list[ToolRun] = field(default_factory=list)
    errors: list[str] = field(default_factory=list)

    def counts_by_severity(self) -> dict[str, int]:
        """Count findings grouped by severity."""
        counts: dict[str, int] = {}
        for finding in self.findings:
            counts[finding.severity] = counts.get(finding.severity, 0) + 1
        return counts

    def counts_by_rule(self) -> dict[str, int]:
        """Count findings grouped by rule identifier."""
        counts: dict[str, int] = {}
        for finding in self.findings:
            counts[finding.rule_id] = counts.get(finding.rule_id, 0) + 1
        return counts

    def to_dict(self) -> dict[str, Any]:
        """Serialize an entire scan result for machine-readable output."""
        return {
            "findings": [finding.to_dict() for finding in self.findings],
            "counts_by_severity": self.counts_by_severity(),
            "counts_by_rule": self.counts_by_rule(),
            "scanned_files": self.scanned_files,
            "elapsed_ms": self.elapsed_ms,
            "tool_runs": [tool.to_dict() for tool in self.tool_runs],
            "errors": self.errors,
        }


@dataclass(slots=True)
class GoFile:
    """In-memory representation of one Go source file."""

    path: Path
    content: str

    @property
    def lines(self) -> list[str]:
        """Return file lines split once for rule checks."""
        return self.content.splitlines()
