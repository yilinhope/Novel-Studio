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
	if status, err := a.revisionStatusService().GetRevisionStatus(); err == nil && status.ProjectID == outputDir && status.HasUnsynced {
		switch state.State {
		case viewmodel.RuntimeIdle, viewmodel.RuntimePaused, viewmodel.RuntimeStopped, viewmodel.RuntimeCompleted:
			state.State = viewmodel.RuntimeWaitingSync
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
	return engine.ResumeWriting(projectDir, outputDir)
}

func (a *App) PauseWriting() (viewmodel.Runtime, error) {
	engine := a.engineService()
	if engine == nil {
		return viewmodel.Runtime{}, fmt.Errorf("Studio Engine Service 尚未启动")
	}
	return engine.PauseWriting()
}

func (a *App) StopWriting() (viewmodel.Runtime, error) {
	engine := a.engineService()
	if engine == nil {
		return viewmodel.Runtime{}, fmt.Errorf("Studio Engine Service 尚未启动")
	}
	return engine.StopWriting()
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
