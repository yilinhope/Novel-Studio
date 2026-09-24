package bridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/entry/startup"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/host/imp"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/app"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	mu                 sync.RWMutex
	projectMu          sync.RWMutex
	ctx                context.Context
	eventCancel        context.CancelFunc
	service            app.Service
	engine             *app.EngineService
	revisions          *app.RevisionService
	projectDir         string
	outputDir          string
	operationMu        sync.Mutex
	operation          string
	operationID        uint64
	operationRequestID string
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
	if operation := a.activeOperation(); operation != "" {
		return viewmodel.Project{}, fmt.Errorf("当前%s操作仍在进行，请先完成或取消后再切换项目", operation)
	}
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

// SetAdvanceMode 仅在用户显式切换时创建 Host；切换本身不启动或恢复 Engine。
func (a *App) SetAdvanceMode(mode string) (viewmodel.ControlResult, error) {
	advanceMode := domain.ChapterAdvanceMode(strings.TrimSpace(mode))
	if !advanceMode.Valid() {
		return viewmodel.ControlResult{}, fmt.Errorf("不支持的章节推进模式：%q", mode)
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	projectDir, outputDir := a.openProjectPaths()
	if projectDir == "" || outputDir == "" {
		return viewmodel.ControlResult{}, fmt.Errorf("请先打开小说项目")
	}
	engine := a.ensureEngineService()
	if _, err := engine.SetAdvanceMode(projectDir, outputDir, advanceMode); err != nil {
		return viewmodel.ControlResult{}, err
	}
	return a.refreshControlResult(outputDir)
}

// AdvanceOneChapter 只委托 Core Host.AdvanceOneChapter；permit 和 ReviewEntry 均由 Core 管理。
func (a *App) AdvanceOneChapter() (viewmodel.ControlResult, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	projectDir, outputDir := a.openProjectPaths()
	if projectDir == "" || outputDir == "" {
		return viewmodel.ControlResult{}, fmt.Errorf("请先打开小说项目")
	}
	status, err := a.revisionStatusService().CheckChapterRevisions()
	if err != nil {
		return viewmodel.ControlResult{}, fmt.Errorf("检查章节修订失败，暂不能继续下一章：%w", err)
	}
	if !sameProjectPath(status.ProjectID, outputDir) || status.HasUnsynced || status.State != viewmodel.RevisionSynced {
		return viewmodel.ControlResult{}, fmt.Errorf("存在未同步或待恢复的章节修订，请先完成同步")
	}
	projection, err := a.service.GetAdvanceProjection(status)
	if err != nil {
		return viewmodel.ControlResult{}, fmt.Errorf("读取 Core 下一章推进门失败：%w", err)
	}
	if !projection.RequiresAdvancePermit || !projection.CanAdvance {
		reason := projection.AdvanceBlockedReason
		if reason == "" {
			reason = "当前 Core 状态不允许继续下一章"
		}
		return viewmodel.ControlResult{}, fmt.Errorf("不能继续下一章：%s", reason)
	}
	if _, err := a.ensureEngineService().AdvanceOneChapter(projectDir, outputDir); err != nil {
		return viewmodel.ControlResult{}, err
	}
	return a.refreshControlResult(outputDir)
}

// SubmitSteer 将用户明确输入的创作干预交给 Core Arbiter，不作为普通 Resume 使用。
func (a *App) SubmitSteer(text string) (viewmodel.ControlResult, error) {
	if strings.TrimSpace(text) == "" {
		return viewmodel.ControlResult{}, fmt.Errorf("创作指令不能为空")
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	projectDir, outputDir := a.openProjectPaths()
	if projectDir == "" || outputDir == "" {
		return viewmodel.ControlResult{}, fmt.Errorf("请先打开小说项目")
	}
	status, err := a.revisionStatusService().CheckChapterRevisions()
	if err != nil {
		return viewmodel.ControlResult{}, fmt.Errorf("检查章节修订失败，暂不能提交创作指令：%w", err)
	}
	if !sameProjectPath(status.ProjectID, outputDir) || status.HasUnsynced || status.State != viewmodel.RevisionSynced {
		return viewmodel.ControlResult{}, fmt.Errorf("存在未同步或待恢复的章节修订，请先完成同步")
	}
	if _, err := a.ensureEngineService().SubmitSteer(projectDir, outputDir, text); err != nil {
		return viewmodel.ControlResult{}, err
	}
	return a.refreshControlResult(outputDir)
}

func (a *App) openProjectPaths() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.projectDir, a.outputDir
}

func (a *App) ensureEngineService() *app.EngineService {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.engine == nil {
		a.engine = app.NewEngineService(nil)
	}
	return a.engine
}

func (a *App) refreshControlResult(outputDir string) (viewmodel.ControlResult, error) {
	result := viewmodel.ControlResult{Revision: viewmodel.RevisionStatus{ProjectID: outputDir}}
	project, projectErr := a.service.OpenProject(outputDir)
	if projectErr != nil {
		result.RefreshWarning = fmt.Sprintf("Core 操作已成功，但项目视图刷新失败：%v", projectErr)
	} else {
		result.Project = &project
		a.mu.Lock()
		a.projectDir, a.outputDir = project.ProjectRoot, project.OutputDir
		a.mu.Unlock()
	}
	status, statusErr := a.revisionStatusService().CheckChapterRevisions()
	if statusErr != nil {
		warning := fmt.Sprintf("修订状态刷新失败：%v", statusErr)
		if result.RefreshWarning != "" {
			result.RefreshWarning += "；"
		}
		result.RefreshWarning += warning
	} else {
		result.Revision = status
		if review, err := a.service.GetReviewCenter(status); err == nil {
			result.Review = &review
		} else {
			if result.RefreshWarning != "" {
				result.RefreshWarning += "；"
			}
			result.RefreshWarning += fmt.Sprintf("审阅/推进状态刷新失败：%v", err)
		}
	}
	result.Runtime = a.GetRuntimeState()
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
	if outputDir != "" {
		if meta, err := store.NewStore(outputDir).RunMeta.Load(); err == nil && meta != nil {
			state.PendingSteer = meta.PendingSteer
		}
	}
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
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
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

func (a *App) activeOperation() string {
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	return a.operation
}

func (a *App) beginOperation(operation, requestID string) (uint64, error) {
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	if a.operation != "" {
		return 0, fmt.Errorf("当前%s操作仍在进行", a.operation)
	}
	a.operationID++
	a.operation = operation
	a.operationRequestID = strings.TrimSpace(requestID)
	return a.operationID, nil
}

func (a *App) finishOperation(id uint64) {
	a.operationMu.Lock()
	if a.operationID == id {
		a.operation = ""
		a.operationRequestID = ""
	}
	a.operationMu.Unlock()
}

func createProjectPaths(selected string) (string, string, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return "", "", fmt.Errorf("项目目录不能为空")
	}
	abs, err := filepath.Abs(selected)
	if err != nil {
		return "", "", fmt.Errorf("解析项目目录失败：%w", err)
	}
	root, output := abs, filepath.Join(abs, "output", "novel")
	if strings.EqualFold(filepath.Base(abs), "novel") && strings.EqualFold(filepath.Base(filepath.Dir(abs)), "output") {
		root, output = filepath.Dir(filepath.Dir(abs)), abs
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", "", fmt.Errorf("创建项目目录失败：%w", err)
	}
	return root, output, nil
}

func (a *App) publishProject(outputDir string) (viewmodel.Project, error) {
	project, err := a.service.OpenProject(outputDir)
	if err != nil {
		return viewmodel.Project{}, err
	}
	a.mu.Lock()
	a.projectDir, a.outputDir = project.ProjectRoot, project.OutputDir
	a.mu.Unlock()
	return project, nil
}

func (a *App) currentProjectPaths() (string, string, error) {
	projectDir, outputDir := a.openProjectPaths()
	if projectDir == "" || outputDir == "" {
		return "", "", fmt.Errorf("请先打开小说项目")
	}
	return projectDir, outputDir, nil
}

// StartQuickStart 通过现有 Host.PrepareUserRules/StartPrepared 创建项目。
// mode=outline 时 Prompt 被视为 Core 现有 /start 接受的文本文件路径。
func (a *App) StartQuickStart(request viewmodel.CreateProjectRequest) (viewmodel.OperationAck, error) {
	requestID := operationRequestID(request.RequestID, "create")
	mode := strings.ToLower(strings.TrimSpace(request.Mode))
	if mode == "" {
		mode = "quick"
	}
	if mode != "quick" && mode != "outline" {
		return viewmodel.OperationAck{}, fmt.Errorf("不支持的创建模式：%s", mode)
	}
	prompt := request.Prompt
	if mode == "outline" {
		loaded, err := startup.LoadPromptFile(prompt)
		if err != nil {
			return viewmodel.OperationAck{}, fmt.Errorf("读取大纲文件失败：%w", err)
		}
		prompt = loaded
	}
	prompt, err := startup.PrepareQuick(prompt)
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	root, output, err := createProjectPaths(request.ProjectRoot)
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if operation := a.activeOperation(); operation != "" {
		return viewmodel.OperationAck{}, fmt.Errorf("当前%s操作仍在进行", operation)
	}
	if engine := a.engineService(); engine != nil {
		if current := a.openOutputDir(); current != "" && !sameProjectPath(current, output) {
			if err := engine.PrepareProjectSwitch(output); err != nil {
				return viewmodel.OperationAck{}, err
			}
		}
	}
	id, err := a.beginOperation("创建项目", requestID)
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	ack := viewmodel.OperationAck{ProjectID: output, Operation: "create", RequestID: requestID}
	go func() {
		defer a.finishOperation(id)
		runtimeState, runErr := a.ensureEngineService().StartPreparedProject(root, output, prompt)
		event := viewmodel.CreateEvent{ProjectID: output, Generation: runtimeState.Generation, Operation: "create", RequestID: requestID}
		if runErr != nil {
			event.State, event.Error = "error", runErr.Error()
		} else {
			event.State, event.Message = "completed", "Core 已接受创建请求并开始创作"
			event.Runtime = &runtimeState
		}
		if project, projectErr := a.publishProject(output); projectErr == nil {
			event.Project = &project
		} else if runErr == nil {
			event.State, event.Error = "error", fmt.Sprintf("项目已创建，但读取 Overview 失败：%v", projectErr)
		}
		a.emitCreateEvent(event)
	}()
	return ack, nil
}

// PreviewOutline 只读取并复用 Core 的 prompt 校验，不创建 Host 或写入项目。
func (a *App) PreviewOutline(path string) (string, error) {
	loaded, err := startup.LoadPromptFile(path)
	if err != nil {
		return "", fmt.Errorf("读取大纲文件失败：%w", err)
	}
	return startup.PrepareQuick(loaded)
}

func (a *App) StartCoCreate(projectRoot, initial string, stage bool, requestID string) (viewmodel.CoCreateStart, error) {
	requestID = operationRequestID(requestID, "cocreate")
	var root, output string
	var err error
	if stage {
		root, output, err = a.currentProjectPaths()
	} else {
		root, output, err = createProjectPaths(projectRoot)
	}
	if err != nil {
		return viewmodel.CoCreateStart{}, err
	}
	initial = strings.TrimSpace(initial)
	if initial == "" {
		return viewmodel.CoCreateStart{}, fmt.Errorf("共创开场输入不能为空")
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if engine := a.engineService(); engine != nil {
		if current := a.openOutputDir(); current != "" && !sameProjectPath(current, output) {
			if err := engine.PrepareProjectSwitch(output); err != nil {
				return viewmodel.CoCreateStart{}, err
			}
		}
	}
	if operation := a.activeOperation(); operation != "" {
		return viewmodel.CoCreateStart{}, fmt.Errorf("当前%s操作仍在进行", operation)
	}
	id, err := a.beginOperation("共创", requestID)
	if err != nil {
		return viewmodel.CoCreateStart{}, err
	}
	ack := viewmodel.CoCreateStart{OperationAck: viewmodel.OperationAck{ProjectID: output, Operation: "cocreate", RequestID: requestID}, Mode: map[bool]string{true: "stage", false: "cold"}[stage]}
	if !stage {
		a.mu.Lock()
		a.projectDir, a.outputDir = root, output
		a.mu.Unlock()
	}
	// 冷启动 Host.New 会初始化 Store；阶段共创沿用当前 Engine Session。
	go a.runCoCreate(id, requestID, root, output, stage, []host.CoCreateMessage{{Role: "user", Content: initial}}, true)
	return ack, nil
}

func (a *App) SendCoCreate(projectRoot, outputDir string, stage bool, history []viewmodel.CoCreateMessage) error {
	coreHistory := make([]host.CoCreateMessage, 0, len(history))
	for _, item := range history {
		coreHistory = append(coreHistory, host.CoCreateMessage{Role: item.Role, Content: item.Content})
	}
	id := a.currentOperationID("共创")
	if id == 0 {
		return fmt.Errorf("当前没有进行中的共创")
	}
	go a.runCoCreateTurn(id, a.currentOperationRequestID("共创"), projectRoot, outputDir, stage, coreHistory, false)
	return nil
}

func (a *App) GetCoCreateRecovery() (viewmodel.CoCreateRecovery, error) {
	_, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.CoCreateRecovery{}, err
	}
	recovery, err := app.ReadCoCreateRecovery(outputDir)
	if err != nil {
		return viewmodel.CoCreateRecovery{}, err
	}
	recovery.ProjectID = outputDir
	if engine := a.engineService(); engine != nil {
		recovery.Generation = engine.RuntimeState().Generation
	}
	return recovery, nil
}

// ResumeCoCreate 从 Core 已落盘的最后一轮历史继续共创；不会恢复未落盘的请求。
func (a *App) ResumeCoCreate(stage bool, history []viewmodel.CoCreateMessage, requestID string) (viewmodel.CoCreateStart, error) {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.CoCreateStart{}, err
	}
	if len(history) == 0 {
		return viewmodel.CoCreateStart{}, fmt.Errorf("没有可恢复的共创历史")
	}
	requestID = operationRequestID(requestID, "cocreate-recovery")
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if operation := a.activeOperation(); operation != "" {
		return viewmodel.CoCreateStart{}, fmt.Errorf("当前%s操作仍在进行", operation)
	}
	id, err := a.beginOperation("共创", requestID)
	if err != nil {
		return viewmodel.CoCreateStart{}, err
	}
	coreHistory := make([]host.CoCreateMessage, 0, len(history))
	for _, item := range history {
		coreHistory = append(coreHistory, host.CoCreateMessage{Role: item.Role, Content: item.Content})
	}
	go a.runCoCreate(id, requestID, projectDir, outputDir, stage, coreHistory, true)
	return viewmodel.CoCreateStart{OperationAck: viewmodel.OperationAck{ProjectID: outputDir, Operation: "cocreate", RequestID: requestID}, Mode: map[bool]string{true: "stage", false: "cold"}[stage]}, nil
}

