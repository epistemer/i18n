package i18n

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// httpSetup has en (source), ar and fr, with ar as the configured default.
func httpSetup(t *testing.T) *Translator {
	t.Helper()
	db := testDB(t)
	tr := New(db, english, cfg("ar"), withRegistry(release()))
	mustLoad(t, tr)
	for _, c := range []string{"ar", "fr"} {
		if err := tr.CreateLocale(Locale{Code: c}); err != nil {
			t.Fatal(err)
		}
	}
	return tr
}

// serve runs one request through the middleware and returns the locale the
// handler saw and the response.
func serve(t *testing.T, tr *Translator, req *http.Request, opts ...MiddlewareOption) (string, *httptest.ResponseRecorder) {
	t.Helper()
	var seen string
	h := tr.Middleware(opts...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = Code(r.Context()).Code
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return seen, rec
}

func TestMiddleware(t *testing.T) {
	tr := httpSetup(t)
	for _, tt := range []struct {
		name, cookie, header string
		ignore               bool
		want                 string
		vary                 bool
	}{
		{"cookie wins", "fr", "ar", false, "fr", false},
		{"cookie in another case", "FR", "", false, "fr", false},
		{"unknown cookie is ignored", "de", "fr", false, "fr", true},
		{"header", "", "de, fr-CH;q=0.9", false, "fr", true},
		{"nothing matches: the default", "", "de, ja", false, "ar", true},
		{"no header: the default", "", "", false, "ar", true},
		{"ignored header: the default", "", "fr", true, "ar", false},
		{"ignored header: the cookie still wins", "en", "fr", true, "en", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "i18n-language", Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set("Accept-Language", tt.header)
			}
			var opts []MiddlewareOption
			if tt.ignore {
				opts = append(opts, IgnoreAcceptLanguage())
			}
			got, rec := serve(t, tr, req, opts...)
			if got != tt.want {
				t.Errorf("locale = %q, want %q", got, tt.want)
			}
			if vary := rec.Header().Get("Vary") == "Accept-Language"; vary != tt.vary {
				t.Errorf("Vary set = %v, want %v", vary, tt.vary)
			}
		})
	}
}

func TestMiddleware_OneReadPerRequest(t *testing.T) {
	tr := httpSetup(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "de, ja, fr")
	h := tr.Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	before := tr.db.Stats().TxN
	h.ServeHTTP(httptest.NewRecorder(), req)
	if n := tr.db.Stats().TxN - before; n != 1 {
		t.Errorf("the middleware opened %d read transactions, want 1", n)
	}
}

func TestMiddleware_WithCustomCookieName(t *testing.T) {
	tr := New(testDB(t), english, nil, withRegistry(release()), WithCookie("lang"))
	mustLoad(t, tr)
	_ = tr.CreateLocale(Locale{Code: "ar"})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "lang", Value: "ar"})
	if got, _ := serve(t, tr, req); got != "ar" {
		t.Errorf("locale = %q", got)
	}
}

func TestCookies(t *testing.T) {
	tr := httpSetup(t)
	c, err := tr.Cookie("AR")
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "i18n-language" || c.Value != "ar" || c.Path != "/" || c.MaxAge != 365*24*60*60 ||
		c.SameSite != http.SameSiteLaxMode || c.HttpOnly || c.Secure {
		t.Errorf("cookie = %+v", c)
	}
	if _, err := tr.Cookie("de"); !errors.Is(err, ErrUnknownLocale) {
		t.Errorf("Cookie(unknown) = %v", err)
	}
	if _, err := tr.Cookie("not valid"); !errors.Is(err, ErrUnknownLocale) {
		t.Errorf("Cookie(invalid) = %v", err)
	}

	rec := httptest.NewRecorder()
	if err := tr.SetCookie(rec, "fr"); err != nil {
		t.Fatal(err)
	}
	if got := rec.Result().Cookies(); len(got) != 1 || got[0].Value != "fr" {
		t.Errorf("SetCookie wrote %+v", got)
	}
	if err := tr.SetCookie(httptest.NewRecorder(), "de"); !errors.Is(err, ErrUnknownLocale) {
		t.Errorf("SetCookie(unknown) = %v", err)
	}

	rec = httptest.NewRecorder()
	tr.ClearCookie(rec)
	if got := rec.Result().Cookies(); len(got) != 1 || got[0].MaxAge != -1 || got[0].Name != "i18n-language" {
		t.Errorf("ClearCookie wrote %+v", got)
	}
}

func TestMiddleware_MultipleHeaderLines(t *testing.T) {
	tr := httpSetup(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("Accept-Language", "de;q=0.2")
	req.Header.Add("Accept-Language", "fr;q=1.0")
	if got, _ := serve(t, tr, req); got != "fr" {
		t.Errorf("locale = %q, want fr from the second line", got)
	}
}

func TestMiddleware_AttachesTranslator(t *testing.T) {
	tr := httpSetup(t)
	var got *Translator
	h := tr.Middleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if got != tr {
		t.Error("the handler's context doesn't carry the Translator")
	}
}

func TestMiddleware_StoreFailure(t *testing.T) {
	tr := httpSetup(t)
	tr.db.Close()
	got, rec := serve(t, tr, httptest.NewRequest(http.MethodGet, "/", nil))
	if got != "en" {
		t.Errorf("with the store closed, locale = %q, want the source", got)
	}
	if rec.Header().Get("Vary") != "Accept-Language" {
		t.Error("Vary must be set even when the store read fails")
	}
}

func TestClearCookie_Wire(t *testing.T) {
	tr := New(testDB(t), english, nil, withRegistry(release()), WithCookie("lang"))
	rec := httptest.NewRecorder()
	tr.ClearCookie(rec)
	raw := rec.Header().Get("Set-Cookie")
	for _, want := range []string{"lang=", "Path=/", "Max-Age=0", "Expires=Thu, 01 Jan 1970 00:00:01 GMT", "SameSite=Lax"} {
		if !strings.Contains(raw, want) {
			t.Errorf("Set-Cookie %q lacks %q", raw, want)
		}
	}
}
