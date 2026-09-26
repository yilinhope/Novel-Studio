package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/voocel/ainovel-cli/assets"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/host/sim"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/studio/viewmodel"
)

func (s *Service) GetSimulationSources() (viewmodel.SimulationSourcesPage, error) {
	st, projectRoot, err := s.currentProject()
	if err != nil {
		return viewmodel.SimulationSourcesPage{}, err
	}
	sourceDir := filepath.Join(projectRoot, "simulate")
	if _, statErr := os.Stat(sourceDir); statErr != nil {
		if os.IsNotExist(statErr) {
			return viewmodel.SimulationSourcesPage{SourceDir: sourceDir, Items: []viewmodel.SimulationSource{}}, nil
		}
		return viewmodel.SimulationSourcesPage{}, fmt.Errorf("读取仿写语料目录失败：%w", statErr)
	}
	items, err := sim.ListSources(sourceDir)
	if err != nil {
		return viewmodel.SimulationSourcesPage{}, fmt.Errorf("读取仿写语料失败：%w", err)
	}
	profile, err := st.Simulation.Load()
	if err != nil {
		return viewmodel.SimulationSourcesPage{}, fmt.Errorf("读取仿写画像失败：%w", err)
	}
	type knownSource struct{ fingerprint, analyzedAt string }
	known := map[string]knownSource{}
	if profile != nil {
		for _, source := range profile.Corpus.Sources {
			known[source.RelativePath] = knownSource{fingerprint: source.Fingerprint, analyzedAt: source.AnalyzedAt}
		}
		for _, report := range profile.SourceReports {
			item := known[report.RelativePath]
			item.analyzedAt = report.AnalyzedAt
			known[report.RelativePath] = item
		}
	}
	result := viewmodel.SimulationSourcesPage{SourceDir: sourceDir, Items: make([]viewmodel.SimulationSource, 0, len(items))}
	for _, source := range items {
		result.Items = append(result.Items, viewmodel.SimulationSource{
			RelativePath: source.RelativePath, SHA256: source.SHA256, Fingerprint: source.Fingerprint,
			SizeBytes: source.SizeBytes, ModTime: source.ModTime, AnalyzedAt: known[source.RelativePath].analyzedAt,
			Changed: known[source.RelativePath].fingerprint != "" && known[source.RelativePath].fingerprint != source.Fingerprint,
		})
	}
	return result, nil
}

func (s *Service) GetSimulationProfile() (viewmodel.SimulationProfilePage, error) {
	st, _, err := s.currentProject()
	if err != nil {
		return viewmodel.SimulationProfilePage{}, err
	}
	profile, err := st.Simulation.Load()
	if err != nil {
		return viewmodel.SimulationProfilePage{}, fmt.Errorf("读取仿写画像失败：%w", err)
	}
	return viewmodel.SimulationProfilePage{Available: profile != nil, Profile: profile}, nil
}

