# i18n — Implementation

How the locked [interface](interface.md) gets built. Unlike the interface, this
document changes freely; decisions worth keeping get an ADR under `docs/adr/`.

## 1. Packages and files

```
i18n/                    package i18n
  registry.go            the global registry, Key, freezing, registry problems
  key.go                 the key handle (implements TKer), Dynamic, DynamicWrap
  ter.go                 Ter, Ker, TKer, Tfunc, Static* adapters, NoneString
  format.go              verb signatures and the fit check (§5)
  locale.go              Locale, code validation and canonical case, Accept-Language parsing
  translator.go          Translator, New, options, Load
  store.go               bucket layout, record encoding, schema version
  resolve.go             T's resolution order, context helpers (WithContext, Code, Codes, T)
  editor.go              Set, Reset, Search, Orphans, Prune, locale management
  http.go                Middleware, Cookie, SetCookie, ClearCookie
  internal/storepb/      records.proto and its generated Go code
i18nschema/              the cfgx schema fragment (its own module, ADR 0002)
i18ntest/                New and Context for tests
examples/basic/          a net/http app wired with cfgx, run by the e2e tests (its own module)
```

Dependencies: `go.etcd.io/bbolt` (v1.5.0, as cfgx), `google.golang.org/protobuf`.
`i18nschema` adds `github.com/epistemer/cfgx`. Go 1.26, like cfgx.

Planned (roadmap M6, [ADR 0003](adr/0003-labels-generator.md)): the `labels`
generator, a metafile and a template with no Go code. Where it sits in the
module is decided by the neozp generation driver's discovery spike.

## 2. The store

### 2.1 Layout

Everything sits under one top-level bucket, `i18n` by default (`WithBucket`).
The library never touches any other top-level bucket, so it can share cfgx's
file.

| Inside `i18n` | Kind | Key → value |
|---------------|------|-------------|
| `version` | value | schema version, a single byte, `1` |
| `locales` | bucket | canonical code → `LocaleRecord` (protobuf) |
| `catalog` | bucket | key → `CatalogEntry` (protobuf): code text, meta, orphaned |
| `t:<code>` | bucket per locale | key → override text, raw UTF-8 bytes |
| `b:<code>` | bucket per locale | key → `Base` (protobuf): the source text the override was based on, and when it was set |

- A locale exists exactly when its `t:<code>` bucket exists. `CreateLocale`
  creates `locales/<code>`, `t:<code>` and `b:<code>` in one transaction;
  `DeleteLocale` removes all three.
- Codes hold only letters, digits and `-`, so the `t:` and `b:` prefixes can't
  collide with `locales`, `catalog` or `version`.
- Override text is raw so `T` never decodes a message. Everything read only by
  `Load` and the editor is protobuf, so fields can be added later (the
  needs-review state, for one).
- `Load` refuses a store whose `version` is newer than it knows, with an error
  that names both versions. Version 1 needs no migration.
- Every read path tolerates missing buckets, the top one included. Before the
  first `Load`, `T` returns the code's text, `Locales` returns only the source,
  and `Search` returns nothing.
- Nothing read from bbolt leaves a transaction uncopied: strings are converted,
  and byte fields (`meta`) are cloned. `proto.Unmarshal` already copies by
  default, but the clone doesn't depend on that.

### 2.2 Records

```proto
syntax = "proto3";
package i18n.store.v1;
option go_package = "github.com/epistemer/i18n/internal/storepb";

message LocaleRecord {
  string native_name = 1;
  string flag_code   = 2;
  bool   rtl         = 3;
}

message CatalogEntry {
  string text     = 1; // the code's text at the last Load that registered the key
  bytes  meta     = 2;
  bool   orphaned = 3;
}

message Base {
  string text     = 1; // the source text the override was based on
  int64  set_unix = 2; // when Set stored the override
}
```

The generated code is committed, so users never need `protoc`. CI regenerates
it and fails if anything changed. The code generator's version is pinned with a
`tool` directive in go.mod (ADR 0001 records the choice of `protoc` plus
`protoc-gen-go`).

## 3. The registry

```go
type entry struct {
	key, text string
	meta      []byte
	sig       signature // verbs of text, parsed once (§5)
}

var reg struct {
	m        map[string]*entry
	problems []string    // conflicting duplicates, empty keys
	frozen   atomic.Bool
}
```

- `Key` runs during package initialization, which is single-threaded, so the map
  needs no lock. `Key` panics if `frozen` is set.
- A duplicate with the same text and meta returns the existing entry. A
  conflicting one is recorded in `problems`, and the first registration wins
  until `Load` reports it.
