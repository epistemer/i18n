package i18n

import (
	"context"
	"errors"
	"fmt"
	"sort"

	bolt "go.etcd.io/bbolt"
)

// Locales returns every locale, sorted by code. The source locale is always
// included, with its stored record when there is one.
func (t *Translator) Locales() []Locale {
	var out []Locale
	var sourceFound bool
	err := t.db.View(func(tx *bolt.Tx) error {
		top := t.top(tx)
		locales := sub(top, bktLocales)
		if locales == nil {
			return nil
		}
		return locales.ForEach(func(k, _ []byte) error {
			code := string(k)
			if l, ok := t.localeTx(tx, code); ok {
				out = append(out, l)
				if code == t.source.Code {
					sourceFound = true
				}
			}
			return nil
		})
	})
	if err != nil {
		return []Locale{t.source}
	}
	if !sourceFound {
		idx := sort.Search(len(out), func(i int) bool {
			return out[i].Code >= t.source.Code
		})
		out = append(out, Locale{})
		copy(out[idx+1:], out[idx:])
		out[idx] = t.source
	}
	return out
}

// Locale returns the locale with this code, if it exists. An invalid code
// reports false.
func (t *Translator) Locale(code string) (Locale, bool) {
	canon, err := canonical(code)
	if err != nil {
		return Locale{}, false
	}
	var l Locale
	var ok bool
	_ = t.db.View(func(tx *bolt.Tx) error {
		l, ok = t.localeTx(tx, canon)
		return nil
	})
	return l, ok
}

// CreateLocale adds a locale. A new locale starts empty: every key shows
// through the resolution order until someone translates it.
// Errors: ErrInvalidCode, ErrLocaleExists.
func (t *Translator) CreateLocale(l Locale) error {
	canon, err := canonical(l.Code)
	if err != nil {
		return err
	}
	l.Code = canon
	return t.db.Update(func(tx *bolt.Tx) error {
		top, err := tx.CreateBucketIfNotExists(t.bucket)
		if err != nil {
			return err
		}
		if sub(top, t.names(canon).text) != nil {
			return fmt.Errorf("%w: %q", ErrLocaleExists, canon)
		}
		return t.putLocale(top, l)
	})
}

// UpdateLocale changes a locale's native name, flag code and direction.
// Errors: ErrInvalidCode, ErrUnknownLocale.
func (t *Translator) UpdateLocale(l Locale) error {
	canon, err := canonical(l.Code)
	if err != nil {
		return err
	}
	l.Code = canon
	return t.db.Update(func(tx *bolt.Tx) error {
		top := t.top(tx)
		if sub(top, t.names(canon).text) == nil {
			return fmt.Errorf("%w: %q", ErrUnknownLocale, canon)
		}
		return t.putLocale(top, l)
	})
}

// DeleteLocale removes a locale with all its overrides: its record and its
// t:<code> and b:<code> buckets.
// Errors: ErrInvalidCode, ErrSourceLocale, ErrUnknownLocale.
func (t *Translator) DeleteLocale(code string) error {
	canon, err := canonical(code)
	if err != nil {
		return err
	}
	if canon == t.source.Code {
		return fmt.Errorf("%w: %q", ErrSourceLocale, canon)
	}
	return t.db.Update(func(tx *bolt.Tx) error {
		top := t.top(tx)
		if top == nil {
			return fmt.Errorf("%w: %q", ErrUnknownLocale, canon)
		}
		n := t.names(canon)
		if top.Bucket(n.text) == nil {
			return fmt.Errorf("%w: %q", ErrUnknownLocale, canon)
		}
		if locales := sub(top, bktLocales); locales != nil {
			if err := locales.Delete([]byte(canon)); err != nil {
				return err
			}
		}
		if err := top.DeleteBucket(n.text); err != nil {
			return err
		}
		if err := top.DeleteBucket(n.base); err != nil && !errors.Is(err, bolt.ErrBucketNotFound) {
			return err
		}
		return nil
	})
}

// Match returns the closest existing locale for code: the code itself, then
// its base language, then Default(). For codes stored outside a request,
// such as the language a user registered with.
func (t *Translator) Match(code string) Locale {
	var out Locale
	err := t.db.View(func(tx *bolt.Tx) error {
		canon, err := canonical(code)
		if err != nil {
			out = t.defaultTx(tx)
			return nil
		}
		if l, ok := t.localeTx(tx, canon); ok {
			out = l
			return nil
		}
		if base := baseLanguage(canon); base != "" {
			if l, ok := t.localeTx(tx, base); ok {
				out = l
				return nil
			}
		}
		out = t.defaultTx(tx)
		return nil
	})
	if err != nil {
		return t.Default()
	}
	return out
}

// ParseAcceptLanguage returns the best existing locale for an
// Accept-Language header value: each tag in order of preference, first
// exactly, then by its base language; Default() when nothing matches.
func (t *Translator) ParseAcceptLanguage(header string) Locale {
	var out Locale
	err := t.db.View(func(tx *bolt.Tx) error {
		tags := acceptLanguage(header)
		for _, tag := range tags {
			if l, ok := t.localeTx(tx, tag); ok {
				out = l
				return nil
			}
			if base := baseLanguage(tag); base != "" {
				if l, ok := t.localeTx(tx, base); ok {
					out = l
					return nil
				}
			}
		}
		out = t.defaultTx(tx)
		return nil
	})
	if err != nil {
		return t.Default()
	}
	return out
}

// Codes returns every locale, sorted by code, from the Translator in ctx;
// nil without one.
func Codes(ctx context.Context) []Locale {
	t := FromContext(ctx)
	if t == nil {
		return nil
	}
	return t.Locales()
}
