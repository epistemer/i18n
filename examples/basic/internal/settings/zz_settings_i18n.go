// Code generated from labels by github.com/epistemer/i18n. DO NOT EDIT.

package settings

import "github.com/epistemer/i18n"

var (
	L10nI18n              = i18n.Key("settings.i18n", "Translations")
	L10nI18nDefaultLocale = i18n.Key("settings.i18n.default_locale", "Locale for visitors without a chosen locale; empty means the source locale")
	L10nServer            = i18n.Key("settings.server", "HTTP server")
	L10nServerAddr        = i18n.Key("settings.server.addr", "Listen address")
	L10nStorage           = i18n.Key("settings.storage", "Storage")
	L10nStorageBolt       = i18n.Key("settings.storage.bolt", "Store file")
)

// l10n holds every label that has a path.
var l10n = map[string]i18n.TKer{
	"i18n":                L10nI18n,
	"i18n.default_locale": L10nI18nDefaultLocale,
	"server":              L10nServer,
	"server.addr":         L10nServerAddr,
	"storage":             L10nStorage,
	"storage.bolt":        L10nStorageBolt,
}

// L10n returns the label at path, or nil when there is none.
func L10n(path string) i18n.TKer { return l10n[path] }
