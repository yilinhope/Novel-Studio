import { beforeEach, expect, test, vi } from 'vitest'
import { useConfigStore } from './configStore'
import { useCreateProjectStore } from './createProjectStore'
import { useExportStore } from './exportStore'
import type { ConfigSnapshot, CreateEvent, CoCreateEvent, ProviderDraft, StudioBridge } from './types'

const config = (projectRoot: string): ConfigSnapshot => ({
  projectRoot, configPath: `${projectRoot}/.ainovel/config.json`, provider: 'openai', model: 'model-a',
  providers: [], roles: {}, budget: {bookUsd: 0, warnRatio: .8, hardStop: false}, notify: {},
})
const studioListeners: Record<string, (payload: unknown) => void> = {}

beforeEach(() => {
  useConfigStore.setState({config: null, usage: null, busy: false, error: ''})
  useExportStore.setState({busy: false, error: '', result: null})
  useCreateProjectStore.setState({mode: 'quick', projectRoot: '', prompt: '', stage: false, ack: null, requestId: '', event: null, coCreate: null, pendingCreateEvent: null, pendingCoCreateEvent: null, recovery: null, history: [], busy: false, previewing: false, previewText: '', error: ''})
})

test('Config 迟到响应不会覆盖更新后的请求', async () => {
  let resolveOld!: (value: ConfigSnapshot) => void
  let resolveNew!: (value: ConfigSnapshot) => void
  const api: StudioBridge = {GetConfig: vi.fn()
    .mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    .mockImplementationOnce(() => new Promise(resolve => { resolveNew = resolve }))} as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}})
  const oldRequest = useConfigStore.getState().load()
  const newRequest = useConfigStore.getState().load()
  resolveNew(config('D:/new-project'))
  await newRequest
  resolveOld(config('D:/old-project'))
  await oldRequest
  expect(useConfigStore.getState().config?.projectRoot).toBe('D:/new-project')
})

test('Config 读取桥接失败时离开加载态并显示可见错误', async () => {
  vi.stubGlobal('window', {})

  await useConfigStore.getState().load()

  expect(useConfigStore.getState().busy).toBe(false)
  expect(useConfigStore.getState().error).toContain('桌面程序')
})

test('Config 与 Usage 并行读取时不会互相取消配置响应', async () => {
  let resolveConfig!: (value: ConfigSnapshot) => void
  let resolveUsage!: (value: never) => void
  const api: StudioBridge = {
    GetConfig: vi.fn().mockReturnValue(new Promise(resolve => { resolveConfig = resolve })),
    GetUsage: vi.fn().mockReturnValue(new Promise(resolve => { resolveUsage = resolve })),
  } as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}})

  const configRequest = useConfigStore.getState().load()
  const usageRequest = useConfigStore.getState().loadUsage()
  resolveConfig(config('D:/parallel-project'))
  resolveUsage(undefined as never)
  await Promise.all([configRequest, usageRequest])

  expect(useConfigStore.getState().config?.projectRoot).toBe('D:/parallel-project')
  expect(useConfigStore.getState().busy).toBe(false)
})

test('模型配置页面把发现与连接测试委托给 Core Bridge', async () => {
  const draft: ProviderDraft = {provider: 'proxy', type: 'openai', api: 'chat', baseUrl: 'https://example.invalid/v1', models: [{name: 'model-a'}], apiKeyAction: 'keep'}
  const discover = vi.fn().mockResolvedValue([{name: 'model-a'}, {name: 'model-b'}])
  const testConnection = vi.fn().mockResolvedValue(undefined)
  const api: StudioBridge = {DiscoverProviderModels: discover, TestModelConnection: testConnection} as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}})

  await expect(useConfigStore.getState().discoverModels(draft)).resolves.toEqual([{name: 'model-a'}, {name: 'model-b'}])
  await useConfigStore.getState().testModelConnection(draft, 'model-a')
  expect(discover).toHaveBeenCalledWith(draft)
  expect(testConnection).toHaveBeenCalledWith(draft, 'model-a')
})

test('模型配置保存失败时向编辑器传播 Core 错误', async () => {
  const saveProvider = vi.fn().mockRejectedValue(new Error('Provider 配置保存失败'))
  vi.stubGlobal('window', {go: {bridge: {App: {SaveProviderConfig: saveProvider} as unknown as StudioBridge}}})

  await expect(useConfigStore.getState().saveProvider({provider: 'proxy', type: 'openai', api: 'chat', baseUrl: 'https://example.invalid/v1', models: [{name: 'model-a'}], apiKeyAction: 'keep'})).rejects.toThrow('Provider 配置保存失败')
  expect(useConfigStore.getState().error).toContain('Provider 配置保存失败')
  expect(useConfigStore.getState().busy).toBe(false)
})

