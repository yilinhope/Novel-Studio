package app

import (
	"fmt"
	"slices"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// GetStoryPremise 只读取 Core 作品信息和 premise.md，不创建 Host。
func (s *Service) GetStoryPremise() (viewmodel.StoryPremise, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.StoryPremise{}, err
	}
	book, err := st.Book.Load()
	if err != nil {
		return viewmodel.StoryPremise{}, fmt.Errorf("读取作品信息失败：%w", err)
	}
	premise, err := st.Outline.LoadPremise()
	if err != nil {
		return viewmodel.StoryPremise{}, fmt.Errorf("读取故事前提失败：%w", err)
	}
	result := viewmodel.StoryPremise{Premise: premise, PremiseAvailable: premise != ""}
	if book != nil {
		result.Book = viewmodel.BookMetadata{Title: book.Title, Synopsis: book.Synopsis}
		result.BookAvailable = true
	}
	return result, nil
}

func (s *Service) GetStoryCharacters(offset, limit int) (viewmodel.CharacterPage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.CharacterPage{}, err
	}
	items, err := st.Characters.Load()
	if err != nil {
		return viewmodel.CharacterPage{}, fmt.Errorf("读取人物档案失败：%w", err)
	}
	result := make([]viewmodel.Character, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.Character{Name: item.Name, Aliases: slices.Clone(item.Aliases), Role: item.Role, Description: item.Description, Arc: item.Arc, Traits: slices.Clone(item.Traits), Tier: item.Tier})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.CharacterPage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetStoryWorldRules(offset, limit int) (viewmodel.WorldRulePage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.WorldRulePage{}, err
	}
	items, err := st.World.LoadWorldRules()
	if err != nil {
		return viewmodel.WorldRulePage{}, fmt.Errorf("读取世界规则失败：%w", err)
	}
	result := make([]viewmodel.WorldRule, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.WorldRule{Category: item.Category, Rule: item.Rule, Boundary: item.Boundary})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.WorldRulePage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetStoryOutline(offset, limit int) (viewmodel.OutlinePage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.OutlinePage{}, err
	}
	items, err := st.Outline.LoadOutline()
	if err != nil {
		return viewmodel.OutlinePage{}, fmt.Errorf("读取扁平大纲失败：%w", err)
	}
	result := mapOutlineEntries(items)
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.OutlinePage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetStoryLayeredOutline(offset, limit int) (viewmodel.LayeredOutlinePage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.LayeredOutlinePage{}, err
	}
	items, err := st.Outline.LoadLayeredOutline()
	if err != nil {
		return viewmodel.LayeredOutlinePage{}, fmt.Errorf("读取分层大纲失败：%w", err)
	}
	result := make([]viewmodel.LayeredVolume, 0, len(items))
	for _, volume := range items {
		view := viewmodel.LayeredVolume{Index: volume.Index, Title: volume.Title, Theme: volume.Theme, Final: volume.Final, Arcs: make([]viewmodel.LayeredArc, 0, len(volume.Arcs))}
		for _, arc := range volume.Arcs {
			view.Arcs = append(view.Arcs, viewmodel.LayeredArc{Index: arc.Index, Title: arc.Title, Goal: arc.Goal, EstimatedChapters: arc.EstimatedChapters, ChapterCount: len(arc.Chapters)})
		}
		result = append(result, view)
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.LayeredOutlinePage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetStoryLayeredChapters(volume, arc, offset, limit int) (viewmodel.LayeredChapterPage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.LayeredChapterPage{}, err
	}
	volumes, err := st.Outline.LoadLayeredOutline()
	if err != nil {
		return viewmodel.LayeredChapterPage{}, fmt.Errorf("读取分层大纲章节失败：%w", err)
	}
	var entries []domain.OutlineEntry
	global := 1
	found := false
	for _, currentVolume := range volumes {
		for _, currentArc := range currentVolume.Arcs {
			if currentVolume.Index == volume && currentArc.Index == arc {
				entries = slices.Clone(currentArc.Chapters)
				for i := range entries {
					entries[i].Chapter = global + i
				}
				found = true
			}
			global += len(currentArc.Chapters)
		}
	}
	if !found {
		return viewmodel.LayeredChapterPage{Volume: volume, Arc: arc, PageInfo: viewmodel.PageInfo{Offset: 0, Limit: maxReadPageLimit, Total: 0}, Items: []viewmodel.OutlineEntry{}}, nil
	}
	result := mapOutlineEntries(entries)
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.LayeredChapterPage{Volume: volume, Arc: arc, PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetStoryCompass() (viewmodel.StoryCompass, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.StoryCompass{}, err
	}
	compass, err := st.Outline.LoadCompass()
	if err != nil {
		return viewmodel.StoryCompass{}, fmt.Errorf("读取故事指南针失败：%w", err)
	}
	if compass == nil {
		return viewmodel.StoryCompass{}, nil
	}
	return viewmodel.StoryCompass{EndingDirection: compass.EndingDirection, OpenThreads: slices.Clone(compass.OpenThreads), EstimatedScale: compass.EstimatedScale, LastUpdated: compass.LastUpdated, Available: true}, nil
}

func (s *Service) GetStorySummaries(scope string, offset, limit int) (viewmodel.StorySummaryPage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.StorySummaryPage{}, err
	}
	scope = normalizeSummaryScope(scope)
	switch scope {
	case "chapter":
		chapters, err := storyChapterNumbers(st)
		if err != nil {
			return viewmodel.StorySummaryPage{}, err
		}
		start, end, pageLimit := readPageBounds(len(chapters), offset, limit)
		result := make([]viewmodel.ChapterSummary, 0, end-start)
		for _, chapter := range chapters[start:end] {
			sum, loadErr := st.Summaries.LoadSummary(chapter)
			if loadErr != nil {
				return viewmodel.StorySummaryPage{}, fmt.Errorf("读取第 %d 章摘要失败：%w", chapter, loadErr)
			}
			if sum != nil {
				result = append(result, viewmodel.ChapterSummary{Chapter: sum.Chapter, Title: sum.Title, Summary: sum.Summary, Characters: slices.Clone(sum.Characters), KeyEvents: slices.Clone(sum.KeyEvents)})
			}
		}
		return viewmodel.StorySummaryPage{Scope: scope, PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(chapters), HasMore: pageHasMore(end, len(chapters))}, Chapters: result}, nil
	case "arc":
		volumes, err := st.Outline.LoadLayeredOutline()
		if err != nil {
			return viewmodel.StorySummaryPage{}, fmt.Errorf("读取分层大纲失败：%w", err)
		}
		refs := make([][2]int, 0)
		for _, volume := range volumes {
			for _, arc := range volume.Arcs {
				refs = append(refs, [2]int{volume.Index, arc.Index})
			}
		}
		start, end, pageLimit := readPageBounds(len(refs), offset, limit)
		result := make([]viewmodel.ArcSummary, 0, end-start)
		for _, ref := range refs[start:end] {
			sum, loadErr := st.Summaries.LoadArcSummary(ref[0], ref[1])
			if loadErr != nil {
				return viewmodel.StorySummaryPage{}, fmt.Errorf("读取 V%d A%d 摘要失败：%w", ref[0], ref[1], loadErr)
			}
			if sum != nil {
				result = append(result, viewmodel.ArcSummary{Volume: sum.Volume, Arc: sum.Arc, Title: sum.Title, Summary: sum.Summary, KeyEvents: slices.Clone(sum.KeyEvents)})
			}
		}
		return viewmodel.StorySummaryPage{Scope: scope, PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(refs), HasMore: pageHasMore(end, len(refs))}, Arcs: result}, nil
	case "volume":
		volumes, err := st.Outline.LoadLayeredOutline()
		if err != nil {
			return viewmodel.StorySummaryPage{}, fmt.Errorf("读取分层大纲失败：%w", err)
		}
		start, end, pageLimit := readPageBounds(len(volumes), offset, limit)
		result := make([]viewmodel.VolumeSummary, 0, end-start)
		for _, volume := range volumes[start:end] {
			sum, loadErr := st.Summaries.LoadVolumeSummary(volume.Index)
			if loadErr != nil {
				return viewmodel.StorySummaryPage{}, fmt.Errorf("读取第 %d 卷摘要失败：%w", volume.Index, loadErr)
			}
			if sum != nil {
				result = append(result, viewmodel.VolumeSummary{Volume: sum.Volume, Title: sum.Title, Summary: sum.Summary, KeyEvents: slices.Clone(sum.KeyEvents)})
			}
		}
		return viewmodel.StorySummaryPage{Scope: scope, PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(volumes), HasMore: pageHasMore(end, len(volumes))}, Volumes: result}, nil
	default:
		return viewmodel.StorySummaryPage{}, fmt.Errorf("不支持的摘要范围：%s", scope)
	}
}

