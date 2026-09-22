# Novel Studio V1 — Stitch 最终收尾修改清单

> 目标：第二版已经基本确定视觉方向。本轮**不要重做页面**，只修正状态一致性、删除仍残留的超规格能力，并整理最终交付目录。完成本轮后即可停止 UI 设计，进入 React + Wails 开发。

---

## 1. 全局 Shell 的运行状态必须真正联动

目前部分新页面主体已经正确，但顶部全局 Shell 仍残留同一套固定状态，例如：

- `Running - Writer`
- `1个未同步章节`
- `继续创作 / 暂停 / 停止`

导致 `Synced / Idle`、`Paused`、`Error` 页面顶部与页面主体互相矛盾。

请将顶部 Shell 作为同一个“状态驱动组件”，按以下规则输出。

### Idle + Synced

显示：

```text
状态：Idle
[继续创作]
```

不要显示：

```text
Running - Writer
1个未同步章节
暂停
停止
```

### Running

显示：

```text
状态：Running · Writer
[暂停] [停止]
```

不要显示：

```text
[继续创作]
```

Running 时原则上不能同时存在 `Unsynced`，因为 Unsynced 会阻断 Engine 继续运行。

### Paused

显示：

```text
状态：Paused
[恢复创作] [停止]
```

不要继续显示：

```text
Running - Writer
[继续创作] [暂停]
```

### Waiting Sync

显示：

```text
状态：Waiting Sync
[立即同步]
```

并禁用：

```text
继续创作
生成下一章
```

### Error

显示：

```text
状态：Error
[查看错误] [重试] [停止]
```

不要继续显示 `Running - Writer`。

---

## 2. `synced_idle` 页面修正

当前页面主体已经正确显示：

```text
已同步 (Synced)
就绪 (Idle)
```

但顶部仍显示：

```text
Running - Writer
1个未同步章节
```

必须删除这一矛盾。

最终应统一为：

```text
已同步
Idle
[继续创作]
Core Ready
```

底部状态栏也不能再显示：

```text
Agent: Writer
Step: draft_chapter 4/6
```

Idle 时建议：

```text
Idle
Last: Chapter 014 completed
Project Cost: $14.28
Logs
```

---

## 3. `writer_running` 页面修正

正文只读锁定设计已经正确，继续保留。

需要修正顶部 Shell：

当前仍出现：

```text
[继续创作]
[暂停]
[停止]
1个未同步章节
```

Running 正确状态应为：

```text
Writer Running
[暂停]
[停止]
```

不要出现：

```text
继续创作
未同步章节
```

如果确实存在其他章节的未同步人工修改，则 Engine 本身应处于 Waiting Sync，而不是 Running。

---

## 4. Unsynced 页面以代码版本为准，并重新生成截图

`novel_studio_6/code.html` 已经正确实现：

- `[立即同步]`
- `继续创作 Disabled`
- `进入下一章 Disabled`
- `已保存 · 尚未同步`
- `Waiting Sync`

这是正确逻辑。

但当前 `screen.png` 与 `code.html` 内容不一致，截图仍显示旧状态。

请重新生成截图，确保：

```text
screen.png == code.html 当前状态
```

最终页面必须明确显示：

```text
⚠ 已保存 · 尚未同步

当前存在尚未同步的人工修改。
请先同步修改，再继续创作。

[立即同步]

[继续创作] Disabled
[进入下一章] Disabled
```

---

## 5. Paused 后人工编辑的流程必须增加 Sync

当前 Paused Runtime 页面写了类似：

> 暂停后可手动编辑，编辑后点击“恢复创作”即可继续。

V1 不采用这个逻辑。

如果用户暂停后修改正文，流程必须是：

```text
Pause
→ 手动编辑
→ Save
→ Waiting Sync
→ Sync
→ Resume
```

因此 Paused 页面可以写：

> 当前流水线已安全暂停，可以打开章节编辑器进行人工修改。若修改正文，保存后必须先完成 Sync，才能恢复创作。

不能写：

> 修改后直接 Resume。

---

## 6. Runtime Center 基本通过，作为标准页面保留

以下三个页面整体方向正确：

- `multi_agent_runtime_running`
- `multi_agent_runtime_paused`
- `multi_agent_runtime_error`

尤其保留：

- 当前 Agent
- 当前 Chapter
- 6-step Pipeline
- Token / Cost
- Agent 状态矩阵
- Runtime Log
- Pause / Resume / Stop
- Error Recovery

但它们的顶部 Shell 同样必须遵守第 1 条的状态联动规则。

### Error 页额外要求

“切换备用模型”“等待 xx 秒重试”等操作：

- 只有 Core / Provider 实际返回对应能力时才显示；
- 没有配置 fallback 时不要显示“切换备用模型”；
- `Retry-After` 只有 Provider 返回时才显示具体秒数。

不要让 UI 自己推导恢复策略。

---

## 7. Welcome 页面仍然没有按 V1 收敛，需要继续修改

当前 `novel_studio_1` 仍包含：

```text
Templates
Diagnostics
Narrative Models
LLM Runtime
ChromaDB
本地矢量化
大纲蓝图 Graph
高级 DevOps / Runtime 信息
```

这些仍不属于 V1 Welcome。

请收敛成：

```text
Novel Studio

[新建小说]
[打开本地项目]
[导入已有小说]

创建方式：
- 快速开始
- 共创规划
- 从大纲创建
- 导入已有小说

最近项目

系统状态：
- Core Ready
- Provider Ready / Not Configured
- 最近错误（如有）
```

### 必须删除

```text
Templates
Diagnostics
Narrative Models
ChromaDB
Vector Database
大纲 Graph 入口
RAM / VRAM
远程 Workspace
Git Clone
云端同步
```

