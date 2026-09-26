# V2-M2 Fact Engine 审计（Deferred Research）

> 路线状态（2026-09-26）：Deferred Research。本文保留为源码研究记录，不构成 Fact Engine、Character Knowledge 或 Conflict 功能的实施承诺。当前正式路线以 [产品目标章程](PRODUCT_GOAL_CHARTER.md) 和 [Core GUI Parity 完成计划](CORE_GUI_PARITY_COMPLETION_PLAN.md) 为准；不回滚 V2-M1，也不继续原 V2-M2 实现。

## 审计范围

本审计基于 `origin/main` 合并 M1 与模型配置后的源码，目标是确认 V2-M2 的事实源、Sync 边界、持久化与 UI 接入点。V2-M2 规格来自用户粘贴的需求文档；本文记录仓库当时的真实能力与历史研究边界。

## 当前真实事实源

### ChapterRecord.Facts

`internal/domain/revision.go` 的 `ChapterFacts` 是已接纳章节的结构化事实，字段为：

- `Title`、`Summary`、`Characters`、`KeyEvents`
- `TimelineEvents`
- `ForeshadowUpdates`
- `RelationshipChanges`
- `StateChanges`
- `CastIntros`
- `HookType`、`DominantStrand`、`Feedback`

`ChapterRecord` 同时携带 `Revision`、`ContentSHA256`、`Content`、`AcceptedAt`。M2 只能读取已接纳 `ChapterRecord`；工作正文和 `PendingRevision` 不得进入 canonical facts。

### Sync 与现有投影

`internal/revision/service.go` 的 `Service.Sync` 扫描工作正文变化，调用模型分析生成新的 `ChapterRecord`，随后在 `applyPending` 中：

1. 写入接纳记录；
2. `revision.Projector.Apply` 从完整已接纳记录集重建摘要、时间线、伏笔账本、人物关系、状态变化、进度和作者修订风格；
3. 失效章节聚合；
4. 清理 pending revision。

因此 M2 的 reconcile 必须发生在 Core Sync 成功并重新读取 Store 之后。Fact reconcile 失败只能作为 warning 返回，不得把 Core Sync 变成失败。

### 可复用投影

- 时间线：`WorldStore.LoadTimeline`，源于 `ChapterFacts.TimelineEvents`。
- 伏笔：`WorldStore.LoadForeshadowLedger`，由 `ForeshadowUpdates` 确定性重放。
- 关系：`WorldStore.LoadRelationships`，由 `RelationshipChanges` 按人物对重放。
- 角色状态：`WorldStore.LoadStateChanges`，源于 `StateChanges`。
- 章节摘要：`SummaryStore.LoadSummary`，源于 Title/Summary/Characters/KeyEvents。
- 首次出场与配角：`domain.ProjectCast` 从 Characters/CastIntros 计算；主角/核心角色来自 `CharacterStore`。
- 世界规则：`WorldStore.LoadWorldRules` 是 Architect 配置事实，但没有章节证据绑定；历史研究只识别出读取型基础事实映射的可能性，没有虚构章节证据。
- Review 一致性问题：保存在 `reviews/*.json`，没有稳定 Fact ID；历史研究没有把 ReviewIssue 自动当作 canonical Fact，也没有据此批准 Conflict→Proposal 实现。

### 当前不存在的能力

源码中没有 Fact、CharacterKnowledge、FactConflict 的 canonical 类型、索引目录、查询 API、Sync 后重建入口或前端 store。现有投影不能直接作为 M2 API 返回，因为它们没有统一 ID、状态、证据和 source revision/hash。

## 原拟议设计（历史研究，未获实施批准）

以下章节记录原 V2-M2 设计讨论中的假设方案，仅供历史研究和未来重新立项时参考，不代表当前路线、已批准实现或未来实现约束。未来若重新立项，必须先完成 Core GUI Parity，并基于届时真实产品缺口和最新源码重新审计，重新确认事实源、持久化、互斥边界、恢复语义与验收范围。

### 原拟议事实模型与来源边界（历史假设）

原拟议方案曾设想新增 `internal/studio/v2/facts` 领域/存储服务，目录固定为 `meta/studio-v2/facts/`，包含 schema、generation manifest、facts、knowledge、conflicts 三类 JSON 数据。该设想定位为 Core Accepted State 的可删除派生索引，而不是第二套小说事实源；本审计未实施这些内容。

原草案曾设想只映射当时 Core 能可靠提供的类型：

