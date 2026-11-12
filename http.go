package i18n

import (
	"fmt"
	"net/http"

	bolt "go.etcd.io/bbolt"
)

// MiddlewareOption configures Middleware.
type MiddlewareOption func(*middlewareOptions)

type middlewareOptions struct {
	ignoreAcceptLanguage bool
}

// IgnoreAcceptLanguage makes the deployment's default locale beat the
// browser's languages, for example on a dashboard white-labeled in Arabic for
// staff with English browsers. A visitor's chosen locale (the cookie) still
// wins.
func IgnoreAcceptLanguage() MiddlewareOption {
	return func(o *middlewareOptions) { o.ignoreAcceptLanguage = true }
}

// Middleware picks each request's locale and attaches it, with t, to the
// request's context (interface §7). The order: the locale cookie, then the
// Accept-Language header unless ignored, then Default(). It reads the store
// once per request.
func (t *Translator) Middleware(opts ...MiddlewareOption) func(http.Handler) http.Handler {
	var o middlewareOptions
	for _, opt := range opts {
		opt(&o)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l, negotiated := t.requestLocale(r, o)
			if negotiated {
				// The header took part, even if the default won: a cache must
				// not serve this response to a visitor whose header matches.
				w.Header().Add("Vary", "Accept-Language")
			}
			next.ServeHTTP(w, r.WithContext(WithContext(r.Context(), t, l)))
		})
	}
}

// requestLocale resolves a request's locale in one read transaction, and
// reports whether Accept-Language took part in the choice.
func (t *Translator) requestLocale(r *http.Request, o middlewareOptions) (Locale, bool) {
	var l Locale
	negotiated := false
	err := t.db.View(func(tx *bolt.Tx) error {
		if c, err := r.Cookie(t.cookie); err == nil {
			if code, err := canonical(c.Value); err == nil {
				if found, ok := t.localeTx(tx, code); ok {
					l = found
					return nil
				}
			}
		}
		if !o.ignoreAcceptLanguage {
			negotiated = true
			if found, ok := t.acceptTx(tx, r.Header.Get("Accept-Language")); ok {
				l = found
				return nil
			}
		}
		l = t.defaultTx(tx)
		return nil
	})
	if err != nil {
		t.readFailed(err)
		return t.source, negotiated
	}
	return l, negotiated
}

// acceptTx finds the best existing locale for an Accept-Language header:
// each tag in order of preference, first exactly, then by its base language.
func (t *Translator) acceptTx(tx *bolt.Tx, header string) (Locale, bool) {
	for _, tag := range acceptLanguage(header) {
		if l, ok := t.localeTx(tx, tag); ok {
			return l, true
		}
		if base := baseLanguage(tag); base != "" {
			if l, ok := t.localeTx(tx, base); ok {
				return l, true
			}
		}
	}
	return Locale{}, false
}

// Cookie returns the cookie that records a visitor's chosen locale: one year,
// Path=/, SameSite=Lax, readable by scripts. Set Domain or Secure on it
// before writing it if the app needs them.
// Errors: ErrUnknownLocale.
func (t *Translator) Cookie(code string) (*http.Cookie, error) {
	l, ok := t.Locale(code)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownLocale, code)
	}
	return &http.Cookie{
		Name:     t.cookie,
		Value:    l.Code,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	}, nil
}

// SetCookie writes Cookie(code) to w.
// Errors: ErrUnknownLocale.
func (t *Translator) SetCookie(w http.ResponseWriter, code string) error {
	c, err := t.Cookie(code)
	if err != nil {
		return err
	}
	http.SetCookie(w, c)
	return nil
}

// ClearCookie removes the visitor's chosen locale, so Accept-Language and the
// default decide again.
func (t *Translator) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: t.cookie, Value: "", Path: "/", MaxAge: -1, SameSite: http.SameSiteLaxMode})
}
