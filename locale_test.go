package i18n

import (
	"errors"
	"reflect"
	"testing"
)

func TestCanonical_Valid(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "spec example: complex extension", code: "EN-us-U-CA-buddhist", want: "en-US-u-ca-buddhist"},
		{name: "spec example: script and region", code: "zh-hant-tw", want: "zh-Hant-TW"},
		{name: "spec example: language uppercase", code: "AR", want: "ar"},
		{name: "spec example: 3-digit region", code: "es-419", want: "es-419"},
		{name: "spec example: private use lower", code: "x-acme", want: "x-acme"},
		{name: "spec example: private use upper", code: "X-ACME", want: "x-acme"},
		{name: "private use with one-letter subtag", code: "x-a", want: "x-a"},
		{name: "private use after language", code: "EN-X-A-b", want: "en-x-a-b"},
		{name: "simple language", code: "en", want: "en"},
		{name: "simple language upper", code: "EN", want: "en"},
		{name: "language and region lowercase", code: "en-us", want: "en-US"},
		{name: "language and region uppercase", code: "EN-US", want: "en-US"},
		{name: "script, region and language mixed", code: "az-latn-az", want: "az-Latn-AZ"},
		{name: "language and script", code: "sr-cyrl", want: "sr-Cyrl"},
		{name: "language and 5-letter variant", code: "sl-rozaj", want: "sl-rozaj"},
		{name: "language and another variant", code: "sl-nedis", want: "sl-nedis"},
		{name: "language and 8-letter variant", code: "ca-valencia", want: "ca-valencia"},
		{name: "private-use extension at end", code: "en-US-x-custom", want: "en-US-x-custom"},
		{name: "digits after singleton lowercase", code: "en-US-u-ca-123", want: "en-US-u-ca-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonical(tt.code)
			if err != nil {
				t.Fatalf("canonical(%q) returned unexpected error: %v", tt.code, err)
			}
			if got != tt.want {
				t.Errorf("canonical(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestCanonical_RejectionRules(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		// Rule: Reject empty code
		{name: "empty code", code: ""},

		// Rule: Reject empty subtags
		{name: "leading hyphen", code: "-en"},
		{name: "trailing hyphen", code: "en-"},
		{name: "consecutive hyphens", code: "en--us"},
		{name: "hyphen only", code: "-"},
		{name: "double hyphen only", code: "--"},

		// Rule: Reject any byte other than ASCII letters, digits and "-"
		{name: "underscore", code: "en_US"},
		{name: "dot", code: "en.US"},
		{name: "space in tag", code: "en us"},
		{name: "symbol @", code: "en@US"},
		{name: "symbol !", code: "en-US!"},
		{name: "non-ASCII unicode", code: "en-é"},
		{name: "newline", code: "en\nUS"},
		{name: "tab", code: "en\tUS"},

		// Rule: First subtag is 2-8 letters, or singleton "x"
		{name: "first subtag 1 letter not x: a", code: "a"},
		{name: "first subtag 1 letter not x: a-us", code: "a-us"},
		{name: "first subtag 1 letter not x: u-ca", code: "u-ca"},
		{name: "first subtag digits only: 12", code: "12"},
		{name: "first subtag alphanumeric: e1", code: "e1"},
		{name: "first subtag alphanumeric: 1en", code: "1en"},
		{name: "first subtag alphanumeric: en1", code: "en1"},
		{name: "first subtag > 8 letters", code: "toolonglanguage"},
		{name: "first subtag > 8 letters with region", code: "toolonglanguage-US"},

		// Rule: Every other subtag is 1-8 letters or digits
		{name: "second subtag > 8 chars", code: "en-toolongsubtag"},
		{name: "third subtag > 8 chars", code: "zh-Hant-toolongsubtag"},

		// Rule: After a singleton, at least one subtag must follow ("en-u" is invalid)
		{name: "singleton u at end", code: "en-u"},
		{name: "singleton x alone", code: "x"},
		{name: "singleton X alone", code: "X"},
		{name: "singleton u at end after region", code: "en-US-u"},
		{name: "singleton x at end", code: "en-x"},
		{name: "singleton followed by singleton", code: "en-u-v-foo"},
		{name: "singleton at end after extension", code: "en-US-u-ca-buddhist-x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonical(tt.code)
			if err == nil {
				t.Fatalf("canonical(%q) = %q, expected error wrapping ErrInvalidCode", tt.code, got)
			}
			if !errors.Is(err, ErrInvalidCode) {
				t.Errorf("canonical(%q) error = %v, does not wrap ErrInvalidCode", tt.code, err)
			}
		})
	}
}

