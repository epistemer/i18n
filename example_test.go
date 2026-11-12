package i18n_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/epistemer/i18n"
	bolt "go.etcd.io/bbolt"
)

// Keys are package-level variables, declared next to their use with their
// default text in the source language.
var (
	L10nWelcome = i18n.Key("example.welcome", "Welcome")
	L10nHello   = i18n.Key("example.hello", "Hello, %s!")
)

var english = i18n.Locale{Code: "en", NativeName: "English"}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// newTranslator opens a bbolt file and loads a Translator for the examples.
// An app passes cfgx's Store().DB() instead of opening its own file.
func newTranslator() (*i18n.Translator, func()) {
	dir, err := os.MkdirTemp("", "i18n-example")
	must(err)
	db, err := bolt.Open(filepath.Join(dir, "example.db"), 0o600, nil)
	must(err)
	tr := i18n.New(db, english, nil)
	must(tr.Load())
	return tr, func() { db.Close(); os.RemoveAll(dir) }
}

// Translations are edits stored per locale. The code's text shows wherever
// nobody wrote one.
func Example() {
	tr, done := newTranslator()
	defer done()
	must(tr.CreateLocale(i18n.Locale{Code: "ar", NativeName: "العربية", RTL: true}))
	must(tr.Set("ar", "example.welcome", "أهلاً"))

	ctx := i18n.WithContext(context.Background(), tr, tr.Match("ar"))
	fmt.Println(L10nWelcome.T(ctx))
	fmt.Println(L10nHello.T(ctx, "Sara")) // not translated yet: the code's text
	// Output:
	// أهلاً
	// Hello, Sara!
}

// An edit to the source language beats the code too. That's what makes the
// library a white-labeling mechanism. A translation must use the same format
// verbs as the code's text.
func ExampleTranslator_Set() {
	tr, done := newTranslator()
	defer done()
	ctx := i18n.WithContext(context.Background(), tr, tr.Source())

	must(tr.Set("en", "example.welcome", "Welcome to Acme"))
	fmt.Println(L10nWelcome.T(ctx))

	err := tr.Set("en", "example.hello", "Hello, %d!")
	fmt.Println(errors.Is(err, i18n.ErrFormat))
	// Output:
	// Welcome to Acme
	// true
}

// Reset removes an edit and returns the text that shows instead.
func ExampleTranslator_Reset() {
	tr, done := newTranslator()
	defer done()
	must(tr.Set("en", "example.welcome", "Welcome to Acme"))

	shown, err := tr.Reset("en", "example.welcome")
	must(err)
	fmt.Println(shown)
	// Output: Welcome
}

// The middleware picks each request's locale: the cookie, then
// Accept-Language, then the default.
func ExampleTranslator_Middleware() {
	tr, done := newTranslator()
	defer done()
	must(tr.CreateLocale(i18n.Locale{Code: "ar"}))
	must(tr.Set("ar", "example.welcome", "أهلاً"))

	h := tr.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, L10nWelcome.T(r.Context()))
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "ar-EG, en;q=0.5")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	fmt.Println(rec.Body.String())
	// Output: أهلاً
}
