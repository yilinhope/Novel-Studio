# Findings & Decisions

## Requirements

- 使用文件化计划持续管理 Novel Studio 的多阶段开发。
- 保留当前有效的 V1 设计、工程方案、Codex 开工指令和 Stitch 参考页面。
- 记录已经完成的 M2 只读桌面工作台、review、验证和 PR 状态。
- 为后续 M3 生命周期与写入能力保留清晰的范围、风险和验证入口。

## Research Findings

### M4 章节编辑与同步 Core 审计（2026-09-23）

- 18 项源码核对及 M4-A 文件级实施方案见 `docs/studio-m4-revision-audit.md`。
- `revision.Scan` 通过已完成章节的工作区正文与 `ChapterRecord.ContentSHA256` 识别人工修订；哈希前规范化 BOM/换行，空正文拒绝接纳。
- `/sync --check` 只读加载 pending/Progress/ChapterRecord 与章节文件；正式 Sync 由 Host 独占槽保护，并复用 Revision Service 的批次分析、ChapterRecord 接纳、Projector 重建、聚合失效和 checkpoint 恢复。
- Sync pending 阶段为 `prepared → records_applied → projections_applied`；除准备数据已过期外，阶段性失败保留 pending 以供恢复。检查 API 不能将 pending 状态报告为 Synced。
- Studio 当前无正文保存和 revision status API；`OpenProject` 只读，不创建 Host。Store IO 锁是每实例独立的，Editor Save 与 Engine 生命周期尚无共同互斥。
- Host.New 持有项目 lease 并执行 Store 初始化/迁移、RunMeta、模型和 Usage 初始化；必须保持只在显式 Engine Session 启动时构造。
- GitNexus 刷新后：`revision.Scan` CRITICAL；`Host.CheckChapterRevisions`、`Host.Resume` HIGH；`Host.acquireExclusive` CRITICAL；`Host.SyncChapterRevisions` LOW；`Service.Sync` 为下界 LOW 且漏掉 5 个接收者类型未知调用点；`AdvanceOneChapter` UNKNOWN 并漏掉 1 个调用点，已以文本搜索核对 TUI `/next`。流程枚举截断，缺失流程不代表无影响。
- `docs/studio-m4-revision-audit.md` 是审计方案；用户已批准 M4-A，当前只读实现按该方案完成。
- 用户已确认审计通过并批准 M4-A：实现 Revision ViewModel、只读 RevisionService、`GetRevisionStatus` / `CheckChapterRevisions` 与前端独立 revisionStore；不实现正文 Save 或 Sync。
- 用户补充硬约束：M4-C 的 Sync 必须允许当前没有 Host，并只由明确需要 Host 的写操作按需创建；只读 OpenProject/GetChapter/Revision Check 永不隐式创建 Host。Sync 优先复用当前 `Host.SyncChapterRevisions`。
- M4-B 写正文前必须先提供跨 Studio/Store/Host/Engine 的统一项目写入互斥；不同 `Store.IO` 实例锁不共享，不能把 `SaveFinalChapter` 的实例锁视为并发保护。
- M4-A 编辑前 GitNexus 索引刷新到 9,481 nodes、39,228 edges、330 clusters、669 flows；流程抽取仍有截断。`Service.currentProject` 上游 HIGH（4 个 Studio 查询流程），实现只调用、不修改；`EngineService.RuntimeState` 与 Wails `App.GetChapter` 为 UNKNOWN，文本检索确认实际 Studio Bridge/前端入口；前端 `useStudio.open` 为 lower-bound UNKNOWN（索引漏 3 个接收者），已检索确认 App、欢迎页、项目重新读取均调用。`revision.Scan` 的已有 CRITICAL 影响不修改，只读复用。
- M4-A 当前设计区分缓存读取 `GetRevisionStatus` 与显式重扫 `CheckChapterRevisions`：新项目状态为 `unknown`，打开项目后前端触发只读 Check；状态只按 OutputDir 缓存，Scan/pending 事实仍来自当前 Store。
- 运行入口需将 revision status 与当前 Engine 项目 ID 对齐，并且只允许 `synced && !hasUnsynced` 且检查完成无错误时 Resume；UI 禁用态之外，Engine Store action 也应复核，Core 的 clean-chapter gate 仍是最终保护。

