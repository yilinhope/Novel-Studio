package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/host/exp"
	"github.com/voocel/ainovel-cli/internal/host/imp"
	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func ReadConfigSnapshot(projectRoot string) (viewmodel.ConfigSnapshot, error) {
	projectRoot = absPath(projectRoot)
	cfg, err := bootstrap.LoadConfigFromDir(projectRoot)
	if err != nil {
		return viewmodel.ConfigSnapshot{}, fmt.Errorf("读取项目配置失败：%w", err)
	}
	cfg.FillDefaults()
	result := viewmodel.ConfigSnapshot{
		ProjectRoot: projectRoot, ConfigPath: bootstrap.EffectiveConfigPathFromDir(projectRoot),
		Provider: cfg.Provider, Model: cfg.ModelName, ReasoningEffort: cfg.ReasoningEffort,
		Style: cfg.Style, ContextWindow: cfg.ContextWindow, Roles: map[string]viewmodel.RoleConfig{},
		Budget: viewmodel.BudgetConfig{BookUSD: cfg.Budget.BookUSD, WarnRatio: cfg.Budget.WarnRatio, HardStop: cfg.Budget.HardStop},
		Notify: viewmodel.NotifyConfig{Enabled: cloneBool(cfg.Notify.Enabled), Command: cfg.Notify.Command, Events: append([]string(nil), cfg.Notify.Events...)},
	}
	providers := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		providers = append(providers, name)
	}
	sort.Strings(providers)
	for _, name := range providers {
		pc := cfg.Providers[name]
		models := make([]viewmodel.ModelConfig, 0, len(pc.Models))
		for _, model := range pc.Models {
			models = append(models, viewmodel.ModelConfig{Name: model.Name, ContextWindow: model.ContextWindow, JSONSchema: cloneBool(model.JSONSchema)})
		}
		result.Providers = append(result.Providers, viewmodel.ProviderConfig{
			Name: name, Type: pc.Type, API: pc.API, BaseURL: pc.BaseURL, StreamIdleTimeout: pc.StreamIdleTimeout,
			Models: models, HasAPIKey: pc.APIKey != "", APIKeyHint: host.MaskAPIKey(pc.APIKey), RequiresAPIKey: pc.RequiresAPIKey(name),
		})
	}
	for role, rc := range cfg.Roles {
		roleView := viewmodel.RoleConfig{Provider: rc.Provider, Model: rc.Model, ReasoningEffort: rc.ReasoningEffort}
		for _, fallback := range rc.Fallbacks {
			roleView.Fallbacks = append(roleView.Fallbacks, viewmodel.ModelRef{Provider: fallback.Provider, Model: fallback.Model})
		}
		result.Roles[role] = roleView
	}
	return result, nil
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func BuildModelDraft(draft viewmodel.ProviderDraft) host.ModelConfigurationDraft {
	models := make([]bootstrap.ModelConfig, 0, len(draft.Models))
	for _, model := range draft.Models {
		models = append(models, bootstrap.ModelConfig{Name: model.Name, ContextWindow: model.ContextWindow, JSONSchema: cloneBool(model.JSONSchema)})
	}
	renames := make([]host.ModelRename, 0, len(draft.Renames))
	for _, rename := range draft.Renames {
		renames = append(renames, host.ModelRename{From: rename.From, To: rename.To})
	}
	return host.ModelConfigurationDraft{
		Provider: draft.Provider, Type: draft.Type, API: draft.API, BaseURL: draft.BaseURL,
		Models: models, Renames: renames, APIKeyAction: host.APIKeyAction(draft.APIKeyAction), APIKey: draft.APIKey,
	}
}

func ReadCoCreateRecovery(outputDir string) (viewmodel.CoCreateRecovery, error) {
	recovery, err := host.ReadCoCreateRecovery(outputDir)
	if err != nil {
		return viewmodel.CoCreateRecovery{}, err
	}
	history := make([]viewmodel.CoCreateMessage, 0, len(recovery.History))
	for _, item := range recovery.History {
		history = append(history, viewmodel.CoCreateMessage{Role: item.Role, Content: item.Content})
	}
	return viewmodel.CoCreateRecovery{ProjectID: outputDir, Exists: recovery.Exists, Interrupted: recovery.Interrupted, Mode: recovery.Mode, History: history, Draft: recovery.Draft, Ready: recovery.Ready, Suggestions: append([]string(nil), recovery.Suggestions...), Error: recovery.Error}, nil
}

