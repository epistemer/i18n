package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
)

// entry is one registered key.
type entry struct {
	key, text string
	meta      []byte
	sig       signature // verbs of text, parsed once
}

// registry holds the keys declared with Key. The package uses one, reg.
// Tests build their own to simulate releases.
type registry struct {
	m        map[string]*entry
	problems []string
	frozen   atomic.Bool
}

func newRegistry() *registry {
	return &registry{m: make(map[string]*entry)}
}

// reg is filled by Key during package initialization, which runs on one
// goroutine, and frozen by the first Load. After that it's only read.
var reg = newRegistry()

// Key declares a translation key with its default text in the source locale,
// and optional meta stored once per key.
//
// Call Key only in package-level variable initializers. After the first Load
// it panics.
func Key(key, text string, meta ...byte) TKer {
	return reg.add(key, text, meta)
}

func (r *registry) add(k, text string, meta []byte) *key {
	if r.frozen.Load() {
		panic(fmt.Sprintf("i18n: Key(%q) called after Load; declare keys in package-level variables", k))
	}
	if k == "" {
		r.problems = append(r.problems, fmt.Sprintf("empty key with text %q", text))
		return &key{}
	}
	if e, ok := r.m[k]; ok {
		switch {
		case e.text != text:
			r.problems = append(r.problems, fmt.Sprintf("key %q registered twice, with texts %q and %q", k, e.text, text))
		case !bytes.Equal(e.meta, meta):
			r.problems = append(r.problems, fmt.Sprintf("key %q registered twice, with different meta", k))
		}
		return &key{name: k, e: e}
	}
	e := &entry{key: k, text: text, meta: bytes.Clone(meta), sig: parseSignature(text)}
	r.m[k] = e
	return &key{name: k, e: e}
}

// freeze stops further registration and reports every problem found while
// keys were registered.
func (r *registry) freeze() error {
	r.frozen.Store(true)
	if len(r.problems) == 0 {
		return nil
	}
	errs := make([]error, len(r.problems))
	for i, p := range r.problems {
		errs[i] = errors.New("i18n: " + p)
	}
	return errors.Join(errs...)
}

// keys returns the registered keys, sorted.
func (r *registry) keys() []string {
	ks := make([]string, 0, len(r.m))
	for k := range r.m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}
