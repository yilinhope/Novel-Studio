package v2

import (
	"fmt"
	"strings"
)

type DiffLine struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

const maxDiffCells = 4_000_000

// TextDiff 返回确定性的行级 unified 视图；它不调用模型，也不改变正文。
func TextDiff(before, after string) []DiffLine {
	left := strings.Split(strings.ReplaceAll(before, "\r\n", "\n"), "\n")
	right := strings.Split(strings.ReplaceAll(after, "\r\n", "\n"), "\n")
	m := len(left)
	n := len(right)
	if m > 0 && n > 0 && m > maxDiffCells/n {
		return []DiffLine{{Kind: "summary", Text: fmt.Sprintf("正文过长，已省略逐行 Diff（修改前 %d 行，修改后 %d 行）", m, n)}}
	}
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if left[i] == right[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	result := make([]DiffLine, 0, m+n)
	i, j := 0, 0
	for i < m && j < n {
		switch {
		case left[i] == right[j]:
			result = append(result, DiffLine{Kind: "context", Text: left[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			result = append(result, DiffLine{Kind: "removed", Text: left[i]})
			i++
		default:
			result = append(result, DiffLine{Kind: "added", Text: right[j]})
			j++
		}
	}
	for ; i < m; i++ {
		result = append(result, DiffLine{Kind: "removed", Text: left[i]})
	}
	for ; j < n; j++ {
		result = append(result, DiffLine{Kind: "added", Text: right[j]})
	}
	return result
}
