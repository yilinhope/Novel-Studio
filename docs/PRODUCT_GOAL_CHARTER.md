# Novel Studio — Product Goal Charter & Scope Guardrails

> 用途：作为 Novel Studio 长期“北极星”文档。任何新 Milestone、PR、架构改造或功能吸收，都必须先与本文件对照。如果某项工作与本文件冲突，应先修改目标文档并明确原因，而不是在实现过程中自然漂移。

## 1. 一句话总目标

> **Novel Studio = ainovel-cli 的完整桌面工作台 + 对 AI-Novel-Writer 中真正有价值功能/交互的选择性吸收。**

```text
ainovel-cli
= 核心写作引擎、状态机、Store、长篇规划、Revision/Sync、Review、Import/Export、Provider 等业务权威

Novel Studio
= 为 ainovel-cli 提供专业桌面 GUI
+ 更好的可视化、编辑、审稿、资料管理和作者交互

AI-Novel-Writer
= 功能与 UX 参考来源
+ 只吸收真正有价值、且不破坏 ainovel 架构的部分
```

Novel Studio **不是**重新实现 ainovel-cli，也不是借机重新设计一套全新的小说 AI 平台。

## 2. 永久架构原则

### 2.1 ainovel-cli Core 是业务权威

以下能力优先继续由 Core 提供：

- Engine / Host 生命周期
- Volume / Arc / Chapter 长篇规划
- Architect / Writer / Editor / Arbiter
- Progress / RunMeta
- ChapterRecord
- Revision / Sync
- Review / Rewrite / Polish
- Next / Advance Gate
- Steer
- Import / Export
- Provider / Model / Budget / Usage
- Timeline / Foreshadow / Relationship / State Changes / Snapshots 等既有连续性数据
- Project lease / mutation safety

Studio 不平行创建第二套相同业务系统。

### 2.2 GUI 优先“暴露 Core”，而不是“复制 Core”

```text
Core 已有
→ Studio Facade / Bridge
→ ViewModel
→ GUI
```

禁止：

```text
Core 已有
→ React/Studio 再写一套类似业务逻辑
```

### 2.3 只有两种情况允许扩展 Core

1. Core 缺少安全、通用、CLI/GUI 都应该拥有的公共接口；
2. Core 本身存在真实缺陷。

不能以“GUI 更方便”为理由顺带修改业务语义。

## 3. AI-Novel-Writer 的正确角色

AI-Novel-Writer 不是第二个 Core。我们研究它主要是为了发现：

- 更好的作者工作流
- 更好的桌面 UI
- 更好的资料管理
- 更好的审稿交互
- 更好的候选稿/版本体验
- 更好的角色/故事线可视化
- Novel Studio 当前确实缺失的高价值功能

吸收方式：

```text
产品思路 / UX / 工作流
→ 重新设计并实现到 Novel Studio
```

而不是复制其桌面源码。

## 4. 产品开发顺序

### 第一阶段：ainovel-cli GUI Parity

目标：

> **所有面向普通作者的 ainovel-cli 功能，在 Novel Studio 中都有可用的 GUI 入口。**

允许保留 CLI-only 的仅应是：

- Headless automation
- Docker/脚本集成
- 开发者级调试参数
- 明确不属于桌面作者工作流的运维能力

任何作者日常功能都不应要求打开终端。

### 第二阶段：AI-Novel-Writer Feature Absorption

在 Core GUI Parity 基本完成后：

1. 对照 Novel Studio 与 AI-Novel-Writer；
2. 列出功能差异；
3. 分类为“已有 / UI 改进 / 真缺且高价值 / 不做”；
4. 根据真实差异重新生成 Roadmap。

Milestone 必须来自真实功能差异，而不是预先构造宏大架构。

### 第三阶段：Novel Studio 自有创新

只有在以下条件满足后才进入：

- ainovel-cli 作者功能 GUI parity 达标；
- AI-Novel-Writer 高价值能力已完成主要吸收；
- 用户实际使用暴露出稳定的新需求。

