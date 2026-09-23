package app

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func TestAdvanceProjectionSeparatesPermitFromCanAdvance(t *testing.T) {
	tests := []struct {
		name       string
		mode       domain.ChapterAdvanceMode
		permit     int
		mutate     func(*store.Store)
		wantPermit bool
		wantCan    bool
		wantReason string
		revision   viewmodel.RevisionState
	}{
		{name: "逐章模式等待许可", mode: domain.ChapterAdvanceReview, wantPermit: true, wantCan: true, wantReason: "逐章验收模式等待用户许可"},
		{name: "自动模式不等待许可", mode: domain.ChapterAdvanceAuto, wantReason: "当前为自动推进模式"},
		{name: "未同步修订优先阻塞许可", mode: domain.ChapterAdvanceReview, revision: viewmodel.RevisionSavedUnsynced, wantReason: "存在未同步章节修订"},
		{name: "待恢复修订阻塞许可", mode: domain.ChapterAdvanceReview, revision: viewmodel.RevisionRecoveryPending, wantReason: "存在待恢复的修订同步"},
		{name: "已有许可不再等待", mode: domain.ChapterAdvanceReview, permit: 2, wantReason: "第 2 章已持有一次性推进许可"},
		{name: "待提交阻塞", mode: domain.ChapterAdvanceReview, mutate: func(st *store.Store) {
			if err := st.Signals.SavePendingCommit(domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted}); err != nil {
				t.Fatal(err)
			}
		}, wantReason: "存在待恢复的章节提交"},
		{name: "待返工阻塞", mode: domain.ChapterAdvanceReview, mutate: func(st *store.Store) {
			p, err := st.Progress.Load()
			if err != nil {
				t.Fatal(err)
			}
			p.PendingRewrites = []int{1}
			if err := st.Progress.Save(p); err != nil {
				t.Fatal(err)
			}
		}, wantReason: "待返工章节尚未排空"},
		{name: "暂停意图阻塞", mode: domain.ChapterAdvanceReview, mutate: func(st *store.Store) {
			m, err := st.RunMeta.Load()
			if err != nil {
				t.Fatal(err)
			}
			m.AdvanceHold = &domain.AdvanceHold{Reason: "用户暂停", After: domain.AdvanceHoldAtChapter}
			if err := st.RunMeta.Save(*m); err != nil {
				t.Fatal(err)
			}
		}, wantReason: "存在一次性暂停意图：用户暂停"},
		{name: "Core 审阅流程不等许可", mode: domain.ChapterAdvanceReview, mutate: func(st *store.Store) {
			p, err := st.Progress.Load()
			if err != nil {
				t.Fatal(err)
			}
			p.Flow = domain.FlowReviewing
			if err := st.Progress.Save(p); err != nil {
				t.Fatal(err)
			}
		}, wantReason: "Core 当前处于审阅流程"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fixture(t)
			st := store.NewStore(path)
			if err := st.RunMeta.Save(domain.RunMeta{AdvanceMode: tt.mode, AdvancePermitChapter: tt.permit}); err != nil {
				t.Fatal(err)
			}
			if tt.mutate != nil {
				tt.mutate(st)
			}
			project := &Service{}
			if _, err := project.OpenProject(path); err != nil {
				t.Fatal(err)
			}
			revisionState := tt.revision
			if revisionState == "" {
				revisionState = viewmodel.RevisionSynced
			}
			got, err := project.GetReviewCenter(viewmodel.RevisionStatus{ProjectID: path, State: revisionState, HasUnsynced: revisionState != viewmodel.RevisionSynced})
			if err != nil {
				t.Fatal(err)
			}
			if got.RequiresAdvancePermit != tt.wantPermit || got.CanAdvance != tt.wantCan || got.AdvanceBlockedReason != tt.wantReason {
				t.Fatalf("推进投影不符：requires=%v can=%v reason=%q", got.RequiresAdvancePermit, got.CanAdvance, got.AdvanceBlockedReason)
			}
		})
	}
}

func TestRevisionStateRequiredBeforeAdvanceCanBeTrue(t *testing.T) {
	path := fixture(t)
	st := store.NewStore(path)
	if err := st.RunMeta.Save(domain.RunMeta{AdvanceMode: domain.ChapterAdvanceReview}); err != nil {
		t.Fatal(err)
	}
	project := &Service{}
	if _, err := project.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	got, err := project.GetReviewCenter(viewmodel.RevisionStatus{ProjectID: path, State: viewmodel.RevisionUnknown})
	if err != nil {
		t.Fatal(err)
	}
	if got.CanAdvance || got.RequiresAdvancePermit || got.AdvanceBlockedReason != "章节修订状态尚未确认" {
		t.Fatalf("修订状态未确认前不得声称正在等待许可/可以推进：%+v", got)
	}
}
