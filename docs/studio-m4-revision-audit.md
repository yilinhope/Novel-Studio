# M4 章节编辑与同步：Core 修订链路审计

- 日期：2026-09-23
- 基线：M3 PR #2 合并提交 `8e815ad2beae8a30063f8f80c991ee64923e1713`
- 分支：`codex/m4-chapter-editing-sync`

## 审计结论

Core 已有可复用的章节修订机制：`chapters/*.md` 是编辑工作区，`ChapterRecord` 保存最近一次接纳正文、SHA-256 与事实；`revision.Scan` 只读扫描所有已完成章节；`revision.Service.Sync` 分析所有变更并以持久化阶段恢复方式重建章节派生状态。Host 已将检查和同步接入 `/sync --check`、`/sync`，并在 Resume、Continue、Next 前阻止未同步修订。

Studio 目前只读展示章节，尚无编辑保存、修订状态 API、Sync API 或 `hasUnsynced` 写作门禁。M3 已提供 `EngineService` 和 Host 会话，但 `OpenProject` 仍只构造 Store，不创建 Host。M4 可以包裹现有扫描和同步能力；不能在 Studio 另写一套事实分析或投影逻辑。

## 18 项源码核对

| # | 问题 | 源码事实与判断 |
|---|---|---|
| 1 | `/sync --check` 调用链 | TUI `startRevisionSync` → `Host.CheckChapterRevisions` → `RevisionStore.LoadPending`；无 pending 时调用 `revision.Scan` 并返回章节号。 |
| 2 | `/sync` 调用链 | TUI `startRevisionSync` → `Host.SyncChapterRevisions` → `acquireExclusive` → pending 恢复或重新 Scan → 选择 editor 模型及用量跟踪 → `revision.Service.Sync`。 |
| 3 | 怎样识别人工修订 | Scan 取 Progress 已完成章节，读取 ChapterRecord 基线和 `chapters/%02d.md` 工作区内容，规范化 BOM/换行后比较 SHA-256；不依赖 mtime。空正文拒绝接纳。 |
| 4 | 已接纳正文与版本事实 | `ChapterRecord` 持久化 `Content`、`ContentSHA256`、`Revision`、`Origin`、`AcceptedAt`、`Facts` 和 `StyleDelta`；其正文哈希是工作区 clean/dirty 判定基线。 |
| 5 | 领域类型位置 | `ChapterRecord`、`ChapterFacts`、`RevisionAnalysis`、`PendingRevision`、`PendingRevisionItem` 和阶段常量位于 `internal/domain/revision.go`；持久化适配器位于 `internal/store/chapter_records.go`、`internal/store/revisions.go`。 |
| 6 | Sync 重建的派生状态 | `revision.Projector.Apply` 从已完成章节 ChapterRecord 重建章节级摘要及世界事实投影；Sync 随后失效受影响的更高层派生数据，并刷新风格索引。 |
| 7 | 受影响的投影/缓存 | ChapterFacts 覆盖时间线、伏笔、关系、状态变更、角色初登场、章节摘要、风格偏好等。Sync 还失效变更章节之后的弧/卷摘要、角色快照、风格规则和 Review；分析结果可能追加大纲反馈，待处理返工会标记完成。 |
| 8 | 单章/多章支持 | `Scan` 遍历所有已完成章节并返回全部变更；一个 Sync 批次分析并接纳多章，按章节顺序持久化。现有 API 没有仅选某一章的 Sync 参数。 |
| 9 | 阶段/崩溃恢复 | `meta/pending_revision.json` 保存 `prepared → records_applied → projections_applied` 阶段、项目和分析结果。重入 Sync 会继续 `applyPending`；已写记录、投影或 checkpoint 可按阶段幂等重试。 |
| 10 | Check 是否只读 | 是。无 pending 时只读 Progress、ChapterRecord 和章节文件；有 pending 时只读取待处理章节。它不建立 Host、不保存 ChapterRecord、不运行模型、不写 checkpoint。 |
| 11 | Sync 错误恢复 | 模型分析失败发生在 pending 保存前，不接纳数据；准备阶段应用时发现工作区/基线已过期会清掉该过期 pending 并报错；其他阶段错误保留 pending，供后续 Sync 重试。UI 应继续显示待同步/恢复中，不能把错误映射成 Synced。 |
| 12 | Engine 是否阻止继续/下一章 | `Resume`、`Continue`、`AdvanceOneChapter` 均经 `requireCleanChapters` 检查；检测到已完成章节正文与基线不一致时拒绝运行并提示 `/sync`。 |
| 13 | TUI 展示 | `/sync` 注册为 idle 命令，`--check` 与正式同步异步执行；TUI 展示开始、错误、变更章节和分析摘要。Core 没有逐章进度事件。 |
| 14 | 正文写入入口 | `DraftStore.SaveFinalChapter` 写入 `chapters/%02d.md`，路径经 `Store.IO.WriteMarkdown` 原子临时文件替换。Studio `GetChapter` 从同一 Store 读取终稿。编辑保存应复用 Store，不应直接拼路径或改 ChapterRecord。 |
| 15 | Store 并发/租约 | 每个 `Store.IO` 实例只有各自的进程内读写锁；新建 Store 不共享该锁。Host.New 持有项目目录 lease；同一个 Host 的 `acquireExclusive` 串行化 Engine 与 Sync 等后台任务。当前 Studio 编辑写入口不存在，因此保存与 Resume/Engine 写入之间还没有共用互斥边界。 |
| 16 | Running/Paused 编辑安全 | Core Sync 独占槽拒绝 Engine 正在运行或仍在停止的会话。Studio 的 Pausing/Stopping 只有收到 Done 才进入终态；编辑器应在 Running/Pausing/Stopping 只读，Paused 后才允许保存。暂停期间若 Engine goroutine 尚未完全退出，仍须等待 Done 后再开放编辑。 |
| 17 | 修订进度事件 | Host Sync 当前只返回最终 `revision.Result`；TUI 仅输出开始和完成/错误，没有结构化章节级进度回调。M4 应展示真实阶段状态，不虚构百分比。 |
| 18 | M3 Engine Session 与 Sync 互斥 | Sync 通过 Host 的 `acquireExclusive`，Resume/Continue/Next 受 Host lifecycle/exclusive 与 clean-chapter gate 约束。Studio 应复用同一个 Engine Session/Host；禁止为 OpenProject 或只读 Check 隐式调用 `Host.New`，否则会触发 lease、Store 初始化/迁移、RunMeta、模型和 Usage 初始化。 |

