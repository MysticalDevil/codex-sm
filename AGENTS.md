# Repository Agent Notes

## Local Repo Layout

- The active local repo path is `~/Project/codexsm`.
- This local repo should point directly at the GitHub remote for `MysticalDevil/codexsm`.
- When describing local paths in docs or notes, prefer `~`-based paths instead of `/home/...`.

## Querying Rules

- For structural code queries (e.g. pass-through wrappers, thin adapters, duplicated call patterns), prefer `ast-grep` (`sg`) first.
- Use `rg` as a supplement for plain text lookups, file discovery, and quick keyword filtering.
