package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/epistemer/cfgx"
	"github.com/epistemer/i18n"
	"github.com/epistemer/i18n/examples/basic/internal/app"
	"github.com/epistemer/i18n/examples/basic/internal/settings"
)

// TestE2E runs the example over HTTP, with its settings loaded through cfgx's
// real pipeline: the cookie, Accept-Language, the default from config, an edit
// visible on the next request, and the default changed at runtime.
func TestE2E(t *testing.T) {
	ctx := context.Background()
	s, err := settings.LoadServe(ctx, cfgx.WithEnv(map[string]string{
		"BASIC_STORAGE_BOLT":        filepath.Join(t.TempDir(), "basic.db"),
		"BASIC_I18N_DEFAULT_LOCALE": "ar",
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tr, err := Setup(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "basic.title", "أهلاً"); err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "basic.greeting", "مرحباً يا %s!"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(tr.Middleware()(app.Handler(tr)))
	defer srv.Close()

	get := func(path, header, cookie string) (string, int, []*http.Cookie) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if header != "" {
			req.Header.Set("Accept-Language", header)
		}
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "i18n-language", Value: cookie})
		}
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return string(body), res.StatusCode, res.Cookies()
	}

	const arabic = "أهلاً\nمرحباً يا visitor!\n"
	const english = "Welcome\nHello, visitor!\n"
	for _, tt := range []struct{ name, header, cookie, want string }{
		{"the configured default", "", "", arabic},
		{"Accept-Language", "en-GB, ar;q=0.5", "", english},
		{"the cookie beats the header", "ar", "en", english},
	} {
		if got, _, _ := get("/", tt.header, tt.cookie); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}

	_, status, cookies := get("/locale/en", "", "")
	if status != http.StatusSeeOther || len(cookies) != 1 || cookies[0].Value != "en" {
		t.Fatalf("/locale/en: status %d, cookies %v", status, cookies)
	}
	if got, _, _ := get("/", "ar", cookies[0].Value); got != english {
		t.Errorf("the picked locale applies on the next request: %q", got)
	}

	if err := tr.Set("ar", "basic.title", "أهلاً وسهلاً"); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := get("/", "", ""); got != "أهلاً وسهلاً\nمرحباً يا visitor!\n" {
		t.Errorf("an edit shows on the next request: %q", got)
	}

	// A setting's label is a translation key too, generated through genx:
	// a settings page shows it in the visitor's locale.
	if err := tr.Set("ar", "settings.server.addr", "عنوان الاستماع"); err != nil {
		t.Fatal(err)
	}
	ar := i18n.WithContext(ctx, tr, tr.Match("ar"))
	if got := settings.L10nServerAddr.T(ar); got != "عنوان الاستماع" {
		t.Errorf("translated settings label: %q", got)
	}
	if got := settings.L10n("server.addr").T(i18n.WithContext(ctx, tr, tr.Match("en"))); got != "Listen address" {
		t.Errorf("settings label by path, in English: %q", got)
	}

	err = s.Update(ctx, func(u *settings.Update) error {
		u.SetI18nDefaultLocale("en")
		return nil
	}, cfgx.By("e2e"), cfgx.Reason("white-label switch"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _, _ := get("/", "", ""); got != english {
		t.Errorf("after changing the Mutable default locale: %q", got)
	}
}
