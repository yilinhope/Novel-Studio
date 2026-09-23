package app

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/revision"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

type fakeEngineSession struct {
	mu             sync.Mutex
	snapshot       host.UISnapshot
	outcome        host.RunOutcome
	closed         bool
	abortCalls     int
	resumeCalls    int
	syncCalls      int
	syncErr        error
	setModeCalls   int
	mode           domain.ChapterAdvanceMode
	advanceCalls   int
	advanceErr     error
	advanceStarted chan struct{}
	allowAdvance   chan struct{}
	steerCalls     []string
	steerErr       error
	continueCalls  []string
	continueErr    error
	events         chan host.Event
	stream         chan string
	done           chan struct{}
	closeOnce      sync.Once
}

func newFakeEngineSession() *fakeEngineSession {
	return &fakeEngineSession{events: make(chan host.Event), stream: make(chan string), done: make(chan struct{}, 1)}
}
func (f *fakeEngineSession) Snapshot() host.UISnapshot {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snapshot
}
func (f *fakeEngineSession) Resume() (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resumeCalls++
	return "继续", nil
}
func (f *fakeEngineSession) SyncChapterRevisions(context.Context) (*revision.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.syncCalls++
	return &revision.Result{}, f.syncErr
}
func (f *fakeEngineSession) SetAdvanceMode(mode domain.ChapterAdvanceMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setModeCalls++
	f.mode = mode
	return nil
}
func (f *fakeEngineSession) AdvanceOneChapter() error {
	f.mu.Lock()
	f.advanceCalls++
	started, allow, err := f.advanceStarted, f.allowAdvance, f.advanceErr
	f.mu.Unlock()
	if started != nil {
		close(started)
	}
	if allow != nil {
		<-allow
	}
	return err
}
func (f *fakeEngineSession) Steer(text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.steerCalls = append(f.steerCalls, text)
	if f.steerErr != nil {
		f.snapshot.PendingSteer = text
	}
	return f.steerErr
}
func (f *fakeEngineSession) Continue(text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.continueCalls = append(f.continueCalls, text)
	if f.continueErr != nil {
		f.snapshot.PendingSteer = text
	}
	return f.continueErr
}
func (f *fakeEngineSession) Abort() bool {
	f.mu.Lock()
	f.abortCalls++
	f.mu.Unlock()
	return true
}
func (f *fakeEngineSession) Close() {
	f.closeOnce.Do(func() {
		f.mu.Lock()
		f.closed = true
		close(f.events)
		close(f.stream)
		close(f.done)
		f.mu.Unlock()
	})
}
func (f *fakeEngineSession) Events() <-chan host.Event       { return f.events }
func (f *fakeEngineSession) Stream() <-chan string           { return f.stream }
func (f *fakeEngineSession) Done() <-chan struct{}           { return f.done }
func (f *fakeEngineSession) LastRunOutcome() host.RunOutcome { return f.outcome }
func (f *fakeEngineSession) FinishRun()                      { f.done <- struct{}{} }

func TestPrepareProjectSwitchReleasesInactiveSessionOnly(t *testing.T) {
	fake := newFakeEngineSession()
	s := NewEngineService(nil)
	s.engine, s.outputDir, s.runActive = fake, `C:\A\output\novel`, true
	if err := s.PrepareProjectSwitch(`C:\B\output\novel`); err == nil {
		t.Fatal("运行中的项目不应切换")
	}
	if fake.closed {
		t.Fatal("拒绝切换时不应关闭活动会话")
	}
	s.runActive = false
	s.monitorDone = make(chan struct{})
	close(s.monitorDone)
	if err := s.PrepareProjectSwitch(`C:\B\output\novel`); err != nil {
		t.Fatal(err)
	}
	if !fake.closed || s.engine != nil {
		t.Fatal("切换项目前应关闭并释放旧 Host")
	}
}

func TestCompleteRunDistinguishesFailureFromRecoverablePause(t *testing.T) {
	for _, tc := range []struct {
		name      string
		requested viewmodel.RuntimeState
		outcome   host.RunOutcome
		want      viewmodel.RuntimeState
	}{
		{name: "内部故障", outcome: host.RunOutcomeFailed, want: viewmodel.RuntimeError},
		{name: "手动暂停保留暂停态", requested: viewmodel.RuntimePaused, outcome: host.RunOutcomeFailed, want: viewmodel.RuntimePaused},
		{name: "可恢复暂停不受历史错误影响", outcome: host.RunOutcomePaused, want: viewmodel.RuntimePaused},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeEngineSession()
			fake.outcome = tc.outcome
			s := NewEngineService(nil)
			s.engine, s.outputDir, s.runID, s.runActive = fake, `C:\A\output\novel`, 1, true
			s.requestedEnd = tc.requested
			s.lastError = "此前工具调用失败"
			s.runtime = viewmodel.Runtime{State: viewmodel.RuntimeRunning}
			s.completeRun(fake, 0)
			got := s.RuntimeState()
			if got.State != tc.want {
				t.Fatalf("运行终态 = %s，期望 %s", got.State, tc.want)
			}
			if got.State == viewmodel.RuntimePaused && got.Error != "此前工具调用失败" {
				t.Fatalf("暂停态应保留诊断文本但不能误报为运行错误: %+v", got)
			}
		})
	}
}

