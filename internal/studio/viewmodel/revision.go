package viewmodel

import "time"

type RevisionState string

const (
	RevisionUnknown         RevisionState = "unknown"
	RevisionSynced          RevisionState = "synced"
	RevisionSavedUnsynced   RevisionState = "saved_unsynced"
	RevisionRecoveryPending RevisionState = "recovery_pending"
	RevisionError           RevisionState = "error"
)

// RevisionStatus 是由当前项目 Store 事实派生的只读修订状态。
type RevisionStatus struct {
	ProjectID    string            `json:"projectId"`
	State        RevisionState     `json:"state"`
	HasUnsynced  bool              `json:"hasUnsynced"`
	Chapters     []UnsyncedChapter `json:"chapters"`
	PendingStage string            `json:"pendingStage,omitempty"`
	CheckedAt    time.Time         `json:"checkedAt,omitempty"`
	Error        string            `json:"error,omitempty"`
}

type UnsyncedChapter struct {
	Chapter      int    `json:"chapter"`
	AcceptedHash string `json:"acceptedHash"`
	CurrentHash  string `json:"currentHash"`
}
