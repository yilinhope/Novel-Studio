package host

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToolChapterOnlyProjectsCommitChapterArgument(t *testing.T) {
	if got := toolChapter("commit_chapter", json.RawMessage(`{"chapter":17}`)); got != 17 {
		t.Fatalf("章节号 = %d，期望 17", got)
	}
	if got := toolChapter("draft_chapter", json.RawMessage(`{"chapter":17}`)); got != 0 {
		t.Fatalf("非提交工具不应输出章节提交号: %d", got)
	}
}

func TestEventChapterFieldIsBackwardCompatibleAndOptional(t *testing.T) {
	withoutChapter, err := json.Marshal(Event{Tool: "commit_chapter"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(withoutChapter), `"Chapter"`) {
		t.Fatalf("缺省章节字段应省略: %s", withoutChapter)
	}
	withChapter, err := json.Marshal(Event{Tool: "commit_chapter", Chapter: 23})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(withChapter, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["Chapter"] != float64(23) {
		t.Fatalf("结构化章节号没有按兼容字段名编码: %s", withChapter)
	}
}