func (a *App) CompleteCoCreate(stage bool, draft string) (viewmodel.Runtime, error) {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.Runtime{}, err
	}
	if id := a.currentOperationID("共创"); id == 0 {
		return viewmodel.Runtime{}, fmt.Errorf("当前没有进行中的共创")
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	var runtimeState viewmodel.Runtime
	if stage {
		runtimeState, err = a.ensureEngineService().ResumeCoCreate(projectDir, outputDir, draft)
	} else {
		runtimeState, err = a.ensureEngineService().StartPreparedProject(projectDir, outputDir, draft)
	}
	if err != nil {
		return runtimeState, err
	}
	if project, projectErr := a.publishProject(outputDir); projectErr == nil {
		a.emitCreateEvent(viewmodel.CreateEvent{ProjectID: outputDir, Generation: runtimeState.Generation, Operation: "cocreate", RequestID: a.currentOperationRequestID("共创"), State: "completed", Message: "共创已交给 Core，创作已恢复", Project: &project, Runtime: &runtimeState})
	} else {
		a.emitCreateEvent(viewmodel.CreateEvent{ProjectID: outputDir, Generation: runtimeState.Generation, Operation: "cocreate", RequestID: a.currentOperationRequestID("共创"), State: "completed", Message: fmt.Sprintf("共创已交给 Core，但项目视图刷新失败：%v", projectErr), Runtime: &runtimeState})
	}
	a.finishOperation(a.currentOperationID("共创"))
	return runtimeState, nil
}

