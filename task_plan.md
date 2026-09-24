# Task Plan: Novel Studio V1 — M6

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

- [ ] 建立共享 operation/project/generation 隔离和错误分类基础
- [ ] 增加只读 Config/Usage/Export Preview 能力，不隐式创建 Host
- [ ] 按 Core 真实入口实现 Create、Co-create、Outline、Import、Export、Config、Budget 服务
- [ ] 对需要 Host 的显式操作接入现有 lifecycle、exclusive、projectwrite、book lease 和 Core gate
- [ ] 先写失败测试，再实现 Go service/bridge API
- **Status:** in_progress

### Phase 3: Wails Bridge 与前端领域 Store

- [ ] 暴露真实 Wails API 和事件/快照投影
- [ ] 新增 `createProjectStore`、`importStore`、`configStore`、`exportStore`
- [ ] 保持既有 `engineStore`、`revisionStore`、`reviewStore`、`chapterEditorStore` 边界
- [ ] 所有异步请求带 projectId、generation、request sequence，旧项目响应不得污染当前项目
- [ ] API Key 默认遮罩，日志、console、event summary、error 和 snapshot 不出现 secret
- **Status:** pending

### Phase 4: 创建与导入工作流 UI

- [ ] Welcome 真实入口：快速开始、共创创建、从大纲开始、导入已有小说、打开已有项目
- [ ] Quick Start Wizard 映射 Core 必填输入、真实 Agent/Step/Usage 和失败恢复
- [ ] Co-create 映射阶段协议、消息、Agent 输出、动作、完成状态和 unfinished recovery
- [ ] Outline Preview/Validate/Create 使用 Core parser/validation/synthesis
- [ ] Import 8 阶段 UI：选择、Ingest、章节识别、确认、Analyze、Synthesize、Publish、完成
- [ ] Import 长任务不阻塞 UI，显示真实 lifecycle/checkpoint/recovery，遵守互斥
- **Status:** pending

### Phase 5: Export、Config、Model、Budget UI

- [ ] Export Dialog：TXT/EPUB、Core 支持的 range、destination、overwrite、真实输出路径
- [ ] Provider/Model 页面只映射 Core 字段和读写能力
- [ ] 明确 ProjectRoot 与 OutputDir，配置始终从 canonical ProjectRoot 读取，Store 使用 Core 规定 OutputDir
- [ ] Model 切换遵守 Core 的运行中限制和下一次调用语义，不偷偷重启 Engine
- [ ] Budget 页面显示 Spent、Book Budget、Warn Ratio、Hard Stop、Tokens、Cost 及真实 Agent Usage
- **Status:** pending

### Phase 6: 测试、CLI/GUI 对照与交付

- [ ] 补齐 M6 Go/Frontend 定向测试和 stale operation 测试
- [ ] 完成 Go tests、vet、Frontend tests/build、Windows Wails production build
- [ ] 完成 CLI/GUI 最终 Store/Config 语义对照
- [ ] GitNexus `detect-changes --scope all`、diff review、提交中文说明
- [ ] 创建/更新统一 M6 PR，附审计摘要和验证证据
- **Status:** pending

## Global Constraints

- 不重写 Core 初始化、Import、Export、Config、Arbiter 或 Engine 语义。
- 只读 OpenProject/GetConfig/GetReview/GetUsage/Export Preview/Import 状态不创建 Host。
- 只有用户明确触发、且 Core 需要 Host 的 Run/Sync/Import/Co-create/Next/Steer 等操作按需创建 Host。
- 不增加第二套 Provider 配置格式、项目模板、章节识别算法或 TXT/EPUB renderer。
- 不扩展 Fact Engine、Proposal、Knowledge、Impact Analysis、Repair/Replan、云同步、多人协作等功能。

## Errors Encountered

| Error | Attempt | Resolution |
|---|---:|---|
| `rtk proxy Get-Content` 无法解析 PowerShell cmdlet | 1 | 改用 `rtk pwsh -NoProfile -Command` 读取技能文件 |
