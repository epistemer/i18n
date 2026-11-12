# Review 0004: Gemini review of M1's core

| | |
|---|---|
| Date | 2026-11-12 |
| Roadmap items | M1.1, M1.4–M1.6 (M1.2 and M1.3 were written by Gemini and reviewed by Claude) |
| Reviewer | Gemini 3.8 Flash (High), through agy's `adversarial_review` (session `10e2a2b9-4bd8-430e-97c3-f64322b992a8`), about 427k tokens. Argon was out of quota |
| Scope | The diff of `format.go`, `registry.go`, `key.go`, `store.go`, `translator.go`, `resolve.go` and `errors.go`; the resolution and translator tests; excerpts of interface §5.1 and §9 and implementation §3–§6 |
| Result | 11 findings. 7 accepted, 1 accepted in part, 3 rejected |

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| CRIT-1 | critical | `fits` accepts a broken override when the code's text has no verbs | **Rejected.** Interface §5.2: when the code's text has no verbs, any text fits, which is what lets a translator type "50%" |
| CRIT-2 | critical | `Default` and `Source` panic with a nil database | **Rejected.** The database is required; `New`'s doc now says it must be open, and the lone nil check in `resolveCtx` is gone, so behaviour is consistent |
| MAJ-1 | major | Reading the registry before `Load` races with `Key` | **Rejected.** `Key` runs only in package-level initializers, which finish on one goroutine before `main` or any test |
| MAJ-2 | major | An unknown key name containing `%` was parsed as a format | **Accepted.** A key the catalog never saw has no signature, and renders as itself, unformatted |
| MAJ-3 | major | An invalid configured default locale went without a warning | **Accepted** |
| MAJ-4 | major | The source locale's buckets weren't recreated when its record existed without them | **Accepted** |
| MAJ-5 | major | `Codes(ctx)` is missing | **Deferred to M2.1**, which brings `Locales`; the roadmap says so |
| MIN-1 | minor | The orphan's catalog entry was read even when an override won | **Accepted.** It's read only for the signature or the final fallback |
| MIN-2 | minor | Index parsing was stricter than `fmt` (length cap, zero padding) | **Accepted.** Value-based, up to 1e6 |
| MIN-3 | minor | Orphans' overrides weren't checked for fit at `Load` | **Accepted.** They're checked against the catalog text |
| MIN-4 | minor | Test gaps | **Accepted in part.** Added: the 1,850-key scale test and benchmark, a missing base language, an unknown key with `%`, an orphan's signature. The format and registry tests it asked for already existed but weren't in the diff it saw |
