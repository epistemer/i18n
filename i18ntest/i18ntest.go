package i18ntest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/epistemer/i18n"
	bolt "go.etcd.io/bbolt"
)

// New returns a Translator on a temporary bbolt file, already loaded with the
// process's registered keys. The file is closed when the test ends. The
// default locale is the source.
func New(tb testing.TB, source i18n.Locale, opts ...i18n.Option) *i18n.Translator {
	tb.Helper()
	db, err := bolt.Open(filepath.Join(tb.TempDir(), "i18n.db"), 0o600, nil)
	if err != nil {
		tb.Fatalf("i18ntest: open the store: %v", err)
	}
	tb.Cleanup(func() { db.Close() })
	t := i18n.New(db, source, nil, opts...)
	if err := t.Load(); err != nil {
		tb.Fatalf("i18ntest: load: %v", err)
	}
	return t
}

// Context returns a context carrying t and the locale with this code. It fails
// tb if the locale doesn't exist.
func Context(tb testing.TB, t *i18n.Translator, code string) context.Context {
	tb.Helper()
	l, ok := t.Locale(code)
	if !ok {
		tb.Fatalf("i18ntest: no locale %q", code)
	}
	return i18n.WithContext(context.Background(), t, l)
}
