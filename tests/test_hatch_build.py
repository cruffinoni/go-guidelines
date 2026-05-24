from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

from hatch_build import CustomBuildHook, seed_shared_config, shared_config_path


def _write_pyproject(repo: Path) -> None:
    (repo / "pyproject.toml").write_text(
        """
[tool.go_guidelines]
guidelines_path = "~/guidelines/go/GO_BEST_PRACTICES.md"
enable_comments_guidelines = true
target = "./..."
format = "text"
fail_on = "error"
max_line_length = 120
max_workers = 6
include = ["**/*.go"]
exclude = ["**/vendor/**", "**/.git/**", "**/*_generated.go"]

[tool.go_guidelines.logging]
level = "INFO"
format = "text"

[tool.go_guidelines.rules]
enable = []
disable = ["GBP010", "GBP012", "GBP015"]
""".strip(),
        encoding="utf-8",
    )


def test_seed_shared_config_creates_file_from_pyproject_defaults(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    repo.mkdir()
    _write_pyproject(repo)
    home = tmp_path / "home"

    changed = seed_shared_config(repo, home)

    assert changed is True
    target = shared_config_path(home)
    assert target.exists()
    content = target.read_text(encoding="utf-8")
    assert "[tool.go_guidelines]" in content
    assert 'format = "text"' in content
    assert '[tool.go_guidelines.rules]' in content


def test_seed_shared_config_preserves_existing_values_and_adds_missing_keys(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    repo.mkdir()
    _write_pyproject(repo)
    home = tmp_path / "home"
    target = shared_config_path(home)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(
        """
[tool.go_guidelines]
format = "json"

[tool.go_guidelines.logging]
level = "DEBUG"
""".strip(),
        encoding="utf-8",
    )

    changed = seed_shared_config(repo, home)

    assert changed is True
    content = target.read_text(encoding="utf-8")
    assert 'format = "json"' in content
    assert 'fail_on = "error"' in content
    assert 'level = "DEBUG"' in content
    assert '[tool.go_guidelines.rules]' in content


def test_seed_shared_config_is_idempotent(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    repo.mkdir()
    _write_pyproject(repo)
    home = tmp_path / "home"

    assert seed_shared_config(repo, home) is True
    first = shared_config_path(home).read_text(encoding="utf-8")
    assert seed_shared_config(repo, home) is False
    second = shared_config_path(home).read_text(encoding="utf-8")
    assert first == second


def test_seed_shared_config_does_not_clobber_invalid_existing_toml(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    repo.mkdir()
    _write_pyproject(repo)
    home = tmp_path / "home"
    target = shared_config_path(home)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("[tool.go_guidelines\n", encoding="utf-8")
    original = target.read_text(encoding="utf-8")

    changed = seed_shared_config(repo, home)

    assert changed is False
    assert target.read_text(encoding="utf-8") == original


def test_custom_build_hook_only_seeds_for_editable_builds(tmp_path: Path) -> None:
    hook = CustomBuildHook(str(tmp_path), {}, None, None, None, "wheel")

    with patch("hatch_build.seed_shared_config") as seed:
        hook.initialize("standard", {})
        seed.assert_not_called()
        hook.initialize("editable", {})
        seed.assert_called_once_with(Path(tmp_path))
