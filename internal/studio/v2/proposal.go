// Package v2 实现 Proposal、Evidence、Diff 和 Version 的治理层元数据。
//
// 本包不修改 Core 的 ChapterRecord 或 accepted revision。正文 Apply/Restore
// 通过调用方提供的 SaveChapter 回调写入 Core 工作正文，之后仍需显式 Sync。
package v2

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

const (
	SchemaVersion           = 1
	ResourceTypeChapter     = "ChapterText"
	ResourceTypeChapterText = ResourceTypeChapter
	ChangeTypeReplace       = "replace"
	ChangeTypeInsert        = "insert"
	ChangeTypeDelete        = "delete"
)

type ProposalStatus string

const (
	ProposalStatusDraft          ProposalStatus = "Draft"
	ProposalStatusReady          ProposalStatus = "Ready"
	ProposalStatusAccepted       ProposalStatus = "Accepted"
	ProposalStatusRejected       ProposalStatus = "Rejected"
	ProposalStatusStale          ProposalStatus = "Stale"
	ProposalStatusAppliedWorking ProposalStatus = "AppliedWorkingCopy"
	ProposalStatusSyncPending    ProposalStatus = "SyncPending"
	ProposalStatusSynced         ProposalStatus = "Synced"
	ProposalStatusFailed         ProposalStatus = "Failed"
)

type ProposalSource string

const (
	ProposalSourceReviewIssue ProposalSource = "ReviewIssue"
	ProposalSourceManual      ProposalSource = "ManualRequest"
	ProposalSourceVersion     ProposalSource = "VersionRestore"
)

var (
	ErrProposalNotFound = errors.New("proposal 不存在")
	ErrProposalStale    = errors.New("proposal 基于旧正文，已过期")
	ErrDuplicateSource  = errors.New("proposal 来源已存在")
	ErrInvalidStatus    = errors.New("proposal 状态不允许当前操作")
	ErrVersionNotFound  = errors.New("版本快照不存在")
	ErrVersionStale     = errors.New("版本基于旧工作正文，已过期")
)

