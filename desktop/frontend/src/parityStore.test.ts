import { beforeEach, expect, test, vi } from 'vitest'
import { useParityStore } from './parityStore'
import { useStudio } from './store'
import type { CharacterPage, Project, StudioBridge } from './types'

const project: Project = {
  projectId: 'project-a', generation: 7, projectRoot: 'C:/books/a', outputDir: 'C:/books/a/output',
  overview: {title: '项目 A', synopsis: '', path: 'C:/books/a/output', phase: 'writing', flow: 'writing', currentChapter: 1, completedChapters: 0, plannedChapters: 1, wordCount: 0, currentVolume: 1, currentArc: 1},
  tree: [],
}

let api: StudioBridge

beforeEach(() => {
  api = {
    SelectProjectDirectory: vi.fn(), OpenProject: vi.fn(), GetProjectOverview: vi.fn(), GetProjectTree: vi.fn(), GetChapter: vi.fn(), SaveChapter: vi.fn(), GetRevisionStatus: vi.fn(), CheckChapterRevisions: vi.fn(),
    GetStoryCharacters: vi.fn(), GetStoryPremise: vi.fn(), GetStoryWorldRules: vi.fn(), GetStoryOutline: vi.fn(), GetStoryLayeredOutline: vi.fn(), GetStoryCompass: vi.fn(), GetStorySummaries: vi.fn(),
    GetContinuityTimeline: vi.fn(), GetContinuityForeshadow: vi.fn(), GetContinuityRelationships: vi.fn(), GetContinuityStateChanges: vi.fn(), GetContinuitySnapshots: vi.fn(), GetContinuityCast: vi.fn(),
  }
  vi.stubGlobal('window', {go: {bridge: {App: api}}})
  useStudio.setState({project, view: 'overview', busy: false, syncing: false, saveBusy: false, dirty: false})
  useParityStore.getState().reset()
})

function characterPage(name: string): CharacterPage {
  return {projectId: 'C:/books/a/output', generation: 7, projectRoot: 'C:/books/a', outputDir: 'C:/books/a/output', requestId: 'parity-1', sequence: 1, offset: 0, limit: 50, total: 1, hasMore: false, items: [{name, role: '主角', description: '', arc: ''}]}
}

test('故事导航通过桥接读取 Core ViewModel，并携带项目身份', async () => {
  vi.mocked(api.GetStoryCharacters!).mockResolvedValue(characterPage('林渡'))

  useStudio.getState().parity('characters')
  await vi.waitFor(() => expect(useParityStore.getState().data).not.toBeNull())

  expect(useStudio.getState().view).toBe('parity')
  expect(useParityStore.getState().section).toBe('characters')
  expect(api.GetStoryCharacters).toHaveBeenCalledWith(expect.objectContaining({projectId: project.outputDir, generation: 7, offset: 0, limit: 50}))
  expect(useParityStore.getState().data).toMatchObject({items: [{name: '林渡'}]})
})

test('全书大纲可分别切换 Core 分层与扁平数据', async () => {
  vi.mocked(api.GetStoryLayeredOutline!).mockResolvedValue({projectId: project.outputDir, generation: 7, projectRoot: project.projectRoot, outputDir: project.outputDir, offset: 0, limit: 50, total: 1, hasMore: false, items: [{index: 1, title: '第一卷', theme: '归途', final: false, arcs: []}]})
  vi.mocked(api.GetStoryOutline!).mockResolvedValue({projectId: project.outputDir, generation: 7, projectRoot: project.projectRoot, outputDir: project.outputDir, offset: 0, limit: 50, total: 1, hasMore: false, items: [{chapter: 1, title: '出航', coreEvent: '离港', hook: '风暴'}]})

  await useParityStore.getState().selectSection('outline')
  expect(api.GetStoryLayeredOutline).toHaveBeenCalledOnce()
  await useParityStore.getState().selectOutlineMode('flat')

  expect(api.GetStoryOutline).toHaveBeenCalledOnce()
  expect(useParityStore.getState().data).toMatchObject({items: [{title: '出航'}]})
})

