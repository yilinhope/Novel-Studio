package host

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/domain"
	storepkg "github.com/voocel/ainovel-cli/internal/store"
)

func TestUpgradeProjectMigratesLegacyBook(t *testing.T) {
	dir := t.TempDir()
	st := storepkg.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	premise := "# 《寿元账》\n\n## 核心冲突\n\n凡人以寿元换取灵性，在求生与守住人性之间挣扎。\n\n## 主角目标\n\n活下去。"
	if err := st.Outline.SavePremise(premise); err != nil {
		t.Fatalf("SavePremise: %v", err)
	}
	progress := []byte(`{"novel_name":"寿元账"}`)
	if err := os.WriteFile(filepath.Join(dir, "meta", "progress.json"), progress, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := upgradeProject(st); err != nil {
		t.Fatalf("upgradeProject: %v", err)
	}
	book, err := st.Book.Load()
	if err != nil {
		t.Fatalf("Load book: %v", err)
	}
	if book == nil || book.Title != "寿元账" || book.Synopsis != "凡人以寿元换取灵性，在求生与守住人性之间挣扎。" {
		t.Fatalf("unexpected migrated book: %+v", book)
	}
	if checkpoint := st.Checkpoints.LatestByStep(domain.GlobalScope(), "book"); checkpoint == nil {
		t.Fatal("book checkpoint was not recorded")
	}
	version, err := st.LoadProjectFormatVersion()
	if err != nil || version != storepkg.CurrentProjectFormatVersion {
		t.Fatalf("format version = %d, err = %v", version, err)
	}
}

func TestUpgradeProjectRepairsMissingV2ChapterRecord(t *testing.T) {
	dir := t.TempDir()
	st := storepkg.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveProjectFormatVersion(storepkg.ChapterRecordProjectFormatVersion); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	facts := domain.ChapterFacts{
		Title: "第一章", Summary: "旧书开篇", KeyEvents: []string{"启程"},
		HookType: "mystery", DominantStrand: "quest",
	}
	if err := st.Drafts.SaveDraft(1, "历史接纳正文"); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveFinalChapter(1, "历史接纳正文"); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveSummary(domain.ChapterSummary{
		Chapter: 1, Title: facts.Title, Summary: facts.Summary, KeyEvents: facts.KeyEvents,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.StartChapter(1); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.MarkChapterComplete(1, 6, facts.HookType, facts.DominantStrand); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Checkpoints.AppendArtifact(domain.ChapterScope(1), "commit", "chapters/01.md"); err != nil {
		t.Fatal(err)
	}
	if err := upgradeProject(st); err != nil {
		t.Fatal(err)
	}
	version, err := st.LoadProjectFormatVersion()
	if err != nil || version != storepkg.CurrentProjectFormatVersion {
		t.Fatalf("format version = %d, err = %v", version, err)
	}
	record, err := st.ChapterRecords.Load(1)
	if err != nil || record == nil || record.Content != "历史接纳正文" {
		t.Fatalf("chapter record = %+v, err = %v", record, err)
	}
}

func TestInterventionStopsWhenPersistenceFails(t *testing.T) {
	dir := t.TempDir()
	st := storepkg.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := st.RunMeta.Init("default", "test", "model"); err != nil {
		t.Fatalf("RunMeta.Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta", "run.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &Host{store: st, events: make(chan Event, 4)}
	err := h.doIntervention("修改主角性格", false)
	if err == nil || !strings.Contains(err.Error(), "持久化失败") {
		t.Fatalf("expected persistence error, got %v", err)
	}
	// 公共 Steer 必须等待异步任务并把同一业务错误返回给 TUI；不能只表示 goroutine
	// 启动成功，否则界面永远收不到真实失败。
	err = h.Steer("修改主角性格")
	if err == nil || !strings.Contains(err.Error(), "持久化失败") {
		t.Fatalf("Steer should return persistence error, got %v", err)
	}
}

func TestInterventionArbiterErrorKeepsPendingSteer(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init("default", "test", "model"); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	model := failingArbiterChatModel{}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}
	h := &Host{
		store: st, engine: &engine{store: st}, events: make(chan Event, 16),
		models: models, usage: NewUsageTracker(models, st), bundle: assets.Bundle{}, runCtx: context.Background(),
	}
	err := h.doIntervention("保留并稍后重试", false)
	if err == nil {
		t.Fatal("非法 Arbiter 输出应返回错误")
	}
	meta, loadErr := st.RunMeta.Load()
	if loadErr != nil || meta == nil {
		t.Fatalf("读取 RunMeta: %+v %v", meta, loadErr)
	}
	if meta.PendingSteer != "保留并稍后重试" {
		t.Fatalf("Arbiter 失败不得清除待恢复指令，got %q", meta.PendingSteer)
	}
	var arbiterFailure bool
	for len(h.events) > 0 {
		event := <-h.events
		if event.Agent == "arbiter" && event.Category == "ERROR" {
			arbiterFailure = true
		}
	}
	if !arbiterFailure {
		t.Fatal("Arbiter 错误应继续发出真实失败事件")
	}
}

func TestResumeRetriesPendingSteerWithoutDroppingOnArbiterError(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init("default", "test", "model"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPendingSteer("崩溃前尚未裁定的指令"); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	model := failingArbiterChatModel{}
	models := &bootstrap.ModelSet{Default: bootstrap.NewSwappableModel("test", "test", model, nil)}
	h := &Host{
		store: st, engine: &engine{store: st}, events: make(chan Event, 16),
		models: models, usage: NewUsageTracker(models, st), bundle: assets.Bundle{}, runCtx: context.Background(),
	}
	if label, err := h.Resume(); err == nil || label == "" {
		t.Fatalf("恢复应重放 PendingSteer 并返回 Arbiter 错误，label=%q err=%v", label, err)
	}
	meta, err := st.RunMeta.Load()
	if err != nil || meta == nil || meta.PendingSteer != "崩溃前尚未裁定的指令" {
		t.Fatalf("恢复裁定失败后必须保留 PendingSteer，meta=%+v err=%v", meta, err)
	}
}

type failingArbiterChatModel struct{}

func (failingArbiterChatModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, errors.New("Arbiter provider unavailable")
}
func (failingArbiterChatModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return nil, errors.New("unused")
}
func (failingArbiterChatModel) SupportsTools() bool { return true }

func TestCloseWaitsForRegisteredAsyncWork(t *testing.T) {
	h := &Host{
		observer: &observer{},
		engine:   &engine{},
		events:   make(chan Event, 1),
		streamCh: make(chan string, 1),
		done:     make(chan struct{}, 1),
	}
	started := make(chan struct{})
	release := make(chan struct{})
	if !h.launchAsync(func() {
		close(started)
		<-release
	}) {
		t.Fatal("launchAsync unexpectedly refused")
	}
	<-started
	closed := make(chan struct{})
	go func() {
		h.Close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned before async work finished")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after async work finished")
	}
}
