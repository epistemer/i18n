package i18n

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	bolt "go.etcd.io/bbolt"
)

// The four bugs found in zp's i18n on 2026-11-12 (docs/status.md).

func TestRegression_ChangedDefaultReachesUneditedText(t *testing.T) {
	db := testDB(t)
	mustLoad(t, newT(t, db, release("greet", "Old text", "kept", "Old kept")))
	old := newT(t, db, release("greet", "Old text", "kept", "Old kept"))
	if err := old.Set("en", "kept", "Edited by the deployment"); err != nil {
		t.Fatal(err)
	}

	tr := newT(t, db, release("greet", "New text", "kept", "New kept"))
	mustLoad(t, tr)
	if got := T(in(tr, "en"), "greet"); got != "New text" {
		t.Errorf("unedited text should follow the code: %q", got)
	}
	if got := T(in(tr, "en"), "kept"); got != "Edited by the deployment" {
		t.Errorf("an edit should beat the code: %q", got)
	}
}

func TestRegression_RollbackKeepsOverrides(t *testing.T) {
	db := testDB(t)
	newer := release("a", "A", "feature", "New feature")
	tr := newT(t, db, newer)
	mustLoad(t, tr)
	if err := tr.CreateLocale(Locale{Code: "ar"}); err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "feature", "ميزة جديدة"); err != nil {
		t.Fatal(err)
	}
	mustLoad(t, newT(t, db, release("a", "A"))) // roll back
	forward := newT(t, db, release("a", "A", "feature", "New feature"))
	mustLoad(t, forward) // roll forward
	if got := T(in(forward, "ar"), "feature"); got != "ميزة جديدة" {
		t.Errorf("after a rollback and roll forward: %q", got)
	}
}

func TestRegression_PercentWithoutArgs(t *testing.T) {
	tr, _ := arabicSetup(t)
	if err := tr.Set("ar", "k", "خصم 50% اليوم"); err != nil {
		t.Fatal(err)
	}
	if got := T(in(tr, "ar"), "k"); got != "خصم 50% اليوم" {
		t.Errorf("got %q", got)
	}
}

func TestRegression_ResetOnKeyWithMetaReturnsText(t *testing.T) {
	r := newRegistry()
	r.add("route.x", "Route X", []byte{0x0a, 0x01, 0x41})
	tr := newT(t, testDB(t), r)
	mustLoad(t, tr)
	if err := tr.Set("en", "route.x", "Edited route"); err != nil {
		t.Fatal(err)
	}
	got, err := tr.Reset("en", "route.x")
	if err != nil || got != "Route X" {
		t.Errorf("Reset = %q, %v; want the text only", got, err)
	}
}

func TestSet_Errors(t *testing.T) {
	tr, _ := arabicSetup(t)
	for _, tt := range []struct {
		name            string
		code, key, text string
		want            error
	}{
		{"misfit", "ar", "n", "%s عناصر", ErrFormat},
		{"unknown key", "ar", "nope", "x", ErrUnknownKey},
		{"unknown locale", "de", "k", "x", ErrUnknownLocale},
		{"invalid code", "a r", "k", "x", ErrInvalidCode},
	} {
		if err := tr.Set(tt.code, tt.key, tt.text); !errors.Is(err, tt.want) {
			t.Errorf("%s: Set = %v, want %v", tt.name, err, tt.want)
		}
	}
	if err := tr.Set("AR", "n", ""); err != nil {
		t.Errorf("an empty override always fits: %v", err)
	}
	if got := T(in(tr, "ar"), "n", 3); got != "" {
		t.Errorf("empty override with args: %q", got)
	}
	if got, _ := tr.Reset("ar", "n"); got != "%d items" {
		t.Errorf("Reset returned %q", got)
	}
}