- `Load` sets `frozen` and reports `problems` with `errors.Join`, before
  opening a transaction. After that the map is only read, so concurrent `Load`s
  and lookups are race-free.
- The handle is `*key{name string; e *entry}`. `K()` returns `name`, so it
  works whether or not the key is registered. `Key` sets `e`. `DynamicWrap`
  leaves `e` nil, and the resolver looks the name up in `reg.m` (and then in the
  catalog) when it runs.

## 4. Resolution (`T`)

```go
// resolveTx reads within tx, so T (in a View) and Reset (in its Update) share it.
// It returns the text unformatted; resolveCtx formats it after the View.
func (t *Translator) resolveTx(tx *bbolt.Tx, code, name string, e *entry, withArgs bool) (text string, formattable bool)
```

0. Render every argument that implements `Ter` with `ctx` (`renderArgs`, on a
   copy of `args`; no copy when none does). This happens in `resolveCtx`,
   before the `db.View`: a label argument is itself a `T` call with its own
   read transaction, and bbolt can deadlock on a read transaction nested in
   another in one goroutine when a writer waits between them (it waits for
   the outer one, the inner one waits for it, on a remap).
1. Get the translator and locale from `ctx`. With no translator, use the code's
   text (`e.text`, `reg.m[name]`, or `name` itself) and go to step 4.
2. If `e` is nil, look `name` up in `reg.m`. If it's still nil, read
   `catalog/<name>` (an orphan): its text is the code's text, and its signature
   is parsed on the spot. With no catalog entry either, the code's text is
   `name`, with no signature.
3. Look the key up in the `t:` buckets of the locale, its base language and the
   source, in that order, skipping duplicates and buckets that don't exist.
   The first hit wins, unless `args` were passed and the override doesn't fit
   the signature. With no hit, use the code's text.
4. If `args` were passed and the text isn't empty, `fmt.Sprintf(text, args...)`,
   except for a key's own name.

`T` wraps steps 2–3 in one `db.View`, and steps 0 and 4 run outside it, so no
code from an argument (a label's `T`, a `String` or `Error` method) runs
inside the transaction. `Reset` calls `resolveTx` with its own
write transaction after deleting, so it sees its own deletes: a separate `View`
would read the last committed state.

Bucket names (`t:ar-EG`, base `t:ar`) are computed once per locale and cached in
a `sync.Map` keyed by code, so the hot path doesn't allocate them. The value
read from bbolt is copied into a string before the transaction ends.

Read failures: `View` returning an error falls back to the code's text and
logs at Error level through `slog`. An atomic timestamp keeps it to one log
line a minute.

**Budget:** a hit in the request's locale costs one read transaction plus a
`Get` per bucket tried, about 1.3–2.2 µs from the numbers measured on
2026-11-12. A benchmark in `resolve_test.go` checks it. CI only reports it;
it doesn't fail on it.

## 5. The fit check

`format.go` parses a format string the way `fmt` does:

- `%%` is a literal and uses no argument.
- Flags (`+-# 0`), then the width: an optional `[n]` followed by `*`, or
  digits. Then `.` and the precision: an optional `[n]` followed by `*`, or
  digits. Then an optional `[n]` and the verb. That's all three index slots
  `fmt` allows, as in `%[3]*.[2]*[1]f`.
- An explicit index sets the argument counter, as in `fmt`. Each `*` and each
  verb uses the argument at the counter, then advances it.

```go
type signature struct {
	uses      map[int]map[rune]bool // argument index → the verbs it's used with ('*' for a width or precision)
	reordered bool                  // an explicit index appears
	broken    bool                  // %! syntax, a trailing %, an index out of range
}
```

One argument can appear with several verbs, as in `"%[1]d (0x%[1]x)"`, which
is why each index holds a set.

An override **fits** the code text's signature when:
- it isn't `broken`;
- every (argument, verb) pair it uses is one the code uses;
- if it isn't `reordered`, it uses every argument the code uses (otherwise `fmt`
  would print `%!(EXTRA …)`, or the arguments would shift).

When the code's text has no verbs, everything fits (interface §5.2). Fit
ignores flags, width and precision: `%5.2f` fits `%f`.

`FuzzFit` checks the rule against `fmt` itself. For random code texts and
overrides, whenever `fit` says yes, `fmt.Sprintf` with arguments of the right
types must produce no `%!`.

## 6. `Load`

One `db.Update`:

1. Create the top bucket, check `version`, write it if absent.
2. If `locales/<source>` is absent, store the source record and create its
   buckets.
3. For each registered entry, encode its `CatalogEntry` and `Put` it only if
   the bytes differ from what's stored. Re-running `Load` on an unchanged
   store writes nothing.
