# Review 0006: Gemini review of M3, HTTP

| | |
|---|---|
| Date | 2026-11-12 |
| Roadmap items | M3.1, M3.2 |
| Reviewer | Gemini 3.8 Flash (High), through agy's `adversarial_review` (session `6578dec1-700f-4ff9-853b-d23d013efb80`), about 307k tokens. Argon was out of quota |
| Scope | `http.go` and `http_test.go`, with interface §7 and implementation §9 |
| Result | 9 findings. 5 accepted (one as documentation), 4 rejected |

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| F-01 | critical | `ParseAcceptLanguage` is missing | **Rejected.** It's in `locales.go` (M2.1), outside the excerpt it saw; it now shares `acceptTx` with the middleware |
| F-02 | critical | A nil database panics the middleware | **Rejected.** The database is required (`New`'s doc) |
| F-03 | major | No `Vary` when the store read fails, so caches keep the fallback page | **Accepted** |
| F-04 | major | Only the first `Accept-Language` line is read | **Accepted.** All lines are joined, as HTTP allows splitting the list |
| F-05 | major | The clearing cookie lacks `Expires` for clients that ignore `Max-Age` | **Accepted** |
| F-06 | major | `ClearCookie` can't clear a cookie the app gave `Domain` or `Secure` | **Accepted as documentation.** The locked signature has no room for them; the doc says to clear with the same attributes |
| F-07 | minor | A canceled request still opens a read transaction | **Rejected.** One read costs about a microsecond |
| F-08 | minor | `Cookie` reports a store failure as an unknown locale | **Rejected for v1.** `Locale` returns a bool by the locked interface |
| F-09 | minor | Test gaps | **Accepted:** the raw `Set-Cookie` line, the Translator in the handler's context, a store failure, several header lines, a custom cookie name |
