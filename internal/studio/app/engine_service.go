package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/revision"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// EngineSession 是 Studio 控制与观测所需的最小 Host 边界，供工厂注入和生命周期测试使用。
type EngineSession interface {
	Snapshot() host.UISnapshot
	Resume() (string, error)
	SetAdvanceMode(domain.ChapterAdvanceMode) error
	AdvanceOneChapter() error
	Steer(string) error
	Continue(string) error
	SyncChapterRevisions(context.Context) (*revision.Result, error)
	Abort() bool
	Close()
	Events() <-chan host.Event
	Stream() <-chan string
	Done() <-chan struct{}
	LastRunOutcome() host.RunOutcome
}

type EngineHostFactory func(projectDir, outputDir string) (EngineSession, error)

type StudioEngineEvent struct {
	ProjectID  string             `json:"projectId"`
	Generation uint64             `json:"generation"`
	RunID      uint64             `json:"runId"`
	Sequence   uint64             `json:"sequence"`
	Timestamp  time.Time          `json:"timestamp"`
	Type       string             `json:"type"`
	Log        *host.Event        `json:"log,omitempty"`
	Runtime    *viewmodel.Runtime `json:"runtime,omitempty"`
}

// EngineService 只在用户明确恢复或同步时创建 Host，并独占消费运行会话的通道。
type EngineService struct {
	mu        sync.Mutex
	controlMu sync.Mutex
	closeOnce sync.Once
	factory   EngineHostFactory

	engine            EngineSession
	projectDir        string
	outputDir         string
	starting          bool
	closing           bool
	switching         bool
	switchDone        chan struct{}
	startDone         chan struct{}
	runActive         bool
	runID             uint64
	finishedRun       uint64
	requestedEnd      viewmodel.RuntimeState
	runStarted        time.Time
	baseInput         int
	baseOutput        int
	baseCost          float64
	lastError         string
	runtime           viewmodel.Runtime
	monitorDone       chan struct{}
	sessionGeneration uint64
	eventSequence     uint64
	events            chan StudioEngineEvent
}

func NewEngineService(factory EngineHostFactory) *EngineService {
	if factory == nil {
		factory = newHostForProject
	}
	return &EngineService{
		factory: factory,
		runtime: viewmodel.Runtime{State: viewmodel.RuntimeIdle},
		events:  make(chan StudioEngineEvent, 512),
	}
}

