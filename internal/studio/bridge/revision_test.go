package bridge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/revision"
	"github.com/voocel/ainovel-cli/internal/store"
	studioapp "github.com/voocel/ainovel-cli/internal/studio/app"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

type bridgeTestEngine struct {
	events        chan host.Event
	stream        chan string
	done          chan struct{}
	close         sync.Once
	syncFn        func(context.Context) (*revision.Result, error)
	syncs         int
	resumeCalls   int
	modeCalls     int
	mode          domain.ChapterAdvanceMode
	advanceCalls  int
	advanceErr    error
	steerCalls    []string
	steerErr      error
	continueCalls []string
	continueErr   error
}

func TestCreateEventKeepsCompletedWhenProjectRefreshFails(t *testing.T) {
	event := viewmodel.CreateEvent{
		State:   "completed",
		Message: "Core 已接受创建请求并开始创作",
	}

	applyCreateProjectRefresh(&event, viewmodel.Project{}, errors.New("Overview 读取失败"))

	if event.State != "completed" {
		t.Fatalf("Core 创建成功后视图刷新失败不得改成 error：%+v", event)
	}
	if event.Error != "" {
		t.Fatalf("视图刷新失败应使用 message/warning，不应伪装成 Core 创建失败：%+v", event)
	}
	if !strings.Contains(event.Message, "项目已创建，但项目视图刷新失败") {
		t.Fatalf("缺少项目视图刷新 warning：%+v", event)
	}
}

func newBridgeTestEngine() *bridgeTestEngine {
	return &bridgeTestEngine{events: make(chan host.Event), stream: make(chan string), done: make(chan struct{})}
}
func (e *bridgeTestEngine) Snapshot() host.UISnapshot { return host.UISnapshot{} }
func (e *bridgeTestEngine) Resume() (string, error)   { e.resumeCalls++; return "继续", nil }
func (e *bridgeTestEngine) SetAdvanceMode(mode domain.ChapterAdvanceMode) error {
	e.modeCalls++
	e.mode = mode
	return nil
}
func (e *bridgeTestEngine) AdvanceOneChapter() error { e.advanceCalls++; return e.advanceErr }
func (e *bridgeTestEngine) Steer(text string) error {
	e.steerCalls = append(e.steerCalls, text)
	return e.steerErr
}
func (e *bridgeTestEngine) Continue(text string) error {
	e.continueCalls = append(e.continueCalls, text)
	return e.continueErr
}
func (e *bridgeTestEngine) SyncChapterRevisions(ctx context.Context) (*revision.Result, error) {
	e.syncs++
	if e.syncFn != nil {
		return e.syncFn(ctx)
	}
	return &revision.Result{}, nil
}
func (e *bridgeTestEngine) Abort() bool { return true }
func (e *bridgeTestEngine) Close() {
	e.close.Do(func() {
		close(e.events)
		close(e.stream)
		close(e.done)
	})
}
func (e *bridgeTestEngine) Events() <-chan host.Event       { return e.events }
func (e *bridgeTestEngine) Stream() <-chan string           { return e.stream }
func (e *bridgeTestEngine) Done() <-chan struct{}           { return e.done }
func (e *bridgeTestEngine) LastRunOutcome() host.RunOutcome { return host.RunOutcomePaused }

func TestRevisionReadAPIsDoNotCreateHost(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	content := "第一章正文"
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, content); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, content, domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}

	a := &App{outputDir: path}
	if _, err := a.service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetChapter(1); err != nil {
		t.Fatal(err)
	}
	initial, err := a.GetRevisionStatus()
	if err != nil || initial.State != viewmodel.RevisionUnknown {
		t.Fatalf("初始状态读取失败：status=%+v err=%v", initial, err)
	}
	checked, err := a.CheckChapterRevisions()
	if err != nil || checked.State != viewmodel.RevisionSynced {
		t.Fatalf("只读检查失败：status=%+v err=%v", checked, err)
	}
	if a.engine != nil {
		t.Fatal("只读章节读取或修订检查隐式创建了 EngineService/Host")
	}
}

