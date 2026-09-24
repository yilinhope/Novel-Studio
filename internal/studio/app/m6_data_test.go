package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func TestReadConfigSnapshotUsesProjectRootAndMasksAPIKey(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".ainovel")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := bootstrap.Config{
		Provider: "proxy", ModelName: "writer-model",
		Providers: map[string]bootstrap.ProviderConfig{
			"proxy": {Type: "openai", API: "chat", BaseURL: "https://example.invalid", APIKey: "sk-project-secret-123456", Models: []bootstrap.ModelConfig{{Name: "writer-model"}}},
		},
	}
	path := filepath.Join(configDir, "config.json")
	if err := bootstrap.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	snapshot, err := ReadConfigSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ConfigPath != path || snapshot.ProjectRoot != root {
		t.Fatalf("配置根目录未保持为 ProjectRoot: %+v", snapshot)
	}
	var proxy *viewmodel.ProviderConfig
	for i := range snapshot.Providers {
		if snapshot.Providers[i].Name == "proxy" {
			proxy = &snapshot.Providers[i]
		}
	}
	if proxy == nil || !proxy.HasAPIKey || proxy.APIKeyHint == "sk-project-secret-123456" || proxy.APIKeyHint == "" {
		t.Fatalf("API key 未按 Core 脱敏投影: %+v", snapshot.Providers)
	}
}