// ResumeWriting 从项目 Store 事实恢复 Engine；只读打开项目不会创建 Host。
func (s *EngineService) ResumeWriting(projectDir, outputDir string) (viewmodel.Runtime, error) {
	projectDir = strings.TrimSpace(projectDir)
	outputDir = strings.TrimSpace(outputDir)
	if projectDir == "" || outputDir == "" {
		return viewmodel.Runtime{}, fmt.Errorf("项目目录与小说目录不能为空")
	}
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return viewmodel.Runtime{}, fmt.Errorf("解析项目目录: %w", err)
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return viewmodel.Runtime{}, fmt.Errorf("解析小说目录: %w", err)
	}
	st := store.NewStore(outputDir)
	progress, err := st.Progress.Load()
	if err != nil {
		return viewmodel.Runtime{}, fmt.Errorf("读取项目进度失败: %w", err)
	}
	if progress == nil || progress.Phase == domain.PhaseComplete {
		return viewmodel.Runtime{}, fmt.Errorf("当前项目没有可恢复的创作任务")
	}

	s.controlMu.Lock()
	defer s.controlMu.Unlock()
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return viewmodel.Runtime{}, fmt.Errorf("Engine Session 正在关闭")
	}
	if s.switching {
		s.mu.Unlock()
		return viewmodel.Runtime{}, fmt.Errorf("Engine Session 正在切换项目")
	}
	if s.starting || s.runActive {
		s.mu.Unlock()
		return viewmodel.Runtime{}, fmt.Errorf("创作会话正在启动或运行")
	}
	if s.engine != nil && !samePath(s.outputDir, outputDir) {
		s.mu.Unlock()
		return viewmodel.Runtime{}, fmt.Errorf("另一个项目仍由当前 Engine Session 持有")
	}
	s.starting = true
	s.startDone = make(chan struct{})
	engine := s.engine
	isNew := engine == nil
	generation := s.sessionGeneration
	s.mu.Unlock()

	if isNew {
		engine, err = s.factory(projectDir, outputDir)
		if err != nil {
			s.mu.Lock()
			s.starting = false
			close(s.startDone)
			s.startDone = nil
			s.lastError = err.Error()
			s.runtime = viewmodel.Runtime{State: viewmodel.RuntimeError, Error: err.Error(), UpdatedAt: time.Now()}
			s.mu.Unlock()
			return viewmodel.Runtime{}, err
		}
		baseline := engine.Snapshot()
		s.mu.Lock()
		if s.closing {
			s.starting = false
			close(s.startDone)
			s.startDone = nil
			s.mu.Unlock()
			engine.Close()
			return viewmodel.Runtime{}, fmt.Errorf("Engine Session 正在关闭")
		}
		s.engine = engine
		s.projectDir = projectDir
		s.outputDir = outputDir
		s.sessionGeneration++
		generation = s.sessionGeneration
		s.baseInput = baseline.TotalInputTokens
		s.baseOutput = baseline.TotalOutputTokens
		s.baseCost = baseline.TotalCostUSD
		s.lastError = ""
		s.monitorDone = make(chan struct{})
		go s.monitor(engine, s.monitorDone, generation)
		s.mu.Unlock()
	} else {
		baseline := engine.Snapshot()
		s.mu.Lock()
		s.projectDir = projectDir
		generation = s.sessionGeneration
		s.baseInput = baseline.TotalInputTokens
		s.baseOutput = baseline.TotalOutputTokens
		s.baseCost = baseline.TotalCostUSD
		s.lastError = ""
		s.mu.Unlock()
	}

	s.mu.Lock()
	s.runID++
	runID := s.runID
	s.runActive = true
	s.requestedEnd = ""
	s.runStarted = time.Now()
	s.runtime = viewmodel.Runtime{State: viewmodel.RuntimeIdle, UpdatedAt: s.runStarted}
	s.mu.Unlock()

	label, err := engine.Resume()
	s.mu.Lock()
	s.starting = false
	close(s.startDone)
	s.startDone = nil
	if err != nil {
		s.runActive = false
		s.lastError = err.Error()
		s.runtime = viewmodel.Runtime{ProjectID: outputDir, Generation: generation, State: viewmodel.RuntimeError, Error: err.Error(), UpdatedAt: time.Now()}
		s.mu.Unlock()
		if isNew {
			s.closeSession(engine)
		}
		return viewmodel.Runtime{}, err
	}
	if label == "" {
		s.runActive = false
		s.runtime = viewmodel.Runtime{ProjectID: outputDir, Generation: generation, State: viewmodel.RuntimeIdle, UpdatedAt: time.Now()}
		s.mu.Unlock()
		if isNew {
			s.closeSession(engine)
		}
		return viewmodel.Runtime{}, fmt.Errorf("Core 未启动可恢复的创作任务")
	}
	if s.finishedRun != runID {
		s.runtime = runtimeFromSnapshot(engine.Snapshot(), viewmodel.RuntimeRunning, s.runStarted, s.baseInput, s.baseOutput, s.baseCost, "")
	}
	s.runtime.ProjectID = outputDir
	s.runtime.Generation = generation
	runtime := s.runtime
	s.mu.Unlock()
	s.emit(generation, "runtime", nil, &runtime)
	return runtime, nil
}

