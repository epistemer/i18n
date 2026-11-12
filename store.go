package i18n

import (
	"bytes"
	"fmt"

	"github.com/epistemer/i18n/internal/storepb"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// The store's layout is in docs/implementation.md, section 2.
var (
	keyVersion = []byte("version")
	bktLocales = []byte("locales")
	bktCatalog = []byte("catalog")
)

// storeVersion is the layout this code reads and writes.
const storeVersion = 1

// bucketNames are the per-locale bucket names, computed once per code.
type bucketNames struct {
	text, base []byte
}

func (t *Translator) names(code string) *bucketNames {
	if n, ok := t.bucketNames.Load(code); ok {
		return n.(*bucketNames)
	}
	n := &bucketNames{text: []byte("t:" + code), base: []byte("b:" + code)}
	actual, _ := t.bucketNames.LoadOrStore(code, n)
	return actual.(*bucketNames)
}

// top returns the library's top-level bucket, or nil before the first Load.
func (t *Translator) top(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(t.bucket)
}

// sub returns a bucket inside the top one, or nil if either is missing.
func sub(top *bolt.Bucket, name []byte) *bolt.Bucket {
	if top == nil {
		return nil
	}
	return top.Bucket(name)
}

// get reads k from b, which may be nil.
func get(b *bolt.Bucket, k []byte) []byte {
	if b == nil {
		return nil
	}
	return b.Get(k)
}

// localeTx reads the record of an existing locale. A locale exists exactly
// when its t:<code> bucket does.
func (t *Translator) localeTx(tx *bolt.Tx, code string) (Locale, bool) {
	top := t.top(tx)
	if sub(top, t.names(code).text) == nil {
		return Locale{}, false
	}
	l := Locale{Code: code}
	if v := get(sub(top, bktLocales), []byte(code)); v != nil {
		var rec storepb.LocaleRecord
		if err := proto.Unmarshal(v, &rec); err == nil {
			l.NativeName, l.FlagCode, l.RTL = rec.GetNativeName(), rec.GetFlagCode(), rec.GetRtl()
		}
	}
	return l, true
}

// putLocale writes a locale's record and makes sure its buckets exist.
func (t *Translator) putLocale(top *bolt.Bucket, l Locale) error {
	locales, err := top.CreateBucketIfNotExists(bktLocales)
	if err != nil {
		return err
	}
	rec, err := marshal(&storepb.LocaleRecord{NativeName: l.NativeName, FlagCode: l.FlagCode, Rtl: l.RTL})
	if err != nil {
		return err
	}
	if err := locales.Put([]byte(l.Code), rec); err != nil {
		return err
	}
	n := t.names(l.Code)
	if _, err := top.CreateBucketIfNotExists(n.text); err != nil {
		return err
	}
	_, err = top.CreateBucketIfNotExists(n.base)
	return err
}

// catalogEntry decodes catalog/<key>, or returns nil if it's absent or
// unreadable.
func catalogEntry(top *bolt.Bucket, k string) *storepb.CatalogEntry {
	v := get(sub(top, bktCatalog), []byte(k))
	if v == nil {
		return nil
	}
	var ce storepb.CatalogEntry
	if proto.Unmarshal(v, &ce) != nil {
		return nil
	}
	ce.Meta = bytes.Clone(ce.Meta) // never let bytes outlive the transaction
	return &ce
}

func marshal(m proto.Message) ([]byte, error) {
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("i18n: encode record: %w", err)
	}
	return b, nil
}
