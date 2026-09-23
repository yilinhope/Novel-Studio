package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/voocel/ainovel-cli/internal/revision"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// RevisionService 只读取 Core Store 的 revision 事实，不创建 Host 或调用模型。
type RevisionService struct {
	project      *Service
	runtimeState func() viewmodel.Runtime
	mu           sync.RWMutex
	latest       map[string]viewmodel.RevisionStatus
}

func NewRevisionService(project *Service, runtimeState func() viewmodel.Runtime) *RevisionService {
	return &RevisionService{
		project:      project,
		runtimeState: runtimeState,
		latest:       make(map[string]viewmodel.RevisionStatus),
	}
}

// GetRevisionStatus 返回当前进程中最近一次 Store 检查结果；从未检查时返回 unknown。
func (s *RevisionService) GetRevisionStatus() (viewmodel.RevisionStatus, error) {
	st, _, err := s.project.currentProject()
	if err != nil {
		return viewmodel.RevisionStatus{}, err
	}
	projectID := st.Dir()
	s.mu.RLock()
	status, ok := s.latest[revisionProjectKey(projectID)]
	s.mu.RUnlock()
	if !ok {
		return viewmodel.RevisionStatus{
			ProjectID: projectID,
			State:     viewmodel.RevisionUnknown,
			Chapters:  []viewmodel.UnsyncedChapter{},
		}, nil
	}
	status.Chapters = cloneUnsyncedChapters(status.Chapters)
	return status, nil
}

// CheckChapterRevisions 复用 Core pending 记录和 revision.Scan 做只读复核。
func (s *RevisionService) CheckChapterRevisions() (viewmodel.RevisionStatus, error) {
	st, _, err := s.project.currentProject()
	if err != nil {
		return viewmodel.RevisionStatus{}, err
	}
	projectID := st.Dir()
	if s.runtimeState != nil {
		switch state := s.runtimeState().State; state {
		case viewmodel.RuntimeRunning, viewmodel.RuntimePausing, viewmodel.RuntimeStopping:
			return viewmodel.RevisionStatus{}, fmt.Errorf("创作会话处于%s状态，暂不能检查章节修订", state)
		}
	}

	status := viewmodel.RevisionStatus{
		ProjectID: projectID,
		State:     viewmodel.RevisionSynced,
		Chapters:  []viewmodel.UnsyncedChapter{},
		CheckedAt: time.Now(),
	}
	pending, err := st.Revisions.LoadPending()
	if err != nil {
		return s.cacheError(projectID, status.CheckedAt, fmt.Errorf("读取修订恢复记录: %w", err))
	}
	if pending != nil {
		status.State = viewmodel.RevisionRecoveryPending
		status.HasUnsynced = true
		status.PendingStage = string(pending.Stage)
		for _, item := range pending.Items {
			status.Chapters = append(status.Chapters, viewmodel.UnsyncedChapter{
				Chapter:      item.Chapter,
				AcceptedHash: item.BaseSHA256,
				CurrentHash:  item.CurrentSHA256,
			})
		}
		s.cache(status)
		return status, nil
	}

	changes, err := revision.Scan(st)
	if err != nil {
		return s.cacheError(projectID, status.CheckedAt, err)
	}
	if len(changes) > 0 {
		status.State = viewmodel.RevisionSavedUnsynced
		status.HasUnsynced = true
		for _, change := range changes {
			status.Chapters = append(status.Chapters, viewmodel.UnsyncedChapter{
				Chapter:      change.Chapter,
				AcceptedHash: change.BaseSHA256,
				CurrentHash:  change.CurrentSHA256,
			})
		}
	}
	s.cache(status)
	return status, nil
}

func (s *RevisionService) cacheError(projectID string, checkedAt time.Time, cause error) (viewmodel.RevisionStatus, error) {
	status := viewmodel.RevisionStatus{
		ProjectID:   projectID,
		State:       viewmodel.RevisionError,
		HasUnsynced: true,
		Chapters:    []viewmodel.UnsyncedChapter{},
		CheckedAt:   checkedAt,
		Error:       cause.Error(),
	}
	s.cache(status)
	return status, cause
}

func (s *RevisionService) cache(status viewmodel.RevisionStatus) {
	status.Chapters = cloneUnsyncedChapters(status.Chapters)
	s.mu.Lock()
	s.latest[revisionProjectKey(status.ProjectID)] = status
	s.mu.Unlock()
}

func revisionProjectKey(path string) string {
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	return strings.ToLower(filepath.Clean(path))
}

func cloneUnsyncedChapters(chapters []viewmodel.UnsyncedChapter) []viewmodel.UnsyncedChapter {
	return append([]viewmodel.UnsyncedChapter{}, chapters...)
}
