package viewmodel

import (
	"time"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/rules"
)

type SimulationSource struct {
	RelativePath string `json:"relativePath"`
	SHA256       string `json:"sha256"`
	Fingerprint  string `json:"fingerprint"`
	SizeBytes    int64  `json:"sizeBytes"`
	ModTime      string `json:"modTime,omitempty"`
	AnalyzedAt   string `json:"analyzedAt,omitempty"`
	Changed      bool   `json:"changed"`
}

type SimulationProfilePage struct {
	ReadIdentity
	Available bool                      `json:"available"`
	Profile   *domain.SimulationProfile `json:"profile,omitempty"`
}

type SimulationSourcesPage struct {
	ReadIdentity
	SourceDir string             `json:"sourceDir"`
	Items     []SimulationSource `json:"items"`
}

type SimulationEvent struct {
	ProjectID  string    `json:"projectId"`
	Generation uint64    `json:"generation"`
	RequestID  string    `json:"requestId,omitempty"`
	State      string    `json:"state"`
	Stage      string    `json:"stage"`
	Current    int       `json:"current"`
	Total      int       `json:"total"`
	Message    string    `json:"message"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type SimulationImportRequest struct {
	ProjectID  string `json:"projectId,omitempty"`
	Generation uint64 `json:"generation,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	Path       string `json:"path"`
}

type RuleFile struct {
	Name       string `json:"name"`
	Scope      string `json:"scope"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"sizeBytes"`
	ModifiedAt string `json:"modifiedAt"`
	Content    string `json:"content,omitempty"`
}

type RulesWorkspace struct {
	ReadIdentity
	Global             []RuleFile      `json:"global"`
	Project            []RuleFile      `json:"project"`
	Effective          *rules.Snapshot `json:"effective,omitempty"`
	EffectiveAvailable bool            `json:"effectiveAvailable"`
	EffectiveNotice    string          `json:"effectiveNotice,omitempty"`
}

type RuleMutationRequest struct {
	ProjectID  string `json:"projectId,omitempty"`
	Generation uint64 `json:"generation,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	Scope      string `json:"scope"`
	Name       string `json:"name"`
	NewName    string `json:"newName,omitempty"`
	Content    string `json:"content,omitempty"`
}

type StyleState struct {
	ReadIdentity
	SelectedStyle        string   `json:"selectedStyle"`
	StyleNames           []string `json:"styleNames"`
	SelectedStyleText    string   `json:"selectedStyleText,omitempty"`
	StyleSource          string   `json:"styleSource"`
	EffectiveVoice       string   `json:"effectiveVoice"`
	EffectiveVoiceSource string   `json:"effectiveVoiceSource"`
	VoiceGlobal          string   `json:"voiceGlobal"`
	VoiceProject         string   `json:"voiceProject"`
	EffectiveAntiAITone  string   `json:"effectiveAntiAiTone"`
	EffectiveAntiSource  string   `json:"effectiveAntiAiToneSource"`
	AntiAIToneGlobal     string   `json:"antiAiToneGlobal"`
	AntiAIToneProject    string   `json:"antiAiToneProject"`
	GenreReference       string   `json:"genreReference,omitempty"`
	GenreReferenceSource string   `json:"genreReferenceSource,omitempty"`
	EffectiveNotice      string   `json:"effectiveNotice"`
}

type StyleMutationRequest struct {
	ProjectID  string `json:"projectId,omitempty"`
	Generation uint64 `json:"generation,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Name       string `json:"name,omitempty"`
	Content    string `json:"content,omitempty"`
}

func ruleScope(value string) (rules.SourceKind, bool) {
	switch value {
	case "global":
		return rules.SourceGlobal, true
	case "project":
		return rules.SourceProject, true
	default:
		return 0, false
	}
}
