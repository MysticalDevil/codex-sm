# codexsm Architecture Notes

## Decoupling Status

Current codebase is acceptable for the current release scope and does not require a mandatory large refactor before shipping.

Known hot spots:

- `cli/delete.go` and `cli/restore.go` still mix orchestration, output, and guard logic and should continue converging toward thinner wrappers.
- `session/scanner/*` and `session/migrate/*` are now split subpackages, but scan and migration hot paths still need to be tuned with benchmark feedback.
- `tui/*` rendering is more modular, but narrow-width behavior still depends on coordinated changes across layout metrics, keybar rendering, and info-row formatting.

## Architecture Design

`codexsm` follows a layered approach. The current dependency view is:

```text
External/runtime:
- Go std + encoding/json/v2
- cobra
- bubbletea / lipgloss / go-runewidth

                         +----------------------+
                         | config/*             |
                         | path/config resolve  |
                         +----------+-----------+
                                    |
                                    v
+------------------+      +----------------------+----------------------+
| main.go          | ---> | cli/*                                        |
| cli/root.go      |      | list/group/delete/restore/doctor/tui/config |
+------------------+      +----------------------+----------------------+
                                    |                 |
                                    |                 +-----------------> cobra
                                    |
                                    v
                         +----------+-----------+
                         | usecase/*            |
                         | list/group/action    |
                         | preview/doctor/tui   |
                         +----------+-----------+
                                    |                   +-------------------+
                                    |                   | audit/*           |
                                    +------------------> | batch/action logs |
                                    |                   +-------------------+
                                    v
              +---------------------+----------------------+
              | session/*                                  |
              | model/selector/risk/integrity/delete/restore |
              +---------------------+----------------------+
                                    |                  |
                     +--------------+                  +-------------------+
                     v                                                     v
          +-------------------------+                           +-------------------------+
          | session/scanner/*       |                           | session/migrate/*       |
          | scan/head/parse/io      |                           | exec/batch/index/...    |
          +-------------------------+                           +-------------------------+

+-----------------------------+      +----------------------------------+
| tui/*                       | ---> | tui/preview/*                    |
| command/app/state/actions   |      | build/index/model/service/types  |
| text + thin adapters        |      +----------------------------------+
+-----------------------------+
            |
            +-----------------------------------------> tui/layout/*
            |
            +-----------------------------------------> tui/render/*
            |
            +-----------------------------------------> tui/theme/*
            |
            +-----------------------------------------> tui/tree/*
            |
            +-----------------------------------------> bubbletea / lipgloss / go-runewidth

All layers may use:
- Go std + encoding/json/v2
- util/file.go (file move/copy helpers)
```

1. Entry and command wiring:
- `main.go`
- `cli/root.go`

2. Command layer:
- `cli/list.go`
- `cli/list_columns.go`
- `cli/pager.go`
- `cli/ansi.go`
- `cli/group.go`
- `cli/delete.go`
- `cli/restore.go`
- `cli/tui.go`
- `cli/doctor.go`
- `cli/config.go`

3. TUI package:
- `tui/command.go`
- `tui/app_preview.go`
- `tui/view.go`
- `tui/state.go`
- `tui/actions.go`
- `tui/preview/*`
- `tui/layout/*`
- `tui/render/*`
- `tui/theme/*`
- `tui/tree/*`
- `tui/text.go`
- `tui/layout_model.go`
- `tui/render_model.go`
- `tui/theme_model.go`

4. Domain and storage logic:
- `session/*` for model/filter/risk/delete/restore operations
- `session/scanner/*` for scanning and head extraction
- `session/migrate/*` for migration batch/index/rollout execution
- `usecase/*` for command-level orchestration shared by CLI/TUI
- `audit/*` for action logs
- `config/*` for path and app config resolution
- `internal/ops/*` for shared operation helpers (`preview mode`, interactive confirms)
- `util/file.go` for move/copy file helpers

5. Test support:
- `internal/testsupport/*` fixture sandbox helpers

Rules:

- CLI and TUI reuse the same core session/audit logic.
- `cli/tui.go` is an entry bridge; TUI behavior is implemented in `tui/*`.
- destructive actions default to simulation (`dry-run`) paths.
- action logging stays centralized in `audit`.
- each batch operation is tagged with a `batch_id` for traceability and rollback.

