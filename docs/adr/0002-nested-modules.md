# ADR 0002: i18nschema and the example are separate modules

| | |
|---|---|
| Date | 2026-11-12 |
| Status | Accepted |
| Roadmap items | M4.1, M4.3 |

## Context

`i18nschema` imports `github.com/epistemer/cfgx/dsl`, and the example app needs
cfgx, cobra and the `cfgx` code generator. If they lived in the core module,
its go.mod would require cfgx, and every user's module graph would include
cfgx's dependencies (cobra, a YAML library, `golang.org/x/tools`), even users
who never configure the library through cfgx. Interface §2 promises a core
that depends on bbolt and protobuf only.

## Decision

- `github.com/epistemer/i18n/i18nschema` is its own module. It requires only
  cfgx and doesn't import the core, so its go.mod needs no `replace`, and the
  import path stays the one interface §2 names.
- `examples/basic` is its own module, as cfgx's examples are. It points at the
  local core and `i18nschema` with `replace`, and pins the `cfgx` generator
  with a `tool` directive.
- CI runs vet, staticcheck, tidy checks and race tests in all three modules.
  The example's tests check its generated settings with
  `cfgxtest.AssertGenerated` and run the end-to-end test.

## Consequences

- Releases tag `i18nschema` separately (`i18nschema/v0.1.0` alongside `v0.1.0`).
- A change to the core that the example needs is tested in the same PR,
  through the `replace`.
- The example builds only inside a clone of this repository: its `replace`
  directives point at the local core and `i18nschema`, so
  `go run github.com/epistemer/i18n/examples/basic@version` won't work. That's
  the same trade-off cfgx's examples make; the example is for reading and
  running from a clone.
