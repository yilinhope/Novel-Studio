package bridge

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func TestParityReadOnlyUsesCurrentProjectAndDoesNotCreateHost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "项目", "output", "novel")
	st := store.NewStore(path)
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SavePremise("只读前提"); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	project, err := a.OpenProject(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.GetStoryPremise(viewmodel.ReadRequest{ProjectID: project.OutputDir, Generation: project.Generation, RequestID: "req-a", Sequence: 7})
	if err != nil {
		t.Fatal(err)
	}
	if result.Premise != "只读前提" || result.ProjectID != project.OutputDir || result.ProjectRoot != project.ProjectRoot || result.OutputDir != project.OutputDir || result.Generation != project.Generation || result.RequestID != "req-a" || result.Sequence != 7 {
		t.Fatalf("读取响应身份不正确：%+v project=%+v", result, project)
	}
	if a.engine != nil {
		t.Fatal("只读 Story API 不得创建 Host/Engine")
	}
	if _, err := a.GetStoryCharacters(viewmodel.ReadRequest{ProjectID: filepath.Join(t.TempDir(), "other"), Generation: project.Generation}); err == nil || !strings.Contains(err.Error(), "过期") {
		t.Fatalf("旧项目请求必须被拒绝：%v", err)
	}
	if _, err := a.GetContinuityTimeline(viewmodel.ReadRequest{ProjectID: project.OutputDir, Generation: project.Generation + 1}); err == nil || !strings.Contains(err.Error(), "过期") {
		t.Fatalf("旧 generation 请求必须被拒绝：%v", err)
	}
	if a.engine != nil {
		t.Fatal("拒绝 stale 读取也不得创建 Host/Engine")
	}
}
