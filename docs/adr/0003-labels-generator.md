# ADR 0003: The labels generator, shipped as data

| | |
|---|---|
| Date | 2026-11-13 |
| Status | Accepted |
| Roadmap items | M6.1–M6.4 |
| Deciders | epistemer chose the direction: one generation pipeline over shared metadata (neozp ADR 0001). Claude worked out i18n's part. It changes the scope of interface §2 and §11, so it's epistemer's to overturn |

## Context

Interface §2 leaves generators to the packages that own models and routes:
they emit `i18n.Key` calls like hand-written code. §11 puts generators for
keys derived from an app's schema out of scope.

cfgx did exactly that. Its amendment A13 (2026-11-12) added `App.I18n()`,
which makes `cfgx gen` write `zz_<package>_i18n.go` from a template inside
cfgx. Looking at the two repositories side by side a day later showed what
the rule costs:

- A template that calls `i18n.Key` lives in a module that doesn't require
  i18n. Nothing pins the API it targets, and a change here wouldn't show up in
  cfgx until an example failed to build.
- Every later neozp generator (models, enums, routes) would carry the same
  template. In zp, four generators did.
- Nothing sees every key before the app starts. A key declared with two
  different texts is caught by `Load`, at startup (interface §3.1, §9).

**neozp ADR 0001** (2026-11-13) replaces the rule for the whole framework. In
short:

- A generator publishes data files in named, versioned shapes, called
  *domains*. It doesn't write code for another package.
- A package that wants code from that data ships a *trivial generator*: a
  declaration of which domains it reads, queries over them, and templates. It
  contains no code.
- A *driver* runs generators in order, checks for stale output and removes
  leftovers. Its working name is `genx`.
- Labels are the first shared domain, `labels/v1`. Every producer writes it,
  and i18n reads only that.