func TestBaseLanguage(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "plain code", code: "en", want: ""},
		{name: "plain arabic", code: "ar", want: ""},
		{name: "regional code: ar-EG", code: "ar-EG", want: "ar"},
		{name: "regional code: en-US", code: "en-US", want: "en"},
		{name: "regional with numeric region: es-419", code: "es-419", want: "es"},
		{name: "script and region: zh-Hant-TW", code: "zh-Hant-TW", want: "zh"},
		{name: "complex extension: en-US-u-ca-buddhist", code: "en-US-u-ca-buddhist", want: "en"},
		{name: "private-use tag: x-acme", code: "x-acme", want: ""},
		{name: "private-use tag: x-custom-locale", code: "x-custom-locale", want: ""},
		{name: "private-use singleton: x", code: "x", want: ""},
		{name: "empty string", code: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := baseLanguage(tt.code)
			if got != tt.want {
				t.Errorf("baseLanguage(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestAcceptLanguage(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   []string
	}{
		{
			name:   "spec example: whitespace and q values",
			header: "fr-CH, fr;q=0.9, en;q=0.8",
			want:   []string{"fr-CH", "fr", "en"},
		},
		{
			name:   "extra whitespace around delimiters and pieces",
			header: "  fr-CH  ,   fr ;  q=0.9  ,  en ; q = 0.8  ",
			want:   []string{"fr-CH", "fr", "en"},
		},
		{
			name:   "q=0 drops tag",
			header: "fr, en;q=0",
			want:   []string{"fr"},
		},
		{
			name:   "q=0.0 drops tag",
			header: "fr, en;q=0.0",
			want:   []string{"fr"},
		},
		{
			name:   "negative q drops tag",
			header: "fr, en;q=-0.5",
			want:   []string{"fr"},
		},
		{
			name:   "wildcard * dropped",
			header: "*, fr-CH, *;q=0.5",
			want:   []string{"fr-CH"},
		},
		{
			name:   "invalid tags dropped",
			header: "fr-CH, 1234;q=0.9, en_US;q=0.8, en",
			want:   []string{"fr-CH", "en"},
		},
		{
			name:   "invalid q parsed as 1",
			header: "fr;q=bogus, de;q=0.5",
			want:   []string{"fr", "de"},
		},
		{
			name:   "q clamped to at most 1",
			header: "fr;q=2.0, en",
			want:   []string{"fr", "en"},
		},
		{
			name:   "duplicates after canonicalization",
			header: "en-us, EN-US;q=0.5",
			want:   []string{"en-US"},
		},
		{
			name:   "duplicates with higher q appearing later",
			header: "en-US;q=0.5, en-us;q=0.9",
			want:   []string{"en-US"},
		},
		{
			name:   "equal q keeps header order",
			header: "fr, de, es",
			want:   []string{"fr", "de", "es"},
		},
		{
			name:   "equal custom q keeps header order",
			header: "fr;q=0.7, de;q=0.7, es;q=0.7",
			want:   []string{"fr", "de", "es"},
		},
		{
			name:   "sorting by q descending",
			header: "en;q=0.7, fr;q=0.9, de;q=0.8",
			want:   []string{"fr", "de", "en"},
		},
		{
			name:   "empty header returns nil",
			header: "",
			want:   nil,
		},
		{
			name:   "whitespace-only header returns nil",
			header: "   ",
			want:   nil,
		},
		{
			name:   "all-invalid header returns nil",
			header: "*, 1234, en_US, en;q=0",
			want:   nil,
		},
		{
			name:   "case-insensitive parameter name Q",
			header: "fr;Q=0.9, en;Q=0.8",
			want:   []string{"fr", "en"},
		},
		{
			name:   "multiple parameters on tag",
			header: "fr;level=1;q=0.9, en",
			want:   []string{"en", "fr"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := acceptLanguage(tt.header)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("acceptLanguage(%q) = %#v, want %#v", tt.header, got, tt.want)
			}
		})
	}
}

func TestLocale_Value(t *testing.T) {
	l1 := Locale{
		Code:       "ar-EG",
		NativeName: "العربية (مصر)",
		FlagCode:   "eg",
		RTL:        true,
	}

	l2 := l1
	if l1 != l2 {
		t.Errorf("Locale copy failed: l1 != l2")
	}

	l3 := Locale{
		Code:       "ar-EG",
		NativeName: "العربية (مصر)",
		FlagCode:   "eg",
		RTL:        true,
	}
	if l1 != l3 {
		t.Errorf("Locale comparison with == failed: l1 != l3")
	}
}
