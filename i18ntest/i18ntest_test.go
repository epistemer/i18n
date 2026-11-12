package i18ntest_test

import (
	"testing"

	"github.com/epistemer/i18n"
	"github.com/epistemer/i18n/i18ntest"
)

var title = i18n.Key("i18ntest.title", "Welcome")

func TestNewAndContext(t *testing.T) {
	tr := i18ntest.New(t, i18n.Locale{Code: "en", NativeName: "English"})
	if err := tr.CreateLocale(i18n.Locale{Code: "ar"}); err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "i18ntest.title", "أهلاً"); err != nil {
		t.Fatal(err)
	}
	if got := title.T(i18ntest.Context(t, tr, "ar")); got != "أهلاً" {
		t.Errorf("ar: %q", got)
	}
	if got := title.T(i18ntest.Context(t, tr, "en")); got != "Welcome" {
		t.Errorf("en: %q", got)
	}
}