func (a *App) CancelCoCreate(stage bool) error {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return err
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if stage {
		err = a.ensureEngineService().CancelCoCreate(projectDir, outputDir)
	} else {
		_, err = a.ensureEngineService().StopWriting()
	}
	if err == nil {
		a.finishOperation(a.currentOperationID("共创"))
	}
	return err
}

func (a *App) runCoCreate(id uint64, requestID, projectDir, outputDir string, stage bool, history []host.CoCreateMessage, first bool) {
	a.runCoCreateTurn(id, requestID, projectDir, outputDir, stage, history, first)
}

func (a *App) runCoCreateTurn(id uint64, requestID, projectDir, outputDir string, stage bool, history []host.CoCreateMessage, first bool) {
	engine := a.ensureEngineService()
	run := engine.RunCoCreate
	if !first {
		run = engine.ContinueCoCreate
	}
	var generation uint64
	var reply host.CoCreateReply
	var err error
	reply, generation, err = run(context.Background(), projectDir, outputDir, stage, history, func(eventGeneration uint64, kind, text string) {
		a.emitCoCreateEvent(viewmodel.CoCreateEvent{ProjectID: outputDir, Generation: eventGeneration, RequestID: requestID, State: kind, Kind: kind, Text: text})
	})
	if err != nil {
		a.emitCoCreateEvent(viewmodel.CoCreateEvent{ProjectID: outputDir, Generation: generation, RequestID: requestID, State: "error", Error: err.Error()})
		if first {
			a.finishOperation(id)
		}
		return
	}
	a.emitCoCreateEvent(viewmodel.CoCreateEvent{ProjectID: outputDir, Generation: generation, RequestID: requestID, State: "reply", Reply: reply.Message, Draft: reply.Prompt, Ready: reply.Ready, Suggestions: reply.Suggestions, History: convertHistory(history)})
}

