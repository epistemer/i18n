# Changelog

## v0.1.0 (2026-11-12)

The first release, built against the interface locked on 2026-11-12
([docs/interface.md](docs/interface.md)).

- **Keys:**
  - `Key` declares a translation key with its default text and optional meta.
  - `Ter`, `TKer` and the `Static*` adapters make any value a label.
  - `Dynamic` and `DynamicWrap` handle keys known only at runtime.
- **Storage:** bbolt, under one top-level bucket, safe to share with cfgx's
  store.
  - A translation is stored only once someone edits it, with the source text
    it was based on. Untranslated keys never copy the code's text into the
    store.
  - Locale records and a catalog of keys complete the store. `Load` refreshes
    the catalog and writes nothing when the registered keys match it.
- **Resolution:** the locale's edit, then its base language's, then the
  source language's, then the code's text. Text is formatted only when
  arguments are passed, and a translation whose format verbs no longer fit
  the code is skipped.
- **Locales:** canonical BCP 47 codes. A source locale fixed by the app and a
  default locale from config. `Locales`, `CreateLocale`, `UpdateLocale`,
  `DeleteLocale`, `Match`.
- **HTTP:** a `net/http` middleware (cookie, then `Accept-Language` unless
  ignored, then the default; one read per request, `Vary` when the header
  takes part), and cookie helpers.
- **Editing:** `Set`, `Reset`, `Search`, `Orphans`, `Prune`. Keys removed from
  the code are kept as orphans until pruned, so rollbacks lose nothing.
- **Integration:**
  - `i18nschema` (its own module, tagged `i18nschema/v0.1.0`) declares the
    default locale for cfgx apps.
  - `i18ntest` builds Translators for tests.
  - `examples/basic` is a complete cfgx app with an end-to-end test.
