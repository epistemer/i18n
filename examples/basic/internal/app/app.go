// Package app is the basic example's HTTP handler.
package app

import (
	"fmt"
	"net/http"

	"github.com/epistemer/i18n"
)

// The example's keys, declared next to their use.
var (
	L10nTitle    = i18n.Key("basic.title", "Welcome")
	L10nGreeting = i18n.Key("basic.greeting", "Hello, %s!")
)

// Handler serves a greeting in the request's locale, and lets the visitor
// pick a locale at /locale/{code}.
func Handler(tr *i18n.Translator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "visitor"
		}
		fmt.Fprintf(w, "%s\n%s\n", L10nTitle.T(ctx), L10nGreeting.T(ctx, name))
	})
	mux.HandleFunc("GET /locale/{code}", func(w http.ResponseWriter, r *http.Request) {
		if err := tr.SetCookie(w, r.PathValue("code")); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	return mux
}