`Import` 可以存在，但不要写：

```text
自动构建 ChromaDB
本地向量数据库
```

V1 Import 只是 ainovel-cli 现有 Import Pipeline 的 GUI。

---

## 8. Main Workspace / Overview 继续收敛到 Core 真实数据

当前 `novel_studio_2` 的整体布局可以保留。

但以下内容不要自行发明：

```text
伏笔与连续性健康度 99.8%
0 剧情冲突
12 名主角状态网络追踪
Arbiter 在线监护
```

只有 Core 当前真的提供这些指标时才能显示。

推荐 Overview 使用明确可获取的数据：

```text
当前 Chapter
目标 Chapter
总字数
当前 Volume
当前 Arc
Engine Status
Current Agent
Current Step
Review Pending
Unsynced Count
Input Tokens
Output Tokens
Cost
最近活动
```

### Pipeline 必须使用真实步骤

Writer Pipeline：

```text
1. novel_context
2. read_chapter
3. plan_chapter
4. draft_chapter
5. check_consistency
6. commit_chapter
```

不要把：

```text
Arbiter 仲裁校验
伏笔入库
```

擅自替换成固定 Pipeline Step。

---

## 9. Review Center 仍然严重超规格，必须再收敛

当前页面仍出现：

```text
总评分 88
Pass 92分
Pass 95分
节奏张力指数 9.4/10
逻辑一致性 99.8%
人设偏差率 <0.2%
伏笔锚定 3/4
Arbiter-v4
Standard v4
重新全书仲裁
```

这些都不能作为 V1 固定 UI 指标。

V1 Review Center 只展示 Core 已有字段。

建议最终结构：

```text
审稿中心

Chapter 014
状态：Reviewing / Pass / Rewriting / Polishing / Done / Error

问题列表

问题 1
类别：
严重程度：
说明：
证据：      （Core 有则显示）
建议：      （Core 有则显示）
状态：

问题 2
...
```

可显示：

```text
已审稿章节数
待处理章节数
当前 Review 状态
Rewrite / Polish 状态
```

不要自行创建综合评分算法、百分比和雷达图。

---

## 10. Provider / Model Settings 再收敛一次

视觉布局可以保留。

当前页面仍把很多示例值写成产品固定能力，例如：

```text
Anthropic / DeepSeek / OpenAI 固定三组
固定角色绑定
固定 Endpoint
100% 成功率
Thinking Token Cap
并发限制 5 REQ
固定 Hard Stop UI
```

最终设计应该体现“动态配置”。

### Provider Card

字段：

```text
Provider Name
API Type
API Key
Base URL
Models
Connection Status
```

这些由 ainovel-cli config 提供。

不要把：

```text
https://api.anthropic.com/v1
https://api.deepseek.com/v1
https://api.openai.com/v1
```

设计成不可变产品结构。

它们最多只能作为 mock 数据示例。

### Agent Routing

只展示 Core 实际支持的角色：

```text
Architect
Writer
Editor
```

若 Arbiter 的模型不能独立配置，则不要提供 Arbiter 独立模型绑定。

### 高级设置

只有 Core 有配置字段时才显示：

```text
Reasoning Effort
Context Window
Fallback
Budget
Warn Ratio
Notify
```

---

## 11. 禁止使用 Cloud 图标表达本地 Sync

部分页面仍使用：

```text
cloud_done
章节已同步
```

V1 不存在云同步。

请改用：

```text
check_circle
sync
verified
```

文案：

```text
已同步
Core 已接纳
本地项目正常
Store Ready
```

不要出现 Cloud / 云端语义。

---

## 12. 清理 ZIP：不要同时交付“旧页面 + 新页面”

当前 ZIP 同时包含：

```text
novel_studio_1
novel_studio_2
...
novel_studio_7
```

以及：

```text
synced_idle
writer_running
multi_agent_runtime_running
multi_agent_runtime_paused
multi_agent_runtime_error
...
```

其中旧页面仍包含大量 V2/V3 功能和旧状态，会导致开发阶段无法判断哪个才是最终设计。

最终交付请整理为：

```text
v1_final/
├── welcome_projects/
├── project_overview/
├── chapter_synced_idle/
├── chapter_unsynced/
├── chapter_writer_running/
├── runtime_running/
├── runtime_paused/
├── runtime_error/
├── review_center/
└── provider_settings/
```

旧设计如果要保留，请放：

```text
_archive/
```

或：

```text
_future_v2_v3/
```

不要与 V1 Final 混在同一级目录。

---

## 13. 最终截图和 HTML 必须一一对应

当前至少存在：

```text
code.html 已更新
screen.png 仍是旧设计
```

的情况。

最终交付要求：

```text
每一个页面目录：
code.html
screen.png

二者必须是同一个版本。
```

开发人员会同时参考截图和代码，不允许出现内容不一致。

---

# 最终验收标准

完成本轮后，V1 设计只需要回答 5 个问题：

```text
1. 当前小说在哪里？
   Volume / Arc / Chapter

2. Engine 当前是什么状态？
   Idle / Running / Paused / Waiting Review / Waiting Sync / Error

3. 当前哪个 Agent 在干什么？
   Agent / Step / Runtime

4. 作者现在能做什么？
   Continue / Pause / Resume / Stop / Review / Steer / Edit / Sync

5. 为什么某个操作现在不能做？
   清晰的阻断状态和说明
```

如果这五点在任何页面都保持一致，V1 UI 就可以冻结。

完成本轮后：

> **停止继续扩展页面，正式进入 React + TypeScript + Wails 开发。**
