package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
)

type VersionSnapshot struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	Chapter      int       `json:"chapter"`
	ContentHash  string    `json:"contentHash"`
	BaseHash     string    `json:"baseHash,omitempty"`
	Content      string    `json:"content"`
	Source       string    `json:"source"`
	ParentID     string    `json:"parentId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	// CurrentHash 是读取历史版本时观察到的当前工作正文 hash，不写回快照文件。
	CurrentHash string `json:"currentHash,omitempty"`
}

type CreateVersionRequest struct {
	ProjectID    string
	ResourceType string
	ResourceID   string
	Chapter      int
	Content      string
	Source       string
	ParentID     string
}

func (m *Manager) CreateVersion(req CreateVersionRequest) (VersionSnapshot, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createVersionUnlocked(req)
}

func (m *Manager) createVersionUnlocked(req CreateVersionRequest) (VersionSnapshot, bool, error) {
	if req.Chapter <= 0 {
		return VersionSnapshot{}, false, errors.New("version chapter 必须大于 0")
	}
	if strings.TrimSpace(req.Content) == "" {
		return VersionSnapshot{}, false, errors.New("version 正文不能为空")
	}
	if req.ResourceType == "" {
		req.ResourceType = ResourceTypeChapter
	}
	if req.ResourceID == "" {
		req.ResourceID = fmt.Sprintf("chapter:%d", req.Chapter)
	}
	if req.Source == "" {
		req.Source = "manual"
	}
	if req.ProjectID == "" {
		req.ProjectID = m.projectID
	}
	hash := domain.ChapterContentSHA256(req.Content)
	entries, err := os.ReadDir(m.versionsDir())
	if os.IsNotExist(err) {
		entries = nil
	} else if err != nil {
		return VersionSnapshot{}, false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var snapshot VersionSnapshot
		data, err := os.ReadFile(filepath.Join(m.versionsDir(), entry.Name()))
		if err != nil {
			return VersionSnapshot{}, false, err
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return VersionSnapshot{}, false, err
		}
		if snapshot.ResourceType == req.ResourceType && snapshot.ResourceID == req.ResourceID && snapshot.ContentHash == hash {
			return snapshot, false, nil
		}
	}
	if err := m.ensureMetadata(); err != nil {
		return VersionSnapshot{}, false, err
	}
	current, err := m.currentContent(req.Chapter)
	if err != nil {
		return VersionSnapshot{}, false, err
	}
	snapshot := VersionSnapshot{
		ID:           newID(),
		ProjectID:    req.ProjectID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Chapter:      req.Chapter,
		ContentHash:  hash,
		BaseHash:     domain.ChapterContentSHA256(current),
		Content:      req.Content,
		Source:       req.Source,
		ParentID:     req.ParentID,
		CreatedAt:    time.Now().UTC(),
	}
	if err := atomicWriteJSON(filepath.Join(m.versionsDir(), snapshot.ID+".json"), snapshot); err != nil {
		return VersionSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func (m *Manager) GetVersion(id string) (VersionSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot, err := m.readVersion(id)
	if err != nil {
		return VersionSnapshot{}, err
	}
	snapshot.CurrentHash, err = m.observeCurrentHash(snapshot.Chapter)
	if err != nil {
		return VersionSnapshot{}, err
	}
	return snapshot, nil
}

func (m *Manager) ListVersions(chapter int) ([]VersionSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(m.versionsDir())
	if os.IsNotExist(err) {
		return []VersionSnapshot{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]VersionSnapshot, 0, len(entries))
	currentHashes := make(map[int]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if !safeMetadataID(id) {
			continue
		}
		snapshot, err := m.readVersion(id)
		if err != nil {
			return nil, err
		}
		if chapter <= 0 || snapshot.Chapter == chapter {
			currentHash, ok := currentHashes[snapshot.Chapter]
			if !ok {
				currentHash, err = m.observeCurrentHash(snapshot.Chapter)
				if err != nil {
					return nil, err
				}
				currentHashes[snapshot.Chapter] = currentHash
			}
			snapshot.CurrentHash = currentHash
			result = append(result, snapshot)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (m *Manager) RestoreVersion(id string, expectedCurrentHash string) (VersionSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(expectedCurrentHash) == "" {
		return VersionSnapshot{}, fmt.Errorf("Restore 必须提供发起操作时看到的当前正文 hash")
	}
	snapshot, err := m.readVersion(id)
	if err != nil {
		return VersionSnapshot{}, err
	}
	if m.saveChapter == nil {
		return VersionSnapshot{}, errors.New("version Restore 未配置 Core SaveChapter 适配器")
	}
	current, err := m.currentContent(snapshot.Chapter)
	if err != nil {
		return VersionSnapshot{}, err
	}
	currentHash := domain.ChapterContentSHA256(current)
	if currentHash != expectedCurrentHash {
		return VersionSnapshot{}, fmt.Errorf("%w：当前正文已变化，不能直接恢复版本", ErrVersionStale)
	}
	journal := operationJournal{
		ID: newID(), Kind: "version", VersionID: snapshot.ID, Stage: "planned",
		Planned:   []journalItem{{Chapter: snapshot.Chapter, BaseHash: currentHash, AfterHash: snapshot.ContentHash}},
		UpdatedAt: time.Now().UTC(),
	}
	if err := m.writeJournal(journal); err != nil {
		return VersionSnapshot{}, err
	}
	if _, _, err := m.createVersionUnlocked(CreateVersionRequest{
		ProjectID:    snapshot.ProjectID,
		ResourceType: snapshot.ResourceType,
		ResourceID:   snapshot.ResourceID,
		Chapter:      snapshot.Chapter,
		Content:      current,
		Source:       "restore-before",
		ParentID:     snapshot.ID,
	}); err != nil {
		return VersionSnapshot{}, err
	}
	journal.Stage = "applying"
	journal.UpdatedAt = time.Now().UTC()
	if err := m.writeJournal(journal); err != nil {
		return VersionSnapshot{}, err
	}
	if err := m.saveChapter(snapshot.Chapter, snapshot.Content); err != nil {
		journal.Stage = "failed"
		journal.Error = err.Error()
		journal.UpdatedAt = time.Now().UTC()
		_ = m.writeJournal(journal)
		return VersionSnapshot{}, err
	}
	journal.Stage = "applied"
	journal.UpdatedAt = time.Now().UTC()
	if err := m.writeJournal(journal); err != nil {
		return VersionSnapshot{}, err
	}
	if _, _, err := m.createVersionUnlocked(CreateVersionRequest{
		ProjectID:    snapshot.ProjectID,
		ResourceType: snapshot.ResourceType,
		ResourceID:   snapshot.ResourceID,
		Chapter:      snapshot.Chapter,
		Content:      snapshot.Content,
		Source:       "restore-after",
		ParentID:     snapshot.ID,
	}); err != nil {
		return VersionSnapshot{}, err
	}
	journal.Stage = "committed"
	journal.UpdatedAt = time.Now().UTC()
	if err := m.writeJournal(journal); err != nil {
		return VersionSnapshot{}, err
	}
	return snapshot, nil
}

func (m *Manager) observeCurrentHash(chapter int) (string, error) {
	content, err := m.currentContent(chapter)
	if err != nil {
		return "", err
	}
	return domain.ChapterContentSHA256(content), nil
}

func (m *Manager) readVersion(id string) (VersionSnapshot, error) {
	if !safeMetadataID(id) {
		return VersionSnapshot{}, ErrVersionNotFound
	}
	data, err := os.ReadFile(filepath.Join(m.versionsDir(), id+".json"))
	if os.IsNotExist(err) {
		return VersionSnapshot{}, fmt.Errorf("%w: %s", ErrVersionNotFound, id)
	}
	if err != nil {
		return VersionSnapshot{}, err
	}
	var snapshot VersionSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return VersionSnapshot{}, err
	}
	return snapshot, nil
}