func newAdvanceRuntimeFixture(t *testing.T) (*App, *store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "novel")
	st := store.NewStore(path)
	content := "第一章正文"
	for _, err := range []error{
		st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowWriting, CurrentChapter: 2, CompletedChapters: []int{1}}),
		st.RunMeta.Save(domain.RunMeta{AdvanceMode: domain.ChapterAdvanceReview}),
		st.Drafts.SaveFinalChapter(1, content),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, content, domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	a := &App{outputDir: path}
	if _, err := a.service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	return a, st, path
}

func TestGetRuntimeStateDerivesWaitingReviewOnlyAfterCleanRevisionCheck(t *testing.T) {
	a, st, path := newAdvanceRuntimeFixture(t)
	synced, err := a.CheckChapterRevisions()
	if err != nil || synced.State != viewmodel.RevisionSynced {
		t.Fatalf("clean revision check failed: %+v %v", synced, err)
	}
	runtime := a.GetRuntimeState()
	if runtime.State != viewmodel.RuntimeWaitingReview || !runtime.RequiresAdvancePermit || !runtime.CanAdvance {
		t.Fatalf("clean review gate should wait for explicit continuation: %+v", runtime)
	}
	if err := st.Drafts.SaveFinalChapter(1, "第一章正文被修改"); err != nil {
		t.Fatal(err)
	}
	unsynced, err := a.CheckChapterRevisions()
	if err != nil || !unsynced.HasUnsynced {
		t.Fatalf("unsynced revision check failed: %+v %v", unsynced, err)
	}
	runtime = a.GetRuntimeState()
	if runtime.State != viewmodel.RuntimeWaitingSync || runtime.RequiresAdvancePermit || runtime.CanAdvance {
		t.Fatalf("WaitingSync must outrank waiting_review and block permit: %+v", runtime)
	}
	if err := st.Drafts.SaveFinalChapter(1, "第一章正文"); err != nil {
		t.Fatal(err)
	}
	if err := st.Revisions.SavePending(domain.PendingRevision{Stage: domain.RevisionStageRecordsApplied, Items: []domain.PendingRevisionItem{{Chapter: 1, BaseSHA256: "accepted", CurrentSHA256: "pending"}}}); err != nil {
		t.Fatal(err)
	}
	recovery, err := a.CheckChapterRevisions()
	if err != nil || recovery.State != viewmodel.RevisionRecoveryPending {
		t.Fatalf("recovery check failed: %+v %v", recovery, err)
	}
	runtime = a.GetRuntimeState()
	if runtime.State != viewmodel.RuntimeWaitingSync || runtime.CanAdvance || runtime.RequiresAdvancePermit {
		t.Fatalf("pending recovery must remain WaitingSync, not a user permit gate: %+v", runtime)
	}
	if runtime.ProjectID != path {
		t.Fatalf("project id changed: %+v", runtime)
	}
}

func TestGetRuntimeStateDoesNotAttachStaleProjectGate(t *testing.T) {
	a, _, path := newAdvanceRuntimeFixture(t)
	a.outputDir = filepath.Join(t.TempDir(), "other-project")
	runtime := a.GetRuntimeState()
	if runtime.ProjectID == path || runtime.RequiresAdvancePermit || runtime.CanAdvance || runtime.State == viewmodel.RuntimeWaitingReview {
		t.Fatalf("stale projection must not leak across project switch: %+v", runtime)
	}
}

