export interface Overview {
  title: string; synopsis: string; path: string; phase: string; flow: string
  currentChapter: number; completedChapters: number; plannedChapters: number
  wordCount: number; currentVolume: number; currentArc: number
}
export interface TreeNode { id: string; kind: string; title: string; chapter: number; children: TreeNode[] }
export interface Project { projectRoot: string; outputDir: string; overview: Overview; tree: TreeNode[] }
export interface ChapterCommitConfirmation { confirmed: boolean; project: Project }
export interface Chapter { number: number; title: string; content: string; wordCount: number; hasContent: boolean; canEdit: boolean }
export interface ReviewIssue { type: string; severity: string; description: string; evidence?: string; suggestion?: string; chapters?: number[]; requires_change: boolean }
export interface ReviewDimension { dimension: string; score: number; verdict?: string; comment?: string }
export interface ReviewEntry {
  chapter: number; scope: string; issues?: ReviewIssue[] | null; dimensions?: ReviewDimension[]
  contract_status?: string; contract_misses?: string[]; contract_notes?: string
  verdict: string; summary: string; affected_chapters?: number[]
}
export interface ReviewCenter {
  projectId: string; reviews: ReviewEntry[]; currentReviews: ReviewEntry[]; requiresAdvancePermit: boolean
  canAdvance: boolean
  nextChapter: number; hasCurrentReview: boolean; advanceMode: string
  advancePermitChapter: number; advanceHoldReason?: string; advanceBlockedReason?: string
}
export type ProposalStatus = 'Draft' | 'Ready' | 'Accepted' | 'Rejected' | 'Stale' | 'AppliedWorkingCopy' | 'SyncPending' | 'Synced' | 'Failed'
export interface ProposalChange {
  resourceType: string; resourceId: string; chapter: number; baseHash: string; before: string; after: string; afterHash: string; changeType: string
}
export interface EvidenceRef {
  resourceType: string; resourceId: string; chapter: number; revision: string; contentHash: string
  startOffset: number; endOffset: number; quotePreview: string
}
export interface Proposal {
  id: string; projectId: string; title: string; summary?: string; rationale?: string; source: string; sourceKey?: string
  status: ProposalStatus; changes: ProposalChange[]; evidence: EvidenceRef[]; baseRevision?: string
  operationId?: string; error?: string; createdAt: string; updatedAt: string
}
export interface ProposalChangeInput {
  resourceType?: string; resourceId?: string; chapter: number; baseHash?: string; before?: string; after: string; changeType?: string
}
export interface CreateProposalRequest {
  projectId?: string; title: string; summary?: string; rationale?: string; source?: string; sourceKey?: string
  changes: ProposalChangeInput[]; evidence?: Partial<EvidenceRef>[]
}
export interface DiffLine { kind: 'context' | 'removed' | 'added'; text: string }
export interface VersionSnapshot {
  id: string; projectId: string; resourceType: string; resourceId: string; chapter: number; contentHash: string
  content: string; source: string; parentId?: string; createdAt: string
}
export interface ApplyResult { proposalId: string; status: ProposalStatus; chapters: number[]; message?: string }
export type RevisionState = 'unknown' | 'synced' | 'saved_unsynced' | 'recovery_pending' | 'error'
export interface UnsyncedChapter { chapter: number; acceptedHash: string; currentHash: string }
export interface RevisionStatus {
  projectId: string; state: RevisionState; hasUnsynced: boolean; chapters: UnsyncedChapter[]
  pendingStage?: string; checkedAt?: string; error?: string
}
export interface ChapterSaveResult { chapter: Chapter; revision: RevisionStatus }
export interface ChapterSyncResult { project?: Project; chapter?: Chapter; revision: RevisionStatus; refreshWarning?: string }
export interface ControlResult { project?: Project; review?: ReviewCenter; runtime: Runtime; revision: RevisionStatus; refreshWarning?: string }
export type RuntimeState = 'idle' | 'running' | 'pausing' | 'paused' | 'stopping' | 'stopped' | 'waiting_review' | 'waiting_sync' | 'completed' | 'error'
export interface RuntimeAgent { name: string; state: string; tool?: string; summary?: string }
export interface Runtime {
  projectId: string; generation: number; state: RuntimeState; phase: string; flow: string
  requiresAdvancePermit?: boolean; canAdvance?: boolean; advanceBlockedReason?: string
  nextChapter?: number; hasCurrentReview?: boolean
  pendingSteer?: string
  agent?: string; chapter?: number; step?: string; elapsedSeconds: number
  inputTokens: number; outputTokens: number; projectInputTokens: number; projectOutputTokens: number
  runCostUsd: number; projectCostUsd: number; error?: string; agents: RuntimeAgent[]; updatedAt: string
}
export interface RuntimeLog {
  ID: string; Time: string; FinishedAt: string; Failed: boolean; Category: string
  Agent: string; Summary: string; Detail: string; Tool: string; Chapter?: number; Level: string; Depth: number
}
export interface StudioEngineEvent {
  projectId: string; generation: number; runId: number; sequence: number; timestamp: string
  type: 'runtime' | 'log'; log?: RuntimeLog; runtime?: Runtime
}
export interface OperationAck { projectId: string; generation: number; operation: string; requestId?: string }
export interface CreateProjectRequest { projectRoot: string; prompt: string; mode: 'quick' | 'outline'; requestId?: string }
export interface CreateEvent { projectId: string; generation: number; operation: string; requestId?: string; state: string; message?: string; error?: string; project?: Project; runtime?: Runtime }
export interface CoCreateMessage { role: 'user' | 'assistant'; content: string }
export interface CoCreateStart extends OperationAck { mode: 'cold' | 'stage' }
export interface CoCreateEvent { projectId: string; generation: number; requestId?: string; state: string; kind?: string; text?: string; reply?: string; draft?: string; ready: boolean; suggestions?: string[]; history?: CoCreateMessage[]; error?: string }
export interface CoCreateRecovery { projectId: string; generation?: number; exists: boolean; interrupted: boolean; mode: string; history?: CoCreateMessage[]; draft?: string; ready: boolean; suggestions?: string[]; error?: string }
export interface ImportOptions { projectRoot: string; sourcePath: string; autoConfirm: boolean; acceptSegmentation: boolean; storyResolution: string; continueAfter: boolean; guidance: string; requestId?: string }
export interface ImportChapter { number: number; title: string; startByte: number; endByte: number; uncertain: boolean }
export interface ImportStatus { projectId: string; generation: number; requestId?: string; active: boolean; stage: string; current: number; total: number; message: string; level?: string; key?: string; retryAt?: string; error?: string; continued: boolean; recoveryHint?: string; chapters?: ImportChapter[]; uncertain?: number[]; notes?: string[] }
export interface ExportOptions { format: 'txt' | 'epub'; outPath: string; from: number; to: number; overwrite: boolean }
export interface ExportResult { path: string; chapters: number; bytes: number; skipped?: number[] }
export interface ConfigModel { name: string; contextWindow?: number; jsonSchema?: boolean }
export interface ConfigProvider { name: string; type: string; api: string; baseUrl: string; streamIdleTimeout?: string; models: ConfigModel[]; hasApiKey: boolean; apiKeyHint?: string; requiresApiKey: boolean }
export interface ConfigModelRef { provider: string; model: string }
export interface ConfigRole { provider: string; model: string; reasoningEffort?: string; fallbacks?: ConfigModelRef[] }
export interface BudgetConfig { bookUsd: number; warnRatio: number; hardStop: boolean }
export interface ConfigSnapshot { projectRoot: string; configPath: string; provider: string; model: string; reasoningEffort?: string; style?: string; contextWindow?: number; providers: ConfigProvider[]; roles: Record<string, ConfigRole>; budget: BudgetConfig; notify: { enabled?: boolean; command?: string; events?: string[] } }
export interface ProviderDraft { provider: string; type: string; api: string; baseUrl: string; models: ConfigModel[]; renames?: { from: string; to: string }[]; apiKeyAction: 'keep' | 'replace' | 'clear'; apiKey?: string }
export interface ModelSelection { role: string; provider: string; model: string }
export interface RoleThinking { role: string; level: string }
export interface UsageTotals { input: number; output: number; cacheRead: number; cacheWrite: number; costUsd: number; savedUsd: number; cacheCapable: boolean; cacheBreaks: number }
export interface AgentUsage { role?: string; model?: string; input: number; output: number; cacheRead: number; cacheWrite: number; costUsd: number; savedUsd: number; cacheCapable: boolean }
export interface UsageSnapshot { projectId: string; updatedAt: string; overall: UsageTotals; perAgent: AgentUsage[]; perModel: AgentUsage[]; missingUsage: number; budget: BudgetConfig }
export interface StudioBridge {
  SelectProjectDirectory(): Promise<string>
  OpenProject(path: string): Promise<Project>
  GetProjectOverview(): Promise<Overview>
  GetProjectTree(): Promise<TreeNode[]>
  GetChapter(number: number): Promise<Chapter>
  GetReviewCenter?(): Promise<ReviewCenter>
  ListProposals?(): Promise<Proposal[]>
  GetProposal?(id: string): Promise<Proposal>
  CreateProposal?(request: CreateProposalRequest): Promise<Proposal>
  CreateProposalFromReviewIssue?(chapter: number, scope: string, issueIndex: number, after: string): Promise<Proposal>
  AcceptProposal?(id: string): Promise<Proposal>
  RejectProposal?(id: string): Promise<Proposal>
  ApplyProposal?(id: string): Promise<ApplyResult>
  GetProposalDiff?(id: string, changeIndex: number): Promise<DiffLine[]>
  GetVersionHistory?(chapter: number): Promise<VersionSnapshot[]>
  GetVersion?(id: string): Promise<VersionSnapshot>
  RestoreVersion?(id: string): Promise<VersionSnapshot>
  SaveChapter(number: number, content: string): Promise<ChapterSaveResult>
  SyncChapterRevisions?(chapter: number): Promise<ChapterSyncResult>
  GetRevisionStatus(): Promise<RevisionStatus>
  CheckChapterRevisions(): Promise<RevisionStatus>
  GetRuntimeState?(): Promise<Runtime>
  ResumeWriting?(): Promise<Runtime>
  PauseWriting?(): Promise<Runtime>
  StopWriting?(): Promise<Runtime>
  SetAdvanceMode?(mode: 'auto' | 'review'): Promise<ControlResult>
  AdvanceOneChapter?(): Promise<ControlResult>
  SubmitSteer?(text: string): Promise<ControlResult>
  ConfirmChapterCommit?(chapter: number, startedAt: string): Promise<ChapterCommitConfirmation>
  StartQuickStart?(request: CreateProjectRequest): Promise<OperationAck>
  PreviewOutline?(path: string): Promise<string>
  StartCoCreate?(projectRoot: string, initial: string, stage: boolean, requestId?: string): Promise<CoCreateStart>
  SendCoCreate?(projectRoot: string, outputDir: string, stage: boolean, history: CoCreateMessage[]): Promise<void>
  CompleteCoCreate?(stage: boolean, draft: string): Promise<Runtime>
  CancelCoCreate?(stage: boolean): Promise<void>
  GetCoCreateRecovery?(): Promise<CoCreateRecovery>
  ResumeCoCreate?(stage: boolean, history: CoCreateMessage[], requestId?: string): Promise<CoCreateStart>
  StartImport?(options: ImportOptions): Promise<OperationAck>
  CancelImport?(): Promise<void>
  GetImportStatus?(): Promise<ImportStatus>
  ExportProject?(options: ExportOptions): Promise<ExportResult>
  GetConfig?(): Promise<ConfigSnapshot>
  SaveProviderConfig?(draft: ProviderDraft): Promise<ConfigSnapshot>
  TestModelConnection?(draft: ProviderDraft, model: string): Promise<void>
  SwitchModel?(selection: ModelSelection): Promise<ConfigSnapshot>
  SetRoleThinking?(setting: RoleThinking): Promise<ConfigSnapshot>
  SaveBudgetConfig?(budget: BudgetConfig): Promise<ConfigSnapshot>
  GetUsage?(): Promise<UsageSnapshot>
}
declare global {
  interface Window {
    go?: { bridge: { App: StudioBridge } }
    runtime?: { EventsOn(eventName: string, callback: (payload: unknown) => void): () => void }
  }
}
