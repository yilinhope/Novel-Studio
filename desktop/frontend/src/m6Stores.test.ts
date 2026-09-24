import { beforeEach, expect, test, vi } from 'vitest'
import { useConfigStore } from './configStore'
import { useExportStore } from './exportStore'
import type { ConfigSnapshot, StudioBridge } from './types'

const config = (projectRoot: string): ConfigSnapshot => ({
  projectRoot, configPath: `${projectRoot}/.ainovel/config.json`, provider: 'openai', model: 'model-a',
  providers: [], roles: {}, budget: {bookUsd: 0, warnRatio: .8, hardStop: false}, notify: {},
})

beforeEach(() => {
  useConfigStore.setState({config: null, usage: null, busy: false, error: ''})
  useExportStore.setState({busy: false, error: '', result: null})
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

test('Export 错误保留 Core 错误语义', async () => {
  const exportProject = vi.fn().mockRejectedValue(new Error('EPUB 输出路径已存在'))
  vi.stubGlobal('window', {go: {bridge: {App: {ExportProject: exportProject} as unknown as StudioBridge}}})
  await useExportStore.getState().exportProject({format: 'epub', outPath: 'D:/book.epub', from: 0, to: 0, overwrite: false})
  expect(useExportStore.getState().error).toContain('EPUB 输出路径已存在')
})