// Proposal 是 V2 治理层对象，不代表 Core 已接纳正文。
type Proposal struct {
	ID           string           `json:"id"`
	ProjectID    string           `json:"projectId"`
	Title        string           `json:"title"`
	Summary      string           `json:"summary,omitempty"`
	Rationale    string           `json:"rationale,omitempty"`
	Source       ProposalSource   `json:"source"`
	SourceKey    string           `json:"sourceKey,omitempty"`
	Status       ProposalStatus   `json:"status"`
	Changes      []ProposalChange `json:"changes"`
	Evidence     []EvidenceRef    `json:"evidence"`
	BaseRevision string           `json:"baseRevision,omitempty"`
	OperationID  string           `json:"operationId,omitempty"`
	Error        string           `json:"error,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

type ProposalChange struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Chapter      int    `json:"chapter"`
	BaseHash     string `json:"baseHash"`
	Before       string `json:"before"`
	After        string `json:"after"`
	AfterHash    string `json:"afterHash"`
	ChangeType   string `json:"changeType"`
}

type EvidenceRef struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Chapter      int    `json:"chapter"`
	Revision     string `json:"revision"`
	ContentHash  string `json:"contentHash"`
	StartOffset  int    `json:"startOffset"`
	EndOffset    int    `json:"endOffset"`
	QuotePreview string `json:"quotePreview"`
}

type CreateProposalRequest struct {
	ProjectID string                `json:"projectId,omitempty"`
	Title     string                `json:"title"`
	Summary   string                `json:"summary,omitempty"`
	Rationale string                `json:"rationale,omitempty"`
	Source    ProposalSource        `json:"source,omitempty"`
	SourceKey string                `json:"sourceKey,omitempty"`
	Changes   []ProposalChangeInput `json:"changes"`
	Evidence  []EvidenceInput       `json:"evidence,omitempty"`
}

type ProposalChangeInput struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Chapter      int    `json:"chapter"`
	BaseHash     string `json:"baseHash,omitempty"`
	Before       string `json:"before,omitempty"`
	After        string `json:"after"`
	ChangeType   string `json:"changeType,omitempty"`
}

type EvidenceInput struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Chapter      int    `json:"chapter"`
	Revision     string `json:"revision,omitempty"`
	ContentHash  string `json:"contentHash,omitempty"`
	StartOffset  int    `json:"startOffset"`
	EndOffset    int    `json:"endOffset"`
	QuotePreview string `json:"quotePreview"`
}

type ApplyResult struct {
	ProposalID string         `json:"proposalId"`
	Status     ProposalStatus `json:"status"`
	Chapters   []int          `json:"chapters"`
	Message    string         `json:"message,omitempty"`
}

type SyncObservation struct {
	Chapter      int
	AcceptedHash string
	Synced       bool
	HasUnsynced  bool
}

// ReviewIssueSourceKey 为没有稳定 ID 的 Core ReviewIssue 生成确定性来源键。
func ReviewIssueSourceKey(review domain.ReviewEntry, issueIndex int) string {
	if issueIndex < 0 || issueIndex >= len(review.Issues) {
		return fmt.Sprintf("review:%s:%d:%d:invalid", review.Scope, review.Chapter, issueIndex)
	}
	payload := struct {
		Scope   string                  `json:"scope"`
		Chapter int                     `json:"chapter"`
		Index   int                     `json:"index"`
		Issue   domain.ConsistencyIssue `json:"issue"`
	}{review.Scope, review.Chapter, issueIndex, review.Issues[issueIndex]}
	data, _ := json.Marshal(payload)
	digest := sha256.Sum256(data)
	return fmt.Sprintf("review:%s:%d:%d:%x", review.Scope, review.Chapter, issueIndex, digest[:8])
}

type SaveChapterFunc func(chapter int, content string) error

type Manager struct {
	outputDir   string
	projectID   string
	saveChapter SaveChapterFunc
	mu          sync.Mutex
}

type operationJournal struct {
	ID         string        `json:"id"`
	Kind       string        `json:"kind,omitempty"`
	ProposalID string        `json:"proposalId"`
	VersionID  string        `json:"versionId,omitempty"`
	Stage      string        `json:"stage"`
	Planned    []journalItem `json:"planned"`
	Applied    []int         `json:"applied"`
	Error      string        `json:"error,omitempty"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type journalItem struct {
	Chapter   int    `json:"chapter"`
	BaseHash  string `json:"baseHash"`
	AfterHash string `json:"afterHash"`
}

// NewManager 创建 V2 治理层管理器。创建本身是只读的，不会创建 metadata 目录。
func NewManager(outputDir string) *Manager {
	return &Manager{outputDir: filepath.Clean(outputDir), projectID: filepath.Clean(outputDir)}
}

func sameProjectID(left, right string) bool {
	a, errA := filepath.Abs(left)
	b, errB := filepath.Abs(right)
	if errA == nil {
		left = filepath.Clean(a)
	}
	if errB == nil {
		right = filepath.Clean(b)
	}
	return strings.EqualFold(left, right)
}

func (m *Manager) SetProjectID(projectID string) { m.projectID = strings.TrimSpace(projectID) }

func (m *Manager) SetSaveChapter(save SaveChapterFunc) { m.saveChapter = save }

func (m *Manager) ValidateProjectID(projectID string) error {
	if strings.TrimSpace(projectID) == "" || sameProjectID(projectID, m.projectID) {
		return nil
	}
	return fmt.Errorf("proposal projectId 与当前项目不一致")
}

func (m *Manager) Create(req CreateProposalRequest) (Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ValidateProjectID(req.ProjectID); err != nil {
		return Proposal{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return Proposal{}, errors.New("proposal 标题不能为空")
	}
	if len(req.Changes) == 0 {
		return Proposal{}, errors.New("proposal 至少需要一个 change")
	}
	if req.Source == "" {
		req.Source = ProposalSourceManual
	}
	if err := m.ensureMetadata(); err != nil {
		return Proposal{}, err
	}
	if req.SourceKey != "" {
		existing, err := m.listUnlocked()
		if err != nil {
			return Proposal{}, err
		}
		for _, item := range existing {
			if item.ProjectID == m.projectID && item.SourceKey == req.SourceKey {
				return Proposal{}, fmt.Errorf("%w: %s", ErrDuplicateSource, req.SourceKey)
			}
		}
	}

	proposal := Proposal{
		ID:        newID(),
		ProjectID: m.projectID,
		Title:     strings.TrimSpace(req.Title),
		Summary:   strings.TrimSpace(req.Summary),
		Rationale: strings.TrimSpace(req.Rationale),
		Source:    req.Source,
		SourceKey: strings.TrimSpace(req.SourceKey),
		Status:    ProposalStatusReady,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	seenChanges := make(map[string]struct{}, len(req.Changes))
	for _, input := range req.Changes {
		resourceType := input.ResourceType
		if resourceType == "" {
			resourceType = ResourceTypeChapter
		}
		resourceID := input.ResourceID
		if resourceID == "" {
			resourceID = fmt.Sprintf("chapter:%d", input.Chapter)
		}
		changeKey := fmt.Sprintf("%s:%s:%d", resourceType, resourceID, input.Chapter)
		if _, exists := seenChanges[changeKey]; exists {
			return Proposal{}, fmt.Errorf("同一资源不能包含多个 change：%s", changeKey)
		}
		seenChanges[changeKey] = struct{}{}
		change, record, err := m.prepareChange(input)
		if err != nil {
			return Proposal{}, err
		}
		if proposal.BaseRevision == "" && record != nil {
			proposal.BaseRevision = fmt.Sprintf("%d", record.Revision)
		}
		proposal.Changes = append(proposal.Changes, change)
	}
	for _, change := range proposal.Changes {
		if _, _, err := m.createVersionUnlocked(CreateVersionRequest{
			ProjectID: m.projectID, ResourceType: change.ResourceType, ResourceID: change.ResourceID,
			Chapter: change.Chapter, Content: change.Before, Source: "proposal-base", ParentID: proposal.ID,
		}); err != nil {
			return Proposal{}, err
		}
	}
	for _, input := range req.Evidence {
		evidence, err := m.prepareEvidence(input, proposal.Changes)
		if err != nil {
			return Proposal{}, err
		}
		proposal.Evidence = append(proposal.Evidence, evidence)
	}
	if len(proposal.Evidence) == 0 {
		for _, change := range proposal.Changes {
			record, err := store.NewStore(m.outputDir).ChapterRecords.Load(change.Chapter)
			if err != nil {
				return Proposal{}, err
			}
			revision := ""
			if record != nil {
				revision = fmt.Sprintf("%d", record.Revision)
			}
			text := change.Before
			runes := []rune(text)
			endOffset := len(runes)
			if endOffset > 160 {
				endOffset = 160
			}
			preview := string(runes[:endOffset])
			proposal.Evidence = append(proposal.Evidence, EvidenceRef{
				ResourceType: change.ResourceType,
				ResourceID:   change.ResourceID,
				Chapter:      change.Chapter,
				Revision:     revision,
				ContentHash:  change.BaseHash,
				StartOffset:  0,
				EndOffset:    endOffset,
				QuotePreview: preview,
			})
		}
	}
	if err := m.writeProposal(proposal); err != nil {
		return Proposal{}, err
	}
	return proposal, nil
}

func (m *Manager) prepareChange(input ProposalChangeInput) (ProposalChange, *domain.ChapterRecord, error) {
	if input.Chapter <= 0 {
		return ProposalChange{}, nil, errors.New("change chapter 必须大于 0")
	}
	if input.ResourceType == "" {
		input.ResourceType = ResourceTypeChapter
	}
	if input.ResourceType != ResourceTypeChapter {
		return ProposalChange{}, nil, fmt.Errorf("M1 不支持资源类型 %q", input.ResourceType)
	}
	if input.ResourceID == "" {
		input.ResourceID = fmt.Sprintf("chapter:%d", input.Chapter)
	}
	if input.ChangeType == "" {
		input.ChangeType = ChangeTypeReplace
	}
	current, err := m.currentContent(input.Chapter)
	if err != nil {
		return ProposalChange{}, nil, err
	}
	if input.Before == "" {
		input.Before = current
	}
	if input.BaseHash == "" {
		input.BaseHash = domain.ChapterContentSHA256(input.Before)
	}
	if domain.ChapterContentSHA256(input.Before) != input.BaseHash {
		return ProposalChange{}, nil, fmt.Errorf("第 %d 章 Before 与 BaseHash 不匹配", input.Chapter)
	}
	if strings.TrimSpace(input.After) == "" {
		return ProposalChange{}, nil, fmt.Errorf("第 %d 章候选正文不能为空", input.Chapter)
	}
	record, err := store.NewStore(m.outputDir).ChapterRecords.Load(input.Chapter)
	if err != nil {
		return ProposalChange{}, nil, err
	}
	return ProposalChange{
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		Chapter:      input.Chapter,
		BaseHash:     input.BaseHash,
		Before:       input.Before,
		After:        input.After,
		AfterHash:    domain.ChapterContentSHA256(input.After),
		ChangeType:   input.ChangeType,
	}, record, nil
}

func (m *Manager) prepareEvidence(input EvidenceInput, changes []ProposalChange) (EvidenceRef, error) {
	if input.Chapter <= 0 {
		return EvidenceRef{}, errors.New("evidence chapter 必须大于 0")
	}
	if input.ResourceType == "" {
		input.ResourceType = ResourceTypeChapter
	}
	if input.ResourceID == "" {
		input.ResourceID = fmt.Sprintf("chapter:%d", input.Chapter)
	}
	var baseHash string
	var sourceLength int
	var sourceText string
	for _, change := range changes {
		if change.Chapter == input.Chapter && change.ResourceID == input.ResourceID && change.ResourceType == input.ResourceType {
			baseHash = change.BaseHash
			sourceText = change.Before
			sourceLength = len([]rune(sourceText))
			break
		}
	}
	if baseHash == "" {
		return EvidenceRef{}, fmt.Errorf("evidence 未找到对应的 chapter change: %s", input.ResourceID)
	}
	if input.ContentHash == "" {
		input.ContentHash = baseHash
	}
	if input.ContentHash != baseHash {
		return EvidenceRef{}, fmt.Errorf("evidence %s 未绑定对应 BaseHash", input.ResourceID)
	}
	if input.StartOffset < 0 || input.EndOffset < input.StartOffset || input.EndOffset > sourceLength {
		return EvidenceRef{}, errors.New("evidence 正文区间无效")
	}
	expectedQuote := string([]rune(sourceText)[input.StartOffset:input.EndOffset])
	if input.QuotePreview == "" {
		input.QuotePreview = expectedQuote
	} else if input.QuotePreview != expectedQuote {
		return EvidenceRef{}, errors.New("evidence QuotePreview 与正文区间不一致")
	}
	record, err := store.NewStore(m.outputDir).ChapterRecords.Load(input.Chapter)
	if err != nil {
		return EvidenceRef{}, err
	}
	if input.Revision == "" && record != nil {
		input.Revision = fmt.Sprintf("%d", record.Revision)
	}
	return EvidenceRef{
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		Chapter:      input.Chapter,
		Revision:     input.Revision,
		ContentHash:  input.ContentHash,
		StartOffset:  input.StartOffset,
		EndOffset:    input.EndOffset,
		QuotePreview: input.QuotePreview,
	}, nil
}

func (m *Manager) Accept(id string) (Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proposal, err := m.readProposal(id)
	if err != nil {
		return Proposal{}, err
	}
	if proposal.Status != ProposalStatusReady && proposal.Status != ProposalStatusDraft {
		return proposal, fmt.Errorf("proposal %s 当前状态 %s 不能 Accept: %w", id, proposal.Status, ErrInvalidStatus)
	}
	proposal.Status = ProposalStatusAccepted
	proposal.UpdatedAt = time.Now().UTC()
	return proposal, m.writeProposal(proposal)
}

func (m *Manager) Reject(id string) (Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proposal, err := m.readProposal(id)
	if err != nil {
		return Proposal{}, err
	}
	if proposal.Status == ProposalStatusAppliedWorking || proposal.Status == ProposalStatusSyncPending || proposal.Status == ProposalStatusSynced {
		return proposal, fmt.Errorf("proposal %s 已应用，不能 Reject: %w", id, ErrInvalidStatus)
	}
	proposal.Status = ProposalStatusRejected
	proposal.UpdatedAt = time.Now().UTC()
	return proposal, m.writeProposal(proposal)
}

func (m *Manager) Apply(id string) (ApplyResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proposal, err := m.readProposal(id)
	if err != nil {
		return ApplyResult{}, err
	}
	if proposal.Status != ProposalStatusAccepted {
		return ApplyResult{ProposalID: id, Status: proposal.Status}, fmt.Errorf("proposal %s 当前状态 %s 不能 Apply: %w", id, proposal.Status, ErrInvalidStatus)
	}
	if m.saveChapter == nil {
		return ApplyResult{}, errors.New("proposal Apply 未配置 Core SaveChapter 适配器")
	}
	journal := operationJournal{ID: newID(), Kind: "proposal", ProposalID: id, Stage: "planned", UpdatedAt: time.Now().UTC()}
	proposal.OperationID = journal.ID
	proposal.Error = ""
	for _, change := range proposal.Changes {
		current, err := m.currentContent(change.Chapter)
		if err != nil {
			return ApplyResult{}, err
		}
		if domain.ChapterContentSHA256(current) != change.BaseHash {
			proposal.Status = ProposalStatusStale
			proposal.Error = fmt.Sprintf("第 %d 章当前正文已变化，拒绝覆盖", change.Chapter)
			proposal.UpdatedAt = time.Now().UTC()
			_ = m.writeProposal(proposal)
			return ApplyResult{ProposalID: id, Status: proposal.Status, Message: proposal.Error}, fmt.Errorf("%w: 第 %d 章", ErrProposalStale, change.Chapter)
		}
		journal.Planned = append(journal.Planned, journalItem{Chapter: change.Chapter, BaseHash: change.BaseHash, AfterHash: change.AfterHash})
	}
	if err := m.writeJournal(journal); err != nil {
		return ApplyResult{}, err
	}
	// 先保存 Apply 前版本；任何快照失败都发生在正文写入之前。
	for _, change := range proposal.Changes {
		if _, _, err := m.createVersionUnlocked(CreateVersionRequest{
			ProjectID: m.projectID, ResourceType: change.ResourceType, ResourceID: change.ResourceID,
			Chapter: change.Chapter, Content: change.Before, Source: "proposal-before", ParentID: proposal.ID,
		}); err != nil {
			return ApplyResult{}, err
		}
		if _, _, err := m.createVersionUnlocked(CreateVersionRequest{
			ProjectID: m.projectID, ResourceType: change.ResourceType, ResourceID: change.ResourceID,
			Chapter: change.Chapter, Content: change.After, Source: "proposal-candidate", ParentID: proposal.ID,
		}); err != nil {
			return ApplyResult{}, err
		}
	}
	journal.Stage = "applying"
	journal.UpdatedAt = time.Now().UTC()
	if err := m.writeJournal(journal); err != nil {
		return ApplyResult{}, err
	}
	for _, change := range proposal.Changes {
		if err := m.saveChapter(change.Chapter, change.After); err != nil {
			journal.Stage = "failed"
			journal.Error = err.Error()
			journal.UpdatedAt = time.Now().UTC()
			_ = m.writeJournal(journal)
			proposal.Status = ProposalStatusFailed
			proposal.Error = fmt.Sprintf("第 %d 章写入失败：%v", change.Chapter, err)
			proposal.UpdatedAt = time.Now().UTC()
			_ = m.writeProposal(proposal)
			return ApplyResult{ProposalID: id, Status: proposal.Status, Chapters: append([]int(nil), journal.Applied...), Message: proposal.Error}, err
		}
		journal.Applied = append(journal.Applied, change.Chapter)
		journal.UpdatedAt = time.Now().UTC()
		if err := m.writeJournal(journal); err != nil {
			return ApplyResult{}, err
		}
	}
	journal.Stage = "applied"
	journal.UpdatedAt = time.Now().UTC()
	if err := m.writeJournal(journal); err != nil {
		return ApplyResult{}, err
	}
	proposal.Status = ProposalStatusAppliedWorking
	proposal.UpdatedAt = time.Now().UTC()
	if err := m.writeProposal(proposal); err != nil {
		return ApplyResult{}, err
	}
	chapters := make([]int, 0, len(proposal.Changes))
	for _, change := range proposal.Changes {
		chapters = append(chapters, change.Chapter)
	}
	return ApplyResult{ProposalID: id, Status: proposal.Status, Chapters: chapters, Message: "已应用到工作正文，等待 Core Sync"}, nil
}

func (m *Manager) Get(id string) (Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readProposal(id)
}

func (m *Manager) List() ([]Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(m.proposalsDir())
	if os.IsNotExist(err) {
		return []Proposal{}, nil
	}
	if err != nil {
		return nil, err
	}
	proposals := make([]Proposal, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))]
		if !safeMetadataID(id) {
			continue
		}
		proposal, err := m.readProposal(id)
		if err != nil {
			return nil, err
		}
		proposals = append(proposals, proposal)
	}
	sort.Slice(proposals, func(i, j int) bool { return proposals[i].UpdatedAt.After(proposals[j].UpdatedAt) })
	return proposals, nil
}

