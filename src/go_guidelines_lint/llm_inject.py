"""Inject gg-lint usage instructions into LLM context files."""

from __future__ import annotations

import logging
import shutil
from pathlib import Path

logger = logging.getLogger(__name__)

_LLM_TARGET_DIRS: dict[str, tuple[str, str]] = {
    "claude": (".claude", "CLAUDE.md"),
    "codex": (".codex", "AGENTS.md"),
}

_SENTINEL_START = "<!-- gg-lint-instructions -->"
_SENTINEL_END = "<!-- /gg-lint-instructions -->"

_INSTRUCTIONS_BLOCK = f"""\
{_SENTINEL_START}
## gg-lint — Go Guidelines Linter

`gg-lint` statically analyses Go source code for violations of team coding
guidelines (naming conventions, error handling, concurrency patterns,
comment quality, and more).

### Running gg-lint

```sh
# Lint changed Go files and report findings on changed lines
gg-lint ./... --git --changed-lines

# Output machine-readable JSON (useful for scripting)
gg-lint ./... --format json

# List all active rules and their enabled/disabled state
gg-lint --list-guidelines

# Treat warnings as failures (strict CI mode)
gg-lint ./... --fail-on warning

# Restrict scan to files changed in the current git diff
gg-lint ./... --git

# Run a full-project audit when intentionally reviewing existing debt
gg-lint ./...
```

### Workflow expectation

Before committing Go code, run `gg-lint ./... --git --changed-lines`.
Treat findings as review signals, not automatic rewrite instructions.
Fix only clear, behavior-preserving issues in code you changed.

Do not change public APIs, function signatures, interfaces, data models,
or behavior solely to satisfy `gg-lint`. If a finding appears to be a
false positive or would make the code less idiomatic, leave the code
unchanged and mention the finding in your final summary.

A non-zero exit code means findings require review; it does not by itself
prove that every reported item must be changed. Use full-project
`gg-lint ./...` only for intentional audits or CI context, not as a mandate
to clean up unrelated existing debt.
{_SENTINEL_END}"""


_MAX_BACKUPS = 5


def _backup_file(target: Path) -> None:
    """Rotate backups of *target* up to _MAX_BACKUPS deep.

    Slot assignment (slot 1 = most recent, slot 5 = oldest):
      target.bak.5  <- dropped
      target.bak.4  -> target.bak.5
      ...
      target.bak    -> target.bak.2
      target        -> target.bak
    """
    for i in range(_MAX_BACKUPS - 1, 1, -1):
        src = target.with_suffix(target.suffix + f".bak.{i}")
        dst = target.with_suffix(target.suffix + f".bak.{i + 1}")
        if src.exists():
            src.replace(dst)
    bak = target.with_suffix(target.suffix + ".bak")
    if bak.exists():
        bak.replace(target.with_suffix(target.suffix + ".bak.2"))
    shutil.copy2(target, bak)


def inject_llm_instructions(llm: str) -> bool:
    """Write gg-lint instructions into the appropriate LLM global config file.

    Targets the agent's home directory config:
    - claude: ~/.claude/CLAUDE.md
    - codex:  ~/.codex/AGENTS.md

    If the sentinel is already present the old block is replaced with the
    current one so instructions stay up to date.  A rotating backup is always
    created before any write.  Write failures are logged as warnings so callers
    can continue regardless.  Returns True when the file was modified.
    """
    if llm not in _LLM_TARGET_DIRS:
        logger.warning("Unknown LLM %r, skipping injection.", llm)
        return False
    dir_name, filename = _LLM_TARGET_DIRS[llm]
    target = Path.home() / dir_name / filename

    try:
        existing = target.read_text(encoding="utf-8", errors="replace") if target.exists() else ""

        if _SENTINEL_START in existing:
            before = existing[: existing.index(_SENTINEL_START)].rstrip("\n")
            if _SENTINEL_END in existing:
                after = existing[existing.index(_SENTINEL_END) + len(_SENTINEL_END) :].lstrip("\n")
            else:
                after = ""
            new_content = before + ("\n\n" if before else "") + _INSTRUCTIONS_BLOCK + ("\n\n" + after if after else "\n")
        else:
            new_content = existing + ("\n" if existing and not existing.endswith("\n") else "") + "\n" + _INSTRUCTIONS_BLOCK + "\n"

        target.parent.mkdir(parents=True, exist_ok=True)
        if target.exists():
            _backup_file(target)
        target.write_text(new_content, encoding="utf-8")
        logger.info("Injected gg-lint instructions into %s", target)
        return True
    except OSError as exc:
        logger.warning("Could not write to %s: %s", target, exc)
        return False
