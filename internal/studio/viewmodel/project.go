package viewmodel

// Project 是一次打开操作返回的只读快照。
type Project struct {
	ProjectRoot string   `json:"projectRoot"`
	OutputDir   string   `json:"outputDir"`
	Overview    Overview `json:"overview"`
	Tree        []Node   `json:"tree"`
}

type ChapterCommitConfirmation struct {
	Confirmed bool    `json:"confirmed"`
	Project   Project `json:"project"`
}

// ChapterSaveResult 明确区分正文保存与 Core 接纳/同步。
type ChapterSaveResult struct {
	Chapter  Chapter        `json:"chapter"`
	Revision RevisionStatus `json:"revision"`
}

// ChapterSyncResult 包含显式同步完成后的项目、当前章节与 Store 修订状态快照。
type ChapterSyncResult struct {
	Project  Project        `json:"project"`
	Chapter  *Chapter       `json:"chapter,omitempty"`
	Revision RevisionStatus `json:"revision"`
}

type Overview struct {
	Title             string `json:"title"`
	Synopsis          string `json:"synopsis"`
	Path              string `json:"path"`
	Phase             string `json:"phase"`
	Flow              string `json:"flow"`
	CurrentChapter    int    `json:"currentChapter"`
	CompletedChapters int    `json:"completedChapters"`
	PlannedChapters   int    `json:"plannedChapters"`
	WordCount         int    `json:"wordCount"`
	CurrentVolume     int    `json:"currentVolume"`
	CurrentArc        int    `json:"currentArc"`
}

type Node struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Chapter  int    `json:"chapter"`
	Children []Node `json:"children"`
}

type Chapter struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	WordCount  int    `json:"wordCount"`
	HasContent bool   `json:"hasContent"`
	CanEdit    bool   `json:"canEdit"`
}
