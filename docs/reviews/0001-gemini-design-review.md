# Review 0001: Gemini review of the draft interface

| | |
|---|---|
| Date | 2026-11-12 |
| Reviewer | Gemini 3.8 Flash (High), through agy's `adversarial_review` (session `2bff9f59-fd8a-4ae8-baa1-8c5a4392a50d`), about 86k tokens. Every Argon tier was out of quota for about 68 hours, so Claude picked the strongest Gemini model available instead of waiting |
| Scope | [interface.md](../interface.md) and [roadmap.md](../roadmap.md) as first drafted, plus a context note with the constraints the owner set (not up for review). The run's working directory held only those three files |
| Result | 15 findings. Claude checked each against the draft and against Go's `fmt` and bbolt. 10 accepted (some only as clarifications), 5 rejected. The accepted ones are amended in place (interface §13) |

## Findings and resolutions

| ID | Reviewer's severity | Finding | Resolution |
|----|---------------------|---------|------------|
| CRIT-01 | high | §5.1 skips the base language: with `ar` and `ar-EG` both present, an `ar-EG` visitor gets English for every key `ar-EG` doesn't override | **Accepted.** New step 2: the base language's override, if that locale exists |
| CRIT-02 | high | `Dynamic` keys can't be edited with `Set` and end up orphaned | **Clarified.** The finding assumed `Dynamic` keys are unregistered. In zp they're registered with `Key` (`service/gl.go`); `Dynamic` only looks one up by name. §3.3 now says so, and that text people create as data isn't a translation key |
| CRIT-03 | high | An empty override with `args` prints `%!(EXTRA int=5)` | **Accepted**, confirmed with `fmt`. Empty text stays empty |
| CRIT-04 | high | D1 says "nothing else is stored, no sync step", but `Load` stores each key's code text and meta | **Accepted as a wording fix.** D1 now names the catalog and says `T` never reads a registered key's text from it, which is also why an older release rewriting the catalog is harmless (§9) |
| CRIT-05 | high | Case-insensitive codes on top of byte-keyed bbolt can split one locale in two | **Accepted.** Every code is converted to canonical BCP 47 case on the way in, and only that form is stored (§6.1) |
| MAJ-01 | medium | The frozen global registry breaks parallel tests | **Rejected.** Package initialization finishes before any test runs, so the registry is read-only by then and concurrent `Load`s are safe. §3.1 now says so. A `Registry` type would add API surface for a case the rule already excludes |
| MAJ-02 | medium | The verb rule blocks legitimate omissions, and a code change can leave overrides that print `%!(EXTRA …)` | **Accepted, both parts.** `fmt` reports unused arguments only when no explicit index is used (checked), so the fit rule now allows omission through indexes. `T` skips an override that no longer fits, `Load` logs them, and `Entry.Misfit` shows them (§5.2) |
| MAJ-03 | medium | `SetCookie` can't clear the cookie or set `Domain`/`Secure` | **Accepted.** `Cookie(code)` returns the cookie for the app to adjust, plus `ClearCookie` |
| MAJ-04 | medium | `Code(ctx)` returns a `Locale`, not a string; rename it `CurrentLocale` | **Rejected.** `Code` and `Codes` are names the owner chose to keep |
| MAJ-05 | medium | Code outside a request has to misuse `ParseAcceptLanguage` on a stored code | **Accepted.** `Match(code)`: exact, base language, then default |
| MAJ-06 | medium | `Search` would open thousands of read transactions and block writers | **Clarified.** It runs in one read transaction (now stated). The claim that reads block writers is wrong for bbolt: readers block only a remap when the file grows |
| MAJ-07 | medium | `T` has no stated behaviour when the read fails | **Accepted.** It returns the code's text and logs, at most once a minute (§5.1). A health-check method was left out |
| MIN-01 | low | `StaticOptional` uses the global `NoneString`, while the int variants take `none` | **Rejected for v1.** zp's signatures are kept. `NoneString` is set at startup, like any other package setting |
| MIN-02 | low | `Key` should return a concrete type | **Rejected.** `TKer` is the owner's name and type. The dispatch cost is negligible next to the bbolt read |
| MIN-03 | low | `Prune` has no dry run | **Rejected.** `Orphans` is the preview; §8.4 now says `Prune` deletes exactly what it lists |
