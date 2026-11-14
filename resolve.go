package i18n

import (
	"context"
	"reflect"

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
// with args when there are any (interface §5.1, §5.2). An argument that
// implements Ter is rendered in the same context first, so a label passed as
// an argument shows in the same locale.
func T(ctx context.Context, key string, args ...any) string {
	return resolveCtx(ctx, key, nil, args)
}

// resolveCtx is T for any handle: e is the registered entry when the caller
// has it, nil when only the name is known.
func resolveCtx(ctx context.Context, name string, e *entry, args []any) string {
	// Labels among the arguments render first, outside any transaction: a
	// nested read transaction in one goroutine can wait behind a writer that
	// waits for the outer one (interface §5.2).
	args = renderArgs(ctx, args)
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
	var formattable bool
	err := t.db.View(func(tx *bolt.Tx) error {
		code := v.l.Code
		if !v.hasLocale {
			code = t.defaultTx(tx).Code
		}
		out, formattable = t.resolveTx(tx, code, name, e, len(args) > 0)
		return nil
	})
	if err != nil {
		t.readFailed(err)
		if text := t.codeText(nil, name, e); text != name || e != nil {
			return format(text, args)
		}
		return name
	}
	// Formatting runs after the transaction too: an argument's String or
	// Error method may call T.
	if !formattable {
		return out
	}
	return format(out, args)
}

// resolveTx resolves within tx, so T (in a View) and Reset (in its Update)
// share it. See docs/implementation.md, section 4. It returns the text
// unformatted; formattable is false for a key's own name, which is never
// formatted. withArgs says whether the caller will format, so overrides that
// no longer fit are skipped.
func (t *Translator) resolveTx(tx *bolt.Tx, code, name string, e *entry, withArgs bool) (text string, formattable bool) {
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
			if withArgs {
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
			return ov, true
		}
	}
	if e != nil {
		return e.text, true
	}
	if ce := readOrphan(); ce != nil {
		return ce.GetText(), true
	}
	return name, false // the key itself, never formatted
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

// renderArgs returns args with every argument that implements Ter replaced by
// its text in ctx (interface §5.2). args itself is never modified; when no
// argument is a Ter, it's returned as is, without allocating.
func renderArgs(ctx context.Context, args []any) []any {
	i := 0
	for ; i < len(args); i++ {
		if _, ok := args[i].(Ter); ok {
			break
		}
	}
	if i == len(args) {
		return args
	}
	out := make([]any, len(args))
	copy(out, args)
	for ; i < len(out); i++ {
		if l, ok := out[i].(Ter); ok {
			out[i] = renderArg(ctx, l)
		}
	}
	return out
}

// renderArg renders one label with no arguments of its own. A nil pointer or
// nil func label whose T panics renders as "<nil>", as fmt prints a nil
// pointer Stringer; any other panic is the label's own and propagates.
func renderArg(ctx context.Context, l Ter) (out string) {
	defer func() {
		if p := recover(); p != nil {
			if isNil(l) {
				out = "<nil>"
				return
			}
			panic(p)
		}
	}()
	return l.T(ctx)
}

func isNil(x any) bool {
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Func:
		return v.IsNil()
	}
	return false
}
