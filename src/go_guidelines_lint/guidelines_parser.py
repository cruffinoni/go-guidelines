"""Parser for guideline markdown sections."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import re


_SECTION_RE = re.compile(r"^##\s+(\d+)\.\s+(.+)$", re.MULTILINE)


@dataclass(slots=True)
class GuidelineSection:
    """Parsed guideline section details."""

    number: int
    title: str
    content: str
    do: str
    dont: str
    rationale: str


def _extract_block(content: str, heading: str) -> str:
    pattern = re.compile(
        rf"\*\*{re.escape(heading)}\*\*\s*(.*?)(?=\n\*\*[A-Za-z`' ]+\*\*|\Z)",
        re.DOTALL,
    )
    match = pattern.search(content)
    if not match:
        return ""
    return match.group(1).strip()


def parse_guidelines(path: Path) -> dict[int, GuidelineSection]:
    """Parse guideline markdown into indexed sections."""

    text = path.read_text(encoding="utf-8")
    matches = list(_SECTION_RE.finditer(text))
    sections: dict[int, GuidelineSection] = {}

    for i, match in enumerate(matches):
        number = int(match.group(1))
        title = match.group(2).strip()
        start = match.end()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
        content = text[start:end].strip()

        sections[number] = GuidelineSection(
            number=number,
            title=title,
            content=content,
            do=_extract_block(content, "Do"),
            dont=_extract_block(content, "Don't"),
            rationale=_extract_block(content, "Rationale"),
        )

    return sections
