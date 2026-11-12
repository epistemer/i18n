package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epistemer/i18n"
	"github.com/epistemer/i18n/i18ntest"
)

// A handler test that needs no cfgx: i18ntest provides the Translator.
func TestHandler(t *testing.T) {
	tr := i18ntest.New(t, i18n.Locale{Code: "en", NativeName: "English"})
	if err := tr.CreateLocale(i18n.Locale{Code: "ar"}); err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "basic.greeting", "مرحباً يا %s!"); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(i18ntest.Context(t, tr, "ar"), http.MethodGet, "/?name=Sara", nil)
	Handler(tr).ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Body)
	if got, want := string(body), "Welcome\nمرحباً يا Sara!\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