Boundary intent:

- `cli/*` should stay thin orchestration and output adaptation.
- `tui/*` should own interaction state, key handling, and rendering.
- `tui/layout/*`, `tui/tree/*`, `tui/render/*`, and `tui/theme/*` are leaf utility packages and must not depend on `tui/command` or `tui/app` orchestration files.
- `session/*`, `audit/*`, and `config/*` should remain reusable by both CLI and TUI.

Shared session-processing boundaries:

- `session/scanner/head.go` and `session/scanner/parse.go` build conversation heads used by list/group/TUI tree flows.
- `usecase/preview.go` extracts normalized preview messages; `tui/preview/*` renders/stores preview lines and index records.
- `session/risk.go` and `session/integrity.go` separate base health risk detection from optional integrity verification.
- `usecase/list.go` and `usecase/group.go` own most list/group data preparation so CLI wrappers mainly handle arguments and rendering.

Rollback flow:

1. `delete` (soft-delete) writes one `batch_id` into action logs.
2. `restore --batch-id <id>` resolves session ids from audit logs.
3. restore scans trash and restores matched sessions under normal safety guards.

## Performance Hot Paths

The current hot paths that deserve benchmark attention are:

- session tree scanning and selector filtering:
  - `session/bench_test.go`
  - `session/scanner/*.go`
- CLI table/JSON/risk rendering:
  - `cli/list_bench_test.go`
  - `cli/doctor*.go`
- TUI preview construction and preview index persistence:
  - `tui/bench_test.go`
  - `tui/preview/*.go`

Current baselines and rerun commands are tracked in [docs/BENCHMARKS.md](./BENCHMARKS.md).

## Theme And Color Conventions

Built-in themes:

- `tokyonight` (default)
- `catppuccin`
- `gruvbox`
- `onedark`
- `nord`
- `dracula`

Theme resolution order:

1. Built-in palette by `--theme` (or `tui.theme` from config)
2. Config overrides (`tui.colors`)
3. CLI overrides (`--theme-color key=value`, highest priority)

Recommended semantic keys:

- base: `bg`, `fg`, `border`, `border_focus`
- titles: `title_tree`, `title_preview`, `group`
- selection: `selected_fg`, `selected_bg`, `cursor_active`, `cursor_inactive`
- keybar: `keys_label`, `keys_key`, `keys_sep`, `keys_text`
- info/status: `info_header`, `info_value`, `status`
- preview roles: `prefix_user`, `prefix_assistant`, `prefix_other`, `prefix_default`
- tag highlighting: `tag_default`, `tag_system`, `tag_lifecycle`, `tag_danger`, `tag_success`

Rendering note:

- main panes inherit the terminal's default background instead of painting the theme `bg`
- theme `bg` is reserved for local contrast needs such as foreground-on-accent combinations

## Third-Party Packages

Core CLI:

- `github.com/spf13/cobra`: command tree and help UX

TUI:

- `github.com/charmbracelet/bubbletea`: event loop/model-update-view architecture
- `github.com/charmbracelet/lipgloss`: styles/layout/borders/colors
- `github.com/mattn/go-runewidth`: width-safe CJK and mixed text rendering

Rationale:

- mature ecosystem
- predictable cross-terminal behavior
- strong fit for keyboard-first interfaces

## Config Usage

Config model:

- `config.AppConfig`
- default file: `~/.config/codexsm/config.json`
- override path: `$CSM_CONFIG`

Main keys:

- `sessions_root`
- `trash_root`
- `log_file`
- `tui.group_by`
- `tui.source`
- `tui.theme`
- `tui.colors`

Resolution and precedence:

1. command flags
2. config file (`$CSM_CONFIG` or default path)
3. built-in defaults

Path behavior:

- `~` is expanded via `config.ResolvePath`
- missing config file is non-fatal (zero-value config)

## JSON Runtime Requirement

This project uses:

- `encoding/json/v2`
- `encoding/json/jsontext`

Build/install/test must enable:

- `GOEXPERIMENT=jsonv2`
