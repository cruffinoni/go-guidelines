"""Configuration loading and CLI override handling."""

from __future__ import annotations

from dataclasses import dataclass, field, replace
from pathlib import Path
from typing import Any
import tomllib

DEFAULT_CONFIG_FILE = "pyproject.toml"
DEFAULT_GUIDELINES_PATH = "~/guidelines/go/GO_BEST_PRACTICES.md"
DEFAULT_TARGET = "./..."
DEFAULT_MAX_LINE_LENGTH = 120
DEFAULT_INCLUDE = ["**/*.go"]
DEFAULT_EXCLUDE = ["**/vendor/**", "**/.git/**", "**/*_generated.go"]
DEFAULT_DISABLED_RULES = ["GBP010", "GBP012", "GBP015"]

_VALID_LEVELS = {"DEBUG", "INFO", "WARNING", "ERROR"}
_VALID_OUTPUT_FORMATS = {"text", "json"}
_VALID_FAIL_ON = {"error", "warning"}


@dataclass(slots=True)
class LoggingConfig:
    """Logging settings for CLI runtime."""

    level: str = "INFO"
    format: str = "text"


@dataclass(slots=True)
class ToolsConfig:
    """Optional external Go tool checks."""

    run_gofmt: bool = False
    run_goimports: bool = False
    run_go_vet: bool = False
    run_go_test_race: bool = False


@dataclass(slots=True)
class RulesConfig:
    """Rule filtering controls."""

    enable: list[str] = field(default_factory=list)
    disable: list[str] = field(default_factory=lambda: list(DEFAULT_DISABLED_RULES))


@dataclass(slots=True)
class AppConfig:
    """Top-level linter configuration."""

    guidelines_path: str = DEFAULT_GUIDELINES_PATH
    comments_guidelines_path: str | None = None
    enable_comments_guidelines: bool = True
    target: str = DEFAULT_TARGET
    format: str = "text"
    fail_on: str = "error"
    max_line_length: int = DEFAULT_MAX_LINE_LENGTH
    max_workers: int = 6
    include: list[str] = field(default_factory=lambda: list(DEFAULT_INCLUDE))
    exclude: list[str] = field(default_factory=lambda: list(DEFAULT_EXCLUDE))
    logging: LoggingConfig = field(default_factory=LoggingConfig)
    tools: ToolsConfig = field(default_factory=ToolsConfig)
    rules: RulesConfig = field(default_factory=RulesConfig)


class ConfigError(ValueError):
    """Raised when configuration is invalid."""


def _as_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        return [str(v) for v in value]
    raise ConfigError(f"Expected a list value, got {type(value)!r}")


def _load_toml(path: Path) -> dict[str, Any]:
    with path.open("rb") as f:
        return tomllib.load(f)


def _extract_tool_config(raw: dict[str, Any]) -> dict[str, Any]:
    tool_root = raw.get("tool", {})
    if not isinstance(tool_root, dict):
        raise ConfigError("[tool] must be a TOML table")
    gg = tool_root.get("go_guidelines", {})
    if gg == {}:
        return {}
    if not isinstance(gg, dict):
        raise ConfigError("[tool.go_guidelines] must be a TOML table")
    return gg


def _validate(config: AppConfig) -> None:
    if config.format not in _VALID_OUTPUT_FORMATS:
        raise ConfigError(f"Invalid format: {config.format!r}")
    if config.fail_on not in _VALID_FAIL_ON:
        raise ConfigError(f"Invalid fail_on: {config.fail_on!r}")
    if config.logging.level.upper() not in _VALID_LEVELS:
        raise ConfigError(f"Invalid logging.level: {config.logging.level!r}")
    if config.logging.format not in _VALID_OUTPUT_FORMATS:
        raise ConfigError(f"Invalid logging.format: {config.logging.format!r}")
    if config.max_line_length < 1:
        raise ConfigError("max_line_length must be >= 1")
    if config.max_workers < 1:
        raise ConfigError("max_workers must be >= 1")


def load_config(config_path: Path | None = None) -> AppConfig:
    """Load linter config from pyproject.toml."""

    config = AppConfig()
    path = config_path or Path(DEFAULT_CONFIG_FILE)
    if not path.exists():
        _validate(config)
        return config

    raw = _load_toml(path)
    data = _extract_tool_config(raw)

    logging_data = data.get("logging", {}) if isinstance(data.get("logging", {}), dict) else {}
    rules_data = data.get("rules", {}) if isinstance(data.get("rules", {}), dict) else {}

    config.guidelines_path = str(data.get("guidelines_path", config.guidelines_path))
    comments_guidelines_path = data.get("comments_guidelines_path", config.comments_guidelines_path)
    config.comments_guidelines_path = str(comments_guidelines_path) if comments_guidelines_path else None
    config.enable_comments_guidelines = bool(data.get("enable_comments_guidelines", config.enable_comments_guidelines))
    config.target = str(data.get("target", config.target))
    config.format = str(data.get("format", config.format))
    config.fail_on = str(data.get("fail_on", config.fail_on))
    config.max_line_length = int(data.get("max_line_length", config.max_line_length))
    config.max_workers = int(data.get("max_workers", config.max_workers))

    include = data.get("include")
    exclude = data.get("exclude")
    if include is not None:
        config.include = _as_list(include)
    if exclude is not None:
        config.exclude = _as_list(exclude)

    config.logging = LoggingConfig(
        level=str(logging_data.get("level", config.logging.level)).upper(),
        format=str(logging_data.get("format", config.logging.format)),
    )
    config.rules = RulesConfig(
        enable=_as_list(rules_data.get("enable", config.rules.enable)),
        disable=_as_list(rules_data.get("disable", config.rules.disable)),
    )

    _validate(config)
    return config


def merge_cli_overrides(config: AppConfig, overrides: dict[str, Any]) -> AppConfig:
    """Merge explicit CLI override values into a config object."""

    merged = replace(config)
    merged.logging = replace(config.logging)
    merged.tools = replace(config.tools)
    merged.rules = replace(config.rules)

    if overrides.get("guidelines_path"):
        merged.guidelines_path = str(overrides["guidelines_path"])
    if overrides.get("comments_guidelines_path"):
        merged.comments_guidelines_path = str(overrides["comments_guidelines_path"])
    if overrides.get("enable_comments_guidelines") is not None:
        merged.enable_comments_guidelines = bool(overrides["enable_comments_guidelines"])
    if overrides.get("target"):
        merged.target = str(overrides["target"])
    if overrides.get("format"):
        merged.format = str(overrides["format"])
    if overrides.get("fail_on"):
        merged.fail_on = str(overrides["fail_on"])
    if overrides.get("max_line_length") is not None:
        merged.max_line_length = int(overrides["max_line_length"])
    if overrides.get("max_workers") is not None:
        merged.max_workers = int(overrides["max_workers"])

    include = overrides.get("include")
    if include:
        merged.include = [str(v) for v in include]
    exclude = overrides.get("exclude")
    if exclude:
        merged.exclude = [str(v) for v in exclude]

    if overrides.get("enable_rules"):
        merged.rules.enable = [str(v) for v in overrides["enable_rules"]]
    if overrides.get("disable_rules"):
        merged.rules.disable = [str(v) for v in overrides["disable_rules"]]

    if overrides.get("log_level"):
        merged.logging.level = str(overrides["log_level"]).upper()
    if overrides.get("log_format"):
        merged.logging.format = str(overrides["log_format"])

    _validate(merged)
    return merged


def expand_user_path(path: str) -> Path:
    """Expand a user path and return an absolute path object."""

    return Path(path).expanduser().resolve()
