package viewmodel

// Project 是一次打开操作返回的只读快照。
type Project struct {
	Overview Overview `json:"overview"`
	Tree     []Node   `json:"tree"`
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
}
