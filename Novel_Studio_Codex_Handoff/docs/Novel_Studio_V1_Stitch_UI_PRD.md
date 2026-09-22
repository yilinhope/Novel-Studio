# Novel Studio V1 — Stitch UI 产品需求书

> 版本：V1.0  
> 定位：`ainovel-cli` 的桌面可视化版本  
> 核心原则：**V1 只做 UI 与交互映射，不主动改变 ainovel-cli 的核心创作逻辑。**

---

## 1. 产品目标

Novel Studio V1 的唯一目标：

**用户完全不打开终端，仅使用桌面 GUI，就能够完成当前 ainovel-cli 已支持的完整小说创作流程。**

V1 不是重新设计小说生成引擎，也不是引入新的 Fact / Proposal / Dependency Graph 体系。

V1 的产品定义：

- Core = ainovel-cli
- CLI/TUI = 基准实现
- Novel Studio GUI = 新表现层
- GUI 与 CLI 行为不一致时，默认视为 GUI Bug，除非后续明确修改 Core

---

## 2. 产品定位

这是一个面向长篇小说生成的 AI 创作工作台。

与普通 AI 写作软件的区别：

- 具备自动推进整本小说的 Engine
- 有 Volume → Arc → Chapter 的长篇结构
- Architect / Writer / Editor / Arbiter 多角色协作
- 可以持续自动创作
- 可以逐章验收
- 可以在创作过程中实时 Steer
- 可以人工修改正文后 Sync 回系统
- 可以查看审稿、状态、伏笔、时间线、人物关系等长期信息
- 可以配置多个模型和不同 Agent 使用不同模型

视觉上应更接近“专业创作 IDE / 小说工作台”，而不是聊天机器人。

---

## 3. UI 设计参考

可以参考 AI-Novel-Writer 的桌面应用：

- 左侧项目树
- 中央主要编辑工作区
- 右侧 AI / 属性 / 上下文面板
- 底部运行状态

但不要直接复制其信息架构。

Novel Studio 必须突出：

**Volume → Arc → Chapter → Agent → Step**

这是 ainovel-cli 的核心工作方式。

---

# 4. 全局布局

建议整体使用固定工作台结构：

```text
┌───────────────────────────────────────────────────────────────┐
│ 顶部工具栏                                                    │
├──────────────┬─────────────────────────────┬──────────────────┤
│              │                             │                  │
│ 左侧项目树   │          主工作区           │   右侧上下文     │
│              │                             │   AI / 属性      │
│              │                             │                  │
├──────────────┴─────────────────────────────┴──────────────────┤
│ 底部运行状态 / Agent / Token / Cost / 日志                   │
└───────────────────────────────────────────────────────────────┘
```

### 顶部工具栏必须包含

- 当前项目名
- 当前章节
- 当前 Volume / Arc（空间允许时）
- 开始 / 继续
- 暂停
- 停止
- 全局状态提示
- 设置入口

### 左侧项目树

承载项目导航和小说层级。

### 中央区域

用于：
- 正文编辑
- 故事设定
- 世界观
- 人物
- 大纲
- Review
- Timeline
- Foreshadowing
- 运行详情等

### 右侧区域

根据当前页面切换：
- 当前对象属性
- AI 操作
- Agent 状态
- 审稿问题
- 当前章节状态
- 上下文摘要

### 底部状态栏

必须始终可见：

- 当前 Agent
- 当前 Step
- 当前 Chapter
- 运行时间
- Input Tokens
- Output Tokens
- Cost
- 日志入口

---

# 5. 左侧项目树

推荐结构：

```text
▾ 项目
   ◉ 总览

▾ 故事
   ◇ 故事前提
   ♙ 人物
   ◎ 世界观
   ▤ 全书大纲

▾ 长篇规划
   ▾ 第一卷
      ▾ Arc 01
         第001章
         第002章
         第003章
      ▸ Arc 02
   ▸ 第二卷

▾ 连续性
   ⤴ 伏笔
   ◷ 时间线
   ⇆ 人物关系
   ♙ 人物状态

▾ 质量
   ✓ 审稿
   ↻ 重写 / 润色

▾ 工具
   ✦ 参考作品
   ↓ 小说导入
   ⟳ 人工修改同步

▾ 系统
   ⚙ 模型与 Provider
   ◫ Token / 成本
   ≡ 运行日志
```

要求：

- Volume / Arc / Chapter 必须支持折叠
- 当前章节要有明显高亮
- 当前 Arc / Volume 要可识别
- 未同步章节应显示警告状态
- 等待 Review 的章节应显示状态
- 正在生成的章节应显示 Running 状态
- 出错章节应有 Error 状态

---

