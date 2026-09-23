package bridge

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/voocel/ainovel-cli/internal/studio/app"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	mu          sync.RWMutex
	projectMu   sync.RWMutex
	ctx         context.Context
	eventCancel context.CancelFunc
	service     app.Service
	engine      *app.EngineService
	revisions   *app.RevisionService
	projectDir  string
	outputDir   string
}

func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.engine = app.NewEngineService(nil)
	eventCtx, cancel := context.WithCancel(ctx)
	a.eventCancel = cancel
	engine := a.engine
	a.mu.Unlock()

	go func() {
		for {
			select {
			case event := <-engine.Events():
				runtime.EventsEmit(ctx, "studio:engine-event", event)
			case <-eventCtx.Done():
				return
			}
		}
	}()
}

func (a *App) Shutdown(context.Context) {
	a.mu.RLock()
	engine := a.engine
	cancel := a.eventCancel
	a.mu.RUnlock()
	if engine != nil {
		engine.Close()
	}
	if cancel != nil {
		cancel()
	}
}

func (a *App) SelectProjectDirectory() (string, error) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: "打开小说项目"})
}

func (a *App) OpenProject(path string) (viewmodel.Project, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	preview, err := a.service.PreviewProject(path)
	if err != nil {
		return viewmodel.Project{}, err
	}
	a.mu.RLock()
	currentOutputDir, engine := a.outputDir, a.engine
	a.mu.RUnlock()
	if currentOutputDir != "" && !sameProjectPath(currentOutputDir, preview.OutputDir) {
		if engine != nil {
			switch engine.RuntimeState().State {
			case viewmodel.RuntimeRunning, viewmodel.RuntimePausing, viewmodel.RuntimeStopping:
				return viewmodel.Project{}, fmt.Errorf("当前项目仍在创作中，请先停止当前 Engine Session 后再切换项目")
			}
		}
		status, err := a.revisionStatusService().CheckChapterRevisions()
		if err != nil {
			return viewmodel.Project{}, fmt.Errorf("无法确认当前项目修订状态，暂不能切换项目：%w", err)
		}
		if status.HasUnsynced || status.State != viewmodel.RevisionSynced {
			return viewmodel.Project{}, fmt.Errorf("当前项目存在未同步人工修改，请先完成同步后再切换项目")
		}
	}
	if engine := a.engineService(); engine != nil {
		if err := engine.PrepareProjectSwitch(preview.OutputDir); err != nil {
			return viewmodel.Project{}, err
		}
	}
	project, err := a.service.OpenProject(path)
	if err != nil {
		return project, err
	}
	a.mu.Lock()
	a.projectDir = project.ProjectRoot
	a.outputDir = project.OutputDir
	a.mu.Unlock()
	return project, nil
}

func sameProjectPath(left, right string) bool {
	a, errA := filepath.Abs(left)
	b, errB := filepath.Abs(right)
	if errA == nil {
		left = filepath.Clean(a)
	}
	if errB == nil {
		right = filepath.Clean(b)
	}
	return strings.EqualFold(left, right)
}

func (a *App) GetProjectOverview() (viewmodel.Overview, error)  { return a.service.GetProjectOverview() }
func (a *App) GetProjectTree() ([]viewmodel.Node, error)        { return a.service.GetProjectTree() }
func (a *App) GetChapter(number int) (viewmodel.Chapter, error) { return a.service.GetChapter(number) }
func (a *App) GetReviewCenter() (viewmodel.ReviewCenter, error) {
	status, err := a.revisionStatusService().CheckChapterRevisions()
	if err != nil {
		status, _ = a.revisionStatusService().GetRevisionStatus()
		status.State = viewmodel.RevisionError
		status.HasUnsynced = true
		status.Error = err.Error()
	}
	return a.service.GetReviewCenter(status)
}

// GetRevisionStatus 读取当前项目最近一次修订检查结果；首次检查前返回 unknown。
func (a *App) GetRevisionStatus() (viewmodel.RevisionStatus, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	service := a.revisionStatusService()
	return service.GetRevisionStatus()
}

// SaveChapter 保存当前项目中的章节正文，并重新读取 Core 修订状态；不会改接纳记录或触发 Sync。
func (a *App) SaveChapter(chapter int, content string) (viewmodel.ChapterSaveResult, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	a.mu.RLock()
	outputDir := a.outputDir
	engine := a.engine
	a.mu.RUnlock()
	if outputDir == "" {
		return viewmodel.ChapterSaveResult{}, fmt.Errorf("请先打开小说项目")
	}
	if engine != nil {
		runtime := engine.RuntimeState()
		if runtime.ProjectID == outputDir {
			switch runtime.State {
			case viewmodel.RuntimeRunning, viewmodel.RuntimePausing, viewmodel.RuntimeStopping:
				return viewmodel.ChapterSaveResult{}, fmt.Errorf("创作会话处于%s状态，请等待暂停或停止完成后再编辑", runtime.State)
			}
		}
	}
	saved, err := a.service.SaveChapter(chapter, content)
	if err != nil {
		return viewmodel.ChapterSaveResult{}, err
	}
	status, checkErr := a.revisionStatusService().CheckChapterRevisions()
	if checkErr != nil && status.Error == "" {
		status.State = viewmodel.RevisionError
		status.HasUnsynced = true
		status.Error = checkErr.Error()
	}
	return viewmodel.ChapterSaveResult{Chapter: saved, Revision: status}, nil
}

