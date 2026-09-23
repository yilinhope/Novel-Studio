package app

import (
	"fmt"
	"slices"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// GetReviewCenter 只读读取 Store 中的真实审阅记录和推进门状态，不创建 Host，
// 不初始化/写入 RunMeta，也不更改 ReviewEntry 或 AdvancePermit。
func (s *Service) GetReviewCenter() (viewmodel.ReviewCenter, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.ReviewCenter{}, err
	}
	result, progress, err := loadAdvanceProjection(st)
	if err != nil || progress == nil {
		result.ProjectID = st.Dir()
		return result, err
	}
	result.ProjectID = st.Dir()

	// 只有 Core 真实存储的 ReviewEntry 才能进入 Review Center。推进等待本身
	// 不会合成审阅条目；按已完成章节枚举可同时覆盖 chapter/arc/global 记录。
	completed := slices.Clone(progress.CompletedChapters)
	slices.Sort(completed)
	for _, chapter := range completed {
		for _, load := range []func(int) (*domain.ReviewEntry, error){st.World.LoadReview, st.World.LoadGlobalReview} {
			review, err := load(chapter)
			if err != nil {
				return viewmodel.ReviewCenter{}, fmt.Errorf("读取第 %d 章 ReviewEntry 失败：%w", chapter, err)
			}
			if review == nil || review.Chapter != chapter {
				continue
			}
			result.Reviews = append(result.Reviews, *review)
		}
	}
	return result, nil
}

// GetAdvanceProjection 为 Runtime 提供常量工作量的推进门/当前 Review 只读事实。
func (s *Service) GetAdvanceProjection() (viewmodel.ReviewCenter, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.ReviewCenter{}, err
	}
	result, _, err := loadAdvanceProjection(st)
	result.ProjectID = st.Dir()
	return result, err
}

func loadAdvanceProjection(st *store.Store) (viewmodel.ReviewCenter, *domain.Progress, error) {
	progress, err := st.Progress.Load()
	if err != nil {
		return viewmodel.ReviewCenter{}, nil, fmt.Errorf("读取项目进度失败：%w", err)
	}
	meta, err := st.RunMeta.Load()
	if err != nil {
		return viewmodel.ReviewCenter{}, nil, fmt.Errorf("读取章节推进状态失败：%w", err)
	}
	pending, err := st.Signals.LoadPendingCommit()
	if err != nil {
		return viewmodel.ReviewCenter{}, nil, fmt.Errorf("读取待提交状态失败：%w", err)
	}
	result := viewmodel.ReviewCenter{Reviews: []domain.ReviewEntry{}, CurrentReviews: []domain.ReviewEntry{}}
	if progress == nil {
		return result, nil, nil
	}
	result.NextChapter = progress.NextChapter()
	latest := progress.LatestCompleted()
	if latest > 0 {
		for _, load := range []func(int) (*domain.ReviewEntry, error){st.World.LoadReview, st.World.LoadGlobalReview} {
			review, err := load(latest)
			if err != nil {
				return viewmodel.ReviewCenter{}, nil, fmt.Errorf("读取当前第 %d 章 ReviewEntry 失败：%w", latest, err)
			}
			if review != nil && review.Chapter == latest {
				result.CurrentReviews = append(result.CurrentReviews, *review)
			}
		}
	}
	result.HasCurrentReview = len(result.CurrentReviews) > 0
	if meta == nil {
		result.AdvanceBlockedReason = "推进状态尚未初始化"
		return result, progress, nil
	}
	result.AdvanceMode = string(meta.AdvanceMode)
	result.AdvancePermitChapter = meta.AdvancePermitChapter
	if meta.AdvanceHold != nil {
		result.AdvanceHoldReason = meta.AdvanceHold.Reason
	}
	// Advance Gate 不是 Editor Review。该字段仅投影当前正向下一章是否缺少
	// review 模式许可；目标章节范围内的 chapter hold 是 Core 明确允许的例外。
	permitMatches := result.NextChapter > 0 && meta.AdvancePermitChapter == result.NextChapter
	holdAllows := meta.AdvanceHold != nil &&
		meta.AdvanceHold.After == domain.AdvanceHoldAtChapter &&
		result.NextChapter > 0 && result.NextChapter <= meta.AdvanceHold.TargetChapter
	forwardWorkReady := progress.Phase == domain.PhaseWriting && progress.Flow != domain.FlowReviewing &&
		progress.Flow != domain.FlowSteering && progress.Flow != domain.FlowRewriting &&
		progress.Flow != domain.FlowPolishing && len(progress.PendingRewrites) == 0 &&
		progress.InProgressChapter == 0 && pending == nil
	result.RequiresAdvancePermit = forwardWorkReady &&
		meta.AdvanceMode == domain.ChapterAdvanceReview && !permitMatches && !holdAllows
	switch {
	case pending != nil:
		result.AdvanceBlockedReason = "存在待恢复的章节提交"
	case len(progress.PendingRewrites) > 0:
		result.AdvanceBlockedReason = "待返工章节尚未排空"
	case progress.Flow == domain.FlowRewriting || progress.Flow == domain.FlowPolishing:
		result.AdvanceBlockedReason = "当前流程正在返工或打磨"
	case progress.Flow == domain.FlowReviewing:
		result.AdvanceBlockedReason = "Core 当前处于审阅流程"
	case progress.Flow == domain.FlowSteering:
		result.AdvanceBlockedReason = "Core 当前正在处理创作干预"
	case progress.InProgressChapter > 0:
		result.AdvanceBlockedReason = "当前章节仍在生成"
	case progress.Phase != domain.PhaseWriting:
		result.AdvanceBlockedReason = fmt.Sprintf("当前创作阶段为 %s", progress.Phase)
	case meta.AdvanceHold != nil && !holdAllows:
		result.AdvanceBlockedReason = "存在一次性暂停意图：" + meta.AdvanceHold.Reason
	case permitMatches:
		result.AdvanceBlockedReason = fmt.Sprintf("第 %d 章已持有一次性推进许可", result.NextChapter)
	case meta.AdvanceMode == domain.ChapterAdvanceAuto:
		result.AdvanceBlockedReason = "当前为自动推进模式"
	case result.RequiresAdvancePermit:
		result.AdvanceBlockedReason = "逐章验收模式等待用户许可"
	}
	return result, progress, nil
}
