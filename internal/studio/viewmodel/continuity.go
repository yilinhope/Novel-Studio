package viewmodel

type TimelineEvent struct {
	Chapter    int      `json:"chapter"`
	Time       string   `json:"time"`
	Event      string   `json:"event"`
	Characters []string `json:"characters,omitempty"`
}

type TimelinePage struct {
	ReadIdentity
	PageInfo
	Items []TimelineEvent `json:"items"`
}

type ForeshadowEntry struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	PlantedAt   int    `json:"plantedAt"`
	Status      string `json:"status"`
	ResolvedAt  int    `json:"resolvedAt,omitempty"`
}

type ForeshadowPage struct {
	ReadIdentity
	PageInfo
	Items []ForeshadowEntry `json:"items"`
}

type RelationshipEntry struct {
	CharacterA string `json:"characterA"`
	CharacterB string `json:"characterB"`
	Relation   string `json:"relation"`
	Chapter    int    `json:"chapter"`
}

type RelationshipPage struct {
	ReadIdentity
	PageInfo
	Items []RelationshipEntry `json:"items"`
}

type StateChange struct {
	Chapter  int    `json:"chapter"`
	Entity   string `json:"entity"`
	Field    string `json:"field"`
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue"`
	Reason   string `json:"reason,omitempty"`
}

type StateChangePage struct {
	ReadIdentity
	PageInfo
	Items []StateChange `json:"items"`
}

type CharacterSnapshot struct {
	Volume     int    `json:"volume"`
	Arc        int    `json:"arc"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Power      string `json:"power,omitempty"`
	Motivation string `json:"motivation"`
	Relations  string `json:"relations,omitempty"`
}

type SnapshotPage struct {
	ReadIdentity
	PageInfo
	Items []CharacterSnapshot `json:"items"`
}

type CastEntry struct {
	Name               string `json:"name"`
	BriefRole          string `json:"briefRole,omitempty"`
	FirstSeenChapter   int    `json:"firstSeenChapter"`
	LastSeenChapter    int    `json:"lastSeenChapter"`
	AppearanceCount    int    `json:"appearanceCount"`
	AppearanceChapters []int  `json:"appearanceChapters,omitempty"`
}

type CastPage struct {
	ReadIdentity
	PageInfo
	Items []CastEntry `json:"items"`
}
