# M5-B/C 执行账本

- 基线：`ff2d28c7382e0f68b69bf1dc9ab2ef7ab852e203`（M5-A 已验收并冻结）。
- 工作区：隔离 worktree，分支 `codex/m5-review-steer-bc`。
- 范围决策：M5-B 与 M5-C 合为一个阶段、一次验收、一个最终提交；不把改动堆入仍待合并的 M5-A PR 分支。
- GitNexus：索引较 HEAD 落后 4 个提交；Core 方法影响中出现 UNKNOWN/receiver typing 缺口，需对调用文本与源码复核；未见 HIGH/CRITICAL 已确认风险。
- 生产代码当前未修改。

## 进度

- [x] Task 1：Core 边界与写入矩阵
- [x] Task 2：EngineService 与后端动作（测试先行）
- [x] Task 3：Bridge 互斥、代际隔离与刷新
- [x] Task 4：PendingSteer 与 Arbiter 事件投影
- [x] Task 5：前端控件与事件状态
- [ ] Task 6：整体验收、审查、单一提交（验证完成；待最终提交）

## 每任务记录

### Task 1

- 产物：`task-1.md`，记录 TUI/Core 事实、Idle/Completed 边界、锁边界、GitNexus 风险与索引陈旧情况。
- 测试：只读源码审计，无产品代码修改。
- 偏差：执行计划脚本是 Bash 文件，但当前 Windows Bash/WSL 无法访问其工作目录；按脚本契约手动建立同等 brief/账本记录，继续按 TDD 推进。

### Tasks 2–5

- EngineService 增加显式模式切换、Next、Steer；Steer Running 分流到 Host.Steer，允许恢复的非运行态分流到 Host.Continue(text)。拒绝空文本、WaitingSync、Pausing、Stopping、exclusive、co-create；Idle/Completed 按任务 1 中确认的 Store/Core phase 事实处理。
- Bridge 对模式切换、Next、Steer 复用项目写控制锁；Next 先读 revision/advance projection 拦截恢复门，再委托 Host.AdvanceOneChapter；成功操作后回读 Project/Review/Revision/Runtime，刷新错误单独作为 warning。
- Host Snapshot 向后兼容增加 co-create/exclusive 观测字段；Runtime 暴露 Store/Core 的真实 PendingSteer。Arbiter 失败时保留 pending 指令并发出既有错误 Event。
- 前端加入 Auto/Review、真实推进许可按钮及 Steer 输入；不乐观改写 Core 状态，过滤旧项目与旧代际事件。
- 定向 RED-GREEN：AdvanceOneChapter 调用被阻塞时，Runtime 保持 WaitingReview；Core 接受命令后才转为 Running。该测试先失败、再修正并通过。

### Task 6 验证与最终审查

- 通过：`go test ./...`、`go vet ./...`、Vitest（30/30）、Vite production build、Wails Windows production build（与 CI 相同的 Wails v2.15.0 命令）、`git diff --check`。
- race 检查未能执行：Go `-race` 需要 CGO；本机未发现 gcc/clang/clang-cl。记录为环境限制，不将其描述为通过。
- 自审结论：Strengths — 后端服务与 Bridge 双层门禁、Core/Store 权威刷新、测试覆盖用户列出的路由/门禁/恢复/陈旧事件；Issues — 无 Critical/Important/Minor 发现；Recommendations — 后续 Windows runner 可补 race-enabled CI（本阶段不扩范围）；Declined to judge — Proposal/高级 Diff/Fact Engine/Impact Analysis/自动 Repair/Replan/新评分体系，均按用户明确范围排除。
- 本次环境没有独立 reviewer/subagent 工具；最终审查由主执行者按完整 diff 自审，强度低于独立复核。
- Wails 构建清理过 `desktop/frontend/dist/.gitkeep`；已从 HEAD 原样恢复，未纳入改动。
- GitNexus staged：`changed_count=100`、18 个 changed files、8 个 affected processes、`risk_level=high`，未标记 partial/truncated。HIGH 主要来自 `Host.Snapshot` 的既有调用/生命周期流；其索引报告有 1 个 receiver-typing 缺口且索引落后 4 commits。调用方核查显示本次仅给 `UISnapshot` 追加可选 CoCreating/Exclusive 字段，既有 TUI 消费字段和 Engine 路由不变；`monitor` 相关流由全仓测试和 Wails 构建覆盖。该风险已记录并经人工核对，不以空调用集或 shared axes 降级。
