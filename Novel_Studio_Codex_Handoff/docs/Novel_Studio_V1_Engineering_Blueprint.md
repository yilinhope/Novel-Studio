# Novel Studio V1 — 工程实施蓝图

> 目标：在**不改变 ainovel-cli 核心创作行为**的前提下，为其增加完整桌面 GUI。  
> 技术路线：**Go Core + Wails + React + TypeScript**。  
> V1 定义：**ainovel-cli Desktop GUI**，CLI/TUI 保留并作为行为基准。

---

# 1. V1 工程原则

## 1.1 Core 不重写

V1 不重写以下核心能力：

- Engine / Route
- Architect / Writer / Editor / Arbiter
- Store
- Checkpoint / Saga
- Volume / Arc 滚动规划
- Context 管理
- Review / Rewrite / Polish
- Steer
- `/sync`
- `/review on|off`
- `/next`
- Import / Export
- Provider / Model 配置

GUI 只负责：

1. 展示 Core 当前事实；
2. 调用 Core 已有动作；
3. 把 Core 运行事件映射到 UI；
4. 提供编辑器和桌面交互。

---

# 2. 推荐仓库策略

## 2.1 直接基于 ainovel-cli Fork 开发

第一阶段建议保持**一个 Go Module**。

原因：

- ainovel-cli 大量实现位于 `internal/`
- 同一 module 下新增 Desktop command 可以直接复用内部 package
- CLI 和 GUI 能共享同一 Core
- 后续更容易做 CLI / GUI 对照测试

目标结构：

```text
novel-studio/
├── cmd/
│   ├── ainovel-cli/              # 原 CLI/TUI，保留
│   └── novel-studio/             # 新 Wails Desktop 入口
│
├── internal/
│   ├── ...                       # ainovel-cli 原代码
│   │
│   └── studio/
│       ├── app/                  # Application Facade
│       ├── bridge/               # Wails bindings
│       ├── events/               # GUI Event DTO
│       └── viewmodel/            # GUI ViewModel
│
├── desktop/
│   └── frontend/
│       ├── src/
│       │   ├── app/
│       │   ├── components/
│       │   ├── features/
│       │   ├── stores/
│       │   ├── services/
│       │   └── types/
│       ├── package.json
│       └── ...
│
├── go.mod
└── ...
```

不要第一版拆成：

```text
ainovel-core repo
novel-studio repo
```

否则 `internal` package 边界会立刻增加迁移工作。

---

# 3. 四层架构

```text
┌─────────────────────────────┐
│ React / TypeScript UI       │
│ Stitch Design Implementation│
└──────────────┬──────────────┘
               │
         Wails Binding
               │
┌──────────────▼──────────────┐
│ Studio Application Facade   │
│ Project / Engine / Chapter  │
│ Revision / Review / Config  │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│ ainovel-cli Core            │
│ Host / Engine / Workers     │
│ Store / Import / Sync       │
└──────────────┬──────────────┘
               │
┌──────────────▼──────────────┐
│ Project Filesystem          │
│ output/novel + .ainovel     │
└─────────────────────────────┘
```

核心原则：

> React 不直接读取 ainovel 的 JSON / Markdown 内部文件结构。

React 只调用 Application API。

这样以后 Core 文件格式变化，前端不需要跟着重写。

---

# 4. Application Facade

建议新增：

```text
internal/studio/app/
```

它不是新的业务引擎。

它只是把现有 Core 整理成 GUI 可调用的稳定接口。

建议服务：

```text
ProjectService
EngineService
ChapterService
RevisionService
ReviewService
SteerService
OutlineService
CharacterService
WorldService
ContinuityService
ConfigService
UsageService
ImportService
ExportService
LogService
```

V1 不要求所有服务一次写完，按开发阶段逐步增加。

---

# 5. ProjectService

职责：

- 打开项目目录
- 识别是否为 ainovel 项目
- 获取书籍信息
- 获取项目总览
- 获取项目导航树
- 创建新项目
- 最近项目列表由 Studio 自己维护

API 示例：

```go
type ProjectService interface {
    Open(path string) (ProjectView, error)
    Close() error

    GetOverview() (ProjectOverview, error)
    GetTree() (ProjectTree, error)

    CreateQuick(req QuickStartRequest) error
    CreateFromOutline(req OutlineStartRequest) error
}
```

