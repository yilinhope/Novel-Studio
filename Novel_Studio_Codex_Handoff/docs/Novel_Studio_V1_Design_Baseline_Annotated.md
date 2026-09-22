# Novel Studio V1 — 当前 Stitch 设计基线与页面状态标注

> 基于当前 Stitch 导出包的设计审查结果。  
> 本文**不要求删除任何页面**，只规定每个页面在后续开发中的用途与优先级。  
> 开发时以本文标签为准，避免把旧版 / V2-V3 概念稿误当作 V1 功能规格。

---

# 1. 当前总体结论

当前视觉体系已经满足 Novel Studio V1 的总体方向：

- 深色专业 Writing Studio / IDE 风格：✅
- 左侧项目导航 + 中央工作区 + 右侧上下文：✅
- Volume → Arc → Chapter 长篇层级：✅
- 正文作为视觉中心：✅
- Agent / Step / Token / Cost 运行信息：✅
- Sync / Review / Steer 有明确入口：✅
- Writer Running 时正文只读：✅
- Idle / Running 页面开始按状态拆分：✅

但当前导出包仍然混有：

1. V1 正式页面；
2. 旧版页面；
3. V2/V3 概念页面；
4. 视觉可用但业务字段不能直接照搬的页面。

因此：

> **当前设计可以作为 V1 开发的视觉基线，但不能把整个 ZIP 中所有页面和字段都直接当成功能规格。**

V1 开发必须按照以下标签使用。

---

# 2. 标签定义

## `[V1-FINAL]`

可以作为 V1 正式开发视觉基准。

允许开发人员直接参考：

- 页面布局
- 组件层级
- 状态表达
- 交互位置
- 信息密度

具体数据字段仍由 ainovel-cli Core / Bridge API 决定。

---

## `[V1-REFERENCE]`

页面属于 V1，但其中部分 mock 数据、指标或行为不是 ainovel-cli 现有真实能力。

可以参考视觉，不得照搬所有字段。

---

## `[V1-NEEDS-FIX]`

属于 V1 必需状态，但当前截图/状态逻辑仍然是旧版或存在矛盾。

不要按当前截图开发。

---

## `[V2-V3-CONCEPT]`

保留作为未来产品设计参考。

V1 不实现。

不要删除，后续 V2/V3 可重新评估。

---

# 3. 当前文件夹状态表

| 当前目录 | 页面含义 | 标签 | V1 开发处理 |
|---|---|---|---|
| `novel_studio_1` | Welcome / Projects | `[V1-REFERENCE]` | 保留整体布局，部分信息需按真实 Core 数据收敛 |
| `novel_studio_2` | Project Overview | `[V1-REFERENCE]` | 主工作台视觉基准，但部分指标和 Pipeline 文案不能直接实现 |
| `novel_studio_6` | 旧版 Chapter Editor / Unsynced | `[V1-NEEDS-FIX]` | 当前截图不可作为开发基准 |
| `novel_studio_7` | Continuity / Causality Live Sync | `[V2-V3-CONCEPT]` | V1 完全不实现 |
| `synced_idle` | Chapter Editor — Synced / Idle | `[V1-FINAL]` | 正式 Idle 编辑器视觉基准 |
| `writer_running` | Chapter Editor — Writer Running | `[V1-FINAL]` | 正式 Running 编辑器视觉基准 |
| `provider_novel_studio` | Provider / Model Settings | `[V1-REFERENCE]` | 页面需要，但字段必须动态来源于 ainovel config |

---

# 4. `synced_idle` — `[V1-FINAL]`

这是当前 V1 完成度最高的页面之一。

## 已符合要求

- 顶部状态为 `Idle`
- 当前正文状态为 `Synced`
- 只有 `[继续创作]`
- 正文可以人工编辑
- 右侧明确显示章节状态
- Steer 保持为工作台指令，而不是 Chat
- 底部状态栏正确显示 Idle / Last Chapter / Cost
- Volume / Arc / Chapter 清晰可见

## V1 开发原则

可以直接作为：

> **章节编辑器 Idle 状态的正式视觉基准。**

开发时数据必须来自 Core / Bridge，不硬编码截图中的具体数值。

---

# 5. `writer_running` — `[V1-FINAL]`

这一版已经符合我们对 Writer Running 的核心要求。

## 已符合要求

- 顶部正确显示 `Running - Writer`
- 主操作只有 Pause / Stop
- 不再显示 Continue
- 正文区域明确显示“只读锁定”
- 提示暂停后才能人工编辑
- 当前 Step 为 `draft_chapter 4/6`
- Writer Pipeline 进度明确
- Token / Cost / Elapsed 可见
- Steer 在运行过程中保持可用
- 章节正文仍然是视觉中心

## V1 开发原则

可作为：

> **Writer Running 状态正式视觉基准。**

---

# 6. `novel_studio_6` — `[V1-NEEDS-FIX]`

当前截图仍属于旧设计。

## 当前问题

截图同时出现：