func TestZeroNextChapterNeverProjectsWaitingReview(t *testing.T) {
	a, _, path := newAdvanceRuntimeFixture(t)
	synced, err := a.CheckChapterRevisions()
	if err != nil || synced.State != viewmodel.RevisionSynced {
		t.Fatalf("clean revision check failed: %+v %v", synced, err)
	}
	if err := os.Remove(filepath.Join(path, "meta", "progress.json")); err != nil {
		t.Fatal(err)
	}
	projection, err := a.service.GetAdvanceProjection(synced)
	if err != nil {
		t.Fatal(err)
	}
	if projection.NextChapter != 0 || projection.RequiresAdvancePermit || projection.CanAdvance {
		t.Fatalf("zero next chapter must not be presented as an advance gate: %+v", projection)
	}
	runtime := a.GetRuntimeState()
	if runtime.State == viewmodel.RuntimeWaitingReview || runtime.RequiresAdvancePermit || runtime.CanAdvance {
		t.Fatalf("zero next chapter must not derive WaitingReview: %+v", runtime)
	}
}

func TestExplicitReviewModeCreatesHostWithoutResume(t *testing.T) {
	a, _, path := newAdvanceRuntimeFixture(t)
	a.projectDir = path
	fake := newBridgeTestEngine()
	factories := 0
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { factories++; return fake, nil })
	for _, mode := range []string{"review", "auto"} {
		if _, err := a.SetAdvanceMode(mode); err != nil {
			t.Fatal(err)
		}
	}
	if factories != 1 || fake.modeCalls != 2 || fake.mode != domain.ChapterAdvanceAuto {
		t.Fatalf("模式调用未复用显式 Host: factories=%d calls=%d mode=%s", factories, fake.modeCalls, fake.mode)
	}
	if fake.modeCalls != 2 || fake.resumeCalls != 0 || a.engine.RuntimeState().State == viewmodel.RuntimeRunning {
		t.Fatal("模式切换本身不得 Resume Engine")
	}
	a.engine.Close()
}

func TestAdvanceOneChapterRejectsWaitingSyncBeforeHost(t *testing.T) {
	a, st, path := newAdvanceRuntimeFixture(t)
	a.projectDir = path
	if err := st.Drafts.SaveFinalChapter(1, "第一章正文被手工修改"); err != nil {
		t.Fatal(err)
	}
	factories := 0
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { factories++; return newBridgeTestEngine(), nil })
	if _, err := a.AdvanceOneChapter(); err == nil {
		t.Fatal("WaitingSync 必须拒绝 Next")
	}
	if factories != 0 {
		t.Fatal("WaitingSync 下不得创建 Host")
	}
}

func TestAdvanceOneChapterRejectsProjectedCoreRecoveryGates(t *testing.T) {
	for _, gate := range []string{"AdvanceHold", "PendingCommit", "PendingRewrite"} {
		t.Run(gate, func(t *testing.T) {
			a, st, path := newAdvanceRuntimeFixture(t)
			a.projectDir = path
			switch gate {
			case "AdvanceHold":
				if err := st.RunMeta.SetAdvanceHold(domain.AdvanceHold{After: domain.AdvanceHoldAtBoundary, Reason: "先停一下"}); err != nil {
					t.Fatal(err)
				}
			case "PendingCommit":
				if err := st.Signals.SavePendingCommit(domain.PendingCommit{Chapter: 2, Stage: domain.CommitStageStarted}); err != nil {
					t.Fatal(err)
				}
			case "PendingRewrite":
				if err := st.Progress.SetPendingRewrites([]int{1}, "返工未排空"); err != nil {
					t.Fatal(err)
				}
			}
			factoryCalls := 0
			a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { factoryCalls++; return newBridgeTestEngine(), nil })
			if _, err := a.AdvanceOneChapter(); err == nil {
				t.Fatalf("%s 必须由后端推进门拒绝", gate)
			}
			if factoryCalls != 0 {
				t.Fatalf("%s gate 下不得创建 Host，factory=%d", gate, factoryCalls)
			}
		})
	}
}

