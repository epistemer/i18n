# i18n

Translations for Go web apps where each deployment owns its text.

Keys are declared in code with their default text. People edit any text at
runtime, the source language included, and an edit always beats the code. So
the library is as much a white-labeling mechanism as a translation library.
Edits live in [bbolt](https://github.com/etcd-io/bbolt) and are read on every
lookup, so they show at once.

> **Status: v0.1.** It works and it's tested on Linux, macOS and Windows. The
> API may still change before v1.0.0.

- **One declaration per text.** `var Welcome = i18n.Key("home.welcome", "Welcome")`
  sits next to its use, and `Welcome.T(ctx)` renders it in the request's
  locale.
- **The database wins.** A person's edit is never overwritten by a code
  default, a sync or an upgrade. Text nobody edited follows the code, so a
  changed default shows everywhere at once.
- **Nothing is lost by deploying.** Keys that leave the code are hidden, not
  deleted. A rollback followed by a roll forward keeps every translation.
- **Safe formatting.** Text is formatted only when arguments are passed, so a
  translator's `50%` stays `50%`. A translation must use the same format verbs
  as the code's text, and a stale one is skipped instead of printing
  `%!(EXTRA …)`.
- **Fast enough to call hundreds of times a page.** A lookup takes about
  1.4 µs, straight from bbolt, with no cache to invalidate.

## Quickstart

You need Go 1.26 or later.

```sh
mkdir hello && cd hello
go mod init example.com/hello
go get github.com/epistemer/i18n@latest go.etcd.io/bbolt
```

Save this as `main.go`:

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/epistemer/i18n"
	bolt "go.etcd.io/bbolt"
)

// Keys are package-level variables with their default text.
var (
	Welcome = i18n.Key("home.welcome", "Welcome")
	Hello   = i18n.Key("home.hello", "Hello, %s!")
)

func main() {
	db, err := bolt.Open("app.db", 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tr := i18n.New(db, i18n.Locale{Code: "en", NativeName: "English"}, nil)
	if err := tr.Load(); err != nil {
		log.Fatal(err)
	}

	// Translations are usually typed by people in an admin page. This seeds
	// Arabic once, when the locale is first created, so later edits survive
	// restarts.
	err = tr.CreateLocale(i18n.Locale{Code: "ar", NativeName: "العربية", RTL: true})
	switch {
	case err == nil:
		for key, text := range map[string]string{"home.welcome": "أهلاً", "home.hello": "مرحباً يا %s!"} {
			if err := tr.Set("ar", key, text); err != nil {
				log.Fatal(err)
			}
		}
	case !errors.Is(err, i18n.ErrLocaleExists):
		log.Fatal(err)
	}

	page := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, Welcome.T(r.Context()))
		fmt.Fprintln(w, Hello.T(r.Context(), "Sara"))
	})
	log.Fatal(http.ListenAndServe("localhost:8080", tr.Middleware()(page)))
}
```

```console
$ go run . &
$ curl localhost:8080
Welcome
Hello, Sara!
$ curl -H "Accept-Language: ar-EG,ar;q=0.9" localhost:8080
أهلاً
مرحباً يا Sara!
```

`ar-EG` matched the `ar` locale through its base language. The middleware
tries the locale cookie first (`SetCookie` writes it for a language picker),
then `Accept-Language`, then the configured default.

## How text is chosen

For a key in a locale, `T` shows the first of:

1. the locale's own edit;
2. the edit in its base language (`ar` for `ar-EG`);
3. the edit in the source language, which is where a deployment's white-label
   changes live;
4. the code's text;
5. for a key looked up by name (`i18n.T(ctx, "some.key")`) that no release
   ever registered, the name itself.

Text is formatted with `fmt` only when arguments are passed. There's no
plural support yet, so prefer texts whose grammar doesn't depend on a count.

## Editing

An admin page is built on a few calls:
- `Search(code, Query)` lists keys with their text, the source text, whether
  they were edited, and what an edit was based on.
- `Set` and `Reset` change one key.
- `CreateLocale`, `UpdateLocale` and `DeleteLocale` manage locales.
- `Orphans` and `Prune` clean up keys the code no longer has.

Who may edit, and the page itself, are up to the app.

## With cfgx

Apps configured with [cfgx](https://github.com/epistemer/cfgx) declare the
library's one setting with `i18nschema.Group()`: the default locale, which can
change at runtime. `i18nschema` is its own module:

```sh
go get github.com/epistemer/i18n/i18nschema@latest
```

The app then shares cfgx's bbolt file, and passes its generated settings as
the config:

```go
var English = i18n.Locale{Code: "en", NativeName: "English"}

tr := i18n.New(s.Store().DB(), English, s) // s is the command's *settings.Serve
```

[examples/basic](examples/basic) is a complete app with an end-to-end test.

## Generated labels

Code generators don't write `i18n.Key` calls. They write their labels as
data, `labels/v1` files, and i18n's `labels` generator turns them into keys.
[genx](https://github.com/epistemer/genx) runs both; switch the generator on
in the app's `genx.yaml`:

```yaml
use:
  - github.com/epistemer/i18n: [labels]
```

For cfgx's settings, add `.Labels()` to the schema's `App` and name cfgx as
a producer: genx's README has the steps. Each package with labels gets
`zz_<package>_i18n.go`, which declares the variable each label names (cfgx
names them after its getters) and `L10n(path)` to find one by name:

```go
settings.L10nServerAddr.T(ctx) // "Listen address", or its translation
```

The generator stops the run when one key has two texts in the app, or a
package declares a name twice, and names the producers involved. Interface
§3.4 has the details.

## Testing

`i18ntest.New(t, source)` (import `github.com/epistemer/i18n/i18ntest`)
gives a loaded Translator on a temporary store, and
`i18ntest.Context(t, tr, "ar")` a context in that locale.

## Documentation

- [docs/interface.md](docs/interface.md): the full API contract.
- [docs/implementation.md](docs/implementation.md): how it's built.
- [docs/roadmap.md](docs/roadmap.md): what's next, including ideas left out
  of v1 (plurals, locale-aware formatting, import and export).
- [CHANGELOG.md](CHANGELOG.md).

Part of the neozp framework, alongside cfgx. License: [MIT](LICENSE).
