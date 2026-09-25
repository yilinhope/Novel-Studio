package v2

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func newProposalFixture(t *testing.T) (*Manager, *store.Store, string) {
	t.Helper()
	outputDir := filepath.Join(t.TempDir(), "项目", "output", "novel")
	core := store.NewStore(outputDir)
	if err := core.Init(); err != nil {
		t.Fatal(err)
	}
	if err := core.Progress.Save(&domain.Progress{
		Phase:             domain.PhaseWriting,
		CompletedChapters: []int{1, 2},
		CurrentChapter:    3,
		TotalChapters:     2,
	}); err != nil {
		t.Fatal(err)
	}
	for chapter, content := range map[int]string{
		1: "第一章\n\n旧正文。",
		2: "第二章\n\n第二段旧正文。",
	} {
		if err := core.Drafts.SaveFinalChapter(chapter, content); err != nil {
			t.Fatal(err)
		}
		if _, err := core.ChapterRecords.Accept(chapter, domain.ChapterOriginGenerated, content, domain.ChapterFacts{}, domain.StyleDelta{}); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewManager(outputDir)
	manager.SetSaveChapter(func(chapter int, content string) error {
		return core.Drafts.SaveFinalChapter(chapter, content)
	})
	return manager, core, outputDir
}

func TestCreateProposalCapturesBaseHashAndEvidence(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	content, err := core.Drafts.LoadChapterText(1)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := manager.Create(CreateProposalRequest{
		Title:     "修正第一章",
		Source:    ProposalSourceReviewIssue,
		SourceKey: "review:chapter:1:0:abc",
		Changes: []ProposalChangeInput{{
			ResourceType: ResourceTypeChapterText,
			ResourceID:   "chapter:1",
			Chapter:      1,
			Before:       content,
			After:        "第一章\n\n新正文。",
			ChangeType:   ChangeTypeReplace,
		}},
		Evidence: []EvidenceInput{{
			ResourceType: ResourceTypeChapterText,
			ResourceID:   "chapter:1",
			Chapter:      1,
			StartOffset:  0,
			EndOffset:    len([]rune("第一章")),
			QuotePreview: "第一章",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != ProposalStatusReady {
		t.Fatalf("新 Proposal 应为 Ready，得到 %s", proposal.Status)
	}
	if proposal.Changes[0].BaseHash != domain.ChapterContentSHA256(content) {
		t.Fatalf("BaseHash 未绑定当前正文：%s", proposal.Changes[0].BaseHash)
	}
	if proposal.Changes[0].AfterHash != domain.ChapterContentSHA256("第一章\n\n新正文。") {
		t.Fatalf("AfterHash 不正确：%s", proposal.Changes[0].AfterHash)
	}
	if len(proposal.Evidence) != 1 || proposal.Evidence[0].ContentHash != proposal.Changes[0].BaseHash || proposal.Evidence[0].QuotePreview != "第一章" {
		t.Fatalf("Evidence 未绑定 BaseHash：%+v", proposal.Evidence)
	}
}

func TestCreateProposalRejectsDuplicateResourceChanges(t *testing.T) {
	manager, _, _ := newProposalFixture(t)
	_, err := manager.Create(CreateProposalRequest{Title: "重复章节", Changes: []ProposalChangeInput{
		{Chapter: 1, After: "候选 A。"},
		{Chapter: 1, After: "候选 B。"},
	}})
	if err == nil || !strings.Contains(err.Error(), "多个 change") {
		t.Fatalf("重复资源 change 应拒绝，得到 %v", err)
	}
}

func TestCreateProposalRejectsDuplicateSourceKey(t *testing.T) {
	manager, _, _ := newProposalFixture(t)
	request := CreateProposalRequest{Title: "同一 ReviewIssue", Source: ProposalSourceReviewIssue, SourceKey: "review:key", Changes: []ProposalChangeInput{{Chapter: 1, After: "候选。"}}}
	if _, err := manager.Create(request); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Create(request); !errors.Is(err, ErrDuplicateSource) {
		t.Fatalf("重复 SourceKey 应拒绝，得到 %v", err)
	}
}

func TestCreateProposalRejectsMismatchedEvidenceQuote(t *testing.T) {
	manager, _, _ := newProposalFixture(t)
	_, err := manager.Create(CreateProposalRequest{Title: "证据区间", Changes: []ProposalChangeInput{{Chapter: 1, After: "候选。"}}, Evidence: []EvidenceInput{{
		Chapter: 1, StartOffset: 0, EndOffset: 3, QuotePreview: "错误引文",
	}}})
	if err == nil || !strings.Contains(err.Error(), "QuotePreview") {
		t.Fatalf("不匹配的 evidence 引文应拒绝，得到 %v", err)
	}
}

func TestApplyProposalRejectsStaleWithoutWriting(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	proposal, err := manager.Create(CreateProposalRequest{
		Title: "修正第一章",
		Changes: []ProposalChangeInput{{
			ResourceType: ResourceTypeChapterText,
			ResourceID:   "chapter:1",
			Chapter:      1,
			After:        "候选正文。",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Accept(proposal.ID); err != nil {
		t.Fatal(err)
	}
	if err := core.Drafts.SaveFinalChapter(1, "用户手工改过。\n"); err != nil {
		t.Fatal(err)
	}
	_, err = manager.Apply(proposal.ID)
	if err == nil || !errors.Is(err, ErrProposalStale) {
		t.Fatalf("手工编辑后应返回 stale，得到 %v", err)
	}
	got, err := core.Drafts.LoadChapterText(1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "用户手工改过。\n" {
		t.Fatalf("stale Apply 不得覆盖手工正文：%q", got)
	}
	stored, err := manager.Get(proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != ProposalStatusStale {
		t.Fatalf("stale Proposal 状态错误：%s", stored.Status)
	}
}

func TestApplyProposalPrechecksAllChangesBeforeWriting(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	proposal, err := manager.Create(CreateProposalRequest{
		Title: "多章修正",
		Changes: []ProposalChangeInput{
			{ResourceType: ResourceTypeChapterText, ResourceID: "chapter:1", Chapter: 1, After: "第一章新正文。"},
			{ResourceType: ResourceTypeChapterText, ResourceID: "chapter:2", Chapter: 2, After: "第二章新正文。"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Accept(proposal.ID); err != nil {
		t.Fatal(err)
	}
	if err := core.Drafts.SaveFinalChapter(2, strings.ReplaceAll("第二章\n\n第二段旧正文。", "旧正文", "用户改动")); err != nil {
		t.Fatal(err)
	}
	_, err = manager.Apply(proposal.ID)
	if err == nil || !errors.Is(err, ErrProposalStale) {
		t.Fatalf("任一 change stale 时应拒绝整批 Apply，得到 %v", err)
	}
	first, err := core.Drafts.LoadChapterText(1)
	if err != nil {
		t.Fatal(err)
	}
	if first != "第一章\n\n旧正文。" {
		t.Fatalf("多变更 precondition 失败时第一章被部分写入：%q", first)
	}
}

func TestRecoverApplyCrashKeepsProposalAwaitingSync(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	proposal, err := manager.Create(CreateProposalRequest{
		Title: "恢复测试",
		Changes: []ProposalChangeInput{{
			ResourceType: ResourceTypeChapterText,
			ResourceID:   "chapter:1",
			Chapter:      1,
			After:        "崩溃前已写入的候选正文。",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err = manager.Accept(proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	change := proposal.Changes[0]
	if err := core.Drafts.SaveFinalChapter(change.Chapter, change.After); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	manager.writeJournal(operationJournal{
		ID: "operation-1", ProposalID: proposal.ID, Stage: "applying",
		Planned: []journalItem{{Chapter: change.Chapter, BaseHash: change.BaseHash, AfterHash: change.AfterHash}},
	})
	proposal.OperationID = "operation-1"
	manager.writeProposal(proposal)
	manager.mu.Unlock()

	if err := manager.RecoverOperations(); err != nil {
		t.Fatal(err)
	}
	recovered, err := manager.Get(proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != ProposalStatusAppliedWorking {
		t.Fatalf("Apply 崩溃恢复不能伪造 Sync，状态=%s", recovered.Status)
	}
	record, err := core.ChapterRecords.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if record.ContentSHA256 == change.AfterHash {
		t.Fatal("Apply 崩溃恢复不得直接改 Core accepted revision")
	}
}

func TestRecoverApplyCrashAfterJournalApplied(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	proposal, err := manager.Create(CreateProposalRequest{Title: "Stage applied 恢复", Changes: []ProposalChangeInput{{Chapter: 1, After: "候选正文。"}}})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err = manager.Accept(proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	change := proposal.Changes[0]
	if err := core.Drafts.SaveFinalChapter(change.Chapter, change.After); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	if err := manager.writeJournal(operationJournal{ID: "operation-2", Kind: "proposal", ProposalID: proposal.ID, Stage: "applied", Planned: []journalItem{{Chapter: 1, BaseHash: change.BaseHash, AfterHash: change.AfterHash}}}); err != nil {
		t.Fatal(err)
	}
	manager.mu.Unlock()
	if err := manager.RecoverOperations(); err != nil {
		t.Fatal(err)
	}
	recovered, err := manager.Get(proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != ProposalStatusAppliedWorking {
		t.Fatalf("applied journal 应恢复工作副本状态，得到 %s", recovered.Status)
	}
}

func TestReconcileSyncFailureAndAcceptedHash(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	proposal, err := manager.Create(CreateProposalRequest{
		Title: "同步复核",
		Changes: []ProposalChangeInput{{
			ResourceType: ResourceTypeChapterText,
			ResourceID:   "chapter:1",
			Chapter:      1,
			After:        "等待同步的候选正文。",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Accept(proposal.ID); err != nil {
		t.Fatal(err)
	}
	manager.SetSaveChapter(func(chapter int, content string) error { return core.Drafts.SaveFinalChapter(chapter, content) })
	if _, err := manager.Apply(proposal.ID); err != nil {
		t.Fatal(err)
	}
	proposal, err = manager.MarkSyncPending(proposal.ID)
	if err != nil || proposal.Status != ProposalStatusSyncPending {
		t.Fatalf("Apply 后应等待 Core Sync：proposal=%+v err=%v", proposal, err)
	}
	if _, err := manager.ReconcileSync(false); err != nil {
		t.Fatal(err)
	}
	proposal, err = manager.Get(proposal.ID)
	if err != nil || proposal.Status != ProposalStatusSyncPending {
		t.Fatalf("Core Sync 失败不能把 Proposal 标记完成：proposal=%+v err=%v", proposal, err)
	}
	change := proposal.Changes[0]
	if _, err := core.ChapterRecords.Accept(1, domain.ChapterOriginUser, change.After, domain.ChapterFacts{}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ReconcileSync(true); err != nil {
		t.Fatal(err)
	}
	proposal, err = manager.Get(proposal.ID)
	if err != nil || proposal.Status != ProposalStatusSynced {
		t.Fatalf("只有 accepted hash 命中候选后才应 Synced：proposal=%+v err=%v", proposal, err)
	}
}

func TestV2SchemaVersionRejectsCorruption(t *testing.T) {
	manager, _, outputDir := newProposalFixture(t)
	proposal, err := manager.Create(CreateProposalRequest{Title: "初始化 metadata", Changes: []ProposalChangeInput{{
		Chapter: 1, After: "候选正文。",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.ID == "" {
		t.Fatal("应创建 proposal")
	}
	path := filepath.Join(outputDir, "meta", "studio-v2", "schema_version.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Create(CreateProposalRequest{Title: "不得兼容未知 schema", Changes: []ProposalChangeInput{{
		Chapter: 1, After: "另一候选正文。",
	}}}); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("未知 schema 必须显式拒绝：%v", err)
	}
}

func TestRestoreVersionWritesWorkingCopyWithoutChangingAcceptedRecord(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	recordBefore, err := core.ChapterRecords.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := manager.CreateVersion(CreateVersionRequest{Chapter: 1, Content: "第一章\n\n历史正文。", Source: "manual-save"})
	if err != nil {
		t.Fatal(err)
	}
	manager.SetSaveChapter(func(chapter int, content string) error { return core.Drafts.SaveFinalChapter(chapter, content) })
	if _, err := manager.RestoreVersion(snapshot.ID); err != nil {
		t.Fatal(err)
	}
	working, err := core.Drafts.LoadChapterText(1)
	if err != nil || working != "第一章\n\n历史正文。" {
		t.Fatalf("Restore 应只恢复工作正文：%q err=%v", working, err)
	}
	recordAfter, err := core.ChapterRecords.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if recordAfter.ContentSHA256 != recordBefore.ContentSHA256 || recordAfter.Revision != recordBefore.Revision {
		t.Fatalf("Restore 不得改变 accepted ChapterRecord：before=%+v after=%+v", recordBefore, recordAfter)
	}
}

func TestRestoreVersionRejectsStaleWorkingCopy(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	snapshot, _, err := manager.CreateVersion(CreateVersionRequest{Chapter: 1, Content: "历史候选正文。", Source: "manual-save"})
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Drafts.SaveFinalChapter(1, "创建快照后的人工改动。\n"); err != nil {
		t.Fatal(err)
	}
	manager.SetSaveChapter(func(chapter int, content string) error { return core.Drafts.SaveFinalChapter(chapter, content) })
	_, err = manager.RestoreVersion(snapshot.ID)
	if !errors.Is(err, ErrVersionStale) {
		t.Fatalf("陈旧版本恢复应拒绝，得到 %v", err)
	}
	got, err := core.Drafts.LoadChapterText(1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "创建快照后的人工改动。\n" {
		t.Fatalf("陈旧版本恢复覆盖了人工正文：%q", got)
	}
}

func TestRecoverRestoreAfterJournalApplied(t *testing.T) {
	manager, core, _ := newProposalFixture(t)
	snapshot, _, err := manager.CreateVersion(CreateVersionRequest{Chapter: 1, Content: "历史恢复正文。", Source: "manual-save"})
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Drafts.SaveFinalChapter(1, snapshot.Content); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	if err := manager.writeJournal(operationJournal{
		ID: "restore-op", Kind: "version", VersionID: snapshot.ID, Stage: "applied",
		Planned: []journalItem{{Chapter: 1, BaseHash: snapshot.BaseHash, AfterHash: snapshot.ContentHash}},
	}); err != nil {
		t.Fatal(err)
	}
	manager.mu.Unlock()
	if err := manager.RecoverOperations(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(manager.operationsDir())
	if err != nil {
		t.Fatal(err)
	}
	foundCommitted := false
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "restore-") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(manager.operationsDir(), entry.Name()))
		if readErr != nil {
			t.Fatal(readErr)
		}
		var recovered operationJournal
		if err := json.Unmarshal(data, &recovered); err != nil {
			t.Fatal(err)
		}
		foundCommitted = recovered.Stage == "committed"
	}
	if !foundCommitted {
		t.Fatal("恢复版本的 applied journal 应标记为 committed")
	}
}

func TestProposalAndVersionIDsRejectPathTraversal(t *testing.T) {
	manager, _, _ := newProposalFixture(t)
	if _, err := manager.Get("..\\schema_version"); !errors.Is(err, ErrProposalNotFound) {
		t.Fatalf("非法 proposal ID 应拒绝，得到 %v", err)
	}
	if _, err := manager.GetVersion("..\\schema_version"); !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("非法 version ID 应拒绝，得到 %v", err)
	}
}
