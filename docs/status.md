# Status log

Newest first. Every working session adds an entry: what landed, the decisions
taken for you, what's next, and what's waiting on you. The terms are in
[ownership.md](ownership.md).

## 2026-11-13: the labels generator (ADR 0003); design only

- **Landed:** design records, no code:
  [ADR 0003](adr/0003-labels-generator.md), the amendment in interface §13
  with notes in §2 and §11, roadmap milestone M6 under a new v0.2.0, and a
  line in implementation §1. They are in
  [PR #7](https://github.com/epistemer/i18n/pull/7), which is open and not
  merged: a change of scope is yours to approve.
- **Asked by you:** a second, critical look at the generation pipeline you
  proposed for neozp (generators publish metadata, other packages ship
  declarative generators over it, a driver runs them), and then the records
  for it here and in cfgx.
- **Decided by you:** the direction. It replaces the rule in interface §2
  that each generator writes its own `i18n.Key` calls, which cfgx's
  `App.I18n()` followed.
- **Decided for you, overturnable:**
  - i18n ships one generator, `labels`, as a metafile and a template. No Go
    package, no command, no new dependency.
  - It writes `zz_<package>_i18n.go` with the same names cfgx's `App.I18n()`
    wrote, so moving over changes only comments.
  - It checks what no single producer can see: one key with two texts
    across the app, and a variable or a path declared twice in a package.
  - `labels/v1` belongs to the driver's documents, not to i18n.
  - `meta` bytes for labels wait for the routes generator.
- **Checked:** a prototype (one jq query, one Go template; both are in the
  ADR) rendered the 62 labels of cfgx's zp example with the same 117 lines of
  declarations as cfgx's file. It also merged two files for one package and
  stopped on both checks.
- **Not reviewed yet:** this session ran in Claude's cloud workspace, which
  can't reach agy, so Gemini hasn't seen the ADR. The PR is the place for
  that review.
- **Next:** nothing in i18n until the driver freezes its metafile format;
  then M6.1.
- **Waiting on you:**
  - the PR;
  - a name and a repository for the driver (working name `genx`);
  - earlier: deleting the merged branches.

## 2026-11-12: a skill for using i18n

- **Landed:** a global Claude Code skill, `~/.claude/skills/i18n/`, like
  cfgx's. `SKILL.md` covers the model, the workflow for a new text, key
  naming, the rules, locales, use outside requests, an admin page, tests and
  gotchas. `reference.md` lists every identifier. Every API claim in it was
  compiled against v0.1.0 from the module proxy. Your global CLAUDE.md points
  to it.
- **Corrected:** the previous entry asked whether to move zp onto the
  library. That's not a question: neozp is built from scratch.
- **Next:** unchanged. The Argon (High) whole-repo review once its quota
  returns, then the roadmap's "Later" list.
- **Waiting on you:** nothing beyond deleting the merged branches.

## 2026-11-12: M2–M5, v0.1.0 released

- **Landed:**
  - **M2** ([PR #3](https://github.com/epistemer/i18n/pull/3)): locale
    management, `Set`, `Reset`, `Search`, orphans. The four zp regression
    tests pass. `Search` over 1,850 keys takes about 3.6 ms.
  - **M3** ([PR #4](https://github.com/epistemer/i18n/pull/4)): the
    middleware (one read per request, `Vary` whenever the header takes part)
    and the cookie helpers.
  - **M4** ([PR #5](https://github.com/epistemer/i18n/pull/5)): `i18nschema`
    as its own module ([ADR 0002](adr/0002-nested-modules.md)), `i18ntest`,
    and `examples/basic`. The example's end-to-end test changes the default
    locale at runtime through cfgx's store.
  - **M5** ([PR #6](https://github.com/epistemer/i18n/pull/6)): godoc
    examples, a README whose quickstart was run for real,
    [api-review.md](api-review.md), and the CHANGELOG.
  - **v0.1.0** is tagged, with `i18nschema/v0.1.0`. Both resolve through
    proxy.golang.org: `go get github.com/epistemer/i18n@v0.1.0` in an empty
    module builds and runs.
- **Reviewed:**
  - [0005](reviews/0005-gemini-m2-review.md): Flash (High), about 382k
    tokens.
  - [0006](reviews/0006-gemini-m3-review.md): Flash (High), about 307k.
  - [0007](reviews/0007-gemini-m4-review.md): Flash (High), about 341k.
  - [0008](reviews/0008-gemini-m5-review.md): Flash (Medium), about 311k.
  - Gemini also wrote M2.1 (Flash High, about 205k), which I reviewed.
  - Across the session, Gemini spent about 3.1M tokens on Flash models;
    Argon was out of quota throughout.
  - The catches that mattered:
    - `Prune` before `Load` could delete a key that came back;
    - `Vary` was missing when the store read failed;
    - the first quickstart reseeded translations on every boot, overwriting
      people's edits.
- **Decided for you:**
  - `i18nschema` and the example are separate modules (ADR 0002).
  - I wrote M3.2 and M4 myself instead of delegating them; each was smaller
    than writing its brief.
  - M4.2's done-when now names the example's handler test, since the e2e
    test runs on cfgx's real store on purpose.
- **Next:** after v0.1.0, the roadmap's "Later" list. The first Argon (High)
  review is due once its quota returns, around 2026-11-15. It's a whole-repo
  review the Flash runs couldn't replace.
- **Waiting on you:** deleting the merged branches (`m0-foundations` through
  `m5-release`), which the safety checks leave to you. (An earlier version
  of this entry asked whether to move zp onto the library. That was wrong:
  zp isn't ported. neozp is built from scratch, and zp's business side moves
  over in one go once the framework is mature.)

## 2026-11-12: M0 and M1, the read path

- **Landed:**
  - **M0** ([PR #1](https://github.com/epistemer/i18n/pull/1), merged with
    12 green checks): the module, CI on three OSes and two Go versions, and
    [ADR 0001](adr/0001-protobuf-generation.md) on protobuf generation.
  - **M1** (PR #2): the format check, locale codes, the `Ter` adapters, the
    registry, the store, `New`, `Load`, the resolution order, and the context
    helpers.
- **Measured:** a hit in the visitor's locale takes 1.4 µs; a miss through
  three buckets to the code's text, 2.1 µs. Both are within the plan's
  budget. `Load` on 1,850 unchanged keys takes 4.9 ms and writes nothing.
  `FuzzFit` ran 10 minutes (about 10.8 million cases) clean, after it caught
  one real divergence from `fmt`: a width over 1,000,000.
- **Reviewed:**
  - [Review 0003](reviews/0003-gemini-m0-review.md), M0: Gemini 3.8 Flash
    (Medium), about 274k tokens. 5 of 7 findings accepted; the Windows
    line-ending fix was the important one.
  - [Review 0004](reviews/0004-gemini-m1-review.md), M1: Gemini 3.8 Flash
    (High), about 427k tokens. 8 of 11 findings accepted.
  - Gemini wrote M1.2 and M1.3 (about 244k tokens), and I reviewed them. I
    fixed one bug: the singleton rule for private-use tags.
  - Every Argon tier is still out of quota until about 2026-11-15.
- **Decided for you:**
  - I wrote M1.1 (the format check) myself instead of delegating it, because
    the rest of M1 depends on it.
  - `Codes(ctx)` moves to M2.1 with `Locales`.
  - An unknown key renders as itself, never formatted.
- **Next:** M2. Locale management, `Set` and `Reset`, `Search`, orphans.
- **Waiting on you:** nothing. Deleting merged branches (`m0-foundations`) is
  blocked by the safety checks, so they stay until you remove them.

## 2026-11-12: interface locked, implementation plan, milestones

- **Landed:** the interface is locked, with your decisions and the proposed
  answers to its open questions. [implementation.md](implementation.md) covers
  the bucket layout, the resolution path, the fit check, `Load`, the editor
  operations, locale codes, HTTP and the test plan. The
  [roadmap](roadmap.md) has milestones M0–M5 up to v0.1.0, each item with a
  done-when, and a split between Claude and Gemini. The repository is pushed
  to [github.com/epistemer/i18n](https://github.com/epistemer/i18n).
- **Reviewed:** [review 0002](reviews/0002-gemini-plan-review.md), Gemini
  3.8 Flash (High), about 147k tokens, because Argon is still out of quota.
  All 16 findings accepted (two as clarifications), none touching the
  interface. The most serious: `Reset` would have returned stale text, and
  `DynamicWrap` would have panicked on unregistered keys.
- **Next:** M0. The module, CI, and ADR 0001 on protobuf generation.
- **Waiting on you:** nothing.

## 2026-11-12: study of zp's i18n, draft interface, ownership

- **Landed:** the first commit of the repository: [interface.md](interface.md)
  (draft, not locked), [roadmap.md](roadmap.md) (the ideas parked for after
  v1), this log, [ownership.md](ownership.md), a README and the MIT license.
- **Learned from zp (master):** keys are package-level `Key` vars, about 1,850
  of them, ~1,600 generated from tables, errors, routes and enums. Translations
  live only in production's bbolt, typed by hand, and double as white-labeling.
  Four bugs were confirmed with a throwaway test: changed English defaults never
  reach the store, rollbacks delete translations, a `%` in a translation breaks
  the text, and restoring a route key returns its meta bytes.
- **Decided by you:** the database wins over the code; record edits and their
  base text; format only with `args`; split source and default locale; bbolt
  read directly (it measured about 1.3 µs per lookup); protobuf for records;
  errors, authorization and the editor UI out of scope; soft delete if cheap.
- **Decided for you:**
  - The store holds only overrides plus a key catalog, instead of a value and
    an "edited" flag for every key (D1).
  - `New` takes a `*bbolt.DB` with no storage interface; tests use a temp file.
  - [Review 0001](reviews/0001-gemini-design-review.md): 10 of 15 findings
    accepted and amended in place. Gemini 3.8 Flash (High), about 86k tokens,
    because every Argon tier was out of quota.
- **Next:** the roadmap's v1 milestones and implementation.md, then M0
  (module, CI) once the GitHub repository exists.
- **Waiting on you:**
  - Creating the empty `github.com/epistemer/i18n` repository, since gh can't
    create repos under epistemer.
  - Locking the interface. It's API design, so it's yours. That includes D1's
    override-only store and the three open questions in §12, each of which
    has a proposed answer.
