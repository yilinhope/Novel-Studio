package viewmodel

// ReadRequest 是 Story/Continuity 只读请求的身份与分页参数。
// ProjectID 和 Generation 由当前打开项目提供，Bridge 只接受当前作用域；
// RequestID/Sequence 原样回显，供前端拒绝项目切换后的晚到响应。
type ReadRequest struct {
	ProjectID  string `json:"projectId,omitempty"`
	Generation uint64 `json:"generation,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	Sequence   uint64 `json:"sequence,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type ReadIdentity struct {
	ProjectID   string `json:"projectId"`
	Generation  uint64 `json:"generation"`
	ProjectRoot string `json:"projectRoot"`
	OutputDir   string `json:"outputDir"`
	RequestID   string `json:"requestId,omitempty"`
	Sequence    uint64 `json:"sequence,omitempty"`
}

type PageInfo struct {
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	Total   int  `json:"total"`
	HasMore bool `json:"hasMore"`
}

type BookMetadata struct {
	Title    string `json:"title"`
	Synopsis string `json:"synopsis"`
}

type StoryPremise struct {
	ReadIdentity
	Book             BookMetadata `json:"book"`
	Premise          string       `json:"premise"`
	BookAvailable    bool         `json:"bookAvailable"`
	PremiseAvailable bool         `json:"premiseAvailable"`
}

type Character struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Role        string   `json:"role"`
	Description string   `json:"description"`
	Arc         string   `json:"arc"`
	Traits      []string `json:"traits,omitempty"`
	Tier        string   `json:"tier,omitempty"`
}

type CharacterPage struct {
	ReadIdentity
	PageInfo
	Items []Character `json:"items"`
}

type WorldRule struct {
	Category string `json:"category"`
	Rule     string `json:"rule"`
	Boundary string `json:"boundary"`
}

type WorldRulePage struct {
	ReadIdentity
	PageInfo
	Items []WorldRule `json:"items"`
}

type OutlineEntry struct {
	Chapter   int      `json:"chapter"`
	Title     string   `json:"title"`
	CoreEvent string   `json:"coreEvent"`
	Hook      string   `json:"hook"`
	Scenes    []string `json:"scenes,omitempty"`
}

type OutlinePage struct {
	ReadIdentity
	PageInfo
	Items []OutlineEntry `json:"items"`
}

type LayeredArc struct {
	Index             int    `json:"index"`
	Title             string `json:"title"`
	Goal              string `json:"goal"`
	EstimatedChapters int    `json:"estimatedChapters,omitempty"`
	ChapterCount      int    `json:"chapterCount"`
}

type LayeredVolume struct {
	Index int          `json:"index"`
	Title string       `json:"title"`
	Theme string       `json:"theme"`
	Final bool         `json:"final"`
	Arcs  []LayeredArc `json:"arcs"`
}

type LayeredOutlinePage struct {
	ReadIdentity
	PageInfo
	Items []LayeredVolume `json:"items"`
}

type LayeredChapterPage struct {
	ReadIdentity
	Volume int `json:"volume"`
	Arc    int `json:"arc"`
	PageInfo
	Items []OutlineEntry `json:"items"`
}

type StoryCompass struct {
	ReadIdentity
	EndingDirection string   `json:"endingDirection"`
	OpenThreads     []string `json:"openThreads,omitempty"`
	EstimatedScale  string   `json:"estimatedScale,omitempty"`
	LastUpdated     int      `json:"lastUpdated,omitempty"`
	Available       bool     `json:"available"`
}

type ChapterSummary struct {
	Chapter    int      `json:"chapter"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Characters []string `json:"characters,omitempty"`
	KeyEvents  []string `json:"keyEvents,omitempty"`
}

type ArcSummary struct {
	Volume    int      `json:"volume"`
	Arc       int      `json:"arc"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	KeyEvents []string `json:"keyEvents,omitempty"`
}

type VolumeSummary struct {
	Volume    int      `json:"volume"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	KeyEvents []string `json:"keyEvents,omitempty"`
}

type StorySummaryPage struct {
	ReadIdentity
	Scope string `json:"scope"`
	PageInfo
	Chapters []ChapterSummary `json:"chapters,omitempty"`
	Arcs     []ArcSummary     `json:"arcs,omitempty"`
	Volumes  []VolumeSummary  `json:"volumes,omitempty"`
}
