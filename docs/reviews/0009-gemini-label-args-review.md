# Review 0009: Gemini review of label arguments

| | |
|---|---|
| Date | 2026-11-14 |
| Change | Label arguments (interface §5.2, amendment of 2026-11-14), v0.3.0 |
| Reviewer | Gemini 4 Hydrogen (High), through agy's `adversarial_review` (session `3c7af4cf-3d61-41ee-bef2-54fd6eb9868f`), about 70k tokens. Every Argon tier was out of quota |
| Scope | The diff of resolve.go, key.go, args_test.go, interface §5.2 and §13, implementation §4 |
| Result | 6 findings. 3 accepted, 2 partly accepted, 1 rejected |

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| MAJ-01 | major | Rendering a `Ter` to a string breaks `Static*` adapters used with `%d` or `%x`, and bypasses a `fmt.Formatter` on a `Ter` | **Rejected for the adapters:** they are string-typed, so `%d` already printed `%!d(...)` and `%x` already printed the digits' hex before the change. **Accepted as documentation** for `fmt.Formatter`: `Ter` wins over it, now stated in §5.2 and the CHANGELOG. The test that used `StaticInt` is renamed from "OtherArgumentsUntouched" to `TestArgs_Mixed` |
| MAJ-02 | major | `fmt.Sprintf` still ran inside the `View`, so an argument's `String` or `Error` that calls `T` or `Set` could still deadlock | **Accepted.** `resolveTx` returns the text unformatted; `resolveCtx` formats after the transaction. `TestArgs_OutsideTransaction/Stringer` covers it, and fails (deadlock detected) when formatting is moved back inside |
| MAJ-03 | major | `recover` turned any panic of a nil slice, map or chan label into `<nil>`; `reflect.Interface` is dead code | **Accepted.** Only nil pointers and nil funcs, as fmt does for pointers |
| MAJ-04 | major | The stress test stops remapping after the first cycle, its validity check varied only one key, and failures surfaced only once a minute | **Partly accepted.** `unreachable` now alternates between two texts and the reader accepts only the valid combinations; failures are checked every second. Remaps happen only when bbolt's mapping doubles, so they are rare in any run; the deterministic `TestArgs_OutsideTransaction` covers them instead, as the test's comment says |
| MIN-01 | minor | The deadlock test uses `os.Exit`, skipping cleanup | **Kept, explained in a comment:** testing runs cleanups after `t.Fatal` and after a panic, and closing the store waits for the stuck reader, so the test would hang instead of failing. `Set`'s error now comes back over a channel |
| MIN-02 | minor | Labels render even when the text doesn't use them; a wrapped error isn't a `Ter`; the approval date is a placeholder | **No change.** Eager rendering is required to stay outside the transaction and matches Go's argument evaluation; a wrapped error is the errors package's concern (its `T` handles wrapping); epistemer approved on 2026-11-14 and the date is filled in |
