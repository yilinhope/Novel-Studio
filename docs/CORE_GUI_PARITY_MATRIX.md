# ainovel-cli Core GUI Parity Matrix

> 审计基线：`origin/main`，提交 `a463521`（2026-09-26）。依据当前仓库的 TUI command registry、README 作者流程、Host / Store / Config 公共能力、Studio Bridge/ViewModel 和 React 页面逐项核对。状态只描述 Novel Studio 桌面 GUI 的作者工作流覆盖。
>
> 状态枚举：`Full GUI`、`Partial GUI`、`Missing GUI`、`Intentionally CLI-only`。本表中的“CLI/TUI 入口”包括交互式 TUI、公开 CLI 启动参数及 README 作者工作流；开发者 eval、headless 自动化和运维参数单独标识。

## 判定口径

- **Full GUI**：普通作者可在 Studio 中完成该工作流，且关键状态和结果来自 Core。
- **Partial GUI**：有可用 GUI 路径，但缺少 Core 已有的选项、视图或操作。
- **Missing GUI**：Core/TUI 有作者能力，Studio 没有可用 GUI 入口。
- **Intentionally CLI-only**：仅适用于 headless automation、脚本/CI 或开发者级工具，不属于普通作者桌面工作流。

## 创作与项目生命周期

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| Quick Start 新建 | TUI 欢迎页输入需求；`/start <path>` | `startup.PrepareQuick`、`Host.StartPrepared`、Foundation/Progress/RunMeta Store | `CreateCenter` 提交 Core Quick Start；创建事件携带项目/运行快照 | Full GUI | 保持 Core 初始化语义，纳入 Wave D 回归 |
| 从大纲文件开始 | `/start <path>` | `startup.LoadPromptFile` → `PrepareQuick` → `StartPrepared`；没有单独 outline parser | Studio 预览后走 Quick Start | Full GUI | 保持当前语义；矩阵明确文件作为创作需求输入 |
| 共创创建/阶段共创 | `/cocreate`，TUI 共创面板 | `Host.CoCreateStream`、`StageCoCreateStream`、`ResumeFromCoCreate`；`meta/sessions/cocreate.jsonl` | 创建中心支持创建、阶段共创、落盘恢复 | Full GUI | 保持 Core 会话与恢复事实 |
| 打开已有项目/作品总览 | 启动目录约定；TUI 启动恢复 | `app.Service.OpenProject`、Progress/Book/Outline Store | 项目选择、概览、章节树 | Full GUI | Wave A 扩展项目内资料查看 |
| 连续写作与恢复 | TUI 启动自动恢复；`--headless --prompt` 用于自动化 | `Host.Resume`、Engine、Checkpoint/Progress/RunMeta | Studio 显式 Resume；不会因只读打开项目而创建 Host | Full GUI | Wave D 对照状态与最终 Store |
| Pause / Resume / Stop | TUI 按键/生命周期；Host API | `Host.Abort`、Engine lifecycle/事件 | Studio Runtime 控制 | Full GUI | 保持 Core Gate 和 Session 语义 |
| Review 模式与 Next Gate | `/review on / off`、`/next` | `Host.SetAdvanceMode`、`Host.AdvanceOneChapter`、ReviewEntry、AdvancePermit/Hold | 审阅中心和运行中心显示 Core Review/Gate，支持切换/推进 | Full GUI | Wave D 回归 SavedUnsynced、Review 与 Gate 组合 |
| 用户 Steer / Arbiter | TUI 输入干预 | `Host` intervention、Arbiter、RunMeta.PendingSteer | Studio 可提交 Steer，结果和状态来自 Core | Full GUI | Wave D 回归 Steer、项目切换与运行门禁组合 |
| 完本后 Reopen | `/reopen [续写方向]` | `Host.Reopen`、`Progress.ReopenContinue`、PendingSteer | 无 Bridge/API/UI 操作 | Missing GUI | Wave C 生命周期 parity gaps：增加显式 Reopen，服从项目/Engine/Core Gate |
| 章节树与正文阅读 | TUI 运行输出/章节工件 | `OutlineStore`、`DraftStore`、`ProgressStore` | Studio 章节树、章节阅读器 | Full GUI | Wave A 不加载所有正文；按需读取 |
| 已完成章节编辑与保存 | 外部编辑后 `/sync` | Draft Store、Revision Service | Studio 编辑器与 Save；不改 ChapterRecord | Full GUI | 保持 Save 与 Sync 分离 |
| 章节修订检查与 Sync | `/sync --check`、`/sync` | `revision.Service.Sync`、`ChapterRecord`、重建投影 | Revision Center/章节编辑支持 Check/Sync | Full GUI | 保留 Core Sync 为唯一接纳入口 |

