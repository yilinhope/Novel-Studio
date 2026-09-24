package viewmodel

import "time"

// OperationAck 表示一个已被 Studio 接受的异步操作。
type OperationAck struct {
	ProjectID  string `json:"projectId"`
	Generation uint64 `json:"generation"`
	Operation  string `json:"operation"`
}

type CreateProjectRequest struct {
	ProjectRoot string `json:"projectRoot"`
	Prompt      string `json:"prompt"`
	Mode        string `json:"mode"` // quick / outline / cocreate
}

type CreateEvent struct {
	ProjectID  string   `json:"projectId"`
	Generation uint64   `json:"generation"`
	Operation  string   `json:"operation"`
	State      string   `json:"state"` // started / running / completed / error
	Message    string   `json:"message,omitempty"`
	Error      string   `json:"error,omitempty"`
	Project    *Project `json:"project,omitempty"`
	Runtime    *Runtime `json:"runtime,omitempty"`
}

type CoCreateMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CoCreateStart struct {
	OperationAck
	Mode string `json:"mode"` // cold / stage
}

type CoCreateEvent struct {
	ProjectID   string            `json:"projectId"`
	Generation  uint64            `json:"generation"`
	State       string            `json:"state"` // thinking / reply / ready / error / cancelled
	Kind        string            `json:"kind,omitempty"`
	Text        string            `json:"text,omitempty"`
	Reply       string            `json:"reply,omitempty"`
	Draft       string            `json:"draft,omitempty"`
	Ready       bool              `json:"ready"`
	Suggestions []string          `json:"suggestions,omitempty"`
	History     []CoCreateMessage `json:"history,omitempty"`
	Error       string            `json:"error,omitempty"`
}

