package host

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CoCreateRecovery 是从 Core 现有 JSONL 会话日志读取的只读恢复视图。
// 它不新增持久化格式；仅最后一轮完整/中断输入可恢复。
type CoCreateRecovery struct {
	Exists      bool
	Interrupted bool
	Mode        string
	History     []CoCreateMessage
	Draft       string
	Ready       bool
	Suggestions []string
	Error       string
}

// ReadCoCreateRecovery 读取 Core 已有的 meta/sessions/cocreate.jsonl 最后一条记录。
func ReadCoCreateRecovery(outputDir string) (CoCreateRecovery, error) {
	path := filepath.Join(strings.TrimSpace(outputDir), "meta", "sessions", "cocreate.jsonl")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CoCreateRecovery{}, nil
		}
		return CoCreateRecovery{}, fmt.Errorf("读取共创恢复记录失败：%w", err)
	}
	defer file.Close()

	var latest struct {
		InputHistory []CoCreateMessage `json:"input_history"`
		ParsedReply  string            `json:"parsed_reply"`
		ParsedDraft  string            `json:"parsed_draft"`
		ParsedReady  bool              `json:"parsed_ready"`
		ParsedSugs   []string          `json:"parsed_sugs"`
		Error        string            `json:"error"`
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	found := false
	for scanner.Scan() {
		line := scanner.Bytes()
		var record struct {
			InputHistory []CoCreateMessage `json:"input_history"`
			ParsedReply  string            `json:"parsed_reply"`
			ParsedDraft  string            `json:"parsed_draft"`
			ParsedReady  bool              `json:"parsed_ready"`
			ParsedSugs   []string          `json:"parsed_sugs"`
			Error        string            `json:"error"`
		}
		if err := json.Unmarshal(line, &record); err != nil {
			continue // 忽略文件尾部可能存在的未完整 JSONL 行。
		}
		latest, found = record, true
	}
	if err := scanner.Err(); err != nil {
		return CoCreateRecovery{}, fmt.Errorf("读取共创恢复记录失败：%w", err)
	}
	if !found {
		return CoCreateRecovery{}, nil
	}
	history := append([]CoCreateMessage(nil), latest.InputHistory...)
	if strings.TrimSpace(latest.ParsedReply) != "" {
		history = append(history, CoCreateMessage{Role: "assistant", Content: latest.ParsedReply})
	}
	mode := "cold"
	if len(history) > 0 && strings.TrimSpace(history[0].Content) == "我先暂停一下，想和你一起规划接下来的走向。" {
		mode = "stage"
	}
	return CoCreateRecovery{Exists: true, Interrupted: strings.TrimSpace(latest.Error) != "", Mode: mode, History: history, Draft: latest.ParsedDraft, Ready: latest.ParsedReady, Suggestions: append([]string(nil), latest.ParsedSugs...), Error: latest.Error}, nil
}
