package i18nschema

import "testing"

// The generated getter is checked end to end by examples/basic, which builds
// an app from this group and asserts that its settings type satisfies
// i18n.Config.
func TestGroup(t *testing.T) {
	if Group() == nil {
		t.Fatal("Group returned nil")
	}
}
