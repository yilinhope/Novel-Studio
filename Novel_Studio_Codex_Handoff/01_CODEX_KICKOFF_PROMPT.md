# Codex 开工 Prompt

请先阅读本目录中的：

1. `00_README_FIRST.md`
2. `docs/Novel_Studio_V1_Engineering_Blueprint.md`
3. `docs/Novel_Studio_V1_Design_Baseline_Annotated.md`

然后检查当前工作区中的 `ainovel-cli` 源码（若尚未克隆，请从 `https://github.com/voocel/ainovel-cli` 获取当前主分支）。

## 你的当前任务只做 M0 + M1 + M2 的第一条闭环

目标：

```text
启动 Novel Studio Desktop
→ 选择已有 ainovel 项目目录
→ Go 后端读取真实项目状态
→ React 显示 Project Overview
→ 显示 Volume / Arc / Chapter 树
→ 点击章节读取并显示真实 Markdown 正文
```

### 开工前先做源码审计，不要直接改

请先输出一份简洁的代码级映射：

- 当前 `cmd/` 入口结构
- Engine / Host / Store / Domain / Config 的实际 package 路径
- Project 状态从哪些现有 Store/API 读取
- Volume / Arc / Chapter Plan 的真实数据结构和存储位置
- Chapter 正文的真实读取路径/API
- 哪些逻辑目前绑在 TUI command handler 上、以后需要抽成 Application Service
- 哪些 package 可以直接复用，哪些需要极薄的 adapter
- 当前 Go module 名、Go 版本、主要依赖
- `go test ./...` / build 的当前基线结果

确认实际源码后，再实现；**不要按照设计文档猜 package 名。**

## 架构要求

推荐新增：

```text
cmd/novel-studio/
internal/studio/app/
internal/studio/bridge/
internal/studio/events/
internal/studio/viewmodel/
desktop/frontend/
```

但最终目录必须服从当前 ainovel-cli 的真实结构；如果仓库实际结构更适合另一种薄适配方式，请先说明再做。

### Backend 原则

React 不能直接解析：

- progress JSON
- outline JSON
- chapter files
- checkpoints
- config 文件

这些都由 Go 层读取，然后返回稳定 ViewModel。

第一批只需要：

```text
OpenProject(path)
GetProjectOverview()
GetProjectTree()
GetChapter(chapter)
```

不要现在实现：

```text
Start / Continue / Pause / Resume
Sync
Review
Steer
Import
Export
```

等 Read-only 项目闭环稳定后再做。

## 前端要求

技术栈：

```text
Wails
React
TypeScript
Zustand
```

视觉参考优先级：

### 可直接作为基准
- `design/v1_reference/synced_idle`
- `design/v1_reference/writer_running`

### 布局/视觉参考
- `design/v1_reference/novel_studio_1`
- `design/v1_reference/novel_studio_2`
- `design/v1_reference/multi_agent_runtime_*`
- `design/v1_reference/master_outline_novel_studio`

### 只参考视觉，字段由 Core 决定
- `design/v1_reference/provider_novel_studio`
- `design/v1_reference/editor_review_center_novel_studio`

### 注意
- `design/v1_reference/novel_studio_6/screen.png` 是旧状态，不要照着实现。
- Waiting Sync 的正确规则以设计基线文档为准。

Stitch HTML 只用于提取布局、间距、字体和视觉 token。  
不要直接把大量静态 HTML 粘贴为 React 生产代码。

## V1 禁止扩张

本阶段不要实现：

- Fact Engine
- Character Knowledge
- Dependency Graph
- ContextManifest
- Impact Analysis
- Repair Engine
- Proposal Inbox
- 高级版本历史
- Evidence Diff
- 云同步
- 多人协作
- ChromaDB / 新向量数据库
- Workflow Builder

## 验收标准

完成后必须满足：

1. 原 `ainovel-cli` CLI 仍能 build/run；
2. 尽量保证 `go test ./...` 通过；
3. Novel Studio 能启动；
4. 能选择并打开真实 ainovel 项目；
5. Overview 数据来自真实 Core/Store；
6. 左侧能显示真实 Volume / Arc / Chapter；
7. 点击章节可以显示真实正文；
8. 前端没有直接解析 Core 文件；
9. 没有修改小说生成逻辑；
10. 给出改动文件列表和后续 M3 建议。

每完成一个可运行阶段再提交下一阶段，不要一次实现整个 V1。