- `ChapterSummary`：标题、摘要、关键事件；
- `CharacterState`：`StateChanges`；
- `CharacterRelationship`：`RelationshipChanges`；
- `TimelineEvent`：`TimelineEvents`；
- `Foreshadowing`：`ForeshadowUpdates` 重放得到的伏笔生命周期；
- `CharacterAttribute`：`CastIntros` 与角色首次出现；
- `PlotCommitment`：仅在可由章节事实明确表示时生成，不用模型臆测；
- `WorldRule`、`LocationFact`、`ObjectFact` 在 Core 没有章节证据时不伪造，保留扩展枚举但不会生成空证据记录。

原草案曾设想每个 Fact 保存 `ProjectID`、稳定 ID、类型、subject/predicate/object、state、`EvidenceRef`、source chapter/revision/hash、创建/更新时间。Evidence 复用 M1 的资源/章节/revision/content hash 结构；ChapterRecord 没有精确偏移时不猜 offset，并明确 `ExactRangeAvailable=false`。

原草案曾设想稳定 ID 使用规范化后的 type、subject、predicate、object 与 source chapter/semantic key 哈希；同一事实内容变化时保持 Fact ID，新增证据不创建重复事实。

### 原拟议 Character Knowledge 与 Conflict（历史假设）

原草案曾设想 CharacterKnowledge 由 accepted facts 按章节顺序确定性生成，记录角色、Fact、`LearnedAtChapter`、`LastChangedAtChapter`、Evidence、SourceRevision 和 Known/Believed/Suspected/Unknown/Disproved 状态。该假设不把“角色在章节中出现”直接升级为 Known；缺少可靠知识边界时使用 Suspected/Unknown 并保留来源。

原草案曾设想 FactConflict 采用确定性规则：同一主体/谓词在同一有效范围出现不同对象、时间事件顺序冲突、关系或状态互斥、角色知识早于首次证据等。Conflict 引用 Fact ID 与证据，状态为 Open/Dismissed/Resolved/Stale。CreateProposalFromFactConflict 只创建 `ProposalSourceFactConflict` 提案，不自动 Apply；这些内容均未获本轮实施批准。

### 原拟议持久化、原子发布与恢复（历史假设）

- 原草案曾设想使用 `meta/studio-v2/facts/`，不复制章节正文。
- 原草案曾设想重建流程读取完整 accepted ChapterRecord 集，生成临时 generation，校验 schema/项目作用域/证据 hash/稳定 ID，再原子替换 manifest 与数据文件。
- 原草案曾设想失败时保留上一份完整 generation；残留临时目录和未完成 manifest 在下次读取时丢弃。
- 原草案曾设想删除/重建在相同输入下得到稳定 ID 与稳定排序。
- 原草案曾设想只在已有 projectMu + book lease 的短临界区读取/发布 metadata，不持有 Core projectwrite 进行模型或长时间计算。

### 原拟议 Bridge 与前端边界（历史假设）

原草案曾设想 Bridge 从当前 `outputDir` 创建 Manager，并强制注入当前项目 ID；请求中的 ProjectID 只用于竞态校验。所有 Fact/Knowledge/Conflict API 只读或显式 Rebuild，返回 projectId、generation、operationId；旧项目响应不得更新当前前端 store。

原草案曾设想前端新增独立 `factStore`、`knowledgeStore`、`conflictStore`，拥有 loading/rebuilding/error/selected/filters/generation/operationId 字段。Explorer 只展示 Core 派生事实和 warning，不在前端推导事实状态。

### 原拟议实施顺序与测试矩阵（历史草案）

1. 原草案曾设想新增领域模型、规范化 ID、Evidence 转换、确定性 conflict/knowledge builder。
2. 原草案曾设想新增 generation 存储、manifest、atomic publish、恢复与删除重建。
3. 原草案曾设想 Bridge 暴露 List/Get/Query/Rebuild/Stats 与 CharacterKnowledge/Conflict API；Sync 成功后 reconcile 并返回 warning。
4. 原草案曾设想新增 Fact Explorer、Character Knowledge、Conflict 页面/面板和 Conflict→Proposal 操作。
5. 原草案曾设想补 Go 测试：unsynced 不变、sync 更新、历史重建、稳定 ID、evidence hash、删除重建、失败保留旧索引、knowledge boundary、conflict/dismiss/stale、proposal source、项目作用域、sync success+warning。
6. 原草案曾设想补前端测试：筛选、分页、项目切换 stale、重建失败、generation reset。
7. 原草案曾设想运行 Go/Frontend/vet/build/Wails 与 GitNexus detect-changes；静态验证与真实模型/GUI 验收仍需分开报告。

## 明确不做

本次 Deferred Research 未改动 ChapterRecord、Revision Sync、Writer Prompt/Context，也未实施 Dependency Graph、Impact Analysis、Repair/Replan、Graph DB、Cloud Sync、协作、插件或 Workflow Builder。未来若重新立项，以上范围必须随最新源码和真实缺口重新确认。