func TestAdvanceOneChapterDelegatesCoreGateFailures(t *testing.T) {
	for _, gate := range []string{"AdvanceHold", "PendingCommit", "PendingRewrite"} {
		t.Run(gate, func(t *testing.T) {
			a, _, path := newAdvanceRuntimeFixture(t)
			a.projectDir = path
			fake := newBridgeTestEngine()
			fake.advanceErr = errors.New("Core gate: " + gate)
			a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { return fake, nil })
			if _, err := a.AdvanceOneChapter(); err == nil || !strings.Contains(err.Error(), gate) {
				t.Fatalf("Next 必须返回 Core %s gate 错误，got %v", gate, err)
			}
			if fake.advanceCalls != 1 {
				t.Fatalf("Studio 应委托 Core AdvanceOneChapter，got %d calls", fake.advanceCalls)
			}
			a.engine.Close()
		})
	}
}

func TestAdvanceOneChapterDoesNotChangeReviewEntry(t *testing.T) {
	a, st, path := newAdvanceRuntimeFixture(t)
	a.projectDir = path
	want := domain.ReviewEntry{Chapter: 1, Scope: "chapter", Verdict: "rewrite", Summary: "保留真实审阅结果"}
	if err := st.World.SaveReview(want); err != nil {
		t.Fatal(err)
	}
	fake := newBridgeTestEngine()
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { return fake, nil })
	if _, err := a.AdvanceOneChapter(); err != nil {
		t.Fatal(err)
	}
	got, err := st.World.LoadReview(1)
	if err != nil || got == nil || got.Chapter != want.Chapter || got.Scope != want.Scope || got.Verdict != want.Verdict || got.Summary != want.Summary {
		t.Fatalf("Next 不得改写 ReviewEntry: got=%+v err=%v", got, err)
	}
	if fake.advanceCalls != 1 || fake.resumeCalls != 0 {
		t.Fatalf("Next 必须走 AdvanceOneChapter，不能走 Resume: advance=%d resume=%d", fake.advanceCalls, fake.resumeCalls)
	}
	a.engine.Close()
}

func TestGetRuntimeStateProjectsPendingSteerFromStoreWithoutHost(t *testing.T) {
	a, st, path := newAdvanceRuntimeFixture(t)
	if err := st.RunMeta.SetPendingSteer("崩溃恢复中的用户指令"); err != nil {
		t.Fatal(err)
	}
	runtime := a.GetRuntimeState()
	if runtime.PendingSteer != "崩溃恢复中的用户指令" {
		t.Fatalf("只读 Runtime 应投影 Store 中真实 PendingSteer: %+v", runtime)
	}
	if a.engine != nil {
		t.Fatal("只读 PendingSteer 投影不得创建 Host")
	}
	if runtime.ProjectID != path {
		t.Fatalf("project mismatch: %+v", runtime)
	}
}

func TestSubmitSteerRejectsUnsyncedAndEmptyText(t *testing.T) {
	a, st, path := newAdvanceRuntimeFixture(t)
	a.projectDir = path
	fake := newBridgeTestEngine()
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { return fake, nil })
	if _, err := a.SubmitSteer(" \n "); err == nil || !strings.Contains(err.Error(), "不能为空") {
		t.Fatalf("空文本必须拒绝：%v", err)
	}
	if fake.steerCalls != nil || fake.continueCalls != nil {
		t.Fatal("空文本不得触发 Host")
	}
	if err := st.Drafts.SaveFinalChapter(1, "人工修改"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SubmitSteer("调整节奏"); err == nil {
		t.Fatal("未同步修订必须拒绝 Steer")
	}
	if fake.steerCalls != nil || fake.continueCalls != nil {
		t.Fatal("WaitingSync 不得触发 Host")
	}
}

