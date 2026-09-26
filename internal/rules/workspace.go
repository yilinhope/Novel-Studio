package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RuleFile 是 rules 目录中一个用户文件的事实元数据。Content 仅在显式读取时填充。
type RuleFile struct {
	Name       string     `json:"name"`
	Scope      SourceKind `json:"scope"`
	Path       string     `json:"path"`
	SizeBytes  int64      `json:"size_bytes"`
	ModifiedAt time.Time  `json:"modified_at"`
}

func ruleDir(opts LoadOptions, scope SourceKind) (string, error) {
	var dir string
	switch scope {
	case SourceGlobal:
		dir = opts.HomeRulesDir
	case SourceProject:
		dir = opts.ProjectRulesDir
	default:
		return "", fmt.Errorf("未知规则来源: %d", scope)
	}
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("规则目录未配置: %s", scope.String())
	}
	return filepath.Clean(dir), nil
}

func validateRuleName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || filepath.Base(name) != name || strings.HasPrefix(name, ".") || !strings.EqualFold(filepath.Ext(name), ".md") {
		return fmt.Errorf("规则文件名必须是顶层 .md 文件: %q", name)
	}
	return nil
}

// ListRuleFiles 按 Core 的目录约定列出规则文件，不触发归一化或 Host。
func ListRuleFiles(opts LoadOptions) ([]RuleFile, error) {
	result := make([]RuleFile, 0)
	for _, scope := range []SourceKind{SourceGlobal, SourceProject} {
		dir, err := ruleDir(opts, scope)
		if err != nil {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
				continue
			}
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		for _, name := range names {
			info, err := os.Stat(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			result = append(result, RuleFile{Name: name, Scope: scope, Path: filepath.Join(dir, name), SizeBytes: info.Size(), ModifiedAt: info.ModTime()})
		}
	}
	return result, nil
}

// ReadRuleFile 读取指定规则文件的原文。
func ReadRuleFile(opts LoadOptions, scope SourceKind, name string) (string, error) {
	if err := validateRuleName(name); err != nil {
		return "", err
	}
	dir, err := ruleDir(opts, scope)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".rule-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o644); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

// SaveRuleFile 以原子替换方式写入既有规则语义，不创建额外 schema。
func SaveRuleFile(opts LoadOptions, scope SourceKind, name, content string) error {
	if err := validateRuleName(name); err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("规则内容不能为空")
	}
	dir, err := ruleDir(opts, scope)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, name), []byte(content))
}

func DeleteRuleFile(opts LoadOptions, scope SourceKind, name string) error {
	if err := validateRuleName(name); err != nil {
		return err
	}
	dir, err := ruleDir(opts, scope)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func RenameRuleFile(opts LoadOptions, scope SourceKind, oldName, newName string) error {
	if err := validateRuleName(oldName); err != nil {
		return err
	}
	if err := validateRuleName(newName); err != nil {
		return err
	}
	dir, err := ruleDir(opts, scope)
	if err != nil {
		return err
	}
	target := filepath.Join(dir, newName)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("目标规则文件已存在: %q", newName)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(filepath.Join(dir, oldName), filepath.Join(dir, newName))
}
