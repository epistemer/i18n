package i18n

import "errors"

// Errors returned by the Translator. Compare with errors.Is: the returned
// error usually wraps one of these with details.
var (
	ErrInvalidCode   = errors.New("i18n: invalid locale code")
	ErrUnknownLocale = errors.New("i18n: unknown locale")
	ErrLocaleExists  = errors.New("i18n: locale already exists")
	ErrSourceLocale  = errors.New("i18n: the source locale can't be deleted")
	ErrUnknownKey    = errors.New("i18n: unknown key")
	ErrFormat        = errors.New("i18n: text doesn't fit the code's format verbs")
)
