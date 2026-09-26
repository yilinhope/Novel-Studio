package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

const (
	modelDiscoveryTimeout    = 15 * time.Second
	maxModelDiscoveryEntries = 500
	maxModelDiscoveryBytes   = 2 << 20
)

type modelDiscoveryHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// DiscoverProviderModels 读取服务商公开的模型目录；它只返回待保存的草稿，不修改配置。
func DiscoverProviderModels(ctx context.Context, draft viewmodel.ProviderDraft) ([]viewmodel.ModelConfig, error) {
	client := &http.Client{
		Timeout: modelDiscoveryTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return discoverProviderModels(ctx, draft, client)
}

func discoverProviderModels(ctx context.Context, draft viewmodel.ProviderDraft, client modelDiscoveryHTTPClient) ([]viewmodel.ModelConfig, error) {
	protocol := strings.ToLower(strings.TrimSpace(draft.Type))
	if protocol == "" {
		protocol = "openai"
	}
	if protocol != "openai" && protocol != "deepseek" && protocol != "gemini" {
		return nil, fmt.Errorf("当前协议不支持自动读取模型列表，请手动输入模型名")
	}
	if (protocol == "gemini" || protocol == "deepseek") && strings.TrimSpace(draft.APIKey) == "" {
		return nil, fmt.Errorf("%s 读取模型列表需要 API Key", map[string]string{"gemini": "Gemini", "deepseek": "DeepSeek"}[protocol])
	}
	endpoint, err := modelDiscoveryEndpoint(draft.BaseURL, protocol)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("模型列表请求地址无效")
	}
	request.Header.Set("Accept", "application/json")
	if protocol == "gemini" {
		request.Header.Set("x-goog-api-key", strings.TrimSpace(draft.APIKey))
	} else if key := strings.TrimSpace(draft.APIKey); key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("读取模型列表失败：网络请求未完成")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, modelDiscoveryHTTPError(response.StatusCode)
	}
	if response.ContentLength > maxModelDiscoveryBytes {
		return nil, fmt.Errorf("模型列表响应过大，已拒绝读取")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelDiscoveryBytes+1))
	if err != nil || len(body) > maxModelDiscoveryBytes {
		return nil, fmt.Errorf("读取模型列表响应失败")
	}
	if protocol == "gemini" {
		return parseGeminiModels(body)
	}
	return parseOpenAIModels(body)
}

func modelDiscoveryEndpoint(rawBaseURL, protocol string) (string, error) {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	parsed, err := url.Parse(rawBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("Base URL 必须是无凭据、无查询参数的 HTTP(S) 地址")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("Base URL 只支持 HTTP 或 HTTPS")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if protocol == "gemini" {
		switch {
		case strings.HasSuffix(path, "/v1beta/models"):
		case strings.HasSuffix(path, "/v1beta"):
			path += "/models"
		default:
			path += "/v1beta/models"
		}
	} else {
		switch {
		case strings.HasSuffix(path, "/chat/completions"):
			path = strings.TrimSuffix(path, "/chat/completions")
		case strings.HasSuffix(path, "/chat"):
			path = strings.TrimSuffix(path, "/chat")
		}
		if strings.HasSuffix(path, "/models") {
			// 已经是模型目录地址。
		} else if path == "" {
			if protocol == "deepseek" {
				path = "/models"
			} else {
				path = "/v1/models"
			}
		} else {
			path += "/models"
		}
	}
	parsed.Path = "/" + strings.TrimLeft(path, "/")
	parsed.RawPath = ""
	return parsed.String(), nil
}

func parseOpenAIModels(body []byte) ([]viewmodel.ModelConfig, error) {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Data == nil {
		return nil, fmt.Errorf("服务商返回的模型列表格式无效")
	}
	models := make([]viewmodel.ModelConfig, 0, len(payload.Data))
	seen := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		name := strings.TrimSpace(item.ID)
		if !safeDiscoveredModelName(name) {
			return nil, fmt.Errorf("服务商返回了无效模型名称")
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, viewmodel.ModelConfig{Name: name})
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("服务商没有返回可用模型")
	}
	if len(models) > maxModelDiscoveryEntries {
		return nil, fmt.Errorf("服务商返回的模型数量超过上限")
	}
	return models, nil
}

func parseGeminiModels(body []byte) ([]viewmodel.ModelConfig, error) {
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Models == nil {
		return nil, fmt.Errorf("服务商返回的模型列表格式无效")
	}
	models := make([]viewmodel.ModelConfig, 0, len(payload.Models))
	seen := make(map[string]struct{}, len(payload.Models))
	for _, item := range payload.Models {
		name := strings.TrimPrefix(strings.TrimSpace(item.Name), "models/")
		if !safeDiscoveredModelName(name) {
			return nil, fmt.Errorf("服务商返回了无效模型名称")
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, viewmodel.ModelConfig{Name: name})
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("服务商没有返回可用模型")
	}
	if len(models) > maxModelDiscoveryEntries {
		return nil, fmt.Errorf("服务商返回的模型数量超过上限")
	}
	return models, nil
}

func safeDiscoveredModelName(value string) bool {
	if value == "" || len([]byte(value)) > 512 {
		return false
	}
	for _, r := range value {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return false
		}
	}
	return true
}

func modelDiscoveryHTTPError(status int) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("读取模型列表失败：服务商认证失败（HTTP %d）", status)
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests:
		return fmt.Errorf("读取模型列表失败：服务商暂时不可用（HTTP %d）", status)
	default:
		if status >= 500 {
			return fmt.Errorf("读取模型列表失败：服务商暂时不可用（HTTP %d）", status)
		}
		return fmt.Errorf("读取模型列表失败：服务商拒绝请求（HTTP %d）", status)
	}
}
