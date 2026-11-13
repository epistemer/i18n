package settings

import (
	"testing"

	"github.com/epistemer/genx/genxtest"
)

// TestGenerated fails when a generated file is stale: zz_settings.go and
// the labels file against the schema (cfgx gen -check), and
// zz_settings_i18n.go against the labels (i18n's generator).
func TestGenerated(t *testing.T) { genxtest.AssertGenerated(t) }