### ProjectOverview

只返回 GUI 真正需要的数据：

```go
type ProjectOverview struct {
    Title          string
    ProjectPath    string

    Phase          string
    Flow           string

    CurrentChapter int
    TargetChapters int

    CurrentVolume  string
    CurrentArc     string

    WordCount      int

    Engine         EngineView

    UnsyncedCount  int
    ReviewPending  int

    Usage          UsageView
}
```

不要让 React 自己从多个 Store 文件拼这些数据。

---

# 6. EngineService

这是 V1 最关键的服务。

必须提供：

```go
Start(...)
Continue(...)
Pause()
Resume()
Stop()

GetState()
SetReviewMode(...)
NextChapter()
```

但要特别注意：

> GUI 不决定 Route。

例如 React 点击：

```text
Continue
```

只是告诉 Core：

```text
允许继续执行现有 Engine
```

到底运行 Architect / Writer / Editor，由原 Engine 根据 Store 决定。

---

# 7. 统一 Engine UI State

Core 原本有 Phase / Flow。

GUI 再派生一个简单的 UI State：

```go
type EngineUIState string

const (
    EngineIdle          EngineUIState = "idle"
    EngineRunning       EngineUIState = "running"
    EnginePaused        EngineUIState = "paused"
    EngineWaitingReview EngineUIState = "waiting_review"
    EngineWaitingSync   EngineUIState = "waiting_sync"
    EngineError         EngineUIState = "error"
)
```

注意：

这只是：

> **ViewModel**

不能反过来替代 Core 的 Phase / Flow。

---

# 8. Engine 状态映射

GUI 全局 Shell 统一从一个 selector 获取状态：

```text
Core Facts
    ↓
deriveEngineUIState()
    ↓
Topbar
Runtime Center
Chapter Editor
Bottom Statusbar
```

禁止每个页面自己判断。

按钮规则：

| State | 操作 |
|---|---|
| Idle | Continue |
| Running | Pause / Stop |
| Paused | Resume / Stop |
| WaitingReview | Approve & Next / Stop |
| WaitingSync | Sync |
| Error | Retry / View Error / Stop |

这样可以避免 Stitch 之前出现：

```text
主体 Idle
顶部 Running
```

这种状态矛盾。

---

# 9. ChapterService

职责：

- 读取章节
- 保存用户编辑后的 Markdown
- 返回章节保存 / Sync 状态
- 获取 Chapter Plan
- 获取所属 Volume / Arc

API：

```go
GetChapter(chapter int) (ChapterView, error)

SaveChapter(
    chapter int,
    content string,
) (ChapterSaveResult, error)

GetChapterPlan(
    chapter int,
) (ChapterPlanView, error)
```

### ChapterView

```go
type ChapterView struct {
    Number int
    Title  string

    Volume string
    Arc    string

    Content string
    WordCount int

    EditState ChapterEditState

    Plan *ChapterPlanView
}
```

---

# 10. ChapterEditState

建议 GUI 层统一：

```go
type ChapterEditState string

const (
    ChapterClean         = "clean"
    ChapterModified      = "modified"
    ChapterSavedUnsynced = "saved_unsynced"
    ChapterSyncing       = "syncing"
    ChapterSynced        = "synced"
    ChapterGenerating    = "generating"
    ChapterReviewPending = "review_pending"
    ChapterError         = "error"
)
```

规则：

```text
编辑器内修改
→ Modified

保存文件
→ SavedUnsynced

/sync 成功
→ Synced
```

最重要的不变量：

```text
SavedUnsynced
    =>
EngineWaitingSync
```

从而自动禁用：

```text
Continue
Next Chapter
```

---

# 11. RevisionService / Sync

V1 不重新实现修订逻辑。

只需要把现有 `/sync --check` 与 `/sync` 背后的逻辑暴露成 API。

建议：

```go
type RevisionService interface {
    Check(ctx context.Context) ([]UnsyncedChapter, error)

    Sync(
        ctx context.Context,
        chapters []int,
    ) (SyncResult, error)
}
```

GUI：

```text
保存章节
↓
Check()
↓
Unsynced
↓
Sync()
↓
Core 重建派生状态
↓
Synced
```

不要在前端实现：

- 摘要重建
- Timeline 重建
- Foreshadow 重建
- Relationship 重建
- State 重建
- Replan

