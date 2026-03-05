# codexsm TODO (local)

> Note: This file is local planning only and should not be committed.

## CLI

- [x] 支持按时间、host、health、路径前缀、关键词组合过滤
- 当前：`--older-than` + `--host-contains` + `--health` + `--path-contains` + `--head-contains` 可组合（AND）
- [x] 输出稳定 ID 列表，便于后续复用
- 当前：可用 `list --format csv --column session_id --no-header --sort session_id --order asc`

- [x] 执行前显示将被删除/恢复的总数与前 20 条样本
- [x] 支持 `--preview full|sample|none`，默认 `sample`

- [ ] 增加 `codexsm plan create ... > plan.json`
- [ ] 增加 `codexsm plan apply plan.json --confirm`
- [ ] 目标：可审阅、可保存、可重复执行、可用于 CI

- [ ] 按组确认（例如 day/host）而不是逐条确认
- [ ] 保持安全性并提升批量执行效率

- [x] 每次批量动作生成 `batch_id` 并写入日志
- [x] 支持 `codexsm restore --batch-id <id>` 一键回滚软删批次

- [x] 保持 `--dry-run=true` 默认值
- [x] `--yes` 仅显式开启时跳过二次确认
- [x] 为高频参数增加短别名（如 `--limit`、`--older-than`）
- 当前：已支持 `-l/-o/-i/-p/-H`（按命令场景），以及常用 `-f/-s/-b/-n/-y`

## TUI

- [ ] 支持批量选择与批量动作（不仅单条 `d/r`）
- [ ] 批量执行前增加分组确认（按 day/host）
- [ ] TUI 内对齐预览模式（`sample|full|none`）与样本上限
- [ ] 增加 TUI 搜索/过滤（id/host/head/path）
- [ ] 增加按 `batch_id` 回滚入口（例如在 trash/source 下触发）
- [ ] 继续拆分 `cli/tui.go`（renderer/keymap/actions）降低维护复杂度

## Architecture / Decoupling

- [x] 当前分层可发布：`cli/*`、`session/*`、`audit/*`、`config/*`
- [ ] 热点治理：继续拆分 `cli/tui.go`，降低渲染/状态/按键耦合
- [ ] 拆分优先级 1：keymap + action dispatch
- [ ] 拆分优先级 2：view/render
- [ ] 拆分优先级 3：preview parser/cache
- [ ] 拆分优先级 4：status/command feedback model

## MVP Priority

- [ ] 先落地：`plan create/apply`（`batch_id` 已完成）