func TestSaveChapterLeavesAcceptedRecordUnchangedAndMarksWaitingSync(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	original := "第一章接纳正文"
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, original); err != nil {
		t.Fatal(err)
	}
	accepted, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, original, domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{})
	if err != nil {
		t.Fatal(err)
	}

	a := &App{}
	if _, err := a.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	result, err := a.SaveChapter(1, "第一章人工修订正文")
	if err != nil {
		t.Fatal(err)
	}
	if result.Chapter.Content != "第一章人工修订正文" {
		t.Fatalf("返回正文不匹配：%q", result.Chapter.Content)
	}
	if !result.Chapter.CanEdit {
		t.Fatal("已完成且具有接纳基线的正文应可编辑")
	}
	if result.Revision.State != viewmodel.RevisionSavedUnsynced || !result.Revision.HasUnsynced {
		t.Fatalf("保存后应报告待同步，而非 Synced：%+v", result.Revision)
	}
	after, err := st.ChapterRecords.Load(1)
	if err != nil || after == nil || accepted.Revision != after.Revision || accepted.Content != after.Content || accepted.ContentSHA256 != after.ContentSHA256 {
		t.Fatalf("Save 不得更新 ChapterRecord：before=%+v after=%+v err=%v", accepted, after, err)
	}
	if runtime := a.GetRuntimeState(); runtime.State != viewmodel.RuntimeWaitingSync {
		t.Fatalf("保存人工修订后应派生 WaitingSync，得 %+v", runtime)
	}
	if a.engine != nil {
		t.Fatal("SaveChapter 不得隐式创建 Host/Engine Session")
	}
	otherPath := t.TempDir()
	if err := store.NewStore(otherPath).Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.OpenProject(otherPath); err == nil {
		t.Fatal("存在 saved_unsynced 修订时不得切换项目")
	}
	if a.outputDir != path {
		t.Fatalf("被拒绝的项目切换不应改变当前项目：%q", a.outputDir)
	}
}

func TestSaveChapterRespectsExistingCoreBookLeaseWithoutCreatingHost(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, "已接纳正文"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "已接纳正文", domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	project, err := a.OpenProject(path)
	if err != nil {
		t.Fatal(err)
	}
	release, err := host.AcquireBookLease(project.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := a.SaveChapter(1, "另一进程正在写入"); err == nil {
		t.Fatal("已有 Core book lease 时 SaveChapter 不得写入")
	}
	if got, err := st.Drafts.LoadChapterText(1); err != nil || got != "已接纳正文" {
		t.Fatalf("被拒绝的 Save 不得修改正文：got=%q err=%v", got, err)
	}
}

func TestSyncChapterRevisionsCreatesHostOnDemandAndRefreshesProjectChapterAndRevision(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	original, edited := "第一章接纳正文", "第一章人工修订正文"
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, original); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, original, domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	project, err := a.OpenProject(path)
	if err != nil {
		t.Fatal(err)
	}
	fake := newBridgeTestEngine()
	fake.syncFn = func(context.Context) (*revision.Result, error) {
		_, err := st.ChapterRecords.Accept(1, domain.ChapterOriginUser, edited, domain.ChapterFacts{Title: "第一章修订"}, domain.StyleDelta{})
		return &revision.Result{}, err
	}
	factoryCalls := 0
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) {
		factoryCalls++
		return fake, nil
	})
	a.projectDir, a.outputDir = project.ProjectRoot, project.OutputDir
	if _, err := a.SaveChapter(1, edited); err != nil {
		t.Fatal(err)
	}

	result, err := a.SyncChapterRevisions(1)
	if err != nil {
		t.Fatal(err)
	}
	if factoryCalls != 1 || fake.syncs != 1 || !fakeClosed(fake) {
		t.Fatalf("显式 Sync 应按需建 Host、委托并关闭临时会话：factory=%d sync=%d closed=%v", factoryCalls, fake.syncs, fakeClosed(fake))
	}
	if result.Project == nil || result.Project.OutputDir != project.OutputDir || result.Project.Overview.Title == "" {
		t.Fatalf("Sync 应返回重新读取的项目快照：%+v", result.Project)
	}
	if result.Chapter == nil || result.Chapter.Content != edited {
		t.Fatalf("Sync 应重新读取选中章节：%+v", result.Chapter)
	}
	if result.Revision.State != viewmodel.RevisionSynced || result.Revision.HasUnsynced {
		t.Fatalf("成功同步必须经 Store 二次确认后报告 Synced：%+v", result.Revision)
	}
}