func TestPauseAndStopRemainInTransitionUntilDone(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		call                 func(*EngineService) (viewmodel.Runtime, error)
		transition, terminal viewmodel.RuntimeState
	}{
		{name: "暂停", call: (*EngineService).PauseWriting, transition: viewmodel.RuntimePausing, terminal: viewmodel.RuntimePaused},
		{name: "停止", call: (*EngineService).StopWriting, transition: viewmodel.RuntimeStopping, terminal: viewmodel.RuntimeStopped},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeEngineSession()
			s := NewEngineService(nil)
			s.engine, s.outputDir, s.runActive, s.runID, s.sessionGeneration = fake, `C:\A\output\novel`, true, 1, 1
			s.runStarted = time.Now()
			s.runtime = viewmodel.Runtime{ProjectID: s.outputDir, Generation: 1, State: viewmodel.RuntimeRunning}
			monitorDone := make(chan struct{})
			s.monitorDone = monitorDone
			go s.monitor(fake, monitorDone, 1)

			if _, err := tc.call(s); err != nil {
				t.Fatal(err)
			}
			if got := s.RuntimeState().State; got != tc.transition {
				t.Fatalf("Done 前状态 = %s，期望过渡态 %s", got, tc.transition)
			}
			fake.FinishRun()
			deadline := time.Now().Add(time.Second)
			for s.RuntimeState().State != tc.terminal && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if got := s.RuntimeState().State; got != tc.terminal {
				t.Fatalf("Done 后状态 = %s，期望终态 %s", got, tc.terminal)
			}
			fake.Close()
			<-monitorDone
		})
	}
}

func TestResumeWritingUsesResumeAndProjectSwitchRejectsRunningHost(t *testing.T) {
	path := t.TempDir()
	if err := store.NewStore(path).Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 1}); err != nil {
		t.Fatal(err)
	}
	fake := newFakeEngineSession()
	called := false
	s := NewEngineService(func(_, _ string) (EngineSession, error) { called = true; return fake, nil })
	if _, err := s.ResumeWriting(path, path); err != nil {
		t.Fatal(err)
	}
	if !called || s.RuntimeState().State != viewmodel.RuntimeRunning {
		t.Fatal("应通过 Host.Resume 启动会话")
	}
	s.Close()
}

func TestSyncChapterRevisionsCreatesTemporaryHostWhenNoneExists(t *testing.T) {
	path := t.TempDir()
	fake := newFakeEngineSession()
	factoryCalls := 0
	s := NewEngineService(func(_, _ string) (EngineSession, error) {
		factoryCalls++
		return fake, nil
	})

	if err := s.SyncChapterRevisions(context.Background(), path, path); err != nil {
		t.Fatal(err)
	}
	if factoryCalls != 1 || fake.syncCalls != 1 || fake.resumeCalls != 0 {
		t.Fatalf("显式 Sync 应按需创建 Host、只委托 Sync 而不 Resume：factory=%d sync=%d resume=%d", factoryCalls, fake.syncCalls, fake.resumeCalls)
	}
	if !fake.closed || s.engine != nil {
		t.Fatal("临时 Sync Host 完成后应关闭且不得注册为 Engine Session")
	}
}

func TestSyncChapterRevisionsReusesPausedHost(t *testing.T) {
	path := t.TempDir()
	fake := newFakeEngineSession()
	s := NewEngineService(nil)
	s.engine, s.outputDir = fake, path
	s.runtime = viewmodel.Runtime{State: viewmodel.RuntimePaused}

	if err := s.SyncChapterRevisions(context.Background(), path, path); err != nil {
		t.Fatal(err)
	}
	if fake.syncCalls != 1 || fake.closed || s.engine != fake {
		t.Fatal("已有同项目 Host 应复用，Sync 后继续保留会话")
	}
}

