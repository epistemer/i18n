# Review 0005: Gemini review of M2's editor operations

| | |
|---|---|
| Date | 2026-11-12 |
| Roadmap items | M2.2, M2.3 (M2.1 was written by Gemini and reviewed by Claude) |
| Reviewer | Gemini 3.8 Flash (High), through agy's `adversarial_review` (session `99236748-6947-4404-bfe7-a9003bec684c`), about 382k tokens. Argon was out of quota |
| Scope | `editor.go` and `editor_test.go`, with interface §8 and implementation §7; `resolve.go` as context |
| Result | 11 findings. 7 accepted (some in part), 2 rejected, 2 merged into others |

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| CRIT-1 | critical | `Prune` keeps `*Bucket` pointers across writes, which go stale for inline buckets | **Rejected.** bbolt caches child buckets per transaction (`top.Bucket` returns the same object every time), and writes go through copied nodes, never into the mapped page. Looking a bucket up again changes nothing |
| CRIT-2 | critical | `top.Bucket(name)` may return nil in `Prune` | **Accepted** as a guard. bbolt reports `v == nil` only for buckets, but the check costs nothing |
| MAJ-1 | major | `Prune` before `Load` could delete a key that's back in the code | **Accepted.** An orphan is a key marked orphaned *and* not registered by the running release |
| MAJ-2 | major | `Search` before `Load` lists keys gone from the code | **Accepted.** `Search` lists only keys this release registers |
| MAJ-3 | major | `Set` records a fallback override that no longer fits as the base | **Accepted.** For keys with verbs, fallbacks that don't fit are skipped, as they are when the page renders |
| MAJ-4 | major | `Entry.Text` can be a fallback that doesn't fit | **Accepted**, with MAJ-3's rule; `Entry.Source` follows it too |
| MIN-1 | minor | `Search` builds entries it then skips for `Offset` | **Accepted.** Fast path when there's no filter |
| MIN-2 | minor | A `Match` filter can mutate `Entry.Meta` | **Rejected.** The filter is the app's own code, and each entry already has its own copy |
| MIN-3 | minor | Test gaps | **Accepted:** `Match` with paging, `Reset` on an orphan, `Prune` deleting base records, updating an override; plus `Prune` before `Load` and the misfit fallbacks |
