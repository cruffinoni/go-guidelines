"""Parallel file analysis service."""

from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor, as_completed
import logging
from pathlib import Path

from go_guidelines_lint.models import Finding
from go_guidelines_lint.rules import RuleDefinition, analyze_file, run_file_rules

logger = logging.getLogger(__name__)


class FileAnalysisService:
    """Analyze Go files with selected rules using a worker pool."""

    def analyze(
        self,
        *,
        files: list[Path],
        rules: list[RuleDefinition],
        max_workers: int,
        max_line_length: int,
    ) -> tuple[list[Finding], list[str]]:
        findings: list[Finding] = []
        errors: list[str] = []

        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_map = {executor.submit(analyze_file, path, max_line_length): path for path in files}
            for future in as_completed(future_map):
                path = future_map[future]
                try:
                    ctx = future.result()
                    findings.extend(run_file_rules(ctx, rules))
                except Exception as exc:  # noqa: BLE001
                    logger.exception("Failed to analyze %s", path)
                    errors.append(f"{path}: {exc}")

        return findings, errors
