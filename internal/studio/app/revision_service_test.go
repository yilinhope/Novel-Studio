package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func newRevisionServiceFixture(t *testing.T, runtime viewmodel.Runtime) (string, *RevisionService) {
	t.Helper()
	path := fixture(t)
	st := store.NewStore(path)
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "# 启航\n\n海水拍打着舷窗。", domain.ChapterFacts{Title: "启航"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	project := &Service{}
	if _, err := project.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	return path, NewRevisionService(project, func() viewmodel.Runtime { return runtime })
}

func TestGetRevisionStatusStartsUnknownAndReturnsLastCheck(t *testing.T) {
	_, service := newRevisionServiceFixture(t, viewmodel.Runtime{State: viewmodel.RuntimeIdle})
	status, err := service.GetRevisionStatus()
	if err != nil || status.State != viewmodel.RevisionUnknown || status.HasUnsynced {
		t.Fatalf("初始状态应为 unknown：status=%+v err=%v", status, err)
	}
	checked, err := service.CheckChapterRevisions()
	if err != nil || checked.State != viewmodel.RevisionSynced || checked.CheckedAt.IsZero() {
		t.Fatalf("clean Store 应检查为 synced：status=%+v err=%v", checked, err)
	}
	got, err := service.GetRevisionStatus()
	if err != nil || !reflect.DeepEqual(got, checked) {
		t.Fatalf("应返回最近检查结果：got=%+v want=%+v err=%v", got, checked, err)
	}
}

func TestCheckChapterRevisionsReturnsHashesAndDoesNotWrite(t *testing.T) {
	path, service := newRevisionServiceFixture(t, viewmodel.Runtime{State: viewmodel.RuntimePaused})
	if err := os.WriteFile(filepath.Join(path, "chapters", "01.md"), []byte("# 启航\n\n海水已不再平静。"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeCheck := fingerprint(t, path)
	status, err := service.CheckChapterRevisions()
	if err != nil {
		t.Fatal(err)
	}
	if status.State != viewmodel.RevisionSavedUnsynced || !status.HasUnsynced || len(status.Chapters) != 1 {
		t.Fatalf("应检测到一章未同步：%+v", status)
	}
	chapter := status.Chapters[0]
	if chapter.Chapter != 1 || chapter.AcceptedHash == "" || chapter.CurrentHash == "" || chapter.AcceptedHash == chapter.CurrentHash {
		t.Fatalf("哈希事实不正确：%+v", chapter)
	}
	if !reflect.DeepEqual(beforeCheck, fingerprint(t, path)) {
		t.Fatal("只读修订检查写入了项目文件")
	}
}

func TestCheckChapterRevisionsFindsAllChangedCompletedChapters(t *testing.T) {
	path, service := newRevisionServiceFixture(t, viewmodel.Runtime{State: viewmodel.RuntimePaused})
	st := store.NewStore(path)
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 3, CompletedChapters: []int{1, 2}, Layered: true, TotalChapters: 999, CurrentVolume: 1, CurrentArc: 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(2, "第二章基线"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Accept(2, domain.ChapterOriginGenerated, "第二章基线", domain.ChapterFacts{Title: "第二章"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, "第一章已修改"); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(2, "第二章已修改"); err != nil {
		t.Fatal(err)
	}
	status, err := service.CheckChapterRevisions()
	if err != nil || status.State != viewmodel.RevisionSavedUnsynced || len(status.Chapters) != 2 || status.Chapters[0].Chapter != 1 || status.Chapters[1].Chapter != 2 {
		t.Fatalf("应按序返回全部变更章节：status=%+v err=%v", status, err)
	}
}

func TestCheckChapterRevisionsReportsPendingRecoveryStage(t *testing.T) {
	path, service := newRevisionServiceFixture(t, viewmodel.Runtime{State: viewmodel.RuntimePaused})
	st := store.NewStore(path)
	if err := st.Revisions.SavePending(domain.PendingRevision{
		Stage:     domain.RevisionStageRecordsApplied,
		Items:     []domain.PendingRevisionItem{{Chapter: 1, BaseSHA256: "accepted", CurrentSHA256: "current"}},
		StartedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	status, err := service.CheckChapterRevisions()
	if err != nil || status.State != viewmodel.RevisionRecoveryPending || !status.HasUnsynced || status.PendingStage != string(domain.RevisionStageRecordsApplied) {
		t.Fatalf("应忠实报告待恢复阶段：status=%+v err=%v", status, err)
	}
	if len(status.Chapters) != 1 || status.Chapters[0].AcceptedHash != "accepted" || status.Chapters[0].CurrentHash != "current" {
		t.Fatalf("pending 修订信息错误：%+v", status.Chapters)
	}
}

func TestCheckChapterRevisionsRejectsActiveTransitions(t *testing.T) {
	for _, state := range []viewmodel.RuntimeState{viewmodel.RuntimeRunning, viewmodel.RuntimePausing, viewmodel.RuntimeStopping} {
		t.Run(string(state), func(t *testing.T) {
			_, service := newRevisionServiceFixture(t, viewmodel.Runtime{State: state})
			if _, err := service.CheckChapterRevisions(); err == nil {
				t.Fatalf("%s 时应拒绝检查", state)
			}
		})
	}
}

func TestCheckChapterRevisionsStoresReadFailureWithoutReportingSynced(t *testing.T) {
	path, service := newRevisionServiceFixture(t, viewmodel.Runtime{State: viewmodel.RuntimePaused})
	if err := os.WriteFile(filepath.Join(path, "meta", "chapter_records", "000001.json"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, err := service.CheckChapterRevisions()
	if err == nil || status.State != viewmodel.RevisionError || !status.HasUnsynced {
		t.Fatalf("读取错误不能变成 synced：status=%+v err=%v", status, err)
	}
	cached, getErr := service.GetRevisionStatus()
	if getErr != nil || cached.State != viewmodel.RevisionError || cached.Error == "" {
		t.Fatalf("应缓存错误状态：status=%+v err=%v", cached, getErr)
	}
}