// SyncChapterRevisions 仅由用户明确的 Sync 操作调用，并复用现有 Host 的 Core 同步与恢复流程。
func (s *EngineService) SyncChapterRevisions(ctx context.Context, projectDir, outputDir string) error {
	projectDir = strings.TrimSpace(projectDir)
	outputDir = strings.TrimSpace(outputDir)
	if projectDir == "" || outputDir == "" {
		return fmt.Errorf("项目目录与小说目录不能为空")
	}
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("解析项目目录: %w", err)
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("解析小说目录: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.controlMu.Lock()
	defer s.controlMu.Unlock()
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return fmt.Errorf("Engine Session 正在关闭")
	}
	if s.switching {
		s.mu.Unlock()
		return fmt.Errorf("Engine Session 正在切换项目")
	}
	if s.starting || s.runActive || s.runtime.State == viewmodel.RuntimeRunning ||
		s.runtime.State == viewmodel.RuntimePausing || s.runtime.State == viewmodel.RuntimeStopping {
		state := s.runtime.State
		s.mu.Unlock()
		return fmt.Errorf("创作会话处于%s状态，暂不能同步章节修订", state)
	}
	engine := s.engine
	if engine != nil && !samePath(s.outputDir, outputDir) {
		s.mu.Unlock()
		return fmt.Errorf("另一个项目仍由当前 Engine Session 持有")
	}
	s.mu.Unlock()

	temporary := engine == nil
	if temporary {
		engine, err = s.factory(projectDir, outputDir)
		if err != nil {
			return fmt.Errorf("创建章节同步 Host 失败: %w", err)
		}
		defer engine.Close()
	}
	if _, err := engine.SyncChapterRevisions(ctx); err != nil {
		return fmt.Errorf("同步章节修订失败: %w", err)
	}
	return nil
}

// SetAdvanceMode 只由用户显式切换模式时创建 Host；绝不隐式恢复 Engine。
func (s *EngineService) SetAdvanceMode(projectDir, outputDir string, mode domain.ChapterAdvanceMode) (viewmodel.Runtime, error) {
	if !mode.Valid() {
		return viewmodel.Runtime{}, fmt.Errorf("不支持的章节推进模式：%q", mode)
	}
	engine, generation, err := s.lockedProjectHost(projectDir, outputDir)
	if err != nil {
		return viewmodel.Runtime{}, err
	}
	defer s.controlMu.Unlock()
	if err := engine.SetAdvanceMode(mode); err != nil {
		return s.RuntimeState(), err
	}
	return s.projectRuntime(engine, generation), nil
}

// AdvanceOneChapter 把唯一的推进许可与 Engine 启动交由 Core Host 原子处理。
func (s *EngineService) AdvanceOneChapter(projectDir, outputDir string) (viewmodel.Runtime, error) {
	engine, generation, err := s.lockedProjectHost(projectDir, outputDir)
	if err != nil {
		return viewmodel.Runtime{}, err
	}
	defer s.controlMu.Unlock()
	if err := s.ensureStoppedForControl("继续下一章"); err != nil {
		return s.RuntimeState(), err
	}
	err = engine.AdvanceOneChapter()
	return s.finishCoreStart(engine, generation, err)
}

// SubmitSteer 用当前 Core 生命周期选择 TUI 同款路由；Continue 只携带 Arbiter 干预文本。
func (s *EngineService) SubmitSteer(projectDir, outputDir, text string) (viewmodel.Runtime, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return viewmodel.Runtime{}, fmt.Errorf("创作指令不能为空")
	}
	engine, generation, err := s.lockedProjectHost(projectDir, outputDir)
	if err != nil {
		return viewmodel.Runtime{}, err
	}
	defer s.controlMu.Unlock()

	s.mu.Lock()
	state, runActive := s.runtime.State, s.runActive
	s.mu.Unlock()
	if state == viewmodel.RuntimeWaitingSync || state == viewmodel.RuntimePausing || state == viewmodel.RuntimeStopping {
		return s.RuntimeState(), fmt.Errorf("当前状态为%s，暂不能提交创作指令", state)
	}
	snapshot := engine.Snapshot()
	if snapshot.CoCreating {
		return s.RuntimeState(), fmt.Errorf("阶段共创进行中，请先结束共创")
	}
	if snapshot.Exclusive != "" {
		return s.RuntimeState(), fmt.Errorf("%s进行中，请先完成后再提交创作指令", snapshot.Exclusive)
	}

	if state == viewmodel.RuntimeRunning || runActive {
		if !snapshot.IsRunning {
			return s.RuntimeState(), fmt.Errorf("Studio 与 Core 运行状态尚未收敛，请刷新后重试")
		}
		if err := engine.Steer(text); err != nil {
			return s.projectRuntime(engine, generation), err
		}
		return s.projectRuntime(engine, generation), nil
	}
	if state == viewmodel.RuntimeIdle && snapshot.Phase != string(domain.PhaseWriting) {
		return s.projectRuntime(engine, generation), fmt.Errorf("空闲项目尚无可恢复的写作阶段，不能提交创作指令")
	}
	if state == viewmodel.RuntimeCompleted && snapshot.Phase != string(domain.PhaseComplete) {
		return s.projectRuntime(engine, generation), fmt.Errorf("Core 尚未确认作品完成状态")
	}
	if !isRecoverableSteerState(state) {
		return s.projectRuntime(engine, generation), fmt.Errorf("当前状态为%s，暂不能提交创作指令", state)
	}
	err = engine.Continue(text)
	return s.finishCoreStart(engine, generation, err)
}

