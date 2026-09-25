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

### M6 实施结果

- `Host.New` impact 为 HIGH（4 个直接调用方、主流程受影响）；修改仅增加可选 `WithConfigPath`，默认调用路径未改变，CLI/TUI/headless/eval 继续使用 `EffectiveConfigPath()`，Studio 显式传入 canonical ProjectRoot 配置路径；Go 全量测试与 Wails 构建通过。
- `PrepareProjectSwitch` impact 为 LOW；新增 exclusive/CoCreating 拒绝，避免长 Import/阶段共创在项目切换时被 Studio 关闭。
- Core 没有独立 outline synthesis API；现有 `/start <file>` 只是读取 prompt 文件后进入同一 `PrepareQuick`/`StartPrepared` 链，因此 Studio 的“从大纲开始”只做文件预览/校验并复用 Quick Start。
- Co-create 现有日志没有 unfinished 标记或 reader；Studio 只读最后完整 JSONL 记录并把已解析回复接回输入历史。阶段模式依赖现有 TUI opener 字符串识别，未新增日志格式。
- Budget 配置保存使用现有 `bootstrap.SaveConfig` effective path；由于 `BudgetSentinel` 在 Host.New 创建且 Core 没有热更新语义，Studio 在已有 Engine Session 时拒绝 Budget mutation，UI 明示“下一次 Host 生效”。
- 最终 GitNexus `detect-changes --scope all` 覆盖 15 个变更文件、261 个变更符号、39 个受影响符号，整体标为 `critical`。其中可定位的高风险点是 `Host.New`（HIGH）：已通过可选 `WithConfigPath` 保持 CLI/TUI/headless/eval 默认路径不变，并用 Go 全量测试与 Wails 生产构建验证；Bridge 的 `StartImport`、`StartCoCreate` 和跨 Wails 的 `SaveBudgetConfig` 为 `UNKNOWN`，原因是动态 Wails/TypeScript 属性调用无法完整解析，已用前端调用点搜索、领域 Store 测试、Go 全量测试和生产构建补证，不能将 UNKNOWN 解释为无影响。

### M6 Freeze Fixes

- Budget 写入改为 `bootstrap.SaveBudgetConfig(ProjectConfigPathFromDir(projectRoot), budget)`，只读取/重写项目层自身；即使项目配置原本不存在，也不会把全局 Provider、Model 或 API Key 复制到项目文件。
- Quick Start 与 Co-create 的 ack/event 共享前端生成的 `requestId`；Store 在 ack 尚未返回时暂存匹配的 terminal event，ack 到达后回放，覆盖立即失败/快速完成竞态。
- Co-create recovery 仍只读取最后一条完整 JSONL；Store hydrate 已落盘 History/Draft/Ready/Suggestions，并通过显式 `ResumeCoCreate` 继续 Core 共创，未落盘请求不会被恢复。
- Import UI 文案改为“文本文件（UTF-8 / GB18030）”，不再暗示 Core 支持 EPUB Import。

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

## M7 Audit Kickoff

- M6 冻结提交为 `5de0506`，PR #6 当前已同步且 CI 4/4 通过；本次 M7 从该基线开始，不新增产品能力。
- 当前 M7 首要审计范围：Studio API → Core/Host/Store 调用链、React 直接文件/config 访问、GUI 自行推导事实、Host.New 只读边界、ProjectRoot/OutputDir、异步 identity、Core success/view refresh failure、stale project/event、secret 泄漏。
- M6 已修正的创建完成态语义必须纳入回归：Core create 成功后即为 `completed`，视图刷新失败只能作为 message/warning。
- M7 回归必须把 GUI 状态（WaitingSync/WaitingReview/Pausing/Stopping/Syncing）视为展示投影，不能替代 Core Store/Host 事实。
- 跨进程写入 backlog 优先复用既有 Core/book lease；不新增第二套 lock file 协议。

### 初步源码审计（2026-09-24）

