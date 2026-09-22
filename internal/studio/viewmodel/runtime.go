package viewmodel

import "time"

type RuntimeState string

const (
	RuntimeIdle          RuntimeState = "idle"
	RuntimeRunning       RuntimeState = "running"
	RuntimePausing       RuntimeState = "pausing"
	RuntimePaused        RuntimeState = "paused"
	RuntimeStopping      RuntimeState = "stopping"
	RuntimeStopped       RuntimeState = "stopped"
	RuntimeWaitingReview RuntimeState = "waiting_review"
	RuntimeWaitingSync   RuntimeState = "waiting_sync"
	RuntimeCompleted     RuntimeState = "completed"
	RuntimeError         RuntimeState = "error"
)

// Runtime 是从 Core 当前事实派生的运行时只读视图。
type Runtime struct {
	State               RuntimeState `json:"state"`
	Phase               string       `json:"phase"`
	Flow                string       `json:"flow"`
	Agent               string       `json:"agent,omitempty"`
	Chapter             int          `json:"chapter,omitempty"`
	Step                string       `json:"step,omitempty"`
	ElapsedSeconds      int64        `json:"elapsedSeconds"`
	InputTokens         int          `json:"inputTokens"`
	OutputTokens        int          `json:"outputTokens"`
	ProjectInputTokens  int          `json:"projectInputTokens"`
	ProjectOutputTokens int          `json:"projectOutputTokens"`
	RunCostUSD          float64      `json:"runCostUsd"`
	ProjectCostUSD      float64      `json:"projectCostUsd"`
	Error               string       `json:"error,omitempty"`
	UpdatedAt           time.Time    `json:"updatedAt"`
}
