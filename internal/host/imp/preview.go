package imp

import (
	"os"

	"github.com/voocel/ainovel-cli/internal/store"
)

// LoadSegmentation 读取已经落盘的切分工件，供 TUI/Studio 展示人工确认预览。
// 它不推进管线、不写入确认工件，也不把未完成的切分猜成已完成。
func LoadSegmentation(st *store.Store) (*Segmentation, error) {
	if st == nil {
		return nil, os.ErrInvalid
	}
	w := OpenWorkspace(st.Dir())
	artifact, err := readArtifact[Segmentation](w, fileSegmentation)
	if err != nil {
		return nil, err
	}
	return &artifact.Payload, nil
}
