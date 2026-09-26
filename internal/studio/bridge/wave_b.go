package bridge

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/voocel/ainovel-cli/internal/host/sim"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetSimulationSources(request viewmodel.ReadRequest) (viewmodel.SimulationSourcesPage, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	identity, err := a.readIdentity(request)
	if err != nil {
		return viewmodel.SimulationSourcesPage{}, err
	}
	result, err := a.service.GetSimulationSources()
	result.ReadIdentity = identity
	return result, err
}

func (a *App) GetSimulationProfile(request viewmodel.ReadRequest) (viewmodel.SimulationProfilePage, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	identity, err := a.readIdentity(request)
	if err != nil {
		return viewmodel.SimulationProfilePage{}, err
	}
	result, err := a.service.GetSimulationProfile()
	result.ReadIdentity = identity
	return result, err
}

// SelectSimulationProfile 只打开文件选择器；文件内容由 Core ImportSimulationProfile 读取和校验。
func (a *App) SelectSimulationProfile() (string, error) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{Title: "选择仿写画像", Filters: []runtime.FileFilter{{DisplayName: "Simulation Profile", Pattern: "*.json"}}})
}

func (a *App) StartSimulation(request viewmodel.ReadRequest) (viewmodel.OperationAck, error) {
	requestID := operationRequestID(request.RequestID, "simulation")
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if operation := a.activeOperation(); operation != "" {
		return viewmodel.OperationAck{}, fmt.Errorf("当前%s操作仍在进行", operation)
	}
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	identity, err := a.readIdentity(request)
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	id, err := a.beginOperation("仿写画像分析", requestID)
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	ch, generation, err := a.ensureEngineService().StartSimulation(a.contextOrBackground(), projectDir, outputDir, filepath.Join(projectDir, "simulate"))
	if err != nil {
		a.finishOperation(id)
		return viewmodel.OperationAck{}, err
	}
	go a.consumeSimulation(id, outputDir, generation, requestID, ch)
	return viewmodel.OperationAck{ProjectID: identity.ProjectID, Generation: generation, Operation: "simulation", RequestID: requestID}, nil
}

func (a *App) ImportSimulationProfile(request viewmodel.SimulationImportRequest) (viewmodel.OperationAck, error) {
	requestID := operationRequestID(request.RequestID, "simulation-import")
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	if operation := a.activeOperation(); operation != "" {
		return viewmodel.OperationAck{}, fmt.Errorf("当前%s操作仍在进行", operation)
	}
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	identity, err := a.readIdentity(viewmodel.ReadRequest{ProjectID: request.ProjectID, Generation: request.Generation, RequestID: request.RequestID})
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	if strings.TrimSpace(request.Path) == "" {
		return viewmodel.OperationAck{}, fmt.Errorf("仿写画像文件不能为空")
	}
	id, err := a.beginOperation("导入仿写画像", requestID)
	if err != nil {
		return viewmodel.OperationAck{}, err
	}
	ch, generation, err := a.ensureEngineService().StartSimulationImport(a.contextOrBackground(), projectDir, outputDir, request.Path)
	if err != nil {
		a.finishOperation(id)
		return viewmodel.OperationAck{}, err
	}
	go a.consumeSimulation(id, outputDir, generation, requestID, ch)
	return viewmodel.OperationAck{ProjectID: identity.ProjectID, Generation: generation, Operation: "simulation-import", RequestID: requestID}, nil
}

func (a *App) CancelSimulation() error {
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return err
	}
	if op := a.activeOperation(); op != "仿写画像分析" && op != "导入仿写画像" {
		return fmt.Errorf("当前没有进行中的仿写画像任务")
	}
	return a.ensureEngineService().CancelExclusive(projectDir, outputDir)
}

func (a *App) consumeSimulation(id uint64, outputDir string, generation uint64, requestID string, events <-chan sim.Event) {
	for event := range events {
		state := "running"
		if event.Stage == sim.StageDone {
			state = "completed"
		} else if event.Stage == sim.StageError {
			state = "error"
		}
		mapped := viewmodel.SimulationEvent{ProjectID: outputDir, Generation: generation, RequestID: requestID, State: state, Stage: string(event.Stage), Current: event.Current, Total: event.Total, Message: event.Message, Timestamp: event.Time}
		if mapped.Timestamp.IsZero() {
			mapped.Timestamp = time.Now()
		}
		if event.Err != nil {
			mapped.Error = event.Err.Error()
		}
		a.mu.RLock()
		ctx := a.ctx
		a.mu.RUnlock()
		if ctx != nil {
			runtime.EventsEmit(ctx, "studio:simulation-event", mapped)
		}
	}
	a.finishOperation(id)
}

func (a *App) GetRulesWorkspace(request viewmodel.ReadRequest) (viewmodel.RulesWorkspace, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	identity, err := a.readIdentity(request)
	if err != nil {
		return viewmodel.RulesWorkspace{}, err
	}
	result, err := a.service.GetRuleFiles()
	result.ReadIdentity = identity
	return result, err
}