func isRecoverableSteerState(state viewmodel.RuntimeState) bool {
	switch state {
	case viewmodel.RuntimeIdle, viewmodel.RuntimePaused, viewmodel.RuntimeWaitingReview, viewmodel.RuntimeStopped, viewmodel.RuntimeError, viewmodel.RuntimeCompleted:
		return true
	default:
		return false
	}
}

// lockedProjectHost 返回持有 controlMu 的同项目 Host；调用方必须恰好 Unlock 一次。
func (s *EngineService) lockedProjectHost(projectDir, outputDir string) (EngineSession, uint64, error) {
	projectDir, outputDir = strings.TrimSpace(projectDir), strings.TrimSpace(outputDir)
	if projectDir == "" || outputDir == "" {
		return nil, 0, fmt.Errorf("项目目录与小说目录不能为空")
	}
	var err error
	projectDir, err = filepath.Abs(projectDir)
	if err != nil {
		return nil, 0, err
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return nil, 0, err
	}
	s.controlMu.Lock()
	s.mu.Lock()
	if s.closing || s.switching || s.starting {
		s.mu.Unlock()
		s.controlMu.Unlock()
		return nil, 0, fmt.Errorf("Engine Session 当前不可执行控制操作")
	}
	engine := s.engine
	if engine != nil && !samePath(s.outputDir, outputDir) {
		s.mu.Unlock()
		s.controlMu.Unlock()
		return nil, 0, fmt.Errorf("另一个项目仍由当前 Engine Session 持有")
	}
	generation := s.sessionGeneration
	if engine != nil {
		s.mu.Unlock()
		return engine, generation, nil
	}
	s.mu.Unlock()
	engine, err = s.factory(projectDir, outputDir)
	if err != nil {
		s.controlMu.Unlock()
		return nil, 0, fmt.Errorf("创建 Studio Host 失败：%w", err)
	}
	baseline := engine.Snapshot()
	s.mu.Lock()
	if s.closing || s.switching {
		s.mu.Unlock()
		s.controlMu.Unlock()
		engine.Close()
		return nil, 0, fmt.Errorf("Engine Session 当前不可执行控制操作")
	}
	s.engine, s.projectDir, s.outputDir = engine, projectDir, outputDir
	s.sessionGeneration++
	generation = s.sessionGeneration
	s.baseInput, s.baseOutput, s.baseCost = baseline.TotalInputTokens, baseline.TotalOutputTokens, baseline.TotalCostUSD
	s.lastError = ""
	s.monitorDone = make(chan struct{})
	monitorDone := s.monitorDone
	s.runtime = runtimeFromSnapshot(baseline, runtimeStateFromHost(baseline), time.Time{}, s.baseInput, s.baseOutput, s.baseCost, "")
	s.runtime.ProjectID, s.runtime.Generation = outputDir, generation
	s.mu.Unlock()
	go s.monitor(engine, monitorDone, generation)
	return engine, generation, nil
}

func runtimeStateFromHost(snapshot host.UISnapshot) viewmodel.RuntimeState {
	switch snapshot.RuntimeState {
	case "running":
		return viewmodel.RuntimeRunning
	case "paused":
		return viewmodel.RuntimePaused
	case "completed":
		return viewmodel.RuntimeCompleted
	default:
		return viewmodel.RuntimeIdle
	}
}

func (s *EngineService) ensureStoppedForControl(action string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runActive || s.runtime.State == viewmodel.RuntimeRunning || s.runtime.State == viewmodel.RuntimePausing || s.runtime.State == viewmodel.RuntimeStopping {
		return fmt.Errorf("创作会话处于%s状态，暂不能%s", s.runtime.State, action)
	}
	return nil
}

