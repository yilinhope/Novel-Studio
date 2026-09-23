# M3 Engine Bridge：源码审计与 M3-A 实施方案

审计日期：2026-09-23。基线：`codex/m2-novel-studio-readonly`；工作分支：`codex/m3-engine-bridge-audit`。本文件只记录当前源码事实与待实施方案，不表示 M3-A 已接入或通过运行验收。对应交接文档第 2、19 节要求审计确认后再修改 Core。

## 1. 真实调用链

| 问题 | 源码级结论 |
| --- | --- |
| Engine / Host 入口 | `cmd/ainovel-cli/main.go` 加载配置和 `assets.Bundle` 后分派 TUI/headless；`internal/entry/tui/app.go:20` 调 `host.New`。`internal/host/host.go:102` 构造 Host、Store、模型、Worker、Observer、Gate 和私有 Engine；`Host.startEngine` (`host.go:474`) 调 `engine.start` (`engine.go:89`)，后者启动 `engine.run` (`engine.go:150`)。Engine 每轮依据 Store 事实运行 `flow.LoadState`、`flow.Route`，由 Engine 而非 UI 决定派单。GUI 可直接调用 Go Host，无需 CLI 子进程。 |
| TUI 如何继续 | `internal/entry/tui/events.go:153` 的 `bootstrapRuntime` 在启动时无条件尝试 `Host.Resume()`；`resumeBook` (`events.go:180`) 在会话内再做一次。`Host.Resume` (`host.go:551`) 先算 `resumeLabel`、清洁与预算门禁，再处理 PendingSteer 或 `startEngine(nil)`。GUI 打开项目必须保持只读，不能复制 TUI 的自动续跑。 |
| Pause / Resume / Stop | `Host.Abort` (`host.go:901`) 把 lifecycle 设为 paused 并取消 Engine context，返回值仅说明取消已请求。`engine.abort` (`engine.go:117`) 不等待 Worker/LLM 完成；`Host.Done` (`host.go:1080`) 在 `runEnded` (`host.go:979`) 收尾后通知。`Host.Close` (`host.go:935`) 取消、等待 Engine、保存 Usage、关闭通道并释放项目租约。没有独立的 force-stop API；M3 不应伪造强杀。`Host.Continue(text)` (`host.go:766`) 是有文本的 Arbiter 干预，不是无文本的“继续创作”；`Host.AdvanceOneChapter` (`host.go:824`) 是 review 放行。 |
| Agent / Phase / Flow / Step | `Host.Snapshot` (`host.go:1146`) 从 Progress/RunMeta/Observer/Usage 汇总 runtime、phase、flow、chapter、agents 和费用；`observer.agentSnapshots` (`observer.go:243`) 给 agent 状态和当前 tool。Step 来自 Worker `ProgressPayload` 经 `observer.workerProgress` (`observer.go:172`) 转为 Host `Event`，并非独立持久化的固定状态。`Snapshot` 会读多份 Store，不能每秒全量轮询。 |
| Writer 流程 | `internal/agents/build.go:137-150` 注册 `novel_context`、`read_chapter`、`plan_chapter`、`draft_chapter`、`edit_chapter`、`check_consistency`、`commit_chapter`；`build.go:261` 以 `commit_chapter` 结束 Writer。前六项不是每次必经的固定状态，`edit_chapter` 也真实存在。`observer_tools.go:24,47,94` 发出工具开始/完成/失败事件；当前 `Event` 只有显示用 Summary，没有稳定的 Tool 字段。映射应使用结构化 Tool 信息，不解析本地化文案。 |
| Usage | `UsageTracker.Record` (`host/usage.go:135`) 累加上游消息 Usage，`Totals()` 给项目累计费用与 token；`Host.Snapshot` 已暴露 `TotalInputTokens`、`TotalOutputTokens`、`TotalCostUSD`。Run Cost 可在本次启动时记录累计费用基线后计算差值；Elapsed 以本次启动的单调时钟计算。上游不返回 Usage 时显示不可用/告警，不把零当准确费用。 |
| 日志 | `Host.emitEvent` (`host.go:1085`) 先写 `slog` 再送有界 `Events()` 通道；Observer 的完成事件经 `persistEvent` (`observer.go:212`) 写 Runtime Queue，`Host.ReplayQueue` (`host.go:1534`) 可回放。TUI 通过 `WithFileLog("tui.log")` 写文件；GUI 可用自己的日志文件名。普通 `slog` 消息不等于结构化 Runtime Event，不能承诺所有文件日志都会实时出现在 UI。 |
| 章节提交 | `CommitChapterTool` (`tools/commit_chapter.go:400-435`) 先标记 Progress、追加 checkpoint，再清理 InProgress/PendingCommit；只有工具成功完成且 Store 事实确认后，才能触发 Overview/Tree/当前章节/Usage 定向刷新。单靠 Progress 已标记会误报中途失败；`SignalStore.SaveLastCommit` 当前没有写入调用，不能当主信号。 |
| Review gate | `ChapterAdvanceGate.Allow` (`host/advance_gate.go:164`) 使用 RunMeta 的 AdvanceMode/Permit、Progress 和 PendingCommit，并通过 `flow.StartsForwardChapter` (`flow/advance.go:11`) 判断是否真正开始新章；不足许可时调用暂停。`Resume()` 不能绕过 gate，但 GUI 应在可确认的等待验收事实下禁用普通 Continue，不能只看 `AdvanceMode=review` 就断定所有运行都要停，维护/返工仍可能继续。 |
| Event / callback | `Host.Events`、`Host.Stream`、`Host.Done` 都是已有单消费者通道；`observer.workerProgress` 是现有进度中继。Studio 应独占其 Host 的消费权，事件出口做 Wails 转发和有界日志缓存；不要让两个读者竞争同一个通道。 |
| 需抽的 TUI handler | 不应抽 Bubble Tea 的 `tea.Cmd` 本身。抽成 Studio 薄服务的只有：配置/Bundle/Host 初始化、显式 `Resume`、`Abort` 与 Done 等待、`Close` 收尾、Event/Stream/Done 转发、Store 确认后的章节失效通知。TUI 的 `bootstrapRuntime` 自动续跑行为必须留在 TUI，不进入共享服务。 |
| M2 边界 | `internal/studio/app/project.go:22` 的 `OpenProject` 只持有 `store.Store` 并只读快照；`internal/studio/bridge/app.go:21` 只转发查询，未持有 Host。边界仍干净。`Host.New` 会取得独占 lease、`Store.Init`、项目迁移、RunMeta 初始化、Usage 回填/保存及后台任务，因此不能在 M2 的只读 `OpenProject` 里偷偷构造 Host。 |