test('分层大纲翻页时同步当前卷列表，避免 Arc Selector 停留在旧页', async () => {
  vi.mocked(api.GetStoryLayeredOutline!).mockResolvedValueOnce({projectId: project.outputDir, generation: 7, projectRoot: project.projectRoot, outputDir: project.outputDir, offset: 0, limit: 50, total: 51, hasMore: true, items: Array.from({length: 50}, (_, index) => ({index: index + 1, title: `第${index + 1}卷`, theme: '', final: false, arcs: []}))})
    .mockResolvedValueOnce({projectId: project.outputDir, generation: 7, projectRoot: project.projectRoot, outputDir: project.outputDir, offset: 50, limit: 50, total: 51, hasMore: false, items: [{index: 51, title: '第51卷', theme: '', final: true, arcs: [{index: 1, title: '终章弧', goal: '收束', chapterCount: 1}]}]})

  await useParityStore.getState().selectSection('outline')
  await useParityStore.getState().page(50)

  expect(useParityStore.getState().data).toMatchObject({offset: 50, items: [{index: 51}]})
  expect(useParityStore.getState().layeredOutline?.items).toHaveLength(1)
  expect(useParityStore.getState().layeredOutline?.items[0]?.arcs[0]?.title).toBe('终章弧')
})

test('角色快照可在最新与指定卷弧之间切换', async () => {
  vi.mocked(api.GetContinuitySnapshots!).mockResolvedValueOnce({projectId: project.outputDir, generation: 7, projectRoot: project.projectRoot, outputDir: project.outputDir, offset: 0, limit: 50, total: 1, hasMore: false, scopes: [{volume: 1, arc: 1, title: '离港'}], items: [{volume: 1, arc: 1, name: '林渡', status: '坚定', motivation: '回家'}]})
    .mockResolvedValueOnce({projectId: project.outputDir, generation: 7, projectRoot: project.projectRoot, outputDir: project.outputDir, offset: 0, limit: 50, total: 1, hasMore: false, scopes: [{volume: 1, arc: 1, title: '离港'}, {volume: 1, arc: 2, title: '回响'}], items: [{volume: 1, arc: 2, name: '林渡', status: '觉醒', motivation: '面对真相'}]})

  await useParityStore.getState().selectSection('snapshots')
  await useParityStore.getState().selectSnapshotScope(1, 2)

  expect(api.GetContinuitySnapshots).toHaveBeenNthCalledWith(1, expect.objectContaining({projectId: project.outputDir}), 0, 0)
  expect(api.GetContinuitySnapshots).toHaveBeenNthCalledWith(2, expect.objectContaining({projectId: project.outputDir}), 1, 2)
  expect(useParityStore.getState().data).toMatchObject({items: [{arc: 2, status: '觉醒'}]})
})

test('项目切换后旧的连续性响应不会污染当前页面', async () => {
  let resolveOld!: (page: CharacterPage) => void
  vi.mocked(api.GetStoryCharacters!).mockImplementation(() => new Promise(resolve => {resolveOld = resolve}))

  const pending = useParityStore.getState().selectSection('characters')
  useStudio.setState({project: {...project, projectId: 'project-b', outputDir: 'C:/books/b/output', overview: {...project.overview, path: 'C:/books/b/output'}}})
  useParityStore.getState().reset()
  resolveOld(characterPage('旧项目角色'))
  await pending

  expect(useParityStore.getState().data).toBeNull()
  expect(useParityStore.getState().loading).toBe(false)
})

test('读取失败和空列表保持可见状态，分页请求继续携带当前项目', async () => {
  vi.mocked(api.GetContinuityTimeline!).mockRejectedValueOnce(new Error('读取失败'))
  await useParityStore.getState().selectSection('timeline')
  expect(useParityStore.getState().error).toContain('读取失败')

  vi.mocked(api.GetContinuityTimeline!).mockResolvedValue({projectId: project.outputDir, generation: 7, projectRoot: project.projectRoot, outputDir: project.outputDir, requestId: 'parity-2', sequence: 2, offset: 0, limit: 50, total: 0, hasMore: false, items: []})
  await useParityStore.getState().selectSection('timeline')
  expect(useParityStore.getState().data).toMatchObject({total: 0, items: []})
  await useParityStore.getState().page(50)
  expect(api.GetContinuityTimeline).toHaveBeenLastCalledWith(expect.objectContaining({projectId: project.outputDir, generation: 7, offset: 50, limit: 50}))
})
