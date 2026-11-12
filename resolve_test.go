package i18n

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/epistemer/i18n/internal/storepb"
	bolt "go.etcd.io/bbolt"
)

// arabicSetup loads keys k and n, and creates ar and ar-EG next to en.
func arabicSetup(t testing.TB) (*Translator, *registry) {
	r := release("k", "Code text", "n", "%d items")
	tr := newT(t, testDB(t), r)
	mustLoad(t, tr)
	addLocale(t, tr, "ar")
	addLocale(t, tr, "ar-EG")
	return tr, r
}

func TestResolve_Order(t *testing.T) {
	tr, r := arabicSetup(t)
	k := &key{name: "k", e: r.m["k"]}
	ctx := in(tr, "ar-EG")

	if got := k.T(ctx); got != "Code text" {
		t.Errorf("step 4, code's text: %q", got)
	}
	putOverride(t, tr, "en", "k", "Deployment English")
	if got := k.T(ctx); got != "Deployment English" {
		t.Errorf("step 3, source override: %q", got)
	}
	putOverride(t, tr, "ar", "k", "عربي")
	if got := k.T(ctx); got != "عربي" {
		t.Errorf("step 2, base language: %q", got)
	}
	putOverride(t, tr, "ar-EG", "k", "مصري")
	if got := k.T(ctx); got != "مصري" {
		t.Errorf("step 1, the locale's own override: %q", got)
	}
	if got := k.T(in(tr, "ar")); got != "عربي" {
		t.Errorf("ar sees its own override, not ar-EG's: %q", got)
	}
	if got := k.T(in(tr, "en")); got != "Deployment English" {
		t.Errorf("source: %q", got)
	}
}

func TestResolve_ByName(t *testing.T) {
	tr, _ := arabicSetup(t)
	putOverride(t, tr, "ar", "k", "عربي")
	if got := T(in(tr, "ar"), "k"); got != "عربي" {
		t.Errorf("T by name: %q", got)
	}
	if got := DynamicWrap("k").T(in(tr, "ar")); got != "عربي" {
		t.Errorf("DynamicWrap: %q", got)
	}
	if got := Dynamic("k").T(in(tr, "ar")); got != "عربي" {
		t.Errorf("Dynamic: %q", got)
	}
	if got := T(in(tr, "ar"), "never.seen"); got != "never.seen" {
		t.Errorf("an unknown key renders as itself: %q", got)
	}
}

func TestResolve_Orphan(t *testing.T) {
	db := testDB(t)
	mustLoad(t, newT(t, db, release("gl.old", "Old journal")))
	tr := newT(t, db, release()) // a release without the key
	mustLoad(t, tr)
	if got := T(in(tr, "en"), "gl.old"); got != "Old journal" {
		t.Errorf("an orphan renders its last code text, got %q", got)
	}
}

func TestResolve_Formatting(t *testing.T) {
	tr, r := arabicSetup(t)
	n := &key{name: "n", e: r.m["n"]}
	ctx := in(tr, "ar")

	putOverride(t, tr, "ar", "n", "%s عناصر") // wrong verb
	if got := n.T(ctx, 3); got != "3 items" {
		t.Errorf("a misfit override is skipped when args are passed: %q", got)
	}
	if got := n.T(ctx); got != "%s عناصر" {
		t.Errorf("without args the override is shown as is: %q", got)
	}
	putOverride(t, tr, "ar", "n", "%d عناصر")
	if got := n.T(ctx, 3); got != "3 عناصر" {
		t.Errorf("a fitting override is formatted: %q", got)
	}
	putOverride(t, tr, "ar", "n", "")
	if got := n.T(ctx, 3); got != "" {
		t.Errorf("an empty override stays empty with args: %q", got)
	}
	putOverride(t, tr, "ar", "k", "خصم 50% اليوم")
	if got := T(ctx, "k"); got != "خصم 50% اليوم" {
		t.Errorf("a %% without args is a plain character: %q", got)
	}
}

func TestResolve_WithoutTranslator(t *testing.T) {
	r := release("k", "Code text")
	k := &key{name: "k", e: r.m["k"]}
	if got := k.T(context.Background()); got != "Code text" {
		t.Errorf("got %q", got)
	}
	if got := DynamicWrap("unknown.key").T(context.Background()); got != "unknown.key" {
		t.Errorf("got %q", got)
	}
	if got := Code(context.Background()); got != (Locale{}) {
		t.Errorf("Code without a translator = %+v", got)
	}
}

func TestResolve_BeforeLoad(t *testing.T) {
	r := release("k", "Code text")
	tr := newT(t, testDB(t), r)
	if got := (&key{name: "k", e: r.m["k"]}).T(in(tr, "ar")); got != "Code text" {
		t.Errorf("before the first Load: %q", got)
	}
	if got := tr.Default().Code; got != "en" {
		t.Errorf("Default before Load = %q", got)
	}
}

