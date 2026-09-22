import { create } from 'zustand'
import { subscribeEngineEvents } from './services'
import type { Runtime, RuntimeLog, RuntimeState, StudioEngineEvent } from './types'

const emptyRuntime = (projectId = '', generation = 0): Runtime => ({
  projectId, generation, state: 'idle', phase: '', flow: '', elapsedSeconds: 0,
  inputTokens: 0, outputTokens: 0, projectInputTokens: 0, projectOutputTokens: 0,
  runCostUsd: 0, projectCostUsd: 0, agents: [], updatedAt: new Date().toISOString(),
})
const normalizePath = (path: string) => path.replaceAll('\\', '/').replace(/\/+$/, '').toLocaleLowerCase()
const isFinished = (timestamp: string) => Date.parse(timestamp) > 0
const logLimit = 500

export interface PipelineStep { tool: string; state: 'queued' | 'running' | 'complete' | 'error' }
interface EngineStore {
  projectId: string
  runtime: Runtime
  logs: RuntimeLog[]
  pipeline: Record<string, PipelineStep>
  lastSequence: number
  selectProject(path: string, getRuntime?: (() => Promise<Runtime>) | undefined): Promise<void>
  handleEvent(event: StudioEngineEvent): void
}

let unsubscribe: (() => void) | undefined
let selection = 0

export const useEngineStore = create<EngineStore>((set, get) => ({
  projectId: '', runtime: emptyRuntime(), logs: [], pipeline: {}, lastSequence: 0,
  async selectProject(path, getRuntime) {
    const ticket = ++selection
    const projectId = path
    set({projectId, runtime: emptyRuntime(projectId), logs: [], pipeline: {}})
    if (!unsubscribe) unsubscribe = subscribeEngineEvents(event => get().handleEvent(event))
    if (!getRuntime) return
    try {
      const runtime = await getRuntime()
      const current = get()
      if (ticket !== selection || normalizePath(runtime.projectId) !== normalizePath(current.projectId) || runtime.generation < current.runtime.generation) return
      set(state => ({runtime, lastSequence: state.lastSequence}))
    } catch {
      // 项目仍保持可浏览，Runtime Center 展示尚未建立 Engine Session 的状态。
    }
  },
  handleEvent(event) {
    const current = get()
    if (!current.projectId || normalizePath(event.projectId) !== normalizePath(current.projectId)) return
    if (event.generation < current.runtime.generation || event.sequence <= current.lastSequence) return
    if (event.generation > current.runtime.generation) {
      set({runtime: emptyRuntime(current.projectId, event.generation), logs: [], pipeline: {}})
    }
    const next = get()
    if (event.log) {
      const log = event.log
      const logs = [...next.logs, log].slice(-logLimit)
      let pipeline = next.pipeline
      if (log.Category === 'TOOL' && log.Tool) {
        const state: PipelineStep['state'] = log.Failed
          ? 'error'
          : isFinished(log.FinishedAt)
            ? 'complete'
            : 'running'
        pipeline = {...pipeline, [log.Tool]: {tool: log.Tool, state}}
      }
      set({logs, pipeline})
    }
    if (event.runtime) set({runtime: event.runtime})
    set({lastSequence: event.sequence})
  },
}))

export function runtimeLabel(state: RuntimeState): string {
  const labels: Record<RuntimeState, string> = {
    idle: '待启动', running: '运行中', pausing: '正在暂停', paused: '已暂停',
    stopping: '正在停止', stopped: '已停止', waiting_review: '等待审核',
    waiting_sync: '等待同步', completed: '已完成', error: '发生错误',
  }
  return labels[state]
}
