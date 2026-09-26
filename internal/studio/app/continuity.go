package app

import (
	"fmt"
	"slices"

	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func (s *Service) GetContinuityTimeline(offset, limit int) (viewmodel.TimelinePage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.TimelinePage{}, err
	}
	items, err := st.World.LoadTimeline()
	if err != nil {
		return viewmodel.TimelinePage{}, fmt.Errorf("读取时间线失败：%w", err)
	}
	result := make([]viewmodel.TimelineEvent, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.TimelineEvent{Chapter: item.Chapter, Time: item.Time, Event: item.Event, Characters: slices.Clone(item.Characters)})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.TimelinePage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetContinuityForeshadow(offset, limit int) (viewmodel.ForeshadowPage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.ForeshadowPage{}, err
	}
	items, err := st.World.LoadForeshadowLedger()
	if err != nil {
		return viewmodel.ForeshadowPage{}, fmt.Errorf("读取伏笔账本失败：%w", err)
	}
	result := make([]viewmodel.ForeshadowEntry, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.ForeshadowEntry{ID: item.ID, Description: item.Description, PlantedAt: item.PlantedAt, Status: item.Status, ResolvedAt: item.ResolvedAt})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.ForeshadowPage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetContinuityRelationships(offset, limit int) (viewmodel.RelationshipPage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.RelationshipPage{}, err
	}
	items, err := st.World.LoadRelationships()
	if err != nil {
		return viewmodel.RelationshipPage{}, fmt.Errorf("读取人物关系失败：%w", err)
	}
	result := make([]viewmodel.RelationshipEntry, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.RelationshipEntry{CharacterA: item.CharacterA, CharacterB: item.CharacterB, Relation: item.Relation, Chapter: item.Chapter})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.RelationshipPage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetContinuityStateChanges(offset, limit int) (viewmodel.StateChangePage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.StateChangePage{}, err
	}
	items, err := st.World.LoadStateChanges()
	if err != nil {
		return viewmodel.StateChangePage{}, fmt.Errorf("读取人物状态变化失败：%w", err)
	}
	result := make([]viewmodel.StateChange, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.StateChange{Chapter: item.Chapter, Entity: item.Entity, Field: item.Field, OldValue: item.OldValue, NewValue: item.NewValue, Reason: item.Reason})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.StateChangePage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetContinuitySnapshots(offset, limit int) (viewmodel.SnapshotPage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.SnapshotPage{}, err
	}
	items, err := st.Characters.LoadLatestSnapshots()
	if err != nil {
		return viewmodel.SnapshotPage{}, fmt.Errorf("读取角色快照失败：%w", err)
	}
	result := make([]viewmodel.CharacterSnapshot, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.CharacterSnapshot{Volume: item.Volume, Arc: item.Arc, Name: item.Name, Status: item.Status, Power: item.Power, Motivation: item.Motivation, Relations: item.Relations})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.SnapshotPage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}

func (s *Service) GetContinuityCast(offset, limit int) (viewmodel.CastPage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.CastPage{}, err
	}
	progress, err := st.Progress.Load()
	if err != nil {
		return viewmodel.CastPage{}, fmt.Errorf("读取进度失败：%w", err)
	}
	if progress == nil || len(progress.CompletedChapters) == 0 {
		return viewmodel.CastPage{PageInfo: viewmodel.PageInfo{Offset: 0, Limit: maxReadPageLimit, Total: 0}, Items: []viewmodel.CastEntry{}}, nil
	}
	items, err := st.BuildCast(progress.CompletedChapters)
	if err != nil {
		return viewmodel.CastPage{}, fmt.Errorf("读取配角出场投影失败：%w", err)
	}
	result := make([]viewmodel.CastEntry, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodel.CastEntry{Name: item.Name, BriefRole: item.BriefRole, FirstSeenChapter: item.FirstSeenChapter, LastSeenChapter: item.LastSeenChapter, AppearanceCount: item.AppearanceCount, AppearanceChapters: slices.Clone(item.AppearanceChapters)})
	}
	start, end, pageLimit := readPageBounds(len(result), offset, limit)
	return viewmodel.CastPage{PageInfo: viewmodel.PageInfo{Offset: start, Limit: pageLimit, Total: len(result), HasMore: pageHasMore(end, len(result))}, Items: result[start:end]}, nil
}
