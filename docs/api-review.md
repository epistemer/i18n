# API review before v0.1.0 (M5.2)

Date: 2026-11-12. Every exported identifier of `github.com/epistemer/i18n`,
`i18ntest` and `i18nschema`, checked against the locked
[interface](interface.md) for five things:

1. it's in the interface;
2. its name and signature match;
3. it has a doc comment;
4. misuse fails loudly;
5. it doesn't expose internals.

The listing came from `go doc -short` for each package, and the method set from
`go doc Translator`.

## Result

All the identifiers the interface names exist with the locked signatures. The
code adds nothing the interface doesn't name except the `Entry.Misfit` field,
which amendment §13 (review 0001) added. Every exported identifier has a doc
comment. Nothing blocks v0.1.0.

| Area | Identifiers | Notes |
|------|-------------|-------|
| Keys (§3) | `Key`, `Ter`, `Ker`, `TKer`, `Tfunc`, `Dynamic`, `DynamicWrap` | `Key` after `Load` panics with the key's name; conflicting duplicates fail `Load` all at once |
| Adapters (§3.3) | `Static`, `StaticOptional`, `StaticInt`, `StaticInt32`, `StaticInt64`, `StaticFloat64`, `StaticBool`, `StaticDate`, `StaticStringer`, `StaticOptionalInt`, `StaticOptionalInt32`, `StaticOptionalInt64`, `StaticOptionalFloat64`, `NoneString` | `NoneString` is a mutable package variable, kept from zp. It's read at render time, so set it at startup |
| Setup (§4) | `Translator`, `New`, `(*Translator).Load`, `Config`, `Option`, `WithLogger`, `WithBucket`, `WithCookie` | `Option` exposes an unexported struct in its type, the usual functional-options pattern. An invalid source code fails `Load` |
| Resolution (§5) | `T`, `WithContext`, `FromContext`, `Code`, `Codes` | `Code` and `Codes` return `Locale` values. A reviewer suggested renaming them; the names are the owner's |
| Locales (§6) | the `Locale` type; `(*Translator).Source`, `.Default`, `.Match`, `.Locales`, `.Locale`, `.CreateLocale`, `.UpdateLocale`, `.DeleteLocale` | Every code is canonicalized on the way in |
| HTTP (§7) | `(*Translator).Middleware`, `MiddlewareOption`, `IgnoreAcceptLanguage`, `(*Translator).ParseAcceptLanguage`, `.Cookie`, `.SetCookie`, `.ClearCookie` | `ClearCookie` can't clear a cookie the app gave `Domain` or `Secure`; its doc says so |
| Editing (§8) | `(*Translator).Set`, `.Reset`, `.Search`, `.Orphans`, `.Prune`; the `Entry` and `Query` types | Errors wrap the sentinels below with details |
| Errors | `ErrInvalidCode`, `ErrUnknownLocale`, `ErrLocaleExists`, `ErrSourceLocale`, `ErrUnknownKey`, `ErrFormat` | Compared with `errors.Is` |
| Tests (§10) | `i18ntest.New`, `i18ntest.Context` | |
| Config (§4.2) | `i18nschema.Group` | Its own module (ADR 0002) |

## Known limits, left for after v0.1.0

- **`Cookie` hides store failures.** When the store can't be read, it reports
  `ErrUnknownLocale`, because `Locale` returns a bool (review 0006, F-08).
  Changing that is an amendment.
- **`DynamicWrap`'s parameter is named `name`, and `Dynamic`'s is `key`.**
  Cosmetic; the names don't affect callers.