func (s *EngineService) beginCoreStart(engine EngineSession, generation uint64) {
	s.mu.Lock()
	s.runID++
	s.runActive, s.requestedEnd, s.runStarted = true, "", time.Now()
	s.runtime = runtimeFromSnapshot(engine.Snapshot(), viewmodel.RuntimeRunning, s.runStarted, s.baseInput, s.baseOutput, s.baseCost, "")
	s.runtime.ProjectID, s.runtime.Generation = s.outputDir, generation
	s.mu.Unlock()
}

func (s *EngineService) finishCoreStart(engine EngineSession, generation uint64, actionErr error) (viewmodel.Runtime, error) {
	if actionErr != nil {
		s.mu.Lock()
		if s.engine == engine && !engine.Snapshot().IsRunning {
			s.runActive = false
			s.runtime = runtimeFromSnapshot(engine.Snapshot(), runtimeStateFromHost(engine.Snapshot()), time.Time{}, s.baseInput, s.baseOutput, s.baseCost, s.lastError)
			s.runtime.ProjectID, s.runtime.Generation = s.outputDir, generation
		}
		runtime := s.runtime
		s.mu.Unlock()
		return runtime, actionErr
	}
	// Core may reject a Next/Continue before it accepts an Engine run. Do not
	// publish Running until the command has returned successfully.
	s.beginCoreStart(engine, generation)
	return s.projectRuntime(engine, generation), nil
}

func (s *EngineService) projectRuntime(engine EngineSession, generation uint64) viewmodel.Runtime {
	snapshot := engine.Snapshot()
	s.mu.Lock()
	if s.engine == engine {
		s.runtime.PendingSteer = snapshot.PendingSteer
		s.runtime.Generation = generation
		s.runtime.ProjectID = s.outputDir
		s.runtime.UpdatedAt = time.Now()
	}
	runtime := s.runtime
	s.mu.Unlock()
	return runtime
}

