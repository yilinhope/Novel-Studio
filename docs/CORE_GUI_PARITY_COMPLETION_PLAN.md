# Novel Studio — Core GUI Parity Completion Plan

> 目标：补齐 ainovel-cli 已有、但 Novel Studio 尚未完整暴露给桌面用户的能力。
> 本阶段不新增第二套业务系统，不继续原 V2-M2 Fact Engine 实施。
> 最终交付标准：作者使用 ainovel-cli 的日常功能时，不需要回终端。

## 1. 本阶段定位

V1 已完成“主要创作闭环 GUI 化”。本阶段解决的是：

> **ainovel-cli 功能覆盖率，而不是新产品功能。**

优先级：

```text
Core 已有能力但 GUI 缺失
>
AI-Novel-Writer 新功能吸收
>
Novel Studio 原生创新
```

## 2. 第一件事：完整 Core Parity Inventory

不要直接从已有印象开工。

先从当前 ainovel-cli / Novel Studio 源码做一次穷举审计，生成：

```text
docs/CORE_GUI_PARITY_MATRIX.md
```

必须从以下来源枚举：

- CLI/TUI command registration
- README 中面向作者的命令
- Host public methods
- Store 中作者可查看的业务数据
- Config schema
- Import/Export options
- style/rules/simulation/diagnostics
- runtime observability

矩阵至少包含：

| Core Feature | CLI/TUI Entry | Core API/Data | Studio Current | Status | Action |
|---|---|---|---|---|---|
| ... | ... | ... | ... | Full/Partial/Missing/CLI-only | ... |

后续发现遗漏，先补矩阵，再开发。

## 3. 已知优先补齐区域

以下是当前已确认的主要缺口；最终以完整源码审计矩阵为准。

### P1 — Story Data Center

建议入口：

```text
故事
  故事前提
  人物
  世界观
  全书大纲
  Compass / 当前方向
```

审计并暴露：

- premise
- characters
- world rules
- outline
- layered outline
- compass
- chapter / arc / volume summaries（放到合适位置）

第一阶段以只读可视化为主。Core 有安全编辑入口时再开放编辑。

禁止 React 直接解析业务 JSON。

### P2 — Continuity Center

建议入口：

```text
连续性
  时间线
  伏笔
  人物关系
  人物状态
  角色快照
```

复用 ainovel 既有数据：

- timeline
- foreshadow
- relationships
- state changes
- character snapshots
- first appearance
- summaries / continuity projections

目标：把 Core 已经维护的连续性能力“看得见”。

**不要先造新 Fact Engine。**

### P3 — Reference Simulation / 仿写画像

完整 GUI 化 Core 已有：

```text
/simulate
/importsim
simulation profile
```

建议页面：

```text
参考作品 / 仿写画像
```

功能：

- 查看 simulate source files
- 运行分析
- 显示分析状态
- 查看当前 simulation profile
- 导入已有 profile
- profile 来源/更新时间
- Core 支持时显示增量分析状态

所有分析仍走原 Core。

### P4 — Writing Rules & Style Workspace

补齐 Core 已有：

```text
global rules
project rules
voice
anti-AI-tone
style profile / style selection
```

建议页面：

```text
写作规则与文风
```

功能：

- 全局规则
- 项目规则
- 当前有效规则预览
- 新建/编辑规则（只通过安全 Core/Facade 写入）
- Voice Layer
- anti-AI-tone
- 当前 style
- 覆盖来源说明

后续可参考 AI-Novel-Writer Writing Skills 优化 UX，但第一步先忠实 GUI 化 ainovel。

### P5 — Diagnostics Center

GUI 化 Core `/diag`。

建议至少包括：

- 运行诊断
- findings 分类
- severity
- 相关章节/对象
- 建议
- diag export
- 最近诊断时间

不得在 Studio 重写诊断规则。

### P6 — Advanced Configuration Parity

完整审计当前 config schema，并补齐作者有实际用途的字段。

重点核对：

- fallbacks
- notify
- style
- stream idle timeout
- provider advanced fields
- provider extra / extra body
- headers / protocol-specific options
- model capabilities
- reasoning-related options
- 其他当前 schema 字段

原则：

