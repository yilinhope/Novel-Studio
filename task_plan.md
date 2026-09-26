# Task Plan: Novel Studio V1 — M6 / M7

## Goal

在已合并的 M5 基线上，把 ainovel-cli 的日常工作流接入 Novel Studio GUI：Quick Start、Co-create、Start from Outline、Import Novel、TXT/EPUB Export、Provider/Model 配置和 Budget/Usage。Core 继续是唯一事实源，GUI 不复制项目初始化、Import、Export 或配置语义。

## Baseline

- 分支：`codex/m6-v1`
- 基线：`origin/main`，M5 PR #5 合并提交 `72add9b`
- 工作区：创建计划时干净
- 约束：先完成源码审计并写入 `docs/studio-m6-audit.md`，审计后连续实施，不等待阶段确认；只有 Core 能力不存在、必须改 Core 语义、需要持久化迁移或存在高风险 CLI 破坏时停下汇报。

## Phases

### Phase 1: M6 计划与源码审计

- [x] 从最新 `origin/main` 创建 `codex/m6-v1`
- [x] 刷新 GitNexus 索引并检查当前分支状态
- [x] 审计 Quick Start / Co-create / Outline / Import / Export / Config / Budget 的真实 Core 调用链
- [x] 做关键符号 upstream impact 分析，处理 HIGH/CRITICAL 与 UNKNOWN 风险
- [x] 将审计结果、调用链、互斥边界和实施方案写入 `docs/studio-m6-audit.md`
- [x] 记录审计发现并进入连续实现
- **Status:** complete

### Phase 2: M6 后端基础与 Studio 领域服务

- [x] 建立共享 operation/project/generation 隔离和错误分类基础
- [x] 增加只读 Config/Usage/Export Preview 能力，不隐式创建 Host
- [x] 按 Core 真实入口实现 Create、Co-create、Outline、Import、Export、Config、Budget 服务
- [x] 对需要 Host 的显式操作接入现有 lifecycle、exclusive、projectwrite、book lease 和 Core gate
- [x] 先写失败测试，再实现 Go service/bridge API
- **Status:** complete

### Phase 3: Wails Bridge 与前端领域 Store

- [x] 暴露真实 Wails API 和事件/快照投影
- [x] 新增 `createProjectStore`、`importStore`、`configStore`、`exportStore`
- [x] 保持既有 `engineStore`、`revisionStore`、`reviewStore`、`chapterEditorStore` 边界
- [x] 所有异步请求带 projectId、generation、request sequence，旧项目响应不得污染当前项目
- [x] API Key 默认遮罩，日志、console、event summary、error 和 snapshot 不出现 secret
- **Status:** complete

### Phase 4: 创建与导入工作流 UI

- [x] Welcome 真实入口：快速开始、共创创建、从大纲开始、导入已有小说、打开已有项目
- [x] Quick Start Wizard 映射 Core 必填输入、真实 Agent/Step/Usage 和失败恢复
- [x] Co-create 映射阶段协议、消息、Agent 输出、动作、完成状态和 unfinished recovery
- [x] Outline Preview/Validate/Create 使用 Core parser/validation/synthesis
- [x] Import 8 阶段 UI：选择、Ingest、章节识别、确认、Analyze、Synthesize、Publish、完成
- [x] Import 长任务不阻塞 UI，显示真实 lifecycle/checkpoint/recovery，遵守互斥
- **Status:** complete

### Phase 5: Export、Config、Model、Budget UI

- [x] Export Dialog：TXT/EPUB、Core 支持的 range、destination、overwrite、真实输出路径
- [x] Provider/Model 页面只映射 Core 字段和读写能力
- [x] 明确 ProjectRoot 与 OutputDir，配置始终从 canonical ProjectRoot 读取，Store 使用 Core 规定 OutputDir
- [x] Model 切换遵守 Core 的运行中限制和下一次调用语义，不偷偷重启 Engine
- [x] Budget 页面显示 Spent、Book Budget、Warn Ratio、Hard Stop、Tokens、Cost 及真实 Agent Usage
- **Status:** complete

### Phase 6: 测试、CLI/GUI 对照与交付