// ReconcileSync 将 Core 已确认的 accepted hash 映射回 Proposal 状态。
func (m *Manager) MarkSyncPending(id string) (Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proposal, err := m.readProposal(id)
	if err != nil {
		return Proposal{}, err
	}
	if proposal.Status != ProposalStatusAppliedWorking && proposal.Status != ProposalStatusSyncPending {
		return proposal, fmt.Errorf("proposal %s 当前状态 %s 不能进入待同步: %w", id, proposal.Status, ErrInvalidStatus)
	}
	proposal.Status = ProposalStatusSyncPending
	proposal.UpdatedAt = time.Now().UTC()
	return proposal, m.writeProposal(proposal)
}

// ReconcileSync 只有显式 Core Sync 成功后才能检查 ChapterRecord 并提升 Proposal。
func (m *Manager) ReconcileSync(succeeded bool) ([]Proposal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proposals, err := m.listUnlocked()
	if err != nil {
		return nil, err
	}
	for index := range proposals {
		proposal := &proposals[index]
		if proposal.Status != ProposalStatusAppliedWorking && proposal.Status != ProposalStatusSyncPending {
			continue
		}
		if !succeeded {
			proposal.Status = ProposalStatusSyncPending
			proposal.UpdatedAt = time.Now().UTC()
			if err := m.writeProposal(*proposal); err != nil {
				return nil, err
			}
			continue
		}
		allAccepted := true
		core := store.NewStore(m.outputDir)
		for _, change := range proposal.Changes {
			record, err := core.ChapterRecords.Load(change.Chapter)
			if err != nil {
				return nil, err
			}
			if record == nil || record.ContentSHA256 != change.AfterHash {
				allAccepted = false
				break
			}
		}
		if allAccepted {
			proposal.Status = ProposalStatusSynced
		} else {
			proposal.Status = ProposalStatusSyncPending
		}
		proposal.UpdatedAt = time.Now().UTC()
		if err := m.writeProposal(*proposal); err != nil {
			return nil, err
		}
	}
	return proposals, nil
}

