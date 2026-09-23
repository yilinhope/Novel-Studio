package app

import (
	"sync"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

type fakeEngineSession struct {
	mu         sync.Mutex
	snapshot   host.UISnapshot
	outcome    host.RunOutcome
	closed     bool
	abortCalls int
	events     chan host.Event
	stream     chan string
	done       chan struct{}
	closeOnce  sync.Once
}

func newFakeEngineSession() *fakeEngineSession {
	return &fakeEngineSession{events: make(chan host.Event), stream: make(chan string), done: make(chan struct{}, 1)}
}
func (f *fakeEngineSession) Snapshot() host.UISnapshot {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snapshot
}
func (f *fakeEngineSession) Resume() (string, error) { return "继续", nil }
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
