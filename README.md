# i18n

Translations for Go web apps where each deployment owns its text. Keys are
declared in code with their default text. People edit any text at runtime,
the source language included, and an edit always beats the code. Edits are
stored in bbolt and read on every lookup, so they show up at once. That makes
it a white-labeling mechanism as much as a translation library.

> **Status: design.** No code yet. The API is drafted in
> [docs/interface.md](docs/interface.md) and isn't locked. The ideas parked for
> after v1 are in [docs/roadmap.md](docs/roadmap.md), and progress is logged in
> [docs/status.md](docs/status.md).

Part of the neozp framework, alongside
[cfgx](https://github.com/epistemer/cfgx). License: [MIT](LICENSE).
