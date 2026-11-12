# Roadmap

The order of work up to v0.1.0, then the ideas parked for after v1. The
contract is the locked [interface](interface.md); how it's built is in
[implementation.md](implementation.md). Progress is logged in
[status.md](status.md).

## v0.1.0

Each item lands as a PR with green CI and a review by the other model family.
A milestone is done when every item meets its done-when.

### M0. Foundations

- [x] **M0.1** Module `github.com/epistemer/i18n` (Go 1.26), package docs, the
  layout of implementation §1 with empty packages.
  *Done when:* `go build ./...` and `go vet ./...` pass.
- [x] **M0.2** CI like cfgx's: gofmt, vet, staticcheck, `go test -race` on
  Linux, macOS and Windows with Go 1.26 and 1.27, and a check that generated
  code is current.
  *Done when:* CI runs green on a PR.
- [x] **M0.3** ADR 0001: protobuf generation (`protoc` plus a pinned
  `protoc-gen-go` tool), `records.proto` and its generated code.
  *Done when:* `go generate ./...` reproduces the committed code.

### M1. The read path

- [x] **M1.1** `format.go`: signatures and fit, with `FuzzFit` (implementation §5).
  *Done when:* table tests cover every rule in interface §5.2, including
  reused arguments and all three index slots, and the fuzzer runs 10 minutes
  clean.
- [x] **M1.2** `locale.go`: `Locale`, canonical codes, base language,
  Accept-Language parsing (implementation §8).
  *Done when:* table tests cover case, scripts, regions, singletons, invalid
  codes, whitespace, `q=0` and `*` in headers.
- [x] **M1.3** `ter.go`: `Ter`, `Ker`, `TKer`, `Tfunc`, the `Static*`
  adapters, `NoneString`.
  *Done when:* each adapter has a test, including nil pointers.
- [x] **M1.4** `registry.go`, `key.go`: `Key`, freezing, registry problems,
  `Dynamic`, `DynamicWrap` (implementation §3).
  *Done when:* child-process tests show that `Key` after `Load` panics with the
  key's name, conflicting duplicates fail `Load` all at once, and identical
  duplicates pass.
- [x] **M1.5** `store.go`, `translator.go`: layout, records, version check,
  `New`, options, `Load` (implementation §2, §6).
  *Done when:* tests show `Load` writes nothing on an unchanged store, marks
  and unmarks orphans, refuses a newer version, and stores the source record
  only once. Overrides are written straight into the buckets, since `Set`
  comes in M2.
- [x] **M1.6** `resolve.go`: `resolveTx`, the resolution order, failure
  behaviour and context helpers (implementation §4).
  *Done when:* table tests cover every step of interface §5.1 using
  hand-written buckets; `-race` passes with concurrent `T` and `Load`; the
  benchmark is within the §4 budget.

### M2. Locales and editing

- [x] **M2.1** Locale management: `Locales`, `Locale`, `CreateLocale`,
  `UpdateLocale`, `DeleteLocale`, `Match`, and `Codes(ctx)`, which needs
  `Locales`. (`Source` and `Default` landed in M1, because resolution needs
  them.)
  *Done when:* tests cover canonical codes on every entry point, the source
  locale's protections, and deleting the configured default.
- [x] **M2.2** `Set` and `Reset`, with base records and the fit check.
  *Done when:* the four zp regression tests pass (implementation §10),
  `-race` passes with concurrent `T`, `Set` and `Reset`, and a misfit `Set` is
  rejected with `ErrFormat`.
- [x] **M2.3** `Search`, `Orphans`, `Prune`, and `Entry.Misfit`.
  *Done when:* the orphan lifecycle test passes; a route-style `Prefix` plus
  `Match` query returns only matching keys; `Search` over 1,850 keys is
  benchmarked.

### M3. HTTP

- [x] **M3.1** `Middleware` with `IgnoreAcceptLanguage`, `Vary`, and one read
  transaction per request.
  *Done when:* `httptest` cases cover cookie, header, default and the
  ignore option, and `Vary` is set exactly when implementation §9 says.
- [x] **M3.2** `Cookie`, `SetCookie`, `ClearCookie`.
  *Done when:* tests check every cookie attribute, and that an unknown locale
  returns `ErrUnknownLocale`.

### M4. Integration

- [ ] **M4.1** `i18nschema`: the fragment, with a test that generates a cfgx
  app from it and checks the getter name.
  *Done when:* the generated app's settings type satisfies `i18n.Config`.
- [ ] **M4.2** `i18ntest`: `New` and `Context`.
  *Done when:* the e2e test and at least one package test use it.
- [ ] **M4.3** `examples/basic`: a net/http app on cfgx, plus the e2e test
  (implementation §10).
  *Done when:* the e2e test passes in CI on all three operating systems.

### M5. Release