func convertHistory(history []host.CoCreateMessage) []viewmodel.CoCreateMessage {
	result := make([]viewmodel.CoCreateMessage, 0, len(history))
	for _, item := range history {
		result = append(result, viewmodel.CoCreateMessage{Role: item.Role, Content: item.Content})
	}
	return result
}

func (a *App) StartImport(options viewmodel.ImportOptions) (viewmodel.OperationAck, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if operation := a.activeOperation(); operation != "" {
		return viewmodel.OperationAck{}, fmt.Errorf("当前%s操作仍在进行", operation)
	}
	var projectDir, outputDir string
	var err error
	if strings.TrimSpace(options.ProjectRoot) != "" {
		projectDir, outputDir, err = createProjectPaths(options.ProjectRoot)
	} else {
		projectDir, outputDir, err = a.currentProjectPaths()
	}
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	if engine := a.engineService(); engine != nil {
		if current := a.openOutputDir(); current != "" && !sameProjectPath(current, outputDir) {
			if err := engine.PrepareProjectSwitch(outputDir); err != nil {
				return viewmodel.OperationAck{}, err
			}
		}
	}
	a.mu.Lock()
	a.projectDir, a.outputDir = projectDir, outputDir
	a.mu.Unlock()
	id, err := a.beginOperation("导入", "")
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	coreOptions := imp.Options{SourcePath: strings.TrimSpace(options.SourcePath), AutoConfirm: options.AutoConfirm, AcceptSegmentation: options.AcceptSegmentation, StoryResolution: options.StoryResolution, ContinueAfter: options.ContinueAfter, Guidance: options.Guidance}
	ctx := a.contextOrBackground()
	ch, generation, err := a.ensureEngineService().StartImport(ctx, projectDir, outputDir, coreOptions)
	if err != nil {
		a.finishOperation(id)
		return viewmodel.OperationAck{}, err
	}
	go a.consumeImport(id, outputDir, generation, ch)
	return viewmodel.OperationAck{ProjectID: outputDir, Generation: generation, Operation: "import"}, nil
}

