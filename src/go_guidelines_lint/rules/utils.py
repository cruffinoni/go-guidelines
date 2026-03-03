"""Utilities for lightweight Go-source analysis."""

from __future__ import annotations

from dataclasses import dataclass
import re


FUNC_START_RE = re.compile(r"^func\b", re.MULTILINE)


@dataclass(slots=True)
class FuncDecl:
    """A parsed top-level Go function declaration."""

    signature: str
    body: str
    start_idx: int
    end_idx: int
    line: int
    name: str
    params: str
    returns: str
    is_method: bool


def index_to_line_col(text: str, idx: int) -> tuple[int, int]:
    """Convert byte index in text to line/column coordinates (1-based)."""

    line = text.count("\n", 0, idx) + 1
    line_start = text.rfind("\n", 0, idx)
    if line_start == -1:
        col = idx + 1
    else:
        col = idx - line_start
    return line, col


def find_iter(pattern: re.Pattern[str], text: str) -> list[re.Match[str]]:
    """Materialize regex matches to simplify repeated checks."""

    return list(pattern.finditer(text))


def extract_go_functions(text: str) -> list[FuncDecl]:
    """Extract top-level function signatures and bodies with a brace scanner."""

    out: list[FuncDecl] = []
    for match in FUNC_START_RE.finditer(text):
        start = match.start()
        sig_end = text.find("{", start)
        if sig_end == -1:
            continue
        signature = text[start:sig_end].strip()
        if not signature:
            continue

        brace_depth = 0
        i = sig_end
        end = -1
        in_string = False
        string_quote = ""
        in_raw_string = False
        in_line_comment = False
        in_block_comment = False

        while i < len(text):
            ch = text[i]
            nxt = text[i + 1] if i + 1 < len(text) else ""

            if in_line_comment:
                if ch == "\n":
                    in_line_comment = False
                i += 1
                continue
            if in_block_comment:
                if ch == "*" and nxt == "/":
                    in_block_comment = False
                    i += 2
                    continue
                i += 1
                continue
            if in_raw_string:
                if ch == "`":
                    in_raw_string = False
                i += 1
                continue
            if in_string:
                if ch == "\\":
                    i += 2
                    continue
                if ch == string_quote:
                    in_string = False
                    string_quote = ""
                i += 1
                continue

            if ch == "/" and nxt == "/":
                in_line_comment = True
                i += 2
                continue
            if ch == "/" and nxt == "*":
                in_block_comment = True
                i += 2
                continue
            if ch in {'"', "'"}:
                in_string = True
                string_quote = ch
                i += 1
                continue
            if ch == "`":
                in_raw_string = True
                i += 1
                continue
            if ch == "{":
                brace_depth += 1
            elif ch == "}":
                brace_depth -= 1
                if brace_depth == 0:
                    end = i + 1
                    break
            i += 1

        if end == -1:
            continue

        body = text[sig_end + 1 : end - 1]
        line, _ = index_to_line_col(text, start)

        name = ""
        params = ""
        returns = ""
        is_method = False

        full_sig = " ".join(signature.split())
        sig_re = re.match(
            r"func\s*(\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\((.*?)\)\s*(.*)",
            full_sig,
            re.DOTALL,
        )
        if sig_re:
            receiver = sig_re.group(1) or ""
            name = sig_re.group(2)
            params = sig_re.group(3)
            returns = (sig_re.group(4) or "").strip()
            is_method = bool(receiver)

        out.append(
            FuncDecl(
                signature=signature,
                body=body,
                start_idx=start,
                end_idx=end,
                line=line,
                name=name,
                params=params,
                returns=returns,
                is_method=is_method,
            )
        )
    return out


def find_line_with_substring(lines: list[str], needle: str, fallback: int = 1) -> int:
    """Find first line containing a substring."""

    for idx, line in enumerate(lines, start=1):
        if needle in line:
            return idx
    return fallback
