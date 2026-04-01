"""Tests for LLM instruction injection."""

from __future__ import annotations

from pathlib import Path

import pytest

from go_guidelines_lint.llm_inject import _SENTINEL, inject_llm_instructions


def test_inject_creates_claude_md_when_missing(tmp_path: Path) -> None:
    inject_llm_instructions("claude", tmp_path)
    target = tmp_path / "CLAUDE.md"
    assert target.exists()
    content = target.read_text()
    assert _SENTINEL in content


def test_inject_creates_agents_md_for_codex(tmp_path: Path) -> None:
    inject_llm_instructions("codex", tmp_path)
    target = tmp_path / "AGENTS.md"
    assert target.exists()
    content = target.read_text()
    assert _SENTINEL in content


def test_inject_is_idempotent(tmp_path: Path) -> None:
    inject_llm_instructions("claude", tmp_path)
    inject_llm_instructions("claude", tmp_path)
    content = (tmp_path / "CLAUDE.md").read_text()
    assert content.count(_SENTINEL) == 1


def test_inject_skips_when_sentinel_already_present(tmp_path: Path) -> None:
    target = tmp_path / "CLAUDE.md"
    target.write_text(f"existing content\n{_SENTINEL}\nmore content\n")
    inject_llm_instructions("claude", tmp_path)
    content = target.read_text()
    assert content.count(_SENTINEL) == 1
    assert "existing content" in content


def test_inject_appends_to_existing_file_without_sentinel(tmp_path: Path) -> None:
    target = tmp_path / "CLAUDE.md"
    target.write_text("# My Project\n\nExisting instructions.\n")
    inject_llm_instructions("claude", tmp_path)
    content = target.read_text()
    assert "# My Project" in content
    assert _SENTINEL in content