## 5. 当前历史状态

### V1

V1 已完成的是：

> ainovel-cli **主要日常创作闭环**的桌面 GUI 化。

包括项目打开/切换、Quick Start、Co-create、Outline Start、Engine 控制、Runtime、编辑/Save/Sync、Review/Next/Steer、Import 主流程、Export、Provider/Model/Budget/Usage、恢复/锁/回归/Windows build。

正确评价：

```text
ainovel-cli 主创作闭环 GUI 化：完成
ainovel-cli 全功能 GUI parity：尚未完成
```

### V2-M1

V2-M1 已完成 Proposal / Version / Diff / Evidence。

这部分**保留，不回滚**。它不替代 Core，仍通过 Working Copy → SavedUnsynced → Sync，属于作者工作台增强。

但 V2-M1 **不自动意味着**后续必须做 Fact Engine → Dependency → Repair。

### 原 V2-M2

原计划 Fact Engine / Character Knowledge / Conflict：

> **暂停实施。**

原因：目标已开始偏离“先完整 GUI 化 ainovel-cli”。ainovel-cli 已有 timeline / foreshadow / relationship / state / snapshot 等事实资产，应先完整暴露到 GUI。

已完成的源码审计可以保留为研究资料，但不构成实施承诺。

## 6. Scope Guardrails

每个新需求进入开发前必须回答：

1. **ainovel-cli 已经有这个能力吗？** 有就优先 GUI 化。
2. **这是 AI-Novel-Writer 值得吸收的作者体验吗？** 是就尽量在 Studio 层实现并复用 Core。
3. **是否必须修改 Core？** 只有缺安全公共接口或 Core 真缺陷时才允许。
4. **是否在创建第二套小说事实？** 如果“可能”，默认停止。
5. **这个功能是否要求普通作者回终端？** 如果是作者能力，则 parity 尚未完成。
6. **这个 Milestone 是否来自真实产品缺口？** 如果只是技术完整性或未来可能性，默认不进入当前 Roadmap。

## 7. 禁止自然漂移的典型模式

```text
“既然有 Proposal，不如顺便做 Fact Engine”
“既然有 Facts，不如顺便做 Dependency Graph”
“既然有 Dependency，不如做自动 Repair”
“既然做桌面版，不如重新设计 Core”
```

每一步都必须有真实用户价值和现有能力差异证明。

## 8. 当前正确 Roadmap

```text
V1
主要创作闭环 GUI 化
✅ 完成

V2-M1
Proposal / Version / Diff / Evidence
✅ 完成并保留

NEXT
Core GUI Parity Completion
⏳ 当前最高优先级

之后
AI-Novel-Writer Feature Absorption
⏳ Parity 后重新制定

再之后
Novel Studio Native Innovation
⏳ 暂不规划
```

## 9. Core GUI Parity 完成标准

只有满足以下条件后，才可以说“ainovel-cli 基本完整 GUI 化”：

1. 对 ainovel-cli 当前作者功能做完整 inventory；
2. 每项功能标记 Full GUI / Partial GUI / Missing GUI / Intentionally CLI-only；
3. 所有作者日常功能达到 Full GUI；
4. Partial/Missing 只允许存在于明确的非作者工作流；
5. 同一项目使用 GUI 不需要终端补功能；
6. GUI 不改变 Core 状态语义；
7. CLI 仍保持可用。

## 10. 最终产品定位

Novel Studio 的长期定位不是“重做一个比 ainovel-cli 更复杂的新引擎”，而是：

> **以 ainovel-cli 为可靠创作核心，提供专业桌面作者工作台，并持续吸收成熟小说写作产品中真正有价值的交互和能力。**

```text
ainovel-cli Core
+
Complete Desktop GUI
+
Selective AI-Novel-Writer Feature Absorption
+
Validated Novel Studio Innovations
=
Novel Studio
```
