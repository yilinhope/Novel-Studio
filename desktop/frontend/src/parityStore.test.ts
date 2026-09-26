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
