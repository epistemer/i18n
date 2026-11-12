# Roadmap

The v1 milestones will be written together with the design. For now this file
holds what v1 deliberately leaves out: ideas that came up while studying zp's
i18n (master, 2026-11-12), parked for later. Each entry says why the idea came
up, what it might look like, and what's still open.

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
