package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func parityFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "故事项目", "output", "novel")
	st := store.NewStore(path)
	if err := st.Book.Save(domain.BookMetadata{Title: "星海回声", Synopsis: "一段穿越风暴的旅程。"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SavePremise("主角必须在风暴中找到归途。"); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{1, 2}, CurrentChapter: 3, Layered: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{Name: "林渡", Role: "主角", Description: "航海者", Arc: "学会信任", Traits: []string{"谨慎"}, Tier: "core"}, {Name: "周漪", Role: "配角", Description: "守门人"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveWorldRules([]domain.WorldRule{{Category: "geography", Rule: "潮汐受月门影响", Boundary: "不可逆转"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "出航", CoreEvent: "离港", Hook: "风暴来临", Scenes: []string{"码头"}}, {Chapter: 2, Title: "风眼"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Title: "潮门", Theme: "寻找归途", Arcs: []domain.ArcOutline{{Index: 1, Title: "离港", Goal: "离开故乡", Chapters: []domain.OutlineEntry{{Title: "出航"}, {Title: "风眼"}}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "回到故乡", OpenThreads: []string{"月门真相"}, EstimatedScale: "一卷", LastUpdated: 2}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 1, Title: "出航", Summary: "离港", Characters: []string{"林渡"}, KeyEvents: []string{"风暴"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1, Title: "离港", Summary: "离开故乡"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveVolumeSummary(domain.VolumeSummary{Volume: 1, Title: "潮门", Summary: "寻找归途"}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveTimeline([]domain.TimelineEvent{{Chapter: 1, Time: "清晨", Event: "离港", Characters: []string{"林渡"}}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveForeshadowLedger([]domain.ForeshadowEntry{{ID: "moon", Description: "月门真相", PlantedAt: 1, Status: "planted"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveRelationships([]domain.RelationshipEntry{{CharacterA: "林渡", CharacterB: "沈遥", Relation: "同伴", Chapter: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveStateChanges([]domain.StateChange{{Chapter: 1, Entity: "林渡", Field: "location", NewValue: "海上", Reason: "离港"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.SaveSnapshots(1, 1, []domain.CharacterSnapshot{{Volume: 1, Arc: 1, Name: "林渡", Status: "坚定", Motivation: "回家"}}); err != nil {
		t.Fatal(err)
	}
	for _, chapter := range []int{1, 2} {
		if err := st.ChapterRecords.Save(domain.ChapterRecord{Version: domain.ChapterRecordVersion, Chapter: chapter, Revision: 1, Origin: domain.ChapterOriginGenerated, Content: "正文", ContentSHA256: domain.ChapterContentSHA256("正文"), AcceptedAt: time.Now(), Facts: domain.ChapterFacts{Characters: []string{"沈遥"}, CastIntros: []domain.CastIntro{{Name: "沈遥", BriefRole: "测绘师"}}}}); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestStoryAndContinuityReadFromCoreStoreWithPaging(t *testing.T) {
	path := parityFixture(t)
	service := &Service{}
	if _, err := service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	premise, err := service.GetStoryPremise()
	if err != nil || premise.Premise != "主角必须在风暴中找到归途。" || !premise.BookAvailable {
		t.Fatalf("故事前提必须来自 Core Store：%+v %v", premise, err)
	}
	characters, err := service.GetStoryCharacters(0, 1)
	if err != nil || characters.Total != 2 || len(characters.Items) != 1 || !characters.HasMore {
		t.Fatalf("人物分页不正确：%+v %v", characters, err)
	}
	layered, err := service.GetStoryLayeredOutline(0, 50)
	if err != nil || layered.Total != 1 || layered.Items[0].Arcs[0].ChapterCount != 2 {
		t.Fatalf("分层大纲应只返回卷弧元数据：%+v %v", layered, err)
	}
	chapters, err := service.GetStoryLayeredChapters(1, 1, 0, 1)
	if err != nil || chapters.Total != 2 || len(chapters.Items) != 1 || chapters.Items[0].Chapter != 1 {
		t.Fatalf("分层章节惰性分页不正确：%+v %v", chapters, err)
	}
	timeline, err := service.GetContinuityTimeline(0, 50)
	if err != nil || timeline.Total != 1 || timeline.Items[0].Event != "离港" {
		t.Fatalf("时间线必须来自 Core Store：%+v %v", timeline, err)
	}
	cast, err := service.GetContinuityCast(0, 50)
	if err != nil || cast.Total != 1 || cast.Items[0].Name != "沈遥" || cast.Items[0].FirstSeenChapter != 1 {
		t.Fatalf("配角投影必须来自 Core BuildCast：%+v %v", cast, err)
	}
	empty, err := service.GetStorySummaries("chapter", 1, 1)
	if err != nil || empty.Total != 2 || len(empty.Chapters) != 0 {
		t.Fatalf("缺失摘要应为空而不是推导正文：%+v %v", empty, err)
	}
}

func TestParityReadMissingOptionalDataIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty")
	st := store.NewStore(path)
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	if _, err := service.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	premise, err := service.GetStoryPremise()
	if err != nil || premise.BookAvailable || premise.PremiseAvailable {
		t.Fatalf("缺失可选资料应返回空状态：%+v %v", premise, err)
	}
	compass, err := service.GetStoryCompass()
	if err != nil || compass.Available {
		t.Fatalf("缺失 Compass 应返回不可用状态：%+v %v", compass, err)
	}
	snapshots, err := service.GetContinuitySnapshots(0, 50)
	if err != nil || snapshots.Total != 0 || len(snapshots.Items) != 0 {
		t.Fatalf("缺失快照应返回空页：%+v %v", snapshots, err)
	}
}

func TestContinuityLargeListUsesBoundedPages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large")
	st := store.NewStore(path)
	items := make([]domain.TimelineEvent, 137)
	for i := range items {
		items[i] = domain.TimelineEvent{Chapter: i + 1, Time: "清晨", Event: "事件"}
	}
	if err := st.World.SaveTimeline(items); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	if _, err := service.OpenProject(path); err != nil {
		t.Fatal(err)
	}

	first, err := service.GetContinuityTimeline(0, 1000)
	if err != nil || first.Total != 137 || first.Limit != maxReadPageLimit || len(first.Items) != maxReadPageLimit || !first.HasMore {
		t.Fatalf("首屏应受最大页大小约束：total=%d limit=%d items=%d more=%v err=%v", first.Total, first.Limit, len(first.Items), first.HasMore, err)
	}
	last, err := service.GetContinuityTimeline(100, maxReadPageLimit)
	if err != nil || last.Offset != 100 || len(last.Items) != 37 || last.HasMore {
		t.Fatalf("尾页分页边界不正确：offset=%d items=%d more=%v err=%v", last.Offset, len(last.Items), last.HasMore, err)
	}
}
