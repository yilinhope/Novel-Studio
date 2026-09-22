# Novel Studio V1 — Codex Handoff

这是 Novel Studio V1 进入 Codex 开发阶段的当前上下文包。

## 项目目标

Novel Studio V1 = **ainovel-cli Desktop GUI**。

V1 不重新实现小说引擎，不改变 ainovel-cli 的核心创作行为。  
CLI/TUI 保留，并作为 GUI 行为的基准实现。

核心技术路线：

```text
ainovel-cli Core (Go)
        ↓
Studio Application Facade
        ↓
Wails Bridge
        ↓
React + TypeScript + Zustand
```

## 开始前按此顺序阅读

1. `docs/Novel_Studio_V1_Engineering_Blueprint.md`
2. `docs/Novel_Studio_V1_Design_Baseline_Annotated.md`
3. `01_CODEX_KICKOFF_PROMPT.md`
4. 需要视觉参考时查看 `design/v1_reference/`
5. 完整 Stitch 原始导出不纳入 Git 仓库；精选参考页面和设计说明位于 `design/v1_reference/` 与 `docs/`

`docs/Novel_Studio_V1_Stitch_UI_PRD.md` 是原始 UI 产品需求，作为背景参考。  
`docs/Novel_Studio_V1_Stitch_Final_Cleanup.md` 记录状态一致性约束。

## 上游仓库

- ainovel-cli: https://github.com/voocel/ainovel-cli
- AI-Novel-Writer（仅作为产品思路背景，V1 不依赖其代码）:
  https://github.com/EthanYoQ/AI-Novel-Writer

## 重要约束

- **不要复制 AI-Novel-Writer 的 GPL 桌面源码进 V1。**
- Stitch 的 `code.html` 是高精度视觉参考，不是生产架构。
- React 不直接读取 ainovel Store 文件；必须经 Go Application/Bridge。
- 不建立第二套小说数据库/真相源。
- 不因为 Stitch mock 中出现某个指标，就为它新增 Core 功能。
- V1 不做 Fact Engine、Dependency Graph、ContextManifest、Proposal、Cloud Sync 等 V2/V3 功能。
- 每次改动必须保持原 CLI 可构建、可运行，尽量保持 `go test ./...` 通过。

## 第一条开发闭环

```text
启动 Novel Studio
→ 选择已有 ainovel 项目目录
→ 读取真实 Project Overview
→ 显示 Volume / Arc / Chapter
→ 打开真实章节正文
```

在这条闭环稳定前，不接 Engine 写作动作。
