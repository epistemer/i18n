package i18nschema

import "github.com/epistemer/cfgx/dsl"

// Group declares the settings i18n reads (i18n.Config), so every app declares
// them the same way:
//
//	var I18n = i18nschema.Group()
//	var Serve = dsl.Command("Serve").Uses(I18n)
//
// That gives the file key i18n.default_locale, the env var
// <PREFIX>_I18N_DEFAULT_LOCALE, the flag --i18n-default-locale and the
// getter I18nDefaultLocale(). The key is Mutable, so the app needs a cfgx
// store (App.Store), and a dashboard can change a deployment's default
// locale without a restart.
func Group() *dsl.GroupDef {
	return dsl.Group("i18n", "Translations",
		dsl.String("default_locale",
			"Locale for visitors without a chosen locale; empty means the source locale").
			Mutable(),
	).GoName("I18n") // pins the getter prefix, whatever the app's initialisms
}
