package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func TestResolveProviderDraftAPIKeyUsesSavedKeyForDiscovery(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".ainovel")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := bootstrap.SaveConfig(filepath.Join(configDir, "config.json"), bootstrap.Config{
		Provider: "proxy", ModelName: "model-a",
		Providers: map[string]bootstrap.ProviderConfig{
			"proxy": {Type: "openai", APIKey: "saved-secret"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	draft, err := ResolveProviderDraftAPIKey(root, viewmodel.ProviderDraft{Provider: "proxy", APIKeyAction: "keep"})
	if err != nil {
		t.Fatal(err)
	}
	if draft.APIKey != "saved-secret" {
		t.Fatalf("未从 Core 配置补回已保存 API Key：%q", draft.APIKey)
	}
}

func TestResolveProviderDraftAPIKeyDoesNotReplaceExplicitDraftKey(t *testing.T) {
	root := t.TempDir()
	draft, err := ResolveProviderDraftAPIKey(root, viewmodel.ProviderDraft{Provider: "proxy", APIKeyAction: "replace", APIKey: "draft-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if draft.APIKey != "draft-secret" {
		t.Fatalf("显式草稿 API Key 被覆盖：%q", draft.APIKey)
	}
}
