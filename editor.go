package i18n

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/epistemer/i18n/internal/storepb"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// Set stores text as the override of key in the locale code, and records the
// source text it was based on (interface §8.1). Only registered keys can be
// set, and text must fit the key's format verbs (interface §5.2); an empty
// text always fits.
// Errors: ErrInvalidCode, ErrUnknownKey, ErrFormat, ErrUnknownLocale.
func (t *Translator) Set(code, key, text string) error {
	canon, err := canonical(code)
	if err != nil {
		return err
	}
	e := t.reg.m[key]
	if e == nil {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	if !fitsText(e.sig, text) {
		return fmt.Errorf("%w: %q for key %q, whose code text is %q", ErrFormat, text, key, e.text)
	}
	return t.db.Update(func(tx *bolt.Tx) error {
		top := t.top(tx)
		n := t.names(canon)
		texts := sub(top, n.text)
		if texts == nil {
			return fmt.Errorf("%w: %q", ErrUnknownLocale, canon)
		}
		bases, err := top.CreateBucketIfNotExists(n.base)
		if err != nil {
			return err
		}
		rec, err := marshal(&storepb.Base{Text: t.baseTextTx(top, canon, key, e), SetUnix: t.now().Unix()})
		if err != nil {
			return err
		}
		if err := texts.Put([]byte(key), []byte(text)); err != nil {
			return err
		}
		return bases.Put([]byte(key), rec)
	})
}

// baseTextTx is the source text an override in code is based on: the code's
// text for the source locale; for any other locale, what it would show
// without its own override (interface §5.1, steps 2–4).
func (t *Translator) baseTextTx(top *bolt.Bucket, code, key string, e *entry) string {
	if code != t.source.Code {
		for _, c := range t.chain(code)[1:] {
			if v := get(sub(top, t.names(c).text), []byte(key)); v != nil && rendersWithArgs(e.sig, string(v)) {
				return string(v)
			}
		}
	}
	return e.text
}

// Reset removes the override of key in the locale code, and returns the text
// the locale shows now.
// Errors: ErrInvalidCode, ErrUnknownLocale.
func (t *Translator) Reset(code, key string) (string, error) {
	canon, err := canonical(code)
	if err != nil {
		return "", err
	}
	var shown string
	err = t.db.Update(func(tx *bolt.Tx) error {
		top := t.top(tx)
		n := t.names(canon)
		texts := sub(top, n.text)
		if texts == nil {
			return fmt.Errorf("%w: %q", ErrUnknownLocale, canon)
		}
		if err := texts.Delete([]byte(key)); err != nil {
			return err
		}
		if bases := sub(top, n.base); bases != nil {
			if err := bases.Delete([]byte(key)); err != nil {
				return err
			}
		}
		// The same transaction, so it sees its own delete.
		shown = t.resolveTx(tx, canon, key, nil, nil)
		return nil
	})
	return shown, err
}

// Entry describes one key in one locale, for an editor.
type Entry struct {
	Key      string
	Text     string // what T shows in this locale (§5.1), unformatted
	Source   string // what T shows in the source locale
	Override bool   // a person set this value in this locale
	Base     string // the source text the override was based on; "" without one
	Misfit   bool   // the override no longer fits the code's text, so T skips it when given args
	Meta     []byte // the key's meta
}

// Query selects entries for Search.
type Query struct {
	Prefix   string           // keys starting with this
	Contains string           // case-insensitive, in the key, Text or Source; "" matches all
	Match    func(Entry) bool // optional; applied before Offset and Limit
	Offset   int
	Limit    int // 0 means no limit
}

// Search lists the keys of the locale code that match q, sorted by key.
// Orphaned keys are left out. It runs in one read transaction.
// Errors: ErrInvalidCode, ErrUnknownLocale.
func (t *Translator) Search(code string, q Query) ([]Entry, error) {
	canon, err := canonical(code)
	if err != nil {
		return nil, err
	}
	contains := strings.ToLower(q.Contains)
	prefix := []byte(q.Prefix)
	var out []Entry
	err = t.db.View(func(tx *bolt.Tx) error {
		top := t.top(tx)
		if _, ok := t.localeTx(tx, canon); !ok {
			return fmt.Errorf("%w: %q", ErrUnknownLocale, canon)
		}
		catalog := sub(top, bktCatalog)
		if catalog == nil {
			return nil
		}
		names := t.names(canon)
		texts, bases := sub(top, names.text), sub(top, names.base)
		skipped := 0
		c := catalog.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var ce storepb.CatalogEntry
			if err := proto.Unmarshal(v, &ce); err != nil {
				return fmt.Errorf("i18n: catalog entry %q: %w", k, err)
			}
			// Only keys this release registers: that also covers a Search
			// that runs before Load has marked the orphans.
			if ce.GetOrphaned() || t.reg.m[string(k)] == nil {
				continue
			}
			if contains == "" && q.Match == nil && skipped < q.Offset {
				skipped++ // no need to build an entry that's skipped anyway
				continue
			}
			en := t.entryTx(top, canon, string(k), &ce, texts, bases)
			if contains != "" &&
				!strings.Contains(strings.ToLower(en.Key), contains) &&
				!strings.Contains(strings.ToLower(en.Text), contains) &&
				!strings.Contains(strings.ToLower(en.Source), contains) {
				continue
			}
			if q.Match != nil && !q.Match(en) {
				continue
			}
			if skipped < q.Offset {
				skipped++
				continue
			}
			out = append(out, en)
			if q.Limit > 0 && len(out) == q.Limit {
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// entryTx builds the Entry of key in code. The code's text comes from the
// registry, like T's, and from the catalog only for a key this release
// doesn't register.
func (t *Translator) entryTx(top *bolt.Bucket, code, key string, ce *storepb.CatalogEntry, texts, bases *bolt.Bucket) Entry {
	codeText, meta, sig := ce.GetText(), ce.GetMeta(), signature{}
	if e := t.reg.m[key]; e != nil {
		codeText, meta, sig = e.text, e.meta, e.sig
	} else {
		sig = parseSignature(codeText)
	}
	en := Entry{Key: key, Meta: bytes.Clone(meta)}

	// The source's text: its override, else the code's text.
	en.Source = codeText
	if v := get(sub(top, t.names(t.source.Code).text), []byte(key)); v != nil && rendersWithArgs(sig, string(v)) {
		en.Source = string(v)
	}

	if v := get(texts, []byte(key)); v != nil {
		en.Override = true
		en.Text = string(v)
		en.Misfit = sig.hasArgs() && !fitsText(sig, en.Text)
		if b := get(bases, []byte(key)); b != nil {
			var rec storepb.Base
			if proto.Unmarshal(b, &rec) == nil {
				en.Base = rec.GetText()
			}
		}
		return en
	}
	// No override here: the base language's, then the source's, then the code's.
	en.Text = codeText
	for _, c := range t.chain(code)[1:] {
		if v := get(sub(top, t.names(c).text), []byte(key)); v != nil && rendersWithArgs(sig, string(v)) {
			en.Text = string(v)
			break
		}
	}
	return en
}

// rendersWithArgs reports whether a fallback override is what a key shows in
// practice. A key whose code text has verbs is called with arguments, so an
// override that no longer fits is skipped, as T skips it (interface §5.2).
func rendersWithArgs(sig signature, text string) bool {
	return !sig.hasArgs() || fitsText(sig, text)
}

// Orphans lists the keys gone from the code, sorted. Prune deletes exactly
// these, unless a Load runs in between.
func (t *Translator) Orphans() ([]string, error) {
	var out []string
	err := t.db.View(func(tx *bolt.Tx) error {
		var err error
		out, err = t.orphansTx(sub(t.top(tx), bktCatalog))
		return err
	})
	return out, err
}

// orphansTx lists catalog keys marked orphaned that this release doesn't
// register. The second check matters before Load: a key that came back must
// never be pruned.
func (t *Translator) orphansTx(catalog *bolt.Bucket) ([]string, error) {
	if catalog == nil {
		return nil, nil
	}
	var out []string
	err := catalog.ForEach(func(k, v []byte) error {
		var ce storepb.CatalogEntry
		if err := proto.Unmarshal(v, &ce); err != nil {
			return fmt.Errorf("i18n: catalog entry %q: %w", k, err)
		}
		if ce.GetOrphaned() && t.reg.m[string(k)] == nil {
			out = append(out, string(k))
		}
		return nil
	})
	return out, err
}

// Prune deletes the orphaned keys from the catalog, with every override and
// base record they have in any locale, and returns them.
func (t *Translator) Prune() ([]string, error) {
	var removed []string
	err := t.db.Update(func(tx *bolt.Tx) error {
		top := t.top(tx)
		catalog := sub(top, bktCatalog)
		var err error
		if removed, err = t.orphansTx(catalog); err != nil || len(removed) == 0 {
			return err
		}
		// Every t: and b: bucket, whether or not its locale record exists.
		var perLocale []*bolt.Bucket
		err = top.ForEach(func(name, v []byte) error {
			if v == nil && (bytes.HasPrefix(name, []byte("t:")) || bytes.HasPrefix(name, []byte("b:"))) {
				if b := top.Bucket(name); b != nil {
					perLocale = append(perLocale, b)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		for _, k := range removed {
			if err := catalog.Delete([]byte(k)); err != nil {
				return err
			}
			for _, b := range perLocale {
				if err := b.Delete([]byte(k)); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return removed, nil
}
