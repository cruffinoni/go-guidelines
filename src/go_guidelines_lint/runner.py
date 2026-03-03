"""Compatibility facade for scan and catalog operations."""

from __future__ import annotations

from go_guidelines_lint.config import AppConfig
from go_guidelines_lint.models import ScanResult
from go_guidelines_lint.services import CatalogService, ScanService

_scan_service = ScanService()
_catalog_service = CatalogService()


def _severity_blocking_level(fail_on: str) -> set[str]:
    if fail_on == "warning":
        return {"warning", "error"}
    return {"error"}


def run_scan(config: AppConfig) -> ScanResult:
    """Run the full lint scan using effective configuration."""

    return _scan_service.run(config)


def list_guidelines(config: AppConfig) -> list[dict[str, object]]:
    """Return guidelines in source order with effective enablement metadata."""

    return [entry.to_dict() for entry in _catalog_service.list_guidelines(config)]


def has_blocking_findings(result: ScanResult, fail_on: str) -> bool:
    """Return true when scan result contains CI-blocking severities."""

    blocking = _severity_blocking_level(fail_on)
    return any(finding.severity in blocking for finding in result.findings)
