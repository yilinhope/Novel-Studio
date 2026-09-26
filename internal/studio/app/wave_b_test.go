package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestWaveBReadServicesUseProjectRootAndDoNotCreateHost(t *testing.T) {
	output := fixture(t)
	root := filepath.Dir(filepath.Dir(output))
	if err := os.MkdirAll(filepath.Join(root, "simulate"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "simulate", "sample.txt"), []byte("sample"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".ainovel", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ainovel", "rules", "project.md"), []byte("project rule"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := store.NewStore(output)
	profile := domain.SimulationProfile{Version: domain.SimulationProfileVersion, Corpus: domain.SimulationCorpusManifest{Sources: []domain.SimulationSource{{RelativePath: "sample.txt", SHA256: "old", Fingerprint: "sample.txt:old"}}}}
	if err := st.Simulation.Save(profile); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	if _, err := service.OpenProject(root); err != nil {
		t.Fatal(err)
	}
	sources, err := service.GetSimulationSources()
	if err != nil || len(sources.Items) != 1 || !sources.Items[0].Changed {
		t.Fatalf("simulation source scope/change mismatch: %+v %v", sources, err)
	}
	workspace, err := service.GetRuleFiles()
	if err != nil || len(workspace.Project) != 1 {
		t.Fatalf("project rules scope mismatch: %+v %v", workspace, err)
	}
	if workspace.EffectiveAvailable {
		t.Fatal("read-only rules view must not build a missing effective snapshot")
	}
	if _, err := service.GetSimulationProfile(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetStyleState(); err != nil {
		t.Fatal(err)
	}
}
