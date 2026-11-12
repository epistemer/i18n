# ADR 0001: Generating the protobuf records

| | |
|---|---|
| Date | 2026-11-12 |
| Status | Accepted |
| Roadmap item | M0.3 |

## Context

The store's records are protobuf messages (interface D4, implementation §2.2).
Their Go code is generated from `internal/storepb/records.proto`. Users of the
library must never need `protoc`, and CI must catch generated code that has
drifted from the `.proto` file.

## Decision

- **Generator:** `protoc` plus `protoc-gen-go`. `protoc-gen-go` is pinned with
  a `tool` directive in go.mod. It lives in the same module as the protobuf
  runtime the library already depends on, so pinning it adds nothing to the
  module graph.
- **Command:** `go generate ./...`. The directive in
  `internal/storepb/generate.go` runs `protoc` through `sh -c`, with
  `go tool -n protoc-gen-go` as the plugin. Generation needs a POSIX shell,
  but only maintainers and the Linux CI job generate.
- **Committed output:** `records.pb.go` is committed.
- **Drift check:** on Linux, CI installs `protoc` from the distribution,
  regenerates, and runs `git diff --exit-code` with
  `-I '^//[[:space:]]+protoc[[:space:]]'`. That ignores only the header line
  that records the `protoc` version, which differs between machines. Any other
  difference fails the build.

## Alternatives

- **buf** (`go tool buf generate`) needs no `protoc` binary and writes a stable
  header. But as a `tool` dependency it would add buf's large module graph to
  this library's go.mod, which every user's module graph would then see. That's
  too much for three small messages.
- **Pinning protoc's exact version in CI** would avoid the ignored header line,
  but distributions ship different versions. Downloading a pinned release adds
  a fragile step for a comment line that has no effect on the code.

## Consequences

- A maintainer needs `protoc` (any 3.x or later) and a POSIX shell to change
  `records.proto`. Nobody else does.
- Upgrading `google.golang.org/protobuf` also upgrades the generator, so the
  drift check catches a regeneration that wasn't committed.
