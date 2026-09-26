package app

import (
	"fmt"
	"strings"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

// ResolveProviderDraftAPIKey 在受信任的本地边界内补回已保存 Provider 的凭证。
// API Key 不进入 ConfigSnapshot；发现结果也不会返回该值。
func ResolveProviderDraftAPIKey(projectRoot string, draft viewmodel.ProviderDraft) (viewmodel.ProviderDraft, error) {
	apiKeyAction := strings.TrimSpace(draft.APIKeyAction)
	if strings.TrimSpace(draft.APIKey) != "" || (apiKeyAction != "" && apiKeyAction != "keep") {
		return draft, nil
	}
	cfg, err := bootstrap.LoadConfigFromDir(projectRoot)
	if err != nil {
		return viewmodel.ProviderDraft{}, fmt.Errorf("读取项目配置失败：%w", err)
	}
	if provider, ok := cfg.Providers[strings.TrimSpace(draft.Provider)]; ok {
		draft.APIKey = provider.APIKey
	}
	return draft, nil
}
