# M6 Progress Log

## Session: 2026-09-24 — M6 kickoff

- 读取用户提供的 M6 完整要求。
- 读取 `planning-with-files`、GitNexus exploring/impact、executing-plans、TDD、verification-before-completion 技能。
- 拉取远端 `main` 后确认 M5 PR #5 已合并：`origin/main` 为 `72add9b`。
- 从 `origin/main` 创建 `codex/m6-v1`；创建时工作区干净。
- 重置本阶段计划文件，后续所有审计发现、错误和验证结果持续写入本文件与 `findings.md`。
- 完成第一轮 GitNexus 图审计：`Host.New` 候选歧义但 Host 版本风险 LOW；`StartPrepared`、`CoCreateStream`、`ImportFrom`、`ConfigureModels` 直接调用方风险 LOW；Host `Snapshot` 的候选结果包含 UNKNOWN，已记录为需要源码补证而非安全结论。
- 完成 Core 初审：Quick Start、Co-create、Import checkpoint、TXT/EPUB Export、Provider 配置合并、Usage/Budget 持久化与 Host 生命周期边界已定位；Start from Outline 仍需确认是否存在正式公开入口。
- 补充源码审计文档 `docs/studio-m6-audit.md` 并提交 `84a942c`；审计确认没有独立 outline parser，GUI 复用 `/start` 的文件 prompt 语义。
- 修正 Studio Host 创建时的配置根目录：`newHostForProject` 使用 `EffectiveConfigPathFromDir(ProjectRoot)`，CLI 未传 option 时仍保持原 cwd 语义；补项目根配置回归测试。
- 新增 M6 Go Facade/Bridge：Quick Start、Outline preview、冷/阶段 Co-create、Core Import status/recovery、TXT/EPUB Export、Provider/Model/Thinking、Budget、Usage；只读读取不创建 Host，显式写操作沿用现有 Host/lifecycle/exclusive。
- 共创恢复仅读取既有 `meta/sessions/cocreate.jsonl`，不新增持久化格式；项目切换在 Create/Import/Co-create 长操作期间受 operation 与 Engine switch gate 保护。
- 新增前端 `createProjectStore`、`importStore`、`configStore`、`exportStore` 与对应 Create/Import/Settings/Export 页面；事件与异步响应按 projectId/generation/request sequence 过滤，API key 仅显示 Core 脱敏结果。
- 设置页补齐 Core 已有的 Agent 角色模型与 Reasoning Effort 控制，仍通过既有 `SwitchModel` / `SetRoleThinking` 写回并重新读取 effective config。
- 最终 GitNexus 检查发现跨 Wails 动态边界带来的 `UNKNOWN`/`critical` 风险包络，已对 `Host.New` 做 HIGH 影响说明，并以调用点搜索、全量测试、vet 与 Wails 构建补证，未发现 CLI 默认路径或现有 Core 语义变化。
- M6 冻结修正：Budget 改为项目层补丁写入；Quick Start/Co-create 增加 requestId 与 ack 前 terminal event 暂存；共创恢复 hydrate 已落盘历史/草稿并可显式 Resume；Import 文案改为 UTF-8 / GB18030 文本文件。

## Verification

| Command | Result |
|---|---|
| `git fetch origin main codex/m5-review-steer-bc` | passed |
| `git switch -c codex/m6-v1 origin/main` | passed |
| `go test ./... -count=1` | passed: 1017 tests |
| `go vet ./...` | passed |
| `npm test -- --run` | passed: 37 tests |
| `npm run build` | passed |
| `go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0 build` | passed: Windows production `cmd/novel-studio/build/bin/Novel-Studio.exe` |

## Session: 2026-09-24 — M7 kickoff

- 读取用户提供的 M7 Regression / Hardening / Release Readiness 要求；确认 M7 不新增产品功能、不扩展 V2、不重写 Core。
- 确认当前 M6 冻结基线为 `5de0506`，工作区在 M7 审计开始前干净，PR #6 CI 4/4 通过。
- 已创建 M7 计划阶段：源码审计 → CLI/GUI 回归与生命周期 → 跨进程写入/项目切换 → 长篇/Windows/发布验证。
- 首轮源码审计进行中；下一步先输出 Studio API、路径、Host 创建、异步 identity、事实源和 secret 扫描结果，再对真实回归直接实施最小修复。

