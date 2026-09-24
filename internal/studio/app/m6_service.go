package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/host/imp"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// m6Session 是 M6 所需的扩展 Host 边界。EngineSession 保持精简，旧测试替身和
// M3-M5 生命周期不被迫实现创建/导入/配置方法；只有真实 Host 才能进入这些入口。
type m6Session interface {
	EngineSession
	PrepareUserRules(string) error
	StartPrepared(string) error
	CoCreateStream(context.Context, []host.CoCreateMessage, func(string, string)) (host.CoCreateReply, error)
	StageCoCreateStream(context.Context, []host.CoCreateMessage, func(string, string)) (host.CoCreateReply, error)
	PauseForCoCreate() bool
	ResumeFromCoCreate(string) error
	CancelCoCreate()
	ImportFrom(context.Context, imp.Options) (<-chan imp.Event, error)
	ConfigureModels(host.ModelConfigurationDraft) error
	TestModelConnection(context.Context, host.ModelConfigurationDraft, string) error
	SwitchModel(string, string, string) error
	SetRoleThinking(string, string) error
}

func (s *EngineService) lockedM6Host(projectDir, outputDir string) (m6Session, uint64, error) {
	engine, generation, err := s.lockedProjectHost(projectDir, outputDir)
	if err != nil {
		return nil, 0, err
	}
	ext, ok := engine.(m6Session)
	if !ok {
		s.controlMu.Unlock()
		return nil, 0, fmt.Errorf("当前 Engine Session 不支持 M6 Core 操作")
	}
	return ext, generation, nil
}

// StartPreparedProject 复用 Core 的规则快照与启动裁定，供 Quick Start/Outline/冷启动
// Co-create 完成动作使用。调用方必须已经得到用户明确的创建/开始授权。
func (s *EngineService) StartPreparedProject(projectDir, outputDir, prompt string) (viewmodel.Runtime, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return viewmodel.Runtime{}, fmt.Errorf("创作需求不能为空")
	}
	ext, generation, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return viewmodel.Runtime{}, err
	}
	defer s.controlMu.Unlock()
	if err := s.ensureStoppedForControl("开始创作"); err != nil {
		return s.RuntimeState(), err
	}
	if err := ext.PrepareUserRules(prompt); err != nil {
		return s.RuntimeState(), err
	}
	return s.finishCoreStart(ext, generation, ext.StartPrepared(prompt))
}

// RunCoCreate 执行一轮真实 Core 共创。长调用由 Bridge 在 goroutine 中调用，
// 这里仅负责按项目复用 Host、阶段共创前置暂停和 generation 归属。
func (s *EngineService) RunCoCreate(ctx context.Context, projectDir, outputDir string, stage bool, history []host.CoCreateMessage, onProgress func(uint64, string, string)) (host.CoCreateReply, uint64, error) {
	ext, generation, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return host.CoCreateReply{}, 0, err
	}
	if stage {
		if !ext.PauseForCoCreate() {
			s.controlMu.Unlock()
			return host.CoCreateReply{}, 0, fmt.Errorf("当前 Core 不允许进入阶段共创")
		}
	} else {
		if err := s.ensureStoppedForControl("进入共创"); err != nil {
			s.controlMu.Unlock()
			return host.CoCreateReply{}, 0, err
		}
		snapshot := ext.Snapshot()
		if snapshot.Exclusive != "" || snapshot.CoCreating {
			s.controlMu.Unlock()
			return host.CoCreateReply{}, 0, fmt.Errorf("当前 Core 正在%s，不能进入冷启动共创", firstOperation(snapshot))
		}
	}
	s.controlMu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	if stage {
		reply, err := ext.StageCoCreateStream(ctx, history, generationProgress(generation, onProgress))
		return reply, generation, err
	}
	reply, err := ext.CoCreateStream(ctx, history, generationProgress(generation, onProgress))
	return reply, generation, err
}

// ContinueCoCreate 复用已经进入的共创窗口，不重复调用 PauseForCoCreate。
func (s *EngineService) ContinueCoCreate(ctx context.Context, projectDir, outputDir string, stage bool, history []host.CoCreateMessage, onProgress func(uint64, string, string)) (host.CoCreateReply, uint64, error) {
	ext, generation, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return host.CoCreateReply{}, 0, err
	}
	snapshot := ext.Snapshot()
	if stage && !snapshot.CoCreating {
		s.controlMu.Unlock()
		return host.CoCreateReply{}, 0, fmt.Errorf("阶段共创窗口已结束")
	}
	if !stage && (snapshot.Exclusive != "" || snapshot.CoCreating) {
		s.controlMu.Unlock()
		return host.CoCreateReply{}, 0, fmt.Errorf("当前 Core 正在%s，不能开始冷启动共创回合", firstOperation(snapshot))
	}
	s.controlMu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	if stage {
		reply, err := ext.StageCoCreateStream(ctx, history, generationProgress(generation, onProgress))
		return reply, generation, err
	}
	reply, err := ext.CoCreateStream(ctx, history, generationProgress(generation, onProgress))
	return reply, generation, err
}

