# Novel Studio V2-M1 源码审计与实施边界

## 审计基线

- 审计日期：2026-09-25
- 分支：`codex/v2-m1-proposals`
- V1 基线提交：`0f0fe4b`
- 需求文档：`C:\Users\linn\Downloads\Novel_Studio_V2_Product_Architecture_Blueprint.md`
- 第一闭环：`ReviewIssue → Proposal → Diff/Evidence → Accept → Apply → SavedUnsynced → Core Sync → Synced`

本轮只实现 V2-M1。Proposal、Evidence、Diff、Version 是治理层元数据，不能成为第二套章节事实源；Core Store 的章节正文、ChapterRecord、Revision 和 Sync 结果仍是唯一业务事实源。

## 13 项源码审计

| # | 审计项 | 当前真实能力 | M1 实施决定 |
|---:|---|---|---|
| 1 | ReviewIssue → Chapter 映射 | `internal/domain/review.go` 的 `ConsistencyIssue` 只带 `Chapters`；`save_review` 会按 scope 校验 chapter/global/arc 边界，并从 `RequiresChange` 派生 affected chapters。Review Center 由 `internal/studio/app/review_service.go` 读取 Core ReviewEntry。 | 以 `ReviewEntry + issue index + issue 字段哈希` 生成确定性来源键；M1 的 Review 入口先绑定审阅截止章节，涉及多章的 issue 不用同一候选正文盲写，转由手工 `CreateProposal` 分别提供多 change。 |
| 2 | Rewrite / Polish 调用链 | `save_review` 将 verdict 与 `PendingRewrites` 写入 Core Progress，`internal/flow/router.go` 再调度 Writer；`commit_chapter` 负责返工提交。 | Proposal Apply 不复用自动返工链，不触碰 PendingRewrites；Proposal 是人审候选正文，最后仍调用 V1 章节 Save。 |
| 3 | Chapter Save / Sync 入口 | `internal/studio/app/project.go` 的 `Service.SaveChapter` 写 Drafts 工作正文；`internal/studio/bridge/app.go` 的 `App.SaveChapter` 负责 project mutex、book lease、revision check。`App.SyncChapterRevisions` 调用既有 Engine/Host Sync 并重新检查 Revision。 | Proposal Apply 和 Version Restore 都经同一 `Service.SaveChapter` 适配；Sync 只调用现有 `App.SyncChapterRevisions`。不增加新的 accepted 写入口。 |
| 4 | ChapterRecord hash / revision | `internal/domain/revision.go` 的 `ChapterRecord` 保存 accepted content、revision、`ContentSHA256`；`ChapterContentSHA256` 统一 BOM/newline 后计算 SHA-256。 | ProposalChange 保存 BaseHash/Before/AfterHash；Apply 前比较当前工作正文 hash，Sync 后用 RevisionStatus 与 Core accepted hash 复核。 |
| 5 | 历史版本/快照 | 现有 checkpoint 只记录运行过程；没有可供 Studio 使用的章节完整内容快照或 restore API。 | 新增 `<OutputDir>/meta/studio-v2/versions/`，只保存去重后的正文快照；Restore 写回工作正文并保持 SavedUnsynced。 |
| 6 | Diff 工具 | Core 没有 Studio 章节 Diff API；现有 revision scan 不是用户可见的文本 diff。 | M1 使用本地确定性 line diff/unified diff，不调用 LLM，不把 Diff 当事实源。 |
| 7 | metadata extension | `internal/store/store.go` 的 Core `meta/format.json` 当前为 V3；各 Core 子 Store 有既定目录。 | V2 使用独立 `meta/studio-v2/schema_version.json` 与 proposals/versions/operations 子目录，不修改 Core format version，不参与 V1 migration。 |
| 8 | Sync success 观测点 | `App.SyncChapterRevisions` 在 Host Sync 返回后读取 `RevisionStatus`；只有 `RevisionSynced` 且无 unsynced 才报告成功，视图刷新失败作为 warning 分离。 | Proposal 不在 Apply 时标记 Synced；Sync 成功后按 change 的 chapter/hash 与 RevisionStatus 对账，只有完全匹配才进入 Synced。 |
| 9 | Proposal 已被 Core accepted 的判断 | Core 没有 Proposal 概念；可观测事实是章节 accepted `ContentSHA256`、revision 与 `HasUnsynced=false`。 | 使用全量 change 的 accepted hash 对账；对账前状态只能是 SyncPending/AppliedWorkingCopy。 |
| 10 | project upgrade / migration | Core `CurrentProjectFormatVersion=3`，升级由 Store/Migration 负责。V2 metadata 不在 Core upgrade 中。 | V2 首次写入创建 schema_version=1；未知/损坏 V2 metadata 返回错误，不修改 Core 文件；后续以独立 V2 migration 扩展。 |
| 11 | atomic write | `internal/store/io.go` 的 JSON 写入为 temp + fsync + close + rename。该 IO 未对外暴露。 | V2 metadata 使用等价的本地 temp + fsync + rename；章节正文仍只使用 Core Drafts Save，不复制 Core 写语义。 |
| 12 | projectwrite / book lease | Core 通过 project write 与 `.ainovel.lock` book lease 保护写入；Bridge 的 Save/Sync 已有复用既有 Engine lease 的路径。 | Bridge Apply/Restore 复用 `withProjectBookLease` 与 `projectMu`；V2 不增加第二套 lock file。多 change 先全部做 precondition，再逐章走 Core Save。 |
| 13 | crash recovery 边界 | Core revision 有 `pending_revision.json`，可恢复 records/projections 阶段；V2 尚无 operation journal。 | 新增 Proposal operation journal。Apply 前记录 planned/current hashes；重启读取 journal：全量 AfterHash 命中则恢复为 AppliedWorkingCopy/SyncPending，部分命中标 Failed 并禁止静默覆盖；Sync 仍由用户显式触发。 |