## Session: 2026-09-24 — M7 hardening

- Import ack 前事件丢失已修复：ImportOptions/ImportStatus/OperationAck 共享 requestId，前端在 ack 前缓存同 requestId 的事件，ack 后回放；新增前端回归测试。
- SaveChapter/Budget 写入复用 Core book lease；无 Host 时临时取得同一 `.ainovel.lock`，不创建 Engine Session；新增已有 lease 拒绝 Save 的 Go 回归测试。
- ProjectTree 不再默认展开所有 Volume/Arc，仅默认展开当前卷/弧；新增长篇树 SSR 回归测试。Runtime 日志既有 500 条 Store 上限，UI 只渲染末 120 条。
- 当前定向/全量验证：Go `go test ./... -count=1` 1019 passed，`go vet ./...` passed；Frontend 40 tests passed，`npm run build` passed。

## Session: 2026-09-24 — M7 final verification

- 补回 `desktop/frontend/dist/.gitkeep`，避免前端/Wails 生成目录的构建副作用进入提交。
- M7 源码审计结论、跨进程 Core book lease、Import event-before-ack、长篇 ProjectTree 默认折叠和发布文档已记录；没有新增 Core 语义或第二套锁协议。
- 最终证据：Go 1019 tests、Go vet、Frontend 40 tests、Frontend build、Windows Wails production build 均通过；GitNexus 已重建并完成非 partial/non-truncated `detect-changes`，结果为 high risk，已按动态边界保守记录。
- 发布前仍需人工/现场完成真实模型与 GUI 全流程、跨进程 CLI/GUI 同时写入、中文/空格/长路径/权限异常，以及 100/300/500/1000 章实际项目压力检查；这些未被静态测试冒充为已完成。

## Session: 2026-09-25 — V2-M1 kickoff

- 用户确认按 V2 产品架构蓝图开始 M1：先审计并记录实施计划，再连续实现首条 Proposal → Core Sync 闭环。
- 已读取既有计划、findings、progress、本地 `AGENTS.md` 与 RTK 规则；当前分支 `codex/m7-hardening`、提交 `0f0fe4b`、工作树干净。
- V2 M1 约束已写入本轮计划；下一步创建 `codex/v2-m1-proposals`，然后完成 13 项源码审计和代码符号 impact。
- 错误记录：首次使用 GitNexus positional query 含空格时被 CLI 拆分参数；已确认应使用 `-q` 单参数形式。历史 V2 审计已完成 Review/Chapter Save/Sync 基础梳理，本轮将补足 13 项并重新验证当前 HEAD。

## Session: 2026-09-25 — V2-M1 audit complete

- 已在 `codex/v2-m1-proposals` 完成 13 项源码审计，并新增 `docs/studio-v2-m1-audit.md`。
- 已冻结实施边界：Proposal/Version 是治理 metadata；正文 Apply/Restore 只能经 V1 Save；Core Sync 是唯一 accepted 入口；手工候选正文先行；Fact/Impact/Repair 留在后续 milestone。
- 已记录 GitNexus HIGH/UNKNOWN 风险与保守决策：不改 `ChapterRecord`、`SaveFinalChapter`、Host Sync 语义；新增 V2 service/store/viewmodel，再由 Bridge 薄适配。
- 下一步按 TDD 先建立 V2 proposal store/service 的失败测试，再实现 schema_version、atomic metadata、BaseHash stale、防部分 Apply 与 operation journal。

## Session: 2026-09-25 — V2-M1 implementation