- UI 只映射真实 Core 字段；
- 不创造第二套 config；
- API key / secret 永不回显到日志；
- 高级字段可放 Advanced 折叠区。

### P7 — Import Option Parity

当前 Import 主 pipeline 已 GUI 化，但需要和 CLI/TUI option 做完整对照。

审计并补齐实际存在的：

- auto confirm
- story resolution
- continue after import
- guidance
- recovery/resume
- segmentation confirmation
- 其他当前 Core 参数

不能宣传 Core 不支持的文件格式。

### P8 — Runtime Observability Parity

对照 TUI / Host Snapshot / runtime events，检查是否遗漏：

- context usage
- context health
- compression status
- active route/flow details
- current gate/hold
- agent/model detail
- budget warning/hard-stop reason
- 当前 Core 已暴露但 Runtime Center 未显示的状态

只显示真实数据，不推测。

## 4. 实施 Wave

建议按四个大 Wave 推进，而不是十几个碎 PR。

### Wave A — Inventory + Read-only Core Data

目标：

1. 完整 `CORE_GUI_PARITY_MATRIX.md`
2. Story Data Center
3. Continuity Center

原因：风险最低，大量是已有 Core 数据的只读 ViewModel，可立即提升作者工作台完整度。

DoD：

- premise / characters / world / outline / compass 可查看；
- timeline / foreshadow / relationship / states / snapshots 可查看；
- React 不直接解析 Core 文件。

### Wave B — Simulation + Rules + Style

目标：

1. `/simulate`
2. `/importsim`
3. rules
4. voice / anti-AI-tone / style

需要审计：

- 哪些操作只读；
- 哪些会调用模型；
- 哪些会写 Core metadata；
- 是否需要 Host/exclusive；
- global vs project path；
- crash/recovery。

DoD：作者不需要终端即可管理仿写画像、写作规则和文风。

### Wave C — Diagnostics + Config + Import + Runtime Gaps

目标：

1. `/diag`
2. advanced config
3. import option parity
4. runtime observability parity

DoD：CLI/TUI 中剩余的作者向工具基本都有 GUI，高级参数不再强迫用户手改 JSON。

### Wave D — Parity Regression & Closure

重新执行完整矩阵：

```text
Full
Partial
Missing
CLI-only
```

要求：

- 所有普通作者能力：Full
- Partial/Missing：清零或有明确非作者用途说明
- Intentionally CLI-only：逐项说明理由

输出：

```text
docs/CORE_GUI_PARITY_MATRIX.md
docs/CORE_GUI_PARITY_RELEASE_CHECKLIST.md
docs/CORE_GUI_PARITY_KNOWN_LIMITATIONS.md
```

## 5. UI 信息架构建议

```text
项目
  总览

故事
  故事前提
  人物
  世界观
  全书大纲
  当前方向 / Compass

长篇规划
  Volume
    Arc
      Chapter

连续性
  时间线
  伏笔
  人物关系
  人物状态
  角色快照

质量
  Review
  Proposal
  诊断

工具
  参考作品 / 仿写画像
  小说导入
  导出

创作设置
  写作规则
  文风 / Voice
  模型与 Provider
  Budget / Usage

系统
  Runtime
  Logs
```

最终命名可按现有设计系统调整。

## 6. Backend 原则

所有新页面继续：

```text
React
→ Wails Bridge
→ Studio Facade/ViewModel
→ Core Store/Service
```

禁止：

```text
React
→ 直接读 JSON/JSONL/Markdown 业务文件
```

只读功能不得为了方便隐式创建 Host。

写操作继续遵守：

- project scope
- generation/request identity
- book lease
- projectwrite
- Host exclusive（如 Core 需要）
- Core success 与 View refresh 分离

## 7. Project Scope / Stale Safety

所有 async 页面继续使用：

```text
projectId
generation
requestId / sequence
```

项目切换后：

- 旧请求不得写入新项目；
- 旧响应不得更新新项目 UI；
- Backend 写入作用域由当前项目决定；
- Frontend ProjectID 只用于 stale validation，不决定磁盘路径。

## 8. 测试计划

每个 Wave 至少包括：

### Go

