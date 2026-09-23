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
	events chan host.Event
	stream chan string
	done   chan struct{}
	close  sync.Once
	syncFn func(context.Context) (*revision.Result, error)
	syncs  int
}

func newBridgeTestEngine() *bridgeTestEngine {
	return &bridgeTestEngine{events: make(chan host.Event), stream: make(chan string), done: make(chan struct{})}
}
func (e *bridgeTestEngine) Snapshot() host.UISnapshot { return host.UISnapshot{} }
func (e *bridgeTestEngine) Resume() (string, error)   { return "继续", nil }
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
