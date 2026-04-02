"""Rule detector implementations mapped to guideline sections."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import re

from go_guidelines_lint.models import Finding, GoFile, RuleMeta
from go_guidelines_lint.rules.finding_factory import make_finding
from go_guidelines_lint.rules.utils import FuncDecl, extract_go_functions


@dataclass(slots=True)
class FileContext:
    """Prepared data reused by multiple detectors."""

    go_file: GoFile
    funcs: list[FuncDecl]
    interfaces: set[str]
    max_line_length: int
    is_test_file: bool


@dataclass(slots=True)
class RuleDefinition:
    """Definition of one detector and associated metadata."""

    meta: RuleMeta
    detector: callable
    applies_to_test_files: bool = False


_DOT_IMPORT_RE = re.compile(r'^\s*import\s+\.\s+"[^"]+"', re.MULTILINE)
_DOT_IMPORT_BLOCK_RE = re.compile(r'^\s*\.\s+"[^"]+"', re.MULTILINE)
_IMPORT_BLOCK_SPAN_RE = re.compile(r'\bimport\s*\(([^)]*)\)', re.DOTALL)
_LINE_COMMENT_STRIP_RE = re.compile(r'//[^\n]*')
_PACKAGE_RE = re.compile(r'^\s*package\s+([A-Za-z_]\w*)\s*$', re.MULTILINE)
_PANIC_ERR_RE = re.compile(r'panic\(\s*err\s*\)')
_FMT_ERRORF_ERR_RE = re.compile(r'fmt\.Errorf\(\s*"([^"]+)"\s*,\s*err\s*\)')
_FMT_ERRORF_RE = re.compile(r'fmt\.Errorf\(\s*"([^"]+)"')
_STRUCT_BLOCK_RE = re.compile(r'type\s+([A-Za-z_]\w*)\s+struct\s*\{(.*?)\}', re.DOTALL)
_INTERFACE_BLOCK_RE = re.compile(r'type\s+([A-Za-z_]\w*)\s+interface\s*\{(.*?)\}', re.DOTALL)
_GO_STMT_RE = re.compile(r'^\s*go\s+.+$', re.MULTILINE)
_HTTP_GET_RE = re.compile(r'\bhttp\.Get\(')
_NEW_REQUEST_RE = re.compile(r'\bhttp\.NewRequest\(')
_LOG_PRINTF_RE = re.compile(r'\blog\.(?:Print|Printf|Println|Fatal|Fatalf|Fatalln|Panic|Panicf|Panicln)\(')
_AFTER_FUNC_RE = re.compile(r'\btime\.AfterFunc\(')
_NEW_TICKER_ASSIGN_RE = re.compile(r'\b([A-Za-z_]\w*)\s*:=\s*time\.NewTicker\(')
_NEW_TIMER_ASSIGN_RE = re.compile(r'\b([A-Za-z_]\w*)\s*:=\s*time\.NewTimer\(')
_FUNC_INIT_RE = re.compile(r'^\s*func\s+init\s*\(\s*\)', re.MULTILINE)
_VAR_SIMPLE_RE = re.compile(r'^\s*var\s+[A-Za-z_]\w*\s+[^=(\n]+$', re.MULTILINE)
_FREE_FUNC_PTR_PARAM_RE = re.compile(
    r'func\s+([A-Za-z_]\w*)\s*\(\s*([A-Za-z_]\w*)\s+\*([A-Z][A-Za-z0-9_]*)\s*,',
    re.MULTILINE,
)
_NIL_CHECK_RE = re.compile(r'if\s+(?:([A-Za-z_]\w*)\s*==\s*nil|nil\s*==\s*([A-Za-z_]\w*))')
_SKIPPED_NIL_IDENTS = {"err", "ok"}
_FUNC_SIGNATURE_RE = re.compile(r"func\s*(\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\((.*?)\)\s*(.*)", re.DOTALL)
_DIRECT_CALL_RE = re.compile(r"(?<!\.)\b([A-Za-z_]\w*)\s*\(")
_CHANNEL_RANGE_RE = re.compile(r'\bfor\b.+\brange\b.+<-|<-.+\brange\b')
_DEFER_CLOSE_RE = re.compile(r'defer\s+(\w+)\.Close\(\)')
_ASSIGN_ERR_RE = re.compile(r'\b(\w+)\s*,\s*err\s*:=')
_ERR_CHECK_RE = re.compile(r'if\s+err\s*!=\s*nil')
_HTTP_CALL_RE = re.compile(
    r'\bhttp\.(?:Get|Post|Head|Do|PostForm|NewRequest|NewRequestWithContext)\('
)
_IO_READALL_NOERR_RE = re.compile(r'_\s*,\s*_\s*:=\s*io\.ReadAll\(')
_PANIC_RE = re.compile(r'\bpanic\((?!\s*err\s*\))')


def _split_params(params: str) -> list[str]:
    parts: list[str] = []
    current = []
    depth = 0
    for ch in params:
        if ch == ',' and depth == 0:
            part = ''.join(current).strip()
            if part:
                parts.append(part)
            current = []
            continue
        if ch in '([{':
            depth += 1
        elif ch in ')]}':
            depth = max(depth - 1, 0)
        current.append(ch)
    tail = ''.join(current).strip()
    if tail:
        parts.append(tail)
    return parts


def _to_multiline_signature(signature: str) -> str | None:
    compact = " ".join(signature.split())
    sig_match = _FUNC_SIGNATURE_RE.match(compact)
    if not sig_match:
        return None

    receiver = (sig_match.group(1) or "").strip()
    fn_name = sig_match.group(2)
    params = _split_params(sig_match.group(3).strip())
    returns = (sig_match.group(4) or "").strip()

    receiver_prefix = f"{receiver} " if receiver else ""
    if not params:
        header = f"func {receiver_prefix}{fn_name}()"
        return f"{header} {returns}".rstrip() if returns else header

    lines = [f"func {receiver_prefix}{fn_name}("]
    for param in params:
        lines.append(f"\t{param},")
    closing = f") {returns}".rstrip() if returns else ")"
    lines.append(closing)
    return "\n".join(lines)


def _strip_comments_and_strings(body: str) -> str:
    body = re.sub(r'//[^\n]*', '', body)
    body = re.sub(r'"[^"\n]*"', '""', body)
    body = re.sub(r'`[^`]*`', '``', body)
    return body


def _detect_rule_1(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    content = ctx.go_file.content
    stripped = _LINE_COMMENT_STRIP_RE.sub(lambda m: ' ' * len(m.group(0)), content)

    block_spans: list[tuple[int, int]] = [
        (m.start(), m.end()) for m in _IMPORT_BLOCK_SPAN_RE.finditer(stripped)
    ]

    def inside_block(pos: int) -> bool:
        return any(start <= pos < end for start, end in block_spans)

    for match in _DOT_IMPORT_BLOCK_RE.finditer(stripped):
        if not inside_block(match.start()):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Dot-import found in import block.",
                suggestion="Use named imports instead.",
                evidence=content[match.start():match.end()].strip(),
            )
        )

    for match in _DOT_IMPORT_RE.finditer(stripped):
        if inside_block(match.start()):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Avoid `import .`; it pollutes namespace and harms readability.",
                suggestion="Use explicit package-qualified symbols.",
                evidence=content[match.start():match.end()].strip(),
            )
        )

    return findings


def _detect_rule_2(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    content = ctx.go_file.content
    package_match = _PACKAGE_RE.search(content)
    if not package_match:
        return findings

    package_name = package_match.group(1)
    if package_name in {"util", "utils", "common", "helper", "helpers"}:
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                package_match.start(1),
                f"Vague package name `{package_name}` detected.",
                suggestion="Use a package name describing its domain role.",
            )
        )

    package_line_idx = package_match.start()
    prefix = content[:package_line_idx].splitlines()
    previous_non_empty = ""
    for line in reversed(prefix):
        stripped = line.strip()
        if stripped:
            previous_non_empty = stripped
            break

    expected_prefix = f"// Package {package_name}"
    if not previous_non_empty.startswith(expected_prefix):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                package_match.start(),
                "Package declaration is missing a package-level doc comment.",
                suggestion=f"Add a comment like `{expected_prefix} ...` above `package`.",
            )
        )

    return findings


def _detect_rule_3(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for match in _PANIC_ERR_RE.finditer(ctx.go_file.content):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "`panic(err)` used in normal flow; prefer returning wrapped errors.",
                suggestion="Return `fmt.Errorf(""operation: %w"", err)` instead of panic.",
                evidence=match.group(0),
            )
        )

    for match in _FMT_ERRORF_ERR_RE.finditer(ctx.go_file.content):
        fmt_str = match.group(1)
        if "%w" not in fmt_str:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    match.start(),
                    "Error wrapping with `fmt.Errorf` should use `%w` when propagating `err`.",
                    suggestion="Use `fmt.Errorf(""context: %w"", err)`.",
                    evidence=match.group(0),
                )
            )

    for match in _FMT_ERRORF_RE.finditer(ctx.go_file.content):
        fmt_str = match.group(1)
        stripped = fmt_str.strip()
        if not stripped:
            continue
        if stripped[0].isupper() or stripped.endswith((".", "!", "?")):
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    match.start(),
                    "Error messages should be lowercase and without terminal punctuation.",
                    suggestion="Use lowercase short context, e.g. `open config: %w`.",
                    evidence=fmt_str,
                )
            )
    return findings


def _detect_rule_4(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for struct_match in _STRUCT_BLOCK_RE.finditer(ctx.go_file.content):
        body = struct_match.group(2)
        if "context.Context" in body:
            idx = struct_match.start(2) + body.find("context.Context")
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    idx,
                    "Context is stored in a struct field; pass context as first parameter instead.",
                )
            )

    for func in ctx.funcs:
        if "context.Context" not in func.params:
            continue
        params = _split_params(func.params)
        if params and "context.Context" not in params[0]:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx,
                    f"Function `{func.name}` has context parameter not in first position.",
                    suggestion="Place `ctx context.Context` as the first function argument.",
                )
            )
    return findings


def _detect_rule_5(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    text = ctx.go_file.content
    if "WaitGroup" in text and "errgroup.WithContext" not in text and _GO_STMT_RE.search(text):
        idx = text.find("WaitGroup")
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                idx,
                "`sync.WaitGroup` used without structured error propagation.",
                suggestion="Consider `errgroup.WithContext` for cancellation and error handling.",
            )
        )

    for match in _GO_STMT_RE.finditer(text):
        line = match.group(0)
        if "g.Go(" in line:
            continue
        if line.strip().startswith("//"):
            continue
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Potential fire-and-forget goroutine detected.",
                suggestion="Use `errgroup` or explicitly document lifecycle/error handling.",
                evidence=line.strip(),
            )
        )
    return findings


def _detect_rule_6(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    text = ctx.go_file.content

    for match in _HTTP_GET_RE.finditer(text):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Use `http.NewRequestWithContext` + shared client instead of `http.Get`.",
                suggestion="Create a request with context and execute it with a reusable `http.Client`.",
                evidence=match.group(0),
            )
        )

    for match in _NEW_REQUEST_RE.finditer(text):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "`http.NewRequest` used without context.",
                suggestion="Prefer `http.NewRequestWithContext(ctx, ...)`.",
                evidence=match.group(0),
            )
        )

    funcs_with_http = [f for f in ctx.funcs if _HTTP_CALL_RE.search(f.body)]
    for func in funcs_with_http:
        if ".Body.Close()" not in func.body:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx,
                    "HTTP response body may not be closed.",
                    suggestion="Call `defer resp.Body.Close()` after successful request.",
                )
            )

    if not funcs_with_http and (m := _HTTP_CALL_RE.search(text)) and ".Body.Close()" not in text:
        idx = m.start()
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                idx,
                "HTTP response body may not be closed.",
                suggestion="Call `defer resp.Body.Close()` after successful request.",
            )
        )

    for match in _IO_READALL_NOERR_RE.finditer(text):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "`io.ReadAll` result appears to ignore both value and error.",
                suggestion="Handle read errors explicitly.",
            )
        )

    return findings


def _detect_rule_7(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    text = ctx.go_file.content

    for match in _INTERFACE_BLOCK_RE.finditer(text):
        iface_name = match.group(1)
        body = match.group(2)
        method_count = 0
        for line in body.splitlines():
            stripped = line.strip()
            if not stripped or stripped.startswith("//"):
                continue
            if "(" in stripped and ")" in stripped:
                method_count += 1
        if method_count >= 10:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    match.start(1),
                    f"Interface `{iface_name}` has {method_count} methods and may be too broad.",
                    suggestion="Prefer smaller consumer-defined interfaces.",
                )
            )

    for func in ctx.funcs:
        if not func.name.startswith("New"):
            continue
        ret = func.returns.strip()
        if not ret or ret.startswith("*"):
            continue
        return_tokens = [token.strip("()*") for token in re.split(r"[,\s]+", ret) if token.strip()]
        if any(token in ctx.interfaces for token in return_tokens):
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx,
                    f"Constructor `{func.name}` returns an interface type.",
                    suggestion="Return a concrete implementation type from constructors.",
                    evidence=func.signature.replace("\n", " "),
                )
            )

    return findings


def _detect_rule_8(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []

    for func in ctx.funcs:
        body = func.body
        for defer_match in _DEFER_CLOSE_RE.finditer(body):
            var_name = defer_match.group(1)
            defer_pos = defer_match.start()
            assign_match = None
            for m in _ASSIGN_ERR_RE.finditer(body):
                if m.group(1) == var_name and m.start() < defer_pos:
                    assign_match = m
            if assign_match is None:
                continue
            err_check_between = _ERR_CHECK_RE.search(body, assign_match.end(), defer_pos)
            if err_check_between is not None:
                continue
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx + defer_match.start(),
                    "`defer Close()` occurs before checking acquisition error.",
                    suggestion="Check `err` first, then `defer` resource close.",
                    evidence=defer_match.group(0),
                )
            )

    for match in _PANIC_RE.finditer(ctx.go_file.content):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "`panic` found; reserve panic/recover for process boundaries.",
            )
        )
    return findings


def _detect_rule_9(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    empty_slice_re = re.compile(r':=\s*\[\][^{\n]+\{\s*\}')
    for match in empty_slice_re.finditer(ctx.go_file.content):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Empty slice literal initialization found.",
                suggestion="Prefer `var s []T` or `make([]T, 0, n)` when capacity is known.",
                evidence=match.group(0),
            )
        )

    builder_copy_re = re.compile(r'var\s+\w+\s+strings\.Builder\s*=\s*\*?\w+')
    for match in builder_copy_re.finditer(ctx.go_file.content):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Potential copy of initialized `strings.Builder` value.",
                suggestion="Do not copy non-zero `strings.Builder` instances.",
                evidence=match.group(0),
            )
        )
    return findings


def _detect_rule_10(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    line_offset = 0
    for line in ctx.go_file.lines:
        if "json:\"" in line and "[]" in line:
            tag_match = re.search(r'json:"([^"]+)"', line)
            if not tag_match:
                line_offset += len(line) + 1
                continue
            tag = tag_match.group(1)
            if tag == "-" or "omitempty" in tag:
                line_offset += len(line) + 1
                continue
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    line_offset,
                    "Slice JSON field tag missing `omitempty`.",
                    suggestion="Add `omitempty` or initialize to empty slices when encoding must be `[]`.",
                    evidence=line.strip(),
                )
            )
        line_offset += len(line) + 1
    return findings


def _detect_rule_11(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for match in _LOG_PRINTF_RE.finditer(ctx.go_file.content):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Unstructured logger print call found.",
                suggestion="Use `slog` structured logging with key-value fields.",
                evidence=match.group(0),
            )
        )
    return findings


def _detect_rule_13(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    text = ctx.go_file.content

    for match in _AFTER_FUNC_RE.finditer(text):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "`time.AfterFunc` can hide timer lifecycle leaks.",
                suggestion="Prefer explicit contexts/timers and always stop tickers/timers.",
                evidence=match.group(0),
            )
        )

    for match in _NEW_TICKER_ASSIGN_RE.finditer(text):
        var_name = match.group(1)
        if f"{var_name}.Stop()" not in text:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    match.start(1),
                    f"Ticker `{var_name}` is created but `Stop()` was not found.",
                    suggestion="Call `defer ticker.Stop()` after ticker creation.",
                )
            )

    for match in _NEW_TIMER_ASSIGN_RE.finditer(text):
        var_name = match.group(1)
        if f"{var_name}.Stop()" not in text:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    match.start(1),
                    f"Timer `{var_name}` is created but `Stop()` was not found.",
                    suggestion="Call `defer timer.Stop()` when appropriate.",
                )
            )

    return findings


def _detect_rule_14(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for func in ctx.funcs:
        body = func.body
        if _CHANNEL_RANGE_RE.search(body) and "ctx.Done()" not in body:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx,
                    f"Pipeline-like loop in `{func.name}` lacks cancellation handling.",
                    suggestion="Use `select` with `ctx.Done()` when sending/receiving on channels.",
                )
            )
        if "chan" in func.params and "<-" in body and "close(" in body:
            continue
        if "chan" in func.params and "<-" in body and "close(" not in body:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx,
                    f"Function `{func.name}` writes to channels without visible close semantics.",
                    suggestion="Ensure pipeline stages close output channels when done.",
                )
            )
    return findings


def _detect_rule_15(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    if not str(ctx.go_file.path).endswith("_test.go"):
        return []

    findings: list[Finding] = []
    text = ctx.go_file.content

    if "time.Sleep(" in text:
        idx = text.find("time.Sleep(")
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                idx,
                "Tests using `time.Sleep` can become flaky and non-deterministic.",
                suggestion="Use synchronization primitives or polling with deadlines instead.",
            )
        )

    map_vars = set(re.findall(r'([A-Za-z_]\w*)\s*:=\s*map\[', text))
    for var in map_vars:
        loop_re = re.compile(rf'for\s+.*:=\s*range\s+{re.escape(var)}\b')
        for match in loop_re.finditer(text):
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    match.start(),
                    f"Test iterates over map `{var}`; iteration order is non-deterministic.",
                    suggestion="Use slices with fixed ordering for deterministic tests.",
                )
            )
    return findings


def _detect_rule_16(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for match in _FUNC_INIT_RE.finditer(ctx.go_file.content):
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                "Avoid `init()` except trivial deterministic registration.",
                suggestion="Prefer explicit setup functions/constructors.",
                evidence=match.group(0),
            )
        )
    return findings


def _detect_rule_17(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    lines = ctx.go_file.lines
    run_start = None
    run_len = 0

    for idx, line in enumerate(lines, start=1):
        stripped = line.strip()
        if stripped.startswith("var ("):
            run_start = None
            run_len = 0
            continue

        if _VAR_SIMPLE_RE.match(line):
            if run_start is None:
                run_start = idx
            run_len += 1
        else:
            if run_len >= 3 and run_start is not None:
                findings.append(
                    Finding(
                        rule_id=meta.rule_id,
                        guideline_number=meta.guideline_number,
                        title=meta.title,
                        severity=meta.severity,
                        confidence=meta.confidence,
                        message=f"{run_len} consecutive `var` declarations can be grouped in a block.",
                        file=str(ctx.go_file.path),
                        line=run_start,
                        column=1,
                        suggestion="Use `var (...)` to group related declarations.",
                    )
                )
            run_start = None
            run_len = 0

    if run_len >= 3 and run_start is not None:
        findings.append(
            Finding(
                rule_id=meta.rule_id,
                guideline_number=meta.guideline_number,
                title=meta.title,
                severity=meta.severity,
                confidence=meta.confidence,
                message=f"{run_len} consecutive `var` declarations can be grouped in a block.",
                file=str(ctx.go_file.path),
                line=run_start,
                column=1,
                suggestion="Use `var (...)` to group related declarations.",
            )
        )

    return findings


def _detect_rule_18(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for match in _FREE_FUNC_PTR_PARAM_RE.finditer(ctx.go_file.content):
        fn_name = match.group(1)
        type_name = match.group(3)
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                match.start(),
                f"Function `{fn_name}` takes `*{type_name}` as first parameter and may be a better fit as a method.",
                suggestion=f"Convert to method: `func (x *{type_name}) {fn_name}(...)`.",
                evidence=match.group(0),
            )
        )
    return findings


def _detect_rule_19(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    for func in ctx.funcs:
        if not func.name.startswith("New"):
            continue
        for match in _NIL_CHECK_RE.finditer(func.body):
            ident = match.group(1) or match.group(2) or ""
            if ident in _SKIPPED_NIL_IDENTS or ident.endswith("Err"):
                continue
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx + match.start(),
                    f"Constructor `{func.name}` performs nil-check guarding required dependencies.",
                    suggestion="Treat required dependencies as constructor contract and fail fast in wiring/tests.",
                    evidence=match.group(0),
                )
            )
    return findings


def _detect_rule_20(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    max_len = ctx.max_line_length
    for func in ctx.funcs:
        sig = func.signature
        compact = " ".join(sig.split())

        if "\n" in sig and len(compact) > max_len:
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx,
                    f"Multiline function signature compact form still exceeds {max_len} characters.",
                    suggestion="Reduce parameter count, introduce type aliases, or split into multiple functions.",
                    evidence=compact,
                )
            )
            continue

        if "\n" not in sig and len(compact) > max_len:
            suggestion = "Break signature into multiple lines with one parameter per line."
            multiline = _to_multiline_signature(sig)
            if multiline:
                suggestion = f"Use multiline signature:\n{multiline}"
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    func.start_idx,
                    f"Single-line function signature exceeds {max_len} characters.",
                    suggestion=suggestion,
                    evidence=compact,
                )
            )
    return findings


def _strongly_connected_components(node_count: int, edges: dict[int, set[int]]) -> list[list[int]]:
    """Compute SCCs with Tarjan algorithm."""

    index = 0
    stack: list[int] = []
    on_stack: set[int] = set()
    indices = [-1] * node_count
    lowlinks = [0] * node_count
    components: list[list[int]] = []

    def dfs(v: int) -> None:
        nonlocal index
        indices[v] = index
        lowlinks[v] = index
        index += 1
        stack.append(v)
        on_stack.add(v)

        for w in edges.get(v, set()):
            if indices[w] == -1:
                dfs(w)
                lowlinks[v] = min(lowlinks[v], lowlinks[w])
            elif w in on_stack:
                lowlinks[v] = min(lowlinks[v], indices[w])

        if lowlinks[v] != indices[v]:
            return

        component: list[int] = []
        while stack:
            w = stack.pop()
            on_stack.remove(w)
            component.append(w)
            if w == v:
                break
        components.append(component)

    for node in range(node_count):
        if indices[node] == -1:
            dfs(node)
    return components


def _detect_rule_21(ctx: FileContext, meta: RuleMeta) -> list[Finding]:
    findings: list[Finding] = []
    if not ctx.funcs:
        return findings

    declarations = [func for func in ctx.funcs if func.name]
    if len(declarations) < 2:
        return findings

    # Keep only unique declaration names for deterministic local-call resolution.
    by_name: dict[str, list[int]] = {}
    for idx, func in enumerate(declarations):
        by_name.setdefault(func.name, []).append(idx)
    unique_name_to_idx = {name: indices[0] for name, indices in by_name.items() if len(indices) == 1}

    edges: dict[int, set[int]] = {}
    for caller_idx, func in enumerate(declarations):
        callees: set[int] = set()
        stripped_body = _strip_comments_and_strings(func.body)
        for match in _DIRECT_CALL_RE.finditer(stripped_body):
            callee_name = match.group(1)
            callee_idx = unique_name_to_idx.get(callee_name)
            if callee_idx is None:
                continue
            callees.add(callee_idx)
        if callees:
            edges[caller_idx] = callees

    components = _strongly_connected_components(len(declarations), edges)
    component_by_node: dict[int, int] = {}
    for component_idx, members in enumerate(components):
        for member in members:
            component_by_node[member] = component_idx

    # Enforce contiguous declaration blocks for mutually recursive groups.
    for members in components:
        if len(members) <= 1:
            continue
        ordered = sorted(members)
        if ordered[-1] - ordered[0] + 1 == len(ordered):
            continue
        names = ", ".join(declarations[index].name for index in ordered)
        findings.append(
            make_finding(
                meta,
                ctx.go_file,
                declarations[ordered[0]].start_idx,
                f"Mutually recursive functions must be declared contiguously: {names}.",
                suggestion="Place mutually recursive functions in one adjacent declaration block (any internal order).",
            )
        )

    # Enforce declaration-before-use for non-mutual local calls.
    for caller_idx, callee_indices in edges.items():
        caller = declarations[caller_idx]
        for callee_idx in sorted(callee_indices):
            if callee_idx <= caller_idx:
                continue
            caller_component = component_by_node.get(caller_idx)
            callee_component = component_by_node.get(callee_idx)
            if caller_component is not None and caller_component == callee_component:
                # Mutual recursion group can be in any internal order.
                continue
            callee = declarations[callee_idx]
            findings.append(
                make_finding(
                    meta,
                    ctx.go_file,
                    caller.start_idx,
                    f"Function `{caller.name}` calls local function `{callee.name}` before its declaration.",
                    suggestion=f"Declare `{callee.name}` before `{caller.name}` for top-down readability.",
                )
            )

    return findings


def build_rules(guideline_titles: dict[int, str]) -> list[RuleDefinition]:
    """Build the rule catalog with titles sourced from parsed guidelines."""

    def title(n: int, fallback: str) -> str:
        return guideline_titles.get(n, fallback)

    catalog: list[tuple[RuleMeta, callable, bool]] = [
        (RuleMeta("GBP001", 1, title(1, "Imports and Formatting"), "error", "high"), _detect_rule_1, False),
        (RuleMeta("GBP002", 2, title(2, "Package Design and Documentation"), "warning", "medium"), _detect_rule_2, False),
        (RuleMeta("GBP003", 3, title(3, "Errors"), "error", "high"), _detect_rule_3, False),
        (RuleMeta("GBP004", 4, title(4, "Context Usage"), "warning", "high"), _detect_rule_4, False),
        (RuleMeta("GBP005", 5, title(5, "Concurrency"), "warning", "low"), _detect_rule_5, False),
        (RuleMeta("GBP006", 6, title(6, "HTTP and I/O"), "warning", "high"), _detect_rule_6, False),
        (RuleMeta("GBP007", 7, title(7, "Interfaces"), "warning", "medium"), _detect_rule_7, False),
        (RuleMeta("GBP008", 8, title(8, "Defer, Panic, and Recover"), "warning", "high"), _detect_rule_8, False),
        (RuleMeta("GBP009", 9, title(9, "Slices, Maps, and Strings"), "warning", "medium"), _detect_rule_9, False),
        (RuleMeta("GBP010", 10, title(10, "JSON and Zero Values"), "warning", "medium"), _detect_rule_10, False),
        (RuleMeta("GBP011", 11, title(11, "Logging"), "warning", "high"), _detect_rule_11, False),
        (RuleMeta("GBP013", 13, title(13, "Timeouts, Tickers, and Timers"), "warning", "high"), _detect_rule_13, False),
        (RuleMeta("GBP014", 14, title(14, "Pipelines and Cancellation"), "warning", "low"), _detect_rule_14, False),
        (RuleMeta("GBP015", 15, title(15, "Testing"), "warning", "medium"), _detect_rule_15, True),
        (RuleMeta("GBP016", 16, title(16, "Usage of `init()`"), "warning", "high"), _detect_rule_16, False),
        (RuleMeta("GBP017", 17, title(17, "Variable Declarations"), "warning", "high"), _detect_rule_17, False),
        (RuleMeta("GBP018", 18, title(18, "Method Organization"), "info", "medium"), _detect_rule_18, False),
        (RuleMeta("GBP019", 19, title(19, "Required Dependencies and Nil Checks"), "warning", "medium"), _detect_rule_19, False),
        (RuleMeta("GBP020", 20, title(20, "Function Signature Line Length"), "warning", "high"), _detect_rule_20, False),
        (RuleMeta("GBP021", 21, title(21, "Function Declaration Order"), "warning", "high"), _detect_rule_21, False),
    ]
    return [
        RuleDefinition(meta=meta, detector=detector, applies_to_test_files=applies_to_test_files)
        for meta, detector, applies_to_test_files in catalog
    ]


def analyze_file(path: Path, max_line_length: int) -> FileContext:
    """Read one Go file and create reusable analysis data."""

    content = path.read_text(encoding="utf-8", errors="replace")
    go_file = GoFile(path=path, content=content)
    funcs = extract_go_functions(content)
    interfaces = {match.group(1) for match in _INTERFACE_BLOCK_RE.finditer(content)}
    return FileContext(
        go_file=go_file,
        funcs=funcs,
        interfaces=interfaces,
        max_line_length=max_line_length,
        is_test_file=path.name.endswith("_test.go"),
    )


def run_file_rules(ctx: FileContext, rules: list[RuleDefinition]) -> list[Finding]:
    """Execute selected detectors for one file context."""

    findings: list[Finding] = []
    for rule in rules:
        if ctx.is_test_file and not rule.applies_to_test_files:
            continue
        findings.extend(rule.detector(ctx, rule.meta))
    return findings
