"""Tests for LLM instruction injection."""

from __future__ import annotations

import logging
from pathlib import Path
from unittest.mock import patch

import pytest

from go_guidelines_lint.llm_inject import _SENTINEL_START, _SENTINEL_END, _MAX_BACKUPS, inject_llm_instructions


def test_inject_creates_claude_md_when_missing(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    inject_llm_instructions("claude")
    target = tmp_path / ".claude" / "CLAUDE.md"
    assert target.exists()
    assert _SENTINEL_START in target.read_text(encoding="utf-8")


def test_inject_creates_agents_md_for_codex(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    inject_llm_instructions("codex")
    target = tmp_path / ".codex" / "AGENTS.md"
    assert target.exists()
    assert _SENTINEL_START in target.read_text(encoding="utf-8")


def test_inject_replaces_existing_block(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    inject_llm_instructions("claude")
    inject_llm_instructions("claude")
    content = (tmp_path / ".claude" / "CLAUDE.md").read_text(encoding="utf-8")
    assert content.count(_SENTINEL_START) == 1


def test_inject_uses_review_signal_wording(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    inject_llm_instructions("claude")
    content = (tmp_path / ".claude" / "CLAUDE.md").read_text(encoding="utf-8")

    assert "gg-lint ./... --git --changed-lines" in content
    assert "Treat findings as review signals" in content
    assert "Do not change public APIs, function signatures" in content
    assert "fix all" not in content.lower()
    assert "non-zero exit code indicates blocking findings" not in content


def test_inject_replaces_block_preserves_surrounding_content(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    target = tmp_path / ".claude" / "CLAUDE.md"
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(
        f"before content\n{_SENTINEL_START}\nold block\n{_SENTINEL_END}\nafter content\n"
    )
    inject_llm_instructions("claude")
    content = target.read_text(encoding="utf-8")
    assert content.count(_SENTINEL_START) == 1
    assert content.count(_SENTINEL_END) == 1
    assert "before content" in content
    assert "after content" in content
    assert "old block" not in content


def test_inject_appends_to_existing_file_without_sentinel(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    target = tmp_path / ".claude" / "CLAUDE.md"
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("# My Project\n\nExisting instructions.\n")
    inject_llm_instructions("claude")
    bak = target.with_suffix(".md.bak")
    assert bak.exists()
    assert "# My Project" in bak.read_text(encoding="utf-8")
    content = target.read_text(encoding="utf-8")
    assert "# My Project" in content
    assert _SENTINEL_START in content


def test_inject_unknown_llm_does_not_raise(tmp_path: Path, monkeypatch: pytest.MonkeyPatch, caplog: pytest.LogCaptureFixture) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with caplog.at_level(logging.WARNING, logger="go_guidelines_lint.llm_inject"):
        inject_llm_instructions("gemini")
    assert any("Unknown LLM" in r.message for r in caplog.records)


def test_inject_does_not_propagate_oserror(tmp_path: Path, monkeypatch: pytest.MonkeyPatch, caplog: pytest.LogCaptureFixture) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with caplog.at_level(logging.WARNING, logger="go_guidelines_lint.llm_inject"):
        with patch("go_guidelines_lint.llm_inject.Path.write_text", side_effect=OSError("permission denied")):
            inject_llm_instructions("claude")
    assert any("Could not write" in r.message for r in caplog.records)


def test_inject_logs_info_when_injecting(tmp_path: Path, monkeypatch: pytest.MonkeyPatch, caplog: pytest.LogCaptureFixture) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with caplog.at_level(logging.INFO, logger="go_guidelines_lint.llm_inject"):
        inject_llm_instructions("claude")
    assert any("Injected" in r.message for r in caplog.records)


def test_inject_returns_true_on_write(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    assert inject_llm_instructions("claude") is True


def test_inject_returns_false_on_error(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    with patch("go_guidelines_lint.llm_inject.Path.write_text", side_effect=OSError("permission denied")):
        assert inject_llm_instructions("claude") is False


# --- Backup rotation tests ---

def _claude_target(tmp_path: Path) -> Path:
    return tmp_path / ".claude" / "CLAUDE.md"


def _setup_target(tmp_path: Path, content: str = "# Existing\n") -> Path:
    target = _claude_target(tmp_path)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(content, encoding="utf-8")
    return target


def test_backup_not_created_when_file_missing(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    inject_llm_instructions("claude")
    bak = _claude_target(tmp_path).with_suffix(".md.bak")
    assert not bak.exists()


def test_backup_created_on_first_inject(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    target = _setup_target(tmp_path, "original content\n")
    inject_llm_instructions("claude")
    bak = target.with_suffix(".md.bak")
    assert bak.exists()
    assert bak.read_text(encoding="utf-8") == "original content\n"


def test_second_inject_creates_bak2(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    _setup_target(tmp_path, "v1\n")
    inject_llm_instructions("claude")
    target = _claude_target(tmp_path)
    target.write_text("v2\n", encoding="utf-8")
    inject_llm_instructions("claude")
    bak = target.with_suffix(".md.bak")
    bak2 = target.with_suffix(".md.bak.2")
    assert bak2.exists()
    assert bak2.read_text(encoding="utf-8") == "v1\n"
    assert bak.exists()
    assert bak.read_text(encoding="utf-8") == "v2\n"


def test_rotation_up_to_max_backups(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    target = _claude_target(tmp_path)
    target.parent.mkdir(parents=True, exist_ok=True)
    for i in range(_MAX_BACKUPS):
        target.write_text(f"v{i}\n", encoding="utf-8")
        inject_llm_instructions("claude")
        target.write_text(f"v{i}\n", encoding="utf-8")
    bak = target.with_suffix(".md.bak")
    assert bak.exists()
    for n in range(2, _MAX_BACKUPS + 1):
        assert target.with_suffix(f".md.bak.{n}").exists()
    assert not target.with_suffix(f".md.bak.{_MAX_BACKUPS + 1}").exists()


def test_oldest_backup_dropped_beyond_max(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Path, "home", lambda: tmp_path)
    target = _claude_target(tmp_path)
    target.parent.mkdir(parents=True, exist_ok=True)
    for i in range(_MAX_BACKUPS + 1):
        target.write_text(f"v{i}\n", encoding="utf-8")
        inject_llm_instructions("claude")
        target.write_text(f"v{i}\n", encoding="utf-8")
    assert not target.with_suffix(f".md.bak.{_MAX_BACKUPS + 1}").exists()
    assert target.with_suffix(f".md.bak.{_MAX_BACKUPS}").exists()
