package i18n_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epistemer/i18n"
	bolt "go.etcd.io/bbolt"
)

var (
	errTestItems     = i18n.Key("errtest.items", "%d items")
	errTestReordered = i18n.Key("errtest.reordered", "%[2]d %[1]s and %[2]d")
)

type publicError interface {
	error
	i18n.TKer
	Kind() string
}

func openTestTranslator(t *testing.T) *i18n.Translator {
	t.Helper()
	db, err := bolt.Open(filepath.Join(t.TempDir(), "errors.db"), 0o600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	tr := i18n.New(db, english, nil)
	must(tr.Load())
	return tr
}

func TestPublicErrors_AllSix(t *testing.T) {
	tr := openTestTranslator(t)
	must(tr.CreateLocale(i18n.Locale{Code: "ar", NativeName: "العربية"}))
	ctx := i18n.WithContext(context.Background(), tr, tr.Source())

	tests := []struct {
		name     string
		trigger  func() error
		sentinel error
		wantErr  string
		wantKey  string
		wantKind string
		wantText string
	}{
		{
			name:     "ErrInvalidCode",
			trigger:  func() error { return tr.CreateLocale(i18n.Locale{Code: "a r"}) },
			sentinel: i18n.ErrInvalidCode,
			wantErr:  `i18n: invalid locale code: "a r"`,
			wantKey:  "errors.i18n.invalid_code",
			wantKind: "invalid",
			wantText: `"a r" isn't a valid locale code`,
		},
		{
			name:     "ErrUnknownLocale",
			trigger:  func() error { return tr.UpdateLocale(i18n.Locale{Code: "DE"}) },
			sentinel: i18n.ErrUnknownLocale,
			wantErr:  `i18n: unknown locale: "de"`,
			wantKey:  "errors.i18n.unknown_locale",
			wantKind: "not_found",
			wantText: "there's no locale with the code de",
		},
		{
			name:     "ErrLocaleExists",
			trigger:  func() error { return tr.CreateLocale(i18n.Locale{Code: "AR"}) },
			sentinel: i18n.ErrLocaleExists,
			wantErr:  `i18n: locale already exists: "ar"`,
			wantKey:  "errors.i18n.locale_exists",
			wantKind: "conflict",
			wantText: "the locale ar already exists",
		},
		{
			name:     "ErrSourceLocale",
			trigger:  func() error { return tr.DeleteLocale("EN") },
			sentinel: i18n.ErrSourceLocale,
			wantErr:  `i18n: the source locale can't be deleted: "en"`,
			wantKey:  "errors.i18n.source_locale",
			wantKind: "conflict",
			wantText: "en is the source locale, so it can't be deleted",
		},
		{
			name:     "ErrUnknownKey",
			trigger:  func() error { return tr.Set("ar", "no.such.key", "x") },
			sentinel: i18n.ErrUnknownKey,
			wantErr:  `i18n: unknown key: "no.such.key"`,
			wantKey:  "errors.i18n.unknown_key",
			wantKind: "not_found",
			wantText: "there's no text with the key no.such.key",
		},
		{
			name:     "ErrFormat",
			trigger:  func() error { return tr.Set("ar", errTestItems.K(), "%s items") },
			sentinel: i18n.ErrFormat,
			wantErr:  `i18n: text doesn't fit the code's format verbs: "%s items" for key "errtest.items", whose code text is "%d items"`,
			wantKey:  "errors.i18n.format",
			wantKind: "invalid",
			wantText: "the translation's placeholders must match the original's: %d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false", err, tt.sentinel)
			}
			if errors.Unwrap(err) != tt.sentinel {
				t.Errorf("errors.Unwrap(%v) = %v, want %v", err, errors.Unwrap(err), tt.sentinel)
			}
			if got := err.Error(); got != tt.wantErr {
				t.Errorf("Error() = %q, want %q", got, tt.wantErr)
			}
			pub, ok := err.(publicError)
			if !ok {
				t.Fatalf("%T does not implement TKer + Kind() string", err)
			}
			if got := pub.K(); got != tt.wantKey {
				t.Errorf("K() = %q, want %q", got, tt.wantKey)
			}
			if got := pub.Kind(); got != tt.wantKind {
				t.Errorf("Kind() = %q, want %q", got, tt.wantKind)
			}
			if got := pub.T(ctx); got != tt.wantText {
				t.Errorf("T(ctx) = %q, want %q", got, tt.wantText)
			}
			if got := pub.T(ctx, "ignored", 42); got != tt.wantText {
				t.Errorf("T(ctx, args...) = %q, want %q (must ignore args)", got, tt.wantText)
			}
		})
	}
}