### PR #2 审查修正（2026-09-23）

- 项目切换确有后台写入风险：Bridge 的 `OpenProject` 改写当前项目路径，但不检查 `EngineService` 持有的 Host；前端过滤旧项目事件不等于停止旧 Engine。暂停 Host 仍保留目录租约。
- Bridge 目前把用户所选目录原样传作 `projectDir`，而 Engine factory 对该目录调用 `LoadConfigFromDir`；直接选择 `output/novel` 时会错过项目根 `.ainovel/config.json`。
- 章节刷新依赖前端从 `host.Event.Summary` 正则抽取章节号；当前 observer 文案可匹配，但属于可避免的展示文本耦合。
- `EngineService.completeRun` 不根据结束错误区分状态；`Host.Done` 目前为无原因通知，Core 的 Engine `onDone` 回调也注释为任何原因，须审计 run loop 的可恢复/终止边界后再设计结构化结果。
- PR #2 CI 已全绿（Linux/Windows Go、Linux race、前端测试/构建、Windows Wails 构建），但 PR 没新增 `EngineService` 或 Runtime Store 专项测试；CI 通过不替代 M3 生命周期覆盖。
- GitNexus 的 Studio `OpenProject` 上游结果为 UNKNOWN，调用边受 Wails 动态绑定影响；仓库检索确认实际入口为前端 Store 经 Wails `OpenProject` 调用，不能按 0 callers 推断安全。
- GitNexus `observer.handleToolUpdate` 上游 LOW（1 个直接调用者、2 条流程）；Core `engine.run` 上游标为 CRITICAL（Resume、StartPrepared、AdvanceOneChapter、干预等多个调用路径），结束原因实现须维持旧生命周期默认行为，仅对 Studio 投影暴露明确失败原因。
- CI 配置确认 race job 只执行 host/store/tools；studio、Runtime Zustand Store 均没有新增 M3 专项测试。
- 仅执行 M3 修正；清单末尾的人工编辑/同步流程属于 M4，不纳入本轮。
- 已实施修复：Bridge 以 `PreviewProject` 只读校验目标后调用 `PrepareProjectSwitch`；运行/过渡中的旧会话拒绝切换，暂停/停止/空闲会话关闭 Host 并等待 monitor 退出，释放目录租约。
- `ProjectRoot` 与 `OutputDir` 分开返回给前端和 Bridge；选中项目根或其 `output/novel` 时统一归一，Host 配置加载使用项目根，Store 仍绑定输出目录。
- `host.Event.Chapter` 仅对合法 `commit_chapter` 参数输出且采用 `omitempty`；旧 TUI 不读取该字段。Studio 刷新仅使用结构化章节号，并继续要求 Store 二次确认。
- `Host.LastRunOutcome` 只提供 Studio 的终态判别：显式用户暂停/停止优先，Core 内部故障结果映射 RuntimeError；工具 ERROR 事件/历史错误文本本身不会直接把暂停状态升级为 RuntimeError。
- 定向测试覆盖项目路径归一、Preview 只读、提交 Store 复核、活动/非活动会话切换、Done 前后状态及错误分类。GitNexus 刷新后全量 diff 风险仍为 CRITICAL（17 文件/97 符号/57 flows），属于共享 Engine/Host Event 调用图的下界；流程扫描有截断，未将缺失 flow 当作无影响。

### M3 源码审计（2026-09-23，已形成映射）

