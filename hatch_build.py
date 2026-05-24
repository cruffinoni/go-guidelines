from __future__ import annotations

from copy import deepcopy
import json
import logging
from pathlib import Path
import re
from typing import Any
import tomllib

logger = logging.getLogger(__name__)

_SHARED_CONFIG_DIRNAME = ".gg-lint"
_SHARED_CONFIG_FILENAME = "config.toml"
_BARE_KEY_RE = re.compile(r"^[A-Za-z0-9_-]+$")

try:
    from hatchling.builders.hooks.plugin.interface import BuildHookInterface
except ModuleNotFoundError:  # pragma: no cover
    class BuildHookInterface:  # type: ignore[no-redef]
        def __init__(
            self,
            root: str,
            config: dict[str, Any] | None = None,
            build_config: Any = None,
            metadata: Any = None,
            directory: str | None = None,
            target_name: str | None = None,
            app: Any = None,
        ) -> None:
            self.root = root
            self.config = config or {}
            self.build_config = build_config
            self.metadata = metadata
            self.directory = directory
            self.target_name = target_name
            self.app = app


def shared_config_path(home: Path | None = None) -> Path:
    return (home or Path.home()) / _SHARED_CONFIG_DIRNAME / _SHARED_CONFIG_FILENAME


def _load_toml(path: Path) -> dict[str, Any]:
    with path.open("rb") as f:
        return tomllib.load(f)


def _load_defaults(project_root: Path) -> dict[str, Any]:
    raw = _load_toml(project_root / "pyproject.toml")
    tool = raw.get("tool", {})
    if not isinstance(tool, dict):
        raise ValueError("[tool] must be a TOML table")
    config = tool.get("go_guidelines", {})
    if config == {}:
        return {}
    if not isinstance(config, dict):
        raise ValueError("[tool.go_guidelines] must be a TOML table")
    return config


def _merge_missing(existing: Any, defaults: Any) -> tuple[Any, bool]:
    if isinstance(existing, dict) and isinstance(defaults, dict):
        merged = {key: deepcopy(value) for key, value in existing.items()}
        changed = False
        for key, default_value in defaults.items():
            if key not in merged:
                merged[key] = deepcopy(default_value)
                changed = True
                continue
            merged_value, child_changed = _merge_missing(merged[key], default_value)
            if child_changed:
                merged[key] = merged_value
                changed = True
        return merged, changed
    return existing, False


def _format_key(key: str) -> str:
    return key if _BARE_KEY_RE.match(key) else json.dumps(key)


def _format_value(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    if isinstance(value, str):
        return json.dumps(value)
    if isinstance(value, list):
        return "[" + ", ".join(_format_value(item) for item in value) + "]"
    raise TypeError(f"Unsupported TOML value: {value!r}")


def _render_table(path: tuple[str, ...], data: dict[str, Any]) -> list[str]:
    lines = [f"[{'.'.join(_format_key(part) for part in path)}]"]
    nested: list[tuple[str, dict[str, Any]]] = []
    for key, value in data.items():
        if isinstance(value, dict):
            nested.append((key, value))
            continue
        lines.append(f"{_format_key(key)} = {_format_value(value)}")

    for key, value in nested:
        if lines[-1] != "":
            lines.append("")
        lines.extend(_render_table((*path, key), value))
    return lines


def render_toml_document(data: dict[str, Any]) -> str:
    lines: list[str] = []
    nested: list[tuple[str, dict[str, Any]]] = []
    for key, value in data.items():
        if isinstance(value, dict):
            nested.append((key, value))
            continue
        lines.append(f"{_format_key(key)} = {_format_value(value)}")

    for key, value in nested:
        if lines and lines[-1] != "":
            lines.append("")
        lines.extend(_render_table((key,), value))
    return "\n".join(lines) + "\n"


def seed_shared_config(project_root: Path, home: Path | None = None) -> bool:
    target = shared_config_path(home)

    try:
        defaults = _load_defaults(project_root)
        if not defaults:
            return False

        default_document = {"tool": {"go_guidelines": defaults}}
        if target.exists():
            merged, changed = _merge_missing(_load_toml(target), default_document)
            if not changed:
                return False
            document = merged
        else:
            document = default_document

        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(render_toml_document(document), encoding="utf-8")
        return True
    except (OSError, tomllib.TOMLDecodeError, TypeError, ValueError) as exc:
        logger.warning("Could not seed shared gg-lint config at %s: %s", target, exc)
        return False


class CustomBuildHook(BuildHookInterface):
    def initialize(self, version: str, build_data: dict[str, Any]) -> None:
        del build_data
        if version != "editable":
            return
        seed_shared_config(Path(self.root))