func (a *App) consumeImport(id uint64, outputDir string, generation uint64, events <-chan imp.Event) {
	for event := range events {
		status, _ := app.ReadImportStatus(outputDir, outputDir, generation)
		status.Stage, status.Current, status.Total = string(event.Stage), event.Current, event.Total
		status.Message, status.Level, status.Key, status.RetryAt, status.Continued = event.Message, event.Level, event.Key, event.RetryAt, event.Continued
		if event.Err != nil {
			status.Error = event.Err.Error()
		}
		a.emitImportEvent(status)
	}
	a.finishOperation(id)
}

func (a *App) CancelImport() error {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return err
	}
	if a.activeOperation() != "导入" {
		return fmt.Errorf("当前没有进行中的导入")
	}
	return a.ensureEngineService().CancelExclusive(projectDir, outputDir)
}

func (a *App) GetImportStatus() (viewmodel.ImportStatus, error) {
	_, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.ImportStatus{}, err
	}
	generation := uint64(0)
	if engine := a.engineService(); engine != nil {
		generation = engine.RuntimeState().Generation
	}
	return app.ReadImportStatus(outputDir, outputDir, generation)
}

func (a *App) ExportProject(options viewmodel.ExportOptions) (viewmodel.ExportResult, error) {
	_, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.ExportResult{}, err
	}
	return app.Export(a.contextOrBackground(), outputDir, options)
}

