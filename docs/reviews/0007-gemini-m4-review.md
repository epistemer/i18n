# Review 0007: Gemini review of M4, integration

| | |
|---|---|
| Date | 2026-11-12 |
| Roadmap items | M4.1–M4.3 |
| Reviewer | Gemini 3.8 Flash (High), through agy's `adversarial_review` (session `96a2185e-00dc-45bf-b5ff-966cc384276a`), about 341k tokens. Argon was out of quota |
| Scope | The M4 diff without generated code and go.sum, with interface §2, §4 and §10 |
| Result | 12 findings. 9 accepted (one as documentation), 3 rejected |

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| F-01 | major | The example's shutdown goroutine leaks when listening fails, and `Close` drops connections | **Accepted.** A done channel and `Shutdown` with a timeout |
| F-02 | major | The example ignores SIGINT and SIGTERM | **Accepted.** `signal.NotifyContext` and `ExecuteContext` |
| F-03 | major | `Setup` loads before creating its locale, so `Load` warns about a missing default | **Accepted.** The locale is created first |
| F-04 | major | CI never regenerates the example's settings | **Rejected.** `cfgxtest.AssertGenerated` runs in the example's tests, which CI runs |
| F-05 | major | M4.2's done-when says the e2e test uses `i18ntest`; it doesn't | **Accepted.** The example's handler test uses it; the done-when now names that test, since the e2e test uses cfgx's real store on purpose |
| F-06 | major | The example can't build outside a clone | **Accepted as documentation** in ADR 0002, the same trade-off as cfgx's examples |
| F-07 | minor | The e2e test never uses the cookie it receives | **Accepted** |
| F-08 | minor | The e2e helper returns a response whose body is closed | **Accepted** |
| F-09 | minor | `i18ntest.Context` should derive from `tb.Context()` | **Accepted** |
| F-10 | minor | `i18nschema`'s test checks almost nothing | **Rejected.** The example's compile-time `i18n.Config` assertion and the e2e test (which changes the Mutable key at runtime) check the group end to end |
| F-11 | minor | The key's description differs from locked §4.2 | **Accepted.** Restored to the locked text |
| F-12 | minor | `i18ntest.Context` panics on a nil Translator | **Rejected.** A test helper; the panic names the problem |