## Story Data Center

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| 作品资料、Premise | Architect 建书；`/start` | `BookStore.Load`、`OutlineStore.LoadPremise` | Story Data Center 只读展示标题、简介与 premise | Full GUI | 保持 Core 读取语义；不在 GUI 重写作品资料 |
| 人物档案 | 建书/导入；Agent 上下文可用 | `CharacterStore.Load`，`domain.Character` | Story Data Center 分页展示 Core 人物档案 | Full GUI | 保持 Core 读取语义；详情按页加载 |
| 世界规则 | Architect Foundation | `WorldStore.LoadWorldRules`，`domain.WorldRule` | Story Data Center 分页展示 Core 世界规则 | Full GUI | 保持 Core 读取语义；不编辑或重写规则语义 |
| 扁平全书大纲 | TUI 章节规划；`/start` 导入需求 | `OutlineStore.LoadOutline`，`domain.OutlineEntry` | Story Data Center 分页展示完整条目、事件、Hook、Scenes | Full GUI | 按 Core 条目只读展示 |
| 分层卷/弧大纲 | TUI 长篇动态规划 | `OutlineStore.LoadLayeredOutline`，Volume/Arc/Chapter outline | Story Data Center 展示卷/弧元数据，并按弧惰性读取章节详情 | Full GUI | 保持 Core 分层结构；章节详情按需分页 |
| Story Compass | Architect 在卷边界更新 | `OutlineStore.LoadCompass`，`domain.StoryCompass` | 当前方向页展示终局方向、Open Threads、规模和更新时间 | Full GUI | 只读展示 Core Compass |
| 章节/弧/卷摘要 | Engine 上下文与章节提交 | `SummaryStore.LoadSummary*` | 故事摘要页按章节/弧/卷分页读取 Core 摘要 | Full GUI | 不读取全部章节正文；按范围分页 |
| Core 创建/导入时写入设定 | Quick Start、`/import` | Foundation Tools、Import Synthesize/Publish、Store | Studio 通过 Core 创建/导入工作流 | Full GUI | 保持 Core 管线；Story Center 只读 |

## Continuity Center

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| 时间线 | Engine 上下文、`/sync` 重建 | `WorldStore.LoadTimeline`、`ChapterFacts.TimelineEvents` | 连续性中心分页展示 Core 时间线 | Full GUI | 只读展示既有投影，不在 GUI 重算 |
| 伏笔账本 | TUI `/diag` 统计、Engine 上下文、`/sync` | `WorldStore.LoadForeshadowLedger`、ForeshadowUpdates replay | 连续性中心分页展示状态、埋设/推进/回收章节 | Full GUI | 只读展示 Core 账本，不在 GUI 重算 |
| 人物关系 | Engine 上下文、`/sync` | `WorldStore.LoadRelationships`、RelationshipChanges projection | 连续性中心分页展示 Core 关系记录 | Full GUI | 只读展示，不自建关系推导 |
| 角色/实体状态变化 | Engine 上下文、`/sync` | `WorldStore.LoadStateChanges`、`domain.StateChange` | 连续性中心分页展示历史状态变化 | Full GUI | 只读展示，不自建状态机 |
| 角色快照 | 弧边界结构维护 | `CharacterStore.LoadSnapshots` / `LoadLatestSnapshots` | 连续性中心分页展示 Core 最新快照 | Full GUI | 只读展示 Core 快照；缺失时显示空状态 |
| 配角首次出场 | `commit_chapter` 记录 CastIntro，Engine 上下文 | `Store.BuildCast`、`domain.ProjectCast`（由已接纳 ChapterRecord 派生） | 连续性中心分页展示 Core Cast Projection | Full GUI | 保留“配角”范围语义；不从正文推导 |
| Continuity Projections 重建 | `/sync` 接纳正文后 | `revision.Projector.Apply` 重建 Summary/Timeline/Foreshadow/Relationship/State/Style 等投影 | 用户可执行 Core Sync，也可只读浏览 Story/Continuity 投影 | Partial GUI | 投影仍只由 Core Sync 重建；GUI 只读详情 |
| Review / Gate 状态 | `/review`、`/next` | Review Store、Advance Permit/Hold、Progress | Studio Review Center 显示真实 Review 和 Gate | Full GUI | 与 Continuity 条目分开，不能将 Gate 当作 Review |