// RecoverOperations 根据 Core 工作正文与 Apply journal 恢复崩溃边界。
// 此操作只修改 V2 metadata；不会写入章节正文或 accepted revision。
func (m *Manager) RecoverOperations() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(m.operationsDir())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(m.operationsDir(), entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var journal operationJournal
		if err := json.Unmarshal(data, &journal); err != nil {
			return fmt.Errorf("读取 Apply journal %s: %w", entry.Name(), err)
		}
		if journal.Kind == "version" || journal.VersionID != "" {
			if journal.Stage != "planned" && journal.Stage != "applying" && journal.Stage != "applied" {
				continue
			}
			if err := m.recoverVersionOperation(journal, path); err != nil {
				return err
			}
			continue
		}
		if journal.Stage != "planned" && journal.Stage != "applying" && journal.Stage != "applied" {
			continue
		}
		proposal, err := m.readProposal(journal.ProposalID)
		if err != nil {
			return err
		}
		allAfter, allBase := true, true
		applied := make([]int, 0, len(journal.Planned))
		for _, item := range journal.Planned {
			current, err := m.currentContent(item.Chapter)
			if err != nil {
				return err
			}
			hash := domain.ChapterContentSHA256(current)
			if hash == item.AfterHash {
				applied = append(applied, item.Chapter)
			} else {
				allAfter = false
			}
			if hash != item.BaseHash {
				allBase = false
			}
		}
		proposal.UpdatedAt = time.Now().UTC()
		journal.UpdatedAt = proposal.UpdatedAt
		switch {
		case allAfter && len(journal.Planned) > 0:
			proposal.Status = ProposalStatusAppliedWorking
			proposal.Error = "已从中断的 Apply 恢复；正文仍等待 Core Sync。"
			journal.Stage = "recovered_applied"
			journal.Applied = applied
		case allBase:
			proposal.Status = ProposalStatusAccepted
			proposal.Error = "Apply 在写入正文前中断，可以重新尝试。"
			journal.Stage = "recovered_no_write"
		case len(applied) > 0:
			proposal.Status = ProposalStatusFailed
			proposal.Error = "Apply 中断且仅部分章节写入；请逐章检查工作正文后人工恢复。"
			journal.Stage = "recovered_partial"
			journal.Applied = applied
		default:
			proposal.Status = ProposalStatusStale
			proposal.Error = "Apply 中断后正文与基线及候选均不匹配，拒绝覆盖。"
			journal.Stage = "recovered_stale"
		}
		if err := m.writeProposal(proposal); err != nil {
			return err
		}
		if err := m.writeJournal(journal); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) recoverVersionOperation(journal operationJournal, path string) error {
	if journal.VersionID == "" || len(journal.Planned) != 1 {
		return fmt.Errorf("恢复版本操作日志无效：%s", filepath.Base(path))
	}
	snapshot, err := m.readVersion(journal.VersionID)
	if err != nil {
		return err
	}
	current, err := m.currentContent(snapshot.Chapter)
	if err != nil {
		return err
	}
	hash := domain.ChapterContentSHA256(current)
	item := journal.Planned[0]
	switch hash {
	case item.AfterHash:
		if _, _, err := m.createVersionUnlocked(CreateVersionRequest{
			ProjectID: snapshot.ProjectID, ResourceType: snapshot.ResourceType, ResourceID: snapshot.ResourceID,
			Chapter: snapshot.Chapter, Content: snapshot.Content, Source: "restore-after-recovered", ParentID: snapshot.ID,
		}); err != nil {
			return err
		}
		journal.Stage = "committed"
		journal.Error = ""
	case item.BaseHash:
		journal.Stage = "failed"
		journal.Error = "恢复操作尚未写入正文"
	default:
		journal.Stage = "failed"
		journal.Error = "恢复期间正文发生变化，未自动覆盖"
	}
	journal.UpdatedAt = time.Now().UTC()
	return atomicWriteJSON(path, journal)
}

