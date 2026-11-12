package i18n

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestFits(t *testing.T) {
	tests := []struct {
		name     string
		code, ov string
		want     bool
	}{
		{"no verbs in code: anything fits", "Hello", "50% off", true},
		{"no verbs in code: broken text fits", "Hello", "%", true},
		{"same verb", "Welcome %s", "مرحبا %s", true},
		{"different verb", "%d items", "%s items", false},
		{"flags, width and precision ignored", "%.2f", "%8.1f", true},
		{"literal percent", "%d%% off", "خصم %d%%", true},
		{"lone percent with verbs in code", "%d%% off", "%d% off", false},
		{"trailing percent", "%d items", "%d items %", false},
		{"missing argument without index", "%s by %s", "%s", false},
		{"extra argument", "%s", "%s %s", false},
		{"reordered", "%s by %d", "%[2]d من %[1]s", true},
		{"reordered with wrong verb", "%s by %d", "%[2]s %[1]s", false},
		{"omission through an index", "%d items selected by %s", "%[1]d عناصر", true},
		{"index out of the code's range", "%s", "%[2]s", false},
		{"zero index", "%s", "%[0]s", false},
		{"malformed index", "%s", "%[x]s", false},
		{"argument reused with two verbs", "%[1]d (0x%[1]x)", "%[1]x", true},
		{"reused argument, other verb", "%[1]d (0x%[1]x)", "%[1]s", false},
		{"star width", "%*d", "%*d", true},
		{"star width dropped", "%*d", "%d", false},
		{"three index slots", "%[3]*.[2]*[1]f", "%[3]*.[2]*[1]f", true},
		{"empty override", "%d items", "", false},
		{"index then digits is bad", "%s", "%[1]2s", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fits(parseSignature(tt.code), parseSignature(tt.ov)); got != tt.want {
				t.Errorf("fits(%q, %q) = %v, want %v", tt.code, tt.ov, got, tt.want)
			}
		})
	}
}

func TestParseSignature(t *testing.T) {
	tests := []struct {
		format    string
		uses      string // "arg:verbs" pairs, sorted
		reordered bool
		broken    bool
	}{
		{"plain", "", false, false},
		{"%s and %d", "0:s 1:d", false, false},
		{"%%", "", false, false},
		{"100%", "", false, true},
		{"%[2]d %[1]s", "0:s 1:d", true, false},
		{"%[1]d %[1]x", "0:dx", true, false},
		{"%*.*f", "0:* 1:* 2:f", false, false},
		{"%[3]*.[2]*[1]f", "0:f 1:* 2:*", true, false},
		{"%-08.3f", "0:f", false, false},
		{"%[x]d", "", true, true},
		{"%[2", "", true, true},
		{"%ä", "0:ä", false, false},
	}
	for _, tt := range tests {
		sig := parseSignature(tt.format)
		var pairs []string
		for arg, verbs := range sig.uses {
			var vs []string
			for v := range verbs {
				vs = append(vs, string(v))
			}
			sort.Strings(vs)
			pairs = append(pairs, fmt.Sprintf("%d:%s", arg, strings.Join(vs, "")))
		}
		sort.Strings(pairs)
		if got := strings.Join(pairs, " "); got != tt.uses || sig.reordered != tt.reordered || sig.broken != tt.broken {
			t.Errorf("parseSignature(%q) = {%s reordered=%v broken=%v}, want {%s reordered=%v broken=%v}",
				tt.format, got, sig.reordered, sig.broken, tt.uses, tt.reordered, tt.broken)
		}
	}
}

func TestFormat(t *testing.T) {
	if got := format("50% off", nil); got != "50% off" {
		t.Errorf("no args: got %q", got)
	}
	if got := format("", []any{5}); got != "" {
		t.Errorf("empty text with args: got %q", got)
	}
	if got := format("%d items", []any{3}); got != "3 items" {
		t.Errorf("with args: got %q", got)
	}
}

// argFor returns a value every verb in verbs formats without an error, or
// false when the verbs need different types.
func argFor(verbs map[rune]bool) (any, bool) {
	kind := ""
	for v := range verbs {
		var k string
		switch v {
		case 'd', 'b', 'o', 'O', 'x', 'X', 'c', 'U', 'q', '*':
			k = "int"
		case 's':
			k = "string"
		case 'e', 'E', 'f', 'F', 'g', 'G':
			k = "float"
		case 't':
			k = "bool"
		case 'v':
			continue
		default:
			return nil, false
		}
		if k == "int" && v == 'q' {
			k = "string"
		}
		if kind != "" && kind != k {
			return nil, false
		}
		kind = k
	}
	switch kind {
	case "string":
		return "a", true
	case "float":
		return 1.5, true
	case "bool":
		return true, true
	default:
		return 3, true
	}
}

// FuzzFit checks fits against fmt itself: when fits says an override can
// replace the code's text, formatting it with arguments that suit the code's
// text must not produce a %! error.
func FuzzFit(f *testing.F) {
	seeds := [][2]string{
		{"%s by %d", "%[2]d %[1]s"},
		{"%d items", "%d عناصر"},
		{"%[1]d (0x%[1]x)", "%[1]x"},
		{"%*d", "%*d"},
		{"%d%% off", "%d%%"},
		{"%.2f", "%v"},
		{"%[3]*.[2]*[1]f", "%[1]f"},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, code, ov string) {
		cs := parseSignature(code)
		if cs.broken || !cs.hasArgs() {
			return
		}
		maxArg := -1
		for arg := range cs.uses {
			maxArg = max(maxArg, arg)
		}
		if maxArg > 8 {
			return
		}
		args := make([]any, maxArg+1)
		for i := range args {
			a, ok := argFor(cs.uses[i])
			if !ok {
				return
			}
			args[i] = a
		}
		if out := fmt.Sprintf(code, args...); strings.Contains(out, "%!") {
			return // the code's own text doesn't format cleanly
		}
		if fits(cs, parseSignature(ov)) {
			if out := fmt.Sprintf(ov, args...); strings.Contains(out, "%!") && !strings.Contains(ov, "%!") {
				t.Errorf("fits(%q, %q), but formatting gives %q", code, ov, out)
			}
		}
	})
}