- [x] 补齐 M6 Go/Frontend 定向测试和 stale operation 测试
- [x] 完成 Go tests、vet、Frontend tests/build、Windows Wails production build
- [x] 完成 CLI/GUI 最终 Store/Config 语义对照
- [x] GitNexus `detect-changes --scope all`、diff review、提交中文说明
- [x] 创建/更新统一 M6 PR，附审计摘要和验证证据（PR #6）
- **Status:** complete

## Global Constraints

- 不重写 Core 初始化、Import、Export、Config、Arbiter 或 Engine 语义。
- 只读 OpenProject/GetConfig/GetReview/GetUsage/Export Preview/Import 状态不创建 Host。
- 只有用户明确触发、且 Core 需要 Host 的 Run/Sync/Import/Co-create/Next/Steer 等操作按需创建 Host。
- 不增加第二套 Provider 配置格式、项目模板、章节识别算法或 TXT/EPUB renderer。
- 不扩展 Fact Engine、Proposal、Knowledge、Impact Analysis、Repair/Replan、云同步、多人协作等功能。

## M7 Goal

在 M6 冻结基线上完成 Novel Studio V1 的回归、生命周期/并发/恢复加固和发布准备。GUI 继续复用 Core/Host/Store，不能建立第二套小说事实、配置语义或 Engine 状态；发现真实回归直接修复，不等待新的 milestone 确认。

## M7 Phases

### Phase 7: M7 源码审计与事实源核对

- [x] 逐项核对 M2-M6 Studio API 到 Core/Host/Store 的真实调用链
- [x] 扫描 React 是否直接读写小说文件或 config JSON，扫描 GUI 自行推导的 Core 事实
- [x] 核对 Host.New 只读页面边界、ProjectRoot/OutputDir、projectId/generation/sequence/requestId
- [x] 核对 terminal-before-ack、Core success/view refresh failure、stale project/event、secret 泄漏
- [x] 用 GitNexus impact/query/context 和源码搜索补证 UNKNOWN/动态边界
- **Status:** complete（动态 Wails/跨语言边界以源码、测试和构建补证；GitNexus 仍保留 UNKNOWN 下界）

### Phase 8: CLI/GUI 回归与生命周期加固

- [ ] 对照 Quick Start、Co-create、Outline、连续生成、Pause/Resume、Stop、Review/Next/Steer、编辑/Sync
- [ ] 对照 Import、TXT/EPUB Export、Provider/Model/Budget 的 Store/Config 最终状态
- [ ] 覆盖 crash recovery、pending gates、项目切换 stale response/event 和前端操作竞态
- [ ] 发现真实回归后以最小范围补测试并修复
- **Status:** in_progress（已完成静态回归与定向修复；真实模型/GUI 全矩阵仍需现场运行）

### Phase 9: 跨进程写入与项目切换 Hardening

- [ ] 验证并修正 GUI 与 CLI 同时写同一项目时的 Core/book lease 互斥
- [ ] 覆盖 Save、Sync、Run、Import、Next、Steer/Continue、Create/Publish mutation
- [ ] 覆盖 Idle/Paused/Running/Pausing/Stopping/WaitingSync/Syncing/WaitingReview/Importing/Co-create/Steering 项目切换
- **Status:** in_progress（已复用 Core book lease 覆盖 Save/Budget 与项目切换保护；完整 CLI/GUI 跨进程矩阵需现场运行）

### Phase 10: 长篇、Windows 文件系统与发布验证

- [ ] 对 100/300/500/1000 章项目检查树、切章、Review、Revision、Runtime logs、Usage
- [ ] 验证中文/空格/长路径、不同盘符、只读/权限/临时 rename 失败和 secret 脱敏
- [x] 运行 Go、Frontend、Wails Windows production CI 等价验证
- [x] 生成 `docs/V1_RELEASE_CHECKLIST.md` 与 `docs/V1_KNOWN_LIMITATIONS.md`
- **Status:** in_progress（Wails/CI 等价验证及发布文档已完成；100/300/500/1000 章现场压力矩阵待运行）

## M7 Definition of Done

- [ ] CLI/GUI 使用同一 Core 且主要日常工作流最终 Store/Config 状态一致
- [ ] 生命周期、Save/Sync、Review/Next/Steer、Import/Export、Config/Model/Budget 可恢复且不绕过 Core gates
- [ ] 跨进程写入不破坏项目，项目切换不受 stale event/response 污染
- [x] 无完整 API Key 泄漏路径，Windows production build 已通过，仓库 CI 已覆盖 Go/Frontend/Wails
- [x] V1 release checklist 与 known limitations 已创建；新能力进入 V2

