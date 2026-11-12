package settings

import (
	"testing"

	"github.com/epistemer/cfgx/cfgxtest"
)

// TestGenerated fails when zz_settings.go is stale against the schema.
func TestGenerated(t *testing.T) { cfgxtest.AssertGenerated(t) }
