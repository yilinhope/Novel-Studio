import { beforeEach, expect, test, vi } from 'vitest'
import { useImportStore } from './importStore'
import type { ImportOptions, ImportStatus, StudioBridge } from './types'

const listeners: Record<string, (payload: unknown) => void> = {}

const options: ImportOptions = {
  projectRoot: 'D:/book', sourcePath: 'D:/source.txt', autoConfirm: true,
  acceptSegmentation: true, storyResolution: 'auto', continueAfter: false, guidance: '',
}

beforeEach(() => {
  useImportStore.setState({projectId: '', generation: 0, requestId: '', status: null, pendingStatus: null, busy: false, error: '', sequence: 0})
})

test('Import 终态事件先于 ack 到达时不会丢失', async () => {
  let request!: ImportOptions
  let resolveAck!: (value: {projectId: string; generation: number; operation: string; requestId: string}) => void
  const api: StudioBridge = {
    StartImport: vi.fn(value => {
      request = value
      return new Promise(resolve => { resolveAck = resolve })
    }),
  } as unknown as StudioBridge
  vi.stubGlobal('window', {
    go: {bridge: {App: api}},
    runtime: {EventsOn: vi.fn((name, callback) => { listeners[name] = callback; return () => {} })},
  })

  const started = useImportStore.getState().start(options)
  await Promise.resolve()
  const event: ImportStatus = {
    projectId: 'D:/book/output/novel', generation: 1, requestId: request.requestId,
    active: false, stage: 'done', current: 1, total: 1, message: '导入完成', continued: false,
  }
  listeners['studio:import-event'](event)
  resolveAck({projectId: event.projectId, generation: event.generation, operation: 'import', requestId: request.requestId!})
  await started

  expect(useImportStore.getState().status?.stage).toBe('done')
  expect(useImportStore.getState().busy).toBe(false)
})
