"""Main scan orchestration service."""

from __future__ import annotations

import logging
from pathlib import Path
import re
import time

from go_guidelines_lint.config import AppConfig, expand_user_path
from go_guidelines_lint.discovery import _normalize_target, resolve_go_files
from go_guidelines_lint.git_filter import GitError, get_git_changed_files
from go_guidelines_lint.guidelines_parser import parse_guidelines
from go_guidelines_lint.models import Finding, RuleMeta, ScanResult
from go_guidelines_lint.rules import RuleDefinition
from go_guidelines_lint.rules.contracts import RuleSetSpec
from go_guidelines_lint.rules.registry import get_default_rule_sets
from go_guidelines_lint.services.file_analysis_service import FileAnalysisService
from go_guidelines_lint.services.rule_helpers import build_rule12_meta, is_rule_enabled, to_rel
from go_guidelines_lint.services.tooling_service import ToolingService
from go_guidelines_lint.services.types import ScanInputs

logger = logging.getLogger(__name__)


class ScanService:
    """Run rule-based static analysis and optional external tool checks."""

    def __init__(
        self,
        *,
        rule_sets: list[RuleSetSpec] | None = None,
        file_analysis_service: FileAnalysisService | None = None,
        tooling_service: ToolingService | None = None,
    ) -> None:
        self._rule_sets = rule_sets or get_default_rule_sets()
        self._file_analysis = file_analysis_service or FileAnalysisService()
        self._tooling = tooling_service or ToolingService()

    def _build_scan_inputs(self, config: AppConfig) -> ScanInputs:
        cwd = Path.cwd().resolve()
        best_guidelines_path = expand_user_path(config.guidelines_path)
        rule_set_paths = {
            spec.name: spec.resolve_guidelines_path(config, best_guidelines_path) for spec in self._rule_sets
        }
        files = resolve_go_files(config.target, config.include, config.exclude, cwd=cwd)
        if config.git_only:
            target_path, _ = _normalize_target(config.target, cwd)
            git_root = target_path if target_path.is_dir() else target_path.parent
            try:
                git_changed = get_git_changed_files(git_root)
            except GitError as exc:
                raise RuntimeError(str(exc)) from exc
            git_changed_set = set(git_changed)
            files = [f for f in files if f in git_changed_set]
            if not files:
                logger.info("No changed .go files in scope for git diff HEAD. Nothing to scan.")
        return ScanInputs(
            cwd=cwd,
            target=config.target,
            best_guidelines_path=best_guidelines_path,
            rule_set_paths=rule_set_paths,
            files=files,
        )

    def _filter_rules(self, rule_defs: list[RuleDefinition], enable: list[str], disable: list[str]) -> list[RuleDefinition]:
        return [rule for rule in rule_defs if is_rule_enabled(rule.meta.rule_id, enable, disable)]

    def _load_enabled_rules(self, config: AppConfig, inputs: ScanInputs) -> tuple[list[RuleDefinition], dict[int, str]]:
        rules: list[RuleDefinition] = []
        best_titles: dict[int, str] = {}

        for spec in self._rule_sets:
            if not spec.is_enabled(config):
                continue

            path = inputs.rule_set_paths[spec.name]
            if not path.exists():
                if spec.required_when_enabled:
                    raise FileNotFoundError(f"Guidelines file not found for set '{spec.name}': {path}")
                logger.warning("Guidelines file not found for set '%s': %s. Skipping rules.", spec.name, path)
                continue

            sections = parse_guidelines(path)
            titles = {number: section.title for number, section in sections.items()}
            if spec.name == "best":
                best_titles = titles
            rules.extend(spec.build_rules(titles))

        return rules, best_titles

    @staticmethod
    def _go_mod_findings(cwd: Path, meta: RuleMeta) -> list[Finding]:
        go_mod = cwd / "go.mod"
        findings: list[Finding] = []
        if not go_mod.exists():
            findings.append(
                Finding(
                    rule_id=meta.rule_id,
                    guideline_number=meta.guideline_number,
                    title=meta.title,
                    severity=meta.severity,
                    confidence="medium",
                    message="`go.mod` file is missing in current working directory.",
                    file=str(go_mod),
                    line=1,
                    column=1,
                    suggestion="Initialize module with `go mod init <module-path>`.",
                )
            )
            return findings

        content = go_mod.read_text(encoding="utf-8", errors="replace")
        if not re.search(r"(?m)^module\s+\S+", content):
            findings.append(
                Finding(
                    rule_id=meta.rule_id,
                    guideline_number=meta.guideline_number,
                    title=meta.title,
                    severity=meta.severity,
                    confidence="high",
                    message="`go.mod` is missing a `module` directive.",
                    file=str(go_mod),
                    line=1,
                    column=1,
                    suggestion="Add `module example.com/project` at top of `go.mod`.",
                )
            )

        if not re.search(r"(?m)^go\s+\d+\.\d+", content):
            findings.append(
                Finding(
                    rule_id=meta.rule_id,
                    guideline_number=meta.guideline_number,
                    title=meta.title,
                    severity=meta.severity,
                    confidence="high",
                    message="`go.mod` is missing a Go version directive.",
                    file=str(go_mod),
                    line=1,
                    column=1,
                    suggestion="Add `go 1.xx` in `go.mod`.",
                )
            )

        return findings

    def run(self, config: AppConfig) -> ScanResult:
        """Run the full scan for configured targets and rule-sets."""

        started = time.perf_counter()
        result = ScanResult()

        inputs = self._build_scan_inputs(config)
        logger.debug("Scanning %d file(s)", len(inputs.files))

        rules, best_titles = self._load_enabled_rules(config, inputs)
        rules = self._filter_rules(rules, config.rules.enable, config.rules.disable)

        file_findings, analysis_errors = self._file_analysis.analyze(
            files=inputs.files,
            rules=rules,
            max_workers=config.max_workers,
            max_line_length=config.max_line_length,
        )
        result.findings.extend(file_findings)
        result.errors.extend(analysis_errors)

        if is_rule_enabled("GBP012", config.rules.enable, config.rules.disable):
            rule12_meta = build_rule12_meta(best_titles)
            result.findings.extend(self._go_mod_findings(inputs.cwd, rule12_meta))
            tool_runs, tool_findings, tool_errors = self._tooling.run_optional_tools(inputs, config, rule12_meta)
            result.tool_runs.extend(tool_runs)
            result.findings.extend(tool_findings)
            result.errors.extend(tool_errors)

        result.scanned_files = [to_rel(path, inputs.cwd) for path in inputs.files]
        result.findings.sort(key=lambda finding: (finding.file, finding.line, finding.column, finding.rule_id))
        result.elapsed_ms = int((time.perf_counter() - started) * 1000)
        return result