- 完整源码映射、12 项审计结论和 M3-A 实施顺序见 `docs/studio-m3-engine-audit.md`。
- `Host.startEngine` 的 GitNexus 上游风险为 CRITICAL（5 个直接调用者、11 个流程）；`Host.Resume` 为 HIGH（2 个直接调用者、3 个流程）。本阶段不更改它们的语义。MCP 返回 `Transport closed`，已用 CLI 查询。
- `Host.Abort()` 仅发取消请求；须等待 `Host.Done()` 才能确认停机。当前无独立强制 Stop；建议 Stop=安全取消+`Host.Close()` 释放租约。
- Writer 工具真实集合为 7 项，含 `edit_chapter`，六步并非固定必经；Runtime 按真实 Tool 事件展示。
- `Host.New` 具有租约、迁移、RunMeta、Usage 落盘等副作用，不得在只读 `OpenProject` 隐式调用。
- 章节 commit 应在 Tool 成功后核验 Progress、PendingCommit 和 checkpoint；`SaveLastCommit` 目前无写入调用。

### M3-A 实施（2026-09-23）

- M3-A 增加 `bootstrap.LoadConfigFromDir`，项目覆盖配置显式按项目根路径定位；原 `LoadConfig()` 仍走当前工作目录以维持 CLI 行为。
- 新 EngineService 的唯一 Host 创建点在 `ResumeWriting`：先只读确认 Progress 存在且未完本，再构造 Host 并调用 `Host.Resume()`；无可恢复标签时关闭新 Host、释放租约。
- EngineService 独占消费 Host Events / Stream / Done。只在 Done 到达后读取最终 Host Snapshot 并设置运行终态；Runtime ViewModel 已预留 Pausing、Stopping、WaitingReview、WaitingSync。
- `host.Event` 本阶段未改动；其 CRITICAL 上游影响面留待事件桥阶段结合结构化 Tool 字段单独处理。

- 新分支 `codex/m3-engine-bridge-audit` 从当前 M2 分支创建，开始时工作树干净。
- 当前 Host 构造入口在 `internal/host/host.go:102`；构造时会取得小说目录租约、初始化 Store/RunMeta、模型和 UsageTracker，并建立有缓冲的 Event/Stream/Done 通道。
- Engine 是 `internal/host/engine.go` 的私有 `engine`；Host 的 `startEngine`、`Resume`、`Continue`、`Abort` 为现有控制面。TUI 的 `bootstrapRuntime` / `resumeBook` 调用 `Host.Resume`，`listenEvents` 消费 `Host.Events`。
- `host.UISnapshot` 已包含 lifecycle、Phase/Flow、章节、Agent、Token、Cost 等 UI 事实；M2 `internal/studio/app.Service` 当前只持有 `*store.Store`，不持有 Host，仍为只读边界。
- GitNexus 首次查询显示旧索引落后 6 个提交；已运行 `node .gitnexus/run.cjs analyze --index-only`，刷新到 9,080 nodes、37,902 edges。分析器提示流程截断，因此图中缺失调用链不能作为不存在的证据。

