package i18n

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/epistemer/i18n/internal/storepb"
	bolt "go.etcd.io/bbolt"
)

var english = Locale{Code: "en", NativeName: "English", FlagCode: "us"}

type cfg string

func (c cfg) I18nDefaultLocale() string { return string(c) }

func testDB(tb testing.TB) *bolt.DB {
	tb.Helper()
	db, err := bolt.Open(filepath.Join(tb.TempDir(), "test.db"), 0o600, nil)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { db.Close() })
	return db
}

// release builds a registry from key/text pairs, as one build of an app
// would register them.
func release(pairs ...string) *registry {
	r := newRegistry()
	for i := 0; i+1 < len(pairs); i += 2 {
		r.add(pairs[i], pairs[i+1], nil)
	}
	return r
}

func newT(tb testing.TB, db *bolt.DB, r *registry, opts ...Option) *Translator {
	tb.Helper()
	return New(db, english, nil, append([]Option{withRegistry(r)}, opts...)...)
}

func mustLoad(tb testing.TB, t *Translator) {
	tb.Helper()
	if err := t.Load(); err != nil {
		tb.Fatal(err)
	}
}

// addLocale writes a locale straight into the store, since CreateLocale
// arrives in M2.
func addLocale(tb testing.TB, t *Translator, code string) {
	tb.Helper()
	err := t.db.Update(func(tx *bolt.Tx) error {
		top, err := tx.CreateBucketIfNotExists(t.bucket)
		if err != nil {
			return err
		}
		return t.putLocale(top, Locale{Code: code, NativeName: code})
	})
	if err != nil {
		tb.Fatal(err)
	}
}

// putOverride writes an override straight into the store, since Set arrives
// in M2.
func putOverride(tb testing.TB, t *Translator, code, key, text string) {
	tb.Helper()
	err := t.db.Update(func(tx *bolt.Tx) error {
		return t.top(tx).Bucket(t.names(code).text).Put([]byte(key), []byte(text))
	})
	if err != nil {
		tb.Fatal(err)
	}
}

func catalogOf(tb testing.TB, t *Translator, k string) *storepb.CatalogEntry {
	tb.Helper()
	var ce *storepb.CatalogEntry
	_ = t.db.View(func(tx *bolt.Tx) error { ce = catalogEntry(t.top(tx), k); return nil })
	return ce
}

func in(t *Translator, code string) context.Context {
	return WithContext(context.Background(), t, Locale{Code: code})
}

// logRecorder is a slog handler that keeps messages.
type logRecorder struct {
	mu   sync.Mutex
	msgs []string
}

func (l *logRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (l *logRecorder) Handle(_ context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.msgs = append(l.msgs, r.Message)
	return nil
}
func (l *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return l }
func (l *logRecorder) WithGroup(string) slog.Handler      { return l }

func (l *logRecorder) count(substr string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, m := range l.msgs {
		if strings.Contains(m, substr) {
			n++
		}
	}
	return n
}