func normalizeSummaryScope(scope string) string {
	if scope == "" {
		return "chapter"
	}
	return scope
}

func storyChapterNumbers(st *store.Store) ([]int, error) {
	entries, err := st.Outline.LoadOutline()
	if err != nil {
		return nil, fmt.Errorf("读取章节编号失败：%w", err)
	}
	seen := make(map[int]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Chapter > 0 {
			seen[entry.Chapter] = struct{}{}
		}
	}
	progress, err := st.Progress.Load()
	if err != nil {
		return nil, fmt.Errorf("读取进度失败：%w", err)
	}
	if progress != nil {
		for _, chapter := range progress.CompletedChapters {
			if chapter > 0 {
				seen[chapter] = struct{}{}
			}
		}
	}
	chapters := make([]int, 0, len(seen))
	for chapter := range seen {
		chapters = append(chapters, chapter)
	}
	slices.Sort(chapters)
	return chapters, nil
}

func mapOutlineEntries(entries []domain.OutlineEntry) []viewmodel.OutlineEntry {
	result := make([]viewmodel.OutlineEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, viewmodel.OutlineEntry{Chapter: entry.Chapter, Title: entry.Title, CoreEvent: entry.CoreEvent, Hook: entry.Hook, Scenes: slices.Clone(entry.Scenes)})
	}
	return result
}