func TestSyncChapterRevisionsRejectsActiveRuntime(t *testing.T) {
	for _, state := range []viewmodel.RuntimeState{viewmodel.RuntimeRunning, viewmodel.RuntimePausing, viewmodel.RuntimeStopping} {
		t.Run(string(state), func(t *testing.T) {
			fake := newFakeEngineSession()
			s := NewEngineService(nil)
			s.engine, s.outputDir = fake, `C:\项目\output\novel`
			s.runtime = viewmodel.Runtime{State: state}
			s.runActive = state == viewmodel.RuntimeRunning || state == viewmodel.RuntimePausing || state == viewmodel.RuntimeStopping
			if err := s.SyncChapterRevisions(context.Background(), s.outputDir, s.outputDir); err == nil {
				t.Fatal("运行或过渡态必须拒绝同步")
			}
			if fake.syncCalls != 0 {
				t.Fatal("运行或过渡态不得调用 Host Sync")
			}
		})
	}
}

func TestSyncChapterRevisionsPropagatesFailureAndClosesTemporaryHost(t *testing.T) {
	path := t.TempDir()
	fake := newFakeEngineSession()
	fake.syncErr = context.DeadlineExceeded
	s := NewEngineService(func(_, _ string) (EngineSession, error) { return fake, nil })

	if err := s.SyncChapterRevisions(context.Background(), path, path); err == nil {
		t.Fatal("Core Sync 错误不得映射为成功")
	}
	if !fake.closed || fake.syncCalls != 1 {
		t.Fatal("临时 Host 返回失败后仍须关闭，且只委托一次")
	}
}

func TestSetAdvanceModeCreatesHostWithoutResuming(t *testing.T) {
	for _, mode := range []domain.ChapterAdvanceMode{domain.ChapterAdvanceReview, domain.ChapterAdvanceAuto} {
		t.Run(string(mode), func(t *testing.T) {
			path := t.TempDir()
			fake := newFakeEngineSession()
			factories := 0
			s := NewEngineService(func(_, _ string) (EngineSession, error) { factories++; return fake, nil })
			if _, err := s.SetAdvanceMode(path, path, mode); err != nil {
				t.Fatal(err)
			}
			if factories != 1 || fake.setModeCalls != 1 || fake.mode != mode || fake.resumeCalls != 0 {
				t.Fatalf("显式切换应只创建 Host 并委托 SetAdvanceMode，不得 Resume: factory=%d set=%d mode=%s resume=%d", factories, fake.setModeCalls, fake.mode, fake.resumeCalls)
			}
			if s.RuntimeState().State == viewmodel.RuntimeRunning {
				t.Fatal("模式切换不得启动 Engine")
			}
			s.Close()
		})
	}
}

func TestAdvanceOneChapterDelegatesCoreAndPreservesReviewFacts(t *testing.T) {
	path := t.TempDir()
	fake := newFakeEngineSession()
	before := domain.ReviewEntry{Chapter: 1, Scope: "chapter", Verdict: "revise", Summary: "保留"}
	var factories int
	s := NewEngineService(func(_, _ string) (EngineSession, error) { factories++; return fake, nil })
	if _, err := s.AdvanceOneChapter(path, path); err != nil {
		t.Fatal(err)
	}
	if factories != 1 || fake.advanceCalls != 1 || fake.resumeCalls != 0 {
		t.Fatalf("Next 应只委托 Core AdvanceOneChapter: factory=%d advance=%d resume=%d", factories, fake.advanceCalls, fake.resumeCalls)
	}
	if before.Verdict != "revise" || before.Summary != "保留" {
		t.Fatalf("Studio 不得篡改 ReviewEntry: %+v", before)
	}
	s.Close()
}

func TestAdvanceOneChapterWaitsForCoreAcceptanceBeforePublishingRunning(t *testing.T) {
	path := t.TempDir()
	fake := newFakeEngineSession()
	fake.advanceStarted, fake.allowAdvance = make(chan struct{}), make(chan struct{})
	s := NewEngineService(nil)
	s.engine, s.outputDir = fake, path
	s.runtime = viewmodel.Runtime{ProjectID: path, State: viewmodel.RuntimeWaitingReview}
	result := make(chan error, 1)
	go func() { _, err := s.AdvanceOneChapter(path, path); result <- err }()
	<-fake.advanceStarted
	if got := s.RuntimeState().State; got != viewmodel.RuntimeWaitingReview {
		close(fake.allowAdvance)
		t.Fatalf("Core 尚未接受 Next 时不能提前投影 Running，got %s", got)
	}
	close(fake.allowAdvance)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if got := s.RuntimeState().State; got != viewmodel.RuntimeRunning {
		t.Fatalf("Core 接受 Next 后应进入运行态，got %s", got)
	}
}

