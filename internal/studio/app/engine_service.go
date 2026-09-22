package app

import (
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
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

type EngineHostFactory func(projectDir, outputDir string) (*host.Host, error)

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

// EngineService 在用户明确恢复创作时才创建 Host，并独占消费它的运行通道。
type EngineService struct {
	mu        sync.Mutex
	controlMu sync.Mutex
	closeOnce sync.Once
	factory   EngineHostFactory

	engine            *host.Host
	projectDir        string
	outputDir         string
	starting          bool
	closing           bool
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

// ResumeWriting 从项目 Store 事实恢复 Engine；此方法是创建 Host 的唯一入口。
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

func (s *EngineService) RuntimeState() viewmodel.Runtime {
	s.mu.Lock()
	defer s.mu.Unlock()
	runtime := s.runtime
	if s.runActive && !s.runStarted.IsZero() {
		runtime.ElapsedSeconds = int64(time.Since(s.runStarted).Seconds())
		runtime.UpdatedAt = time.Now()
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
		if engine != nil && s.runActive {
			s.requestedEnd = viewmodel.RuntimeStopping
			s.runtime.State = viewmodel.RuntimeStopping
			s.runtime.UpdatedAt = time.Now()
		}
		s.mu.Unlock()
		s.controlMu.Unlock()

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

func (s *EngineService) closeSession(engine *host.Host) {
	engine.Close()
	s.mu.Lock()
	if s.engine == engine {
		s.engine = nil
		s.runActive = false
	}
	s.mu.Unlock()
}

func (s *EngineService) monitor(engine *host.Host, done chan struct{}, generation uint64) {
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

func (s *EngineService) completeRun(engine *host.Host, generation uint64) {
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
	case core.Phase == string(domain.PhaseComplete):
		state = viewmodel.RuntimeCompleted
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

func (s *EngineService) refreshRuntime(engine *host.Host, generation uint64) {
	core := engine.Snapshot()
	s.mu.Lock()
	if s.engine != engine || !s.runActive {
		s.mu.Unlock()
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
		if agent.State == "working" {
			runtime.Agent = agent.Name
			runtime.Step = agent.Tool
			break
		}
	}
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

func newHostForProject(projectDir, outputDir string) (*host.Host, error) {
	cfg, err := bootstrap.LoadConfigFromDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("加载项目配置失败: %w", err)
	}
	cfg.OutputDir = outputDir
	cfg.FillDefaults()
	rules.EnsureHomeRulesDir()
	bundle := assets.Load(cfg.Style, assets.DefaultLoadOptions(outputDir))
	engine, err := host.New(cfg, bundle, host.WithFileLog("studio.log", false))
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