这些继续属于 Core。

---

# 12. ReviewService

V1 只暴露 ainovel 已有 Review。

```go
ListReviews() ([]ReviewSummary, error)

GetReview(
    chapter int,
) (ReviewView, error)
```

ReviewView 不要发明新评分：

```go
type ReviewView struct {
    Chapter int
    Status  string

    Issues []ReviewIssue

    RewriteStatus string
    PolishStatus  string
}
```

如果 Core Review 本身有：

- category
- severity
- evidence
- suggestion

就映射。

没有就不制造。

---

# 13. SteerService

Steer 是 V1 必需能力。

```go
Submit(
    ctx context.Context,
    instruction string,
) (SteerReceipt, error)
```

UI 保持：

```text
告诉创作团队：
[............................]

[发送]
```

Steer 在 Engine Running 时仍允许提交。

如果 Core 返回 Arbiter decision，可以作为运行事件展示。

不要让 UI 自己判断影响范围。

---

# 14. Outline / Volume / Arc

只读优先。

V1 首先实现：

```go
GetFoundation()
GetOutline()
GetVolumes()
GetArcs(volumeID)
GetChapterPlan(chapter)
```

页面：

```text
Compass / Outline
Volume
Arc
Chapter Plan
```

GUI 可以展示。

除非 ainovel 当前已有明确编辑入口，否则 V1 不急着增加任意手工修改规划的功能。

---

# 15. Characters / World / Continuity

V1 作为数据浏览器。

API：

```text
ListCharacters
GetCharacter
GetCharacterSnapshots

GetWorld

ListForeshadowing
GetTimeline
GetRelationships
```

优先级低于：

```text
Engine
Chapter
Sync
Review
Steer
```

不要因为 Stitch 有高级图谱，就优先开发复杂图形系统。

---

# 16. ConfigService

必须忠实映射当前 ainovel config。

配置来源：

```text
~/.ainovel/config.json
./.ainovel/config.json
```

项目级覆盖全局配置。

ConfigService：

```go
GetEffectiveConfig()
GetGlobalConfig()
GetProjectConfig()

SaveGlobalConfig(...)
SaveProjectConfig(...)

TestProvider(...)
SetActiveModel(...)
```

前端不直接操作 JSONC 文件。

---

# 17. Provider ViewModel

Provider 页面必须动态生成。

```ts
interface ProviderView {
  id: string
  type: string
  baseUrl?: string
  hasApiKey: boolean
  models: ModelView[]
}
```

不要假定一定有：

```text
Anthropic
DeepSeek
OpenAI
```

用户可能只有：

```text
OpenRouter
Custom Proxy
Ollama
```

---

# 18. Wails Event Bus

桌面 GUI 不能依赖轮询读取 Store。

Engine 运行时应该通过事件更新前端。

建议统一事件：

```text
studio:engine-state
studio:agent
studio:step
studio:chapter
studio:usage
studio:review
studio:sync
studio:log
studio:error
```

例如：

```go
type EngineStateEvent struct {
    State   EngineUIState
    Agent   string
    Chapter int
    Step    string
}
```

前端只订阅一次，全局 Zustand Store 更新状态。

---

# 19. 日志事件

不要把全部日志当成 React state 无限累积。

Runtime Center：

```text
最近 N 条实时事件
```

完整日志：

```text
按需读取日志文件
```

建议：

```go
TailLogs(limit int)
ReadLog(...)
```

前端实时保留：

```text
500~2000 条
```

避免长时间挂机后内存持续增长。

---

# 20. 前端状态管理

推荐：

```text
Zustand
```

至少拆：

```text
useProjectStore
useEngineStore
useChapterStore
useRuntimeStore
useConfigStore
```

不要做一个巨大：

```text
useAppStore
```

---

# 21. 前端目录建议

```text
desktop/frontend/src/
│
├── app/
│   ├── App.tsx
│   ├── router.tsx
│   └── shell/
│
├── features/
│   ├── projects/
│   ├── overview/
│   ├── chapters/
│   ├── runtime/
│   ├── review/
│   ├── steer/
│   ├── outline/
│   ├── characters/
│   ├── world/
│   ├── continuity/
│   ├── providers/
│   ├── import/
│   └── export/
│
├── components/
│   ├── ui/
│   ├── layout/
│   └── common/
│
├── stores/
├── services/
├── types/
└── styles/
```