- 按 TDD 先让 `internal/studio/v2/proposal_test.go` 因缺少 Manager/API 编译失败，再实现 Proposal/Version/Diff service；当前 V2 定向测试 30 个通过。
- Proposal metadata 使用独立 `meta/studio-v2`、schema_version=1、原子 JSON；Apply journal 可在重启后根据正文 hash 区分未写入、完整写入和部分写入。
- Apply 前对全部 change 做 BaseHash precondition；任一手工编辑都会 Stale 且不写任何章节。Apply 前/后保存去重 Version Snapshot，Restore 通过 Core Save 回到 SavedUnsynced。
- Bridge 已提供 List/Get/Create/ReviewIssue/Accept/Reject/Apply/Diff/Version History/Restore；Sync 成功后只用 Core ChapterRecord accepted hash 对账为 Synced，Sync 失败保留 SyncPending。
- 前端新增建议收件箱、Proposal Detail、Diff/Evidence、版本历史/恢复和人工候选正文；Review Center 的 ReviewIssue 可直接进入 Proposal 创建表单。候选正文第一版不触发模型生成。
- 当前验证：Go 全量 1026 tests、`go vet ./...`、Frontend 45 tests、Vite production build、Wails Windows production build 已通过；Wails 仍打印既有 `Not found: time.Time` 绑定警告，未阻断构建。

- 提交前自审：按 Core 单一事实源、V2 metadata 独立目录、Apply 全量前置校验、Sync 对账和项目切换响应保护逐项复核；未发现新的 Critical/Important 问题。
- 最终 GitNexus：analyze --index-only 完成；detect-changes --scope all 为 20 files / 373 symbols / 11 flows / high risk，未见 partial 或 truncated 结果。

## Session: 2026-09-25 — V2-M1 review fixes
- 只读代码审查最初结论为 Not Ready：修复 Apply applied journal 崩溃恢复、同章重复 change、Restore BaseHash/operation journal、前端 Review/Proposal 项目 generation guard、切换 busy gate、Bridge metadata book lease。
- 增加 ID 路径校验、Evidence QuotePreview 绑定、超大正文 Diff 摘要保护、Review SourceKey 重复拒绝，以及对应 Go 回归测试。
- 当前验证：Go 1034 tests、go vet、Frontend 45 tests、Vite build、Wails Windows build 通过；真实桌面和模型运行仍需现场验收。

## Session: 2026-09-26 — Core GUI Parity audit

- 已将产品目标章程和 Core GUI Parity 完成计划加入 `docs/PRODUCT_GOAL_CHARTER.md`、`docs/CORE_GUI_PARITY_COMPLETION_PLAN.md`。
- 已将 `docs/V2_M2_FACT_AUDIT.md` 标记为 Deferred Research；未保留 Fact Engine/Character Knowledge 实现。
- 已基于当前 `origin/main` 源码生成 `docs/CORE_GUI_PARITY_MATRIX.md`，覆盖 TUI 命令、Host/Store/Config、Import/Export、Simulation、Rules/Style、Diagnostics 和 Runtime 观测。
- 本轮停在审计阶段，不进入 Wave A 代码实现；静态矩阵结论不等同于真实桌面运行验收。

## Session: 2026-09-26 — Wave A kickoff

- 阅读 `docs/PRODUCT_GOAL_CHARTER.md`、`docs/CORE_GUI_PARITY_COMPLETION_PLAN.md`、`docs/CORE_GUI_PARITY_MATRIX.md`；确认本轮只进入 Wave A Story Data Center + Continuity Center。
- 从最新 `origin/main` `a463521` 创建 `codex/wave-a-story-continuity`，工作树干净。
- 重建 GitNexus 索引并完成相关 Store 方法 impact；高风险来自共享 Store 读取调用图，本轮采用新增 Studio Facade/Bridge 适配，保持 Core Store 不变。
- 下一步：先实现只读 ViewModel/Facade/Bridge，再实现 React Store/导航/页面和定向测试。

## Session: 2026-09-26 — Wave A implementation

- 新增 Story/Continuity ViewModel、Studio Facade 与 Wails Bridge，只读取现有 Core Store；所有分页响应回显 ProjectRoot、OutputDir、generation、requestId、sequence。
- Story Data Center 已接入 premise、人物、世界规则、扁平/分层大纲、Compass、章节/弧/卷摘要；分层章节和摘要按页惰性读取，扁平与分层大纲可分别切换。
- Continuity Center 已接入时间线、伏笔、关系、状态变化、最新角色快照和 Core BuildCast 首次出场投影；没有新增事实源或前端推导。
- 新增 React parity store、导航页面、loading/error/empty、分页与项目切换 stale response 保护；新增 4 个前端回归测试和 Core/Bridge 大列表、空数据、无 Host 测试。
- 当前定向验证：Go `go test ./internal/studio/...` 124 passed；Frontend 51 tests 与 production build 通过。下一步运行全量 Go/vet、Windows Wails、GitNexus detect-changes 后提交统一 PR。

