package bridge

import (
	"fmt"
	"strings"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/studio/v2"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// v2Manager 只绑定当前项目的输出目录；Proposal metadata 不改变当前 Core Store。
func (a *App) v2Manager() (*v2.Manager, string, error) {
	a.mu.RLock()
	outputDir := a.outputDir
	a.mu.RUnlock()
	if strings.TrimSpace(outputDir) == "" {
		return nil, "", fmt.Errorf("请先打开小说项目")
	}
	manager := v2.NewManager(outputDir)
	manager.SetProjectID(outputDir)
	return manager, outputDir, nil
}

func (a *App) ListProposals() ([]viewmodel.Proposal, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return nil, err
	}
	var proposals []viewmodel.Proposal
	err = a.withProjectBookLease(outputDir, func() error {
		if err := manager.RecoverOperations(); err != nil {
			return err
		}
		var listErr error
		proposals, listErr = manager.List()
		return listErr
	})
	if err != nil {
		return nil, err
	}
	return proposals, nil
}

func (a *App) GetProposal(id string) (viewmodel.Proposal, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return viewmodel.Proposal{}, err
	}
	var proposal viewmodel.Proposal
	err = a.withProjectBookLease(outputDir, func() error {
		if err := manager.RecoverOperations(); err != nil {
			return err
		}
		var getErr error
		proposal, getErr = manager.Get(id)
		return getErr
	})
	if err != nil {
		return viewmodel.Proposal{}, err
	}
	return proposal, nil
}

func (a *App) CreateProposal(request viewmodel.CreateProposalRequest) (viewmodel.Proposal, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return viewmodel.Proposal{}, err
	}
	var proposal viewmodel.Proposal
	err = a.withProjectBookLease(outputDir, func() error {
		var createErr error
		proposal, createErr = manager.Create(request)
		return createErr
	})
	return proposal, err
}

// CreateProposalFromReviewIssue 将 Core ReviewEntry 中的一个 issue 转为人工审核候选。
// 候选正文由用户传入，M1 不创建新的模型生成调用链。
func (a *App) CreateProposalFromReviewIssue(chapter int, scope string, issueIndex int, after string) (viewmodel.Proposal, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return viewmodel.Proposal{}, err
	}
	status, err := a.revisionStatusService().CheckChapterRevisions()
	if err != nil {
		return viewmodel.Proposal{}, fmt.Errorf("读取修订状态失败：%w", err)
	}
	reviewCenter, err := a.service.GetReviewCenter(status)
	if err != nil {
		return viewmodel.Proposal{}, err
	}
	var matched *domain.ReviewEntry
	for index := range reviewCenter.Reviews {
		review := &reviewCenter.Reviews[index]
		if review.Chapter == chapter && (strings.TrimSpace(scope) == "" || review.Scope == scope) {
			matched = review
			break
		}
	}
	if matched == nil {
		return viewmodel.Proposal{}, fmt.Errorf("没有找到第 %d 章 scope=%q 的 ReviewEntry", chapter, scope)
	}
	if issueIndex < 0 || issueIndex >= len(matched.Issues) {
		return viewmodel.Proposal{}, fmt.Errorf("ReviewIssue 索引无效：%d", issueIndex)
	}
	issue := matched.Issues[issueIndex]
	if len(issue.Chapters) > 1 {
		return viewmodel.Proposal{}, fmt.Errorf("该 ReviewIssue 涉及多章，请在建议收件箱中分别提供每章候选正文")
	}
	targetChapter := chapter
	if len(issue.Chapters) == 1 {
		targetChapter = issue.Chapters[0]
	}
	if strings.TrimSpace(after) == "" {
		return viewmodel.Proposal{}, fmt.Errorf("候选正文不能为空")
	}
	request := viewmodel.CreateProposalRequest{
		Title:     fmt.Sprintf("修正第 %d 章：%s", targetChapter, issue.Type),
		Summary:   issue.Description,
		Rationale: issue.Suggestion,
		Source:    v2.ProposalSourceReviewIssue,
		SourceKey: v2.ReviewIssueSourceKey(*matched, issueIndex),
		Changes: []viewmodel.ProposalChangeInput{{
			ResourceType: v2.ResourceTypeChapterText,
			ResourceID:   fmt.Sprintf("chapter:%d", targetChapter),
			Chapter:      targetChapter,
			After:        after,
			ChangeType:   v2.ChangeTypeReplace,
		}},
	}
	var proposal viewmodel.Proposal
	err = a.withProjectBookLease(outputDir, func() error {
		var createErr error
		proposal, createErr = manager.Create(request)
		return createErr
	})
	return proposal, err
}