# 6. 页面 1：欢迎 / 项目选择

## 目标

让用户进入已有项目或创建新项目。

## 内容

### 主操作

- 新建小说
- 打开已有项目
- 导入已有小说

### 最近项目

每张卡片显示：

- 小说名称
- 项目路径
- 当前章节
- 最近打开时间
- 当前状态

状态示例：

- 已完成
- 创作中
- 暂停
- 等待审核
- 存在未同步修改

## 空状态

首次使用时提供明显的新建入口。

---

# 7. 页面 2：新建小说

支持三种创建方式：

## A. 快速开始

输入：

- 一句话小说需求
- 可选题材
- 可选目标章节数
- 可选风格

主按钮：

**开始创作**

## B. 共创规划

由 AI 和用户逐步确认：

- 故事方向
- 主角
- 世界观
- 核心冲突
- 长期目标

UI 应表现为结构化向导，不要设计成普通聊天页。

## C. 从已有内容开始

入口：

- 从大纲文件创建
- 导入已有小说

---

# 8. 页面 3：项目总览

## 目标

一眼看清整本小说当前状态。

## 信息区

### 小说进度

- 当前 Chapter
- 目标 Chapter（如有）
- 总字数
- 已完成字数
- 当前 Volume
- 当前 Arc

### Engine 状态

- Running
- Paused
- Waiting Review
- Waiting Sync
- Error
- Idle

### 当前 Agent

- Architect
- Writer
- Editor
- Arbiter

### 当前 Step

例如 Writer：

- novel_context
- read_chapter
- plan_chapter
- draft_chapter
- check_consistency
- commit_chapter

### 最近活动

- 最近生成章节
- 最近 Review
- 最近 Steer
- 最近 Sync

### 当前警告

例如：

- 存在 2 个未同步章节
- 当前章节等待审核
- API 配置异常
- 预算接近阈值

---

# 9. 页面 4：章节编辑器

这是 V1 最重要的页面。

## 主区域顶部

显示：

- 第 XX 章
- 章节标题
- 所属 Volume
- 所属 Arc
- 当前状态

状态：

- Saved
- Modified
- Unsynced
- Syncing
- Generating
- Waiting Review
- Error

## Tab

- 蓝图
- 正文
- 审稿
- 状态

## 正文 Tab

需要成熟的长文本编辑体验：

- Markdown / 纯文本编辑
- 字数统计
- 保存
- 撤销 / 重做
- 当前保存状态

## 右侧章节操作

### 文件操作

- 保存

### 人工修订

- 检测人工修改
- 同步人工修改

### 自动创作

- 继续创作
- 生成下一章

### Review

- 开启逐章审核
- 关闭逐章审核

### AI 干预

- 输入 Steer 指令
- 发送

---

# 10. 章节编辑器关键状态规则

## Saved

文件与磁盘一致。

## Modified

编辑器存在未保存变化。

## Unsynced

正文文件已经发生人工修改，但 ainovel-cli 尚未接纳。

UI 必须：

- 明显提示
- 禁止继续生成
- 禁止“生成下一章”
- 提供“立即同步”

提示文案示例：

> 当前项目存在尚未同步的人工修改。请先同步修改，再继续创作。

## Syncing

显示同步进度。

## Waiting Review

当前章节已生成，逐章审核模式正在等待用户批准下一章。

按钮：

**批准并继续下一章**

---

# 11. 页面 5：Volume / Arc / Chapter Plan

## Volume 页面

显示：

- 卷名
- 卷目标
- 当前状态
- 包含的 Arc
- 卷摘要 / 规划信息

## Arc 页面

显示：

- Arc 名称
- Arc 目标
- 起止章节
- 当前进度
- Arc 摘要
- 包含章节

## Chapter Plan 页面

显示：

- 本章目标
- 关键事件
- 涉及角色
- 伏笔任务
- 节奏 / Hook 等已有数据

V1 以查看现有数据为主。

---

# 12. 页面 6：人物

## 人物列表

每个人物显示：

- 姓名
- 身份
- 简要描述
- 当前状态

## 人物详情

Tab：

- 基础设定
- 当前状态
- 关系
- 状态变化

### 基础设定

查看 ainovel-cli 已有角色信息。

### 当前状态

例如：

- 所在地点
- 当前身份
- 身体状态
- 情绪 / 阶段状态
- 其他现有快照字段

### 关系

显示当前关系数据。

V1 不要求高级关系图谱。

---

# 13. 页面 7：世界观

显示当前已有世界设定。

可以按类别折叠：

- 世界背景
- 力量体系
- 地理
- 阵营
- 社会规则
- 其他规则

V1 以现有数据展示为主，不重构 World 数据模型。

---

# 14. 页面 8：审稿中心

