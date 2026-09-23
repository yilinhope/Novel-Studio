package bridge

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

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