test('Export 错误保留 Core 错误语义', async () => {
  const exportProject = vi.fn().mockRejectedValue(new Error('EPUB 输出路径已存在'))
  vi.stubGlobal('window', {go: {bridge: {App: {ExportProject: exportProject} as unknown as StudioBridge}}})
  await useExportStore.getState().exportProject({format: 'epub', outPath: 'D:/book.epub', from: 0, to: 0, overwrite: false})
  expect(useExportStore.getState().error).toContain('EPUB 输出路径已存在')
})

test('Quick Start terminal event 先于 ack 到达时不会丢失', async () => {
  let request!: {requestId?: string}
  let resolveAck!: (value: {projectId: string; generation: number; operation: string; requestId: string}) => void
  const api: StudioBridge = {
    StartQuickStart: vi.fn(value => { request = value; return new Promise(resolve => { resolveAck = resolve }) }),
  } as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}, runtime: {EventsOn: vi.fn((name, callback) => { studioListeners[name] = callback; return () => {} })}})
  const started = useCreateProjectStore.getState().start()
  await Promise.resolve()
  const event: CreateEvent = {projectId: 'D:/book/output/novel', generation: 1, operation: 'create', requestId: request.requestId, state: 'error', error: 'Core 立即拒绝'}
  studioListeners['studio:create-event'](event)
  expect(useCreateProjectStore.getState().pendingCreateEvent?.error).toBe('Core 立即拒绝')
  resolveAck({projectId: event.projectId, generation: 1, operation: 'create', requestId: request.requestId!})
  await started
  expect(useCreateProjectStore.getState().event?.error).toBe('Core 立即拒绝')
  expect(useCreateProjectStore.getState().busy).toBe(false)
})

test('Co-create terminal event 先于 ack 到达时会保留回复与历史', async () => {
  useCreateProjectStore.setState({mode: 'cocreate', projectRoot: 'D:/book', prompt: '开场'})
  let requestId = ''
  let resolveAck!: (value: {projectId: string; generation: number; operation: string; requestId: string; mode: 'cold'}) => void
  const api: StudioBridge = {
    StartCoCreate: vi.fn((_root, _initial, _stage, id) => { requestId = id; return new Promise(resolve => { resolveAck = resolve }) }),
  } as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}, runtime: {EventsOn: vi.fn((name, callback) => { studioListeners[name] = callback; return () => {} })}})
  const started = useCreateProjectStore.getState().start()
  await Promise.resolve()
  const event: CoCreateEvent = {projectId: 'D:/book/output/novel', generation: 1, requestId, state: 'reply', reply: '已恢复最后一轮', draft: '## 草稿', ready: true, history: [{role: 'user', content: '开场'}]}
  studioListeners['studio:cocreate-event'](event)
  expect(useCreateProjectStore.getState().pendingCoCreateEvent?.reply).toBe('已恢复最后一轮')
  resolveAck({projectId: event.projectId, generation: 1, operation: 'cocreate', requestId, mode: 'cold'})
  await started
  expect(useCreateProjectStore.getState().coCreate?.draft).toBe('## 草稿')
  expect(useCreateProjectStore.getState().history).toEqual([{role: 'user', content: '开场'}, {role: 'assistant', content: '已恢复最后一轮'}])
  expect(useCreateProjectStore.getState().busy).toBe(false)
})

test('Co-create recovery hydrate 历史草稿并允许继续上次共创', async () => {
  const history = [{role: 'user' as const, content: '开场'}, {role: 'assistant' as const, content: '继续规划'}]
  const resume = vi.fn().mockResolvedValue({projectId: 'D:/book/output/novel', generation: 4, operation: 'cocreate', requestId: 'recovery-1', mode: 'cold'})
  const api: StudioBridge = {
    GetCoCreateRecovery: vi.fn().mockResolvedValue({projectId: 'D:/book/output/novel', generation: 4, exists: true, interrupted: true, mode: 'cold', history, draft: '## 草稿', ready: true, suggestions: ['继续'] }),
    ResumeCoCreate: resume,
  } as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}})
  await useCreateProjectStore.getState().loadRecovery()
  expect(useCreateProjectStore.getState().history).toEqual(history)
  expect(useCreateProjectStore.getState().coCreate?.draft).toBe('## 草稿')
  await useCreateProjectStore.getState().resumeRecovery()
  expect(resume).toHaveBeenCalledWith(false, history, expect.any(String))
})