func generationProgress(generation uint64, callback func(uint64, string, string)) func(string, string) {
	if callback == nil {
		return nil
	}
	return func(kind, text string) { callback(generation, kind, text) }
}

// ResumeCoCreate 把用户确认的阶段 brief 交给 Host.ResumeFromCoCreate。
func (s *EngineService) ResumeCoCreate(projectDir, outputDir, draft string) (viewmodel.Runtime, error) {
	ext, generation, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return viewmodel.Runtime{}, err
	}
	defer s.controlMu.Unlock()
	return s.finishCoreStart(ext, generation, ext.ResumeFromCoCreate(draft))
}

func (s *EngineService) CancelCoCreate(projectDir, outputDir string) error {
	ext, _, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return err
	}
	ext.CancelCoCreate()
	s.controlMu.Unlock()
	return nil
}

// StartImport 只启动 Core imp pipeline；阶段、checkpoint、恢复和写锁仍由 Host/imp 管理。
func (s *EngineService) StartImport(ctx context.Context, projectDir, outputDir string, opts imp.Options) (<-chan imp.Event, uint64, error) {
	ext, generation, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return nil, 0, err
	}
	if err := s.ensureStoppedForControl("导入"); err != nil {
		s.controlMu.Unlock()
		return nil, 0, err
	}
	ch, err := ext.ImportFrom(ctx, opts)
	s.controlMu.Unlock()
	if err != nil {
		return nil, 0, err
	}
	return ch, generation, nil
}

func (s *EngineService) CancelExclusive(projectDir, outputDir string) error {
	ext, _, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return err
	}
	if !ext.Abort() {
		s.controlMu.Unlock()
		return fmt.Errorf("当前 Core 没有可取消的独占任务")
	}
	s.controlMu.Unlock()
	return nil
}

func (s *EngineService) ConfigureModels(projectDir, outputDir string, draft host.ModelConfigurationDraft) error {
	ext, _, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return err
	}
	err = ext.ConfigureModels(draft)
	s.controlMu.Unlock()
	return err
}

func (s *EngineService) TestModelConnection(ctx context.Context, projectDir, outputDir string, draft host.ModelConfigurationDraft, model string) error {
	ext, _, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return err
	}
	err = ext.TestModelConnection(ctx, draft, model)
	s.controlMu.Unlock()
	return err
}

func (s *EngineService) SwitchModel(projectDir, outputDir string, selection viewmodel.ModelSelection) error {
	ext, _, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return err
	}
	role := strings.TrimSpace(selection.Role)
	if role == "" {
		role = "default"
	}
	err = ext.SwitchModel(role, strings.TrimSpace(selection.Provider), strings.TrimSpace(selection.Model))
	s.controlMu.Unlock()
	return err
}

func (s *EngineService) SetRoleThinking(projectDir, outputDir string, setting viewmodel.RoleThinking) error {
	ext, _, err := s.lockedM6Host(projectDir, outputDir)
	if err != nil {
		return err
	}
	err = ext.SetRoleThinking(setting.Role, setting.Level)
	s.controlMu.Unlock()
	return err
}

// SnapshotForProject 是只读快照：已有 Host 时取 live Usage/Agent 数据，没有 Host 时返回 false，
// 调用方再从 Store 的 meta/usage.json 读取；绝不因为 Usage 页面创建 Host。
func (s *EngineService) SnapshotForProject(outputDir string) (host.UISnapshot, bool) {
	s.mu.Lock()
	engine, current := s.engine, s.outputDir
	s.mu.Unlock()
	if engine == nil || !samePath(current, outputDir) {
		return host.UISnapshot{}, false
	}
	return engine.Snapshot(), true
}

// RejectConfigMutation 防止 Budget 这类仅在 Host.New 时建立的配置，在已有
// Engine Session 仍持有旧 BudgetSentinel 时被误报为已即时生效。
func (s *EngineService) RejectConfigMutation(projectDir, outputDir string) error {
	s.controlMu.Lock()
	defer s.controlMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engine == nil {
		return nil
	}
	if !samePath(s.outputDir, outputDir) {
		return fmt.Errorf("另一个项目仍由当前 Engine Session 持有")
	}
	if s.starting || s.runActive || s.runtime.State == viewmodel.RuntimeRunning || s.runtime.State == viewmodel.RuntimePausing || s.runtime.State == viewmodel.RuntimeStopping {
		return fmt.Errorf("创作会话处于%s状态，请先停止后再修改预算", s.runtime.State)
	}
	snapshot := s.engine.Snapshot()
	if snapshot.Exclusive != "" || snapshot.CoCreating {
		return fmt.Errorf("当前 Core 正在%s，请先完成后再修改预算", firstOperation(snapshot))
	}
	return fmt.Errorf("当前 Engine Session 仍持有旧配置，请先停止创作会话后再修改预算")
}

func firstOperation(snapshot host.UISnapshot) string {
	if strings.TrimSpace(snapshot.Exclusive) != "" {
		return snapshot.Exclusive
	}
	if snapshot.CoCreating {
		return "阶段共创"
	}
	return "其他任务"
}

func absPath(value string) string {
	path, err := filepath.Abs(strings.TrimSpace(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	return path
}