## 目标

把 Editor 当前产生的 Review 可视化。

## 章节审稿列表

显示：

- Chapter
- Review 状态
- 是否需要 Rewrite / Polish
- 最近更新时间

## Review 详情

显示 ainovel-cli 当前已有的各类审稿问题。

每个问题建议包含：

- 问题类别
- 严重程度
- 原因
- 正文证据（如果已有）
- 建议
- 当前处理状态

## 当前处理状态

- Pass
- Reviewing
- Rewriting
- Polishing
- Done
- Error

V1 不新增评分体系，只展示 Core 已有结果。

---

# 15. 页面 9：Steer / 人工干预

不需要独立主页面。

建议长期存在于右侧面板。

## 输入框

标题：

**告诉创作团队**

示例 Placeholder：

> 例如：让林晚晴暂时不要知道男主真实身份

## 提交后显示

- 已接收
- Arbiter 处理中
- 当前影响路径（如果 Core 已暴露）
- 处理完成 / 失败

V1 不自行判断影响范围。

---

# 16. 页面 10：Sync / 人工修改同步

## Sync 入口

可以从：

- 左侧“人工修改同步”
- 章节右侧操作
- 全局警告

进入。

## 检测界面

显示：

- 发生修改的章节数量
- 章节号
- 文件状态
- 当前 SHA 状态（不必展示 SHA 文本）
- 是否已同步

操作：

- 检测修改
- 同步全部修改

如果 Core 能提供更多分析进度，展示：

- 提取章节事实
- 更新摘要
- 更新时间线
- 更新伏笔
- 更新关系
- 更新人物状态
- 更新风格记忆
- 使旧 Review / Arc / Volume 摘要失效
- Architect 检查后续规划

但 V1 不自行实现这些逻辑。

---

# 17. 页面 11：参考作品 / Simulation

## 功能

- 添加参考目录 / 文件
- 运行 Simulation
- 查看画像状态
- 导入已有 Simulation Profile

## UI

显示：

- 当前参考文件数量
- 已分析数量
- 上次更新时间
- 当前画像是否最新

操作：

- 重新分析
- 导入画像

V1 不重做风格分析算法。

---

# 18. 页面 12：导入已有小说

对应现有 Import Pipeline。

UI 建议分阶段：

1. 选择文件
2. Ingest
3. 章节切分
4. 用户确认切分
5. Analyze
6. Synthesize
7. Publish
8. 完成

切分确认页需要：

- 章节列表
- 章节标题
- 顺序
- 重新识别 / 添加 Guide
- 确认继续

重点是把现有 CLI/TUI 导入过程可视化。

---

# 19. 页面 13：运行中心

这是 Novel Studio 与普通小说编辑器差异最大的页面之一。

## 当前运行

显示：

- 当前 Agent
- 当前 Chapter
- 当前任务
- 当前 Step
- 运行时间

## Agent 状态

- Architect
- Writer
- Editor
- Arbiter

状态：

- Idle
- Running
- Waiting
- Error

## Writer Step 示例

```text
✓ novel_context
✓ read_chapter
✓ plan_chapter
● draft_chapter
○ check_consistency
○ commit_chapter
```

## Usage

- Input Tokens
- Output Tokens
- Cost
- 本次运行时间
- 累计项目 Cost

## 日志入口

打开完整日志抽屉 / 页面。

---

# 20. 页面 14：模型与 Provider 设置

UI 必须忠实映射 ainovel-cli 当前配置结构。

## 基础模型

- 当前 Provider
- 当前 Model
- Reasoning Effort

## Provider 管理

字段根据当前 Core 配置能力：

- Provider 名称
- API 类型
- API Key
- Base URL
- Models

## Agent 模型

可分别设置：

- Architect
- Writer
- Editor

支持：

- Provider
- Model
- Reasoning Effort
- Fallbacks（如果 Core 暴露）

## 高级

- Context Window
- Style
- Budget
- Notify

不要在 V1 发明另一套配置格式。

---

# 21. 页面 15：预算 / Token

如果当前项目启用预算：

显示：

- 已花费
- Book Budget
- Warn Ratio
- 当前百分比
- 是否 Hard Stop

展示：

```text
$23.40 / $50.00
█████████░░░░░ 46.8%
```

同时显示：

- Input Tokens
- Output Tokens
- 各 Agent 使用情况（Core 有数据时）

---

# 22. 页面 16：日志

开发阶段非常重要。

## 功能

- 实时日志
- 日志级别筛选
- Agent 筛选
- Error 高亮
- 一键滚动到底部
- 清空视图（不删除原日志）

需要方便排查：

- GUI 调用失败
- Provider/API Error
- Engine Error
- Sync Error
- Import Error
- Review Error

