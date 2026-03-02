# Codex Session Manager 目标文档（v0.1）

## 1. 项目目标
在当前目录实现一个本地可运行的 `codex session manager`，用于：
1. 查看 Codex 会话（session）列表与关键元信息。
2. 安全删除指定会话（默认防误删，支持预览与二次确认）。
3. 支持两种交互模式：`CLI`（默认）和 `TUI`（可选）。
4. 支持模拟运行（simulation/dry-run），用于测试流程且不实际删除数据。

目标是先完成一个可稳定使用的 MVP（最小可用版本），优先保障“可见性”和“删除安全性”。

## 2. 范围定义

### 2.1 In Scope（本期范围）
1. 扫描本地 Codex session 存储目录。
2. 列出会话：`id`、创建时间、最近更新时间、大小、状态（可读/损坏）。
3. 按条件筛选：
   - 按时间范围（如 N 天前）
   - 按会话 ID 前缀
   - 按状态
4. 删除会话（安全模式）：
   - 默认先 dry-run（预览将删除的对象）
   - 必须显式确认后执行
   - 删除前可选备份（move 到回收目录而非直接永久删除）
5. 操作日志记录：谁在何时删除了哪些会话（本地日志）。
6. 提供 `TUI` 可选入口，复用同一套核心逻辑（扫描/筛选/删除/日志）。

### 2.2 Out of Scope（本期不做）
1. 远程服务端 session 管理。
2. 多用户权限系统（仅本机当前用户）。
3. 自动定时清理（后续版本考虑）。

## 3. 用户故事
1. 作为开发者，我想看到所有会话和大小，便于定位空间占用。
2. 作为开发者，我想按时间筛出旧会话，再决定是否删除。
3. 作为开发者，我删除前希望看到清单并二次确认，避免误删。
4. 作为开发者，我希望删除后可追踪日志，必要时可恢复（若启用回收目录）。

## 4. 功能需求

### 4.0 模式要求（CLI + TUI）
1. `CLI` 为默认必选模式，完整支持查看与删除。
2. `TUI` 为可选模式，至少支持：
   - 浏览会话列表
   - 查看关键字段（ID/时间/大小/健康状态）
   - 触发删除并走同等安全确认流程
3. 两种模式必须共享同一核心服务层，避免逻辑分叉导致行为不一致。
4. 若 `TUI` 依赖缺失或启动失败，需回退提示用户使用 `CLI`。
5. 两种模式都必须支持模拟运行，且模拟结果格式与真实执行结果尽可能一致（仅副作用不同）。

### 4.1 查看会话
1. 命令：`session-manager list [options]`
2. 输出字段：
   - `session_id`
   - `created_at`
   - `updated_at`
   - `size_bytes`
   - `path`
   - `health`（`ok` / `corrupted` / `missing-meta`）
3. 支持输出格式：`table`（默认）、`json`。

### 4.2 删除会话（安全删除）
1. 命令：`session-manager delete [selector] [options]`
2. 安全机制：
   - 默认 `--dry-run=true`
   - 非 dry-run 时必须 `--confirm`
   - 批量删除必须带 `--yes`（或逐条确认）
3. 删除策略：
   - 默认软删除：移动到 `trash/`（同根目录）并记录映射关系
   - 可选硬删除：`--hard`（需额外确认）
4. 防呆限制：
   - 未指定 selector 时禁止删除
   - 单次删除数量超阈值（如 >50）需额外确认短语（例如输入 `DELETE 50`）

### 4.4 模拟运行（Simulation）
1. 模拟运行不产生任何数据副作用，不移动、不删除、不修改原 session 数据。
2. 模拟运行输出应包含：
   - 命中的 session 数量
   - 将执行的动作类型（soft-delete/hard-delete）
   - 预计影响大小（bytes）
   - 逐项结果预判（可执行/不可执行及原因）
3. 日志中需明确标记 `simulation=true`，便于审计区分真实删除。
4. 测试场景默认使用模拟运行，避免误删真实数据。

