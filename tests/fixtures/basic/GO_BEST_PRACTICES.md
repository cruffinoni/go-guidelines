# GO_BEST_PRACTICES

## 1. Imports and Formatting
**Do**
Use gofmt.
**Don't**
Use dot imports.
**Rationale**
Readability.

## 2. Package Design and Documentation
**Do**
Use clear package docs.
**Don't**
Use util.
**Rationale**
Clarity.

## 3. Errors
**Do**
Wrap errors with %w.
**Don't**
panic(err)
**Rationale**
Error values.

## 4. Context Usage
**Do**
Pass context first.
**Don't**
Store context in structs.
**Rationale**
Cancellation.

## 5. Concurrency
**Do**
Use errgroup.
**Don't**
Fire-and-forget goroutines.
**Rationale**
Control.

## 6. HTTP and I/O
**Do**
Use request with context and close response body.
**Don't**
Use http.Get directly.
**Rationale**
Safety.

## 7. Interfaces
**Do**
Keep interfaces small.
**Don't**
Return interfaces from constructors.
**Rationale**
Flexibility.

## 8. Defer, Panic, and Recover
**Do**
Check err then defer close.
**Don't**
panic in normal flow.
**Rationale**
Correctness.

## 9. Slices, Maps, and Strings
**Do**
Use nil slices or prealloc.
**Don't**
Copy strings.Builder.
**Rationale**
Performance.

## 10. JSON and Zero Values
**Do**
Use omitempty.
**Don't**
Assume nil slice becomes [].
**Rationale**
Encoding behavior.

## 11. Logging
**Do**
Use structured logging.
**Don't**
Use log.Printf.
**Rationale**
Observability.

## 12. Tooling and Modules
**Do**
Have valid go.mod.
**Don't**
Miss go directive.
**Rationale**
Build integrity.

## 13. Timeouts, Tickers, and Timers
**Do**
Stop tickers and timers.
**Don't**
Use time.AfterFunc casually.
**Rationale**
Leaks.

## 14. Pipelines and Cancellation
**Do**
Handle ctx.Done.
**Don't**
Ignore cancellation.
**Rationale**
Goroutine lifecycle.

## 15. Testing
**Do**
Keep tests deterministic.
**Don't**
Depend on map order.
**Rationale**
Reliability.

## 16. Usage of `init()`
**Do**
Prefer explicit setup.
**Don't**
Hide behavior in init.
**Rationale**
Testability.

## 17. Variable Declarations
**Do**
Group related var declarations.
**Don't**
Scatter many single var lines.
**Rationale**
Readability.

## 18. Method Organization
**Do**
Use methods for struct-bound logic.
**Don't**
Use orphan helper functions with struct pointer first arg.
**Rationale**
Discoverability.

## 19. Required Dependencies and Nil Checks
**Do**
Treat required deps as constructor contract.
**Don't**
Silently nil-check required deps.
**Rationale**
Fail fast.

## 20. Function Signature Line Length
**Do**
Keep short signatures on one line.
**Don't**
Split short signatures across lines.
**Rationale**
Consistency.

## 21. Function Declaration Order
**Do**
Declare local functions before their first use in the file.
Keep mutually recursive functions contiguous.
**Don't**
Call a local function before its declaration.
Split mutually recursive functions with unrelated declarations.
**Rationale**
Improve readability with top-down flow similar to C-style organization.