That record is kept in the neozp project until the driver has a repository.
Where this ADR repeats part of it, that record is the reference.
[cfgx ADR 0008](https://github.com/epistemer/cfgx/blob/main/docs/adr/0008-labels-as-data.md)
is the first producer's side.

This ADR settles what i18n ships.

## Decision

i18n ships one trivial generator, `labels`.

- **Input:** every `labels/v1` file in the app.
- **Output:** for each package that has labels, `zz_<package>_i18n.go` in
  that package.
- **It's files only:** a metafile and a Go template. No Go package, no
  command, no new dependency, and no new exported identifier. The code it
  writes uses only `i18n.Key` and `i18n.TKer`.
- **Where the files sit in the module** is decided by the driver's discovery
  spike. `go mod vendor` copies package directories only, so a folder of YAML
  and templates might not survive vendoring.
- **Tests live in a nested module,** as ADR 0002 does for cfgx, so the core
  never requires the driver.
- **i18n doesn't define `labels/v1`.** The driver's documents do. i18n is its
  first reader.

### What a label is

A `labels/v1` file names a package and lists labels. Each label has a `key`
and a `text`, and optionally a `path` and a `var`:

```json
{
  "domain": "labels/v1",
  "producer": "cfgx",
  "package": { "name": "settings", "dir": "internal/settings" },
  "labels": [
    { "key": "settings.server.port", "text": "Listen port",
      "path": "server.port", "var": "L10nServerPort" },
    { "key": "settings.log.level.debug", "text": "debug",
      "path": "log.level.debug" }
  ]
}
```

### What it writes

```go
// Code generated from labels by github.com/epistemer/i18n. DO NOT EDIT.

package settings

import "github.com/epistemer/i18n"

var (
	L10nServerPort = i18n.Key("settings.server.port", "Listen port")
)

// l10n holds every label that has a path.
var l10n = map[string]i18n.TKer{
	"server.port":     L10nServerPort,
	"log.level.debug": i18n.Key("settings.log.level.debug", "debug"),
}

// L10n returns the label at path, or nil when there is none.
func L10n(path string) i18n.TKer { return l10n[path] }
```

- A label with a `var` becomes a package-level variable.
- A label with a `path` gets an entry in `l10n`, and `L10n(path)` finds it.
  The table and the function are written only when some label has a path.
- A label with neither is still declared, as `_ = i18n.Key(...)`, so it's
  registered and can be edited. That suits a key reached with `Dynamic`.
- Labels keep the order of their file. Several files for one package are
  merged, in the order of the files' paths.
- A package without labels gets no file, and the driver removes one left
  from before.
- The file declares the labels' variables, `L10n` and `l10n`, and nothing
  else at package level. `labels/v1` reserves those names, so a producer
  keeps them free.

### What it checks

Generation fails, with a message naming the key or identifier and the
producers involved, when:

1. one key has two different texts anywhere in the app;
2. two labels in a package have the same `var`, or the same `path`.

The first is what `Load` reports at startup (interface §9). Here it's
reported when code is generated.

### Prototype

On 2026-11-13 the query and template below were run with jq 1.7 and Go's
`text/template` and `go/format`:

- On the 62 labels of cfgx's `examples/zp` (48 with a variable), copied by
  a script from cfgx's generated file into a `labels/v1` file, the result
  has the same 117 lines of declarations as the `zz_settings_i18n.go` cfgx
  writes at commit `5a1759c`. Only comments differ.
- Two files for one package were merged, a second package got its own file,
  and both checks stopped the run with their message.

The driver will embed gojq, not the C implementation used here, and its
metafile format isn't frozen, so M6.1 re-runs this on the real thing.

```jq
# Input: every labels/v1 file of the app, as one array, ordered by file path.
# Output: one unit per package.
def norm: {key, text, path: (.path // ""), var: (.var // "")};
def twice(f): group_by(f) | map(select(length > 1) | .[0] | f);

# One key has one text across the app.
([.[] | .producer as $p | .labels[] | {key, text, producer: $p}]
   | group_by(.key)
   | map(select((map(.text) | unique | length) > 1)
         | "\(.[0].key) (\(map(.producer) | unique | join(", ")))")) as $clash
| if ($clash | length) > 0
  then error("labels: one key, two texts: " + ($clash | join("; ")))
  else . end
| group_by(.package.dir)[]
| { package: .[0].package,
    out: "\(.[0].package.dir)/zz_\(.[0].package.name)_i18n.go",
    labels: [.[].labels[] | norm] }
# In a package, a variable or a path names one label.
| ([.labels[] | select(.var != "")] | twice(.var)) as $vars
| ([.labels[] | select(.path != "")] | twice(.path)) as $paths
| if ($vars + $paths | length) > 0
  then error("labels: declared twice in \(.package.dir): " + ($vars + $paths | join(", ")))
  else . end
| .vars   = [.labels[] | select(.var != "")]
| .hidden = [.labels[] | select(.var == "" and .path == "")]
| .paths  = [.labels[] | select(.path != "")]
```

```gotemplate
// Code generated from labels by github.com/epistemer/i18n. DO NOT EDIT.

package {{.package.name}}

import "github.com/epistemer/i18n"
{{if or .vars .hidden}}
var (
{{- range .vars}}
	{{.var}} = i18n.Key({{printf "%q" .key}}, {{printf "%q" .text}})
{{- end}}
{{- range .hidden}}
	_ = i18n.Key({{printf "%q" .key}}, {{printf "%q" .text}})
{{- end}}
)
{{end}}
{{- if .paths}}
// l10n holds every label that has a path.
var l10n = map[string]i18n.TKer{
{{- range .paths}}
	{{printf "%q" .path}}: {{if .var}}{{.var}}{{else}}i18n.Key({{printf "%q" .key}}, {{printf "%q" .text}}){{end}},
{{- end}}
}

// L10n returns the label at path, or nil when there is none.
func L10n(path string) i18n.TKer { return l10n[path] }
{{end}}
```

### Not in this version

- **`meta`.** `Key` takes meta bytes, and zp's route labels use them.
  `labels/v1` will gain an optional `meta` field when a routes generator
  needs it; until then the generator writes none.
- **Errors.** They wait on the errors redesign (D11).
- **How producers name keys.** `settings.<path>` is cfgx's choice. The
  generator takes keys as given.

## Alternatives

- **Leave §2 and §11 as they are.** Every generator keeps writing `i18n.Key`
  calls from its own template. The costs are the ones in Context.
- **A Go command, `cmd/i18n-labels`,** that reads label files and writes the
  Go. It would work without the driver, but the core module would gain a
  command, apps would need one more `tool` line and one more step to order
  by hand, and the whole job is a query and a template of 26 lines each.
- **Let cfgx keep the template and fix only the pinning,** for example with
  a test in cfgx against i18n's latest release. That treats one of the three
  costs and leaves the template to be copied into the next generator.

## Consequences

- Interface §2 and §11 change in scope: i18n ships a generator, as data.
  Reading an app's schema stays out of scope; that's the producers' side.
  The amendment is in §13, and M6.4 rewrites the two sections.
- cfgx's `App.I18n()` goes away (cfgx amendment A14). cfgx writes the labels
  file, and this generator writes the Go.
- An app moving from cfgx's A13 output sees the same file name and the same
  declarations. Only comments change.
- The core module is unchanged: bbolt and protobuf, as interface §2 promises.
  i18n now depends on a file format, the driver's metafile, and not on any Go
  package of the driver.
- An app needs the driver to get the generated labels. i18n itself works as
  before without it.
- The roadmap gains M6 and a `v0.2.0`. M6.1 and M6.2 wait on the driver's
  frozen metafile format, M6.3 on its first release and on cfgx M11.1.
- The Claude Code skill (`~/.claude/skills/i18n/`) needs a section on
  generated labels once M6 lands.
