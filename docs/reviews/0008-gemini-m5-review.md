# Review 0008: Gemini review of the v0.1.0 docs

| | |
|---|---|
| Date | 2026-11-12 |
| Roadmap items | M5.1, M5.2 |
| Reviewer | Gemini 3.8 Flash (Medium), through agy's `adversarial_review` (session `371c8c25-ebec-456e-abd9-376c232f3886`), about 311k tokens |
| Scope | README.md, CHANGELOG.md, example_test.go, docs/api-review.md |
| Result | 13 findings. 12 accepted, 1 rejected |

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| CRIT-01 | critical | The quickstart lacks `go mod init` and `go get go.etcd.io/bbolt` | **Accepted** |
| MAJ-01 | major | The quickstart called `Set` on every boot, overwriting people's edits, the very thing the library exists to prevent | **Accepted.** It seeds only when `CreateLocale` first succeeds. Rerun, restart included; the README shows that run's output |
| MAJ-02 | major | `bolt.Open` with no timeout hangs on a held lock | **Accepted.** One second |
| MAJ-03 | major | The examples discard errors | **Accepted.** A `must` helper |
| MIN-01 | minor | The cfgx snippet uses an undeclared `English` | **Accepted** |
| MIN-02 | minor | No module paths for `i18nschema` and `i18ntest`; `i18nschema` needs its own `go get` | **Accepted** |
| MIN-03 | minor | "How text is chosen" misses the fifth step, the key's name | **Accepted** |
| MIN-04 | minor | api-review.md's link to interface.md is broken | **Rejected.** Both files are in docs/, so the relative link resolves |
| MIN-05 | minor | api-review.md names methods inconsistently and omits `Translator` | **Accepted** |
| MIN-06 | minor | The CHANGELOG's "only edits are stored" and "Load writes nothing" are imprecise | **Accepted.** Reworded |
| MIN-07 | minor | `Reset` has no example of its own | **Accepted** |
| MIN-08 | minor | `os.MkdirTemp`'s error is ignored | **Accepted** |
| MIN-09 | minor | An Arabic text with a count teaches plural-unsafe formatting | **Accepted.** The quickstart greets by name instead, and the README says there's no plural support yet |