4. Walk `catalog`. Set `orphaned` on unregistered entries that don't have it
   yet.

After the commit, a separate `db.View` checks each override of a key whose code
text has verbs, in every locale, so the write lock isn't held for read-only
work. Then `Load` logs a warning for each override that no longer fits, and one
if the configured default locale doesn't exist.

With about 1,850 keys, a first `Load` writes about 1,850 small records in one
commit. Later ones write only what changed. Both are checked by a test at that
scale.

## 7. The editor operations

- **`Set`** checks, in one `Update`: the locale exists, the key is registered,
  and the text fits. It computes the base, then writes `t:<code>/<key>` and
  `b:<code>/<key>`. For the source locale, the base is the code's text. For any
  other locale, it's what the locale would show without its own override:
  interface §5.1 steps 2–4 (base language, then source, then code). For
  `ar-EG`, that's the `ar` translation when there is one.
- **`Reset`** deletes both, then calls `resolveTx` with no `args` in the same
  transaction to return the text now shown (§4).
- **`Search`** runs in one `View`. A cursor walks `catalog`. When a prefix is
  given it starts at `Seek(Prefix)` and stops at the first key without the
  prefix. It skips orphans. Each `Entry` is built from the overrides (locale,
  base language, source) and the code's text, taken from `reg.m` like `T`
  does, so the editor can't show a different default than the page.
  `Search` then applies `Contains` (case-folded, against key, `Text` and
  `Source`), `Match`, `Offset` and `Limit`, and stops as soon as `Limit` entries
  are collected.
- **`Orphans`** lists orphaned catalog keys. **`Prune`** deletes them from the
  catalog and from every `t:` and `b:` bucket in one `Update`.
- **Locale operations** canonicalize the code first, then act as §2.1
  describes. `Locales` walks `locales`, which bbolt keeps in byte order.

## 8. Locale codes

`canonical(code) (string, error)` with no dependency:

- Subtags split on `-`. The first subtag is 2–8 letters, or the singleton `x`
  (a private-use tag such as `x-acme`). Every other subtag is 1–8 letters or
  digits.
- Case: the language is lowercase. A 4-letter subtag after the language is
  title case (a script). A 2-letter or 3-digit subtag is uppercase (a region).
  Everything else is lowercase, and so is everything after a singleton
  (`u`, `x`, …), so `en-US-u-ca-buddhist` keeps `ca` lowercase.
- Base language = the first subtag. A private-use tag has none.

`ParseAcceptLanguage` splits on `,` and `;` and trims spaces around every part.
It reads `q` (missing or invalid means 1, and `q=0` means "not acceptable" and
drops the tag), ignores `*` and tags that aren't valid, and stable-sorts by `q`.
For each tag it tries the exact code, then the base language, against existing
locales. With no match, or an empty header, it returns `Default()`. The whole
match uses one `View`.

## 9. HTTP

The middleware resolves the locale in one `View` per request: cookie, then
`Accept-Language` (unless ignored), then the configured default, checking that
each candidate locale exists and reading its record. It sets
`Vary: Accept-Language` whenever the header took part in the choice: no valid
cookie, and `IgnoreAcceptLanguage` not set. That holds even when nothing
matched and the default won, because a cache must not serve that default to a
visitor whose header would match. Then it calls `WithContext` on the request.

`Cookie` returns `&http.Cookie{Name, Value: code, Path: "/", MaxAge: 365 days,
SameSite: Lax}`. `ClearCookie` writes the same name with `MaxAge: -1`.

## 10. Tests

- Table tests for each file. The registry tests run in a child process
  (`go test` re-executing itself), because `Key`'s rules apply only at package
  initialization.
- `-race` on every test, including parallel `Load`s and `T` under concurrent
  `Set`.
- Regression tests for the four zp bugs from 2026-11-12: a changed default
  reaches unedited text; a rollback and roll forward keep overrides; `%` in
  text without `args`; `Reset` on a key with meta returns text only.
- `FuzzFit` (§5).
- Edge cases from the interface:
  - an empty override, with and without `args`, and removed by `Reset`;
  - the orphan lifecycle: rollback, `Orphans`, `Prune` deleting from every
    `t:` and `b:` bucket, and a pruned key coming back;
  - `DynamicWrap` and `T(ctx, name)` on registered, orphaned and unknown keys;
  - every read path on a store that was never loaded;
  - a read failure in `T` (closed database): the code's text, and one log line
    a minute, checked with a fake clock.
- An e2e test that runs `examples/basic` over HTTP: cookie, `Accept-Language`,
  default from config, an edit visible on the next request.
- Benchmarks: `T` hit and miss paths, `Search` over 1,850 keys, and `Load` at
  that scale.
