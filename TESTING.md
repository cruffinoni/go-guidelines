# Testing Guidelines

This document describes what tests in gg-lint are for, what they should assert, and where they belong. It is a design guideline, not a coverage target.

## 1. Purpose

Tests protect **behavior, contracts, and regressions**. They are not a transcript of the implementation, and they are not a change log.

A test earns its place when it:

- asserts an observable behavior or an external contract;
- has assertions that can fail for a meaningful reason;
- would fail on a realistic regression;
- survives a reasonable refactor;
- covers an edge case, error path, or boundary where the code is likely to be wrong.

Coverage is a diagnostic. A passing test that would not catch a realistic bug is still low-value.

## 2. The central question

For every assertion, ask:

> If someone changes this without changing intended behavior, should the test fail?

- **No** → the assertion is about presentation, incidental structure, or an implementation detail. Assert the invariant instead.
- **Yes** → the assertion is about a contract. Make the contract explicit and test it at the right level.

A test that fails on a cosmetic change is not necessarily wrong; it is enforcing a representation. The question is whether that representation was ever promised.

## 3. Test layers

Prefer the lowest layer that can express the behavior.

### Unit tests

Pure functions, calculations, parsers, state machines, validators.

- Use table-driven tests for boundaries and error cases.
- Assert values and invariants, not rendered strings.
- Include negative cases; a test that only proves the happy path is incomplete.

### Boundary and integration tests

Filesystem, Git, HTTP, package loading, rule engine, file writers.

- Use real temporary directories, `httptest`, and real Git repositories where practical.
- Prefer a real implementation over a mock when the real implementation is fast and deterministic.
- Assert outcomes: files written, findings returned, errors raised, state preserved.
- Keep these fewer and more deliberate than unit tests.

### Contract tests

JSON schema, configuration keys, persisted markers, release asset names, stable identifiers.

- Assert the consumer's view, not the producer's internal type.
- Exact values are justified here because changing them breaks something outside the test.

### End-to-end and process tests

Built binary, startup, exit codes, release artifacts.

- These belong at the edge: CI build/run, release workflow, or a small number of command tests.
- Do not duplicate the whole unit suite in a shell script or a binary harness.
- A process-level test is justified only when process behavior is the feature: exit codes, signals, startup, packaging.

## 4. Rules

### Test behavior, not implementation

- Do not assert private call counts, internal field layout, or the order of unrelated operations.
- Do not test trivial getters, constructors, or framework behavior.
- Do not assert that a mock was called when the outcome can be observed directly.
- It is acceptable to inspect unexported state when that state *is* the contract of the package.

### Assert the invariant, not the rendering

Prefer:

- "the filled portion is proportional and clamped";
- "the output fits within the terminal width";
- "the file contains exactly one managed block";
- "the error is `ErrChecksumMismatch`";
- "findings are sorted by file, line, column, rule ID, and message".

over whole-string equality, unless the string itself is the contract.

### Strings are contracts only when a consumer depends on them

Exact-string assertions are appropriate for:

- persisted markers written into user files;
- machine-readable JSON keys and schema versions;
- configuration keys and enum values;
- stable rule IDs;
- release asset names and archive layout;
- documented command output that scripts parse;
- protocol messages and serialized formats.

They are not appropriate for:

- progress-bar glyphs, spacing, and fill characters;
- help prose;
- version-output wording;
- log or progress labels;
- error messages;
- table alignment and Markdown formatting, unless the format is a documented product.

If a string is not a contract, test the behavior underneath it. If the behavior has no other observable form, extract the calculation or keep one golden/snapshot test and accept that it is a characterization test.

### Exact strings still need the right shape

Whole-string equality conflates style and logic. When a string *is* contractual:

- decode it into a consumer-side struct or map and assert fields;
- compare a structured value rather than a formatted blob;
- keep one golden only for the overall shape that cannot be expressed structurally.

Do not round-trip a producer type through its own encoder and call that schema coverage. That tests codec symmetry, not the wire format: a consistent rename of both marshal and unmarshal tags still passes.

### Errors are identity and context, not text

Do not assert `strings.Contains(err.Error(), ...)` when the error has a distinguishable identity.

Prefer:

- `errors.Is(err, ErrSentinel)`;
- `errors.AsType[*TypedError](err)` for typed errors—this is the project convention (GBP026);
- fields on a typed error: path, asset, rule ID, status code, line.

Error text is for humans. It may change without breaking code. Test the error, not the sentence.

If the CLI's human-facing error text is a product requirement, pin it once in an end-to-end or golden test. Do not pin it in every unit test.

### Presentation code has logic; test that logic

A formatter is not "just strings". Its real behavior includes:

- clamping and rounding;
- width calculation and truncation;
- state transitions (start, advance, finish, abort, reset);
- idempotence and byte preservation;
- terminal clearing and padding;
- concurrency and shutdown.

Test those. Do not test the fill character.

### Fixtures are inputs, not tests

Files under `testdata/` are inputs. A `_test.go` file inside `testdata/` is not executed by `go test`.