- correct ProjectRoot / OutputDir
- readonly API does not create Host
- current project scope
- stale project request rejected
- read model matches Core source
- mutation goes through Core service
- global/project precedence
- secret redaction
- failure does not corrupt Core state

### Frontend

- loading / error / empty states
- project switch reset
- stale response protection
- large list rendering
- no direct file parsing
- settings secret masking

### Production

- Windows Wails build
- Chinese/space path
- existing project
- large project
- production event binding

## 9. 性能要求

Story/Continuity 页面可能有大量数据。

要求：

- pagination / virtualization when needed
- lazy detail
- avoid loading all chapter bodies
- bounded logs/events
- avoid rebuilding heavy Core projections just to open a page

对 500–1000 章项目至少做基本体验检查。

## 10. 完成标准

Core GUI Parity Completion 结束时：

1. 已有完整 parity inventory；
2. ainovel-cli 普通作者功能均有 GUI；
3. Story 数据可视化完整；
4. Continuity 数据可视化完整；
5. Simulation 可用；
6. Rules/Style/Voice 可管理；
7. Diagnostics 可用；
8. Advanced Config 不再依赖手改 JSON；
9. Import 参数与 Core 对齐；
10. Runtime 关键观测信息对齐；
11. CLI 仍可正常使用；
12. GUI 不复制 Core 业务；
13. 无第二套小说事实；
14. Windows production build 通过；
15. 所有剩余 CLI-only 项都有明确理由。

完成后才进入：

> **AI-Novel-Writer → Novel Studio Feature Absorption Roadmap**

届时重新研究并排序：

- Story Map
- Character Workspace / Relationship UX
- Reference Library
- Writing Skills UX
- Candidate Draft Workspace
- Human Review UX
- Batch Run Controls
- EPUB Import
- 其他当前真实高价值差异

不提前锁死 Milestone。

## 11. 当前源码核对补充（2026-09-26）

本节依据当前 `origin/main` 源码核对，细项见 [Core GUI Parity Matrix](CORE_GUI_PARITY_MATRIX.md)。矩阵状态描述当前桌面 GUI 的覆盖，不代表 Core 能力缺失。

- Studio 当前已提供主要创作闭环：创建/共创、项目打开、Engine 控制、Review/Next/Steer、章节编辑/Save/Sync、Import 主流程、TXT/EPUB Export、Provider/Model 基础配置、Budget/Usage 和 Runtime 事件展示。
- `internal/studio/app/project.go` 的项目快照目前只组装作品概览与章节树；Bridge 没有 Story/Continuity 的独立读取 API。Core Store 已提供 premise、人物、规则、大纲、Compass、摘要、时间线、伏笔、关系、状态、角色快照及配角首次出场投影。这些页面可先以只读 ViewModel 暴露，无需新建事实系统。
- TUI 注册了 `/simulate`、`/importsim` 和 `/diag`；Core 已有 Host 仿写入口、Simulation Store 与诊断/脱敏导出函数；当前 Studio 未提供对应入口。
- 当前 Config ViewModel 有 Provider、Model、Role、Budget、Notify 等映射字段，但编辑器未完整暴露 Config schema。Provider `extra` / `extra_body`、角色 fallbacks、流式空闲超时、style 和通知配置需要按真实字段补齐，且 Secret 不得回显。
- 当前 Import ViewModel 支持项目根目录、源文件、接受切分、故事状态、继续创作和 guidance；GUI 表单未暴露自动确认和故事状态选项，恢复以当前项目状态提示为主。
- 当前 Runtime ViewModel/页面显示运行状态、阶段、流程、Agent/工具事件、Token、成本和章节 Gate；尚未对齐 TUI/Host 可用的上下文健康与压缩等观测数据。具体字段是否可安全投影需 Wave C 再核对。
- `/reopen` 有 Core Host 与 TUI 入口，但没有 Studio Bridge 操作；应列为缺失 GUI 的作者操作。

因此 Wave A 的边界是：从当前打开项目的 Core Store 读取已有数据，经 Studio Facade/ViewModel 与 Wails Bridge 投影到 GUI；只读读取不创建 Host、不写回元数据、不推导新事实。若实现时发现对应数据并不存在于当前项目或只能通过改动 Core 业务语义取得，应暂停该项并更新矩阵。
