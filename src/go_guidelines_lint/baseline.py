"""Baseline serialization and suppression helpers."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from go_guidelines_lint.models import Finding, ScanResult

_BASELINE_VERSION = 1


def _normalize_path(file_path: str, cwd: Path) -> str:
    path = Path(file_path).expanduser()
    if not path.is_absolute():
        path = cwd / path
    try:
        return path.resolve().relative_to(cwd.resolve()).as_posix()
    except ValueError:
        return path.resolve().as_posix()


def _fingerprint(finding: Finding, cwd: Path) -> tuple[object, ...]:
    base: tuple[object, ...] = (
        _normalize_path(finding.file, cwd),
        finding.rule_id,
        finding.message,
    )
    if finding.evidence:
        return (*base, finding.evidence)
    return (*base, finding.line, finding.column)


def _finding_entry(finding: Finding, cwd: Path) -> dict[str, object]:
    entry: dict[str, object] = {
        "file": _normalize_path(finding.file, cwd),
        "rule_id": finding.rule_id,
        "message": finding.message,
    }
    if finding.evidence:
        entry["evidence"] = finding.evidence
    else:
        entry["line"] = finding.line
        entry["column"] = finding.column
    return entry


def _entry_fingerprint(entry: dict[str, Any]) -> tuple[object, ...]:
    base: tuple[object, ...] = (
        str(entry.get("file", "")),
        str(entry.get("rule_id", "")),
        str(entry.get("message", "")),
    )
    evidence = entry.get("evidence")
    if evidence:
        return (*base, str(evidence))
    return (*base, int(entry.get("line", 1)), int(entry.get("column", 1)))


def write_baseline(result: ScanResult, path: Path, cwd: Path) -> None:
    """Write a stable baseline file for the current findings."""

    target = path.expanduser()
    if not target.is_absolute():
        target = cwd / target

    entries = [_finding_entry(finding, cwd) for finding in result.findings]
    entries.sort(key=lambda entry: (str(entry["file"]), str(entry["rule_id"]), str(entry["message"])))
    payload = {
        "version": _BASELINE_VERSION,
        "findings": entries,
    }

    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")


def suppress_baselined_findings(result: ScanResult, path: Path, cwd: Path) -> None:
    """Move findings matching *path*'s baseline out of active findings."""

    source = path.expanduser()
    if not source.is_absolute():
        source = cwd / source

    payload = json.loads(source.read_text(encoding="utf-8"))
    baseline = {_entry_fingerprint(entry) for entry in payload.get("findings", [])}

    active: list[Finding] = []
    suppressed: list[Finding] = []
    for finding in result.findings:
        if _fingerprint(finding, cwd) in baseline:
            suppressed.append(finding)
        else:
            active.append(finding)

    result.findings = active
    result.suppressed_findings.extend(suppressed)
