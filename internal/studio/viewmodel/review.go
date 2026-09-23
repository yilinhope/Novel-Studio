package viewmodel

import "github.com/voocel/ainovel-cli/internal/domain"

// ReviewCenter 是 Store 中真实 Editor Review 与独立 Advance Gate 的只读投影。
type ReviewCenter struct {
	ProjectID             string               `json:"projectId"`
	Reviews               []domain.ReviewEntry `json:"reviews"`
	CurrentReviews        []domain.ReviewEntry `json:"currentReviews"`
	RequiresAdvancePermit bool                 `json:"requiresAdvancePermit"`
	CanAdvance            bool                 `json:"canAdvance"`
	NextChapter           int                  `json:"nextChapter"`
	HasCurrentReview      bool                 `json:"hasCurrentReview"`
	AdvanceMode           string               `json:"advanceMode"`
	AdvancePermitChapter  int                  `json:"advancePermitChapter"`
	AdvanceHoldReason     string               `json:"advanceHoldReason,omitempty"`
	AdvanceBlockedReason  string               `json:"advanceBlockedReason,omitempty"`
}
