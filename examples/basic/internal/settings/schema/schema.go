// Package schema declares the configuration of the basic example.
package schema

import (
	"github.com/epistemer/cfgx/dsl"
	"github.com/epistemer/i18n/i18nschema"
)

// I18n is the library's own fragment: i18n.default_locale, Mutable.
var I18n = i18nschema.Group()

// Bolt is the store file. cfgx keeps Mutable settings in it, and the
// Translator shares it through Store().DB().
var Bolt = dsl.Path("bolt", "Store file").Default("basic.db")

// Server configures the HTTP listener.
var Server = dsl.Group("server", "HTTP server",
	dsl.String("addr", "Listen address").Default("localhost:8080"),
)

// App is the example's schema.
var App = dsl.App("basic", "An i18n example", I18n, Server, dsl.Group("storage", "Storage", Bolt)).
	EnvPrefix("BASIC").
	Store(Bolt).
	Labels() // the settings' labels, as labels/v1 for i18n's generator

// Serve runs the HTTP server.
var Serve = dsl.Command("Serve").Uses(I18n, Server)
