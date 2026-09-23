export interface Overview {
  title: string; synopsis: string; path: string; phase: string; flow: string
  currentChapter: number; completedChapters: number; plannedChapters: number
  wordCount: number; currentVolume: number; currentArc: number
}
export interface TreeNode { id: string; kind: string; title: string; chapter: number; children: TreeNode[] }
export interface Project { projectRoot: string; outputDir: string; overview: Overview; tree: TreeNode[] }
export interface ChapterCommitConfirmation { confirmed: boolean; project: Project }
export interface Chapter { number: number; title: string; content: string; wordCount: number; hasContent: boolean; canEdit: boolean }
export type RevisionState = 'unknown' | 'synced' | 'saved_unsynced' | 'recovery_pending' | 'error'
export interface UnsyncedChapter { chapter: number; acceptedHash: string; currentHash: string }
export interface RevisionStatus {
  projectId: string; state: RevisionState; hasUnsynced: boolean; chapters: UnsyncedChapter[]
  pendingStage?: string; checkedAt?: string; error?: string
}
export interface ChapterSaveResult { chapter: Chapter; revision: RevisionStatus }
export interface ChapterSyncResult { project?: Project; chapter?: Chapter; revision: RevisionStatus; refreshWarning?: string }
export type RuntimeState = 'idle' | 'running' | 'pausing' | 'paused' | 'stopping' | 'stopped' | 'waiting_review' | 'waiting_sync' | 'completed' | 'error'
export interface RuntimeAgent { name: string; state: string; tool?: string; summary?: string }
export interface Runtime {
  projectId: string; generation: number; state: RuntimeState; phase: string; flow: string
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
export interface StudioBridge {
  SelectProjectDirectory(): Promise<string>
  OpenProject(path: string): Promise<Project>
  GetProjectOverview(): Promise<Overview>
  GetProjectTree(): Promise<TreeNode[]>
  GetChapter(number: number): Promise<Chapter>
  SaveChapter(number: number, content: string): Promise<ChapterSaveResult>
  SyncChapterRevisions?(chapter: number): Promise<ChapterSyncResult>
  GetRevisionStatus(): Promise<RevisionStatus>
  CheckChapterRevisions(): Promise<RevisionStatus>
  GetRuntimeState?(): Promise<Runtime>
  ResumeWriting?(): Promise<Runtime>
  PauseWriting?(): Promise<Runtime>
  StopWriting?(): Promise<Runtime>
  ConfirmChapterCommit?(chapter: number, startedAt: string): Promise<ChapterCommitConfirmation>
}
declare global {
  interface Window {
    go?: { bridge: { App: StudioBridge } }
    runtime?: { EventsOn(eventName: string, callback: (payload: StudioEngineEvent) => void): () => void }
  }
}
