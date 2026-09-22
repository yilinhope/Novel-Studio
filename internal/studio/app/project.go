package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// Service 仅持有当前小说目录；所有小说数据仍由 Core Store 管理。
type Service struct {
	mu      sync.RWMutex
	current *store.Store
}

func (s *Service) OpenProject(path string) (viewmodel.Project, error) {
	if strings.TrimSpace(path) == "" {
		return viewmodel.Project{}, fmt.Errorf("请选择小说项目目录")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return viewmodel.Project{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return viewmodel.Project{}, fmt.Errorf("无法打开项目目录：%w", err)
	}
	if !info.IsDir() {
		return viewmodel.Project{}, fmt.Errorf("请选择文件夹")
	}
	// 同时接受小说输出目录，以及包含默认 output/novel 的工作目录。
	for _, candidate := range []string{abs, filepath.Join(abs, "output", "novel")} {
		st := store.NewStore(candidate)
		p, err := st.Progress.Load()
		if err != nil {
			return viewmodel.Project{}, fmt.Errorf("读取项目进度失败：%w", err)
		}
		if p == nil {
			continue
		}
		result, err := snapshot(st)
		if err != nil {
			return viewmodel.Project{}, err
		}
		s.mu.Lock()
		s.current = st
		s.mu.Unlock()
		return result, nil
	}
	return viewmodel.Project{}, fmt.Errorf("未找到小说项目，请选择含 meta/progress.json 的小说输出目录")
}

func (s *Service) currentStore() (*store.Store, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == nil {
		return nil, fmt.Errorf("请先打开小说项目")
	}
	return s.current, nil
}

func (s *Service) GetProjectOverview() (viewmodel.Overview, error) {
	st, err := s.currentStore()
	if err != nil {
		return viewmodel.Overview{}, err
	}
	p, err := snapshot(st)
	return p.Overview, err
}

func (s *Service) GetProjectTree() ([]viewmodel.Node, error) {
	st, err := s.currentStore()
	if err != nil {
		return nil, err
	}
	p, err := snapshot(st)
	return p.Tree, err
}

func (s *Service) GetChapter(number int) (viewmodel.Chapter, error) {
	if number < 1 {
		return viewmodel.Chapter{}, fmt.Errorf("章节号必须为正整数")
	}
	st, err := s.currentStore()
	if err != nil {
		return viewmodel.Chapter{}, err
	}
	p, err := snapshot(st)
	if err != nil {
		return viewmodel.Chapter{}, err
	}
	var selected *viewmodel.Node
	var visit func([]viewmodel.Node)
	visit = func(nodes []viewmodel.Node) {
		for _, n := range nodes {
			if n.Kind == "chapter" && n.Chapter == number {
				copy := n
				selected = &copy
			}
			visit(n.Children)
		}
	}
	visit(p.Tree)
	if selected == nil {
		return viewmodel.Chapter{}, fmt.Errorf("项目中没有第 %d 章", number)
	}
	content, err := st.Drafts.LoadChapterText(number)
	if err != nil {
		return viewmodel.Chapter{}, fmt.Errorf("读取章节正文失败：%w", err)
	}
	return viewmodel.Chapter{Number: number, Title: selected.Title, Content: content, WordCount: domain.WordCount(content), HasContent: content != ""}, nil
}

// ConfirmChapterCommit 在前端收到 commit_chapter 成功事件后，重新从磁盘 Store
// 核验 Progress、PendingCommit、终稿与 checkpoint，再返回可刷新的项目快照。
func (s *Service) ConfirmChapterCommit(chapter int, startedAt time.Time) (viewmodel.Project, bool, error) {
	if chapter <= 0 || startedAt.IsZero() {
		return viewmodel.Project{}, false, nil
	}
	current, err := s.currentStore()
	if err != nil {
		return viewmodel.Project{}, false, err
	}
	// 新建 Store 以从磁盘重载只追加 checkpoint 镜像；当前 UI Store 可能早于 Engine 提交。
	fresh := store.NewStore(current.Dir())
	if err := fresh.Checkpoints.InitError(); err != nil {
		return viewmodel.Project{}, false, fmt.Errorf("读取章节 checkpoint 失败：%w", err)
	}
	progress, err := fresh.Progress.Load()
	if err != nil {
		return viewmodel.Project{}, false, fmt.Errorf("复核章节进度失败：%w", err)
	}
	if progress == nil || !slices.Contains(progress.CompletedChapters, chapter) {
		return viewmodel.Project{}, false, nil
	}
	pending, err := fresh.Signals.LoadPendingCommit()
	if err != nil {
		return viewmodel.Project{}, false, fmt.Errorf("复核待提交状态失败：%w", err)
	}
	if pending != nil {
		return viewmodel.Project{}, false, nil
	}
	content, err := fresh.Drafts.LoadChapterText(chapter)
	if err != nil {
		return viewmodel.Project{}, false, fmt.Errorf("复核第 %d 章终稿失败：%w", chapter, err)
	}
	if strings.TrimSpace(content) == "" {
		return viewmodel.Project{}, false, nil
	}
	checkpoint := fresh.Checkpoints.LatestByStep(domain.ChapterScope(chapter), "commit")
	if checkpoint == nil || checkpoint.OccurredAt.Before(startedAt) || checkpoint.Artifact != fmt.Sprintf("chapters/%02d.md", chapter) {
		return viewmodel.Project{}, false, nil
	}
	project, err := snapshot(fresh)
	if err != nil {
		return viewmodel.Project{}, false, err
	}
	s.mu.Lock()
	if s.current != nil && samePath(s.current.Dir(), current.Dir()) {
		s.current = fresh
	}
	s.mu.Unlock()
	return project, true, nil
}

func snapshot(st *store.Store) (viewmodel.Project, error) {
	result := viewmodel.Project{Tree: []viewmodel.Node{}}
	p, err := st.Progress.Load()
	if err != nil {
		return result, fmt.Errorf("读取进度失败：%w", err)
	}
	if p == nil {
		return result, fmt.Errorf("项目进度文件已不存在，请重新打开项目")
	}
	book, err := st.Book.Load()
	if err != nil {
		return result, fmt.Errorf("读取作品信息失败：%w", err)
	}
	o := viewmodel.Overview{Title: filepath.Base(st.Dir()), Path: st.Dir(), Phase: string(p.Phase), Flow: string(p.Flow), CurrentChapter: p.CurrentChapter, CompletedChapters: len(p.CompletedChapters), WordCount: p.TotalWordCount, CurrentVolume: p.CurrentVolume, CurrentArc: p.CurrentArc}
	if book != nil {
		o.Title = book.Title
		o.Synopsis = book.Synopsis
	}
	volumes, err := st.Outline.LoadLayeredOutline()
	if err != nil {
		return result, fmt.Errorf("读取分层大纲失败：%w", err)
	}
	seen := map[int]bool{}
	chapterNode := func(number int, title string) viewmodel.Node {
		seen[number] = true
		if strings.TrimSpace(title) == "" {
			title = fmt.Sprintf("第 %d 章", number)
		}
		return viewmodel.Node{ID: fmt.Sprintf("chapter-%d", number), Kind: "chapter", Title: title, Chapter: number, Children: []viewmodel.Node{}}
	}
	if len(volumes) > 0 {
		number := 0
		for vi, v := range volumes {
			vn := viewmodel.Node{ID: fmt.Sprintf("volume-%d", vi), Kind: "volume", Title: v.Title, Children: []viewmodel.Node{}}
			for ai, a := range v.Arcs {
				an := viewmodel.Node{ID: fmt.Sprintf("arc-%d-%d", vi, ai), Kind: "arc", Title: a.Title, Children: []viewmodel.Node{}}
				for _, c := range a.Chapters {
					number++
					an.Children = append(an.Children, chapterNode(number, c.Title))
				}
				vn.Children = append(vn.Children, an)
			}
			result.Tree = append(result.Tree, vn)
		}
	} else {
		entries, err := st.Outline.LoadOutline()
		if err != nil {
			return result, fmt.Errorf("读取大纲失败：%w", err)
		}
		for _, e := range entries {
			if e.Chapter > 0 && !seen[e.Chapter] {
				result.Tree = append(result.Tree, chapterNode(e.Chapter, e.Title))
			}
		}
	}
	o.PlannedChapters = len(seen)
	// 已完成但没有现存大纲条目的章节仍保留阅读入口。
	completed := append([]int(nil), p.CompletedChapters...)
	sort.Ints(completed)
	for _, number := range completed {
		if number > 0 && !seen[number] {
			result.Tree = append(result.Tree, chapterNode(number, ""))
		}
	}
	result.Overview = o
	return result, nil
}
