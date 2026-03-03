"""Output renderers for scan results."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any, TextIO

from rich.console import Console
from rich import box
from rich.table import Table

from go_guidelines_lint.models import ScanResult


_SEVERITY_COLOR = {
    "error": "red",
    "warning": "yellow",
    "info": "cyan",
}


def _format_location(file_path: str, line: int, column: int, cwd: Path) -> str:
    """Format finding location with path relative to the execution directory."""

    if not file_path or (file_path.startswith("<") and file_path.endswith(">")):
        base = file_path
    else:
        try:
            base = os.path.relpath(file_path, start=cwd)
        except ValueError:
            base = file_path
    return f"{base}:{line}:{column}"


def render_text(result: ScanResult, stream: TextIO) -> None:
    """Render human-readable output to a stream."""

    # Disable emoji parsing so locations like "file.go:100:1" are rendered literally.
    console = Console(file=stream, emoji=False)
    cwd = Path.cwd()

    summary = result.counts_by_severity()
    summary_bits = [f"files={len(result.scanned_files)}", f"findings={len(result.findings)}", f"elapsed_ms={result.elapsed_ms}"]
    for severity in ["error", "warning", "info"]:
        if severity in summary:
            summary_bits.append(f"{severity}={summary[severity]}")
    console.print("[bold]Scan Summary:[/bold] " + " ".join(summary_bits))

    if result.errors:
        console.print("[bold red]Runtime errors:[/bold red]")
        for err in result.errors:
            console.print(f"- {err}")

    if not result.findings:
        console.print("[green]No violations found.[/green]")
        return

    table = Table(title="Guideline Violations", show_lines=False)
    table.add_column("Rule", no_wrap=True)
    table.add_column("Severity", no_wrap=True)
    table.add_column("Confidence", no_wrap=True)
    table.add_column("Location", no_wrap=True)
    table.add_column("Message")

    for finding in result.findings:
        color = _SEVERITY_COLOR.get(finding.severity, "white")
        table.add_row(
            finding.rule_id,
            f"[{color}]{finding.severity}[/{color}]",
            finding.confidence,
            _format_location(finding.file, finding.line, finding.column, cwd),
            finding.message,
        )

    console.print(table)


def render_json(result: ScanResult, stream: TextIO) -> None:
    """Render machine-readable JSON output."""

    json.dump(result.to_dict(), stream, ensure_ascii=True, indent=2)
    stream.write("\n")


def render_guidelines_text(entries: list[dict[str, Any]], stream: TextIO) -> None:
    """Render guideline catalog with effective enablement state."""

    console = Console(file=stream, emoji=False)
    set_order: list[str] = []
    grouped_entries: dict[str, list[dict[str, Any]]] = {}
    for entry in entries:
        set_name = str(entry.get("set", "default"))
        if set_name not in grouped_entries:
            grouped_entries[set_name] = []
            set_order.append(set_name)
        grouped_entries[set_name].append(entry)

    for set_name in set_order:
        table = Table(title=f"Guideline Catalog ({set_name})", show_lines=True, box=box.SQUARE)
        table.add_column("Rule", no_wrap=True)
        table.add_column("Enabled", no_wrap=True)
        table.add_column("Default", no_wrap=True)
        table.add_column("Severity", no_wrap=True)
        table.add_column("Confidence", no_wrap=True)
        table.add_column("Title", no_wrap=True)
        table.add_column("Description")

        for entry in grouped_entries[set_name]:
            enabled = "yes" if entry["enabled"] else "no"
            default_enabled = "yes" if entry["default_enabled"] else "no"
            enabled_color = "green" if entry["enabled"] else "red"
            color = _SEVERITY_COLOR.get(str(entry["severity"]), "white")
            table.add_row(
                str(entry["rule_id"]),
                f"[{enabled_color}]{enabled}[/{enabled_color}]",
                default_enabled,
                f"[{color}]{entry['severity']}[/{color}]",
                str(entry["confidence"]),
                str(entry["title"]),
                str(entry["description"]),
            )

        console.print(table)


def render_guidelines_json(entries: list[dict[str, Any]], stream: TextIO) -> None:
    """Render guideline catalog as machine-readable JSON."""

    json.dump({"guidelines": entries}, stream, ensure_ascii=True, indent=2)
    stream.write("\n")
