package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OverrideScope string

const (
	OverrideGlobal  OverrideScope = "global"
	OverrideProject OverrideScope = "project"
)

func overrideDir(opts LoadOptions, scope OverrideScope) (string, error) {
	switch scope {
	case OverrideGlobal:
		if strings.TrimSpace(opts.HomeStyleDir) == "" {
			return "", fmt.Errorf("全局文风目录未配置")
		}
		return opts.HomeStyleDir, nil
	case OverrideProject:
		if strings.TrimSpace(opts.BookStyleDir) == "" {
			return "", fmt.Errorf("项目文风目录未配置")
		}
		return opts.BookStyleDir, nil
	default:
		return "", fmt.Errorf("未知文风覆盖范围: %q", scope)
	}
}

func validateOverrideName(name string) error {
	name = filepath.ToSlash(strings.TrimSpace(name))
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	if name == "" || clean != name || name == ".." || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") {
		return fmt.Errorf("文风资产路径无效: %q", name)
	}
	if name == "voice.md" || name == "anti-ai-tone.md" {
		return nil
	}
	if strings.HasPrefix(name, "styles/") || strings.HasPrefix(name, "genres/") {
		return nil
	}
	return fmt.Errorf("不支持的文风资产: %q", name)
}

// SaveOverride 原子写入 voice/anti-ai-tone/style reference 等既有资产路径。
func SaveOverride(opts LoadOptions, scope OverrideScope, name, content string) error {
	name = filepath.ToSlash(strings.TrimSpace(name))
	if err := validateOverrideName(name); err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("文风资产内容不能为空")
	}
	dir, err := overrideDir(opts, scope)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, filepath.FromSlash(name)), []byte(content))
}

func DeleteOverride(opts LoadOptions, scope OverrideScope, name string) error {
	name = filepath.ToSlash(strings.TrimSpace(name))
	if err := validateOverrideName(name); err != nil {
		return err
	}
	dir, err := overrideDir(opts, scope)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, filepath.FromSlash(name)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".asset-*.tmp")
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