不要按照 Stitch：

```text
novel_studio_1
novel_studio_2
...
```

来组织代码。

Stitch 是视觉稿，不是代码架构。

---

# 22. Design Token

把 Stitch 已确定的视觉语言提取为：

```text
tokens.css
```

包括：

```text
background
panel
surface
border
text
muted
accent
warning
danger
success

font-ui
font-body
font-mono

sidebar-width
inspector-width
statusbar-height
```

然后所有页面复用。

不要直接把 Stitch 每个 HTML 页面自己的 CSS 全复制进 React。

---

# 23. 正文编辑器

V1 不建议第一天就上复杂编辑器框架。

小说正文目前是 Markdown 文件。

第一版可以：

```text
Textarea / contenteditable wrapper
```

或者轻量 CodeMirror。

要求：

- 大文本稳定
- 中文 IME 正常
- Undo / Redo
- 字数统计
- Save
- Readonly
- 快捷键 Ctrl/Cmd+S

如果后续需要复杂选择改写，再升级编辑器。

---

# 24. Writer Running 编辑锁

当：

```text
Current Agent = Writer
Current Chapter = 打开的章节
Engine = Running
```

正文：

```text
readonly = true
```

显示：

> Writer 正在生成本章。暂停后可手动编辑。

如果 Writer 正在写第 15 章，而用户查看第 8 章：

V1 可以允许阅读。

是否允许修改历史章节：

- 若修改：Save 后立即触发 WaitingSync
- Core 在安全点停下 / 阻止下一步
- 后续必须 Sync

具体安全边界以 Core 原行为为准。

---

# 25. Runtime Center

正式页面必须包含：

```text
当前 Engine State

Current Agent
Current Chapter
Current Step

Pipeline

Agent Status

Usage

Runtime Log

Pause / Resume / Stop
```

Writer Pipeline 必须使用当前 Core 真实顺序：

```text
novel_context
read_chapter
plan_chapter
draft_chapter
check_consistency
commit_chapter
```

不要自行增加：

```text
Arbiter Validation
Foreshadow Insert
```

成为固定 Step。

---

# 26. Import

V1 映射现有 Import Pipeline：

```text
ingest
→ segment
→ confirm
→ analyze
→ synthesize
→ publish
```

GUI 做：

- 文件选择
- 进度
- 章节切分确认
- guide 修改
- Continue / Cancel

不要加入 ChromaDB / Vector DB。

---

# 27. Export

当前 Core 支持：

```text
TXT
EPUB
章节范围
Overwrite
```

GUI 应加入：

```text
Export
```

这在之前 Stitch 重点页面里容易遗漏，但属于 ainovel-cli 已有功能，因此 V1 应覆盖。

Export Dialog：

```text
Format: TXT / EPUB
Range: All / Chapter x-y
Destination
Overwrite
```

---

# 28. 开发阶段划分

## M0 — Fork + Build Baseline

目标：

```text
原 ainovel-cli
go test ./...
CLI 能正常运行
```

不加 GUI 功能。

---

## M1 — Wails Shell

完成：

```text
Desktop 窗口启动
React Shell
Stitch Design Tokens
Sidebar
Topbar
Statusbar
```

数据先 Mock。

---

## M2 — Read-only Project

完成：

```text
Open Project
Overview
Volume / Arc / Chapter Tree
Read Chapter
Read Outline
Read Characters
```

此时：

> GUI 已经能浏览真实 ainovel 项目。

---

## M3 — Engine Bridge

完成：

```text
Continue
Pause
Resume
Stop
Engine State
Agent
Step
Usage
Runtime Logs
```

Runtime Center 接真实数据。

---

## M4 — Chapter Editing + Sync

完成：

```text
Edit
Save
Unsynced Detection
WaitingSync
Sync
Continue Gate
```

这是 V1 最关键的人机介入闭环。

---

## M5 — Review + Gate + Steer

完成：

```text
Review
Review On/Off
Next
WaitingReview
Steer
Arbiter Result
```

---

## M6 — Create / Import / Export / Config

完成：

```text
Quick Start
Co-create
Start from Outline
Import Novel
Export TXT/EPUB
Provider Config
Model Switch
Budget
```

---

## M7 — Regression