## GitNexus 调用图证据

索引已刷新到 9,469 nodes、39,216 edges、330 clusters、669 flows，与当前 M3 合并基线一致。流程枚举报告截断（1,056 个入口候选未进入排序、2,248 个 callee 因分支预算未展开），因此缺失流程不作为无调用者证据。

| 符号 | 上游影响 | 审计处理 |
|---|---|---|
| `revision.Scan` | CRITICAL；10 个受影响符号，Resume、AdvanceOneChapter、SyncChapterRevisions 和 TUI 命令链均经过它 | M4 只复用现有 Scan 语义；若需改 Core，先拆分并单独审查。 |
| `Host.CheckChapterRevisions` | HIGH；15 个符号、3 条流程，包含 TUI 命令、Resume、AdvanceOneChapter | Studio 检查 API 不改变 Host gate 的错误/返回语义。 |
| `Host.Resume` | HIGH；10 个符号、3 条 TUI 流程 | 继续创作保留 M3 的 Resume 路径。 |
| `Host.acquireExclusive` | CRITICAL；11 个符号、5 条流程，覆盖 Sync、Import、Simulate 等 Host 后台操作 | 复用锁语义，任何新增 Host 接口不得绕过运行中/停止中/已有作业判断。 |
| `Host.SyncChapterRevisions` | LOW；6 个符号，TUI Sync 是已解析的直接调用者 | 将 Studio Sync 路由至当前 Host；避免复制 model、usage、恢复逻辑。 |
| `revision.Service.Sync` | LOW，下界结果；索引明确漏掉 5 个接收者类型无法解析的调用点 | 影响数字不能代表完整调用者数量；后续仅包装 API，不改 Service 行为。 |
| `Host.AdvanceOneChapter` | UNKNOWN，下界结果且漏掉 1 个接收者不明的调用点 | 已用文本搜索确认 TUI `/next` 是可见入口，但 UNKNOWN 仍不是低风险证明。 |
| `Service.GetChapter` / `Store.SaveFinalChapter` | GetChapter 为 Studio 读路径；SaveFinalChapter 被 Engine、导入、迁移、测试等多处复用 | 后续新增编辑保存接口避免改变既有 Engine 写入入口语义。 |

