# Task 1 — Core 边界与写入矩阵

## 发现

- TUI `Model.handleEnterKey`：modeRunning 且 `snapshot.IsRunning` 时走 `Host.Steer`；否则走 `Host.Continue(text)`。modeDone 输入也走 `Host.Continue(text)`，并非 Resume。
- `Host.Continue` 拒绝空文本、co-create、exclusive、revision dirty 和 budget 超限；之后经 Arbiter 干预，允许 Core 按决定继续 Engine。
- TUI 对 Idle 可恢复写作态的输入使用 Continue；已完成书仍可在 done 工作台输入修订/续写意图，故 Studio 的 Completed 采用 Continue 语义。新空项目没有 TUI 对应的“Steer”入口，Studio Idle 仅允许确认 Store 为 writing 且确有可恢复内容后 Continue，否则拒绝。
- `Host.AdvanceOneChapter` 自己检查 running/Engine active、co-create、exclusive、revision、Review mode、AdvanceHold、budget、writing phase、NextChapter，并由它落 permit 后启动 Engine。Studio 不补写 Permit/ReviewEntry。
- `Host.SetAdvanceMode` 经 Host `interMu` 写 RunMeta，不启动 Engine；Studio 仅显式调用它。
- EngineService `controlMu` 已用于 Resume/Sync/Pause/Stop/Project Switch；Bridge `projectMu` 用于 Sync/Save/Project Switch，新的三个写操作必须全用 projectMu 独占锁，避免与 OpenProject/Sync 交错。
- Host Snapshot 已包含真实 `PendingSteer`，但 Studio Runtime 尚未投影此字段。Arbiter 错误事件真实为 agent=arbiter 的 ERROR/模型生命周期事件。
- Core `doIntervention` 的 Arbiter 返回错误分支当前先发失败事件，随后清除 PendingSteer；与本阶段“Arbiter error 不丢 PendingSteer”验收冲突，需仅保留该失败输入供后续恢复，其他成功清除/动作语义不变。
- Host UISnapshot 未暴露 co-create/exclusive 标志；Studio 对运行中路径走 `Host.Steer` 前需要可读事实做后端拒绝。`acquireExclusive` 要求 Engine 非运行，co-create 会暂停 Engine，需仍检查 Core 快照的显式标志以封住边界。

## GitNexus 影响

- `Host.Continue` LOW / exact；既有 TUI 与 Host 内部调用。仅复用。
- `Host.Steer` LOW / exact；TUI 一个直接 caller。仅复用。
- `Host.AdvanceOneChapter` UNKNOWN/lower-bound，索引漏掉 1 个 receiver-typing 调用点；源码已由 TUI `commands.go`、Studio Bridge 确认。
- `Host.SetAdvanceMode` UNKNOWN/lower-bound；真实调用点由 `rg` 核实在 TUI `commands.go` 与 Store setter。
- `Host.Snapshot` MEDIUM/lower-bound，漏掉 1 receiver-typing 点；新增字段必须兼容 TUI 读取，源码确认 UISnapshot 通过快照值多处消费。
- `Host.doIntervention` LOW / exact，直接 caller 是 Continue、Resume、Steer、handleIntervention 等；错误分支改动限定为失败输入恢复持久化。
- `EngineService` LOW / exact，直接使用来自 Bridge Startup，另有 Sync caller。
- GitNexus 索引落后 4 个提交；以上 UNKNOWN 不作为安全结论，已用源码文本调用点补核。

## 实施顺序

先在 `engine_service_test.go`、`bridge/revision_test.go`、前端 Store/组件测试添加失败测试，再实现接口与适配。不得先改生产逻辑。
