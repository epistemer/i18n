// Package storepb holds the protobuf records the i18n library stores.
package storepb

// Regenerate with protoc and the protoc-gen-go version pinned in go.mod.
// See docs/adr/0001-protobuf-generation.md.
//go:generate sh -c "protoc --plugin=protoc-gen-go=$(go tool -n protoc-gen-go) --go_out=. --go_opt=paths=source_relative records.proto"