```text
继续创作
暂停
停止
Agent 运行中
存在本地改动待同步
AI 协同续写中
```

这在状态逻辑上不成立。

如果已经存在人工修改尚未 Sync：

```text
Engine = Waiting Sync
```

就不应该仍然处于 Writer Running。

## V1 正确状态

Unsynced 页面应该表现：

```text
状态：Waiting Sync

⚠ 已保存 · 尚未同步

当前存在尚未同步的人工修改。
请先完成 Sync，再继续创作。

[立即同步]

[继续创作] Disabled
[生成下一章] Disabled
```

并且正文必须处于：

```text
人工可编辑
AI 不运行
```

## 当前处理

保留目录，不删除。

但开发阶段明确标记：

> **DO NOT IMPLEMENT FROM CURRENT SCREENSHOT**

后续可以直接基于 `synced_idle` 页面派生一个 Waiting Sync 状态，而无需重新设计整个页面。

---

# 7. `novel_studio_1` — `[V1-REFERENCE]`

目前 Welcome / Projects 的整体方向已经可以接受。

## 可以保留

- 打开本地项目
- 新建小说
- 快速开始
- 共创规划
- 导入已有小说
- 最近项目
- Core 状态
- Provider 状态
- 项目状态标签

## 注意

以下内容只属于 mock / presentation，不应该直接变成新的后端需求：

- CLI 常用指令对照区域
- 特定 Provider 延迟数字
- 固定 Token 成本
- 某个模型固定绑定
- 项目 Card 中的高级状态诊断

这些数据：

> Core 有则显示，Core 没有则不造新能力。

## V1 结论

页面结构：

> ✅ 可以采用。

业务字段：

> ⚠ 由 Backend API 决定。

---

# 8. `novel_studio_2` — `[V1-REFERENCE]`

这是 V1 Project Overview 的主要视觉基准。

## 可以保留

- 当前小说
- Chapter / Target Chapter
- Word Count
- Volume / Arc
- Current Agent
- 当前章节卡片
- 卷与章节推进
- 最近协作活动
- Runtime Pipeline 概览
- Steer / Pause / Sync 入口
- 底部 Agent / Step / Token / Cost

## 不要直接实现的 mock 指标

当前页面存在：

```text
伏笔与连续性健康度 99.8%
0 剧情冲突
12 名主角状态网络追踪
Architect / Arbiter 在线监护
```

这些不能作为 V1 固定指标。

原则：

```text
如果 ainovel Core 没有对应结构化数据，
V1 就隐藏该组件或替换成真实数据。
```

## Pipeline 文案需要修正

当前页面出现类似：

```text
Step 5: Arbiter 仲裁校验 & 伏笔入库
```

V1 Writer 标准 Pipeline 应以 Core 实际 Step 为准：

```text
1. novel_context
2. read_chapter
3. plan_chapter
4. draft_chapter
5. check_consistency
6. commit_chapter
```

不要因为 UI 设计存在一个 Step，就去修改 Core 配合 UI。

---

# 9. `provider_novel_studio` — `[V1-REFERENCE]`

页面视觉和信息密度可以保留。

Provider / Model 设置也是 V1 必需页面。

但是：

> 当前截图是“视觉 Mock”，不是 Backend Schema。

## 可以保留的页面结构

### Provider 列表

```text
Provider
API Key
Base URL
Models
Connection Status
Test Connection
```

### Agent Routing

根据 Core 实际支持的配置展示：

```text
Architect
Writer
Editor
```

其他 Agent 只有 Core 支持独立配置时才显示。

### Budget

如果 ainovel 配置支持：

```text
Book Budget
Warn Ratio
Hard Stop
```

即可映射。

## 不得视为固定产品能力

截图中的：

```text
Anthropic / DeepSeek / OpenAI 必须同时存在
固定 Endpoint
100% 成功率
固定请求并发限制
固定 Timeout
固定 Thinking Token Cap
固定角色绑定
```

全部只是 mock。

实际 UI 由：

```text
ainovel config
```

动态生成。

---

# 10. `novel_studio_7` — `[V2-V3-CONCEPT]`

这一页设计内容包括：

- Continuity & Entity Inspector
- 因果图谱
- 角色快照
- 伏笔实时关联
- Rule Constraint
- 因果校验
- 状态向量 / 风险指标
- 实时关系追踪

这些与我们未来讨论的：

```text
Fact
Character Knowledge
Dependency Graph
ContextManifest
Impact Analysis
Continuity Engine
```

高度相关。

因此：

> **设计本身有价值，保留。**

但：

> **V1 完全不实现。**

开发人员不得根据该页增加新的 ainovel Core 需求。

---

# 11. Runtime Center 页面状态

V1 必须拥有：

```text
Runtime Center — Running
Runtime Center — Paused
Runtime Center — Error
```

上一轮 Stitch 已经生成过较合理的版本。

当前这份最新导出包中没有包含这些页面。

因此：

## 当前判断

视觉设计方向：

> ✅ 已经确定。