---

# 23. 全局运行状态

整个应用必须统一使用状态模型。

建议视觉状态：

### Idle
正常。

### Running
显示当前 Agent / Step。

### Paused
明显显示已暂停，可恢复。

### Waiting Review
章节生成已完成，等待用户批准。

### Waiting Sync
存在人工修改，必须先同步。

### Error
显示错误摘要和日志入口。

---

# 24. 重要 UX 原则

## 1. 不要聊天化

Novel Studio 是创作 IDE，不是聊天机器人。

## 2. 正文永远是视觉中心

章节编辑页面不要被 AI 卡片挤压。

## 3. 自动运行状态始终可见

用户应该随时知道：

- 谁在运行
- 正在做什么
- 运行到哪一步

## 4. 危险状态不可隐藏

以下状态必须非常明显：

- 未同步正文
- API 失败
- 等待审核
- 预算达到阈值
- Engine Error

## 5. 人工操作与自动操作要区分

例如：

- 保存正文
- Sync
- Steer
- Next Chapter
- Continue Engine

视觉语义不要混淆。

---

# 25. Stitch 第一轮只设计 4 个页面

为了保持设计统一，第一轮不要生成全部页面。

请先输出：

## 页面 A：欢迎 / 项目选择

目的：
确认品牌、整体设计语言和桌面应用风格。

## 页面 B：主工作台 / 项目总览

目的：
确认顶部、左侧树、中央内容、右侧面板、底部状态栏的整体布局。

## 页面 C：章节编辑器

目的：
确认最核心的长时间工作场景。

必须展示：

- Volume / Arc / Chapter
- 正文编辑器
- 保存
- Unsynced 状态
- Sync
- Review
- Steer
- 当前 Agent

## 页面 D：运行中心

目的：
突出 ainovel-cli 的自动长篇创作特点。

必须展示：

- 当前 Agent
- Agent 列表
- 当前 Step
- 当前 Chapter
- Token
- Cost
- Logs
- Pause / Resume / Stop

第一轮确定视觉体系后，再扩展其他页面。

---

# 26. Stitch 设计风格要求

方向：

- 专业桌面创作工具
- IDE / Writing Studio 感
- 高信息密度但不过度拥挤
- 长时间使用舒适
- 默认考虑深色模式与浅色模式
- 不要娱乐化
- 不要“AI 聊天网站”视觉
- 不要大量营销卡片
- 不要过度圆角和巨大留白
- 编辑器区域尽可能安静
- 运行状态和告警应清晰

参考气质：

- 小说创作软件
- IDE
- 现代知识管理工具
- AI-Novel-Writer 的项目工作台结构

但不复制其具体 UI。

---

# 27. 桌面尺寸

优先设计：

- 1440×900
- 1920×1080

最低应能正常使用：

- 1280×720

不以手机端作为 V1 目标。

---

# 28. V1 暂不设计的功能

以下功能属于 V2 / V3，不要在 Stitch V1 页面中主动加入：

- Proposal Inbox
- 完整章节版本历史
- 高级 Diff
- Fact Engine
- Character Knowledge
- Dependency Graph
- ContextManifest
- Impact Analysis
- 自动 Repair
- 高级故事线管理
- 高级知识库
- 节点式 Agent Workflow
- 云同步
- 多人协作
- 小说发布平台

---

# 29. V1 验收目标

V1 UI 成功标准：

**用户不用打开终端，就能完成 ainovel-cli 当前完整日常创作流程。**

至少包括：

```text
新建项目
→ AI 规划
→ 自动生成正文
→ Editor 审稿
→ 连续创作
→ 暂停
→ 恢复
→ Steer
→ 人工编辑正文
→ Sync
→ 逐章审核
→ Next
→ Arc / Volume 正常推进
```

---

# 30. 给 Stitch 的最终指令

请先设计以下 4 个桌面页面：

1. Welcome / Projects
2. Main Workspace / Project Overview
3. Chapter Editor
4. Runtime Center

统一设计系统。

重点表现以下产品特征：

- 长篇小说创作
- Volume → Arc → Chapter 层级
- Architect / Writer / Editor / Arbiter 多 Agent
- 自动持续创作
- 人工可随时介入
- 正文人工修改后需要 Sync
- Review / Pause / Resume / Next
- Token / Cost / Runtime 状态

不要把产品设计成聊天机器人。

不要加入需求中没有的 V2/V3 功能。

第一轮重点确定：
- 信息架构
- 桌面工作台布局
- 导航
- 编辑器视觉
- Agent 运行状态视觉
- 异常 / Waiting / Unsynced 状态视觉

确认第一轮后，再继续扩展剩余页面。
