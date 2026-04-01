from pathlib import Path

from click.testing import CliRunner

from go_guidelines_lint.cli import main
from go_guidelines_lint.config import AppConfig, load_config, merge_cli_overrides


def test_load_config_defaults_when_missing(tmp_path: Path) -> None:
    config = load_config(tmp_path / "missing.toml")
    assert isinstance(config, AppConfig)
    assert config.target == "./..."
    assert config.format == "text"
    assert config.max_line_length == 120
    assert config.max_workers == 6
    assert config.enable_comments_guidelines is True
    assert config.comments_guidelines_path is None
    assert "GBP010" in config.rules.disable
    assert "GBP012" in config.rules.disable
    assert "GBP015" in config.rules.disable
    assert "GBP020" in config.rules.disable


def test_load_config_from_pyproject(tmp_path: Path) -> None:
    pyproject = tmp_path / "pyproject.toml"
    pyproject.write_text(
        """
[tool.go_guidelines]
format = "json"
max_line_length = 100
max_workers = 2
enable_comments_guidelines = false
comments_guidelines_path = "/tmp/comments.md"
include = ["**/*.go"]
exclude = ["**/vendor/**"]

[tool.go_guidelines.logging]
level = "DEBUG"
format = "json"

[tool.go_guidelines.rules]
enable = ["GBP001"]
""".strip(),
        encoding="utf-8",
    )

    config = load_config(pyproject)
    assert config.format == "json"
    assert config.max_line_length == 100
    assert config.max_workers == 2
    assert config.enable_comments_guidelines is False
    assert config.comments_guidelines_path == "/tmp/comments.md"
    assert config.logging.level == "DEBUG"
    assert config.logging.format == "json"
    assert config.rules.enable == ["GBP001"]
    assert "GBP010" in config.rules.disable
    assert "GBP012" in config.rules.disable
    assert "GBP015" in config.rules.disable
    assert "GBP020" in config.rules.disable


def test_merge_cli_overrides() -> None:
    config = AppConfig()
    merged = merge_cli_overrides(
        config,
        {
            "target": "./pkg/...",
            "format": "json",
            "fail_on": "warning",
            "max_line_length": 120,
            "comments_guidelines_path": "./COMMENTS.md",
            "enable_comments_guidelines": False,
            "log_level": "ERROR",
            "enable_rules": ["GBP001", "GBP003"],
        },
    )

    assert merged.target == "./pkg/..."
    assert merged.format == "json"
    assert merged.fail_on == "warning"
    assert merged.max_line_length == 120
    assert merged.comments_guidelines_path == "./COMMENTS.md"
    assert merged.enable_comments_guidelines is False
    assert merged.logging.level == "ERROR"
    assert merged.rules.enable == ["GBP001", "GBP003"]


def test_load_config_ignores_legacy_tools_section(tmp_path: Path) -> None:
    pyproject = tmp_path / "pyproject.toml"
    pyproject.write_text(
        """
[tool.go_guidelines]
format = "json"

[tool.go_guidelines.tools]
run_gofmt = true
run_goimports = true
run_go_vet = true
run_go_test_race = true
""".strip(),
        encoding="utf-8",
    )

    config = load_config(pyproject)
    assert config.format == "json"
    assert config.tools.run_gofmt is False
    assert config.tools.run_goimports is False
    assert config.tools.run_go_vet is False
    assert config.tools.run_go_test_race is False


def test_cli_rejects_unknown_rule_id_from_toml(tmp_path: Path) -> None:
    pyproject = tmp_path / "pyproject.toml"
    pyproject.write_text(
        """
[tool.go_guidelines]
guidelines_path = "tests/fixtures/basic/GO_BEST_PRACTICES.md"
format = "json"

[tool.go_guidelines.rules]
enable = ["XYZ123"]
""".strip(),
        encoding="utf-8",
    )

    runner = CliRunner()
    result = runner.invoke(main, ["--config", str(pyproject), "--list-guidelines"], catch_exceptions=False)

    assert result.exit_code == 2
    assert "Configuration error:" in result.output
    assert "[tool.go_guidelines.rules.enable]" in result.output
    assert "Unknown rule id(s)" in result.output


def test_cli_rejects_malformed_rule_id_from_toml(tmp_path: Path) -> None:
    pyproject = tmp_path / "pyproject.toml"
    pyproject.write_text(
        """
[tool.go_guidelines]
guidelines_path = "tests/fixtures/basic/GO_BEST_PRACTICES.md"
format = "json"

[tool.go_guidelines.rules]
disable = ["GBP1"]
""".strip(),
        encoding="utf-8",
    )

    runner = CliRunner()
    result = runner.invoke(main, ["--config", str(pyproject), "--list-guidelines"], catch_exceptions=False)

    assert result.exit_code == 2
    assert "Configuration error:" in result.output
    assert "[tool.go_guidelines.rules.disable]" in result.output
    assert "Malformed rule id" in result.output


def test_appconfig_defaults_for_new_fields() -> None:
    config = AppConfig()
    assert config.llm is None
    assert config.git_only is False


def test_merge_cli_overrides_sets_llm() -> None:
    config = AppConfig()
    result = merge_cli_overrides(config, {"llm": "claude"})
    assert result.llm == "claude"


def test_merge_cli_overrides_llm_none_does_not_override() -> None:
    config = merge_cli_overrides(AppConfig(), {"llm": "claude"})
    result = merge_cli_overrides(config, {"llm": None})
    assert result.llm == "claude"


def test_merge_cli_overrides_sets_git_only_true() -> None:
    config = AppConfig()
    result = merge_cli_overrides(config, {"git_only": True})
    assert result.git_only is True


def test_merge_cli_overrides_git_only_false_does_not_swallow() -> None:
    config = merge_cli_overrides(AppConfig(), {"git_only": True})
    result = merge_cli_overrides(config, {"git_only": False})
    assert result.git_only is False


def test_load_config_reads_llm_from_toml(tmp_path: Path) -> None:
    toml = tmp_path / "pyproject.toml"
    toml.write_text('[tool.go_guidelines]\nllm = "claude"\n', encoding="utf-8")
    config = load_config(toml)
    assert config.llm == "claude"


def test_load_config_reads_git_only_from_toml(tmp_path: Path) -> None:
    toml = tmp_path / "pyproject.toml"
    toml.write_text('[tool.go_guidelines]\ngit_only = true\n', encoding="utf-8")
    config = load_config(toml)
    assert config.git_only is True