当前 ZIP 完整性：

> ⚠ 不完整。

## 开发处理

可以继续使用上一轮 Runtime Center 设计作为 V1 Reference：

- Running
- Paused
- Error

后续无需重新发明 Runtime Center。

开发时遵循统一的 Engine State Model：

```text
Idle
Running
Paused
Waiting Review
Waiting Sync
Error
```

---

# 12. Review Center 页面状态

V1 需要 Review Center。

上一轮已经存在设计，但其中很多自行生成的质量分数不属于 V1。

因此 Review Center 标记：

> `[V1-REFERENCE]`

V1 实际展示：

```text
Chapter
Review Status
Issue Category
Severity
Description
Evidence（Core 有则显示）
Suggestion（Core 有则显示）
Rewrite / Polish Status
```

不固定实现：

```text
总评分
Radar Chart
99.8% 逻辑一致性
人物偏差率
自创质量指数
自动 Repair
```

---

# 13. 当前 V1 页面开发优先级

## P0 — 正式开发基准

```text
synced_idle
writer_running
novel_studio_1
novel_studio_2
Runtime Center（上一轮版本）
```

其中：

```text
synced_idle
writer_running
```

属于最接近 Final 的页面。

---

## P1 — 需要做，但按 Core 数据重构

```text
provider_novel_studio
Review Center
Master Outline / Volume / Arc
Unsynced / Waiting Sync
```

---

## Future

```text
novel_studio_7
```

以及其他历史高级页面：

```text
人物高级关系图
世界规则冲突图
伏笔因果图
Evidence Diff
高级连续性图
```

全部保留为未来概念稿。

---

# 14. V1 统一状态模型

所有页面开发必须共享同一个状态模型：

```text
Idle
Running
Paused
WaitingReview
WaitingSync
Error
```

对应操作：

| Engine State | 主操作 |
|---|---|
| Idle | Continue |
| Running | Pause / Stop |
| Paused | Resume / Stop |
| WaitingReview | Approve & Next / Stop |
| WaitingSync | Sync |
| Error | Retry / View Error / Stop |

禁止不同页面自行解释 Engine State。

---

# 15. Chapter 状态模型

正文另有独立状态：

```text
Clean
Modified
SavedUnsynced
Syncing
Synced
Generating
ReviewPending
Error
```

其中最关键的是：

```text
SavedUnsynced
        ↓
Engine = WaitingSync
```

此时：

```text
Continue = Disabled
Next = Disabled
```

只有：

```text
Sync Success
```

之后才恢复运行资格。

---

# 16. V1 视觉设计是否已经可以冻结？

## 可以冻结的部分

### Design System
✅ 可以冻结

### 主 Shell
✅ 可以冻结

### Sidebar / Volume / Arc / Chapter
✅ 可以冻结

### Chapter Editor 基础视觉
✅ 可以冻结

### Running 只读状态
✅ 可以冻结

### Steer 面板
✅ 可以冻结

### Bottom Runtime Telemetry
✅ 可以冻结

---

## 还不能视为最终业务规格的部分

### Provider 具体字段
⚠ 以后端 config 为准

### Overview 健康度 / 冲突百分比
⚠ Mock，不属于固定 V1 功能

### Review 分数体系
⚠ Mock，不属于固定 V1 功能

### Unsynced 当前截图
❌ 旧版，不作为开发基准

### Continuity Live Sync
🚫 V2/V3

---

# 17. 最终判断

当前设计：

## 视觉层面

> **符合 V1 要求，可以停止继续探索新的视觉方向。**

## 信息架构层面

> **基本符合。**

## 状态逻辑层面

> **Synced Idle 与 Writer Running 已符合；Waiting Sync 仍需按本文规则实现。**

## 功能边界层面

> **需要严格区分 V1 Reference 与 V2/V3 Concept，不能把 ZIP 中所有东西都实现。**

## 是否可以开始开发？

> **可以。**

前提是开发团队遵守本标注文档，而不是将每个 Stitch Mock 字段都当成必须实现的 Backend 功能。

---

# 18. 从现在开始的原则

UI 设计阶段到这里可以结束。

后续如果发现：

```text
Core 没有某字段
```

处理顺序应该是：

```text
1. 判断该字段是不是 ainovel 原功能所必需
2. 如果不是 → UI 隐藏/替换
3. 如果是 → Bridge 暴露已有 Core 数据
4. 只有确认 Core 本身缺能力时，才考虑修改 Core
```

不要因为 Stitch 画出了一个组件，就自动增加后端业务。

---

# 19. 当前推荐开发基线

```text
Novel Studio V1

Visual Baseline:
    synced_idle
    writer_running
    novel_studio_1
    novel_studio_2
    previous Runtime Center

Reference Only:
    provider_novel_studio
    Review Center
    master outline

Needs State Fix:
    novel_studio_6

Future:
    novel_studio_7
    other advanced graph / diff concepts
```

到此 V1 UI 可以进入“设计冻结（Design Freeze）”状态。