func (m *Manager) currentContent(chapter int) (string, error) {
	core := store.NewStore(m.outputDir)
	text, err := core.Drafts.LoadChapterText(chapter)
	if err != nil {
		return "", err
	}
	if text != "" {
		return text, nil
	}
	record, err := core.ChapterRecords.Load(chapter)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("第 %d 章不存在终稿或接纳记录", chapter)
	}
	return record.Content, nil
}

func (m *Manager) ensureMetadata() error {
	if err := os.MkdirAll(m.proposalsDir(), 0755); err != nil {
		return err
	}
	for _, dir := range []string{m.versionsDir(), m.operationsDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	schemaPath := filepath.Join(m.baseDir(), "schema_version.json")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return atomicWriteJSON(schemaPath, struct {
			SchemaVersion int `json:"schema_version"`
		}{SchemaVersion: SchemaVersion})
	} else if err != nil {
		return err
	}
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}
	var schema struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("读取 V2 schema 版本失败: %w", err)
	}
	if schema.SchemaVersion != SchemaVersion {
		return fmt.Errorf("不支持的 V2 schema 版本 %d（当前支持 %d）", schema.SchemaVersion, SchemaVersion)
	}
	return nil
}

func (m *Manager) writeProposal(proposal Proposal) error {
	if err := m.ensureMetadata(); err != nil {
		return err
	}
	if !safeMetadataID(proposal.ID) {
		return fmt.Errorf("proposal ID 无效")
	}
	return atomicWriteJSON(filepath.Join(m.proposalsDir(), proposal.ID+".json"), proposal)
}