func (a *App) AcceptProposal(id string) (viewmodel.Proposal, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return viewmodel.Proposal{}, err
	}
	var proposal viewmodel.Proposal
	err = a.withProjectBookLease(outputDir, func() error {
		var acceptErr error
		proposal, acceptErr = manager.Accept(id)
		return acceptErr
	})
	return proposal, err
}

func (a *App) RejectProposal(id string) (viewmodel.Proposal, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return viewmodel.Proposal{}, err
	}
	var proposal viewmodel.Proposal
	err = a.withProjectBookLease(outputDir, func() error {
		var rejectErr error
		proposal, rejectErr = manager.Reject(id)
		return rejectErr
	})
	return proposal, err
}

func (a *App) ApplyProposal(id string) (viewmodel.ApplyResult, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return viewmodel.ApplyResult{}, err
	}
	if engine := a.engineService(); engine != nil {
		runtime := engine.RuntimeState()
		if runtime.ProjectID == outputDir {
			switch runtime.State {
			case viewmodel.RuntimeRunning, viewmodel.RuntimePausing, viewmodel.RuntimeStopping:
				return viewmodel.ApplyResult{}, fmt.Errorf("创作会话处于%s状态，请等待暂停或停止完成后再应用 Proposal", runtime.State)
			}
		}
	}
	manager.SetSaveChapter(func(chapter int, content string) error {
		_, err := a.service.SaveChapter(chapter, content)
		return err
	})
	var result viewmodel.ApplyResult
	err = a.withProjectBookLease(outputDir, func() error {
		var applyErr error
		result, applyErr = manager.Apply(id)
		return applyErr
	})
	if err != nil {
		return result, err
	}
	status, statusErr := a.revisionStatusService().CheckChapterRevisions()
	if statusErr == nil && status.HasUnsynced {
		_ = a.withProjectBookLease(outputDir, func() error {
			_, err := manager.MarkSyncPending(id)
			return err
		})
	}
	return result, nil
}

func (a *App) GetProposalDiff(id string, changeIndex int) ([]v2.DiffLine, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	manager, _, err := a.v2Manager()
	if err != nil {
		return nil, err
	}
	proposal, err := manager.Get(id)
	if err != nil {
		return nil, err
	}
	if changeIndex < 0 || changeIndex >= len(proposal.Changes) {
		return nil, fmt.Errorf("Proposal change 索引无效：%d", changeIndex)
	}
	change := proposal.Changes[changeIndex]
	return v2.TextDiff(change.Before, change.After), nil
}

func (a *App) GetVersionHistory(chapter int) ([]viewmodel.VersionSnapshot, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	manager, _, err := a.v2Manager()
	if err != nil {
		return nil, err
	}
	return manager.ListVersions(chapter)
}

func (a *App) GetVersion(id string) (viewmodel.VersionSnapshot, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	manager, _, err := a.v2Manager()
	if err != nil {
		return viewmodel.VersionSnapshot{}, err
	}
	return manager.GetVersion(id)
}

func (a *App) RestoreVersion(id string) (viewmodel.VersionSnapshot, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return viewmodel.VersionSnapshot{}, err
	}
	if engine := a.engineService(); engine != nil {
		runtime := engine.RuntimeState()
		if runtime.ProjectID == outputDir {
			switch runtime.State {
			case viewmodel.RuntimeRunning, viewmodel.RuntimePausing, viewmodel.RuntimeStopping:
				return viewmodel.VersionSnapshot{}, fmt.Errorf("创作会话处于%s状态，请等待暂停或停止完成后再恢复版本", runtime.State)
			}
		}
	}
	manager.SetSaveChapter(func(chapter int, content string) error {
		_, err := a.service.SaveChapter(chapter, content)
		return err
	})
	var snapshot viewmodel.VersionSnapshot
	err = a.withProjectBookLease(outputDir, func() error {
		var restoreErr error
		snapshot, restoreErr = manager.RestoreVersion(id)
		return restoreErr
	})
	return snapshot, err
}

func (a *App) reconcileV2Sync(status viewmodel.RevisionStatus) {
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return
	}
	_ = a.withProjectBookLease(outputDir, func() error {
		_, err := manager.ReconcileSync(status.State == viewmodel.RevisionSynced && !status.HasUnsynced)
		return err
	})
}

func (a *App) markV2SyncPending() {
	manager, outputDir, err := a.v2Manager()
	if err != nil {
		return
	}
	_ = a.withProjectBookLease(outputDir, func() error {
		_, err := manager.ReconcileSync(false)
		return err
	})
}
