# Review 0010: a unique protobuf file path (v0.3.1)

| | |
|---|---|
| Date | 2026-11-15 |
| Reviewer | Gemini 3.8 Flash (High), through the agy CLI |
| Scope | the diff of branch `v0.3.1-proto-path` against `main`, without the status log, inline with spec excerpts; no tools |
| Result | no findings |

## Reviewer output

No findings.

### Verification against spec requirements:
1. **Proto package and message names:** Package remains `i18n.store.v1`; messages (`LocaleRecord`, `CatalogEntry`, `Base`) and field definitions are identical (the `.proto` file is a 100% rename). Only the registered file path changed from `records.proto` to `i18n_records.proto`.
2. **Wire format (bbolt compatibility):** Protobuf wire format encodes field tags and wire types, omitting proto file names and message names. Serialized bytes remain fully compatible and readable.
3. **CI drift check:** `internal/storepb/generate.go` targets `i18n_records.proto` using the pinned `protoc-gen-go v1.36.12`. `records.pb.go` is renamed to `i18n_records.pb.go` in git, so running `go generate ./...` regenerates `i18n_records.pb.go` with zero diff and leaves no untracked files.
4. **Path test:** `internal/storepb/path_test.go` checks descriptor lookup via `protoregistry.GlobalFiles` for `i18n.store.v1.LocaleRecord` and explicitly fails with `t.Fatalf` if `d.ParentFile().Path() == "records.proto"`.
5. **No exported API change:** `internal/storepb` is an internal package inaccessible outside the module. Public message types and methods are unchanged.

## Handled

Nothing to handle.
