package bridge

import (
	"sync"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/store"
	studioapp "github.com/voocel/ainovel-cli/internal/studio/app"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

type bridgeTestEngine struct {
	events chan host.Event
	stream chan string
	done   chan struct{}
	close  sync.Once
}

func newBridgeTestEngine() *bridgeTestEngine {
	return &bridgeTestEngine{events: make(chan host.Event), stream: make(chan string), done: make(chan struct{})}
}
func (e *bridgeTestEngine) Snapshot() host.UISnapshot { return host.UISnapshot{} }
func (e *bridgeTestEngine) Resume() (string, error)   { return "继续", nil }
func (e *bridgeTestEngine) Abort() bool               { return true }
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
