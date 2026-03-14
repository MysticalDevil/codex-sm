# Repository Agent Notes

## Local Repo Layout

- The active local repo path is `~/Project/codexsm`.
- This local repo should point directly at the GitHub remote for `MysticalDevil/codexsm`.
- When describing local paths in docs or notes, prefer `~`-based paths instead of `/home/...`.

## Querying Rules

- For structural code queries (e.g. pass-through wrappers, thin adapters, duplicated call patterns), prefer `ast-grep` (`sg`) first.
- Use `rg` as a supplement for plain text lookups, file discovery, and quick keyword filtering.

## Lint Rules

- Use repository lint configuration from `.golangci.yml` as the source of truth.
- Do not bypass lints with `nolint` or wrapper patterns that only silence checks (for example, `defer func() { _ = f.Close() }()` used only to satisfy linters).
- Prefer real fixes: explicitly handle returned errors and keep control flow clear and maintainable.
- Test-code exception: in `*_test.go`, unchecked `Close()` return values are allowed via lint configuration; do not add `nolint` just for that.
