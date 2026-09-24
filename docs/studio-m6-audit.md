# Novel Studio V1 — M6 源码审计

审计基线：`origin/main` / `72add9b`（M5 已合并）。本文只记录现有 Core 事实和 Studio 适配边界，不引入第二套项目、配置、导入或导出语义。

## 结论摘要

M6 所需的 Quick Start、冷启动/阶段 Co-create、Import pipeline、TXT/EPUB Export、Provider/Model 配置和 Usage/Budget 事实均已存在于 Core。Studio 缺少的是桥接、长任务生命周期和只读 ViewModel，不需要重写 Core 算法。

唯一需要特别处理的是两处适配缺口：

1. `Host.New` 的 `configPath` 目前通过进程 cwd 的 `EffectiveConfigPath()` 计算。Studio 已经用 `LoadConfigFromDir(projectRoot)` 正确读取配置，但 Host 后续保存模型/推理设置时仍可能写错层。M6 增加显式配置路径注入，只改变宿主路径解析，不改变配置合并或 Core 业务语义。
2. Co-create 日志已有 `meta/sessions/cocreate.jsonl`，但 Core 没有公开恢复读取器或 unfinished 标记。Studio 只复用现有日志格式做恢复投影，不新增持久化结构；进行中的单次 LLM 请求无法在进程崩溃后伪造为已完成，恢复以最后一个已落盘回合为准。

## 真实调用链

### Quick Start / Start from Outline

CLI/headless 与 TUI 都走：

```text
prompt 或 /start <文件>
  → startup.PrepareQuick / startup.LoadPromptFile
  → Host.New
  → Host.PrepareUserRules
  → Host.StartPrepared
  → Arbiter.DecidePlanStart
  → RunMeta.SetStartPrompt / SetPlanStart
  → startEngine(planner instruction)
  → Architect / Engine 继续写作
```

`Host.New` 顺序是 book lease → `Store.Init` → project upgrade → `RunMeta.Init` → ModelSet/Usage/Budget/Engine。`StartPrepared` 会写入起始 prompt、规则快照和计划裁定；失败可能留下可恢复的 RunMeta/规则事实，但不会伪造完成状态。现有 `/start` 接受文本文件内容，文件本身不经过专门的 outline parser；因此“从大纲开始”必须继续复用 `LoadPromptFile → PrepareQuick → StartPrepared`。

### Co-create

冷启动：`Host.CoCreateStream → coCreateStream → models.ForRole("thinking") → GenerateStream`。每轮的用户 history、原始模型输出、thinking、解析后的 reply/draft/ready/suggestions 追加到 `<OutputDir>/meta/sessions/cocreate.jsonl`。

阶段共创：`PauseForCoCreate` 设置 `cocreating`，运行中通过 `abortWithEvent` 等待 Engine 收敛；`StageCoCreateStream` 把进度、罗盘、最近卷摘要、主要角色和活跃伏笔组成系统提示；结束时 `ResumeFromCoCreate` 将 `[阶段规划]` brief 交给 `Host.Continue`，由 Arbiter 决定如何落地后恢复 Engine。共创态会阻塞 Resume/Continue/Import 等入口。

### Import

```text
Host.ImportFrom
  → budget.Refuse
  → acquireExclusive("导入") + Store project write
  → imp.Run
  → NextAction(LoadState facts)
  → ingest → segment → await_confirmation → analyze → synthesize
  → await_story_resolution → publish
  → publishFoundation + chapters + checkpoints + AdvanceHold
  → superviseImport / ContinueAfterImport
```

支持 UTF-8、UTF-8 BOM、GB18030 的文本源；当前入口按可解码文本处理，不存在 DOCX/PDF/HTML 导入语义。切分结果包含 chapter number/title/source span/uncertain/notes/matter；用户确认通过 `confirmation.json`，`--guide` 写入 guidance 并使下游 digest 失效。各阶段以 `meta/import` 工件和 digest 恢复，`NextAction` 是唯一阶段事实源。模型只在 segment/analyze/synthesize 三阶段调用。GUI 必须消费真实 `imp.Event`，不能自行维护百分比或重写 split。

### Export

