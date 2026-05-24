"""CLI application orchestration helpers."""

from __future__ import annotations

from pathlib import Path
import logging
import sys

import click

from go_guidelines_lint.config import AppConfig
from go_guidelines_lint.baseline import suppress_baselined_findings, write_baseline
from go_guidelines_lint.llm_inject import inject_llm_instructions
from go_guidelines_lint.logging_setup import configure_logging
from go_guidelines_lint.reporters import (
    render_guidelines_json,
    render_guidelines_text,
    render_json,
    render_text,
)
from go_guidelines_lint.rules.validation import normalize_and_validate_rule_ids, parse_rule_csv_values
from go_guidelines_lint.runner import has_blocking_findings, list_guidelines, run_scan

logger = logging.getLogger(__name__)


def build_overrides(
    *,
    target: str | None,
    guidelines_path: Path | None,
    comments_guidelines_path: Path | None,
    enable_comments_guidelines: bool | None,
    output_format: str | None,
    fail_on: str | None,
    log_level: str | None,
    log_format: str | None,
    include: tuple[str, ...],
    exclude: tuple[str, ...],
    enable_rules: tuple[str, ...],
    disable_rules: tuple[str, ...],
    max_line_length: int | None,
    max_workers: int | None,
    known_rule_ids: set[str],
    llm: str | None = None,
    git_only: bool | None = None,
    changed_lines: bool | None = None,
    baseline_path: Path | None = None,
    write_baseline_path: Path | None = None,
) -> dict[str, object]:
    """Build merge-ready override payload from click arguments."""

    enable_rule_values = parse_rule_csv_values(enable_rules, "--enable-rule")
    disable_rule_values = parse_rule_csv_values(disable_rules, "--disable-rule")

    return {
        "target": target,
        "guidelines_path": str(guidelines_path) if guidelines_path else None,
        "comments_guidelines_path": str(comments_guidelines_path) if comments_guidelines_path else None,
        "enable_comments_guidelines": enable_comments_guidelines,
        "format": output_format.lower() if output_format else None,
        "fail_on": fail_on.lower() if fail_on else None,
        "log_level": log_level.upper() if log_level else None,
        "log_format": log_format.lower() if log_format else None,
        "include": list(include),
        "exclude": list(exclude),
        "enable_rules": normalize_and_validate_rule_ids(enable_rule_values, "--enable-rule", known_rule_ids),
        "disable_rules": normalize_and_validate_rule_ids(disable_rule_values, "--disable-rule", known_rule_ids),
        "max_line_length": max_line_length,
        "max_workers": max_workers,
        "llm": llm,
        "git_only": git_only,
        "changed_lines": changed_lines,
        "baseline_path": str(baseline_path) if baseline_path else None,
        "write_baseline_path": str(write_baseline_path) if write_baseline_path else None,
    }


def execute(config: AppConfig, *, list_guidelines_mode: bool, stream=None) -> int:
    """Execute scan or catalog listing and return process exit code."""

    out = stream or sys.stdout

    configure_logging(config.logging.level, config.logging.format)
    logger.debug("Effective config: %s", config)

    if config.llm:
        dir_name, filename = {"claude": (".claude", "CLAUDE.md"), "codex": (".codex", "AGENTS.md")}[config.llm]
        target = Path.home() / dir_name / filename
        if inject_llm_instructions(config.llm):
            click.echo(f"gg-lint instructions written to {target}")
        else:
            click.echo(f"Failed to write gg-lint instructions to {target}", err=True)
        return 0

    if list_guidelines_mode:
        entries = list_guidelines(config)
        if config.format == "json":
            render_guidelines_json(entries, out)
        else:
            render_guidelines_text(entries, out)
        return 0

    result = run_scan(config)
    if not result.errors:
        if config.write_baseline_path:
            write_baseline(result, Path(config.write_baseline_path), Path.cwd())
        elif config.baseline_path:
            suppress_baselined_findings(result, Path(config.baseline_path), Path.cwd())

    if config.format == "json":
        render_json(result, out)
    else:
        render_text(result, out)

    if result.errors:
        return 2
    if config.write_baseline_path:
        return 0
    if has_blocking_findings(result, config.fail_on):
        return 1
    return 0
