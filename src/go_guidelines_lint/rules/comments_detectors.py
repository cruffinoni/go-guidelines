"""Comment-focused detector implementations mapped to COMMENTS.md."""

from __future__ import annotations

import re

from go_guidelines_lint.models import Finding, RuleMeta
from go_guidelines_lint.rules.detectors import FileContext, RuleDefinition
from go_guidelines_lint.rules.finding_factory import make_finding
from go_guidelines_lint.rules.utils import index_to_line_col


_TYPE_DECL_RE = re.compile(r"^\s*type\s+([A-Za-z_]\w*)\b", re.MULTILINE)
_ERR_VAR_DECL_RE = re.compile(r"^\s*var\s+(Err[A-Z]\w*)\b", re.MULTILINE)
_MARKDOWN_LINK_RE = re.compile(r"\[[^\]]+\]\([^)]+\)")
_MARKDOWN_HEADING_RE = re.compile(r"^\s*#{1,6}\s+\S+")


def _is_exported(name: str) -> bool:
    return bool(name) and name[0].isupper()


def _line_start_index(lines: list[str], line_no: int) -> int:
    if line_no <= 1:
        return 0
    return sum(len(line) + 1 for line in lines[: line_no - 1])


def _doc_comment_block(lines: list[str], decl_line: int) -> list[str]:
    i = decl_line - 2
    if i < 0 or not lines[i].lstrip().startswith("//"):
        return []

    block: list[str] = []
    while i >= 0 and lines[i].lstrip().startswith("//"):
        block.append(lines[i])
        i -= 1
    block.reverse()
    return block


def _comment_body(line: str) -> str:
    stripped = line.lstrip()
    if not stripped.startswith("//"):
        return ""
    return stripped[2:].strip()


def _first_doc_line(block: list[str]) -> str:
    for line in block:
        body = _comment_body(line)
        if body:
            return body
    return ""


def _camel_to_words(name: str) -> str:
    return re.sub(r"([a-z0-9])([A-Z])", r"\1 \2", name).replace("_", " ").strip().lower()


def _is_tautological_comment(name: str, comment: str) -> bool:
    text = " ".join(comment.split()).strip()
    if not text:
        return False

    lower = text.lower().rstrip(".")
    name_lower = name.lower()
    name_words = _camel_to_words(name)
    if lower in {
        f"{name_lower} returns {name_words}",
        f"{name_lower} returns the {name_words}",
    }:
        return True

    if name.startswith("Get"):
        target = _camel_to_words(name[3:]) or name_words
        if lower.startswith(f"{name_lower} returns ") and target and target in lower:
            return True

    return False


def _detect_gcm_001(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    lines = ctx.go_file.lines
    content = ctx.go_file.content
    for match in _TYPE_DECL_RE.finditer(content):
        name = match.group(1)
        if not _is_exported(name) or name.endswith("Config"):
            continue
        line_no, _ = index_to_line_col(content, match.start())
        if _doc_comment_block(lines, line_no):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                f"Exported type `{name}` is missing a doc comment.",
                suggestion=f"Add a doc comment above `{name}` describing purpose and behavior.",
            )
        )
    return findings


def _detect_gcm_002(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    lines = ctx.go_file.lines
    is_test_file = str(ctx.go_file.path).endswith("_test.go")
    for func in ctx.funcs:
        if func.is_method or not _is_exported(func.name):
            continue
        if is_test_file and func.name.startswith(("Test", "Benchmark", "Fuzz", "Example")):
            continue
        if _doc_comment_block(lines, func.line):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                func.start_idx,
                f"Exported function `{func.name}` is missing a doc comment.",
                suggestion="Add a Go doc comment summarizing behavior, side effects, and error semantics.",
            )
        )
    return findings


def _detect_gcm_003(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    lines = ctx.go_file.lines
    is_test_file = str(ctx.go_file.path).endswith("_test.go")
    for func in ctx.funcs:
        if not func.is_method or not _is_exported(func.name):
            continue
        if is_test_file and func.name.startswith(("Test", "Benchmark", "Fuzz", "Example")):
            continue
        if _doc_comment_block(lines, func.line):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                func.start_idx,
                f"Exported method `{func.name}` is missing a doc comment.",
                suggestion="Document method side effects, contracts, and failure behavior.",
            )
        )
    return findings


def _detect_gcm_004(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    lines = ctx.go_file.lines
    content = ctx.go_file.content
    for match in _TYPE_DECL_RE.finditer(content):
        name = match.group(1)
        if not _is_exported(name) or not name.endswith("Config"):
            continue
        line_no, _ = index_to_line_col(content, match.start())
        if _doc_comment_block(lines, line_no):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                f"Exported configuration type `{name}` is missing a doc comment.",
                suggestion="Describe configuration purpose and how values are provided or overridden.",
            )
        )
    return findings


