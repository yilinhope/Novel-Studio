package app

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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
