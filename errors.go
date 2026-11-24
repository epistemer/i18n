package i18n

import (
	"context"
	"errors"
	"fmt"
)

// Sentinel errors wrapped by the Translator's returned errors. Compare with
// errors.Is.
//
// Every error the Translator returns that wraps one of these is public: it
// implements TKer and Kind() string (for github.com/epistemer/errors), while
// keeping the same Error() string and Unwrap() target. i18n never returns any
// of these variables bare; the variables themselves are not public.
var (
	ErrInvalidCode   = errors.New("i18n: invalid locale code")
	ErrUnknownLocale = errors.New("i18n: unknown locale")
	ErrLocaleExists  = errors.New("i18n: locale already exists")
	ErrSourceLocale  = errors.New("i18n: the source locale can't be deleted")
	ErrUnknownKey    = errors.New("i18n: unknown key")
	ErrFormat        = errors.New("i18n: text doesn't fit the code's format verbs")
)

var (
	keyErrInvalidCode   = errKey("errors.i18n.invalid_code", `"%s" isn't a valid locale code`, "invalid")
	keyErrUnknownLocale = errKey("errors.i18n.unknown_locale", "there's no locale with the code %s", "not_found")
	keyErrLocaleExists  = errKey("errors.i18n.locale_exists", "the locale %s already exists", "conflict")
	keyErrSourceLocale  = errKey("errors.i18n.source_locale", "%s is the source locale, so it can't be deleted", "conflict")
	keyErrUnknownKey    = errKey("errors.i18n.unknown_key", "there's no text with the key %s", "not_found")
	keyErrFormat        = errKey("errors.i18n.format", "the translation's placeholders must match the original's: %s", "invalid")
)

// errorKey is a public error's key and its kind, as errors' Kind.String()
// spells it. The kind is written once, here, for both the key's meta and
// Kind().
type errorKey struct {
	TKer
	kind string
}

func errKey(name, text, kind string) errorKey {
	return errorKey{Key(name, text, []byte(`{"kind":"`+kind+`","args":["string"]}`)...), kind}
}

// pubError is what the Translator returns: Error() for logs, as before
// amendment I1, and T, K and Kind for people through
// github.com/epistemer/errors.
type pubError struct {
	err error // the variable, which Unwrap returns
	key errorKey
	arg string
	msg string
}

func (e *pubError) Error() string { return e.msg }
func (e *pubError) Unwrap() error { return e.err }
func (e *pubError) K() string     { return e.key.K() }
func (e *pubError) Kind() string  { return e.key.kind }
func (e *pubError) T(ctx context.Context, _ ...any) string {
	return e.key.T(ctx, e.arg)
}

func newPubError(sentinel error, key errorKey, arg, msg string) *pubError {
	return &pubError{err: sentinel, key: key, arg: arg, msg: msg}
}

func newDetailError(sentinel error, key errorKey, detail string) *pubError {
	return newPubError(sentinel, key, detail, fmt.Sprintf("%s: %q", sentinel, detail))
}

func invalidCodeError(code string) *pubError {
	return newDetailError(ErrInvalidCode, keyErrInvalidCode, code)
}

func unknownLocaleError(code string) *pubError {
	arg := code
	if canon, err := canonical(code); err == nil {
		arg = canon
	}
	return newPubError(ErrUnknownLocale, keyErrUnknownLocale, arg, fmt.Sprintf("%s: %q", ErrUnknownLocale, code))
}

func localeExistsError(canon string) *pubError {
	return newDetailError(ErrLocaleExists, keyErrLocaleExists, canon)
}

func sourceLocaleError(canon string) *pubError {
	return newDetailError(ErrSourceLocale, keyErrSourceLocale, canon)
}

func unknownKeyError(key string) *pubError {
	return newDetailError(ErrUnknownKey, keyErrUnknownKey, key)
}

// formatError's argument is the code text's verbs. A code text that fmt
// can't parse (such as "%*") can have none and still refuse every
// override; the argument is then the code text itself.
func formatError(text, key, codeText string) *pubError {
	verbs := formatVerbs(codeText)
	if verbs == "" {
		verbs = codeText
	}
	return newPubError(
		ErrFormat,
		keyErrFormat,
		verbs,
		fmt.Sprintf("%s: %q for key %q, whose code text is %q", ErrFormat, text, key, codeText),
	)
}
