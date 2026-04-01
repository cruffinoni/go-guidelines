"""Inject gg-lint usage instructions into LLM context files."""

from __future__ import annotations

import logging
from pathlib import Path

logger = logging.getLogger(__name__)

_LLM_TARGET_FILES: dict[str, str] = {
    "claude": "CLAUDE.md",
    "codex": "AGENTS.md",
}

_SENTINEL = "<!-- gg-lint-instructions -->"

_INSTRUCTIONS_BLOCK = f"""\
{_SENTINEL}
## gg-lint — Go Guidelines Linter

`gg-lint` statically analyses Go source code for violations of team coding
guidelines (naming conventions, error handling, concurrency patterns,
comment quality, and more).

### Running gg-lint

```sh
# Lint all Go files from the project root
gg-lint ./...

# Output machine-readable JSON (useful for scripting)
gg-lint ./... --format json

# List all active rules and their enabled/disabled state
gg-lint --list-guidelines

# Treat warnings as failures (strict CI mode)
gg-lint ./... --fail-on warning

# Restrict scan to files changed in the current git diff
gg-lint ./... --git
```

### Workflow expectation

Before committing any Go code change, run `gg-lint ./...` and fix all
reported violations. A non-zero exit code indicates blocking findings
that must be resolved.
"""


def inject_llm_instructions(llm: str, cwd: Path) -> None:
    """Append gg-lint instructions to the appropriate LLM context file.

    The injection is idempotent: if the sentinel marker is already present
    the function returns immediately without modifying the file.  Write
    failures are logged as warnings so the scan continues regardless.
    """
    filename = _LLM_TARGET_FILES[llm]
    target = cwd / filename

    if target.exists():
        existing = target.read_text(encoding="utf-8", errors="replace")
        if _SENTINEL in existing:
            logger.debug("gg-lint instructions already present in %s, skipping.", filename)
            return

    try:
        with target.open("a", encoding="utf-8") as fh:
            fh.write("\n" + _INSTRUCTIONS_BLOCK)
        logger.info("Injected gg-lint instructions into %s", filename)
    except OSError as exc:
        logger.warning("Could not write to %s: %s", filename, exc)