func (a *App) GetConfig() (viewmodel.ConfigSnapshot, error) {
	projectDir, _, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	return app.ReadConfigSnapshot(projectDir)
}

func (a *App) SaveBudgetConfig(budget viewmodel.BudgetConfig) (viewmodel.ConfigSnapshot, error) {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if engine := a.engineService(); engine != nil {
		if err := engine.RejectConfigMutation(projectDir, outputDir); err != nil {
			return viewmodel.ConfigSnapshot{}, err
		}
	}
	cfg, err := bootstrap.LoadConfigFromDir(projectDir)
	if err != nil {
		return viewmodel.ConfigSnapshot{}, fmt.Errorf("读取项目配置失败：%w", err)
	}
	cfg.FillDefaults()
	cfg.Budget = bootstrap.BudgetConfig{BookUSD: budget.BookUSD, WarnRatio: budget.WarnRatio, HardStop: budget.HardStop}
	if err := cfg.ValidateBase(); err != nil {
		return viewmodel.ConfigSnapshot{}, fmt.Errorf("预算配置无效：%w", err)
	}
	if err := bootstrap.SaveBudgetConfig(bootstrap.ProjectConfigPathFromDir(projectDir), cfg.Budget); err != nil {
		return viewmodel.ConfigSnapshot{}, fmt.Errorf("保存预算配置失败：%w", err)
	}
	return app.ReadConfigSnapshot(projectDir)
}

func (a *App) SaveProviderConfig(draft viewmodel.ProviderDraft) (viewmodel.ConfigSnapshot, error) {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if err := a.ensureEngineService().ConfigureModels(projectDir, outputDir, app.BuildModelDraft(draft)); err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	return app.ReadConfigSnapshot(projectDir)
}

func (a *App) TestModelConnection(draft viewmodel.ProviderDraft, model string) error {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return err
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	return a.ensureEngineService().TestModelConnection(a.contextOrBackground(), projectDir, outputDir, app.BuildModelDraft(draft), model)
}

func (a *App) SwitchModel(selection viewmodel.ModelSelection) (viewmodel.ConfigSnapshot, error) {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if err := a.ensureEngineService().SwitchModel(projectDir, outputDir, selection); err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	return app.ReadConfigSnapshot(projectDir)
}

func (a *App) SetRoleThinking(setting viewmodel.RoleThinking) (viewmodel.ConfigSnapshot, error) {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if err := a.ensureEngineService().SetRoleThinking(projectDir, outputDir, setting); err != nil {
		return viewmodel.ConfigSnapshot{}, err
	}
	return app.ReadConfigSnapshot(projectDir)
}

func (a *App) GetUsage() (viewmodel.UsageSnapshot, error) {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.UsageSnapshot{}, err
	}
	cfg, err := bootstrap.LoadConfigFromDir(projectDir)
	if err != nil {
		return viewmodel.UsageSnapshot{}, err
	}
	cfg.FillDefaults()
	var live *host.UISnapshot
	if engine := a.engineService(); engine != nil {
		if snapshot, ok := engine.SnapshotForProject(outputDir); ok {
			live = &snapshot
		}
	}
	return app.ReadUsageSnapshot(outputDir, outputDir, live, cfg.Budget)
}

func (a *App) emitCreateEvent(event viewmodel.CreateEvent) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "studio:create-event", event)
	}
}

func (a *App) emitCoCreateEvent(event viewmodel.CoCreateEvent) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "studio:cocreate-event", event)
	}
}

func (a *App) emitImportEvent(event viewmodel.ImportStatus) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "studio:import-event", event)
	}
}

func (a *App) contextOrBackground() context.Context {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (a *App) openOutputDir() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.outputDir
}

func (a *App) currentOperationID(expected string) uint64 {
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	if a.operation != expected {
		return 0
	}
	return a.operationID
}

func (a *App) currentOperationRequestID(expected string) string {
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	if a.operation != expected {
		return ""
	}
	return a.operationRequestID
}

func operationRequestID(requestID, operation string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID != "" {
		return requestID
	}
	return fmt.Sprintf("studio-%s-%d", operation, time.Now().UnixNano())
}
