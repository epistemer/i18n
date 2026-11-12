# Ownership until v1.0.0

| | |
|---|---|
| Owner | Claude, acting for epistemer (delegated on 2026-11-12) |
| Until | `v1.0.0` is tagged |
| Coding assistant | agy → Gemini. Claude picks the model for each task and never waits for a quota: when a model is cooling down, the strongest available one runs instead |
| Source of truth | [interface.md](interface.md) (the contract), implementation.md (how, once written), [roadmap.md](roadmap.md) (order and done-when) |
| Record | [status.md](status.md), plus a PR for every code change |

## What Claude does without asking

- Plans and sequences the roadmap, and splits items between Claude and agy.
- Writes code, tests and docs. Delegates leaf items to agy, each in its own git
  worktree outside the repository.
- Has every change reviewed by the other model family: Gemini reviews Claude's
  work, and Claude reviews Gemini's. Reviews run in a scratch folder that holds
  only what's under review.
- Opens PRs, and merges them into `main` once CI is green and the review
  findings are handled. Docs-only bookkeeping (status log, roadmap ticks) may go
  straight to `main`.
- Decides implementation questions and spike outcomes, each recorded as an ADR.
- Makes amendments that clarify the interface without changing its intent.
  Each is listed in interface §13 and in the status log, so you can overturn it.
- Adds dependencies an ADR justifies. The core stays on bbolt and protobuf.
- Tags releases from `v0.1.0` up to, but not including, `v1.0.0`, once a
  release's done-when is met.
- Manages agy: which tasks Gemini takes, which model and effort each gets, and
  when to review or redo its work.

## What comes back to you

1. **API design decisions.** Anything that changes what a developer writes or
   gets back, beyond a clarification: new or changed exported API, behaviour an
   application can see, and any change to decisions D1–D13 or to the
   principles in interface §1. Claude brings a recommendation.
2. **Anything Claude lacks permission for.** Work outside this repository (such
   as zp, its staging and its production), your accounts, GitHub settings such
   as creating the `epistemer/i18n` repository, tokens and paid services,
   speaking for you to other people, and operations the safety checks block,
   such as deleting branches or rewriting history.

## How you follow along

- [status.md](status.md) gets a dated entry for every working session: what
  landed, decisions taken for you, what's next, and what's waiting on you.
- Every code change lands as a PR on GitHub, with its review notes and CI.
- Roadmap boxes are ticked as items meet their done-when.

## How work continues

Claude doesn't run in the background. Work happens while a session runs: when
you say "continue", or on a schedule if you set one up. Each session starts
from the roadmap, the status log and the memory notes, so any session can pick
up where the last one stopped.

## Limits

- **v1.0.0 is your call.** Tagging it promises the API won't break, so Claude
  doesn't tag it on its own.
- **Quotas.** agy uses your Gemini quota, and Claude's sessions have context
  limits. Batches of delegations check `agy_status` first.
- **Safety checks still apply.** Claude leaves blocked git operations to you
  rather than working around them.