func (s *Service) GetRuleFiles() (viewmodel.RulesWorkspace, error) {
	st, projectRoot, err := s.currentProject()
	if err != nil {
		return viewmodel.RulesWorkspace{}, err
	}
	opts := studioRulesOptions(projectRoot)
	files, err := rules.ListRuleFiles(opts)
	if err != nil {
		return viewmodel.RulesWorkspace{}, fmt.Errorf("读取写作规则文件失败：%w", err)
	}
	result := viewmodel.RulesWorkspace{Global: []viewmodel.RuleFile{}, Project: []viewmodel.RuleFile{}, EffectiveNotice: "规则快照由 Core 在 Host 准备阶段生成；当前 Host 不会因编辑原文件而热重载。"}
	for _, file := range files {
		item := viewmodel.RuleFile{Name: file.Name, Scope: file.Scope.String(), Path: file.Path, SizeBytes: file.SizeBytes, ModifiedAt: file.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")}
		if file.Scope == rules.SourceGlobal {
			result.Global = append(result.Global, item)
		} else {
			result.Project = append(result.Project, item)
		}
	}
	snapshot, err := st.UserRules.Load()
	if err != nil {
		return viewmodel.RulesWorkspace{}, fmt.Errorf("读取有效规则快照失败：%w", err)
	}
	result.Effective = snapshot
	result.EffectiveAvailable = snapshot != nil
	return result, nil
}

func (s *Service) GetRule(scope, name string) (viewmodel.RuleFile, error) {
	_, projectRoot, err := s.currentProject()
	if err != nil {
		return viewmodel.RuleFile{}, err
	}
	opts := studioRulesOptions(projectRoot)
	kind, ok := viewmodelRuleScope(scope)
	if !ok {
		return viewmodel.RuleFile{}, fmt.Errorf("规则范围无效：%s", scope)
	}
	content, err := rules.ReadRuleFile(opts, kind, name)
	if err != nil {
		return viewmodel.RuleFile{}, err
	}
	return viewmodel.RuleFile{Name: name, Scope: scope, Content: content}, nil
}

func (s *Service) SaveRule(scope, name, content string) error {
	_, projectRoot, err := s.currentProject()
	if err != nil {
		return err
	}
	kind, ok := viewmodelRuleScope(scope)
	if !ok {
		return fmt.Errorf("规则范围无效：%s", scope)
	}
	return rules.SaveRuleFile(studioRulesOptions(projectRoot), kind, name, content)
}

func (s *Service) DeleteRule(scope, name string) error {
	_, projectRoot, err := s.currentProject()
	if err != nil {
		return err
	}
	kind, ok := viewmodelRuleScope(scope)
	if !ok {
		return fmt.Errorf("规则范围无效：%s", scope)
	}
	return rules.DeleteRuleFile(studioRulesOptions(projectRoot), kind, name)
}

func (s *Service) RenameRule(scope, oldName, newName string) error {
	_, projectRoot, err := s.currentProject()
	if err != nil {
		return err
	}
	kind, ok := viewmodelRuleScope(scope)
	if !ok {
		return fmt.Errorf("规则范围无效：%s", scope)
	}
	return rules.RenameRuleFile(studioRulesOptions(projectRoot), kind, oldName, newName)
}

func (s *Service) GetStyleState() (viewmodel.StyleState, error) {
	_, projectRoot, err := s.currentProject()
	if err != nil {
		return viewmodel.StyleState{}, err
	}
	outputDir := s.currentOutputDir()
	cfg, err := bootstrap.LoadConfigFromDir(projectRoot)
	if err != nil {
		return viewmodel.StyleState{}, fmt.Errorf("读取文风配置失败：%w", err)
	}
	cfg.FillDefaults()
	opts := assets.DefaultLoadOptions(outputDir)
	bundle := assets.Load(cfg.Style, opts)
	names := make([]string, 0, len(bundle.Styles))
	for name := range bundle.Styles {
		names = append(names, name)
	}
	sort.Strings(names)
	styleSource := assetSource(opts, filepath.ToSlash(filepath.Join("styles", cfg.Style+".md")), true)
	genreSource := ""
	genreReference := bundle.References.StyleReference
	if cfg.Style != "" {
		genreSource = assetSource(opts, filepath.ToSlash(filepath.Join("genres", cfg.Style, "style-references.md")), true)
	}
	voiceGlobal, err := assets.ReadOverride(opts, assets.OverrideGlobal, "voice.md")
	if err != nil {
		return viewmodel.StyleState{}, fmt.Errorf("读取全局 Voice 覆盖失败：%w", err)
	}
	voiceProject, err := assets.ReadOverride(opts, assets.OverrideProject, "voice.md")
	if err != nil {
		return viewmodel.StyleState{}, fmt.Errorf("读取项目 Voice 覆盖失败：%w", err)
	}
	antiGlobal, err := assets.ReadOverride(opts, assets.OverrideGlobal, "anti-ai-tone.md")
	if err != nil {
		return viewmodel.StyleState{}, fmt.Errorf("读取全局 anti-AI-tone 覆盖失败：%w", err)
	}
	antiProject, err := assets.ReadOverride(opts, assets.OverrideProject, "anti-ai-tone.md")
	if err != nil {
		return viewmodel.StyleState{}, fmt.Errorf("读取项目 anti-AI-tone 覆盖失败：%w", err)
	}
	return viewmodel.StyleState{
		SelectedStyle: cfg.Style, StyleNames: names, SelectedStyleText: bundle.Styles[cfg.Style], StyleSource: styleSource,
		EffectiveVoice: bundle.Voice, EffectiveVoiceSource: appendableSource(opts, "voice.md"), VoiceGlobal: voiceGlobal, VoiceProject: voiceProject,
		EffectiveAntiAITone: bundle.References.AntiAITone, EffectiveAntiSource: appendableSource(opts, "anti-ai-tone.md"), AntiAIToneGlobal: antiGlobal, AntiAIToneProject: antiProject,
		GenreReference: genreReference, GenreReferenceSource: genreSource,
		EffectiveNotice: "文风资产和 Style 选择在 Host 创建时加载；编辑器只修改当前范围的原始覆盖，保存后需停止并重新开始/恢复 Session 才会进入新 Host。",
	}, nil
}

func (s *Service) SaveStyleSelection(style string) error {
	_, projectRoot, err := s.currentProject()
	if err != nil {
		return err
	}
	return bootstrap.SaveStyleConfig(bootstrap.ProjectConfigPathFromDir(projectRoot), style)
}

func (s *Service) SaveStyleAsset(scope, name, content string) error {
	_, _, err := s.currentProject()
	if err != nil {
		return err
	}
	return assets.SaveOverride(assets.DefaultLoadOptions(s.currentOutputDir()), assets.OverrideScope(scope), name, content)
}

func (s *Service) DeleteStyleAsset(scope, name string) error {
	_, _, err := s.currentProject()
	if err != nil {
		return err
	}
	return assets.DeleteOverride(assets.DefaultLoadOptions(s.currentOutputDir()), assets.OverrideScope(scope), name)
}

func (s *Service) currentOutputDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == nil {
		return ""
	}
	return s.current.Dir()
}