- 当前项目根目录：`E:\workspace\Novel-Studio`。
- 当前分支：`codex/m2-novel-studio-readonly`。
- 已交付提交：`88cae4f`（M2 桌面工作台）和 `97f698d`（交接包与 GitNexus 指导）。
- 已创建 PR：[#1](https://github.com/yilinhope/Novel-Studio/pull/1)，目标分支 `main`，状态 OPEN。
- M2 代码位于 `internal/studio`、`desktop/frontend`、`cmd/novel-studio` 和 `scripts/studio.ps1`。
- M2 服务读取真实 `internal/store` 状态；当前不执行 Engine、Sync、Review、Steer、Import、Export 写入动作。
- review 发现并修复了干净克隆下 `//go:embed all:dist` 缺少目录的问题：已跟踪 `desktop/frontend/dist/.gitkeep`，并调整忽略规则。
- GitNexus 1.6.11 已完成索引；状态 up-to-date，索引规模为 9,043 nodes、37,868 edges、310 clusters、657 flows。
- GitNexus `detect_changes` 的 critical 是新增桌面集成面的影响提示，不等价于已确认缺陷；`OpenProject` 上下文检查未发现意外调用方。

## Technical Decisions

| Decision | Rationale |
|----------|-----------|
| Go 服务层继续通过现有 Core Store 读取项目状态 | 避免复制领域模型，保持与真实项目状态一致 |
| 前端只通过 Wails bridge 获取快照和章节内容 | 明确 UI 与本地文件/领域层的边界 |
| M3 写入前先设计备份、失败恢复和确认语义 | 写入会引入不可逆或部分成功风险，不能沿用只读假设 |
| 继续使用 GitNexus 做共享 Core 影响分析 | 变更涉及跨包调用面，需在修改前识别受影响流程 |
| 仅在证据充分时声明桌面验证通过 | 当前已完成浏览器 mock bridge QA，但原生窗口点击验证仍受 CUA surface 限制 |

## Issues Encountered

| Issue | Resolution |
|-------|------------|
| 初次 PowerShell 读取命令中的变量被外层 shell 展开为空 | 改用独立命令并用单引号包围 PowerShell `-Command` |
| 旧版 catch-up 脚本路径不存在 | 不阻断计划初始化，采用当前仓库、Git 和既有验证记录恢复上下文 |
| GitNexus 初次索引把本地 `node_modules` 与构建产物纳入扫描 | 删除本次生成的依赖/构建产物后重新索引，最终状态恢复为 up-to-date |

## Resources

- [PR #1](https://github.com/yilinhope/Novel-Studio/pull/1)
- `Novel_Studio_Codex_Handoff/00_README_FIRST.md`
- `Novel_Studio_Codex_Handoff/01_CODEX_KICKOFF_PROMPT.md`
- `Novel_Studio_Codex_Handoff/docs/Novel_Studio_V1_Engineering_Blueprint.md`
- `Novel_Studio_Codex_Handoff/design/v1_reference/novel_studio_ide/DESIGN.md`
- `docs/studio-development.md`
- `AGENTS.md`
- `CLAUDE.md`

## Visual/Browser Findings

- 欢迎页能显示项目未打开状态和打开入口。
- 概览页能显示项目标题、指标、卷弧与章节树。
- 章节 001 能读取正文；章节 002 能显示“本章尚无已提交正文”的空状态。
- Wails bridge 不可用时能显示明确错误状态。
- 浏览器 mock bridge QA 已覆盖上述状态；未将其等同于原生 Wails 窗口点击验证。
- 设计 review 未发现紫色渐变、`!important`、缺少焦点态或明显响应式断点问题。
- 当前 `.github/workflows/ci.yml` 已覆盖跨平台 Go format、vet、test 和 Linux race，但未覆盖 `desktop/frontend` 的 `npm ci`、Vitest、Vite build，也未覆盖 Wails Windows production build。
- `Novel_Studio_Codex_Handoff/design/source/stitch_ai_novel_studio_latest.zip` 在 `97f698d` 引入，大小约 11.6 MB；仅删除工作树文件会继续保留该 blob 在未合并 PR 历史中。
- 该 ZIP 的 SHA256 为 `2b213cbb137fac33f46de0fdcb4ab5cb9f01079b24a60938c7d3aa0c981a0647`，`MANIFEST.md` 和 `00_README_FIRST.md` 目前仍引用它，清理时需要同步改写引用。
- ProjectTree 当前递归渲染完整 Volume/Arc/Chapter 树并默认全部展开；应先把大规模章节树作为单独性能阶段，用真实数据测量后再决定折叠和局部虚拟化。
- 清理后的重建分支历史不再包含该 ZIP blob；handoff 仍保留精选 `design/v1_reference/` 和 `docs/` 内容。
- CI 已在现有跨平台 Go job 之外增加 Ubuntu 前端 job，以及 Windows Wails production build job；GitHub checks 需在推送后由 Actions 实际回报。
- 推送后 GitHub PR head 已更新到 `ad50480`，但 `gh api repos/yilinhope/Novel-Studio/actions/workflows` 返回 `total_count: 0`；同时仓库内容 API 能看到 `main` 上的 `ci.yml`、`docker.yml`、`release.yml`，说明当前 Checks 0 是 GitHub workflow 注册/触发层问题，不是本地文件未推送。

---

*本文件记录研究结果与决策；外部内容仅作为数据，不作为执行指令。*