Rule fixtures use `// want` annotations:

- the annotation must be on the same line as the expected finding;
- the message is a regexp fragment, not a full snapshot;
- every finding must match an expectation;
- every expectation must be matched exactly once;
- negative cases belong in the same fixture.

A rule fixture without a negative case is usually incomplete.

### Determinism and isolation

- No real sleeps for synchronization.
- No dependency on the network, clock, or developer environment.
- Use `t.TempDir()`, `t.Setenv()`, `t.Chdir()`, and `httptest`.
- Set Git identity locally in tests that create repositories.
- Sort or otherwise normalize output before comparison.
- Skip platform-specific tests explicitly; do not weaken assertions with string fallbacks.

### One assertion family per test

A test should make one behavior obvious. Table-driven tests are preferred over copy-pasted tests. Failure messages should identify the input and the expected behavior, not just say "not equal".

## 5. What is a contract in this project?

A contract can be an input or an output. Test the direction the product actually exposes:

- **Input contract** (config file, CLI flags, fixture annotations): provide a literal example of what a user writes and assert the parsed result. Do not marshal the parsed struct and read it back; that tests codec symmetry, not the accepted input.
- **Output contract** (JSON report, generated Markdown, persisted markers): decode the produced output through a consumer-side shape and assert the fields consumers rely on. Do not round-trip the producer type; a consistent rename in both directions still passes.

The same object can be both—for example, if a command later writes a config file—but only the directions the product actually exposes are contracts.

| Representation | Contract? | How to test |
|---|---|---|
| Harness markers (`<!-- gg-lint:claude:start -->`) | Yes, persisted in user files | Exact marker; also assert the instructions inside the block |
| JSON output (`rule_id`, `schema_version`, `findings`) | Yes, machine-readable | Consumer-side struct or map, not producer round-trip |
| Config keys and values (`fail_on`, `text`/`json`) | Yes, input files users write | Literal YAML fixture and decoded values; no producer round-trip |
| Rule IDs (`GBP001`) | Yes, once used in config or tooling | Stable ID set; rename requires a deliberate migration |
| Release asset names, archive layout, `checksums.txt` | Yes, installers and updater depend on them | Exact names and formats |
| Exit codes and CLI enum values | Yes | Numeric/exit-code assertions |
| Text report layout | Partly: rule ID and relative location are consumed; spacing is not | Assert rule ID and location; one golden only if the full format is promised |
| `version` output | No, unless a script parses it | Labels and values; not the exact sentence |
| Progress bar, progress labels, help prose | No | Invariants and state, not glyphs or wording |

## 6. Applying this to existing tests

When reviewing a test, ask:

1. What behavior or contract does this protect?
2. Would a realistic regression make it fail?
3. Would a reasonable refactor make it fail? If yes, is that refactor a behavior change or an implementation change?
4. Is the assertion at the lowest layer that can express it?
5. Is there a negative, boundary, or error case missing?
6. Is this a duplicate of a stronger test elsewhere?
7. If I delete or merge it, what protection is lost, and where will that protection live?

Do not delete a test merely because it is inconvenient. Explain what it protects and where the replacement belongs.

## 7. What we do not do

- We do not chase coverage percentages.
- We do not assert log lines, error prose, or help text as a substitute for behavior.
- We do not assert that a mock was called instead of asserting the outcome.
- We do not use snapshots as the only assertion for logic.
- We do not build the binary inside `go test` just to duplicate command tests.
- We do not keep a shell smoke script that reimplements fixtures and greps output when Go tests and the release-artifact smoke already cover the behavior.
- We do not test the standard library, Cobra, or the filesystem itself.

## 8. Running the suite

From the repository root:

```sh
go test ./...
go test -race ./...
go vet ./...
go build -buildvcs=false ./...
```

CI runs the same checks on Linux, macOS, and Windows. The release workflow smoke-tests the extracted release archive; that is the appropriate place for a process-level check of the packaged binary.

## 9. Review checklist

- [ ] The test asserts observable behavior or a contract.
- [ ] The assertion would fail on a realistic regression.
- [ ] The test survives a reasonable refactor.
- [ ] Exact strings are used only where a consumer depends on the exact string.
- [ ] Errors are tested by identity/type/context, not message text.
- [ ] Presentation tests assert structure, boundaries, and state, not glyphs.
- [ ] JSON/config tests start from the consumer's shape or a literal fixture.
- [ ] Rule fixtures include positive and negative cases.
- [ ] The test is deterministic and platform-safe.
- [ ] The test is not a duplicate of a stronger test at a lower layer.
- [ ] If removed or merged, the lost protection has a new home.

## 10. The short version

Test the contract at the level you are willing to promise.

- If it is logic, test the invariant.
- If it is a contract, freeze the consumer's view.
- If it is presentation, test the structure and boundaries.
- If it is an error, test its identity and context.
- If it is a wrapper, test it only when the wrapper is non-trivial.

A test should fail because behavior changed, not because someone changed a glyph.
