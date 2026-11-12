# Review 0002: Gemini review of implementation.md and the v0.1.0 milestones

| | |
|---|---|
| Date | 2026-11-12 |
| Reviewer | Gemini 3.8 Flash (High), through agy's `adversarial_review` (session `b91cad16-5610-4bf0-abee-4f0da7bf436c`), about 147k tokens. Argon was still out of quota |
| Scope | [implementation.md](../implementation.md) and the v0.1.0 section of [roadmap.md](../roadmap.md), judged against the locked [interface.md](../interface.md). The run's working directory held only those three files |
| Result | 16 findings. Claude checked each one and accepted all of them, two as clarifications only. None changes the interface |

## Findings and resolutions

| ID | Reviewer's severity | Finding | Resolution |
|----|---------------------|---------|------------|
| REV-01 | critical | `meta` decoded by protobuf could point into bbolt's memory after the transaction | **Clarified.** `proto.Unmarshal` copies by default, but every byte field that leaves a transaction is now cloned explicitly (implementation §2.1) |
| REV-02 | critical | `Reset` resolving through a separate `View` reads the last committed state, not its own deletes | **Accepted.** `resolveTx` takes the transaction; `T` passes a `View`'s and `Reset` its own (§4) |
| REV-03 | critical | `DynamicWrap` handles and unregistered names have no entry: `K()` and the fit check would dereference nil | **Accepted.** The handle stores its name; the resolver looks it up in the registry, then the catalog, and parses an orphan's signature on the spot (§3, §4) |
| REV-04 | critical | `map[int]rune` keeps one verb per argument, but `"%[1]d (0x%[1]x)"` uses two | **Accepted.** A set of verbs per argument (§5) |
| REV-05 | critical | M1.5 and M1.6 had done-whens that need `Set`, which comes in M2.2 | **Accepted.** M1 tests write buckets directly; the zp regression tests and the `Set` race test moved to M2.2 |
| REV-06 | major | `Set`'s base for `ar-EG` skipped the `ar` translation | **Accepted.** The base is interface §5.1 steps 2–4 (§7) |
| REV-07 | major | `Search` with a prefix didn't stop at the prefix's end | **Accepted.** The cursor stops at the first key without the prefix (§7) |
| REV-08 | major | `Search` took default text from the catalog, which `T` never does | **Accepted.** It takes it from the registry, like `T` (§7) |
| REV-09 | major | The parser missed the `[n]` slot before a precision `*` | **Accepted.** All three slots (§5) |
| REV-10 | major | `Load`'s fit check is read-only work inside the write transaction | **Accepted.** It runs in a `View` after the commit (§6) |
| REV-11 | major | `ParseAcceptLanguage` didn't say it trims spaces or falls back to `Default()` | **Accepted** (§8) |
| REV-12 | major | `Vary` must be set whenever the header took part, even when the default won | **Clarified** (§9) |
| REV-13 | major | Most roadmap items had no done-when | **Accepted.** All 20 have one |
| REV-14 | minor | Canonical case uppercased extension subtags, and private-use tags were rejected | **Accepted.** Lowercase after a singleton; `x-…` tags allowed (§8) |
| REV-15 | minor | Read paths before the first `Load` would hit a nil bucket | **Accepted.** Every read path tolerates missing buckets (§2.1) |
| REV-16 | minor | The test plan missed empty overrides, the orphan lifecycle and rate-limited logging | **Accepted** (§10) |