func TestSet_RecordsBase(t *testing.T) {
	tr, _ := arabicSetup(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(tr.Set("en", "k", "Deployment English"))
	must(tr.Set("ar", "k", "عربي"))
	must(tr.Set("ar-EG", "k", "مصري"))

	bases := map[string]string{}
	for _, code := range []string{"en", "ar", "ar-EG"} {
		es, err := tr.Search(code, Query{Prefix: "k"})
		if err != nil || len(es) != 1 {
			t.Fatalf("Search %s: %v %v", code, es, err)
		}
		bases[code] = es[0].Base
	}
	want := map[string]string{"en": "Code text", "ar": "Deployment English", "ar-EG": "عربي"}
	for code, w := range want {
		if bases[code] != w {
			t.Errorf("base of %s = %q, want %q", code, bases[code], w)
		}
	}
}

func TestEditor_Concurrent(t *testing.T) {
	tr, _ := arabicSetup(t)
	var wg sync.WaitGroup
	for i := range 6 {
		wg.Add(3)
		go func() {
			defer wg.Done()
			for j := range 50 {
				if err := tr.Set("ar", "k", fmt.Sprintf("نص %d-%d", i, j)); err != nil {
					t.Error(err)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for range 50 {
				if _, err := tr.Reset("ar", "k"); err != nil {
					t.Error(err)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for range 100 {
				T(in(tr, "ar"), "k")
			}
		}()
	}
	wg.Wait()
}

func TestOrphanLifecycle(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release("a", "A", "b", "B"))
	mustLoad(t, tr)
	if err := tr.CreateLocale(Locale{Code: "ar"}); err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "b", "ب"); err != nil {
		t.Fatal(err)
	}

	rolled := newT(t, db, release("a", "A"))
	mustLoad(t, rolled)
	if got, _ := rolled.Orphans(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("Orphans = %v", got)
	}
	if es, _ := rolled.Search("ar", Query{}); len(es) != 1 || es[0].Key != "a" {
		t.Errorf("Search should leave orphans out: %+v", es)
	}
	if got := T(in(rolled, "ar"), "b"); got != "ب" {
		t.Errorf("an orphan keeps rendering its override: %q", got)
	}

	removed, err := rolled.Prune()
	if err != nil || !slices.Equal(removed, []string{"b"}) {
		t.Fatalf("Prune = %v, %v", removed, err)
	}
	if got, _ := rolled.Orphans(); len(got) != 0 {
		t.Errorf("Orphans after Prune = %v", got)
	}
	if got := T(in(rolled, "ar"), "b"); got != "b" {
		t.Errorf("a pruned key renders as itself: %q", got)
	}

	back := newT(t, db, release("a", "A", "b", "B"))
	mustLoad(t, back)
	if got := T(in(back, "ar"), "b"); got != "B" {
		t.Errorf("a pruned key that comes back starts clean: %q", got)
	}
}

func TestSearch(t *testing.T) {
	r := newRegistry()
	r.add("page.title", "Localizations", nil)
	r.add("route.auth.login", "Login page", []byte("searchable"))
	r.add("route.auth.logout", "Logout", nil)
	r.add("route.settings", "Settings", []byte("searchable"))
	r.add("count", "%d items", nil)
	tr := newT(t, testDB(t), r)
	mustLoad(t, tr)
	if err := tr.CreateLocale(Locale{Code: "ar"}); err != nil {
		t.Fatal(err)
	}
	_ = tr.Set("ar", "route.settings", "الإعدادات")
	_ = tr.Set("en", "page.title", "Translations")

	keys := func(es []Entry) []string {
		var ks []string
		for _, e := range es {
			ks = append(ks, e.Key)
		}
		return ks
	}

	// zp's global search over routes: a prefix plus a filter on meta.
	es, err := tr.Search("ar", Query{Prefix: "route.", Match: func(e Entry) bool { return string(e.Meta) == "searchable" }})
	if err != nil || !slices.Equal(keys(es), []string{"route.auth.login", "route.settings"}) {
		t.Errorf("route search = %v, %v", keys(es), err)
	}
	// Contains matches the key, the text and the source text.
	for q, want := range map[string][]string{
		"LOGIN":         {"route.auth.login"},
		"الإعدادات":     {"route.settings"},
		"translations":  {"page.title"},
		"localizations": nil,
	} {
		es, _ := tr.Search("ar", Query{Contains: q})
		if !slices.Equal(keys(es), want) {
			t.Errorf("Contains %q = %v, want %v", q, keys(es), want)
		}
	}
	es, _ = tr.Search("ar", Query{Offset: 1, Limit: 2})
	if !slices.Equal(keys(es), []string{"page.title", "route.auth.login"}) {
		t.Errorf("Offset and Limit = %v", keys(es))
	}
	es, _ = tr.Search("ar", Query{Prefix: "page."})
	if e := es[0]; e.Text != "Translations" || e.Source != "Translations" || e.Override {
		t.Errorf("an entry without its own override: %+v", e)
	}
	es, _ = tr.Search("ar", Query{Prefix: "route.settings"})
	if e := es[0]; e.Text != "الإعدادات" || !e.Override || e.Base != "Settings" || e.Source != "Settings" {
		t.Errorf("an entry with an override: %+v", e)
	}
	if _, err := tr.Search("de", Query{}); !errors.Is(err, ErrUnknownLocale) {
		t.Errorf("Search on an unknown locale = %v", err)
	}
}

func TestSearch_Misfit(t *testing.T) {
	db := testDB(t)
	old := newT(t, db, release("n", "%d items"))
	mustLoad(t, old)
	if err := old.Set("en", "n", "%d things"); err != nil {
		t.Fatal(err)
	}
	tr := newT(t, db, release("n", "%s items"))
	mustLoad(t, tr)
	es, _ := tr.Search("en", Query{})
	if len(es) != 1 || !es[0].Misfit {
		t.Errorf("entries = %+v, want a misfit", es)
	}
}

func BenchmarkSearch_AtZpScale(b *testing.B) {
	tr := newT(b, testDB(b), zpScale())
	mustLoad(b, tr)
	if err := tr.CreateLocale(Locale{Code: "ar"}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := tr.Search("ar", Query{Contains: "column 18"}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSearch_MatchWithPaging(t *testing.T) {
	r := newRegistry()
	for i := range 6 {
		r.add(fmt.Sprintf("route.r%d", i), fmt.Sprintf("Route %d", i), []byte{byte(i % 2)})
	}
	tr := newT(t, testDB(t), r)
	mustLoad(t, tr)
	even := func(e Entry) bool { return e.Meta[0] == 0 }
	es, _ := tr.Search("en", Query{Match: even, Offset: 1, Limit: 1})
	if len(es) != 1 || es[0].Key != "route.r2" {
		t.Errorf("the filter applies before Offset and Limit: %+v", es)
	}
}

func TestEditor_OrphanResetAndPruneBases(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release("a", "A", "b", "B"))
	mustLoad(t, tr)
	_ = tr.CreateLocale(Locale{Code: "ar"})
	if err := tr.Set("ar", "b", "ب"); err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "b", "ب ٢"); err != nil {
		t.Fatalf("updating an override: %v", err)
	}
	if got := T(in(tr, "ar"), "b"); got != "ب ٢" {
		t.Errorf("after an update: %q", got)
	}

	rolled := newT(t, db, release("a", "A"))
	mustLoad(t, rolled)
	if got, err := rolled.Reset("ar", "b"); err != nil || got != "B" {
		t.Errorf("Reset on an orphan = %q, %v; want its catalog text", got, err)
	}
	_ = rolled.db.Update(func(tx *bolt.Tx) error {
		return rolled.top(tx).Bucket(rolled.names("ar").base).Put([]byte("b"), []byte("stale"))
	})
	if _, err := rolled.Prune(); err != nil {
		t.Fatal(err)
	}
	_ = rolled.db.View(func(tx *bolt.Tx) error {
		if v := rolled.top(tx).Bucket(rolled.names("ar").base).Get([]byte("b")); v != nil {
			t.Error("Prune left a base record behind")
		}
		return nil
	})
}

func TestPrune_BeforeLoadSparesReturnedKeys(t *testing.T) {
	db := testDB(t)
	mustLoad(t, newT(t, db, release("a", "A", "b", "B")))
	rolled := newT(t, db, release("a", "A"))
	mustLoad(t, rolled)
	_ = rolled.Set("en", "a", "kept")

	// A release that has b again, before its Load has run.
	back := newT(t, db, release("a", "A", "b", "B"))
	if got, _ := back.Orphans(); len(got) != 0 {
		t.Errorf("Orphans before Load = %v; b is registered again", got)
	}
	if removed, _ := back.Prune(); len(removed) != 0 {
		t.Errorf("Prune before Load removed %v", removed)
	}
}

func TestEditor_MisfitFallbacksAreSkipped(t *testing.T) {
	db := testDB(t)
	old := newT(t, db, release("n", "%d items"))
	mustLoad(t, old)
	_ = old.CreateLocale(Locale{Code: "ar"})
	_ = old.CreateLocale(Locale{Code: "ar-EG"})
	if err := old.Set("ar", "n", "%d عناصر"); err != nil {
		t.Fatal(err)
	}
	tr := newT(t, db, release("n", "%s items")) // the verb changed
	mustLoad(t, tr)
	es, _ := tr.Search("ar-EG", Query{})
	if es[0].Text != "%s items" {
		t.Errorf("the ar fallback no longer fits, so ar-EG shows the code's text: %q", es[0].Text)
	}
	if err := tr.Set("ar-EG", "n", "%s مصري"); err != nil {
		t.Fatal(err)
	}
	es, _ = tr.Search("ar-EG", Query{})
	if es[0].Base != "%s items" {
		t.Errorf("the base skips the misfit ar override: %q", es[0].Base)
	}
}