func studioRulesOptions(projectRoot string) rules.LoadOptions {
	return rules.LoadOptions{HomeRulesDir: rules.DefaultHomeRulesDir(), ProjectRulesDir: rules.DefaultProjectRulesDir(projectRoot)}
}

func viewmodelRuleScope(scope string) (rules.SourceKind, bool) {
	if scope == "global" {
		return rules.SourceGlobal, true
	}
	if scope == "project" {
		return rules.SourceProject, true
	}
	return 0, false
}

func assetSource(opts assets.LoadOptions, rel string, selected bool) string {
	if selected && strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel)) == "" {
		return "Built-in"
	}
	if opts.BookStyleDir != "" {
		if _, err := os.Stat(filepath.Join(opts.BookStyleDir, filepath.FromSlash(rel))); err == nil {
			return "Project"
		}
	}
	if opts.HomeStyleDir != "" {
		if _, err := os.Stat(filepath.Join(opts.HomeStyleDir, filepath.FromSlash(rel))); err == nil {
			return "Global"
		}
	}
	return "Built-in"
}

func appendableSource(opts assets.LoadOptions, rel string) string {
	sources := []string{"Built-in"}
	if opts.HomeStyleDir != "" {
		if _, err := os.Stat(filepath.Join(opts.HomeStyleDir, filepath.FromSlash(rel))); err == nil {
			sources = append(sources, "Global")
		}
	}
	if opts.BookStyleDir != "" {
		if _, err := os.Stat(filepath.Join(opts.BookStyleDir, filepath.FromSlash(rel))); err == nil {
			sources = append(sources, "Project")
		}
	}
	return strings.Join(sources, " + ")
}