- GitNexus 已按当前代码重建，图规模为 60,069 nodes / 159,894 edges / 1,120 clusters / 802 flows；查询仍受动态分派、Wails 绑定和大文件裁剪影响，缺失调用不能当作不存在。
- `OpenProject` 当前只调用 `PreviewProject` / `service.OpenProject`、`PrepareProjectSwitch` 与 revision check，不直接创建 Host；`SetAdvanceMode`、`AdvanceOneChapter`、`SubmitSteer` 通过 `EngineService.lockedProjectHost` 才按用户写操作创建或复用 Host，符合 M4/M5 边界。
- `StartQuickStart` / `StartCoCreate` 已带 frontend requestId，并在 ack 前缓存事件；创建成功与视图刷新失败已分离为 completed + warning，M6 语义不能回退。
- 前端 Import 曾存在 ack 前事件风险：`importStore` 在 `StartImport` 返回 ack 前收到的事件因 `projectId` 仍为空而被直接丢弃；本轮已为 ImportOptions/ImportStatus/OperationAck 补 requestId 和 pending reconciliation，并以 event-before-ack 回归测试覆盖。
- `engineStore` 对同一项目的新 generation 会保留旧的 advance/review 投影字段，随后异步刷新虽通常会覆盖，但在刷新失败或快速切换时可能短暂显示旧 gate；需核对并以新 generation 的 Runtime/Store 事实重新投影，不能把旧 generation projection 当事实。
- `PauseWriting` / `StopWriting` 目前未经过 bridge 的 `projectMu`，而 Save/Sync/Next/Steer/ProjectSwitch 使用该锁；需要评估是否会让生命周期控制与项目写入/切换交错，修复时保持 Core Done 才确认终态。
- `StartImport` 在 ack 前已启动 `StartImport` 并可能同步发出 channel 事件；这与 M7 的 terminal-before-ack 要求相同，不能只依赖 UI disabled。
- 未发现 React 直接使用 Node `fs` 读写项目文件；前端项目/章节/配置读写均走 Wails bridge。API Key 对外为 `hasApiKey`/遮罩提示，但需继续核对日志和错误路径是否会把原始配置带出。
- 之前一次批量 PowerShell 取行脚本因把多维数组传给 `[Math]::Min` 触发参数类型错误，只影响审计输出，不影响工作树；后续改用单文件定向读取。
- SaveChapter/Budget 的短时 Studio 写操作现通过 Core book lease 保护：已有 Host 时复用其持有的 lease，没有 Host 时临时取得同一 `.ainovel.lock` lease；不会创建 Engine Session，也没有新增锁文件协议。
- `ProjectTree` 原先对所有 Volume/Arc 使用 `open`，已改为仅默认展开当前卷/弧，其余折叠并保留手动展开；Runtime 日志已有 500 条状态上限，展示层进一步只渲染末 120 条。

### M7 最终验证证据（2026-09-24）

- GitNexus 已在当前工作树重建：60,136 nodes / 159,936 edges / 1,137 clusters / 794 flows。索引报告了大文件跳过、动态 property site 和 process truncation；因此图结果是下界，UNKNOWN 仍按未解析边界处理。
- `detect-changes --scope all` 已在重建索引后完成，结果为 77 changed symbols / 14 affected symbols / 12 changed files，`partial=false`、`truncated=false`、整体 risk 为 `high`。高风险集中在 Bridge/App、配置写入和 Core book lease 共享轴，已逐项用源码搜索、定向测试、全量测试和生产构建补证；不能把 high 解读为无风险。
- Go 全量 `go test ./... -count=1` 通过 1019 tests，`go vet ./...` 通过；Frontend `npm test -- --run` 通过 7 files / 40 tests，`npm run build` 通过。
- Windows Wails production build 通过，产物为 `cmd/novel-studio/build/bin/Novel-Studio.exe`；构建仍有已知 `Not found: time.Time` 警告，不影响本次退出码，但需在运行时验收中继续观察。
- `.github/workflows/ci.yml` 已覆盖 Go test/vet、Linux race、Frontend test/build 和 Windows Wails production build，本轮无需重复添加 CI 工作流。

## V2-M1 Kickoff（2026-09-25）

- V2 蓝图位于 `C:\Users\linn\Downloads\Novel_Studio_V2_Product_Architecture_Blueprint.md`；用户已同意开始 M1 审计与实施。
- 第一闭环限定为 ReviewIssue → Proposal → Diff/Evidence → Accept → Apply → SavedUnsynced → Sync → Synced；Fact Engine UI 留待 M2。
- 当前审计已确认 V1 有 ReviewEntry、章节工作区、ChapterRecord SHA-256、原子章节保存、Store 项目写锁、Host book lease、Host Sync exclusive 和可恢复 `pending_revision.json`。
- 当前 V2 Proposal/Version/Evidence/Diff 持久化/API/UI 不存在；`ReviewIssue` 无稳定 ID；M1 的候选正文先由用户手工输入，避免擅自创建模型调用路径。
- 已知重点风险：Core Sync 是项目级扫描，Apply 多章节失败需要 V2 journal 恢复；V2 metadata 与 Core Store 的目录格式版本职责要分开，不能用全项目 format version 破坏 V1 migration。

