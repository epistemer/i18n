package i18n

import (
	"fmt"
	"unicode/utf8"
)

// signature records how a format string uses its arguments, parsed the way
// fmt parses it. See docs/implementation.md, section 5.
type signature struct {
	uses      map[int]map[rune]bool // argument index → the verbs it's used with ('*' for a width or precision)
	reordered bool                  // an explicit argument index appears
	broken    bool                  // fmt would print a %! error for it
}

// hasArgs reports whether the format uses any argument.
func (s signature) hasArgs() bool { return len(s.uses) > 0 }

func (s *signature) use(arg int, verb rune) {
	if s.uses == nil {
		s.uses = make(map[int]map[rune]bool)
	}
	if s.uses[arg] == nil {
		s.uses[arg] = make(map[rune]bool)
	}
	s.uses[arg][verb] = true
}

// parseSignature parses format as fmt.Sprintf would, without arguments.
func parseSignature(format string) signature {
	var sig signature
	argNum := 0
	end := len(format)
	for i := 0; i < end; {
		if format[i] != '%' {
			i++
			continue
		}
		i++
		if i >= end {
			sig.broken = true // %!(NOVERB)
			break
		}
		for i < end && isFlag(format[i]) {
			i++
		}
		good := true

		// An index, then a '*' width or a numeric width.
		var afterIndex bool
		argNum, i, afterIndex, good = argIndex(&sig, format, i, argNum, good)
		if i < end && format[i] == '*' {
			sig.use(argNum, '*')
			argNum++
			i++
			afterIndex = false
		} else {
			start := i
			var ok bool
			if i, ok = digits(format, i); !ok {
				sig.broken = true // fmt skips the rest of the string: %!(NOVERB)
				break
			}
			if afterIndex && i > start { // "%[3]2d"
				good = false
			}
		}

		// A precision: '.', an index, then '*' or digits.
		if i < end && format[i] == '.' {
			i++
			if afterIndex { // "%[3].2d"
				good = false
			}
			argNum, i, afterIndex, good = argIndex(&sig, format, i, argNum, good)
			if i < end && format[i] == '*' {
				sig.use(argNum, '*')
				argNum++
				i++
				afterIndex = false
			} else {
				var ok bool
				if i, ok = digits(format, i); !ok {
					sig.broken = true
					break
				}
			}
		}

		if !afterIndex {
			argNum, i, _, good = argIndex(&sig, format, i, argNum, good)
		}
		if i >= end {
			sig.broken = true // %!(NOVERB)
			break
		}
		verb, size := utf8.DecodeRuneInString(format[i:])
		i += size
		switch {
		case verb == '%':
			// A literal percent sign; it uses no argument.
		case !good:
			sig.broken = true // %!v(BADINDEX)
		default:
			sig.use(argNum, verb)
			argNum++
		}
	}
	return sig
}

// argIndex reads an optional "[n]" at format[i:], as fmt's argNumber does.
// A malformed or zero index makes the verb bad.
func argIndex(sig *signature, format string, i, argNum int, good bool) (int, int, bool, bool) {
	if i >= len(format) || format[i] != '[' {
		return argNum, i, false, good
	}
	sig.reordered = true
	for j := i + 1; j < len(format); j++ {
		if format[j] == ']' {
			n, ok := atoi(format[i+1 : j])
			if !ok {
				return argNum, j + 1, false, false // "[x]"
			}
			if n < 1 {
				return argNum, j + 1, true, false // "[0]"
			}
			return n - 1, j + 1, true, good
		}
	}
	// No closing bracket: fmt reports a bad index and resumes after '['.
	return argNum, i + 1, false, false
}

// digits skips a run of digits at format[i:], as fmt's parsenum does. A
// number over 1e6 is too large for fmt, which then gives up on the string.
func digits(format string, i int) (int, bool) {
	n := 0
	for i < len(format) && isDigit(format[i]) {
		n = n*10 + int(format[i]-'0')
		if n > 1e6 {
			return i, false
		}
		i++
	}
	return i, true
}

// atoi parses an argument index as fmt does: digits only, at most 1e6.
func atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
		if n > 1e6 {
			return 0, false
		}
	}
	return n, true
}

func isFlag(b byte) bool {
	switch b {
	case '+', '-', '#', ' ', '0':
		return true
	}
	return false
}

// fits reports whether an override with signature ov can replace text with
// signature code when T is called with arguments (interface §5.2).
func fits(code, ov signature) bool {
	if !code.hasArgs() {
		return true
	}
	if ov.broken {
		return false
	}
	for arg, verbs := range ov.uses {
		for verb := range verbs {
			if !code.uses[arg][verb] {
				return false
			}
		}
	}
	if !ov.reordered {
		// Without an explicit index fmt reports unused arguments as EXTRA.
		for arg := range code.uses {
			if ov.uses[arg] == nil {
				return false
			}
		}
	}
	return true
}

// fitsText reports whether the override text fits sig. An empty override
// always fits: it renders empty, with or without args (interface §5.2).
func fitsText(sig signature, text string) bool {
	return text == "" || fits(sig, parseSignature(text))
}

// format applies args to text. With no args, or empty text, text is returned
// as is (interface §5.2).
func format(text string, args []any) string {
	if len(args) == 0 || text == "" {
		return text
	}
	return fmt.Sprintf(text, args...)
}
