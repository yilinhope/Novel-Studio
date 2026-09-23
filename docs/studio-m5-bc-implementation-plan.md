# M5-B/C 合并实施计划

## 范围与决策

- 基于 M5-A 冻结提交 `ff2d28c`，在隔离分支一次实施 Review/Auto、Advance Gate/Next、Steer 与 Arbiter 状态反馈。
- B/C 为一个实施、验收和最终提交阶段；中间不发阶段验收，不拆 PR。
- 只复用 Core 的 Host/Arbiter/Engine 行为；Studio 负责 API、门禁、互斥、刷新和真实状态投影。
- 禁止 GUI 直接写 RunMeta、AdvancePermitChapter 或 ReviewEntry；不新增评分、Proposal、Fact Engine、高级 Diff、Impact Analysis、自动 Repair/Replan。
- 只有显式写操作能按需创建 Host；读取/检查仍不创建。项目级控制操作遵循现有 `projectMu`，Core Host 保留 `interMu`、exclusive 和项目写锁语义。
- 操作结果必须重新读取 Store/Host Snapshot；Core Done 才确认 Engine 生命周期终态。

## 执行任务

### 1. Core 边界与写入矩阵

- 阅读并记录 TUI 对 SetAdvanceMode、Next、Steer 的真实语义，明确 Idle/Completed 等状态。
- 追踪 EngineService、Bridge 项目锁、Runtime/Revision/Review 的事实来源和事件转发路径。
- 对 GitNexus 的 stale/UNKNOWN 影响结果以源码搜索确认，不改 Core 路由。
- 验收：计划/发现记录覆盖用户指定门禁和互斥；无生产代码变更。

### 2. Studio EngineService 与后端动作

- 新增 `SetAdvanceMode`、`AdvanceOneChapter`、`SubmitSteer` 服务动作，明确项目、运行态、Revision、exclusive、co-create 检查。
- Running 仅走 `Host.Steer(text)`；Paused/WaitingReview 及经 Core 事实证明可恢复的写作态走 `Host.Continue(text)`；Idle/Completed 在矩阵确认后实现。
- Host 只由显式动作按需创建；创建 Host 不得隐式 Resume，切模式不启动 Engine。
- 测试先失败后实现：模式持久化、未启动 Engine、Next gate、Steer 分流/拒绝/空文本、pending recovery/error。

### 3. Bridge 互斥、项目代际与权威刷新

- 三种动作与 Sync/Project Switch 在现有项目控制边界内串行化，避免并发切换污染。
- 成功后重读 Host Snapshot、Runtime、Project/Tree、Review/Advance projection；不得前端乐观改成功。
- 新增 stale project/generation 的后端保护与定向测试；失败保持 Core 状态。

### 4. Arbiter 反馈事件与 Runtime 投影

- Runtime 投影真实 `PendingSteer`，不合成排队/应用状态或百分比。
- 复用 Host 已有开始/完成/失败 Event；只在现有字段无法区分时添加向后兼容、纯观测可选字段。
- Arbiter 失败后 PendingSteer 必须继续来自 Store/Core，测试其未丢失。

### 5. 前端控件和事件反馈

- 增加 Auto/Review 显式模式切换、满足真实 gate 才可用的“继续下一章 / 允许继续创作”、Steer 输入和 Arbiter 真实事件状态。
- 后端门禁为安全边界；前端 disabled 仅是投影。
- 接受响应后刷新权威 Store；测试模式切换、Next/Steer 调用、事件代际过滤和显示文本。

### 6. 全量验证、影响复核、最终审查与单一提交

- 执行相关 Go、前端测试与构建、`go vet`、Wails Windows production build（环境可用时）。
- 执行 GitNexus staged/all 变更分析，若 partial/truncated 则重试并报告。
- 对整个 B/C 最终变更执行一次整体审查，修复发现后复验。
- 只有全阶段验收后创建一个合并阶段提交；不推送、不创建 PR，除非用户另行明确要求。

## 验收矩阵

- Auto→Review，Review→Auto；切换不 Resume。
- WaitingReview→Next→Engine 启动；WaitingSync、AdvanceHold、PendingCommit、PendingRewrite 拒绝；ReviewEntry 不变。
- Running→Steer 调 `Host.Steer`；Paused/WaitingReview→Steer 调 `Host.Continue`；WaitingSync/Pausing/Stopping、exclusive、co-create、空文本拒绝。
- Idle/Completed 的 Steer 判定与 Core/TUI 实际行为一致，不猜测。
- PendingSteer recovery 与 Arbiter error 保留；旧项目/旧代际事件不污染当前项目。
- Runtime、Review、Advance Gate 最终事实均由 Store/Host Snapshot 派生。
