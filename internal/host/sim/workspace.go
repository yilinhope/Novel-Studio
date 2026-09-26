package sim

import (
	"github.com/voocel/ainovel-cli/internal/domain"
)

// ListSources 只扫描指定语料目录并返回 Core 使用的源文件清单；不会创建 Host、Store 或调用模型。
func ListSources(sourceDir string) ([]domain.SimulationSource, error) {
	sources, err := scanSources(sourceDir)
	if err != nil {
		return nil, err
	}
	result := make([]domain.SimulationSource, 0, len(sources))
	for _, source := range sources {
		result = append(result, source.SimulationSource)
	}
	return result, nil
}
