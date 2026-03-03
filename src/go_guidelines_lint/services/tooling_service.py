"""Optional external tooling integration service."""

from __future__ import annotations

from pathlib import Path
import re
import subprocess
from typing import Iterable

from go_guidelines_lint.config import AppConfig
from go_guidelines_lint.models import Finding, RuleMeta, ToolRun
from go_guidelines_lint.services.types import ScanInputs


class ToolingService:
    """Run optional Go tool checks and convert outputs into findings."""

    @staticmethod
    def _extract_issue_lines(text: str) -> Iterable[tuple[str, int, int, str]]:
        issue_re = re.compile(r"^(.+?\.go):(\d+)(?::(\d+))?:\s*(.+)$")
        for raw_line in text.splitlines():
            match = issue_re.match(raw_line.strip())
            if not match:
                continue
            file_path = match.group(1)
            line = int(match.group(2))
            col = int(match.group(3) or 1)
            msg = match.group(4)
            yield file_path, line, col, msg

    @staticmethod
    def _run_command(command: list[str], cwd: Path) -> tuple[int, str]:
        try:
            proc = subprocess.run(
                command,
                cwd=cwd,
                check=False,
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
            )
            output = "\n".join(part for part in [proc.stdout.strip(), proc.stderr.strip()] if part)
            return proc.returncode, output
        except FileNotFoundError as exc:
            return 127, str(exc)

    @staticmethod
    def _tool_findings_gofmt(tool_name: str, output: str, meta: RuleMeta) -> list[Finding]:
        findings: list[Finding] = []
        for line in output.splitlines():
            stripped = line.strip()
            if not stripped:
                continue
            if stripped.endswith(".go"):
                findings.append(
                    Finding(
                        rule_id=meta.rule_id,
                        guideline_number=meta.guideline_number,
                        title=meta.title,
                        severity="warning",
                        confidence="high",
                        message=f"{tool_name} reported file needs formatting/import normalization.",
                        file=stripped,
                        line=1,
                        column=1,
                        suggestion=f"Run `{tool_name} -w {stripped}`.",
                        evidence=stripped,
                    )
                )
        return findings

    def _tool_findings_issue_output(self, output: str, meta: RuleMeta, tool_name: str) -> list[Finding]:
        findings: list[Finding] = []
        parsed_any = False
        for file_path, line, col, msg in self._extract_issue_lines(output):
            parsed_any = True
            findings.append(
                Finding(
                    rule_id=meta.rule_id,
                    guideline_number=meta.guideline_number,
                    title=meta.title,
                    severity="error",
                    confidence="high",
                    message=f"{tool_name}: {msg}",
                    file=file_path,
                    line=line,
                    column=col,
                    suggestion=f"Address {tool_name} diagnostic.",
                )
            )
        if not parsed_any and output:
            findings.append(
                Finding(
                    rule_id=meta.rule_id,
                    guideline_number=meta.guideline_number,
                    title=meta.title,
                    severity="error",
                    confidence="medium",
                    message=f"{tool_name} reported issues. See tool output.",
                    file="<tool-output>",
                    line=1,
                    column=1,
                    evidence=output.splitlines()[0],
                )
            )
        return findings

    def run_optional_tools(self, inputs: ScanInputs, config: AppConfig, meta: RuleMeta) -> tuple[list[ToolRun], list[Finding], list[str]]:
        tool_runs: list[ToolRun] = []
        findings: list[Finding] = []
        errors: list[str] = []

        if config.tools.run_gofmt and inputs.files:
            command = ["gofmt", "-l", *[str(path) for path in inputs.files]]
            code, output = self._run_command(command, cwd=inputs.cwd)
            file_findings = self._tool_findings_gofmt("gofmt", output, meta)
            findings.extend(file_findings)
            if code == 127:
                errors.append("gofmt binary not found in PATH")
            tool_runs.append(ToolRun("gofmt", command, code, output, len(file_findings)))

        if config.tools.run_goimports and inputs.files:
            command = ["goimports", "-l", *[str(path) for path in inputs.files]]
            code, output = self._run_command(command, cwd=inputs.cwd)
            file_findings = self._tool_findings_gofmt("goimports", output, meta)
            findings.extend(file_findings)
            if code == 127:
                errors.append("goimports binary not found in PATH")
            tool_runs.append(ToolRun("goimports", command, code, output, len(file_findings)))

        if config.tools.run_go_vet:
            command = ["go", "vet", inputs.target]
            code, output = self._run_command(command, cwd=inputs.cwd)
            tool_findings = self._tool_findings_issue_output(output, meta, "go vet") if code != 0 else []
            findings.extend(tool_findings)
            tool_runs.append(ToolRun("go vet", command, code, output, len(tool_findings)))

        if config.tools.run_go_test_race:
            command = ["go", "test", "-race", inputs.target]
            code, output = self._run_command(command, cwd=inputs.cwd)
            tool_findings = self._tool_findings_issue_output(output, meta, "go test -race") if code != 0 else []
            findings.extend(tool_findings)
            tool_runs.append(ToolRun("go test -race", command, code, output, len(tool_findings)))

        return tool_runs, findings, errors
