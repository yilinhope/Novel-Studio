export interface Overview {
  title: string; synopsis: string; path: string; phase: string; flow: string
  currentChapter: number; completedChapters: number; plannedChapters: number
  wordCount: number; currentVolume: number; currentArc: number
}
export interface TreeNode { id: string; kind: string; title: string; chapter: number; children: TreeNode[] }
export interface Project { overview: Overview; tree: TreeNode[] }
export interface Chapter { number: number; title: string; content: string; wordCount: number; hasContent: boolean }
export interface StudioBridge {
  SelectProjectDirectory(): Promise<string>
  OpenProject(path: string): Promise<Project>
  GetProjectOverview(): Promise<Overview>
  GetProjectTree(): Promise<TreeNode[]>
  GetChapter(number: number): Promise<Chapter>
}
declare global { interface Window { go?: { bridge: { App: StudioBridge } } } }