### 4.3 日志与可审计性
1. 记录到 `logs/actions.log`（JSON Lines）。
2. 每条删除记录包含：
   - 时间戳
   - 操作类型（dry-run/soft-delete/hard-delete）
   - session_id 列表
   - 匹配条件
   - 执行结果（成功/失败及原因）

## 5. 非功能需求
1. 安全性：默认不执行破坏性动作。
2. 可靠性：部分失败不影响其他会话处理，并输出失败明细。
3. 可维护性：核心逻辑拆分为 `scanner / selector / deleter / logger` 模块。
4. 性能：1 万会话以内列表响应可接受（目标 <3s，按本机 SSD 基准）。
5. 可移植性：优先 Linux/macOS 路径兼容。

## 6. CLI 设计草案
1. `session-manager list --format table --older-than 30d`
2. `session-manager list --id-prefix abc --format json`
3. `session-manager delete --id abc123 --dry-run`
4. `session-manager delete --older-than 90d --confirm --yes`
5. `session-manager delete --older-than 180d --hard --confirm --yes`
6. `session-manager delete --older-than 30d --simulate`

## 6.1 TUI 设计草案（可选）
1. 启动：`session-manager tui`
2. 主界面：
   - 顶部筛选栏（时间、ID 前缀、状态）
   - 中部列表（可上下选择）
   - 右侧/底部详情（路径、大小、时间）
3. 删除流程：
   - 先展示将删除对象预览
   - 支持“模拟执行”开关（默认开启）
   - 再要求确认（批量场景含额外确认）
   - 执行后展示结果统计与失败项
4. 快捷键（草案）：`/` 搜索，`d` 删除，`r` 刷新，`q` 退出。

## 7. 数据与目录约定（草案）
1. `SESSIONS_ROOT`：从环境变量读取；若未设置，尝试常见默认路径。
2. 项目内状态目录：
   - `./trash/`（软删除回收站）
   - `./logs/actions.log`（审计日志）
   - `./state/`（可选：索引缓存）

> 注：实际 session 存储结构需在实现前做一次探测确认。

## 8. 风险与应对
1. **风险**：session 元数据格式不稳定。  
   **应对**：扫描器容错解析，标记 `missing-meta/corrupted`，不因单条异常中断。
2. **风险**：用户误操作批量删除。  
   **应对**：dry-run 默认开启 + 多重确认 + 删除阈值保护。
3. **风险**：软删除目录占满磁盘。  
   **应对**：提供 `trash prune`（后续需求）并提示剩余空间。

## 9. 里程碑
1. M1（文档与探测）
   - 明确 session 目录结构与字段来源
   - 固化 CLI/TUI 参数与交互边界
2. M2（MVP）
   - `list` 可用
   - `delete` dry-run + soft-delete + confirm 可用
   - 审计日志可用
   - `tui` 基础浏览可用（可选安装或可选命令）
3. M3（增强）
   - TUI 删除流程与 CLI 全量对齐
   - hard-delete 二次确认
   - 性能优化与错误报告完善

## 10. 验收标准（MVP）
1. 能列出本地 session 且字段齐全。
2. 默认删除命令不会真正删除数据（dry-run）。
3. 无 `--confirm` 时拒绝执行真实删除。
4. 软删除后目标会话移动到 `trash/`，且日志可追踪。
5. 对不存在/损坏会话给出明确错误并继续处理其他项。
6. `CLI` 可独立完成所有核心能力；`TUI` 作为可选入口可正常浏览会话，并复用同样删除安全规则。
7. 模拟运行下不产生任何删除副作用，且输出结果可用于测试验证删除路径。

## 11. 下一步实现建议
1. 先写 `session probe` 小脚本，确认真实 session 存储结构。
2. 基于探测结果落定 `Session` 数据模型。
3. 先完成 `list`，再接 `delete`（dry-run -> soft-delete -> hard-delete）。
