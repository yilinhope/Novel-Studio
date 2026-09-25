package v2

import (
	"strings"
	"testing"
)

func TestTextDiffCapsLargeMatrix(t *testing.T) {
	before := strings.Repeat("旧行\n", 2500)
	after := strings.Repeat("新行\n", 2500)
	diff := TextDiff(before, after)
	if len(diff) != 1 || diff[0].Kind != "summary" {
		t.Fatalf("超大正文应返回摘要 Diff，得到 %d 行 %+v", len(diff), diff)
	}
}