func TestSyncChapterRevisionsKeepsSyncedWhenProjectAndChapterRefreshFail(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	original, edited := "第一章接纳正文", "第一章人工修订正文"
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, original); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, original, domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	project, err := a.OpenProject(path)
	if err != nil {
		t.Fatal(err)
	}
	fake := newBridgeTestEngine()
	fake.syncFn = func(context.Context) (*revision.Result, error) {
		_, err := st.ChapterRecords.Accept(1, domain.ChapterOriginUser, edited, domain.ChapterFacts{Title: "第一章修订"}, domain.StyleDelta{})
		if err != nil {
			return nil, err
		}
		return &revision.Result{}, os.WriteFile(filepath.Join(path, "meta", "book.json"), []byte("{"), 0o600)
	}
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { return fake, nil })
	a.projectDir, a.outputDir = project.ProjectRoot, project.OutputDir
	if _, err := a.SaveChapter(1, edited); err != nil {
		t.Fatal(err)
	}

	result, err := a.SyncChapterRevisions(1)
	if err != nil {
		t.Fatalf("章节视图刷新失败不能覆盖 Core Sync 成功：%v", err)
	}
	if result.Revision.State != viewmodel.RevisionSynced || result.Revision.HasUnsynced {
		t.Fatalf("视图刷新失败后仍应返回 Store 确认的 Synced：%+v", result.Revision)
	}
	if result.Project != nil || result.Chapter != nil || !strings.Contains(result.RefreshWarning, "项目视图") || !strings.Contains(result.RefreshWarning, "第 1 章") {
		t.Fatalf("项目与章节刷新失败应单独作为 warning 返回：%+v", result)
	}
}

func TestSyncChapterRevisionsFailureRefreshesPendingRecoveryStatus(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, "已接纳正文"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "已接纳正文", domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	project, err := a.OpenProject(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.SaveChapter(1, "人工修订正文"); err != nil {
		t.Fatal(err)
	}
	fake := newBridgeTestEngine()
	fake.syncFn = func(context.Context) (*revision.Result, error) {
		if err := st.Revisions.SavePending(domain.PendingRevision{
			Stage: domain.RevisionStageRecordsApplied,
			Items: []domain.PendingRevisionItem{{Chapter: 1}},
		}); err != nil {
			return nil, err
		}
		return nil, errors.New("恢复阶段写入失败")
	}
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { return fake, nil })
	a.projectDir, a.outputDir = project.ProjectRoot, project.OutputDir
	if _, err := a.SyncChapterRevisions(1); err == nil {
		t.Fatal("Host Sync 失败不得报告成功")
	}
	status, err := a.GetRevisionStatus()
	if err != nil || status.State != viewmodel.RevisionRecoveryPending || status.PendingStage != string(domain.RevisionStageRecordsApplied) || !status.HasUnsynced {
		t.Fatalf("失败后必须重新检查并继续阻止 Resume：status=%+v err=%v", status, err)
	}
}

func fakeClosed(engine *bridgeTestEngine) bool {
	select {
	case <-engine.done:
		return true
	default:
		return false
	}
}

func TestSaveChapterRejectsWhileEngineIsRunning(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	content := "第一章已接纳正文"
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, content); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, content, domain.ChapterFacts{Title: "第一章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	project, err := a.OpenProject(path)
	if err != nil {
		t.Fatal(err)
	}
	fake := newBridgeTestEngine()
	a.engine = studioapp.NewEngineService(func(_, _ string) (studioapp.EngineSession, error) { return fake, nil })
	a.projectDir, a.outputDir = project.ProjectRoot, project.OutputDir
	if _, err := a.ResumeWriting(); err != nil {
		t.Fatal(err)
	}
	defer a.engine.Close()
	if _, err := a.SaveChapter(1, "不能与 Engine 并发写入"); err == nil {
		t.Fatal("Engine Running 时应拒绝正文保存")
	}
	after, err := st.Drafts.LoadChapterText(1)
	if err != nil || after != content {
		t.Fatalf("运行态拒绝保存不得修改正文：got=%q err=%v", after, err)
	}
}