type CoCreateRecovery struct {
	ProjectID   string            `json:"projectId"`
	Exists      bool              `json:"exists"`
	Interrupted bool              `json:"interrupted"`
	Mode        string            `json:"mode"`
	History     []CoCreateMessage `json:"history,omitempty"`
	Draft       string            `json:"draft,omitempty"`
	Ready       bool              `json:"ready"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Error       string            `json:"error,omitempty"`
}

type ImportOptions struct {
	ProjectRoot        string `json:"projectRoot"`
	SourcePath         string `json:"sourcePath"`
	AutoConfirm        bool   `json:"autoConfirm"`
	AcceptSegmentation bool   `json:"acceptSegmentation"`
	StoryResolution    string `json:"storyResolution"`
	ContinueAfter      bool   `json:"continueAfter"`
	Guidance           string `json:"guidance"`
}

type ImportChapter struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
	Uncertain bool   `json:"uncertain"`
}

type ImportStatus struct {
	ProjectID    string          `json:"projectId"`
	Generation   uint64          `json:"generation"`
	Active       bool            `json:"active"`
	Stage        string          `json:"stage"`
	Current      int             `json:"current"`
	Total        int             `json:"total"`
	Message      string          `json:"message"`
	Level        string          `json:"level,omitempty"`
	Key          string          `json:"key,omitempty"`
	RetryAt      time.Time       `json:"retryAt,omitempty"`
	Error        string          `json:"error,omitempty"`
	Continued    bool            `json:"continued"`
	RecoveryHint string          `json:"recoveryHint,omitempty"`
	Chapters     []ImportChapter `json:"chapters,omitempty"`
	Uncertain    []int           `json:"uncertain,omitempty"`
	Notes        []string        `json:"notes,omitempty"`
}

type ExportOptions struct {
	Format    string `json:"format"`
	OutPath   string `json:"outPath"`
	From      int    `json:"from"`
	To        int    `json:"to"`
	Overwrite bool   `json:"overwrite"`
}

type ExportResult struct {
	Path     string `json:"path"`
	Chapters int    `json:"chapters"`
	Bytes    int    `json:"bytes"`
	Skipped  []int  `json:"skipped,omitempty"`
}

type ModelConfig struct {
	Name          string `json:"name"`
	ContextWindow int    `json:"contextWindow,omitempty"`
	JSONSchema    *bool  `json:"jsonSchema,omitempty"`
}

type ProviderConfig struct {
	Name              string        `json:"name"`
	Type              string        `json:"type"`
	API               string        `json:"api"`
	BaseURL           string        `json:"baseUrl"`
	StreamIdleTimeout string        `json:"streamIdleTimeout,omitempty"`
	Models            []ModelConfig `json:"models"`
	HasAPIKey         bool          `json:"hasApiKey"`
	APIKeyHint        string        `json:"apiKeyHint,omitempty"`
	RequiresAPIKey    bool          `json:"requiresApiKey"`
}

type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type RoleConfig struct {
	Provider        string     `json:"provider"`
	Model           string     `json:"model"`
	ReasoningEffort string     `json:"reasoningEffort,omitempty"`
	Fallbacks       []ModelRef `json:"fallbacks,omitempty"`
}

type BudgetConfig struct {
	BookUSD   float64 `json:"bookUsd"`
	WarnRatio float64 `json:"warnRatio"`
	HardStop  bool    `json:"hardStop"`
}

type NotifyConfig struct {
	Enabled *bool    `json:"enabled,omitempty"`
	Command string   `json:"command,omitempty"`
	Events  []string `json:"events,omitempty"`
}

type ConfigSnapshot struct {
	ProjectRoot     string                `json:"projectRoot"`
	ConfigPath      string                `json:"configPath"`
	Provider        string                `json:"provider"`
	Model           string                `json:"model"`
	ReasoningEffort string                `json:"reasoningEffort,omitempty"`
	Style           string                `json:"style,omitempty"`
	ContextWindow   int                   `json:"contextWindow,omitempty"`
	Providers       []ProviderConfig      `json:"providers"`
	Roles           map[string]RoleConfig `json:"roles"`
	Budget          BudgetConfig          `json:"budget"`
	Notify          NotifyConfig          `json:"notify"`
}

type ProviderDraft struct {
	Provider     string        `json:"provider"`
	Type         string        `json:"type"`
	API          string        `json:"api"`
	BaseURL      string        `json:"baseUrl"`
	Models       []ModelConfig `json:"models"`
	Renames      []ModelRename `json:"renames,omitempty"`
	APIKeyAction string        `json:"apiKeyAction"`
	APIKey       string        `json:"apiKey,omitempty"`
}

type ModelRename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ModelSelection struct {
	Role     string `json:"role"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type RoleThinking struct {
	Role  string `json:"role"`
	Level string `json:"level"`
}

type UsageTotals struct {
	Input        int     `json:"input"`
	Output       int     `json:"output"`
	CacheRead    int     `json:"cacheRead"`
	CacheWrite   int     `json:"cacheWrite"`
	Cost         float64 `json:"costUsd"`
	Saved        float64 `json:"savedUsd"`
	CacheCapable bool    `json:"cacheCapable"`
	CacheBreaks  int     `json:"cacheBreaks"`
}

type AgentUsage struct {
	Role         string  `json:"role,omitempty"`
	Model        string  `json:"model,omitempty"`
	Input        int     `json:"input"`
	Output       int     `json:"output"`
	CacheRead    int     `json:"cacheRead"`
	CacheWrite   int     `json:"cacheWrite"`
	Cost         float64 `json:"costUsd"`
	Saved        float64 `json:"savedUsd"`
	CacheCapable bool    `json:"cacheCapable"`
}

type UsageSnapshot struct {
	ProjectID    string       `json:"projectId"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	Overall      UsageTotals  `json:"overall"`
	PerAgent     []AgentUsage `json:"perAgent"`
	PerModel     []AgentUsage `json:"perModel"`
	MissingUsage int          `json:"missingUsage"`
	Budget       BudgetConfig `json:"budget"`
}
