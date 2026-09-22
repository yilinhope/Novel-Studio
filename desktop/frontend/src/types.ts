export interface Overview {
  title: string; synopsis: string; path: string; phase: string; flow: string
  currentChapter: number; completedChapters: number; plannedChapters: number
  wordCount: number; currentVolume: number; currentArc: number
}
export interface TreeNode { id: string; kind: string; title: string; chapter: number; children: TreeNode[] }
export interface Project { overview: Overview; tree: TreeNode[] }
export interface ChapterCommitConfirmation { confirmed: boolean; project: Project }
export interface Chapter { number: number; title: string; content: string; wordCount: number; hasContent: boolean }
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
  Agent: string; Summary: string; Detail: string; Tool: string; Level: string; Depth: number
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
