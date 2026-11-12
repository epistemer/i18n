package i18n

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestLocales_CanonicalCodes(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	// Create with AR-eg.
	if err := tr.CreateLocale(Locale{Code: "AR-eg", NativeName: "Egyptian Arabic"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}

	// Locale with ar-eg finds it.
	loc, ok := tr.Locale("ar-eg")
	if !ok {
		t.Fatal("Locale(\"ar-eg\") not found")
	}
	if loc.Code != "ar-EG" {
		t.Errorf("Code = %q, want \"ar-EG\"", loc.Code)
	}
	if loc.NativeName != "Egyptian Arabic" {
		t.Errorf("NativeName = %q, want \"Egyptian Arabic\"", loc.NativeName)
	}

	// UpdateLocale with Ar-EG.
	if err := tr.UpdateLocale(Locale{Code: "Ar-EG", NativeName: "Masri", FlagCode: "eg", RTL: true}); err != nil {
		t.Fatalf("UpdateLocale: %v", err)
	}

	loc, ok = tr.Locale("ar-EG")
	if !ok {
		t.Fatal("Locale(\"ar-EG\") not found")
	}
	if loc.NativeName != "Masri" || loc.FlagCode != "eg" || !loc.RTL {
		t.Errorf("after update, got %+v", loc)
	}

	// DeleteLocale with ar-EG.
	if err := tr.DeleteLocale("ar-EG"); err != nil {
		t.Fatalf("DeleteLocale: %v", err)
	}

	if _, ok := tr.Locale("ar-eg"); ok {
		t.Error("Locale(\"ar-eg\") still found after DeleteLocale")
	}
}

func TestLocales_InvalidCodes(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	invalidCodes := []string{
		"",
		" ",
		"bad_code",
		"123-abc",
		"en-u",
		"toolonglanguagecode",
		"-en",
		"en--US",
	}

	for _, code := range invalidCodes {
		t.Run(code, func(t *testing.T) {
			if err := tr.CreateLocale(Locale{Code: code}); !errors.Is(err, ErrInvalidCode) {
				t.Errorf("CreateLocale(%q) error = %v, want ErrInvalidCode", code, err)
			}
			if _, ok := tr.Locale(code); ok {
				t.Errorf("Locale(%q) ok = true, want false", code)
			}
			if err := tr.UpdateLocale(Locale{Code: code}); !errors.Is(err, ErrInvalidCode) {
				t.Errorf("UpdateLocale(%q) error = %v, want ErrInvalidCode", code, err)
			}
			if err := tr.DeleteLocale(code); !errors.Is(err, ErrInvalidCode) {
				t.Errorf("DeleteLocale(%q) error = %v, want ErrInvalidCode", code, err)
			}
			if got := tr.Match(code); got.Code != tr.Default().Code {
				t.Errorf("Match(%q) = %q, want default %q", code, got.Code, tr.Default().Code)
			}
		})
	}
}

func TestLocales_SourceProtection(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	// DeleteLocale on the source fails with ErrSourceLocale.
	if err := tr.DeleteLocale("en"); !errors.Is(err, ErrSourceLocale) {
		t.Fatalf("DeleteLocale(\"en\") error = %v, want ErrSourceLocale", err)
	}
	if err := tr.DeleteLocale("EN"); !errors.Is(err, ErrSourceLocale) {
		t.Fatalf("DeleteLocale(\"EN\") error = %v, want ErrSourceLocale", err)
	}

	// Source locale is left intact.
	loc, ok := tr.Locale("en")
	if !ok || loc.Code != "en" {
		t.Fatalf("source locale missing after failed delete: %+v, ok=%v", loc, ok)
	}
	if tr.Source().Code != "en" {
		t.Fatalf("Source().Code = %q, want \"en\"", tr.Source().Code)
	}

	// UpdateLocale of the source works and Source() reflects it.
	if err := tr.UpdateLocale(Locale{Code: "en", NativeName: "English (US)", FlagCode: "us"}); err != nil {
		t.Fatalf("UpdateLocale: %v", err)
	}
	if got := tr.Source(); got.NativeName != "English (US)" || got.FlagCode != "us" {
		t.Errorf("Source() = %+v, want NativeName \"English (US)\" and FlagCode \"us\"", got)
	}
	if got, ok := tr.Locale("en"); !ok || got.NativeName != "English (US)" {
		t.Errorf("Locale(\"en\") = %+v, ok = %v", got, ok)
	}

	// Locales() reflects the updated source record.
	found := false
	for _, l := range tr.Locales() {
		if l.Code == "en" {
			found = true
			if l.NativeName != "English (US)" || l.FlagCode != "us" {
				t.Errorf("Locales() has source %+v, want updated", l)
			}
		}
	}
	if !found {
		t.Error("Locales() missing source")
	}

	// UpdateLocale with uppercase code also updates source.
	if err := tr.UpdateLocale(Locale{Code: "EN", NativeName: "British English", FlagCode: "gb"}); err != nil {
		t.Fatalf("UpdateLocale(\"EN\"): %v", err)
	}
	if got := tr.Source(); got.NativeName != "British English" || got.FlagCode != "gb" {
		t.Errorf("Source() after UpdateLocale(\"EN\") = %+v", got)
	}
}

func TestLocales_DeleteConfiguredDefault(t *testing.T) {
	db := testDB(t)
	tr := New(db, english, cfg("ar"), withRegistry(release()))
	mustLoad(t, tr)

	if err := tr.CreateLocale(Locale{Code: "ar", NativeName: "Arabic"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}

	if got := tr.Default().Code; got != "ar" {
		t.Fatalf("Default() = %q, want \"ar\"", got)
	}

	if err := tr.DeleteLocale("ar"); err != nil {
		t.Fatalf("DeleteLocale: %v", err)
	}

	if got := tr.Default().Code; got != "en" {
		t.Fatalf("after DeleteLocale(\"ar\"), Default() = %q, want \"en\"", got)
	}
}

func TestLocales_BeforeLoad(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release()) // no mustLoad

	locs := tr.Locales()
	if len(locs) != 1 || locs[0] != english {
		t.Fatalf("Locales before Load = %+v, want [english]", locs)
	}

	// Source bucket doesn't exist yet, so Locale returns false.
	if _, ok := tr.Locale("en"); ok {
		t.Error("Locale(\"en\") before Load ok = true, want false")
	}

	// CreateLocale works before Load.
	if err := tr.CreateLocale(Locale{Code: "fr", NativeName: "Français"}); err != nil {
		t.Fatalf("CreateLocale before Load: %v", err)
	}

	locs = tr.Locales()
	if len(locs) != 2 || locs[0].Code != "en" || locs[1].Code != "fr" {
		t.Fatalf("Locales after CreateLocale before Load = %+v, want [en, fr]", locs)
	}
	if l, ok := tr.Locale("fr"); !ok || l.NativeName != "Français" {
		t.Errorf("Locale(\"fr\") before Load = %+v, ok = %v", l, ok)
	}
}

func TestLocales_SortedAndIncludesCreated(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	codes := []string{"zh", "ar", "de", "fr"}
	for _, c := range codes {
		if err := tr.CreateLocale(Locale{Code: c, NativeName: c}); err != nil {
			t.Fatalf("CreateLocale(%q): %v", c, err)
		}
	}

	locs := tr.Locales()
	var got []string
	for _, l := range locs {
		got = append(got, l.Code)
	}

	want := []string{"ar", "de", "en", "fr", "zh"}
	if !slices.Equal(got, want) {
		t.Errorf("Locales() codes = %v, want %v", got, want)
	}
}

func TestLocales_CreateTwice(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	if err := tr.CreateLocale(Locale{Code: "es", NativeName: "Spanish"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}

	if err := tr.CreateLocale(Locale{Code: "es", NativeName: "Spanish"}); !errors.Is(err, ErrLocaleExists) {
		t.Errorf("CreateLocale second time error = %v, want ErrLocaleExists", err)
	}

	if err := tr.CreateLocale(Locale{Code: "ES", NativeName: "Spanish"}); !errors.Is(err, ErrLocaleExists) {
		t.Errorf("CreateLocale with \"ES\" error = %v, want ErrLocaleExists", err)
	}

	// Source locale already created by Load gives ErrLocaleExists.
	if err := tr.CreateLocale(Locale{Code: "en"}); !errors.Is(err, ErrLocaleExists) {
		t.Errorf("CreateLocale(\"en\") error = %v, want ErrLocaleExists", err)
	}
}

func TestLocales_DeleteRemovesOverrides(t *testing.T) {
	db := testDB(t)
	r := release("welcome", "Welcome to the site")
	tr := newT(t, db, r)
	mustLoad(t, tr)

	if err := tr.CreateLocale(Locale{Code: "fr", NativeName: "Français"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}

	putOverride(t, tr, "fr", "welcome", "Bienvenue sur le site")

	ctx := in(tr, "fr")
	if got := T(ctx, "welcome"); got != "Bienvenue sur le site" {
		t.Fatalf("T with override = %q, want \"Bienvenue sur le site\"", got)
	}

	if err := tr.DeleteLocale("fr"); err != nil {
		t.Fatalf("DeleteLocale: %v", err)
	}
	if _, ok := tr.Locale("fr"); ok {
		t.Fatal("fr still exists after DeleteLocale")
	}

	// Recreate locale. Overrides should be gone.
	if err := tr.CreateLocale(Locale{Code: "fr", NativeName: "Français"}); err != nil {
		t.Fatalf("CreateLocale recreate: %v", err)
	}

	if got := T(ctx, "welcome"); got != "Welcome to the site" {
		t.Fatalf("after recreate, T = %q, want code text \"Welcome to the site\"", got)
	}
}

func TestLocales_Match(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	for _, c := range []string{"ar", "ar-EG", "fr"} {
		if err := tr.CreateLocale(Locale{Code: c, NativeName: c}); err != nil {
			t.Fatalf("CreateLocale(%q): %v", c, err)
		}
	}

	cases := []struct {
		code string
		want string
	}{
		// Exact matches
		{"ar-EG", "ar-EG"},
		{"AR-eg", "ar-EG"},
		{"ar", "ar"},
		{"fr", "fr"},
		{"en", "en"},
		{"EN", "en"},

		// Base-language matches
		{"ar-SA", "ar"},
		{"AR-sa", "ar"},
		{"fr-CA", "fr"},
		{"FR-ca", "fr"},

		// Unknown codes fall back to Default() ("en")
		{"de", "en"},
		{"de-AT", "en"},
		{"ja", "en"},

		// Invalid codes fall back to Default() ("en")
		{"bad!code", "en"},
		{"", "en"},
		{" ", "en"},
	}

	for _, tt := range cases {
		t.Run(tt.code, func(t *testing.T) {
			got := tr.Match(tt.code)
			if got.Code != tt.want {
				t.Errorf("Match(%q) = %q, want %q", tt.code, got.Code, tt.want)
			}
		})
	}
}

func TestLocales_ParseAcceptLanguage(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	if err := tr.CreateLocale(Locale{Code: "ar", NativeName: "Arabic"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}

	// "fr-CH, fr;q=0.9, ar;q=0.8" with ar existing returns ar.
	if got := tr.ParseAcceptLanguage("fr-CH, fr;q=0.9, ar;q=0.8"); got.Code != "ar" {
		t.Errorf("ParseAcceptLanguage = %q, want \"ar\"", got.Code)
	}

	// Empty header returns Default() ("en").
	if got := tr.ParseAcceptLanguage(""); got.Code != "en" {
		t.Errorf("ParseAcceptLanguage(\"\") = %q, want \"en\"", got.Code)
	}
	if got := tr.ParseAcceptLanguage("   "); got.Code != "en" {
		t.Errorf("ParseAcceptLanguage whitespace = %q, want \"en\"", got.Code)
	}

	// Header with no matches returns Default().
	if got := tr.ParseAcceptLanguage("es-MX, es;q=0.9, it;q=0.5"); got.Code != "en" {
		t.Errorf("ParseAcceptLanguage no matches = %q, want \"en\"", got.Code)
	}

	// After creating "fr", base language of "fr-CH" matches "fr" (q=1.0 > 0.8).
	if err := tr.CreateLocale(Locale{Code: "fr", NativeName: "French"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}
	if got := tr.ParseAcceptLanguage("fr-CH, fr;q=0.9, ar;q=0.8"); got.Code != "fr" {
		t.Errorf("ParseAcceptLanguage with fr existing = %q, want \"fr\"", got.Code)
	}

	// q-ordering preferred.
	if got := tr.ParseAcceptLanguage("ar;q=0.9, fr;q=0.5"); got.Code != "ar" {
		t.Errorf("ParseAcceptLanguage q-ordering = %q, want \"ar\"", got.Code)
	}
	if got := tr.ParseAcceptLanguage("fr;q=0.9, ar;q=0.5"); got.Code != "fr" {
		t.Errorf("ParseAcceptLanguage q-ordering = %q, want \"fr\"", got.Code)
	}

	// Base language match from header tag.
	if got := tr.ParseAcceptLanguage("ar-DZ;q=0.9"); got.Code != "ar" {
		t.Errorf("ParseAcceptLanguage base match = %q, want \"ar\"", got.Code)
	}

	// Wildcard and q=0 ignored.
	if got := tr.ParseAcceptLanguage("*, en;q=0, ar;q=0.7"); got.Code != "ar" {
		t.Errorf("ParseAcceptLanguage wildcard and q=0 = %q, want \"ar\"", got.Code)
	}
}

func TestCodes(t *testing.T) {
	// Without translator in ctx.
	if got := Codes(context.Background()); got != nil {
		t.Errorf("Codes without translator = %v, want nil", got)
	}

	// With translator in ctx.
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	if err := tr.CreateLocale(Locale{Code: "ar", NativeName: "Arabic"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}
	if err := tr.CreateLocale(Locale{Code: "fr", NativeName: "French"}); err != nil {
		t.Fatalf("CreateLocale: %v", err)
	}

	ctx := WithContext(context.Background(), tr, english)
	got := Codes(ctx)
	want := tr.Locales()

	if len(got) != len(want) {
		t.Fatalf("len(Codes(ctx)) = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Codes[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLocales_UnknownLocale(t *testing.T) {
	db := testDB(t)
	tr := newT(t, db, release())
	mustLoad(t, tr)

	if err := tr.UpdateLocale(Locale{Code: "es", NativeName: "Spanish"}); !errors.Is(err, ErrUnknownLocale) {
		t.Errorf("UpdateLocale unknown error = %v, want ErrUnknownLocale", err)
	}

	if err := tr.DeleteLocale("es"); !errors.Is(err, ErrUnknownLocale) {
		t.Errorf("DeleteLocale unknown error = %v, want ErrUnknownLocale", err)
	}

	if _, ok := tr.Locale("es"); ok {
		t.Error("Locale(\"es\") ok = true, want false")
	}
}
