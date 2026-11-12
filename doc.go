// Package i18n translates the text of Go web apps whose deployments own
// their text.
//
// Keys are declared in code, as package-level variables, with their default
// text in the source language:
//
//	var L10nTitle = i18n.Key("page.title", "Localizations")
//
// People edit any text at runtime, the source language included, and an edit
// always beats the code. Edits are kept in bbolt and read on every lookup, so
// they show up at once.
//
// The full contract is docs/interface.md in the repository.
package i18n