- [ ] **M5.1** Godoc examples, and a README with a quickstart run step by step.
  *Done when:* the examples run as tests, and the quickstart's output is pasted
  from a real run.
- [ ] **M5.2** API review of every exported identifier against the interface.
  *Done when:* a written review lists each identifier and how it was checked,
  and its findings are fixed or filed.
- [ ] **M5.3** CHANGELOG and the `v0.1.0` tag.
  *Done when:* `go get github.com/epistemer/i18n@v0.1.0` works through the
  module proxy.

### Working with agents

Claude owns the cross-cutting core: M1.4–M1.6, M2.2, M2.3, M3.1, and the
reviews of Gemini's work. Gemini, through agy, takes leaf items with a crisp
done-when: M0.2, M0.3, M1.1–M1.3, M2.1, M3.2, M4.1–M4.3, M5.1. Each delegated
item runs in its own git worktree outside the repository, for example
`../i18n-worktrees/`.

Model choice follows the global rules: Argon (High) for the pre-merge review
of each milestone, Argon (Medium) or Flash (High) for well-specified code,
Flash (Medium) for mechanical work. When a model is cooling down, the strongest
available one runs instead, and the status log says which.

## Later

### "Needs review" for translations whose source changed

- **Why:** when a key's English text changes, translations made from the old
  text go stale without anyone noticing.
- **Sketch:** v1 already stores, per key and locale, the source text each value
  was based on. A value whose base differs from today's source needs review.
  Editing or confirming it clears that state.
- **Open:** whether every source change should trigger a review (a typo fix
  shouldn't send every locale back), and how a translator confirms a value
  without editing it.

### Plural forms

- **Why:** text with a count ("3 transactions") needs each language's plural
  rules. Arabic has six CLDR categories (zero, one, two, few, many, other), and
  `Sprintf` can't express them. zp has no counted text yet.
- **Sketch:** a plural key that stores one value per CLDR category, picked by
  the count when `T` runs. The rules would come from
  `golang.org/x/text/feature/plural` or generated CLDR tables.

### Locale-aware numbers and dates

- **Why:** zp's `StaticDate` always prints `2006-01-02`, and numbers always use
  Western digits and separators, whatever the locale. Month and day names
  already exist as keys (`component.form.months.*`, `component.form.days.*`).
- **Sketch:** formatting helpers that read the locale from the context, for
  example through `golang.org/x/text/number`. The locale record would also hold
  its numbering system and date patterns.

### Translation files in the repository

- **Why:** translations exist only in the production database. A new
  deployment starts in English, and a lost database loses every translation.
- **Sketch:** committed per-locale files, applied at startup. They may only fill
  values nobody has edited, so the database still wins. The per-value record v1
  keeps makes that check possible.

### Import and export

- **Why:** to hand strings to a translator, or to move translations between
  deployments.
- **Sketch:** JSON first, then PO, XLIFF or CSV if a translator needs one.
  Imports follow the same rule as committed files: edited values win. A dry run
  shows what would change.

### Errors

- **Why:** this waits on the errors redesign. In zp, `errors.T` needs an exact
  `err.Error()` match, so a wrapped error shows "unknown error occurred". The
  API also returns the English `err.Error()` whatever language the client asks
  for.
- **Sketch:** errors that carry their own key (implement `Ter`), found by
  walking the wrap chain. The API would return the translated message plus the
  stable key.

### Language matching

- **Why:** v1 matches tags exactly (ignoring case), then by base language
  (`ar-EG` → `ar`). Nothing goes past one level, as in `ar-EG → ar → en`, and
  related tags such as `zh-TW` and `zh-Hant` aren't treated as matches.
- **Sketch:** `golang.org/x/text/language`'s matcher, plus per-locale fallback
  chains.

### Named placeholders

- **Why:** `%s` and `%[2]d` don't tell a translator what goes there.
- **Sketch:** placeholders like `{amount}` filled from named arguments, and a
  save that fails when a translation's placeholders differ from the source's.

### Fewer read transactions

- **Why:** measured with bbolt v1.5.0 on 4 cores: a lookup in its own read
  transaction takes about 1.3 µs, and the transaction is about 0.9 µs of that.
  Read transactions also serialize on bbolt's meta lock, so extra cores barely
  add throughput (about 10% more going from 1 to 4). None of this matters at zp's
  scale, where a page with 200 labels spends about 0.3 ms in lookups.
- **Sketch:** one read transaction per request, opened on first use and closed
  when the request ends. Worth it only if lookups ever show up in a profile.

### More than one app instance

- **Why:** bbolt allows one process per file, so translations live in one
  instance's file.
- **Sketch:** a shared store, or replication of edits, if the framework ever
  runs replicas.

### Renaming keys

- **Why:** keys are permanent because stored data refers to them. zp's
  journals store `acounting.gl.*` keys, typo included.
- **Sketch:** a migration that moves a key's translations and edit records to a
  new key. The app runs it alongside its own data migration.