func TestResolve_ReadFailure(t *testing.T) {
	logs := &logRecorder{}
	now := time.Unix(1_700_000_000, 0)
	r := release("k", "Code text")
	db := testDB(t)
	tr := New(db, english, nil, withRegistry(r), WithLogger(slog.New(logs)))
	tr.now = func() time.Time { return now }
	mustLoad(t, tr)
	db.Close()

	k := &key{name: "k", e: r.m["k"]}
	ctx := in(tr, "en")
	for range 3 {
		if got := k.T(ctx); got != "Code text" {
			t.Fatalf("with the store closed: %q", got)
		}
	}
	if n := logs.count("reading the store failed"); n != 1 {
		t.Errorf("logged %d times within a minute, want 1", n)
	}
	now = now.Add(61 * time.Second)
	k.T(ctx)
	if n := logs.count("reading the store failed"); n != 2 {
		t.Errorf("logged %d times after a minute, want 2", n)
	}
}

func TestCode(t *testing.T) {
	tr, _ := arabicSetup(t)
	if got := Code(in(tr, "ar")).Code; got != "ar" {
		t.Errorf("Code = %q", got)
	}
	bare := context.WithValue(context.Background(), ctxKey{}, &ctxVal{t: tr})
	if got := Code(bare).Code; got != "en" {
		t.Errorf("Code with a translator but no locale = %q, want the default", got)
	}
	if FromContext(in(tr, "ar")) != tr {
		t.Error("FromContext")
	}
}

func TestResolve_ConcurrentWithLoad(t *testing.T) {
	tr, r := arabicSetup(t)
	putOverride(t, tr, "ar", "k", "عربي")
	k := &key{name: "k", e: r.m["k"]}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				if got := k.T(in(tr, "ar")); got != "عربي" {
					t.Errorf("got %q", got)
					return
				}
			}
		}()
	}
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := newT(t, tr.db, r).Load(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestCatalogMetaOutlivesTransaction(t *testing.T) {
	r := newRegistry()
	r.add("route.x", "Route X", []byte{1, 2, 3})
	tr := newT(t, testDB(t), r)
	mustLoad(t, tr)
	var ce *storepb.CatalogEntry
	_ = tr.db.View(func(tx *bolt.Tx) error { ce = catalogEntry(tr.top(tx), "route.x"); return nil })
	if string(ce.GetMeta()) != "\x01\x02\x03" {
		t.Errorf("meta = %v", ce.GetMeta())
	}
}

func benchSetup(b *testing.B) (*Translator, *key, context.Context) {
	b.Helper()
	tr, r := arabicSetup(b)
	putOverride(b, tr, "ar", "k", "عربي")
	return tr, &key{name: "k", e: r.m["k"]}, in(tr, "ar")
}

func BenchmarkT_OwnLocale(b *testing.B) {
	_, k, ctx := benchSetup(b)
	b.ReportAllocs()
	for b.Loop() {
		k.T(ctx)
	}
}

func BenchmarkT_CodeText(b *testing.B) {
	tr, k, _ := benchSetup(b)
	ctx := in(tr, "ar-EG") // misses in ar-EG, hits in ar
	k2 := &key{name: "n", e: tr.reg.m["n"]}
	b.ReportAllocs()
	for b.Loop() {
		k2.T(ctx) // misses everywhere: three buckets, then the code's text
	}
	_ = k
}

func TestResolve_BaseLanguageMustExist(t *testing.T) {
	r := release("k", "Code text")
	tr := newT(t, testDB(t), r)
	mustLoad(t, tr)
	addLocale(t, tr, "pt-BR") // no "pt" locale
	putOverride(t, tr, "en", "k", "Deployment English")
	if got := T(in(tr, "pt-BR"), "k"); got != "Deployment English" {
		t.Errorf("a missing base language is skipped: %q", got)
	}
}

func TestResolve_UnknownKeyWithPercent(t *testing.T) {
	tr, _ := arabicSetup(t)
	if got := T(in(tr, "ar"), "tax%", 10); got != "tax%" {
		t.Errorf("an unknown key renders as itself, unformatted: %q", got)
	}
	putOverride(t, tr, "ar", "tax%", "الضريبة %d")
	if got := T(in(tr, "ar"), "tax%", 10); got != "الضريبة 10" {
		t.Errorf("a key the catalog never saw has no signature, so any override fits: %q", got)
	}
}

func TestResolve_OrphanUsesCatalogSignature(t *testing.T) {
	db := testDB(t)
	mustLoad(t, newT(t, db, release("old", "%d old items")))
	tr := newT(t, db, release())
	mustLoad(t, tr)
	putOverride(t, tr, "en", "old", "%s wrong verb")
	if got := T(in(tr, "en"), "old", 2); got != "2 old items" {
		t.Errorf("a misfit orphan override is skipped: %q", got)
	}
}
