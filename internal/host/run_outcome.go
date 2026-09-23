package host

// RunOutcome 描述最近一次 Engine 循环的结束原因，供 Studio 等非 TUI 调用方区分
// 可恢复暂停与无法继续的内部故障。它不改变 Host/TUI 生命周期与事件路由。
type RunOutcome string

const (
	RunOutcomePaused  RunOutcome = "paused"
	RunOutcomeFailed  RunOutcome = "failed"
	RunOutcomeNatural RunOutcome = "natural"
)