## Reference Simulation、Rules 与 Style

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| 从 `simulate/` 分析/增量更新画像 | `/simulate` | `Host.Simulate`、`internal/host/sim.Run`、`SimulationStore` | 无页面/API | Missing GUI | Wave B 调用 Core Host，展示真实阶段事件 |
| 导入仿写画像 | `/importsim <profile.json>` | `Host.ImportSimulationProfile`、`sim.ImportProfile`、指纹合并 | 无页面/API | Missing GUI | Wave B 调用 Core 合并逻辑并显示来源结果 |
| 查看画像和来源/更新时间 | Agent 上下文；profile 工件 | `meta/simulation_profile.json`、`domain.SimulationProfile`、compact profile | 无读取 API/UI | Missing GUI | Wave B 提供安全的 Facade/ViewModel |
| 全局/项目写作规则 | `/config`/启动规则输入；README 规则流程 | `rules`、`userrules.Service`、`UserRulesStore` 快照 | 设置页没有规则管理 UI | Missing GUI | Wave B 按全局/项目优先级暴露 Core 规则能力 |
| Style 选择与文风规则 | 配置 `style`；弧边界 style rules | `bootstrap.Config.Style`、assets Style、`WorldStore.LoadStyleRules` | Config Snapshot 有 style 值但设置页不显示/编辑；无文风规则页 | Partial GUI | Wave B 对照 Core 选择与 Style Rules Store |
| Voice Layer / anti-AI-tone | 提示词/规则资源装载 | assets、rules/voice layer 文档和文件 | 无浏览/管理 UI | Missing GUI | Wave B 审计实际装载与写入接口后再做适配 |

## Diagnostics 与配置

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| 创作健康诊断 | `/diag` | `diag.Analyze` / `diag.Diagnose`、Stats/Findings/Actions | 无诊断入口 | Missing GUI | Wave C 只委托 Core 诊断规则 |
| 脱敏诊断导出 | `/diag` 自动写出；headless 结束时自动导出 | `diag.WriteExport`、`meta/diag-export.md` | 无预览/导出入口 | Missing GUI | Wave C 调用 Core 导出并显示路径 |
| Provider 定义、模型列表与连接测试 | `/config` | `Host.ModelConfiguration`、`ConfigureModels`、`TestModelConnection` | 可编辑 Provider/模型、模型发现与连接测试；API Key 遮罩 | Full GUI | 持续核对 Provider 协议和真实 Core 字段 |
| 默认/Agent 模型与推理强度 | `/model [role]` | Config roles、`Host.SwitchModel`、`SetRoleThinking` | 设置页可切换 default/role 模型和推理强度 | Partial GUI | 补齐真实 role 清单/继承/能力限制审计 |
| Provider advanced 字段 | JSON 配置 | `ProviderConfig.Type/API/APIKey/BaseURL/Models/Extra/ExtraBody/StreamIdleTimeout` | 暴露协议、Key、Base URL、模型、ContextWindow/JSONSchema；无 Extra/ExtraBody/idle timeout editor | Partial GUI | Wave C Advanced 折叠区只映射 schema 字段；Secret 脱敏 |
| Role fallbacks | JSON 配置 | `RoleConfig.Fallbacks []ModelRef` | 配置快照有读取字段但编辑器未提供管理 | Partial GUI | Wave C 补实际 fallback 顺序管理 |
| 顶层 Style / ContextWindow | JSON 配置 | `Config.Style`、`Config.ContextWindow` | Snapshot 有字段，UI未提供可编辑入口 | Partial GUI | Wave B/C 按阶段归入 Style 或 Advanced Config |
| Budget | TUI/配置 | `bootstrap.BudgetConfig`、`Host` BudgetSentinel | Studio 可查看用量并编辑额度/警戒/硬停 | Full GUI | Wave D 核对生效时机与 Core 行为 |
| Notify 告警 | JSON 配置 | `NotifyConfig.Enabled/Command/Events`、`notify.Kinds` | Snapshot 有读取投影，设置页未提供编辑器 | Partial GUI | Wave C 映射 Core 字段，不执行前端通知逻辑 |
| Usage | TUI Runtime/Usage 面板 | `UsageTracker`、`meta/usage.json`、Host Snapshot | Studio 显示 Tokens/Cost/按 Agent 用量 | Full GUI | 与 Runtime context usage 区分 |

