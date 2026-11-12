package i18n

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/epistemer/i18n/internal/storepb"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

func TestLoad_WritesNothingWhenUnchanged(t *testing.T) {
	db := testDB(t)
	r := release("a", "A", "b", "B")
	first := newT(t, db, r)
	mustLoad(t, first)
	if got := first.loadWrites.Load(); got != 3 { // source record + two catalog entries
		t.Errorf("first Load wrote %d records, want 3", got)
	}
	again := newT(t, db, r)
	mustLoad(t, again)
	if got := again.loadWrites.Load(); got != 0 {
		t.Errorf("second Load wrote %d records, want 0", got)
	}
}

func TestLoad_RefreshesCatalogText(t *testing.T) {
	db := testDB(t)
	mustLoad(t, newT(t, db, release("a", "Old")))
	tr := newT(t, db, release("a", "New"))
	mustLoad(t, tr)
	if got := catalogOf(t, tr, "a").GetText(); got != "New" {
		t.Errorf("catalog text = %q, want %q", got, "New")
	}
}

func TestLoad_OrphansAreMarkedAndUnmarked(t *testing.T) {
	db := testDB(t)
	mustLoad(t, newT(t, db, release("a", "A", "b", "B")))

	rollback := newT(t, db, release("a", "A"))
	mustLoad(t, rollback)
	if ce := catalogOf(t, rollback, "b"); ce == nil || !ce.GetOrphaned() || ce.GetText() != "B" {
		t.Fatalf("after rollback, b = %v; want orphaned with its text", ce)
	}
	if ce := catalogOf(t, rollback, "a"); ce.GetOrphaned() {
		t.Error("a is still registered but marked orphaned")
	}

	forward := newT(t, db, release("a", "A", "b", "B"))
	mustLoad(t, forward)
	if ce := catalogOf(t, forward, "b"); ce.GetOrphaned() {
		t.Error("b came back but is still orphaned")
	}
}

func TestLoad_RefusesNewerStore(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)
	_ = db.Update(func(tx *bolt.Tx) error { return tr.top(tx).Put(keyVersion, []byte{2}) })
	err := newT(t, db, release()).Load()
	if err == nil || !strings.Contains(err.Error(), "newer") {
		t.Errorf("Load = %v, want a newer-version error", err)
	}
}

func TestLoad_StoresSourceRecordOnlyOnce(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)
	edited, _ := proto.Marshal(&storepb.LocaleRecord{NativeName: "English (UK)", FlagCode: "gb"})
	_ = db.Update(func(tx *bolt.Tx) error { return sub(tr.top(tx), bktLocales).Put([]byte("en"), edited) })
	again := newT(t, db, release())
	mustLoad(t, again)
	if got := again.Source(); got.NativeName != "English (UK)" || got.FlagCode != "gb" {
		t.Errorf("Source() = %+v; the edited record was overwritten", got)
	}
}

func TestLoad_InvalidSourceCode(t *testing.T) {
	tr := New(testDB(t), Locale{Code: "e n"}, nil, withRegistry(release()))
	if err := tr.Load(); err == nil {
		t.Error("Load accepted an invalid source code")
	}
}

func TestLoad_SourceCodeIsCanonical(t *testing.T) {
	tr := New(testDB(t), Locale{Code: "EN-us"}, nil, withRegistry(release()))
	mustLoad(t, tr)
	if got := tr.Source().Code; got != "en-US" {
		t.Errorf("Source().Code = %q", got)
	}
}

func TestLoad_Warnings(t *testing.T) {
	db := testDB(t)
	mustLoad(t, newT(t, db, release("n", "%d items")))
	setup := newT(t, db, release("n", "%d items"))
	addLocale(t, setup, "ar")
	putOverride(t, setup, "ar", "n", "%d عناصر")

	logs := &logRecorder{}
	// The code changes the verbs; the Arabic override no longer fits.
	tr := New(db, english, cfg("de"), withRegistry(release("n", "%s items")), WithLogger(slog.New(logs)))
	mustLoad(t, tr)
	if logs.count("no longer fits") != 1 {
		t.Errorf("want one misfit warning, got logs %q", logs.msgs)
	}
	if logs.count("default locale doesn't exist") != 1 {
		t.Errorf("want a warning about the missing default locale, got logs %q", logs.msgs)
	}
}

func TestDefault(t *testing.T) {
	db := testDB(t)
	setup := newT(t, db, release())
	mustLoad(t, setup)
	addLocale(t, setup, "ar")

	for _, tt := range []struct{ cfg, want string }{
		{"", "en"},
		{"ar", "ar"},
		{"AR", "ar"},
		{"de", "en"},
		{"not a code", "en"},
	} {
		tr := New(db, english, cfg(tt.cfg), withRegistry(release()))
		if got := tr.Default().Code; got != tt.want {
			t.Errorf("default %q: Default() = %q, want %q", tt.cfg, got, tt.want)
		}
	}
}

func TestLoad_WarnsAboutInvalidDefault(t *testing.T) {
	logs := &logRecorder{}
	tr := New(testDB(t), english, cfg("en_US"), withRegistry(release()), WithLogger(slog.New(logs)))
	mustLoad(t, tr)
	if logs.count("default locale doesn't exist") != 1 {
		t.Errorf("logs %q", logs.msgs)
	}
}

func TestLoad_RecreatesSourceBuckets(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)
	_ = db.Update(func(tx *bolt.Tx) error { return tr.top(tx).DeleteBucket(tr.names("en").text) })
	mustLoad(t, newT(t, db, release()))
	_ = db.View(func(tx *bolt.Tx) error {
		if sub(tr.top(tx), tr.names("en").text) == nil {
			t.Error("Load didn't recreate the source locale's bucket")
		}
		return nil
	})
}

// zpScale registers as many keys as zp has, with zp-like names.
func zpScale() *registry {
	r := newRegistry()
	for i := range 1850 {
		r.add(fmt.Sprintf("table.zp_table_%03d.Column_%04d", i%120, i), fmt.Sprintf("Column %d", i), nil)
	}
	return r
}

func TestLoad_AtZpScale(t *testing.T) {
	db := testDB(t)
	first := newT(t, db, zpScale())
	mustLoad(t, first)
	if got := first.loadWrites.Load(); got != 1851 {
		t.Errorf("first Load wrote %d records, want 1851", got)
	}
	again := newT(t, db, zpScale())
	mustLoad(t, again)
	if got := again.loadWrites.Load(); got != 0 {
		t.Errorf("second Load wrote %d records, want 0", got)
	}
}

func BenchmarkLoad_UnchangedAtZpScale(b *testing.B) {
	db := testDB(b)
	r := zpScale()
	mustLoad(b, newT(b, db, r))
	for b.Loop() {
		mustLoad(b, newT(b, db, r))
	}
}
