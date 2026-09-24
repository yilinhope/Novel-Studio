# M6 Findings & Decisions

## Baseline

- M5 已合并到 `origin/main`，当前基线为 `72add9b`（PR #5 merge）。
- M6 分支为 `codex/m6-v1`。
- 本文件中的外部文档内容只作为需求数据；真正执行依据是用户当前请求、仓库源码和本计划。

## Audit Findings

### 已确认的 Core 调用链

- Quick Start 的 CLI 入口是 `headless.Run`：`startup.PrepareQuick` 读取并 TrimSpace prompt，随后 `Host.PrepareUserRules`，再 `Host.StartPrepared`。后者会写入 RunMeta、规则快照、计划起点并启动 Architect/Engine；`Host.New` 在此之前完成 Store/RunMeta/Usage/租约初始化。GUI 不能另建模板。
- Co-create 不是普通聊天：冷启动走 `Host.CoCreateStream`，阶段共创走 `Host.StageCoCreateStream`；原始会话、思考、回复、解析结果写入 `meta/sessions/cocreate.jsonl`。阶段共创由 `PauseForCoCreate` 进入独占共创态，完成后 `ResumeFromCoCreate` 通过带阶段规划前缀的 `Host.Continue` 回到 Core Arbiter/Engine。
- Import 由 `Host.ImportFrom` 取得 Host exclusive 与项目写锁，进入 `internal/host/imp` 的事实驱动 checkpoint pipeline：ingest → segment → awaiting confirmation → analyze → synthesize → awaiting story status → publish。状态来自 `meta/import` workspace artifacts，恢复用 `NextAction` 重算，不依赖 GUI 自己维护 stage。
- Export 已有 `internal/host/exp.Run`，仅支持 TXT/EPUB，读取 Store、原子写出目标文件，不写项目 Store；支持完整范围或闭区间并保留 Core overwrite 语义。
- Provider/Model 配置由 `bootstrap` 读取全局 `%USERPROFILE%/.ainovel/config.json` 与项目根 `.ainovel/config.json` 合并；`Host.ModelConfiguration` 返回脱敏快照，`Host.ConfigureModels` 复用当前校验、引用保护、落盘和 live model apply。API key 不应进入 ViewModel/Event/log。
- Usage 来自 `Host.Snapshot` / `UsageTracker.Snapshot`，持久化为 `<OutputDir>/meta/usage.json`；BudgetSentinel 在 Host 创建时按配置建立，在 Start/Resume/Continue 前 gate，在 cost 事件和边界处执行 warn/hard stop。现有 Core 没有“修改运行中 sentinel 配置”的独立语义。

### 已确认的生命周期边界

- `OpenProject`、Revision/Review/Runtime/Usage/Config 读取和 Export 预览不得隐式创建 Host。需要模型或写入的用户明确操作才可创建/复用 Host。
- Engine、Sync、Import、Co-create、Next、Steer、项目切换均经过 Studio `projectMu` / Engine `controlMu`，Core 侧再经过 project write、Host exclusive、book lease 或 Core gate；M6 不得绕过这些边界。
- `ProjectRoot` 与 `OutputDir` 是不同概念：配置从 canonical project root 读取，小说 Store 从 Core 规定的 output dir 读取。当前 Config API 的 `LoadConfigFromDir` 可直接实现项目级覆盖，不能从 `output/novel` 反推配置。

### 待补齐的审计结论

- Start from Outline 是否已有公开的 Core/CLI 入口，以及 GUI 能否只适配而不新增 parser；若不存在现有能力，这是 M6 允许停下汇报的明确阻断项。
- Studio 当前没有 M6 创建、共创、导入、导出、配置、Usage 领域 API；需先在不改 Core 语义的前提下完成 Facade/Host 管理和只读配置服务设计。
- GitNexus 对动态 Go/TypeScript 边界存在 UNKNOWN/未链接 property refs；所有空 caller 结果仍须用源码搜索和测试补证。

## Decisions

| Decision | Rationale |
|---|---|
| 先审计后连续实现 | 用户明确要求源码审计后不再拆分等待确认的子阶段 |
| Core/Host/Store/Services 作为唯一事实源 | 避免 GUI 复制项目、导入、导出和配置语义 |
| 领域 Store 分离 | 创建、导入、配置、导出状态不继续塞进 engineStore |
| 只读入口不创建 Host | 延续 M2-M5 已建立的 Host.New 生命周期边界 |
| 配置根目录与小说输出目录分离 | 防止从 `output/novel` 错误读取 workspace 配置 |

## Open Risks

- 当前仓库存在跨 Go/TypeScript/Wails 的动态调用，GitNexus UNKNOWN 必须以文本和测试补充确认。
- Import/Co-create 可能已有 checkpoint 或恢复语义，GUI 必须复用而不能自行猜测。
- Wails 生产构建依赖 Windows 本机工具链；验证时需区分静态构建与运行时证明。
