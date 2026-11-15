package storepb_test

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	_ "github.com/epistemer/i18n/internal/storepb"
)

// protobuf-go registers every .proto file by its path in one registry per
// process, and panics at start when two packages register the same path. A
// bare "records.proto" collided with another library's file of that name,
// so i18n's path must carry its own name.
func TestRegisteredPathIsUnique(t *testing.T) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName("i18n.store.v1.LocaleRecord"))
	if err != nil {
		t.Fatalf("i18n.store.v1.LocaleRecord is not registered: %v", err)
	}
	path := d.ParentFile().Path()
	if path == "records.proto" {
		t.Fatalf("registered as the bare %q, which other libraries also use", path)
	}
	if !strings.Contains(path, "i18n") {
		t.Fatalf("registered as %q, want a path that names i18n", path)
	}
}
