package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

type discoveryRoundTripper func(*http.Request) (*http.Response, error)

func (f discoveryRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func discoveryResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestDiscoverProviderModelsOpenAICompatible(t *testing.T) {
	var requestURL string
	var authorization string
	models, err := discoverProviderModels(context.Background(), viewmodel.ProviderDraft{
		Provider: "proxy",
		Type:     "openai",
		BaseURL:  "https://example.invalid/v1/chat/completions",
		APIKey:   "sk-secret",
	}, &http.Client{Transport: discoveryRoundTripper(func(request *http.Request) (*http.Response, error) {
		requestURL = request.URL.String()
		authorization = request.Header.Get("Authorization")
		return discoveryResponse(http.StatusOK, `{"data":[{"id":"model-a"},{"id":"model-a"},{"id":"model-b","name":"模型 B"}]}`), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "https://example.invalid/v1/models" {
		t.Fatalf("OpenAI-compatible models URL 错误：%s", requestURL)
	}
	if authorization != "Bearer sk-secret" {
		t.Fatalf("API Key 未以 Authorization 发送：%q", authorization)
	}
	if len(models) != 2 || models[0].Name != "model-a" || models[1].Name != "model-b" {
		t.Fatalf("模型列表去重或解析错误：%+v", models)
	}
}

func TestDiscoverProviderModelsDeepSeekUsesNativeModelsEndpoint(t *testing.T) {
	var requestURL string
	var authorization string
	models, err := discoverProviderModels(context.Background(), viewmodel.ProviderDraft{
		Provider: "deepseek-proxy",
		Type:     "deepseek",
		BaseURL:  "https://proxy.example.invalid",
		APIKey:   "deepseek-secret",
	}, &http.Client{Transport: discoveryRoundTripper(func(request *http.Request) (*http.Response, error) {
		requestURL = request.URL.String()
		authorization = request.Header.Get("Authorization")
		return discoveryResponse(http.StatusOK, `{"data":[{"id":"deepseek-v4-pro"}]}`), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "https://proxy.example.invalid/models" {
		t.Fatalf("DeepSeek 模型目录 URL 错误：%s", requestURL)
	}
	if authorization != "Bearer deepseek-secret" {
		t.Fatalf("DeepSeek API Key 未以 Authorization 发送：%q", authorization)
	}
	if len(models) != 1 || models[0].Name != "deepseek-v4-pro" {
		t.Fatalf("DeepSeek 模型列表解析错误：%+v", models)
	}
}

func TestDiscoverProviderModelsDeepSeekRequiresAPIKey(t *testing.T) {
	_, err := discoverProviderModels(context.Background(), viewmodel.ProviderDraft{
		Provider: "deepseek-proxy",
		Type:     "deepseek",
		BaseURL:  "https://proxy.example.invalid",
	}, &http.Client{})
	if err == nil || !strings.Contains(err.Error(), "DeepSeek") || !strings.Contains(err.Error(), "API Key") {
		t.Fatalf("DeepSeek 缺少 API Key 应明确拒绝：%v", err)
	}
}

func TestDiscoverProviderModelsGeminiUsesHeaderAndNormalizesName(t *testing.T) {
	var requestURL string
	var apiKey string
	models, err := discoverProviderModels(context.Background(), viewmodel.ProviderDraft{
		Provider: "gemini",
		Type:     "gemini",
		BaseURL:  "https://generativelanguage.googleapis.com",
		APIKey:   "gemini-secret",
	}, &http.Client{Transport: discoveryRoundTripper(func(request *http.Request) (*http.Response, error) {
		requestURL = request.URL.String()
		apiKey = request.Header.Get("x-goog-api-key")
		return discoveryResponse(http.StatusOK, `{"models":[{"name":"models/gemini-2.5-flash","displayName":"Gemini Flash"}]}`), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "https://generativelanguage.googleapis.com/v1beta/models" {
		t.Fatalf("Gemini models URL 错误：%s", requestURL)
	}
	if apiKey != "gemini-secret" {
		t.Fatalf("Gemini API Key 未使用专用 header：%q", apiKey)
	}
	if len(models) != 1 || models[0].Name != "gemini-2.5-flash" {
		t.Fatalf("Gemini 模型名未去除 models/ 前缀：%+v", models)
	}
}

func TestDiscoverProviderModelsDoesNotExposeSecretInErrors(t *testing.T) {
	secret := "sk-secret-value"
	_, err := discoverProviderModels(context.Background(), viewmodel.ProviderDraft{
		Provider: "proxy",
		Type:     "openai",
		BaseURL:  "https://example.invalid/v1",
		APIKey:   secret,
	}, &http.Client{Transport: discoveryRoundTripper(func(request *http.Request) (*http.Response, error) {
		return discoveryResponse(http.StatusUnauthorized, `{"error":"unauthorized"}`), nil
	})})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("认证失败应返回不含 API Key 的错误：%v", err)
	}
}

func TestDiscoverProviderModelsRejectsUnsupportedProtocol(t *testing.T) {
	_, err := discoverProviderModels(context.Background(), viewmodel.ProviderDraft{
		Provider: "anthropic",
		Type:     "anthropic",
		BaseURL:  "https://example.invalid",
	}, &http.Client{})
	if err == nil || !strings.Contains(err.Error(), "手动输入") {
		t.Fatalf("不支持自动发现的协议应给出手动输入提示：%v", err)
	}
}