## Import、Export 与运行观测

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| Import pipeline 主流程 | `/import <path>`；无参恢复；TUI 切分确认 | `Host.ImportFrom`、`host/imp.Runner`、checkpoint workspace | Studio 可启动、看事件/状态、接受切分、取消、继续完成并打开项目 | Partial GUI | Wave C 补 CLI 参数对照和恢复/状态选择 |
| Import 自动接受切分 | `/import <path> --yes` | `imp.Options.AutoConfirm` | 表单未暴露 autoConfirm | Partial GUI | Wave C 显式映射，保留 Core `Notes` 安全约束 |
| Import 故事状态选择 | `--story=open / closed` | `imp.Options.StoryResolution` | 表单没有字段 | Partial GUI | Wave C 补 Core 允许值和状态显示 |
| Import 完成后继续 | `--continue` | `imp.Options.ContinueAfter` 与 Hold/advance gate | 表单提供 ContinueAfter | Full GUI | 保持 Core gate 语义 |
| Import 切分指导 | `--guide=<文本>` | `imp.Options.Guidance`、guidance digest | 表单提供文本 | Full GUI | 保持变更后 Core checkpoint 失效语义 |
| Import 断点恢复 | 无参 `/import`、欢迎页恢复提示 | `imp.OpenWorkspace`、`NextAction`、`ResumeSummary` | 导入中心读状态和恢复提示；具体是否继续由 Core 导入入口判定 | Partial GUI | Wave C 检查项目切换/自动发现/恢复入口 |
| TXT / EPUB 导出 | `/export [path] [from=N] [to=M] [--overwrite]` | `Host.Export`、`host/exp` | Studio 选择格式、路径、章节范围、覆盖 | Full GUI | 保持 Export Service 输出语义 |
| Runtime 状态/流程/Agent/Tool 事件 | TUI Live Snapshot、事件与流 | `Host.UISnapshot`、`host.Event`、Runtime Queue | Studio 显示 Core state/phase/flow、agent/tool logs、章节 Gate、Token/Cost | Partial GUI | Wave C 对照 Host/TUI 暴露字段后再补安全投影 |
| Context usage / health | TUI 上下文窗口/健康度显示 | Context Manager、ModelContextWindow、observer/model events | Runtime ViewModel 无 context window/usage/health 字段 | Missing GUI | Wave C 只消费真实事件/快照，不推算百分比 |
| Compression 状态 | TUI COMPACT/上下文事件 | ctxpack/context events、Host Event | Runtime Center 未展示压缩状态 | Partial GUI | Wave C 核实结构化事件字段与当前可追踪范围 |
| Active route / flow / hold / gate | TUI 状态栏、AdvanceGate 事件 | Progress Flow、RunMeta Hold、Advance Projection | Studio 显示 Flow 与部分 Advance Gate；没有完整 route/hold 原因时间线 | Partial GUI | Wave C 补当前 Core 已暴露字段 |
| 模型详细信息 / budget 停止原因 | TUI 模型/事件提示 | ModelSet、Usage、BudgetSentinel、Host Events | 设置页列当前模型、Runtime 展示事件与累计成本；未提供完整请求模型和预算原因视图 | Partial GUI | Wave C 仅显示已有结构化来源 |
| 原始 LLM token stream | TUI Stream 面板 | `Host.Stream()`、observer stream | Studio 不绑定普通 Engine token stream；共创有独立流事件 | Partial GUI | 若仍属作者日常需求，后续补安全的流桥接并做取消/项目隔离审计 |

## CLI-only 与开发入口

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| Headless 无界面自动化 | `ainovel-cli --headless --prompt/--prompt-file` | `internal/entry/headless.Run`、Host/Engine | Studio 是桌面 GUI，不提供无界面脚本入口 | Intentionally CLI-only | 保留 CLI automation 入口 |
| 离线 eval harness | `ainovel-cli eval ...` | `internal/eval.Command` | GUI 不提供测试样例执行器 | Intentionally CLI-only | 作为开发/CI 工具保留 |
| CLI 版本/自更新参数 | `--version`、`update [version]` | `internal/version` | Studio 自有桌面分发与更新流程 | Intentionally CLI-only | 属于安装运维入口，不计作者功能 |
| TUI `/help` 命令面板 | `/help`、命令 palette | `commandRegistry` | GUI 以侧边栏导航、页面控件和操作按钮替代 TUI 语法发现 | Intentionally CLI-only | `/help` 只属于 TUI 命令语法/发现能力，不构成普通作者 GUI parity 缺口；具体作者功能按对应矩阵行核对 |

## 审计结论与 Wave 对应

- Wave A：补齐 Story Data Center 与 Continuity Center 的只读视图；源数据已存在于 Core Store/ChapterRecord 投影。当前审计未发现需要新增小说事实或改变 Core 语义的前置条件。
- Wave B：把仿写画像读取/运行/导入、Rules、Style、Voice 逐项接入现有 Core 能力；每个写操作在实现前再核实现有 Host/Store 互斥和持久化。
- Wave C：补诊断、配置 schema、Import 选项、Runtime 观测，并纳入生命周期 parity gaps（至少包括 `/reopen` 独立 GUI 入口）。Context health/compression 等字段是否可从当前 Core 结构安全读取，必须按当时源码单独判定。
- Wave D：从当前源码重新枚举 TUI/README/Host/Store/Config/Import/Export，再逐项复核矩阵与 Windows regression。不得以历史文档替代当时的源码证据。