### V2-M1 审计结论（2026-09-25）

- 13 项审计已完成，详见 `docs/studio-v2-m1-audit.md`。ReviewIssue 当前没有稳定 ID，M1 使用 `scope/chapter/issueIndex/issue 字段哈希` 生成确定性 source key；候选正文先由人工提供。
- Proposal Apply 与 Version Restore 只写 Core Drafts 工作正文，严格复用既有 `Service.SaveChapter`、Bridge project mutex 与 book lease；Core `ChapterRecord`、`SaveFinalChapter`、Host Sync 均不修改。
- `ChapterRecord` 与 `SaveFinalChapter` 的 GitNexus impact 为 HIGH，Studio `SaveChapter` 与 React `App` 为 UNKNOWN 下界；这些风险已确认不能当作无调用，本轮采用新增 V2 包和薄适配层规避。
- V2 metadata 采用 `<OutputDir>/meta/studio-v2/` 独立 schema_version=1、proposal/version/operation journal 文件，不修改 Core `meta/format.json`，也不参与 V1 migration。
- Apply 前必须为全部 change 做 BaseHash precondition；任何一个章节发生手工编辑都整体拒绝，状态为 Stale/Failed，不允许部分覆盖。Apply 后最高只到 AppliedWorkingCopy/SyncPending，Sync 成功并以 Core accepted hash 对账后才是 Synced。
- 现有 Core checkpoint 不能提供 Studio 版本历史；M1 新增可重建的正文快照，ContentHash 去重，Restore 仍回到 SavedUnsynced，不直接写 accepted revision。

### V2-M1 实施发现（2026-09-25）

- Proposal Manager 放在 `internal/studio/v2`，通过 `SaveChapterFunc` 注入 Core Save；因此 service 本身不能绕过 Bridge 的 project mutex/book lease，也不会直接取得或修改 ChapterRecord。
- ReviewIssue 入口按 `issue.Chapters` 处理单章目标；多章 issue 拒绝使用同一候选正文，提示在 Inbox 分别建立多 change，避免把一份正文盲写到多章。
- Apply journal 每次写章后立即记录已写章节；恢复扫描正文 hash。完整命中候选只恢复到 `AppliedWorkingCopy`，部分命中进入 `Failed`，全量未写入保留 `Accepted` 可重试。
- Sync reconciliation 不信任 UI 的 Applied 状态，只读取 Core ChapterRecord accepted hash；Bridge 只有在现有 Core Sync 返回成功、RevisionStatus 无未同步且 accepted hash 全部命中时才写 `Synced`。
- V2 schema 损坏或未知版本会在 metadata 写入前报错，不会修改 Core `meta/format.json`；版本快照按 resource + ContentHash 去重。
- Wails 生产构建在 `cmd/novel-studio` 目录通过并重新生成了被 gitignore 的 binding；从仓库根目录直接运行会因缺少 `wails.json` 失败，已记录为命令目录要求，不是代码失败。最新证据为 Go 1026 tests、Frontend 45 tests；Sync 失败路径会把 AppliedWorkingCopy 保守降为 SyncPending。

## V2-M1 review 修复（2026-09-25）
- 外部只读审查发现 Apply journal=applied 崩溃窗口、重复章节 change、Restore 陈旧覆盖、前端切项目响应污染、metadata 跨进程租约和路径/证据/Diff 边界问题。
- 已修复：applied journal 可恢复、同一资源重复 change 拒绝、Version BaseHash 与 restore journal、Bridge metadata/book lease、ReviewCenter 与 ProposalStore generation guard、Store 切换 busy gate、ID/QuotePreview 校验和超大 Diff 摘要降级。
- 定向回归：Go V2/Bridge 37 tests；全量 Go 1034 tests；Frontend 45 tests；go vet、Vite、Wails build 均通过。
