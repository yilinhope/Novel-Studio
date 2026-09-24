package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCoCreateRecoveryUsesExistingJSONLAndDoesNotCreateFile(t *testing.T) {
	root := t.TempDir()
	recovery, err := ReadCoCreateRecovery(root)
	if err != nil {
		t.Fatalf("无会话日志时不应失败: %v", err)
	}
	if recovery.Exists {
		t.Fatal("无会话日志不应投影为已有恢复会话")
	}
	path := filepath.Join(root, "meta", "sessions", "cocreate.jsonl")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("只读恢复读取不应创建日志文件: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"input_history":[{"role":"user","content":"我想写悬疑"}],"parsed_reply":"先确定案件地点","parsed_draft":"## 主题\n- 悬疑","parsed_ready":false,"error":"provider interrupted"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	recovery, err = ReadCoCreateRecovery(root)
	if err != nil {
		t.Fatalf("读取已有日志失败: %v", err)
	}
	if !recovery.Exists || !recovery.Interrupted || recovery.Mode != "cold" {
		t.Fatalf("恢复事实不正确: %+v", recovery)
	}
	if len(recovery.History) != 2 || recovery.History[1].Role != "assistant" {
		t.Fatalf("应把最后一轮 Core 回复接回历史: %+v", recovery.History)
	}
	if recovery.Draft == "" || recovery.Ready {
		t.Fatalf("草稿/ready 未按日志投影: %+v", recovery)
	}
}

func TestReadCoCreateRecoveryIgnoresIncompleteTail(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "meta", "sessions", "cocreate.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := "{\"input_history\":[],\"parsed_reply\":\"完整记录\"}\n{\"input_history\":"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	recovery, err := ReadCoCreateRecovery(root)
	if err != nil {
		t.Fatal(err)
	}
	if !recovery.Exists || len(recovery.History) != 1 || recovery.History[0].Content != "完整记录" {
		t.Fatalf("应保留最后一条完整 JSONL: %+v", recovery)
	}
}