func ReadUsageSnapshot(outputDir, projectID string, live *host.UISnapshot, budget bootstrap.BudgetConfig) (viewmodel.UsageSnapshot, error) {
	result := viewmodel.UsageSnapshot{ProjectID: projectID, Budget: viewmodel.BudgetConfig{BookUSD: budget.BookUSD, WarnRatio: budget.WarnRatio, HardStop: budget.HardStop}}
	if live != nil {
		result.UpdatedAt = timeFromHost(live)
		result.Overall = viewmodel.UsageTotals{Input: live.TotalInputTokens, Output: live.TotalOutputTokens, CacheRead: live.TotalCacheReadTokens, CacheWrite: live.TotalCacheWriteTokens, Cost: live.TotalCostUSD, Saved: live.TotalSavedUSD, CacheBreaks: live.TotalCacheBreaks}
		result.PerAgent = make([]viewmodel.AgentUsage, 0, len(live.CachePerAgent))
		for _, stat := range live.CachePerAgent {
			result.PerAgent = append(result.PerAgent, viewmodel.AgentUsage{Role: stat.Role, Model: stat.Model, Input: stat.Input, Output: stat.Output, CacheRead: stat.CacheRead, CacheWrite: stat.CacheWrite, Cost: stat.Cost, Saved: stat.Saved, CacheCapable: stat.CacheCapable})
		}
		result.PerModel = make([]viewmodel.AgentUsage, 0, len(live.CachePerModel))
		for _, stat := range live.CachePerModel {
			result.PerModel = append(result.PerModel, viewmodel.AgentUsage{Role: stat.Role, Model: stat.Model, Input: stat.Input, Output: stat.Output, CacheRead: stat.CacheRead, CacheWrite: stat.CacheWrite, Cost: stat.Cost, Saved: stat.Saved, CacheCapable: stat.CacheCapable})
		}
		result.MissingUsage = live.MissingAssistantUsage
		return result, nil
	}
	state, err := store.NewStore(outputDir).Usage.Load()
	if err != nil {
		return result, fmt.Errorf("读取项目用量失败：%w", err)
	}
	if state == nil {
		return result, nil
	}
	result.UpdatedAt = state.UpdatedAt
	result.Overall = usageTotals(state.Overall)
	for role, total := range state.PerAgent {
		result.PerAgent = append(result.PerAgent, viewmodel.AgentUsage{Role: role, Input: total.Input, Output: total.Output, CacheRead: total.CacheRead, CacheWrite: total.CacheWrite, Cost: total.Cost, Saved: total.Saved, CacheCapable: total.CacheCapable})
	}
	for model, total := range state.PerModel {
		result.PerModel = append(result.PerModel, viewmodel.AgentUsage{Model: model, Input: total.Input, Output: total.Output, CacheRead: total.CacheRead, CacheWrite: total.CacheWrite, Cost: total.Cost, Saved: total.Saved, CacheCapable: total.CacheCapable})
	}
	sort.Slice(result.PerAgent, func(i, j int) bool { return result.PerAgent[i].Cost > result.PerAgent[j].Cost })
	sort.Slice(result.PerModel, func(i, j int) bool { return result.PerModel[i].Cost > result.PerModel[j].Cost })
	result.MissingUsage = state.MissingUsage
	return result, nil
}

func timeFromHost(snapshot *host.UISnapshot) time.Time {
	if snapshot == nil {
		return time.Time{}
	}
	return time.Now()
}

func usageTotals(total domain.AgentUsageTotals) viewmodel.UsageTotals {
	return viewmodel.UsageTotals{Input: total.Input, Output: total.Output, CacheRead: total.CacheRead, CacheWrite: total.CacheWrite, Cost: total.Cost, Saved: total.Saved, CacheCapable: total.CacheCapable, CacheBreaks: total.CacheBreaks}
}

func ReadImportStatus(outputDir, projectID string, generation uint64) (viewmodel.ImportStatus, error) {
	result := viewmodel.ImportStatus{ProjectID: projectID, Generation: generation}
	st := store.NewStore(outputDir)
	w := imp.OpenWorkspace(outputDir)
	if !w.Active() {
		return result, nil
	}
	result.Active = true
	result.RecoveryHint = imp.ResumeSummary(st)
	facts, err := imp.CollectFacts(st, w)
	if err != nil {
		result.Stage, result.Error = "error", err.Error()
		return result, nil
	}
	switch action := imp.NextAction(facts); action {
	case imp.ActionIngest:
		result.Stage = "ingesting"
	case imp.ActionSegment:
		result.Stage = "segmenting"
	case imp.ActionAwaitConfirmation:
		result.Stage, result.Current, result.Total = "awaiting_confirmation", facts.ExpectedChapters, facts.ExpectedChapters
	case imp.ActionAnalyze:
		result.Stage, result.Current, result.Total = "analyzing", facts.AnalyzedChapters, facts.ExpectedChapters
	case imp.ActionSynthesize:
		result.Stage, result.Current, result.Total = "synthesizing", facts.ExpectedChapters, facts.ExpectedChapters
	case imp.ActionAwaitStoryResolution:
		result.Stage = "awaiting_story_resolution"
	case imp.ActionPublish:
		result.Stage, result.Current, result.Total = "publishing", 0, facts.ExpectedChapters
	case imp.ActionDone:
		result.Stage, result.Current, result.Total = "done", facts.ExpectedChapters, facts.ExpectedChapters
	}
	if segmentation, segErr := imp.LoadSegmentation(st); segErr == nil {
		result.Uncertain = append([]int(nil), segmentation.Uncertain...)
		result.Notes = append([]string(nil), segmentation.Notes...)
		for _, chapter := range segmentation.Chapters {
			uncertain := false
			for _, n := range segmentation.Uncertain {
				if n == chapter.Number {
					uncertain = true
					break
				}
			}
			result.Chapters = append(result.Chapters, viewmodel.ImportChapter{Number: chapter.Number, Title: chapter.Title, StartByte: chapter.Start, EndByte: chapter.End, Uncertain: uncertain})
		}
	}
	return result, nil
}

func Export(ctx context.Context, outputDir string, options viewmodel.ExportOptions) (viewmodel.ExportResult, error) {
	result, err := exp.Run(ctx, exp.Deps{Store: store.NewStore(outputDir)}, exp.Options{Format: exp.Format(strings.ToLower(options.Format)), OutPath: options.OutPath, From: options.From, To: options.To, Overwrite: options.Overwrite})
	if err != nil {
		return viewmodel.ExportResult{}, err
	}
	return viewmodel.ExportResult{Path: result.Path, Chapters: result.Chapters, Bytes: result.Bytes, Skipped: result.Skipped}, nil
}