func (s *EngineService) RuntimeState() viewmodel.Runtime {
	s.mu.Lock()
	runtime := s.runtime
	engine := s.engine
	if s.runActive && !s.runStarted.IsZero() {
		runtime.ElapsedSeconds = int64(time.Since(s.runStarted).Seconds())
		runtime.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
	if engine != nil {
		runtime.PendingSteer = engine.Snapshot().PendingSteer
	}
	return runtime
}

// PauseWriting 请求 Core 安全暂停；paused 终态只由 monitor 收到 Done 后发布。
func (s *EngineService) PauseWriting() (viewmodel.Runtime, error) {
	s.controlMu.Lock()
	defer s.controlMu.Unlock()
	s.mu.Lock()
	if s.starting {
		s.mu.Unlock()
		return viewmodel.Runtime{}, fmt.Errorf("Engine Session 正在启动")
	}
	if s.engine == nil || !s.runActive {
		runtime := s.runtime
		s.mu.Unlock()
		return runtime, fmt.Errorf("当前没有正在运行的创作任务")
	}
	if s.runtime.State == viewmodel.RuntimePausing {
		runtime := s.runtime
		s.mu.Unlock()
		return runtime, nil
	}
	engine := s.engine
	previous := s.runtime.State
	s.requestedEnd = viewmodel.RuntimePaused
	s.runtime.State = viewmodel.RuntimePausing
	s.runtime.UpdatedAt = time.Now()
	s.mu.Unlock()
	accepted := engine.Abort()
	s.mu.Lock()
	if !accepted {
		s.requestedEnd = ""
		if s.engine == engine && s.runActive && s.runtime.State == viewmodel.RuntimePausing {
			s.runtime.State = previous
		}
		runtime := s.runtime
		s.mu.Unlock()
		return runtime, fmt.Errorf("Core 当前无法暂停创作")
	}
	runtime := s.runtime
	generation := s.sessionGeneration
	s.mu.Unlock()
	s.emit(generation, "runtime", nil, &runtime)
	return runtime, nil
}

// StopWriting 请求安全停机。活动运行等待 Done；已暂停会话则关闭 Host 并释放目录租约。
func (s *EngineService) StopWriting() (viewmodel.Runtime, error) {
	s.controlMu.Lock()
	defer s.controlMu.Unlock()
	s.mu.Lock()
	engine := s.engine
	if s.starting {
		s.mu.Unlock()
		return viewmodel.Runtime{}, fmt.Errorf("Engine Session 正在启动")
	}
	if engine == nil {
		runtime := s.runtime
		s.mu.Unlock()
		return runtime, fmt.Errorf("当前没有可停止的 Engine Session")
	}
	if s.runActive {
		previous := s.runtime.State
		s.requestedEnd = viewmodel.RuntimeStopped
		s.runtime.State = viewmodel.RuntimeStopping
		s.runtime.UpdatedAt = time.Now()
		s.mu.Unlock()
		accepted := true
		if previous != viewmodel.RuntimePausing {
			accepted = engine.Abort()
		}
		s.mu.Lock()
		if !accepted {
			s.requestedEnd = ""
			if s.engine == engine && s.runActive && s.runtime.State == viewmodel.RuntimeStopping {
				s.runtime.State = previous
			}
			runtime := s.runtime
			s.mu.Unlock()
			return runtime, fmt.Errorf("Core 当前无法停止创作")
		}
		runtime := s.runtime
		generation := s.sessionGeneration
		s.mu.Unlock()
		s.emit(generation, "runtime", nil, &runtime)
		return runtime, nil
	}
	monitorDone := s.monitorDone
	generation := s.sessionGeneration
	s.mu.Unlock()

	engine.Close()
	s.mu.Lock()
	if s.engine == engine {
		s.engine = nil
		s.runtime.State = viewmodel.RuntimeStopped
		s.runtime.UpdatedAt = time.Now()
	}
	runtime := s.runtime
	s.mu.Unlock()
	if monitorDone != nil {
		<-monitorDone
	}
	s.emit(generation, "runtime", nil, &runtime)
	return runtime, nil
}

// PrepareProjectSwitch 在目标项目只读校验成功后，阻止活动会话切换并关闭非活动旧 Host。
func (s *EngineService) PrepareProjectSwitch(nextOutputDir string) error {
	s.controlMu.Lock()
	s.mu.Lock()
	engine := s.engine
	if engine == nil || samePath(s.outputDir, nextOutputDir) {
		s.mu.Unlock()
		s.controlMu.Unlock()
		return nil
	}
	if s.starting || s.runActive || s.runtime.State == viewmodel.RuntimeRunning ||
		s.runtime.State == viewmodel.RuntimePausing || s.runtime.State == viewmodel.RuntimeStopping {
		s.mu.Unlock()
		s.controlMu.Unlock()
		return fmt.Errorf("当前项目仍在创作中，请先停止当前 Engine Session 后再切换项目")
	}
	// Import / revision / simulation 与阶段共创可能在 Engine 非 running 时持有 Core
	// exclusive；此时关闭 Host 会取消长任务或丢失共创上下文，项目切换必须等待它们收敛。
	snapshot := engine.Snapshot()
	if snapshot.Exclusive != "" || snapshot.CoCreating {
		action := snapshot.Exclusive
		if action == "" {
			action = "阶段共创"
		}
		s.mu.Unlock()
		s.controlMu.Unlock()
		return fmt.Errorf("当前项目仍在%s，请先完成后再切换项目", action)
	}
	s.switching = true
	s.switchDone = make(chan struct{})
	s.engine = nil
	s.runtime = viewmodel.Runtime{State: viewmodel.RuntimeIdle, UpdatedAt: time.Now()}
	monitorDone := s.monitorDone
	s.mu.Unlock()
	s.controlMu.Unlock()

	engine.Close()
	if monitorDone != nil {
		<-monitorDone
	}

	s.mu.Lock()
	s.switching = false
	close(s.switchDone)
	s.switchDone = nil
	s.mu.Unlock()
	return nil
}

func (s *EngineService) Events() <-chan StudioEngineEvent { return s.events }

// Close 等待 Core 收尾后释放 Host 与小说目录租约。
func (s *EngineService) Close() {
	s.closeOnce.Do(func() {
		s.controlMu.Lock()
		s.mu.Lock()
		s.closing = true
		startDone := s.startDone
		engine := s.engine
		monitorDone := s.monitorDone
		switchDone := s.switchDone
		if engine != nil && s.runActive {
			s.requestedEnd = viewmodel.RuntimeStopping
			s.runtime.State = viewmodel.RuntimeStopping
			s.runtime.UpdatedAt = time.Now()
		}
		s.mu.Unlock()
		s.controlMu.Unlock()
		if switchDone != nil {
			<-switchDone
			return
		}

		if startDone != nil {
			<-startDone
		}
		if engine == nil {
			return
		}
		s.closeSession(engine)
		if monitorDone != nil {
			<-monitorDone
		}
	})
}

func (s *EngineService) closeSession(engine EngineSession) {
	engine.Close()
	s.mu.Lock()
	if s.engine == engine {
		s.engine = nil
		s.runActive = false
	}
	s.mu.Unlock()
}

func (s *EngineService) monitor(engine EngineSession, done chan struct{}, generation uint64) {
	defer close(done)
	events := engine.Events()
	stream := engine.Stream()
	finished := engine.Done()
	for events != nil || stream != nil || finished != nil {
		select {
		case event, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			s.publishLog(generation, event)
			if event.Agent == "arbiter" || event.Category == "SYSTEM" || event.Category == "ERROR" {
				s.refreshRuntime(engine, generation)
			}
			if event.Category == "ERROR" || event.Level == "error" {
				s.mu.Lock()
				s.lastError = event.Detail
				if s.lastError == "" {
					s.lastError = event.Summary
				}
				s.mu.Unlock()
			}
			if event.Category == "TOOL" || (event.Category == "MODEL" && !event.FinishedAt.IsZero()) {
				s.refreshRuntime(engine, generation)
			}
		case _, ok := <-stream:
			if !ok {
				stream = nil
			}
		case _, ok := <-finished:
			if !ok {
				finished = nil
				continue
			}
			drainHostEvents(events, func(event host.Event) {
				s.publishLog(generation, event)
				if event.Category == "ERROR" || event.Level == "error" {
					s.mu.Lock()
					s.lastError = event.Detail
					if s.lastError == "" {
						s.lastError = event.Summary
					}
					s.mu.Unlock()
				}
			})
			s.controlMu.Lock()
			s.completeRun(engine, generation)
			s.controlMu.Unlock()
		}
	}
}

func (s *EngineService) completeRun(engine EngineSession, generation uint64) {
	s.mu.Lock()
	runID := s.runID
	requestedEnd := s.requestedEnd
	startedAt, baseInput, baseOutput, baseCost := s.runStarted, s.baseInput, s.baseOutput, s.baseCost
	lastError := s.lastError
	s.mu.Unlock()

	core := engine.Snapshot()
	state := viewmodel.RuntimePaused
	switch {
	case requestedEnd == viewmodel.RuntimeStopping || requestedEnd == viewmodel.RuntimeStopped:
		state = viewmodel.RuntimeStopped
	case requestedEnd == viewmodel.RuntimePaused:
		state = viewmodel.RuntimePaused
	case core.Phase == string(domain.PhaseComplete):
		state = viewmodel.RuntimeCompleted
	case engine.LastRunOutcome() == host.RunOutcomeFailed:
		state = viewmodel.RuntimeError
	case core.RuntimeState == "idle" && !core.IsRunning:
		state = viewmodel.RuntimeIdle
	}

	shouldClose := false
	s.mu.Lock()
	if s.engine == engine && s.runID == runID {
		s.runActive = false
		s.finishedRun = runID
		shouldClose = requestedEnd == viewmodel.RuntimeStopping || requestedEnd == viewmodel.RuntimeStopped
		s.requestedEnd = ""
		s.runtime = runtimeFromSnapshot(core, state, startedAt, baseInput, baseOutput, baseCost, lastError)
		s.runtime.ProjectID = s.outputDir
		s.runtime.Generation = generation
	}
	runtime := s.runtime
	s.mu.Unlock()
	s.emit(generation, "runtime", nil, &runtime)
	if shouldClose {
		s.closeSession(engine)
	}
}

func (s *EngineService) refreshRuntime(engine EngineSession, generation uint64) {
	core := engine.Snapshot()
	s.mu.Lock()
	if s.engine != engine {
		s.mu.Unlock()
		return
	}
	if !s.runActive {
		s.runtime.PendingSteer = core.PendingSteer
		s.runtime.UpdatedAt = time.Now()
		runtime := s.runtime
		s.mu.Unlock()
		s.emit(generation, "runtime", nil, &runtime)
		return
	}
	state := s.runtime.State
	if state != viewmodel.RuntimePausing && state != viewmodel.RuntimeStopping {
		state = viewmodel.RuntimeRunning
	}
	s.runtime = runtimeFromSnapshot(core, state, s.runStarted, s.baseInput, s.baseOutput, s.baseCost, s.lastError)
	s.runtime.ProjectID = s.outputDir
	s.runtime.Generation = generation
	runtime := s.runtime
	s.mu.Unlock()
	s.emit(generation, "runtime", nil, &runtime)
}

func (s *EngineService) publishLog(generation uint64, event host.Event) {
	s.emit(generation, "log", &event, nil)
}

func (s *EngineService) emit(generation uint64, eventType string, log *host.Event, runtime *viewmodel.Runtime) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventSequence++
	event := StudioEngineEvent{
		ProjectID:  s.outputDir,
		Generation: generation,
		RunID:      s.runID,
		Sequence:   s.eventSequence,
		Timestamp:  time.Now(),
		Type:       eventType,
		Log:        log,
		Runtime:    runtime,
	}
	select {
	case s.events <- event:
	default:
		select {
		case <-s.events:
		default:
		}
		select {
		case s.events <- event:
		default:
		}
	}
}

