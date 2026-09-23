package app

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func fixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "中文项目", "output", "novel")
	s := store.NewStore(path)
	for _, err := range []error{
		s.Book.Save(domain.BookMetadata{Title: "星渊纪元", Synopsis: "关于远航与归来的故事。"}),
		s.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 2, CompletedChapters: []int{1}, TotalWordCount: 8, Layered: true, TotalChapters: 999, CurrentVolume: 1, CurrentArc: 1}),
		s.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Title: "灰烬破晓", Arcs: []domain.ArcOutline{{Index: 1, Title: "边陲觉醒", Chapters: []domain.OutlineEntry{{Chapter: 8, Title: "启航"}, {Chapter: 9, Title: "归来"}}}}}}),
		s.Drafts.SaveFinalChapter(1, "# 启航\n\n海水拍打着舷窗。"),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func fingerprint(t *testing.T, path string) map[string][32]byte {
	t.Helper()
	files := map[string][32]byte{}
	err := filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestReadProjectPreservesFilesAndCoreNumbering(t *testing.T) {
	path := fixture(t)
	before := fingerprint(t, path)
	s := &Service{}
	p, err := s.OpenProject(filepath.Dir(filepath.Dir(path)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Overview.Title != "星渊纪元" || p.Overview.PlannedChapters != 2 {
		t.Fatalf("总览不正确：%+v", p.Overview)
	}
	if p.Tree[0].Children[0].Children[0].Chapter != 1 {
		t.Fatal("应沿用 Core 连续章号")
	}
	ch, err := s.GetChapter(1)
	if err != nil || ch.Content != "# 启航\n\n海水拍打着舷窗。" {
		t.Fatalf("正文不匹配：%+v %v", ch, err)
	}
	if ch.CanEdit {
		t.Fatal("缺少 Core 接纳基线的已完成正文不得进入编辑态")
	}
	ch, err = s.GetChapter(2)
	if err != nil || ch.HasContent {
		t.Fatalf("未生成章节状态错误：%+v %v", ch, err)
	}
	if _, err = s.GetChapter(100); err == nil {
		t.Fatal("未知章节应报错")
	}
	if _, err = s.GetChapter(-1); err == nil {
		t.Fatal("负章节号应报错")
	}
	if _, err = s.GetProjectTree(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, fingerprint(t, path)) {
		t.Fatal("只读浏览修改了项目文件")
	}
}

func TestSaveChapterRejectsProjectWriteAlreadyHeldByAnotherStore(t *testing.T) {
	path := fixture(t)
	st := store.NewStore(path)
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "# 启航\n\n海水拍打着舷窗。", domain.ChapterFacts{Title: "启航"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	if _, err := service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	other := store.NewStore(path)
	release, acquired := other.TryAcquireProjectWrite()
	if !acquired {
		t.Fatal("测试应先取得外部项目锁")
	}
	defer release()
	if _, err := service.SaveChapter(1, "不应写入"); err == nil {
		t.Fatal("项目锁被另一 Store 持有时应拒绝保存")
	}
	got, err := st.Drafts.LoadChapterText(1)
	if err != nil || got != "# 启航\n\n海水拍打着舷窗。" {
		t.Fatalf("拒绝保存不得改写正文：got=%q err=%v", got, err)
	}
}

func TestSaveChapterRejectsIncompleteChapter(t *testing.T) {
	path := fixture(t)
	service := &Service{}
	if _, err := service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveChapter(2, "未完成章节"); err == nil {
		t.Fatal("未完成章节不能进入 Core 已完成章节的修订接纳流程")
	}
}

func TestSaveEmptyChapterRemainsEditable(t *testing.T) {
	path := fixture(t)
	st := store.NewStore(path)
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "# 启航\n\n海水拍打着舷窗。", domain.ChapterFacts{Title: "启航"}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	if _, err := service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	chapter, err := service.SaveChapter(1, "")
	if err != nil {
		t.Fatal(err)
	}
	if chapter.HasContent || !chapter.CanEdit {
		t.Fatalf("保存空正文后仍须保留已完成章节的编辑入口：%+v", chapter)
	}
	chapter, err = service.GetChapter(1)
	if err != nil || chapter.HasContent || !chapter.CanEdit {
		t.Fatalf("重新读取空正文时仍须可编辑：chapter=%+v err=%v", chapter, err)
	}
}

func TestFailedOpenPreservesCurrentProject(t *testing.T) {
	s := &Service{}
	if _, err := s.GetProjectOverview(); err == nil {
		t.Fatal("未打开项目应报错")
	}
	path := fixture(t)
	if _, err := s.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenProject(t.TempDir()); err == nil {
		t.Fatal("空目录应拒绝")
	}
	p, err := s.GetProjectOverview()
	if err != nil || p.Path != path {
		t.Fatal("失败切换丢失原项目")
	}
}

func TestProjectRootIsStableForWorkspaceAndOutputSelection(t *testing.T) {
	output := fixture(t)
	root := filepath.Dir(filepath.Dir(output))
	for _, selected := range []string{root, output} {
		s := &Service{}
		project, err := s.OpenProject(selected)
		if err != nil {
			t.Fatal(err)
		}
		if project.ProjectRoot != root || project.OutputDir != output {
			t.Fatalf("选择路径 %q 得到错误的项目路径: %+v", selected, project)
		}
	}
}

func TestPreviewProjectDoesNotSwitchCurrentStore(t *testing.T) {
	first := fixture(t)
	second := fixture(t)
	s := &Service{}
	if _, err := s.OpenProject(first); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreviewProject(second); err != nil {
		t.Fatal(err)
	}
	project, err := s.GetProjectOverview()
	if err != nil || project.Path != first {
		t.Fatalf("只读预览改变了当前项目: %+v %v", project, err)
	}
}

func TestConfirmChapterCommitRechecksStoreFacts(t *testing.T) {
	path := fixture(t)
	service := &Service{}
	if _, err := service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-time.Minute)
	if _, confirmed, err := service.ConfirmChapterCommit(1, started); err != nil || confirmed {
		t.Fatalf("缺少 commit checkpoint 时不应确认: confirmed=%v err=%v", confirmed, err)
	}
	st := store.NewStore(path)
	if _, err := st.Checkpoints.AppendArtifact(domain.ChapterScope(1), "commit", "chapters/01.md"); err != nil {
		t.Fatal(err)
	}
	project, confirmed, err := service.ConfirmChapterCommit(1, started)
	if err != nil || !confirmed || project.Overview.CompletedChapters != 1 {
		t.Fatalf("Store 事实齐备时应确认并返回快照: confirmed=%v project=%+v err=%v", confirmed, project, err)
	}
	if err := st.Signals.SavePendingCommit(domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted}); err != nil {
		t.Fatal(err)
	}
	if _, confirmed, err := service.ConfirmChapterCommit(1, started); err != nil || confirmed {
		t.Fatalf("存在未结束提交时不应确认: confirmed=%v err=%v", confirmed, err)
	}
}

func TestFlatOutlineAndCorruptProgress(t *testing.T) {
	path := t.TempDir()
	st := store.NewStore(path)
	if err := st.Progress.Save(&domain.Progress{CompletedChapters: []int{3}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "第一章"}}); err != nil {
		t.Fatal(err)
	}
	s := &Service{}
	p, err := s.OpenProject(path)
	if err != nil || len(p.Tree) != 2 || p.Tree[1].Chapter != 3 {
		t.Fatalf("扁平大纲或孤立章节错误：%+v %v", p, err)
	}
	if err := os.WriteFile(filepath.Join(path, "meta", "progress.json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetProjectOverview(); err == nil {
		t.Fatal("损坏项目不得伪装为空项目")
	}
}