func TestPublicErrors_CallSites(t *testing.T) {
	tr := openTestTranslator(t)
	must(tr.CreateLocale(i18n.Locale{Code: "ar"}))
	ctx := i18n.WithContext(context.Background(), tr, tr.Source())

	// Cookie with valid non-canonical unknown code keeps raw code in Error()
	// and uses canonical code in T.
	_, err := tr.Cookie("DE")
	if !errors.Is(err, i18n.ErrUnknownLocale) {
		t.Fatalf("Cookie(DE): %v", err)
	}
	if err.Error() != `i18n: unknown locale: "DE"` {
		t.Errorf("Cookie(DE).Error() = %q", err.Error())
	}
	if got := err.(publicError).T(ctx); got != "there's no locale with the code de" {
		t.Errorf("Cookie(DE).T(ctx) = %q", got)
	}

	// Cookie with invalid code keeps raw code in both Error() and T.
	_, err = tr.Cookie("not valid")
	if !errors.Is(err, i18n.ErrUnknownLocale) {
		t.Fatalf("Cookie(not valid): %v", err)
	}
	if err.Error() != `i18n: unknown locale: "not valid"` {
		t.Errorf("Cookie(not valid).Error() = %q", err.Error())
	}
	if got := err.(publicError).T(ctx); got != "there's no locale with the code not valid" {
		t.Errorf("Cookie(not valid).T(ctx) = %q", got)
	}

	// Set, Reset, Search, DeleteLocale all return public errors.
	for _, tc := range []struct {
		name     string
		err      error
		sentinel error
	}{
		{"Set invalid", tr.Set("a r", errTestItems.K(), "%d x"), i18n.ErrInvalidCode},
		{"Set unknown locale", tr.Set("DE", errTestItems.K(), "%d x"), i18n.ErrUnknownLocale},
		{"Reset invalid", func() error { _, e := tr.Reset("a r", errTestItems.K()); return e }(), i18n.ErrInvalidCode},
		{"Reset unknown", func() error { _, e := tr.Reset("DE", errTestItems.K()); return e }(), i18n.ErrUnknownLocale},
		{"Search invalid", func() error { _, e := tr.Search("a r", i18n.Query{}); return e }(), i18n.ErrInvalidCode},
		{"Search unknown", func() error { _, e := tr.Search("DE", i18n.Query{}); return e }(), i18n.ErrUnknownLocale},
		{"UpdateLocale invalid", tr.UpdateLocale(i18n.Locale{Code: "a r"}), i18n.ErrInvalidCode},
		{"DeleteLocale invalid", tr.DeleteLocale("a r"), i18n.ErrInvalidCode},
		{"DeleteLocale unknown", tr.DeleteLocale("DE"), i18n.ErrUnknownLocale},
	} {
		if !errors.Is(tc.err, tc.sentinel) {
			t.Errorf("%s: errors.Is = false (%v)", tc.name, tc.err)
		}
		if _, ok := tc.err.(publicError); !ok {
			t.Errorf("%s: %T is not public", tc.name, tc.err)
		}
	}

	// Load with an invalid source code also returns a public ErrInvalidCode.
	db, err := bolt.Open(filepath.Join(t.TempDir(), "bad.db"), 0o600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	loadErr := i18n.New(db, i18n.Locale{Code: "e n"}, nil).Load()
	if !errors.Is(loadErr, i18n.ErrInvalidCode) {
		t.Fatalf("Load invalid source: %v", loadErr)
	}
	if loadErr.Error() != `source locale: i18n: invalid locale code: "e n"` {
		t.Errorf("Load.Error() = %q", loadErr.Error())
	}
	// Load adds context with fmt.Errorf; the public error stays in the tree,
	// where github.com/epistemer/errors finds it.
	var pub publicError
	if !errors.As(loadErr, &pub) || pub.Kind() != "invalid" || pub.T(ctx) != `"e n" isn't a valid locale code` {
		t.Errorf("Load public error = %#v", loadErr)
	}
}

func TestPublicErrors_TranslatedAndFormatVerbs(t *testing.T) {
	tr := openTestTranslator(t)
	must(tr.CreateLocale(i18n.Locale{Code: "ar", NativeName: "العربية"}))

	must(tr.Set("ar", "errors.i18n.unknown_locale", "لا توجد لغة بالرمز %s"))
	_, err := tr.Cookie("fr")
	arCtx := i18n.WithContext(context.Background(), tr, tr.Match("ar"))
	if got := err.(i18n.Ter).T(arCtx); got != "لا توجد لغة بالرمز fr" {
		t.Errorf("translated ErrUnknownLocale = %q", got)
	}

	// ErrFormat from Set names the verbs; for "%d items" it ends with ": %d".
	srcCtx := i18n.WithContext(context.Background(), tr, tr.Source())
	fmtErr := tr.Set("ar", errTestItems.K(), "%s")
	if got := fmtErr.(i18n.Ter).T(srcCtx); !strings.HasSuffix(got, ": %d") {
		t.Errorf("ErrFormat.T for %%d items = %q, want suffix \": %%d\"", got)
	}

	// Distinct verbs in order of first appearance.
	fmtErr = tr.Set("ar", errTestReordered.K(), "%s")
	if got := fmtErr.(i18n.Ter).T(srcCtx); !strings.HasSuffix(got, ": %[2]d %[1]s") {
		t.Errorf("ErrFormat.T for reordered = %q, want suffix \": %%[2]d %%[1]s\"", got)
	}
}

func TestPublicErrors_BareVariablesNotTer(t *testing.T) {
	for _, err := range []error{
		i18n.ErrInvalidCode,
		i18n.ErrUnknownLocale,
		i18n.ErrLocaleExists,
		i18n.ErrSourceLocale,
		i18n.ErrUnknownKey,
		i18n.ErrFormat,
	} {
		if _, ok := any(err).(i18n.Ter); ok {
			t.Errorf("bare variable %v must not implement Ter", err)
		}
		if _, ok := any(err).(interface{ Kind() string }); ok {
			t.Errorf("bare variable %v must not implement Kind() string", err)
		}
	}
}

func TestPublicErrors_CatalogMeta(t *testing.T) {
	tr := openTestTranslator(t)
	entries, err := tr.Search("en", i18n.Query{Prefix: "errors.i18n."})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		text string
		meta string
	}{
		"errors.i18n.format":         {"the translation's placeholders must match the original's: %s", `{"kind":"invalid","args":["string"]}`},
		"errors.i18n.invalid_code":   {`"%s" isn't a valid locale code`, `{"kind":"invalid","args":["string"]}`},
		"errors.i18n.locale_exists":  {"the locale %s already exists", `{"kind":"conflict","args":["string"]}`},
		"errors.i18n.source_locale":  {"%s is the source locale, so it can't be deleted", `{"kind":"conflict","args":["string"]}`},
		"errors.i18n.unknown_key":    {"there's no text with the key %s", `{"kind":"not_found","args":["string"]}`},
		"errors.i18n.unknown_locale": {"there's no locale with the code %s", `{"kind":"not_found","args":["string"]}`},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d catalog entries under errors.i18n., want %d", len(entries), len(want))
	}
	for _, e := range entries {
		w, ok := want[e.Key]
		if !ok {
			t.Errorf("unexpected key %q", e.Key)
			continue
		}
		if e.Text != w.text {
			t.Errorf("%s text = %q, want %q", e.Key, e.Text, w.text)
		}
		if string(e.Meta) != w.meta {
			t.Errorf("%s meta = %s, want %s", e.Key, e.Meta, w.meta)
		}
	}
}