## 2. GitNexus 影响与盲区

已刷新索引到 9,080 nodes、37,902 edges；分析器提示部分执行流程截断。`Host.startEngine` 上游影响风险为 **CRITICAL**（5 个直接调用者，11 个受影响流程，2 个模块）；`Host.Resume` 为 **HIGH**（2 个直接调用者，3 个流程，TUI 模块）。因此 M3-A 优先新增 Studio 适配层，不改这些 Core 函数的语义。GitNexus MCP 此次出现 `Transport closed`，已由仓库 CLI 查询核实上述风险。图的零调用或缺失流程不等于无影响，尤其 Wails/JS 跨语言边界要以文本搜索和集成测试补证。

## 3. M3-A 代码级方案（审计确认后实施）

1. 在 `internal/studio/app` 新建 EngineSession/EngineService，由它独占一个 `*host.Host`、当前项目路径、取消句柄和 generation ID。对外使用 `ResumeWriting`（调用 Host.Resume）、`PauseWriting`（Abort 后等 Done）、`StopWriting`（Close 等待并释放租约）、`RuntimeState` 等明确 API。项目切换或 Wails 退出先收尾旧 Host，再切换；旧 generation 事件不得更新新项目。每个控制调用检查当前 project/generation，拒绝重复启动和并发切换。
2. 配置装配不能直接在用户任意 cwd 上调用 `bootstrap.LoadConfig()`：它的项目级覆盖路径相对 cwd，`EffectiveConfigPath()` 也相对 cwd。先明确选中的 `output/novel` 与项目根目录关系，在显式启动时定向加载该项目配置、设置 `cfg.OutputDir` 为已验证的小说目录，调用 `assets.Load` 与 `host.New`。不在普通打开时初始化 Host。配置缺失/无可用模型时报告错误，不破坏只读浏览。
3. 建 `internal/studio/viewmodel/runtime.go` 和只读派生函数，以 Host Snapshot、Progress、RunMeta、真实事件构造 `Idle/Running/Paused/WaitingReview/Error`；`WaitingSync` 仅保留类型。过渡中的 Pause 不可仅凭 `Host.Abort()` 返回显示 Paused；等 Done/Engine 确认。Stop 与 Pause 都是安全取消，但 Stop 另须 `Host.Close`、释放 lease，UI 清楚区分“可恢复会话”和“已关闭会话”。
4. 建单一 Event pump，把 Host Event、Stream、Done 映射到 Wails `studio:*` 事件。事件统一带 project ID、generation、递增 sequence、timestamp；前端丢弃过期代。完成事件沿用 Host 的同 ID 更新，不增一条假完成日志。日志仅保留有界最近 N 条，历史由 Runtime Queue 按序号加载。Usage 在模型/工具/运行状态等有意义的事件后读取轻量累计值，避免秒级全 Store 快照。
5. 当前 `host.Event` 缺结构化 Tool 名。优先在不改变 Engine 路由的前提下增加可选 `Tool` 字段，由 `observer_tools.go` 填充，旧 TUI 不消费时兼容；再用工具开始/结束事件驱动可选 Pipeline 节点。设计中六项仅作为已知展示顺序，`edit_chapter` 可按实际出现展示，不把未出现步骤标成“已完成”。
6. `commit_chapter` 成功结束时，二次核对 Progress 已完成目标章、PendingCommit 为空、checkpoint 已追加，再发 `studio:project-invalidated`；前端只刷新 Overview、Tree、当前/新章和 Usage。失败事件只入日志/错误状态，不发章节完成。
7. 测试先覆盖状态派生、审阅 gate、Pause/Stop 时序、重复命令、项目切换旧事件隔离、Usage 差值、工具事件映射和 commit 失败窗口。对 Host 使用可控 fake/接口注入；真实 Host 集成测试用临时 Store、假模型，不调用在线 provider。Go/前端测试与 Wails 构建分开报告；没有原生窗口实测不得称完整 M3 验收通过。

## 4. 待确认的语义边界

- 当前 Core 没有独立 force stop。V1 的 Stop 建议定义为“安全取消并关闭本次 Host 会话”，以后可通过新 Host 从 Store Resume；不调用 `os.Exit`。
- `Host.Resume` 在新项目或完本时可能返回空 label 且不启动；Studio 要把这视为无可继续的事实，不伪报 Running。新书冷启动属于单独的 `PrepareUserRules` + `StartPrepared` 路径，不能混入已有项目 Continue。
- Review 的 `WaitingReview` 需要确切的边界判断或 Gate 事件+Store 二次核对。若当前事实不足以判断，只显示 Paused/Review 提示，不假设所有 review 项目必停。
- `Host.New` 会写盘和可能迁移旧格式；启动前应明确提示并验证备份/迁移恢复策略，且不能让打开项目隐式触发。
