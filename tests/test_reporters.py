from io import StringIO
from pathlib import Path

from go_guidelines_lint.models import Finding, ScanResult
from go_guidelines_lint.reporters import render_text


def test_render_text_uses_relative_locations(tmp_path: Path, monkeypatch) -> None:
    target = tmp_path / "pkg" / "main.go"
    target.parent.mkdir(parents=True)
    target.write_text("package main\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)

    result = ScanResult(
        findings=[
            Finding(
                rule_id="GBP001",
                guideline_number=1,
                title="Imports and Formatting",
                severity="error",
                confidence="high",
                message="Dot import found",
                file=str(target.resolve()),
                line=7,
                column=3,
            ),
            Finding(
                rule_id="GBP001",
                guideline_number=1,
                title="Imports and Formatting",
                severity="error",
                confidence="high",
                message="Dot import found",
                file=str(target.resolve()),
                line=100,
                column=1,
            ),
        ],
        scanned_files=["pkg/main.go"],
        elapsed_ms=1,
    )

    stream = StringIO()
    render_text(result, stream)
    output = stream.getvalue()

    assert "pkg/main.go:7:3" in output
    assert "pkg/main.go:100:1" in output
    assert "💯" not in output
    assert str(target.resolve()) not in output


def test_render_text_shows_suppressed_count(tmp_path: Path, monkeypatch) -> None:
    target = tmp_path / "pkg" / "main.go"
    target.parent.mkdir(parents=True)
    target.write_text("package main\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)

    result = ScanResult(
        suppressed_findings=[
            Finding(
                rule_id="GBP001",
                guideline_number=1,
                title="Imports and Formatting",
                severity="error",
                confidence="high",
                message="Dot import found",
                file=str(target.resolve()),
            )
        ],
        scanned_files=["pkg/main.go"],
        elapsed_ms=1,
    )

    stream = StringIO()
    render_text(result, stream)

    assert "suppressed=1" in stream.getvalue()