CLI / GUI 同项目对照。

---

# 29. CLI / GUI 回归测试矩阵

至少覆盖：

| 场景 | CLI | GUI |
|---|---|---|
| 新建快速项目 | ✓ | ✓ |
| 共创规划 | ✓ | ✓ |
| 连续生成 | ✓ | ✓ |
| Pause / Resume | ✓ | ✓ |
| 崩溃恢复 | ✓ | ✓ |
| Review On | ✓ | ✓ |
| Next | ✓ | ✓ |
| Steer | ✓ | ✓ |
| 修改正文 | ✓ | ✓ |
| Sync Check | ✓ | ✓ |
| Sync | ✓ | ✓ |
| Arc 边界 | ✓ | ✓ |
| Volume 边界 | ✓ | ✓ |
| Import | ✓ | ✓ |
| Export TXT | ✓ | ✓ |
| Export EPUB | ✓ | ✓ |
| Provider 配置 | ✓ | ✓ |
| Model 切换 | ✓ | ✓ |

原则：

> 同一输入条件下，GUI 不应该改变 Core 的状态迁移结果。

---

# 30. V1 不开发

明确禁止当前阶段扩张到：

```text
Fact Engine
Character Knowledge
Dependency Graph
ContextManifest
Impact Analyzer
Repair Engine
Proposal Inbox
高级 Revision History
高级 Evidence Diff
复杂 Knowledge Graph
Cloud Sync
多人协作
Plugin Marketplace
Workflow Builder
```

Stitch 中相关设计只作未来参考。

---

# 31. 第一个实际代码目标

不要一开始实现全部页面。

第一条真实闭环应为：

```text
启动 Novel Studio
↓
选择已有 ainovel 项目目录
↓
读取真实 Project Overview
↓
显示 Volume / Arc / Chapter
↓
打开某章正文
```

完成后第二条：

```text
点击 Continue
↓
Core Engine 运行
↓
GUI 收到 Writer / Step Events
↓
Runtime Center 更新
↓
章节提交
↓
GUI 自动刷新正文
```

第三条：

```text
Pause
↓
人工编辑章节
↓
Save
↓
Waiting Sync
↓
Sync
↓
Continue
```

只要这三条闭环稳定：

> Novel Studio V1 的核心架构就成立了。

---

# 32. 当前技术选择

## Backend

```text
Go
ainovel-cli Core
Wails
```

## Frontend

```text
React
TypeScript
Zustand
```

## Styling

基于 Stitch Design Tokens 实现。

不直接使用 Stitch 静态 HTML 作为生产代码。

## Storage

继续沿用 ainovel-cli 原 Store。

Studio 自己的少量 UI 配置（最近项目、窗口偏好等）单独保存，不污染小说 Core Store。

---

# 33. Studio 自有数据

V1 只允许 Studio 保存非小说业务数据：

```text
最近项目
窗口尺寸
Sidebar 展开状态
UI Theme
最近访问章节
```

不要在 V1 建第二套：

```text
角色数据库
大纲数据库
章节数据库
Fact 数据库
```

小说权威数据仍由 ainovel Store 管。

---

# 34. 最终 V1 架构定义

```text
                 Novel Studio
                      │
              React / TypeScript
                      │
                 Wails Bridge
                      │
             Studio App Facade
                      │
        ┌─────────────┼─────────────┐
        │             │             │
      Engine        Store        Existing
      / Host                      Services
        │             │
        └─────────────┼─────────────┘
                      │
              ainovel-cli Core
```

一句话：

> **Novel Studio V1 不是重新实现 ainovel-cli，而是把 ainovel-cli 变成一个拥有专业桌面工作台的应用。**

---

# 35. Definition of Done

Novel Studio V1 完成时：

1. 用户可以完全不打开终端；
2. 能完成 ainovel-cli 现有主要日常工作流；
3. CLI 仍可正常使用；
4. 同一项目可由 CLI / GUI 打开；
5. GUI 不创建第二套小说真相；
6. Unsynced 能可靠阻断继续创作；
7. Pause / Resume / Review / Next / Steer 均走 Core；
8. Engine / Agent / Step / Token / Cost 可视化；
9. Import / Export / Config 可在 GUI 完成；
10. CLI / GUI 回归测试通过。

完成这些之后，再进入 V2 的人机共创增强。