func runtimeFromSnapshot(core host.UISnapshot, state viewmodel.RuntimeState, startedAt time.Time, baseInput, baseOutput int, baseCost float64, lastError string) viewmodel.Runtime {
	runtime := viewmodel.Runtime{
		State:               state,
		Phase:               core.Phase,
		Flow:                core.Flow,
		Chapter:             core.InProgressChapter,
		ProjectInputTokens:  core.TotalInputTokens,
		ProjectOutputTokens: core.TotalOutputTokens,
		InputTokens:         max(0, core.TotalInputTokens-baseInput),
		OutputTokens:        max(0, core.TotalOutputTokens-baseOutput),
		RunCostUSD:          maxFloat(0, core.TotalCostUSD-baseCost),
		ProjectCostUSD:      core.TotalCostUSD,
		Error:               lastError,
		UpdatedAt:           time.Now(),
		Agents:              make([]viewmodel.RuntimeAgent, 0, len(core.Agents)),
	}
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	if state != viewmodel.RuntimeIdle && !startedAt.IsZero() {
		runtime.ElapsedSeconds = int64(time.Since(startedAt).Seconds())
	}
	agents := append([]host.AgentSnapshot(nil), core.Agents...)
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })
	for _, agent := range agents {
		runtime.Agents = append(runtime.Agents, viewmodel.RuntimeAgent{
			Name: agent.Name, State: agent.State, Tool: agent.Tool, Summary: agent.Summary,
		})
		if runtime.Agent == "" && agent.State == "working" {
			runtime.Agent = agent.Name
			runtime.Step = agent.Tool
		}
	}
	runtime.PendingSteer = core.PendingSteer
	if runtime.Chapter == 0 {
		runtime.Chapter = core.CurrentChapter
	}
	return runtime
}

func drainHostEvents(events <-chan host.Event, consume func(host.Event)) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			consume(event)
		default:
			return
		}
	}
}

func newHostForProject(projectDir, outputDir string) (EngineSession, error) {
	cfg, err := bootstrap.LoadConfigFromDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("加载项目配置失败: %w", err)
	}
	cfg.OutputDir = outputDir
	cfg.FillDefaults()
	rules.EnsureHomeRulesDir()
	bundle := assets.Load(cfg.Style, assets.DefaultLoadOptions(outputDir))
	engine, err := host.New(cfg, bundle,
		host.WithFileLog("studio.log", false),
		host.WithConfigPath(bootstrap.EffectiveConfigPathFromDir(projectDir)))
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func samePath(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	a, b = filepath.Clean(a), filepath.Clean(b)
	if filepath.Separator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