### M7 当前边界

- 本轮已完成源码审计、可复现的 Go/Frontend/Wails 验证和最小范围 hardening；真实模型调用、完整 GUI 手工矩阵以及 100/300/500/1000 章 Windows 现场压力仍属于发布前人工验收，不以静态测试替代。

## Errors Encountered

| Error | Attempt | Resolution |
|---|---:|---|
| `rtk proxy Get-Content` 无法解析 PowerShell cmdlet | 1 | 改用 `rtk pwsh -NoProfile -Command` 读取技能文件 |
| 从仓库根目录运行 Wails 找不到 `wails.json` | 1 | 按现有工程结构改在 `cmd/novel-studio` 目录运行，Windows production build 通过 |

## V2-M1 — Proposal + Revision Workspace

### Goal

按 V2 产品架构蓝图实施第一闭环：ReviewIssue → Proposal → Diff/Evidence → Accept → Apply → SavedUnsynced → Core Sync → Synced。Core/Store 是唯一小说事实源；Proposal 和 Version metadata 不能直接更新 ChapterRecord 或 accepted revision。

### Baseline

- 起始分支：`codex/m7-hardening`，起始提交：`0f0fe4b`，起始工作树干净。
- 用户已同意先落审计计划并连续实施 V2-M1。
- 本轮不开始 Fact Engine UI；候选正文先由用户手工提供，不新增模型生成调用链。
- 复用 Core projectwrite、book lease、Engine lifecycle/exclusive、原子文件写入和既有 Save/Sync 语义。

### Phases

1. [x] 建立 V2-M1 分支与计划基线，刷新 GitNexus。
2. [x] 完成蓝图列出的 13 项源码审计，记录调用链、风险、缺口和实施选择到 `docs/studio-v2-m1-audit.md`。
3. [x] 完成 Proposal 持久化模型/存储/服务：ReviewIssue 来源键、Evidence hash 绑定、状态迁移、原子操作记录和 schema_version。
4. [x] 实现 Diff、Stale 检查、Accept/Reject/Apply/Restore 边界；Apply 只走现有章节 Save，绝不触碰 ChapterRecord。
5. [x] 接入 Bridge 和前端 Proposal Inbox/Detail，异步结果按 projectId/generation/requestId 隔离。
6. [ ] 对照 M1 DoD 覆盖 stale、手工编辑、多变更前置条件失败、Sync 失败、Apply 后崩溃恢复和项目切换；执行 Go/Frontend/vet/build 验证。
7. [ ] GitNexus `detect-changes --scope all`，复核 diff、验证工作树；不提交/推送/建 PR，除非后续明确要求。

### Constraints

- Proposal Accepted != Working Copy Applied != Core Synced。
- Proposal Apply 和 Version Restore 都通过 V1 Save 写工作正文并显示 SavedUnsynced；只有 Core Sync 与 ChapterRecord hash 复核后标记 Synced。
- BaseHash 不匹配时标记 Stale 并拒绝覆盖；不做自动三方合并。
- Evidence 必须绑定资源、章节、revision/hash 与正文区间；ReviewIssue 当前无稳定 ID，M1 为其生成确定性来源键。
- 暂不扩展 Outline/Character/World/Arc/Volume mutation、Fact Engine、AI semantic diff 或自动 Repair/Replan。
- 所有用户可见文案、注释和提交说明使用中文。

### Errors Encountered

| Error | Attempt | Resolution |
|---|---:|---|

## V2-M1 review 修复收束
- [x] 处理只读审查 Critical：Apply pplied journal 崩溃恢复。
- [x] 处理 Important：重复资源、Restore 陈旧与日志、前端项目 generation、切换 busy gate、Bridge metadata lease。
- [x] 处理边界：metadata ID、Evidence 引文、超大 Diff、Review SourceKey 重复。
- [ ] 提交、推送并创建 PR；等待 CI。

## V2-M1 后置、M2 前置：模型配置增强

### Goal

在进入 M2 前补齐可实际使用的多服务商模型配置入口。图片仅作为交互参考；事实源仍为 ainovel Core 配置与 Host ModelSet，不新增前端配置格式。

### Scope

