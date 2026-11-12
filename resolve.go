package i18n

import (
	"context"

	"github.com/epistemer/i18n/internal/storepb"
	bolt "go.etcd.io/bbolt"
)

type ctxKey struct{}

type ctxVal struct {
	t         *Translator
	l         Locale
	hasLocale bool
}

// WithContext returns a context carrying t and the locale l. T and every
// key's T read them from it.
func WithContext(ctx context.Context, t *Translator, l Locale) context.Context {
	return context.WithValue(ctx, ctxKey{}, &ctxVal{t: t, l: l, hasLocale: true})
}

// FromContext returns the Translator in ctx, or nil.
func FromContext(ctx context.Context) *Translator {
	if v, ok := ctx.Value(ctxKey{}).(*ctxVal); ok {
		return v.t
	}
	return nil
}

// Code returns the locale in ctx: the one WithContext attached, the default
// locale when only a Translator is there, and Locale{} without one.
func Code(ctx context.Context) Locale {
	v, ok := ctx.Value(ctxKey{}).(*ctxVal)
	if !ok || v.t == nil {
		return Locale{}
	}
	if v.hasLocale {
		return v.l
	}
	return v.t.Default()
}

// T returns the text of the key named key in the context's locale, formatted
// with args when there are any (interface §5.1, §5.2).
func T(ctx context.Context, key string, args ...any) string {
	return resolveCtx(ctx, key, nil, args)
}

// resolveCtx is T for any handle: e is the registered entry when the caller
// has it, nil when only the name is known.
func resolveCtx(ctx context.Context, name string, e *entry, args []any) string {
	v, _ := ctx.Value(ctxKey{}).(*ctxVal)
	if v == nil || v.t == nil {
		if e == nil {
			e = reg.m[name] // read-only once frozen; at init only Key writes
		}
		if e == nil {
			return name
		}
		return format(e.text, args)
	}
	t := v.t
	var out string
	err := t.db.View(func(tx *bolt.Tx) error {
		code := v.l.Code
		if !v.hasLocale {
			code = t.defaultTx(tx).Code
		}
		out = t.resolveTx(tx, code, name, e, args)
		return nil
	})
	if err != nil {
		t.readFailed(err)
		if text := t.codeText(nil, name, e); text != name || e != nil {
			return format(text, args)
		}
		return name
	}
	return out
}

// resolveTx resolves within tx, so T (in a View) and Reset (in its Update)
// share it. See docs/implementation.md, section 4.
func (t *Translator) resolveTx(tx *bolt.Tx, code, name string, e *entry, args []any) string {
	top := t.top(tx)
	if e == nil {
		e = t.reg.m[name]
	}
	// An unregistered key's catalog entry (an orphan) is read only when it's
	// needed: for its signature, or for its text when no override wins.
	var orphan *storepb.CatalogEntry
	orphanRead := e != nil
	readOrphan := func() *storepb.CatalogEntry {
		if !orphanRead {
			orphanRead = true
			orphan = catalogEntry(top, name)
		}
		return orphan
	}

	if top != nil {
		k := []byte(name)
		var sig signature
		sigKnown := false
		for _, c := range t.chain(code) {
			b := top.Bucket(t.names(c).text)
			if b == nil {
				continue
			}
			v := b.Get(k)
			if v == nil {
				continue
			}
			ov := string(v)
			if len(args) > 0 {
				if !sigKnown {
					sigKnown = true
					switch {
					case e != nil:
						sig = e.sig
					case readOrphan() != nil:
						sig = parseSignature(orphan.GetText())
					}
					// A key the catalog has never seen has no signature.
				}
				if !fitsText(sig, ov) {
					continue
				}
			}
			return format(ov, args)
		}
	}
	if e != nil {
		return format(e.text, args)
	}
	if ce := readOrphan(); ce != nil {
		return format(ce.GetText(), args)
	}
	return name // the key itself, never formatted
}

// codeText is the code's text for a key: the registered text, the catalog's
// last text for an orphan, or the name itself. top may be nil.
func (t *Translator) codeText(top *bolt.Bucket, name string, e *entry) string {
	if e != nil {
		return e.text
	}
	if e := t.reg.m[name]; e != nil {
		return e.text
	}
	if top != nil {
		if ce := catalogEntry(top, name); ce != nil {
			return ce.GetText()
		}
	}
	return name
}

// chain returns the locales tried for code, in order: the code, its base
// language, then the source, without repeats.
func (t *Translator) chain(code string) []string {
	out := make([]string, 0, 3)
	out = append(out, code)
	if base := baseLanguage(code); base != "" && base != code {
		out = append(out, base)
	}
	if src := t.source.Code; src != code && (len(out) < 2 || out[1] != src) {
		out = append(out, src)
	}
	return out
}
