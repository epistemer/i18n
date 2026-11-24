package i18n

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/epistemer/i18n/internal/storepb"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// Config is the setting the library reads. cfgx apps get it from
// i18nschema.Group.
type Config interface {
	// I18nDefaultLocale is the code of the locale visitors get when they
	// haven't chosen one; "" means the source locale.
	I18nDefaultLocale() string
}

// Option configures a Translator.
type Option func(*options)

type options struct {
	logger *slog.Logger
	bucket string
	cookie string
	reg    *registry
	now    func() time.Time
}

// WithLogger replaces slog.Default() for the Translator's logs.
func WithLogger(l *slog.Logger) Option { return func(o *options) { o.logger = l } }

// WithBucket names the top-level bbolt bucket the Translator owns; the
// default is "i18n".
func WithBucket(name string) Option { return func(o *options) { o.bucket = name } }

// WithCookie names the cookie that holds a visitor's chosen locale; the
// default is "i18n-language".
func WithCookie(name string) Option { return func(o *options) { o.cookie = name } }

// withRegistry makes a Translator read keys from r instead of the package's
// registry. Tests use it to simulate releases.
func withRegistry(r *registry) Option { return func(o *options) { o.reg = r } }

// Translator resolves translation keys against a bbolt store. All its
// methods are safe for concurrent use.
type Translator struct {
	db     *bolt.DB
	source Locale
	cfg    Config
	log    *slog.Logger
	bucket []byte
	cookie string
	reg    *registry
	now    func() time.Time

	sourceErr   error    // an invalid source code, reported by Load
	bucketNames sync.Map // code → *bucketNames
	lastReadErr atomic.Int64

	loadWrites atomic.Int64 // records written by the last Load; for tests
}

// New returns a Translator for db, which must be open; the Translator never
// closes it. source is the locale the code's default
// texts are written in; cfg may be nil, which means the default locale is
// the source. New doesn't touch db; call Load before serving.
func New(db *bolt.DB, source Locale, cfg Config, opts ...Option) *Translator {
	o := options{bucket: "i18n", cookie: "i18n-language", reg: reg, now: time.Now}
	for _, opt := range opts {
		opt(&o)
	}
	if o.logger == nil {
		o.logger = slog.Default()
	}
	t := &Translator{
		db: db, source: source, cfg: cfg, log: o.logger,
		bucket: []byte(o.bucket), cookie: o.cookie, reg: o.reg, now: o.now,
	}
	if code, err := canonical(source.Code); err != nil {
		t.sourceErr = fmt.Errorf("source locale: %w", err)
	} else {
		t.source.Code = code
	}
	return t
}

// Load freezes the key registry and syncs it into the store: it stores the
// source locale's record if absent, refreshes the catalog of keys, and marks
// keys gone from the code as orphaned (interface §9).
func (t *Translator) Load() error {
	if err := t.reg.freeze(); err != nil {
		return err
	}
	if t.sourceErr != nil {
		return t.sourceErr
	}
	writes := 0
	err := t.db.Update(func(tx *bolt.Tx) error {
		top, err := tx.CreateBucketIfNotExists(t.bucket)
		if err != nil {
			return err
		}
		if v := top.Get(keyVersion); v == nil {
			if err := top.Put(keyVersion, []byte{storeVersion}); err != nil {
				return err
			}
		} else if len(v) != 1 || v[0] > storeVersion {
			return fmt.Errorf("i18n: the store has layout version %v, newer than this release's %d", v, storeVersion)
		}

		if get(sub(top, bktLocales), []byte(t.source.Code)) == nil {
			if err := t.putLocale(top, t.source); err != nil {
				return err
			}
			writes++
		} else {
			// The record may outlive its buckets (a partial restore, say).
			n := t.names(t.source.Code)
			if _, err := top.CreateBucketIfNotExists(n.text); err != nil {
				return err
			}
			if _, err := top.CreateBucketIfNotExists(n.base); err != nil {
				return err
			}
		}

		catalog, err := top.CreateBucketIfNotExists(bktCatalog)
		if err != nil {
			return err
		}
		for _, k := range t.reg.keys() {
			e := t.reg.m[k]
			rec, err := marshal(&storepb.CatalogEntry{Text: e.text, Meta: e.meta})
			if err != nil {
				return err
			}
			if old := catalog.Get([]byte(k)); old == nil || string(old) != string(rec) {
				if err := catalog.Put([]byte(k), rec); err != nil {
					return err
				}
				writes++
			}
		}

		// Mark keys gone from the code. bbolt forbids writes during ForEach,
		// so collect first.
		var orphans [][]byte
		var recs [][]byte
		err = catalog.ForEach(func(k, v []byte) error {
			if _, ok := t.reg.m[string(k)]; ok {
				return nil
			}
			var ce storepb.CatalogEntry
			if err := proto.Unmarshal(v, &ce); err != nil {
				return fmt.Errorf("i18n: catalog entry %q: %w", k, err)
			}
			if ce.GetOrphaned() {
				return nil
			}
			ce.Orphaned = true
			rec, err := marshal(&ce)
			if err != nil {
				return err
			}
			orphans = append(orphans, append([]byte(nil), k...))
			recs = append(recs, rec)
			return nil
		})
		if err != nil {
			return err
		}
		for i, k := range orphans {
			if err := catalog.Put(k, recs[i]); err != nil {
				return err
			}
			writes++
		}
		return nil
	})
	if err != nil {
		return err
	}
	t.loadWrites.Store(int64(writes))
	t.warnAfterLoad()
	return nil
}