func TestSubmitSteerRoutesByCoreLifecycle(t *testing.T) {
	for _, tc := range []struct {
		state                   viewmodel.RuntimeState
		phase                   string
		wantSteer, wantContinue bool
	}{
		{state: viewmodel.RuntimeRunning, phase: "writing", wantSteer: true},
		{state: viewmodel.RuntimePaused, phase: "writing", wantContinue: true},
		{state: viewmodel.RuntimeWaitingReview, phase: "writing", wantContinue: true},
		{state: viewmodel.RuntimeIdle, phase: "writing", wantContinue: true},
		{state: viewmodel.RuntimeCompleted, phase: "complete", wantContinue: true},
	} {
		t.Run(string(tc.state), func(t *testing.T) {
			path := t.TempDir()
			fake := newFakeEngineSession()
			fake.snapshot.Phase = tc.phase
			fake.snapshot.IsRunning = tc.state == viewmodel.RuntimeRunning
			s := NewEngineService(nil)
			s.engine, s.outputDir = fake, path
			s.runtime = viewmodel.Runtime{ProjectID: path, State: tc.state}
			if _, err := s.SubmitSteer(path, path, "保留人物动机"); err != nil {
				t.Fatal(err)
			}
			if (len(fake.steerCalls) == 1) != tc.wantSteer || (len(fake.continueCalls) == 1) != tc.wantContinue {
				t.Fatalf("路由不符合 TUI 语义: steer=%v continue=%v", fake.steerCalls, fake.continueCalls)
			}
		})
	}
}

func TestSubmitSteerRejectsUnsafeStatesAndEmptyInput(t *testing.T) {
	for _, state := range []viewmodel.RuntimeState{viewmodel.RuntimeWaitingSync, viewmodel.RuntimePausing, viewmodel.RuntimeStopping} {
		t.Run(string(state), func(t *testing.T) {
			path := t.TempDir()
			fake := newFakeEngineSession()
			s := NewEngineService(nil)
			s.engine, s.outputDir, s.runtime = fake, path, viewmodel.Runtime{ProjectID: path, State: state}
			if _, err := s.SubmitSteer(path, path, "改变方向"); err == nil {
				t.Fatal("当前状态必须拒绝 Steer")
			}
			if len(fake.steerCalls)+len(fake.continueCalls) != 0 {
				t.Fatal("拒绝状态不得调用 Host")
			}
		})
	}
	path := t.TempDir()
	fake := newFakeEngineSession()
	s := NewEngineService(nil)
	s.engine, s.outputDir, s.runtime = fake, path, viewmodel.Runtime{ProjectID: path, State: viewmodel.RuntimePaused}
	if _, err := s.SubmitSteer(path, path, " \n "); err == nil {
		t.Fatal("空文本必须拒绝")
	}
	if len(fake.continueCalls) != 0 {
		t.Fatal("空文本不得调用 Host")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*host.UISnapshot)
	}{
		{name: "阶段共创", mutate: func(s *host.UISnapshot) { s.CoCreating = true }},
		{name: "独占作业", mutate: func(s *host.UISnapshot) { s.Exclusive = "修订导入" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := t.TempDir()
			session := newFakeEngineSession()
			session.snapshot.IsRunning = true
			tc.mutate(&session.snapshot)
			service := NewEngineService(nil)
			service.engine, service.outputDir = session, path
			service.runActive = true
			service.runtime = viewmodel.Runtime{ProjectID: path, State: viewmodel.RuntimeRunning}
			if _, err := service.SubmitSteer(path, path, "修改节奏"); err == nil {
				t.Fatal("exclusive/co-create 必须由 Studio 后端拒绝")
			}
			if len(session.steerCalls)+len(session.continueCalls) != 0 {
				t.Fatal("被拒绝的 Steer 不得进入 Host action")
			}
		})
	}
}

func TestSubmitSteerArbiterErrorKeepsPendingSteerInRuntime(t *testing.T) {
	path := t.TempDir()
	fake := newFakeEngineSession()
	fake.snapshot.Phase = "writing"
	fake.continueErr = context.DeadlineExceeded
	s := NewEngineService(nil)
	s.engine, s.outputDir = fake, path
	s.runtime = viewmodel.Runtime{ProjectID: path, State: viewmodel.RuntimePaused}
	if _, err := s.SubmitSteer(path, path, "保留重试指令"); err == nil {
		t.Fatal("Arbiter 错误必须传播")
	}
	if got := s.RuntimeState().PendingSteer; got != "保留重试指令" {
		t.Fatalf("错误后 Runtime 必须投影 Core PendingSteer，got %q", got)
	}
}