- [x] 允许 Studio 新建 Provider 配置，并编辑协议、API endpoint、Base URL、API Key 和模型列表。
- [x] 暴露每个模型的上下文窗口与 JSON Schema 能力提示，保存继续复用 `Host.ConfigureModels`。
- [x] 增加只读模型列表发现入口，支持 OpenAI-compatible 与 Gemini；发现结果只有用户保存后才进入 Core 配置。
- [x] 增加连接测试入口，复用现有 Core `TestModelConnection`，不保存草稿。
- [x] API Key 只进受信任 Bridge/Host，快照和错误继续脱敏。
- [x] 补 Core/Bridge/Frontend 回归测试；不扩展 M2 的 Proposal、Fact、Knowledge 或 Repair 语义。

### Fact source and boundaries

- Provider/Model 持久化：现有项目配置文件与 `Host.ConfigureModels`。
- 当前角色模型：现有 `SwitchModel` / `SetRoleThinking`。
- 模型发现：只读网络请求；返回值是待保存草稿，不成为第二套配置事实。
- 保存/测试服从当前项目、`projectMu`、book lease 与 Engine 配置互斥边界；发现只校验当前项目已打开，保持只读且不创建 Host。

### Session 2026-09-26

- 实现 `ModelSettingsEditor`、OpenAI-compatible/Gemini 模型发现和 Bridge 委托。
- 保存失败继续向编辑器传播，避免前端显示伪成功；补 Go 与前端回归测试。
- 本地验证：Go 1044、前端 47；`go vet`、前端 production build、Wails Windows production build 均通过。

## Session 2026-09-26 — Provider 目录与已保存 Key 发现修正

- 增加 OpenAI、NovelAI、DeepSeek、Google Gemini、xAI、SiliconFlow、Ollama、BigModel 和自定义预设；预设只填充 UI 草稿，不新增配置事实源。
- 已保存 Provider 的模型发现由 Bridge 在受信任边界补回 Core Key，修复空 Key 导致的发现失败；Key 不返回前端。
- 补 Provider Key 解析测试；前端模型发现仍统一使用 OpenAI-compatible `/models` 或 Gemini `/v1beta/models`。

## V2-M2 审计暂停记录（Deferred Research，2026-09-26）

- 已从最新 `origin/main`（包含 M1 与模型配置 PR）完成源码审计，结果见 `docs/V2_M2_FACT_AUDIT.md`。
- 已确认 Core `ChapterRecord.Facts`、`revision.Service.Sync`、`revision.Projector` 是 M2 的事实源与 Sync 后重建边界。
- 已确认当前没有 Fact Registry、Character Knowledge、Fact Conflict 或 Explorer 实现。
- 本轮按用户要求停在审计阶段，未保留任何 M2 代码实现、Bridge API、前端 store/UI 或 schema 迁移。
- 后续若重新评估原 V2-M2，只能把该审计作为历史研究参考，不构成未来实现约束。任何重新立项必须在 Core GUI Parity 完成后，基于真实产品缺口和最新源码重新审计，并重新定义事实源、持久化、generation、warning、项目 scope 与恢复语义。

## Core GUI Parity 审计基线（2026-09-26）

- 当前正式路线已从原 V2-M2 切换为 Core GUI Parity Completion；产品约束见 `docs/PRODUCT_GOAL_CHARTER.md`，阶段计划见 `docs/CORE_GUI_PARITY_COMPLETION_PLAN.md`。
- 已从当前 `origin/main`（`6cb873a`）完成 CLI/TUI command registry、README 作者流程、Host/Store/Config、Studio Bridge/ViewModel 和 React 页面核对；结果见 `docs/CORE_GUI_PARITY_MATRIX.md`。
- 本轮按用户要求**停在审计阶段**：不实施 Wave A，不新增 Story/Continuity Bridge、ViewModel、前端 Store/UI，不改变 Core 语义和持久化。
- 审计结论：Story/Continuity 所需的 premise、characters、world rules、outline、Compass、summaries、timeline、foreshadow、relationships、state changes、snapshots 与 cast projection 已存在于 Core Store/ChapterRecord 投影；Studio 主要缺少只读 Facade/ViewModel/GUI 入口。
- 已知 parity 缺口：`/reopen`、`/simulate`、`/importsim`、`/diag`/diag-export、Rules/Style/Voice 管理、Config advanced/fallback/notify、Import auto-confirm/story-resolution、Runtime context/compression 观测等，详见矩阵。
- 后续若开始 Wave A，必须先重新运行相关符号的 GitNexus impact；若实现中发现 Core 语义缺失、第二事实源或持久化迁移需求，立即暂停并汇报。