// warnAfterLoad logs, outside the write transaction, every override that no
// longer fits its code's text and a configured default locale that doesn't
// exist.
func (t *Translator) warnAfterLoad() {
	_ = t.db.View(func(tx *bolt.Tx) error {
		if t.cfg != nil {
			if raw := t.cfg.I18nDefaultLocale(); raw != "" {
				code, err := canonical(raw)
				if _, ok := t.localeTx(tx, code); err != nil || !ok {
					t.log.Warn("i18n: the configured default locale doesn't exist; visitors get the source locale",
						"locale", raw, "source", t.source.Code)
				}
			}
		}
		top := t.top(tx)
		locales := sub(top, bktLocales)
		if locales == nil {
			return nil
		}
		return locales.ForEach(func(code, _ []byte) error {
			texts := sub(top, t.names(string(code)).text)
			if texts == nil {
				return nil
			}
			return texts.ForEach(func(k, v []byte) error {
				text := ""
				if e := t.reg.m[string(k)]; e != nil {
					text = e.text
				} else if ce := catalogEntry(top, string(k)); ce != nil {
					text = ce.GetText() // an orphan
				}
				sig := parseSignature(text)
				if sig.hasArgs() && !fitsText(sig, string(v)) {
					t.log.Warn("i18n: an override no longer fits its key's format verbs and is skipped when the key gets arguments",
						"locale", string(code), "key", string(k), "override", string(v), "code_text", text)
				}
				return nil
			})
		})
	})
}

// configuredDefault returns the configured default locale's canonical code,
// or "" when none is configured or it isn't a valid code.
func (t *Translator) configuredDefault() string {
	if t.cfg == nil {
		return ""
	}
	raw := t.cfg.I18nDefaultLocale()
	if raw == "" {
		return ""
	}
	code, err := canonical(raw)
	if err != nil {
		return ""
	}
	return code
}

// Source returns the source locale, with its record from the store when it
// has one.
func (t *Translator) Source() Locale {
	l := t.source
	_ = t.db.View(func(tx *bolt.Tx) error {
		if stored, ok := t.localeTx(tx, t.source.Code); ok {
			l = stored
		}
		return nil
	})
	return l
}

// Default returns the configured default locale if it exists, else Source().
func (t *Translator) Default() Locale {
	var l Locale
	if err := t.db.View(func(tx *bolt.Tx) error { l = t.defaultTx(tx); return nil }); err != nil {
		return t.source
	}
	return l
}

func (t *Translator) defaultTx(tx *bolt.Tx) Locale {
	if code := t.configuredDefault(); code != "" {
		if l, ok := t.localeTx(tx, code); ok {
			return l
		}
	}
	if l, ok := t.localeTx(tx, t.source.Code); ok {
		return l
	}
	return t.source
}

// readFailed logs a failed read in T at most once a minute.
func (t *Translator) readFailed(err error) {
	now := t.now().UnixNano()
	last := t.lastReadErr.Load()
	if last != 0 && now-last < int64(time.Minute) {
		return
	}
	if t.lastReadErr.CompareAndSwap(last, now) {
		t.log.Error("i18n: reading the store failed; showing the code's text", "error", err)
	}
}
