import json
from pathlib import Path

from click.testing import CliRunner
import pytest

from go_guidelines_lint.cli import main


def test_cli_json_output_and_exit_code(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    bad_go = Path("tests/fixtures/basic/bad.go").read_text(encoding="utf-8")

    (tmp_path / "bad.go").write_text(bad_go, encoding="utf-8")
    (tmp_path / "go.mod").write_text("module example.com/demo\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)

    runner = CliRunner()
    result = runner.invoke(
        main,
        ["./...", "--guidelines", str(guideline), "--format", "json"],
        catch_exceptions=False,
    )

    output = result.output
    payload = json.loads(output[output.find("{") :])
    assert isinstance(payload["findings"], list)
    assert any(f["rule_id"] == "GBP001" for f in payload["findings"])
    assert result.exit_code == 1


@pytest.mark.parametrize(
    ("shell", "marker"),
    [
        ("bash", "_GG_LINT_COMPLETE=bash_complete"),
        ("zsh", "#compdef gg-lint"),
        ("fish", "_GG_LINT_COMPLETE=fish_complete"),
    ],
)
def test_cli_export_completion(shell: str, marker: str) -> None:
    runner = CliRunner()
    result = runner.invoke(main, ["--export-completion", shell], catch_exceptions=False)

    assert result.exit_code == 0
    assert marker in result.output


def test_cli_list_guidelines_text_does_not_show_number_column() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(main, ["--guidelines", str(guideline), "--list-guidelines"], catch_exceptions=False)

    assert result.exit_code == 0
    assert "Guideline Catalog (best)" in result.output
    assert "Guideline Catalog (comments)" in result.output
    assert "Number" not in result.output
    assert "Method Organization" in result.output


def test_cli_list_guidelines_json_honors_rule_overrides(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    monkeypatch.chdir(tmp_path)

    runner = CliRunner()
    result = runner.invoke(
        main,
        [
            "--guidelines",
            str(guideline),
            "--list-guidelines",
            "--format",
            "json",
            "--disable-rule",
            "GBP001",
            "--enable-rule",
            "GBP010",
        ],
        catch_exceptions=False,
    )

    payload = json.loads(result.output[result.output.find("{") :])
    guidelines = payload["guidelines"]
    by_rule = {entry["rule_id"]: entry for entry in guidelines}

    assert result.exit_code == 0
    assert guidelines[0]["rule_id"] == "GBP001"
    assert by_rule["GBP001"]["enabled"] is False
    assert by_rule["GBP001"]["default_enabled"] is True
    assert by_rule["GBP001"]["confidence"] == "high"
    assert "number" not in by_rule["GBP001"]

    assert by_rule["GBP010"]["default_enabled"] is False
    assert by_rule["GBP010"]["enabled"] is True
    assert by_rule["GBP010"]["set"] == "best"
    assert by_rule["GBP020"]["default_enabled"] is True
    assert by_rule["GCM001"]["set"] == "comments"


def test_cli_disable_comments_guidelines_hides_gcm_rules_in_listing() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(
        main,
        ["--guidelines", str(guideline), "--list-guidelines", "--format", "json", "--no-comments-guidelines"],
        catch_exceptions=False,
    )

    payload = json.loads(result.output[result.output.find("{") :])
    by_rule = {entry["rule_id"]: entry for entry in payload["guidelines"]}

    assert result.exit_code == 0
    assert by_rule["GCM001"]["enabled"] is False


def test_cli_disable_rule_accepts_comma_separated_values() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(
        main,
        [
            "--guidelines",
            str(guideline),
            "--list-guidelines",
            "--format",
            "json",
            "--disable-rule",
            "GBP001, GBP003",
        ],
        catch_exceptions=False,
    )

    payload = json.loads(result.output[result.output.find("{") :])
    by_rule = {entry["rule_id"]: entry for entry in payload["guidelines"]}

    assert result.exit_code == 0
    assert by_rule["GBP001"]["enabled"] is False
    assert by_rule["GBP003"]["enabled"] is False


def test_cli_disable_rule_rejects_malformed_csv_values() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(
        main,
        ["--guidelines", str(guideline), "--list-guidelines", "--disable-rule", "GBP001,,"],
        catch_exceptions=False,
    )

    assert result.exit_code == 2
    assert "Configuration error:" in result.output
    assert "--disable-rule" in result.output
    assert "Malformed" in result.output


def test_cli_enable_rule_rejects_unknown_rule_id() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(
        main,
        ["--guidelines", str(guideline), "--list-guidelines", "--enable-rule", "XYZ123"],
        catch_exceptions=False,
    )

    assert result.exit_code == 2
    assert "Configuration error:" in result.output
    assert "--enable-rule" in result.output
    assert "Unknown rule id(s)" in result.output


def test_cli_rule_ids_are_case_insensitive_and_normalized() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(
        main,
        [
            "--guidelines",
            str(guideline),
            "--list-guidelines",
            "--format",
            "json",
            "--enable-rule",
            "gbp010,gcm001",
        ],
        catch_exceptions=False,
    )

    payload = json.loads(result.output[result.output.find("{") :])
    by_rule = {entry["rule_id"]: entry for entry in payload["guidelines"]}
    assert result.exit_code == 0
    assert by_rule["GBP010"]["enabled"] is True
    assert by_rule["GCM001"]["enabled"] is True


def test_cli_enable_rule_accepts_comma_separated_values() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(
        main,
        [
            "--guidelines",
            str(guideline),
            "--list-guidelines",
            "--format",
            "json",
            "--enable-rule",
            "GBP010,GBP015",
        ],
        catch_exceptions=False,
    )

    payload = json.loads(result.output[result.output.find("{") :])
    by_rule = {entry["rule_id"]: entry for entry in payload["guidelines"]}

    assert result.exit_code == 0
    assert by_rule["GBP010"]["enabled"] is True
    assert by_rule["GBP015"]["enabled"] is True


def test_cli_rule_flags_support_mixed_repeated_and_comma_values() -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    runner = CliRunner()
    result = runner.invoke(
        main,
        [
            "--guidelines",
            str(guideline),
            "--list-guidelines",
            "--format",
            "json",
            "--disable-rule",
            "GBP001,GBP003",
            "--disable-rule",
            "GBP011",
        ],
        catch_exceptions=False,
    )

    payload = json.loads(result.output[result.output.find("{") :])
    by_rule = {entry["rule_id"]: entry for entry in payload["guidelines"]}

    assert result.exit_code == 0
    assert by_rule["GBP001"]["enabled"] is False
    assert by_rule["GBP003"]["enabled"] is False
    assert by_rule["GBP011"]["enabled"] is False


def test_cli_max_line_length_overrides_threshold(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    comments = Path("tests/fixtures/basic/COMMENTS.md").resolve()
    (tmp_path / "long_lines.go").write_text(
        """
package sample

// This comment should exceed the custom maximum line length threshold only.
func Build(a string, b int, c bool, d float64, e []byte) error {
    return nil
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    runner = CliRunner()
    result = runner.invoke(
        main,
        [
            "./...",
            "--guidelines",
            str(guideline),
            "--comments-guidelines",
            str(comments),
            "--enable-rule",
            "GBP020",
            "--enable-rule",
            "GCM007",
            "--max-line-length",
            "50",
            "--format",
            "json",
        ],
        catch_exceptions=False,
    )

    output = result.output
    payload = json.loads(output[output.find("{") :])
    messages = [finding["message"] for finding in payload["findings"]]

    assert result.exit_code == 0
    assert any("50" in message for message in messages)


def test_cli_help_removes_run_flags_and_updates_description() -> None:
    runner = CliRunner()
    result = runner.invoke(main, ["--help"], catch_exceptions=False)

    assert result.exit_code == 0
    assert "Lint Go code against guideline sets." in result.output
    assert "--run-gofmt" not in result.output
    assert "--no-run-gofmt" not in result.output
    assert "--run-goimports" not in result.output
    assert "--no-run-goimports" not in result.output
    assert "--run-go-vet" not in result.output
    assert "--no-run-go-vet" not in result.output
    assert "--run-go-test-race" not in result.output
    assert "--no-run-go-test-race" not in result.output


def test_cli_help_shows_default_for_max_workers() -> None:
    runner = CliRunner()
    result = runner.invoke(main, ["--help"], catch_exceptions=False)

    assert result.exit_code == 0
    assert ".gg-" in result.output
    assert "config.toml" in result.output
    assert "--max-workers INTEGER" in result.output
    assert "[default: (6)]" in result.output