func (m *Manager) readProposal(id string) (Proposal, error) {
	if !safeMetadataID(id) {
		return Proposal{}, ErrProposalNotFound
	}
	data, err := os.ReadFile(filepath.Join(m.proposalsDir(), id+".json"))
	if os.IsNotExist(err) {
		return Proposal{}, fmt.Errorf("%w: %s", ErrProposalNotFound, id)
	}
	if err != nil {
		return Proposal{}, err
	}
	var proposal Proposal
	if err := json.Unmarshal(data, &proposal); err != nil {
		return Proposal{}, fmt.Errorf("读取 proposal %s: %w", id, err)
	}
	return proposal, nil
}

func (m *Manager) listUnlocked() ([]Proposal, error) {
	entries, err := os.ReadDir(m.proposalsDir())
	if os.IsNotExist(err) {
		return []Proposal{}, nil
	}
	if err != nil {
		return nil, err
	}
	proposals := make([]Proposal, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if !safeMetadataID(id) {
			continue
		}
		proposal, err := m.readProposal(id)
		if err != nil {
			return nil, err
		}
		proposals = append(proposals, proposal)
	}
	return proposals, nil
}

func (m *Manager) writeJournal(journal operationJournal) error {
	if err := m.ensureMetadata(); err != nil {
		return err
	}
	name := journal.ProposalID
	if journal.VersionID != "" {
		name = "restore-" + journal.VersionID
	}
	if !safeMetadataID(name) {
		return fmt.Errorf("操作日志 ID 无效")
	}
	return atomicWriteJSON(filepath.Join(m.operationsDir(), name+".json"), journal)
}

func (m *Manager) baseDir() string       { return filepath.Join(m.outputDir, "meta", "studio-v2") }
func (m *Manager) proposalsDir() string  { return filepath.Join(m.baseDir(), "proposals") }
func (m *Manager) versionsDir() string   { return filepath.Join(m.baseDir(), "versions") }
func (m *Manager) operationsDir() string { return filepath.Join(m.baseDir(), "operations") }

func atomicWriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".studio-v2-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

var safeMetadataIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func safeMetadataID(id string) bool {
	return safeMetadataIDPattern.MatchString(id)
}

func newID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}
