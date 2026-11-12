# Review 0003: Gemini review of M0

| | |
|---|---|
| Date | 2026-11-12 |
| Roadmap items | M0.1–M0.3, PR [#1](https://github.com/epistemer/i18n/pull/1) |
| Reviewer | Gemini 3.8 Flash (Medium), through agy's `adversarial_review` (session `625bd865-9584-4b46-a4f8-2a6849c9f2dc`), about 274k tokens |
| Scope | The PR's diff, without the generated `.pb.go` and `go.sum`, plus excerpts of roadmap M0 and implementation §2.2 |
| Result | 7 findings. 5 accepted (some in part), 2 rejected |

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| 01 | critical | On Windows, checkout converts to CRLF and gofmt fails every file | **Accepted.** `.gitattributes` forcing LF, copied from cfgx, which has the same CI |
| 02 | major | The Windows race detector needs gcc, absent on runners; it passes now only because there are no tests | **Rejected.** GitHub's Windows images ship MinGW gcc, and cfgx's race tests run on them. M1 adds tests, which will show it either way |
| 03 | major | `go:generate` uses `sh -c`, and the plugin path isn't quoted | **Accepted in part.** The path is now quoted. `sh` stays: ADR 0001 already limits generating to maintainers and Linux CI |
| 04 | major | The drift check misses untracked generated files; protoc versions can differ in more than the header | **Accepted.** CI also fails on untracked files. The ADR pins protoc 3.21.x, matching Ubuntu 24.04, and says moving it means moving CI too |
| 05 | major | The tidy check passes when `go.sum` is untracked | **Accepted.** `go mod tidy -diff` |
| 06 | minor | `tool` dependencies don't reach users' module graphs, so the case against buf is wrong | **Rejected.** A `tool` directive adds `require` lines to go.mod, and those are part of every dependent's module graph. The ADR's other half (needs protoc 3.21.x) was fixed under 04 |
| 07 | minor | Tidy and generation checks run on Go 1.27, though go.mod says 1.26 | **Accepted.** They run on 1.26.x. staticcheck stays on 1.27.x |