// SyncChapterRevisions 只由用户显式 Sync 操作创建或复用 Host；只读入口不创建 Host。
func (a *App) SyncChapterRevisions(chapter int) (viewmodel.ChapterSyncResult, error) {
	if chapter < 0 {
		return viewmodel.ChapterSyncResult{}, fmt.Errorf("章节号不能为负数")
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	a.mu.RLock()
	projectDir, outputDir, engine, ctx := a.projectDir, a.outputDir, a.engine, a.ctx
	a.mu.RUnlock()
	if projectDir == "" || outputDir == "" {
		return viewmodel.ChapterSyncResult{}, fmt.Errorf("请先打开小说项目")
	}
	if engine == nil {
		engine = app.NewEngineService(nil)
		a.mu.Lock()
		if a.engine == nil {
			a.engine = engine
		} else {
			engine = a.engine
		}
		a.mu.Unlock()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := engine.SyncChapterRevisions(ctx, projectDir, outputDir); err != nil {
		// 失败时重新检查 Store，让 Host 保留的 pending 恢复阶段继续阻止写作并允许重试。
		_, _ = a.revisionStatusService().CheckChapterRevisions()
		return viewmodel.ChapterSyncResult{}, err
	}
	status, err := a.revisionStatusService().CheckChapterRevisions()
	if err != nil {
		return viewmodel.ChapterSyncResult{}, fmt.Errorf("Core Sync 已返回成功，但无法复核 Store 修订状态：%w", err)
	}
	if status.State != viewmodel.RevisionSynced || status.HasUnsynced {
		return viewmodel.ChapterSyncResult{}, fmt.Errorf("Core Sync 已返回成功，但 Store 仍报告未同步章节修订")
	}
	result := viewmodel.ChapterSyncResult{Revision: status}
	project, err := a.service.OpenProject(outputDir)
	if err != nil {
		result.RefreshWarning = fmt.Sprintf("重新读取项目视图失败：%v", err)
	} else {
		result.Project = &project
		a.mu.Lock()
		a.projectDir, a.outputDir = project.ProjectRoot, project.OutputDir
		a.mu.Unlock()
	}
	var selected *viewmodel.Chapter
	if chapter > 0 {
		refreshed, err := a.service.GetChapter(chapter)
		if err != nil {
			warning := fmt.Sprintf("重新读取第 %d 章失败：%v", chapter, err)
			if result.RefreshWarning != "" {
				result.RefreshWarning += "；"
			}
			result.RefreshWarning += warning
		} else {
			selected = &refreshed
		}
	}
	result.Chapter = selected
	return result, nil
}

// CheckChapterRevisions 只读检查 Store 中待同步章节，不创建 Host 或 Engine Session。
func (a *App) CheckChapterRevisions() (viewmodel.RevisionStatus, error) {
	// 独占项目控制锁，避免只读扫描与并发 Resume/项目切换交错。
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	service := a.revisionStatusService()
	return service.CheckChapterRevisions()
}

// GetRuntimeState 返回已有的 Runtime 投影，不创建 Host 或 Engine Session。
func (a *App) GetRuntimeState() viewmodel.Runtime {
	a.mu.RLock()
	engine := a.engine
	outputDir := a.outputDir
	a.mu.RUnlock()
	state := viewmodel.Runtime{ProjectID: outputDir, State: viewmodel.RuntimeIdle}
	if engine != nil {
		state = engine.RuntimeState()
		if state.ProjectID != outputDir {
			state = viewmodel.Runtime{ProjectID: outputDir, State: viewmodel.RuntimeIdle}
		}
	}
	a.enrichRuntime(&state)
	status, statusErr := a.revisionStatusService().GetRevisionStatus()
	if statusErr == nil && sameProjectPath(status.ProjectID, outputDir) {
		if status.HasUnsynced || status.State == viewmodel.RevisionSavedUnsynced || status.State == viewmodel.RevisionRecoveryPending {
			switch state.State {
			case viewmodel.RuntimeIdle, viewmodel.RuntimePaused, viewmodel.RuntimeStopped, viewmodel.RuntimeCompleted, viewmodel.RuntimeWaitingReview:
				state.State = viewmodel.RuntimeWaitingSync
			}
		} else if status.State == viewmodel.RevisionSynced && runtimeIsStopped(state.State) {
			if projection, err := a.service.GetAdvanceProjection(status); err == nil && projection.ProjectID == outputDir && projection.RequiresAdvancePermit && projection.CanAdvance {
				state.State = viewmodel.RuntimeWaitingReview
			}
		}
	}
	return state
}

// ResumeWriting 是 M3-A EngineService 的显式启动入口；打开项目本身仍只读。
func (a *App) ResumeWriting() (viewmodel.Runtime, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	a.mu.RLock()
	projectDir, outputDir, engine := a.projectDir, a.outputDir, a.engine
	a.mu.RUnlock()
	if projectDir == "" || outputDir == "" {
		return viewmodel.Runtime{}, fmt.Errorf("请先打开小说项目")
	}
	if engine == nil {
		return viewmodel.Runtime{}, fmt.Errorf("Studio Engine Service 尚未启动")
	}
	status, err := a.revisionStatusService().CheckChapterRevisions()
	if err != nil {
		return viewmodel.Runtime{}, fmt.Errorf("检查章节修订失败，暂不能继续创作：%w", err)
	}
	if status.HasUnsynced || status.State != viewmodel.RevisionSynced {
		return viewmodel.Runtime{}, fmt.Errorf("存在未同步章节修订，请先同步后再继续创作")
	}
	state, err := engine.ResumeWriting(projectDir, outputDir)
	if err != nil {
		return state, err
	}
	a.enrichRuntime(&state)
	return state, nil
}

func (a *App) PauseWriting() (viewmodel.Runtime, error) {
	engine := a.engineService()
	if engine == nil {
		return viewmodel.Runtime{}, fmt.Errorf("Studio Engine Service 尚未启动")
	}
	state, err := engine.PauseWriting()
	if err == nil {
		a.enrichRuntime(&state)
	}
	return state, err
}

func (a *App) StopWriting() (viewmodel.Runtime, error) {
	engine := a.engineService()
	if engine == nil {
		return viewmodel.Runtime{}, fmt.Errorf("Studio Engine Service 尚未启动")
	}
	state, err := engine.StopWriting()
	if err == nil {
		a.enrichRuntime(&state)
	}
	return state, err
}

func (a *App) enrichRuntime(state *viewmodel.Runtime) {
	if state == nil {
		return
	}
	status, statusErr := a.revisionStatusService().GetRevisionStatus()
	if projection, err := a.service.GetAdvanceProjection(status); err == nil {
		// OpenProject publishes the Store and Bridge project ID in two steps. If a
		// read overlaps that transition, never attach one project's gate to another
		// project's Runtime; the next Runtime refresh will fill the matching facts.
		if state.ProjectID == "" || projection.ProjectID == "" || !sameProjectPath(state.ProjectID, projection.ProjectID) {
			return
		}
		state.RequiresAdvancePermit = projection.RequiresAdvancePermit
		state.CanAdvance = projection.CanAdvance && statusErr == nil && sameProjectPath(status.ProjectID, projection.ProjectID) && status.State == viewmodel.RevisionSynced && !status.HasUnsynced
		state.AdvanceBlockedReason = projection.AdvanceBlockedReason
		if !state.CanAdvance && state.RequiresAdvancePermit && state.AdvanceBlockedReason == "" {
			state.AdvanceBlockedReason = "章节修订状态尚未确认"
		}
		if !runtimeIsStopped(state.State) {
			state.CanAdvance = false
			if state.AdvanceBlockedReason == "" {
				state.AdvanceBlockedReason = "Engine 当前状态不能执行下一章推进"
			}
		}
		state.NextChapter = projection.NextChapter
		state.HasCurrentReview = projection.HasCurrentReview
	}
}

func runtimeIsStopped(state viewmodel.RuntimeState) bool {
	switch state {
	case viewmodel.RuntimeIdle, viewmodel.RuntimePaused, viewmodel.RuntimeStopped, viewmodel.RuntimeCompleted, viewmodel.RuntimeWaitingReview:
		return true
	default:
		return false
	}
}

func (a *App) ConfirmChapterCommit(chapter int, startedAt string) (viewmodel.ChapterCommitConfirmation, error) {
	started, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return viewmodel.ChapterCommitConfirmation{}, fmt.Errorf("章节提交事件时间无效：%w", err)
	}
	project, confirmed, err := a.service.ConfirmChapterCommit(chapter, started)
	if err != nil {
		return viewmodel.ChapterCommitConfirmation{}, err
	}
	return viewmodel.ChapterCommitConfirmation{Confirmed: confirmed, Project: project}, nil
}

func (a *App) engineService() *app.EngineService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.engine
}

func (a *App) revisionStatusService() *app.RevisionService {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.revisions == nil {
		a.revisions = app.NewRevisionService(&a.service, a.GetRuntimeState)
	}
	return a.revisions
}
