package bridge

import (
	"context"
	"github.com/voocel/ainovel-cli/internal/studio/app"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	service app.Service
}

func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

func (a *App) SelectProjectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "打开小说项目"})
}

func (a *App) OpenProject(path string) (viewmodel.Project, error) { return a.service.OpenProject(path) }
func (a *App) GetProjectOverview() (viewmodel.Overview, error)    { return a.service.GetProjectOverview() }
func (a *App) GetProjectTree() ([]viewmodel.Node, error)          { return a.service.GetProjectTree() }
func (a *App) GetChapter(number int) (viewmodel.Chapter, error)   { return a.service.GetChapter(number) }