## GitNexus 影响与保守边界

审计前已重建 GitNexus。以下 impact 结果在当前动态 Go/Wails 边界下按下界解释：

| 目标 | 风险 | 结论 |
|---|---|---|
| `SaveChapter`（`internal/studio/app/project.go`） | `UNKNOWN` | 不能按无调用处理；源码确认 Bridge 直接调用。只新增适配调用，不改其语义。 |
| `SaveFinalChapter`（`internal/store/drafts.go`） | `HIGH` | 直接调用方包含 Studio Save、commit、migration、import；不改签名、不改实现。 |
| `SyncChapterRevisions`（`internal/host/host.go`） | `LOW`（精确结果） | 只由现有 Host/Engine 服务调用；V2 复用，不改 Host Sync。 |
| `ChapterRecord`（`internal/domain/revision.go`） | `HIGH` | 11 个受影响点；不增字段、不直接写入。 |
| React `App`（`desktop/frontend/src/App.tsx`） | `UNKNOWN` | Wails/TS 动态绑定无法完整解析；仅添加导航和组件，保持已有页面投影。 |

若后续必须修改上述 HIGH/UNKNOWN 符号，应先重新执行针对目标的 impact，并补定向回归；本轮实现优先新增 V2 包、ViewModel 和 Bridge 薄适配层。

## M1 数据与安全决策

### Proposal 来源

- `ReviewIssue`：保存 `scope/chapter/issueIndex/sourceKey`，sourceKey 对 issue 字段做确定性哈希，避免当前 ReviewIssue 没有稳定 ID。
- `ManualRequest`：允许用户在 Proposal Inbox 输入候选正文，第一版不创建模型调用链。

### BaseHash 与证据

- 创建时从 Core Drafts 当前正文取得 `Before` 与 BaseHash；若不存在工作副本则读取 accepted ChapterRecord 内容。
- Evidence 保存 ResourceType、ResourceID、Chapter、Revision、ContentHash、StartOffset、EndOffset、QuotePreview；区间只引用创建时正文。
- Apply 前再次读取 Drafts 正文并比较 BaseHash。变化即 Stale，禁止强制覆盖，不做自动三方合并。

### 状态与恢复

`Draft → Ready → Accepted → AppliedWorkingCopy → SyncPending → Synced` 是正常路径；Rejected、Stale、Failed 为终态/需人工处理分支。Accept 不写正文；Apply 不写 ChapterRecord；Restore 不写 accepted hash。

Apply 的 operation journal 与 Proposal 状态是可恢复元数据，不代替 Core revision。崩溃后最多恢复为等待 Sync，绝不在没有 Core Sync 证据时伪造 Synced。

## 缺口与本轮不做

- 不新增 Fact、Dependency、Impact、Repair、Replan、AI semantic diff 或批量自动修改。
- 不为 Outline/Character/World/Arc/Volume 创建未经 Core 审计的 mutation API。
- 不把 V2 derived metadata 纳入 Core `meta/format.json` 或 accepted revision。
- 真实 API/模型连续运行、跨进程 CLI/GUI 写入、崩溃现场、长篇 Windows 文件系统压力仍需最终现场验收；自动化验证不能替代这些证据。