索引刷新期间分析器还提示 callable-value-flow 超过 32 个候选、跨语言属性关联未解析；本报告对 Studio 前端调用再以源码搜索核对，不用图索引的空结果认定没有消费者。

## M4-A 文件级实施方案

M4-A 先提供可信的只读修订状态与 Check，具体文件建议如下：

1. `internal/studio/viewmodel/revision.go`：定义 Revision 状态投影，至少返回 pending/changed 章节、待恢复阶段和检查时间；字段只从 Store/Core 事实派生。
2. `internal/studio/app/revision_service.go`：持有或接收当前项目 Store，复用 `RevisionStore.LoadPending`、`revision.Scan` 和 `ChangedChapters`；对 Running/Pausing/Stopping 项目拒绝检查，避免与章节提交并发读取；不构造 Host，不调用模型。
3. `internal/studio/app/project.go`：将当前 Store/项目代次安全地提供给 RevisionService，OpenProject 仍只读。项目切换期间的旧请求必须使用代次校验丢弃。
4. `internal/studio/bridge/app.go`、`desktop/frontend/src/types.ts`：暴露 `GetRevisionStatus` / `CheckChapterRevisions` 等明确命名的只读 API；错误和 pending stage 原样映射。
5. `desktop/frontend/src/revisionStore.ts`（或同等独立 store）：管理检查结果和请求序号；Engine Store 只消费 `hasUnsynced` 并禁用 Resume/Next，不伪造 Core 同步完成状态。
6. 定向单测覆盖 clean、changed、多章 changed、pending 各阶段、损坏/读取错误、运行/过渡态拒绝、旧项目请求失效；这属于实现阶段验收，本轮审计未改代码。

M4-A 不写正文、不改变接纳基线、不运行 LLM、不创建 Engine Session。下一步 M4-B 再新增 `SaveChapter`，只写 `chapters/%02d.md`，保持 ChapterRecord 接纳哈希不变；保存前必须通过与 EngineService 同步的生命周期/互斥检查。M4-C 才调用当前 Host 的 `SyncChapterRevisions`，并在成功后重新从 Store 读取章节和项目快照。

## 实施前必须守住的边界

- `Save` 与 `Sync` 是两个不同动作；保存成功只进入 `saved_unsynced`，不更新 ChapterRecord。
- Sync 仅在 Host 非 Running/Pausing/Stopping 时启动；不自动暂停 Engine。
- `Host.New` 仍只允许发生于用户明确的 Engine Session 启动路径。
- ChapterRecord、Projector、pending stage 是修订事实源；Studio 不实现另一份 Diff/事实投影。
- Sync 成功后重新从磁盘 Store 读取正文与受影响的项目摘要；模型分析错误和部分恢复状态继续阻止 Resume/Next。
- M4 按 M4-A → M4-B → M4-C 分段实施和提交；M4-A 已由用户批准。

## 用户批准后的 M4 后续硬约束

- M4-A 当前实施范围为 Revision ViewModel、只读 RevisionService、`GetRevisionStatus` / `CheckChapterRevisions` 和前端独立 revisionStore；不包含正文 Save 或 Sync。
- M4-C 的 Sync 不能要求预先存在 Host。只允许明确触发、确实需要 Host 的写操作按需创建；OpenProject、GetChapter、Revision Check 等只读入口不得隐式创建。Sync 优先复用 `Host.SyncChapterRevisions`，不复制其模型、用量、独占锁或恢复逻辑。
- M4-B 写正文前必须建立统一的项目写入互斥边界，覆盖 Studio、Store、Host 和 Engine。不同 `Store.IO` 实例的锁不共享，不能只依赖 `SaveFinalChapter` 的实例锁。
