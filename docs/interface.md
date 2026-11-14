# i18n — Developer Interface (v1, locked)

> Status: **locked on 2026-11-12.** Any change needs an explicit amendment,
> recorded in §13. This document covers only what a developer writes and what
> they get back. How it's built (bucket layout, record formats) is in
> [implementation.md](implementation.md). Ideas left out of v1 are in
> [roadmap.md](roadmap.md).

## 0. Decisions so far

| # | Decision | Choice |
|---|----------|--------|
| D1 | Source of truth | **The code holds keys and their default text. The store holds what people wrote (overrides), plus a catalog of keys that `Load` refreshes.** An override always beats the code (§5.1) |
| D2 | Declaring keys | Package-level `var X = i18n.Key("dotted.key", "Default text")`. A global registry, frozen by `Load` |
| D3 | Store | bbolt. The app passes in a `*bbolt.DB`, normally cfgx's `Store().DB()`. Everything lives under one top-level bucket. No storage interface |
| D4 | Record format | Protobuf for records. Translated text is stored as raw bytes, so `T` never decodes a message. Generated protobuf types never appear in the API |
| D5 | Locales | The **source** locale (the language of the code's defaults) is fixed by the app's code. The **default** locale (what visitors get) comes from config and can change at runtime |
| D6 | Formatting | Text goes through `fmt.Sprintf` only when `args` are passed. Saving an override checks its verbs against the code's text |
| D7 | Keys removed from code | Marked orphaned, not deleted. Their overrides survive a rollback. `Prune` deletes them on purpose |
| D8 | Context | The translator and the locale ride in the `context.Context`. With no translator in it, you get the code's text |
| D9 | HTTP | A `net/http` middleware in the core package. No web framework dependency |
| D10 | Logging | `log/slog`, replaceable with `WithLogger`. Apps that use logrus bridge it |
| D11 | Errors | Out of scope until the errors redesign ([roadmap](roadmap.md#errors)) |
| D12 | Module path | `github.com/epistemer/i18n`, package `i18n` |
| D13 | License | MIT, like cfgx |

D1 is how the agreed "record per key and locale whether it was edited, and from
which original" takes shape. An override *is* an edit, and it records the
source text it was based on. Values nobody edited aren't stored, so a changed
default in the code reaches them at once. The catalog (each key's code text and
meta, §9) serves `Search` and orphaned keys. `T` never reads a registered key's
text from it: that comes from the running code.

---

## 1. Principles

1. **The database wins.** Every deployment can rewrite any text, the source
   language included. No code default, sync or upgrade overwrites what a person
   wrote. This is what makes the library the white-labeling mechanism.
2. **One declaration per text.** A key is declared once, next to its use, with
   its default text in the source language. Its handle is a typed Go value, not
   a string to look up.
3. **Defaults follow the code.** Anything nobody edited always shows the code's
   current text.
4. **Nothing is lost by deploying.** Deploying an older or newer release never
   deletes what people wrote.
5. **Reads go to the store.** `T` reads bbolt directly. Edits are visible at
   once, and there's no cache to invalidate.
6. **Fail at startup, not later.** Registry problems (duplicate keys with
   different texts, empty keys) fail `Load` with every problem listed at once.

---

## 2. Project layout

```
github.com/epistemer/i18n             the core: keys, Translator, T, middleware
                                      + genx.meta.yaml, labels.go.tmpl: the labels generator (§3.4)
github.com/epistemer/i18n/i18nschema  cfgx schema fragment (the only package that imports cfgx)
github.com/epistemer/i18n/i18ntest    test helpers
github.com/epistemer/i18n/labels      the generator's tests, a nested module (requires genx)
```

The core depends on bbolt and protobuf only. The `labels` generator is data
in the core package's directory, which genx reads; it adds no Go API and no
dependency.

Code generators that read an app's schema (tables, enums, routes, cfgx's
settings) belong to the packages that own them. They don't import i18n:
they write their labels as data, in `labels/v1` files, and i18n's `labels`
generator turns every such file into `i18n.Key` declarations
([ADR 0003](adr/0003-labels-generator.md)).

---

## 3. Declaring keys

### 3.1 `Key`

```go
package pages

import i "github.com/epistemer/i18n"

var (
	L10nLocalizations = i.Key("page.i18n.title", "Localizations")
	L10nAddLocale     = i.Key("page.i18n.add_locale", "Add Locale")
)

templ Title() {
	<h1>{ L10nLocalizations.T(ctx) }</h1>
}
```

```go
func Key(key, text string, meta ...byte) TKer
```

- `text` is the default in the source locale.
- `meta` is opaque bytes stored with the key, once per key (not per locale).
  Search returns it (§8.3). zp uses it for route details, so global search
  matches route names in the user's language.
- **Call `Key` only in package-level variable initializers.** `Load` freezes
  the registry, and `Key` panics after that with a message naming the key.
  Package initialization finishes before `main` or any test runs, so the
  registry is read-only by the time anything reads it, and concurrent `Load`
  calls (parallel tests) are safe.
- The same key registered twice with the same text and meta is fine.
  Registering it with different text or meta makes `Load` fail.

### 3.2 The label types

```go
type Ter interface {
	T(ctx context.Context, args ...any) string
}
type Ker interface {
	K() string
}
type TKer interface {
	Ter
	Ker
}

// Tfunc adapts a function to Ter.
type Tfunc func(ctx context.Context, args ...any) string
```

`Ter` is the label type for UI code: anything that renders a string in a
context. Translated keys, static values and computed labels all implement it.

### 3.3 Adapters

```go
func Static(s string) Ter                     // s, never translated
func StaticOptional(s *string) Ter            // *s, or NoneString when nil
func StaticInt(n int) Ter
func StaticInt32(n int32) Ter
func StaticInt64(n int64) Ter
func StaticFloat64(f float64, fmt byte, prec, bitSize int) Ter
func StaticBool(b bool) Ter
func StaticDate(t time.Time) Ter              // time.DateOnly
func StaticStringer(s fmt.Stringer) Ter       // replaces zp's StaticDecimal
func StaticOptionalInt(n *int, none string) Ter
func StaticOptionalInt32(n *int32, none string) Ter
func StaticOptionalInt64(n *int64, none string) Ter
func StaticOptionalFloat64(f *float64, fmt byte, prec, bitSize int, none string) Ter

var NoneString = " - "

func Dynamic(key string) Ter     // T(ctx, key) at render time
func DynamicWrap(key string) TKer // a handle for a key known only at runtime
```

`Dynamic` and `DynamicWrap` look up a key by name when the name is only known at
runtime, such as a journal that stores the key of its name. The key itself is
still registered with `Key` somewhere, which is what makes it editable and
keeps it out of the orphans. Text that people create as data (a category's
name, say) isn't a translation key. Keys stored in data are permanent:
renaming one is a new key ([roadmap](roadmap.md#renaming-keys)).

zp's `StaticDecimal` and `StaticDecimalOtional` stay in the app, because they
pull in shopspring/decimal. A `decimal.Decimal` works with `StaticStringer`.

### 3.4 Generated labels: the `labels` generator

Generated code declares keys like hand-written code, through a trivial
generator that the neozp generation driver,
[genx](https://github.com/epistemer/genx), runs. An app switches it on in
its `genx.yaml`:

```yaml
use:
  - github.com/epistemer/i18n: [labels]
```

It reads every `labels/v1` file in the app (genx's interface §11: a key,
the English text, and optionally a `var` and a `path` for each label) and
writes one file per package, `<dir>/zz_<package>_i18n.go`:

```go
// Code generated from labels by github.com/epistemer/i18n. DO NOT EDIT.

package settings

import "github.com/epistemer/i18n"

var (
	L10nServer     = i18n.Key("settings.server", "HTTP server")
	L10nServerAddr = i18n.Key("settings.server.addr", "Listen address")
)

// l10n holds every label that has a path.
var l10n = map[string]i18n.TKer{
	"server":          L10nServer,
	"server.addr":     L10nServerAddr,
	"log.format.json": i18n.Key("settings.log.format.json", "json"),
}

// L10n returns the label at path, or nil when there is none.
func L10n(path string) i18n.TKer { return l10n[path] }
```

- A label with a `var` is a variable of that name. One with only a `path`
  is declared in `l10n`. One with neither is still declared, as `_`, so
  it's editable and not an orphan.
- `l10n` and `L10n(path)` appear only when a label of the package has a
  path. Generic code, such as a settings page, finds a label by the path
  its data gives.
- Labels keep their order: files in path order, then each file's order.
- The generator stops the run, naming the producers involved, when one key
  has two texts anywhere in the app, when a package declares a `var` or a
  `path` twice, or when one directory gets two package names.

Keys from labels files follow §3.1 like any other: they're permanent, and
their texts are the English that people translate.

---

## 4. Setting up

### 4.1 `New` and `Load`

```go
func New(db *bbolt.DB, source Locale, cfg Config, opts ...Option) *Translator

func (t *Translator) Load() error

type Config interface {
	I18nDefaultLocale() string // code of the default locale; "" means the source
}

func WithLogger(l *slog.Logger) Option
func WithBucket(name string) Option // top-level bucket; default "i18n"
func WithCookie(name string) Option // locale cookie; default "i18n-language"
```

- `db` is shared and never closed by the library. `New` doesn't touch it.
- `source` is the language the code's defaults are written in. Its code must
  stay the same for the life of the store. `Load` stores its record only if
  the store has none, so an edited native name or flag survives (§6.3).
- `cfg` is read on every request, so a Mutable default locale takes effect
  immediately.
- Call `Load` once at startup, before serving. It syncs the registry into the
  store (§9). Several Translators may share one process, for example in tests.
- Every method is safe for concurrent use. A `Translator` keeps no in-memory
  copy of the store. The only thing it holds in memory is the frozen registry.

### 4.2 Config through cfgx

The library reads one setting through `Config`. `i18nschema` declares it the
same way in every app:

```go
package i18nschema

func Group() *dsl.GroupDef {
	return dsl.Group("i18n", "Translations",
		dsl.String("default_locale",
			"Locale for visitors without a chosen locale; empty means the source locale").
			Mutable(),
	).GoName("I18n")
}
```

That gives `i18n.default_locale`, `APP_I18N_DEFAULT_LOCALE`,
`--i18n-default-locale` and `I18nDefaultLocale()`. Because the key is Mutable,
a settings dashboard can change a deployment's default locale without a
restart.

### 4.3 Wiring

```go
// internal/settings/schema
var I18n = i18nschema.Group()
var Bolt = dsl.Path("bolt", "Store file").Default("app.db")
var App = dsl.App("zp", "...", I18n, dsl.Group("storage", "Storage", Bolt) /* , ... */).Store(Bolt)
var Serve = dsl.Command("Serve").Uses(I18n /* , ... */)

// the app
var English = i18n.Locale{Code: "en", NativeName: "English", FlagCode: "us"}

settings.ServeCommand(&cobra.Command{Use: "serve"}, func(ctx context.Context, s *settings.Serve) error {
	tr := i18n.New(s.Store().DB(), English, s, i18n.WithLogger(logger))
	if err := tr.Load(); err != nil {
		return err
	}
	dashboard := tr.Middleware(i18n.IgnoreAcceptLanguage())
	api := tr.Middleware()
	// echo: g.Use(echo.WrapMiddleware(dashboard))
	...
})
```

---

## 5. Resolving text

### 5.1 Resolution order

`T` for key `k` in locale `L`, with source locale `S`:

1. the override of `k` in `L`;
2. the override of `k` in `L`'s base language, if that's another existing
   locale (`ar` for `ar-EG`);
3. the override of `k` in `S`, if `L` isn't `S`;
4. the code's text: the registered text, or the catalog's last text for an
   orphaned key (§9);
5. the key itself, for a `Dynamic` key the catalog has never seen.

An untranslated key in Arabic therefore shows the deployment's English,
including any white-label edit, not just the code's English. That's one to
three bbolt reads in a single read transaction.

An override skipped by §5.2 counts as missing, and the next step applies. If
the read itself fails (the database was closed, say), `T` returns the code's
text and logs the error, at most once a minute.

### 5.2 Formatting

- With no `args`, the text is returned as is. A `%` is just a character.
- With `args`, the text goes through `fmt.Sprintf(text, args...)`. An empty
  text stays empty.
- An argument that implements `Ter` is rendered with the same context before
  the text is formatted, so `Hello.T(ctx, settings.L10nServer)` shows the
  label in the request's locale. With no translator in the context, that's
  the label's code text. The label renders with no arguments of its own.
  Labels render before `T` reads the store and the text is formatted after,
  so no code from an argument runs inside `T`'s read transaction. `Ter` wins
  over `error`, `fmt.Stringer` and `fmt.Formatter`, and a nil pointer or
  `Tfunc` renders as `<nil>`. The fit rules below don't change: the argument is a string by the
  time the text is formatted, so the code's text uses `%s` or `%v` for it.
- An override **fits** the code's text when every argument it uses has the
  same verb as in the code's text. When the code's text has verbs, a literal
  `%` is written `%%`. Explicit indexes (`%[2]s`) let a translation reorder
  arguments or leave one out: `fmt` reports unused arguments only when no
  index is used. When the code's text has no verbs, any text fits.
- `Set` (§8.1) rejects an override that doesn't fit.
- The code can change after an override was saved, for example from
  `"Welcome %s"` to `"Welcome %s (%s)"`. `T` checks the fit whenever `args`
  are passed, and skips an override that no longer fits (§5.1), so the page
  never shows `%!(EXTRA …)`. `Load` logs every override that no longer fits,
  and `Entry.Misfit` (§8.2) shows them to editors.

### 5.3 The context

```go
func WithContext(ctx context.Context, t *Translator, l Locale) context.Context
func FromContext(ctx context.Context) *Translator // nil if none
func Code(ctx context.Context) Locale              // the locale in ctx; the default locale if none; Locale{} without a translator
func Codes(ctx context.Context) []Locale           // every locale, sorted by code; nil without a translator
func T(ctx context.Context, key string, args ...any) string // a key by name (§5.1)
```

- `TKer.T(ctx)` with no translator in `ctx` returns the code's text. That's
  what tests and early startup code see.
- Outside a request, for example an email to a merchant in the language the
  merchant registered with, attach a locale yourself. `Match` (§6.2) picks the
  closest existing locale for a code:

```go
ctx = i18n.WithContext(ctx, tr, tr.Match(merchant.RegistrationLocale))
subject := L10nAcceptedSubject.T(ctx)
```

---

## 6. Locales

### 6.1 `Locale`

```go
type Locale struct {
	Code       string // BCP 47 tag, e.g. "ar", "en", "ar-EG"
	NativeName string // "العربية (مصر)"
	FlagCode   string // for UIs, e.g. "eg"; optional
	RTL        bool
}
```

`Locale` is a plain value: you can copy it and compare it with `==`. Codes are
validated as BCP 47 syntax (letters, digits and `-`). Every code that enters
the API (`CreateLocale`, `Set`, `Locale`, `Match`, cookies, headers, config)
is converted to canonical case first: language lowercase, script title case,
region uppercase (`AR-eg` → `ar-EG`). Codes are stored only in that form, so
two spellings can never name two locales.

### 6.2 Source and default

```go
func (t *Translator) Source() Locale  // the source locale
func (t *Translator) Default() Locale // cfg.I18nDefaultLocale() if it names a locale, else Source()
func (t *Translator) Match(code string) Locale // exact code, then its base language, then Default()
```

If the configured default names no locale, `Load` logs a warning. Requests then
get the source locale until the locale exists or the setting changes.

### 6.3 Managing locales

```go
func (t *Translator) Locales() []Locale             // sorted by code; the source is always present
func (t *Translator) Locale(code string) (Locale, bool)
func (t *Translator) CreateLocale(l Locale) error   // ErrInvalidCode, ErrLocaleExists
func (t *Translator) UpdateLocale(l Locale) error   // native name, flag, RTL; ErrUnknownLocale
func (t *Translator) DeleteLocale(code string) error // drops its overrides; ErrSourceLocale, ErrUnknownLocale
```

- A new locale starts empty. Every key shows through the resolution order
  until someone translates it. Nothing is copied.
- The source locale can be updated but not deleted.
- Deleting the configured default locale is allowed. Visitors then get the
  source locale, and `Default()` reflects that.

---

## 7. HTTP

```go
func (t *Translator) Middleware(opts ...MiddlewareOption) func(http.Handler) http.Handler
func IgnoreAcceptLanguage() MiddlewareOption

func (t *Translator) ParseAcceptLanguage(header string) Locale
func (t *Translator) Cookie(code string) (*http.Cookie, error)          // ErrUnknownLocale
func (t *Translator) SetCookie(w http.ResponseWriter, code string) error // http.SetCookie(w, Cookie(code))
func (t *Translator) ClearCookie(w http.ResponseWriter)                  // back to Accept-Language and the default
```

The middleware picks the request's locale, then calls `WithContext`:

1. the locale cookie, if it names a locale;
2. the `Accept-Language` header, unless `IgnoreAcceptLanguage` is set. Tags are
   tried in order of `q`: first an exact match, then the tag's base language
   (`ar-EG` → `ar`). When the header is consulted, the response gets
   `Vary: Accept-Language`;
3. `Default()`.

`IgnoreAcceptLanguage` is for surfaces where the deployment's choice should
beat the browser's, such as a dashboard white-labeled in Arabic for staff with
English browsers. An API for mobile apps normally keeps the header.

`Cookie` builds the locale picker's cookie: one year, `Path=/`,
`SameSite=Lax`, readable by scripts. An app that needs `Domain` or `Secure`
sets them on the returned cookie before writing it; `SetCookie` is the common
case. `ParseAcceptLanguage` applies step 2 to a header value and falls back to
`Default()`. For a single stored code, use `Match`.

---

## 8. Overrides: the editor's API

Who may edit, and the editing UI, are out of scope. These are the operations a
UI calls.

### 8.1 Writing

```go
func (t *Translator) Set(code, key, text string) error   // ErrUnknownLocale, ErrUnknownKey, ErrFormat
func (t *Translator) Reset(code, key string) (string, error) // removes the override; returns the text now shown
```

- `Set` stores `text` as the override of `key` in `code`. It records the source
  text the override was based on: the effective source text (§5.1, steps 2–3)
  for other locales, the code's text for the source locale. That record is what
  "needs review" will build on ([roadmap](roadmap.md)).
- `Set` accepts registered keys only. Orphaned keys can be read but not edited.
- `Reset` is zp's "restore default". With the override gone, the resolution
  order applies again.
- Each write is one bbolt transaction (about 2.4 ms with fsync).

### 8.2 Entries

```go
type Entry struct {
	Key      string
	Text     string // what T shows in this locale (§5.1), unformatted
	Source   string // what T shows in the source locale
	Override bool   // a person set this value in this locale
	Base     string // the source text the override was based on; "" without one
	Misfit   bool   // the override no longer fits the code's text, so T skips it (§5.2)
	Meta     []byte // the key's meta
}
```

An entry with `Override == false` is untranslated (in another locale) or
unedited (in the source locale).

### 8.3 Searching

```go
type Query struct {
	Prefix   string            // keys starting with this
	Contains string            // case-insensitive, in the key, Text or Source; "" matches all
	Match    func(Entry) bool  // optional; applied before Offset and Limit
	Offset   int
	Limit    int               // 0 means no limit
}

func (t *Translator) Search(code string, q Query) ([]Entry, error) // sorted by key; orphans excluded
```

`Search` runs in one read transaction and walks the catalog once, applying
`Offset` and `Limit` as it goes. A read transaction doesn't block edits.

This one call covers both of zp's needs:
- **The editor:** `Contains` set to the search box.
- **Global search over routes:** `Prefix: "route."`, with `Match` decoding
  `Meta` and filtering, for example to searchable routes only.

### 8.4 Orphans

```go
func (t *Translator) Orphans() ([]string, error)       // keys gone from the code, sorted
func (t *Translator) Prune() (removed []string, err error) // deletes orphans and all their overrides
```

`Orphans` is the preview: `Prune` deletes exactly what it lists, unless a
`Load` ran in between.

---

## 9. Startup sync

`Load` runs in one write transaction:

1. Checks the registry: empty keys, and the same key with different texts or
   meta. Every problem is reported at once, and nothing is written.
2. Stores the source locale's record if absent.
3. Refreshes the catalog: for each registered key, its current code text and
   meta, and clears its orphan mark. An older release's `Load` writes its own
   texts back; that's harmless, because registered keys never read text from
   the catalog.
4. Marks every catalog key that's no longer registered as orphaned. Nothing is
   deleted.
5. Warns about the configured default locale if it doesn't exist, and about
   every override that no longer fits its code's text (§5.2).

Overrides are never touched. A rollback followed by a roll forward leaves every
override in place. A `Dynamic` key that's no longer registered keeps rendering
its last text.

---

## 10. Tests

```go
package i18ntest

func New(tb testing.TB, source i18n.Locale, opts ...i18n.Option) *i18n.Translator // temp bbolt file, loaded; closed by tb.Cleanup
func Context(tb testing.TB, t *i18n.Translator, code string) context.Context      // ctx carrying t and that locale; fails tb if it doesn't exist
```

Both use the process's registry, so a component test sees the app's real keys.
`i18ntest.New` uses an empty `Config` (the default locale is the source).

---

## 11. Explicitly out of scope

- Who may edit, and the editing UI.
- Translating errors, until the errors redesign.
- Plural forms, locale-aware numbers and dates, translation files in the
  repository, import and export, fallback chains beyond the base language
  ([roadmap](roadmap.md)).
- Reading zp's existing store layout. Moving zp's data is a one-off migration
  for the app, written against the layout in implementation.md.
- Reading an app's schema. The packages that own tables, enums, routes and
  settings write `labels/v1` files; i18n only turns them into keys (§3.4).

---

## 12. Former open questions (resolved 2026-11-12)

| # | Question | Resolution |
|---|----------|------------|
| 1 | Should the source locale's record (native name, flag) follow the code, or the store once created? | The store, like every other edit (§4.1) |
| 2 | May `Set` store an empty override, for example to hide a line in one deployment? | Yes: `""` is a value, stays empty even with `args` (§5.2), and `Reset` removes it |
| 3 | Should the default locale be Mutable in `i18nschema`? | Yes, so white-labeling can change it at runtime |

---

## 13. Amendments

| Date | Change |
|------|--------|
| 2026-11-12 | From [review 0001](reviews/0001-gemini-design-review.md): base-language step in §5.1; overrides that no longer fit are skipped, and fit allows indexed omission (§5.2, §8.2, §9); empty text stays empty; canonical locale codes (§6.1); `Match` (§6.2); `Cookie` and `ClearCookie` (§7); D1 names the catalog; single-transaction `Search`; failure behavior of `T`; `Dynamic` keys must be registered. Made in place, before the lock |
| 2026-11-12 | Locked by epistemer, with D1's override-only store and the proposed answers to §12 |
| 2026-11-13 | **The `labels` generator** ([ADR 0003](adr/0003-labels-generator.md)). i18n ships one generator, as data: a metafile and a template that the neozp generation driver runs. It reads every `labels/v1` file in the app and writes `zz_<package>_i18n.go` per package: a variable for each label that names one, and `L10n(path)` over the labels that have a path. It fails when one key has two texts in the app, or when a package declares a variable or a path twice. No new exported Go API and no new dependency (§2, §11). Direction chosen by epistemer: one generation pipeline over shared metadata, in place of each generator writing `i18n.Key` calls from its own template. What the file holds and what's checked decided by Claude; overturnable. In effect from v0.2.0 (§2, §3.4, §11) |
| 2026-11-14 | **Label arguments** (§5.2). An argument that implements `Ter` is rendered with the same context before the text is formatted, so a label passed as an argument shows in the request's locale instead of a struct dump; with no translator, its code text. Arguments render before `T` opens its read transaction and the text is formatted after it, because a nested bbolt read transaction in one goroutine can deadlock behind a waiting writer. Fit rules unchanged. Asked for by the errors package (its interface §9), which doesn't depend on it. Proposed by Claude, approved by epistemer on 2026-11-14. In effect from v0.3.0 |

---

## Appendix: from zp

| zp | Here | Note |
|----|------|------|
| `i18n.I18n` | `i18n.Translator` | Avoids `i18n.I18n` |
| `New(cfgs I18nCfgs, errorRegistry)` | `New(db, source, cfg, opts...)` | DB passed in, not read from config; no error registry |
| `GetDefaultCode()` | `Default()` / `Source()` | The two meanings separated |
| `GetCodes()`, `GetLocaleByCode` | `Locales()`, `Locale(code)` | |
| `CreateCode`, `DeleteCode` | `CreateLocale`, `DeleteLocale` | New locales start empty instead of copying the default bucket |
| `UpdateTranslation` | `Set` | Checks verbs, records the base text |
| `RestoreDefault` | `Reset` | No longer returns meta bytes as text |
| `SearchValues`, `PrefixKeySearch` | `Search(code, Query)` | Also searches keys and source text |
| `*zproto.Locale` | `Locale` | `IsRtl` → `RTL` |
| `Dynamic(key, args...)` | `Dynamic(key)` | zp's `args` were never used |
| `StaticDecimal`, `StaticDecimalOtional` | `StaticStringer`, or the app | |
| `middleware.LocalesMiddleware(i, "ar")` | `tr.Middleware(IgnoreAcceptLanguage())` + config | The hard-coded locale is gone |
| `Key` after `Load` panics on a nil map | `Key` after `Load` panics with a message | `Load` can run more than once per process |