`exp.Run(ctx, exp.Deps{Store}, opts)` 是纯本地只读操作，支持 `txt` 与 `epub`、全量或闭区间 `From/To`、未完成章节 skipped、默认目标路径和 `Overwrite`。它读取 Progress/Book/Outline/Summaries/章节终稿，原子写目标文件，不写 Project Store；范围非法、没有完成章节、文件已存在等错误原样返回。

### Provider / Model / Budget / Usage

- 配置读取：全局 `%USERPROFILE%/.ainovel/config.json`，再合并 `ProjectRoot/.ainovel/config.json`；项目层按 key 覆盖全局层。
- Provider 字段：`type`、OpenAI 协议 `api`、`api_key`、`base_url`、models、extra body/extra；模型有 name/context window/json schema。
- 角色字段：default、architect、writer、editor、import_segment、import_analyze、import_synthesize 可独立指定 provider/model/reasoning/fallback；Arbiter 当前复用 default 模型，不是独立配置角色。
- `Host.ModelConfiguration` 只给脱敏快照；`Host.ConfigureModels` 负责校验、引用保护、原子写入、live ModelSet apply。`SwitchModel` / `SetRoleThinking` 保存到当前有效配置层并让后续 Agent 调用使用新选择，不自动重启当前 Engine。
- Budget：`BudgetSentinel` 在 `Host.New` 依据 `Budget.BookUSD/WarnRatio/HardStop` 创建；Start/Resume/Continue/Import 前拒绝已越线，成本记账时发 warn/hard stop。修改 JSON 后不会重建现有 sentinel；Studio 保存后重新读取 effective config，并标注预算政策在新 Host/下一次 Session 生效。
- Usage：`UsageTracker.Snapshot` 的 Overall/PerAgent/PerModel 与 `MissingAssistantUsage`，持久化为 `<OutputDir>/meta/usage.json`，不可用时由 sessions replay 修复。Studio 只读展示真实累计 input/output/cost 和 Core 已有 agent 维度。

## Studio 生命周期与互斥方案

只读的 OpenProject、Config/Usage/Import 状态读取、Export preview、Review/Revision 检查不创建 Host。显式 Run、Next、Steer、Sync、Import、Co-create 和配置 live mutation 才按需创建或复用 Host。现有 `EngineService.controlMu` 负责 GUI 控制串行，Core 继续负责 `projectwrite`、Host exclusive、book lease 和 Advance/Budget gate；长任务通过同一 Host 的事件/事件通道回传。

项目切换前仍先校验目标 Store，再由 EngineService 等待/拒绝当前 Running、Pausing、Stopping；任何异步 UI 响应都必须带 projectId、generation 和 request sequence。

## M6 代码级实施方案

### M6-A：Studio Core Facade 与只读事实

1. 增加 `projectRoot/outputDir` 明确的路径解析和显式 Host config path option。
2. 增加 Quick Start/Outline 请求、Co-create 会话/恢复、Import 状态与事件、Export 请求/结果、Config 快照和 Usage 快照 ViewModel。
3. 扩展 Bridge/App 与 Studio App Facade：只读 API 不创建 Host；显式操作通过现有 Host 方法和 EngineService 互斥执行。
4. 为 imp.Event、Host Event、Host Snapshot 做薄适配，不改变 Engine route、Arbiter、revision 或 import/export 语义。

### M6-B：前端领域 Store 与真实长任务

1. 新增 `createProjectStore`、`importStore`、`configStore`、`exportStore`，保留 engine/revision/review/chapter store。
2. 创建、共创、导入都以 request sequence + project identity 接收响应；导入显示 Core stage/current/total/message，不计算伪百分比。
3. Provider/API key 默认遮罩；前端不读写 JSON、不缓存明文 key。配置保存后重新读取 effective snapshot。
4. Export 只提交 Core options 并展示实际 Path/Skipped/Bytes；Usage/Budget 只展示 Core 返回值。

### M6-C：Welcome / Wizard / Settings / Export UI

Welcome 提供 Quick Start、Co-create、Outline、Import、Open Project 的真实入口；现有项目内增加 Import、Model/Provider、Budget/Usage、Export 视图。UI 文案区分校验、配置、Provider、Model、Import、Export、Budget、文件系统和 Recovery Required 错误，且不伪造 Core 状态。

