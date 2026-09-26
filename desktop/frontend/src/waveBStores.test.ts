import { beforeEach, expect, test, vi } from 'vitest'
import { useSimulationStore } from './simulationStore'
import { useWritingSettingsStore } from './writingSettingsStore'
import { useStudio } from './store'
import type { Project, SimulationProfilePage, SimulationSourcesPage, RulesWorkspace, StyleState, StudioBridge } from './types'

const project: Project = {
  projectRoot: 'C:/books/a', outputDir: 'C:/books/a/output', generation: 3,
  overview: {title: '项目 A', synopsis: '', path: 'C:/books/a/output', phase: 'writing', flow: 'writing', currentChapter: 1, completedChapters: 0, plannedChapters: 1, wordCount: 0, currentVolume: 1, currentArc: 1}, tree: [],
}

let api: StudioBridge

beforeEach(() => {
  api = {
    GetSimulationSources: vi.fn(), GetSimulationProfile: vi.fn(), GetRulesWorkspace: vi.fn(), GetStyleState: vi.fn(),
  } as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}})
  useStudio.setState({project, view: 'overview', busy: false, syncing: false, saveBusy: false, dirty: false})
  useSimulationStore.getState().reset()
  useWritingSettingsStore.getState().reset()
})

test('Simulation 旧项目响应不会污染切换后的项目', async () => {
  let resolveSources!: (value: SimulationSourcesPage) => void
  let resolveProfile!: (value: SimulationProfilePage) => void
  vi.mocked(api.GetSimulationSources!).mockImplementation(() => new Promise(resolve => { resolveSources = resolve }))
  vi.mocked(api.GetSimulationProfile!).mockImplementation(() => new Promise(resolve => { resolveProfile = resolve }))
  const pending = useSimulationStore.getState().load()
  useStudio.setState({project: {...project, outputDir: 'C:/books/b/output'}})
  useSimulationStore.getState().reset()
  resolveSources({projectId: project.outputDir, generation: 3, projectRoot: project.projectRoot, outputDir: project.outputDir, sourceDir: '', items: []})
  resolveProfile({projectId: project.outputDir, generation: 3, projectRoot: project.projectRoot, outputDir: project.outputDir, available: false})
  await pending
  expect(useSimulationStore.getState().sources).toBeNull()
  expect(useSimulationStore.getState().profile).toBeNull()
})

test('写作设置旧项目响应不会恢复旧的 loading 或配置状态', async () => {
  let resolveRules!: (value: RulesWorkspace) => void
  let resolveStyle!: (value: StyleState) => void
  vi.mocked(api.GetRulesWorkspace!).mockImplementation(() => new Promise(resolve => { resolveRules = resolve }))
  vi.mocked(api.GetStyleState!).mockImplementation(() => new Promise(resolve => { resolveStyle = resolve }))
  const pending = useWritingSettingsStore.getState().load()
  useStudio.setState({project: {...project, outputDir: 'C:/books/b/output'}})
  useWritingSettingsStore.getState().reset()
  resolveRules({projectId: project.outputDir, generation: 3, projectRoot: project.projectRoot, outputDir: project.outputDir, global: [], project: [], effectiveAvailable: false})
  resolveStyle({projectId: project.outputDir, generation: 3, projectRoot: project.projectRoot, outputDir: project.outputDir, selectedStyle: 'default', styleNames: [], styleSource: 'Built-in', voice: '', voiceSource: 'Built-in', antiAiTone: '', antiAiToneSource: 'Built-in', effectiveNotice: ''})
  await pending
  expect(useWritingSettingsStore.getState().rules).toBeNull()
  expect(useWritingSettingsStore.getState().style).toBeNull()
  expect(useWritingSettingsStore.getState().loading).toBe(false)
})