func (a *App) GetRule(request viewmodel.ReadRequest, scope, name string) (viewmodel.RuleFile, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	if _, err := a.readIdentity(request); err != nil {
		return viewmodel.RuleFile{}, err
	}
	return a.service.GetRule(scope, name)
}

func (a *App) SaveRule(request viewmodel.RuleMutationRequest) (viewmodel.RulesWorkspace, error) {
	return a.mutateRule(request, func() error { return a.service.SaveRule(request.Scope, request.Name, request.Content) })
}

func (a *App) DeleteRule(request viewmodel.RuleMutationRequest) (viewmodel.RulesWorkspace, error) {
	return a.mutateRule(request, func() error { return a.service.DeleteRule(request.Scope, request.Name) })
}

func (a *App) RenameRule(request viewmodel.RuleMutationRequest) (viewmodel.RulesWorkspace, error) {
	return a.mutateRule(request, func() error { return a.service.RenameRule(request.Scope, request.Name, request.NewName) })
}

func (a *App) mutateRule(request viewmodel.RuleMutationRequest, mutate func() error) (viewmodel.RulesWorkspace, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.RulesWorkspace{}, err
	}
	if _, err := a.readIdentity(viewmodel.ReadRequest{ProjectID: request.ProjectID, Generation: request.Generation, RequestID: request.RequestID}); err != nil {
		return viewmodel.RulesWorkspace{}, err
	}
	if engine := a.engineService(); engine != nil {
		if err := engine.RejectHostMutation(projectDir, outputDir, "写作规则"); err != nil {
			return viewmodel.RulesWorkspace{}, err
		}
	}
	if err := a.withProjectBookLease(outputDir, mutate); err != nil {
		return viewmodel.RulesWorkspace{}, err
	}
	result, err := a.service.GetRuleFiles()
	if err != nil {
		return viewmodel.RulesWorkspace{}, err
	}
	return a.decorateRules(result), nil
}

func (a *App) GetStyleState(request viewmodel.ReadRequest) (viewmodel.StyleState, error) {
	a.projectMu.RLock()
	defer a.projectMu.RUnlock()
	identity, err := a.readIdentity(request)
	if err != nil {
		return viewmodel.StyleState{}, err
	}
	result, err := a.service.GetStyleState()
	result.ReadIdentity = identity
	return result, err
}

func (a *App) SaveStyleSelection(request viewmodel.StyleMutationRequest) (viewmodel.StyleState, error) {
	return a.mutateStyle(request, func() error { return a.service.SaveStyleSelection(request.Name) })
}

func (a *App) SaveStyleAsset(request viewmodel.StyleMutationRequest) (viewmodel.StyleState, error) {
	return a.mutateStyle(request, func() error { return a.service.SaveStyleAsset(request.Scope, request.Name, request.Content) })
}

func (a *App) DeleteStyleAsset(request viewmodel.StyleMutationRequest) (viewmodel.StyleState, error) {
	return a.mutateStyle(request, func() error { return a.service.DeleteStyleAsset(request.Scope, request.Name) })
}

func (a *App) mutateStyle(request viewmodel.StyleMutationRequest, mutate func() error) (viewmodel.StyleState, error) {
	a.projectMu.Lock()
	defer a.projectMu.Unlock()
	projectDir, outputDir, err := a.currentProjectPaths()
	if err != nil {
		return viewmodel.StyleState{}, err
	}
	if _, err := a.readIdentity(viewmodel.ReadRequest{ProjectID: request.ProjectID, Generation: request.Generation, RequestID: request.RequestID}); err != nil {
		return viewmodel.StyleState{}, err
	}
	if engine := a.engineService(); engine != nil {
		if err := engine.RejectHostMutation(projectDir, outputDir, "文风设置"); err != nil {
			return viewmodel.StyleState{}, err
		}
	}
	if err := a.withProjectBookLease(outputDir, mutate); err != nil {
		return viewmodel.StyleState{}, err
	}
	result, err := a.service.GetStyleState()
	if err != nil {
		return viewmodel.StyleState{}, err
	}
	return a.decorateStyle(result), nil
}

func (a *App) decorateRules(result viewmodel.RulesWorkspace) viewmodel.RulesWorkspace {
	a.mu.RLock()
	result.ProjectID = a.outputDir
	result.Generation = a.projectGeneration
	result.ProjectRoot = a.projectDir
	result.OutputDir = a.outputDir
	a.mu.RUnlock()
	return result
}

func (a *App) decorateStyle(result viewmodel.StyleState) viewmodel.StyleState {
	a.mu.RLock()
	result.ProjectID = a.outputDir
	result.Generation = a.projectGeneration
	result.ProjectRoot = a.projectDir
	result.OutputDir = a.outputDir
	a.mu.RUnlock()
	return result
}