## Wave A — Core GUI Parity Completion（2026-09-26）

### Goal

从最新 `origin/main` 完成 Story Data Center 与 Continuity Center 的只读 GUI 闭环。链路固定为 React → Wails Bridge → Studio Facade/ViewModel → Core Store；只读读取不创建 Host、不写回 metadata、不创建第二套小说事实源。

### Scope

- [x] Story Data Center：Premise、Characters、World Rules、Flat Outline、Layered Volume/Arc/Chapter Outline、Story Compass、Chapter/Arc/Volume Summaries。
- [x] Continuity Center：Timeline、Foreshadow Ledger、Relationships、State Changes、Character Snapshots、Cast/First Appearance。
- [x] React 导航、分页/惰性详情、loading/error/empty 和项目切换 stale protection。
- [x] Go Facade/ViewModel/Bridge 与真实 Core Store 读取测试。
- [x] Frontend 导航、数据展示、加载错误、空数据、项目切换和 stale response 测试。
- [x] 更新 `docs/CORE_GUI_PARITY_MATRIX.md` 的 Wave A 状态；不进入 Wave B/C、Deferred Research 或 Fact Engine。

### Guardrails

- 不实现 `/simulate`、`/importsim`、Rules/Style/Voice、`/diag`、Advanced Config、Import option parity、Runtime parity、`/reopen`、Fact/Knowledge/Dependency/Impact/Repair。
- 不让 React 解析 JSON/JSONL/Markdown；不让只读页面创建 Host。
- 数据缺失时返回明确的 optional missing/empty 状态，不在 GUI 推导关系、状态、时间线或事实。
- 所有读取请求携带并校验 projectId、generation、requestId、sequence；项目切换后旧响应不得写入前端。

### Audit

- 基线：`origin/main` merge commit `a463521`。
- GitNexus 已重建；`OutlineStore.LoadOutline` 影响为 CRITICAL（lower-bound，74 下游、6 流程），`LoadPremise` HIGH（39 下游），`LoadLayeredOutline`/World projection 共享高影响边界；本轮只新增读取适配，不修改这些 Core 符号。
- `CharacterStore.LoadSnapshots` 与 `SummaryStore.LoadSummary` 的索引符号存在解析下界，需以源码搜索、Facade 测试和全量验证补证。

### Phases

1. [x] 从最新 `origin/main` 建立 `codex/wave-a-story-continuity` 并完成文档/源码轻量审计与 GitNexus impact。
2. [x] 建立 Story/Continuity ViewModel、只读 Facade 和 Bridge API。
3. [x] 建立前端 Store、导航和 Story/Continuity 页面，接入项目身份与 stale protection。
4. [x] 补齐 Go/Frontend 测试与大列表分页/惰性详情覆盖。
5. [x] 运行 Go tests、vet、Frontend tests/build、Windows Wails production build。
6. [x] 重跑 GitNexus `detect-changes --scope all`，复核 Wave A 矩阵与文档范围。
7. [ ] 提交中文说明、推送并创建统一 PR；CI 全绿后报告结果。

### Errors Encountered

| Error | Attempt | Resolution |
|---|---:|---|
| 旧 GitNexus 索引无法解析目标方法 | 1 | 按项目规则重建 `.gitnexus` 索引后，以完整 Method ID 重跑 impact；UNKNOWN 仅作为索引边界处理 |

### Implementation notes

- Story/Continuity 只读 API 均经过 Bridge project mutex、当前 `ProjectRoot`/`OutputDir` 和 generation 校验；只读调用不创建 Host。
- 前端同时暴露分层与扁平大纲，卷/弧章节和摘要/连续性列表按页读取；React 不读取项目文件。
- Cast 继续使用 Core `Store.BuildCast` 的已接纳 ChapterRecord 投影；没有从正文或 UI 推导新事实。
- `ConfirmChapterCommit` 与 Core Sync 刷新返回的 Project 补回当前 Bridge generation，避免后续只读请求丢失作用域身份。
