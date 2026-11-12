# Status log

Newest first. Every working session adds an entry: what landed, the decisions
taken for you, what's next, and what's waiting on you. The terms are in
[ownership.md](ownership.md).

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
