package bridge

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/voocel/ainovel-cli/internal/studio/app"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	mu          sync.RWMutex
	ctx         context.Context
	eventCancel context.CancelFunc
	service     app.Service
	engine      *app.EngineService
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
	project, err := a.service.OpenProject(path)
	if err != nil {
		return project, err
	}
	projectDir, err := filepath.Abs(path)
	if err != nil {
		return viewmodel.Project{}, err
	}
	a.mu.Lock()
	a.projectDir = projectDir
	a.outputDir = project.Overview.Path
	a.mu.Unlock()
	return project, nil
}

func (a *App) GetProjectOverview() (viewmodel.Overview, error)  { return a.service.GetProjectOverview() }
func (a *App) GetProjectTree() ([]viewmodel.Node, error)        { return a.service.GetProjectTree() }
func (a *App) GetChapter(number int) (viewmodel.Chapter, error) { return a.service.GetChapter(number) }

// GetRuntimeState 返回已有的 Runtime 投影，不创建 Host 或 Engine Session。
func (a *App) GetRuntimeState() viewmodel.Runtime {
	a.mu.RLock()
	engine := a.engine
	outputDir := a.outputDir
	a.mu.RUnlock()
	if engine == nil {
		return viewmodel.Runtime{ProjectID: outputDir, State: viewmodel.RuntimeIdle}
	}
	state := engine.RuntimeState()
	if state.ProjectID != outputDir {
		return viewmodel.Runtime{ProjectID: outputDir, State: viewmodel.RuntimeIdle}
	}
	return state
}

// ResumeWriting 是 M3-A EngineService 的显式启动入口；打开项目本身仍只读。
func (a *App) ResumeWriting() (viewmodel.Runtime, error) {
	a.mu.RLock()
	projectDir, outputDir, engine := a.projectDir, a.outputDir, a.engine
	a.mu.RUnlock()
	if projectDir == "" || outputDir == "" {
		return viewmodel.Runtime{}, fmt.Errorf("请先打开小说项目")
	}
	if engine == nil {
		return viewmodel.Runtime{}, fmt.Errorf("Studio Engine Service 尚未启动")
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
