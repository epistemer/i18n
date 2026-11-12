package i18n

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Locale describes a locale. It is a plain value: copy it, compare it with ==.
type Locale struct {
	// Code is the canonical BCP 47 language tag, e.g. "ar", "en", "ar-EG".
	Code string

	// NativeName is the name of the locale in its own language.
	NativeName string

	// FlagCode is an optional identifier for UI display.
	FlagCode string

	// RTL indicates whether text in this locale is written right-to-left.
	RTL bool
}

// canonical validates code as BCP 47 syntax and returns it in canonical case.
// Errors wrap ErrInvalidCode (use fmt.Errorf("%w: %q", ErrInvalidCode, code)).
func canonical(code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
	}

	for i := 0; i < len(code); i++ {
		b := code[i]
		if !isLetter(b) && !isDigit(b) && b != '-' {
			return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
		}
	}

	subtags := strings.Split(code, "-")
	for _, sub := range subtags {
		if sub == "" {
			return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
		}
	}

	// First subtag: 2-8 letters, or the singleton "x" (or "X").
	first := subtags[0]
	if len(first) == 1 {
		if first[0] != 'x' && first[0] != 'X' {
			return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
		}
	} else if len(first) >= 2 && len(first) <= 8 {
		if !isAllLetters(first) {
			return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
		}
	} else {
		return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
	}

	// Subsequent subtags: 1-8 letters or digits.
	for _, sub := range subtags[1:] {
		if len(sub) < 1 || len(sub) > 8 {
			return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
		}
	}

	// A singleton needs a subtag after it ("en-u" is invalid). Outside private
	// use, that subtag can't be another singleton ("en-u-v-foo" is invalid).
	// After "x", every subtag is private use and may be one character ("x-a").
	for i, sub := range subtags {
		if len(sub) != 1 {
			continue
		}
		if i == len(subtags)-1 {
			return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
		}
		if sub == "x" || sub == "X" {
			break
		}
		if len(subtags[i+1]) == 1 {
			return "", fmt.Errorf("%w: %q", ErrInvalidCode, code)
		}
	}

	var inSingleton bool
	result := make([]string, len(subtags))
	for i, sub := range subtags {
		if i == 0 {
			result[i] = strings.ToLower(sub)
			if len(sub) == 1 {
				inSingleton = true
			}
			continue
		}
		if inSingleton {
			result[i] = strings.ToLower(sub)
			continue
		}
		if len(sub) == 1 {
			inSingleton = true
			result[i] = strings.ToLower(sub)
			continue
		}
		// After the language and before any singleton:
		// a 4-letter alphabetic subtag is title case (script, "Hant");
		// a 2-letter alphabetic or 3-digit subtag is uppercase (region, "EG", "419");
		// everything else lowercase.
		if len(sub) == 4 && isAllLetters(sub) {
			result[i] = toTitle(sub)
		} else if (len(sub) == 2 && isAllLetters(sub)) || (len(sub) == 3 && isAllDigits(sub)) {
			result[i] = strings.ToUpper(sub)
		} else {
			result[i] = strings.ToLower(sub)
		}
	}

	return strings.Join(result, "-"), nil
}

// baseLanguage returns the language subtag of a canonical code ("ar" for
// "ar-EG"), or "" when code is a private-use tag ("x-acme") or has no subtag
// after the language (then the base is the code itself: return "" too, so
// callers only try a base that differs from the code).
func baseLanguage(code string) string {
	if code == "x" || strings.HasPrefix(code, "x-") {
		return ""
	}
	idx := strings.IndexByte(code, '-')
	if idx == -1 {
		return ""
	}
	return code[:idx]
}

type langItem struct {
	code string
	q    float64
	pos  int
}

// acceptLanguage parses an HTTP Accept-Language header value into canonical
// codes, best first. It never returns duplicates.
func acceptLanguage(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}

	parts := strings.Split(header, ",")
	var items []langItem
	pos := 0

	for _, part := range parts {
		pieces := strings.Split(part, ";")
		tag := strings.TrimSpace(pieces[0])
		if tag == "" || tag == "*" {
			continue
		}

		q := 1.0
		for _, param := range pieces[1:] {
			param = strings.TrimSpace(param)
			k, v, ok := strings.Cut(param, "=")
			if ok && strings.EqualFold(strings.TrimSpace(k), "q") {
				val := strings.TrimSpace(v)
				if parsed, err := strconv.ParseFloat(val, 64); err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
					q = parsed
				}
				break
			}
		}

		if q <= 0 {
			continue
		}
		if q > 1.0 {
			q = 1.0
		}

		canon, err := canonical(tag)
		if err != nil {
			continue
		}

		items = append(items, langItem{
			code: canon,
			q:    q,
			pos:  pos,
		})
		pos++
	}

	if len(items) == 0 {
		return nil
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].q > items[j].q
	})

	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, it := range items {
		if !seen[it.code] {
			seen[it.code] = true
			result = append(result, it.code)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isAllLetters(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isLetter(s[i]) {
			return false
		}
	}
	return true
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

func toTitle(s string) string {
	b := []byte(strings.ToLower(s))
	if len(b) > 0 && b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b)
}