def _detect_gcm_005(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    lines = ctx.go_file.lines
    content = ctx.go_file.content
    for match in _ERR_VAR_DECL_RE.finditer(content):
        name = match.group(1)
        line_no, _ = index_to_line_col(content, match.start())
        if _doc_comment_block(lines, line_no):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                f"Exported error variable `{name}` is missing a doc comment.",
                suggestion="Document when this error is returned and expected caller handling.",
            )
        )
    return findings


def _detect_gcm_006(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for line_no, line in enumerate(ctx.go_file.lines, start=1):
        body = _comment_body(line)
        if not body:
            continue
        if "**" in body or _MARKDOWN_LINK_RE.search(body) or _MARKDOWN_HEADING_RE.match(body):
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    _line_start_index(ctx.go_file.lines, line_no),
                    "Markdown formatting found in Go comment text.",
                    suggestion="Use plain Go comment prose without Markdown emphasis/links/headings.",
                    evidence=line.strip(),
                )
            )
    return findings


def _detect_gcm_007(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    max_len = ctx.max_line_length
    for line_no, line in enumerate(ctx.go_file.lines, start=1):
        if not line.lstrip().startswith("//"):
            continue
        if len(line.rstrip()) <= max_len:
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                _line_start_index(ctx.go_file.lines, line_no),
                f"Comment line exceeds {max_len} characters.",
                suggestion="Wrap long comments into multiple short lines.",
                evidence=line.strip(),
            )
        )
    return findings


def _detect_gcm_008(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    if not str(ctx.go_file.path).endswith("_test.go"):
        return []

    findings: list[Finding] = []
    for func in ctx.funcs:
        if not func.name.startswith("Example"):
            continue
        if "// Output:" in func.body or "// Unordered output:" in func.body:
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                func.start_idx,
                f"Example function `{func.name}` is missing an output comment.",
                suggestion="Add `// Output:` (or `// Unordered output:`) to make the example executable and verifiable.",
            )
        )
    return findings


def _detect_gcm_009(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    lines = ctx.go_file.lines
    is_test_file = str(ctx.go_file.path).endswith("_test.go")
    for func in ctx.funcs:
        if not _is_exported(func.name):
            continue
        if is_test_file and func.name.startswith(("Test", "Benchmark", "Fuzz", "Example")):
            continue
        block = _doc_comment_block(lines, func.line)
        if not block:
            continue
        comment_lines = [line for line in (_comment_body(raw) for raw in block) if line]
        tautological = next((line for line in comment_lines if _is_tautological_comment(func.name, line)), None)
        if tautological is None:
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                func.start_idx,
                f"Doc comment for `{func.name}` appears tautological and low-value.",
                suggestion="Explain behavior/contract rather than repeating the identifier meaning.",
                evidence=tautological,
            )
        )
    return findings


def build_comment_rules(guideline_titles: dict[int, str]) -> list[RuleDefinition]:
    """Build comment-focused rule catalog using COMMENTS.md section titles."""

    def title(number: int, fallback: str) -> str:
        return guideline_titles.get(number, fallback)

    catalog: list[tuple[RuleMeta, callable]] = [
        (RuleMeta("GCM001", 3, title(3, "Type Documentation"), "warning", "high"), _detect_gcm_001),
        (RuleMeta("GCM002", 4, title(4, "Function Documentation"), "warning", "medium"), _detect_gcm_002),
        (RuleMeta("GCM003", 5, title(5, "Method Documentation"), "warning", "medium"), _detect_gcm_003),
        (RuleMeta("GCM004", 6, title(6, "Configuration and Error Types"), "warning", "high"), _detect_gcm_004),
        (RuleMeta("GCM005", 6, title(6, "Configuration and Error Types"), "warning", "medium"), _detect_gcm_005),
        (RuleMeta("GCM006", 8, title(8, "Style Guidelines"), "warning", "medium"), _detect_gcm_006),
        (RuleMeta("GCM007", 8, title(8, "Style Guidelines"), "warning", "medium"), _detect_gcm_007),
        (RuleMeta("GCM008", 9, title(9, "Example Documentation Patterns"), "warning", "high"), _detect_gcm_008),
        (RuleMeta("GCM009", 11, title(11, "Documentation Anti-Patterns"), "info", "low"), _detect_gcm_009),
    ]
    return [RuleDefinition(meta=meta, detector=detector) for meta, detector in catalog]