## Session: 2026-09-26 — Wave A final verification

- Go 全量 `go test ./... -count=1` 通过；`go vet ./...` 通过。
- Frontend `npm test -- --run` 通过 10 files / 51 tests；`npm run build` 通过。
- Windows Wails production build 通过，产物为 `cmd/novel-studio/build/bin/Novel-Studio.exe`；仍有既有 `Not found: time.Time` 绑定警告，不影响构建退出码。
- GitNexus 重建后 `detect-changes --scope all` 完成，`partial/truncated` 未报告，225 个变更符号、5 个受影响流程，风险为 medium；风险集中在既有 Bridge/App 动态边界，未修改 Core Store。
- 构建生成目录副作用已还原 `desktop/frontend/dist/.gitkeep`；当前只保留 Wave A 源码、测试、文档与计划变更。

## Wave A review 修复（2026-09-26）

- 修复 parity 导航绕过 dirty 章节离开确认的问题；取消保留当前章节和草稿，确认后清理草稿再进入 Story/Continuity。
- 修复分层卷列表超过 50 项翻页时 Arc Selector 使用旧页的问题；选中 Arc 的章节详情分页仍保留原 selector。
- 角色快照现在同时支持最新快照与指定 Volume/Arc 历史快照，后端分别使用 Core `LoadLatestSnapshots` 与 `LoadSnapshots`。
- 定向验证阶段：Studio 测试通过，Frontend 55 tests 与 production build 通过；最终全量验证待本轮完成。

## Wave B kickoff（2026-09-26）

- PR #12 通过 `gh pr view 12` 确认已于 2026-09-26 合并，merge commit 为 `1721ab4e4e9bb402fa4dd04c216646c89b39cf49`；4 项 CI 均成功。
- 已从最新 `origin/main` 建立 `codex/wave-b-core-gui-parity`，起始工作树干净。
- 已读取 PRODUCT_GOAL_CHARTER、CORE_GUI_PARITY_COMPLETION_PLAN、CORE_GUI_PARITY_MATRIX；Wave B 限定 Simulation 与 Rules/Style/Voice，不扩大到 Wave C 或原 V2-M2。
- 当前阶段为 Core 调用链与 GitNexus 轻量实施审计；审计结论未完成前不写业务代码。

## Wave B implementation and verification（2026-09-26）

- Simulation 已接入显式项目语料目录、Core profile/source 只读读取、导入、分析、取消和真实 stage/current/total 事件；只读路径不创建 Host，运行与导入复用 Host exclusive、project write 和 book lease。
- Rules 已接入全局/项目原文件列表、显式读取、原子新建/编辑/重命名/删除；Effective 只读取已有 Core snapshot，不在只读页面触发模型或隐式写入。Host 的项目规则路径由当前 ProjectRoot 注入，避免依赖进程 cwd。
- Style/Voice/anti-AI-tone 已接入 Core assets/config 的选择、优先级来源展示和项目/全局覆盖写入；保存受 Host mutation gate 与 book lease 保护，页面明确修改在新 Host 生效；WorldStore.LoadStyleRules 仍按矩阵保留 Partial。
- 前端新增参考作品与写作设置导航、Simulation/Rules/Style 页面、dirty navigation guard、项目切换 stale 响应保护；补充 Wave B store stale 回归测试。
- 当前验证：Go `go test ./...` 1055 passed、`go vet ./...` passed；Frontend 11 files / 58 tests、production build passed；Windows Wails production build passed。Wails 仍打印仓库既有 `Not found: time.Time` binding warning，不影响构建退出码。
- GitNexus 已在本轮代码编辑前完成索引与关键符号 impact；提交前仍需运行 `detect-changes --scope all`，确认结果非 partial/truncated。
