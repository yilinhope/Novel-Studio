import { beforeEach, expect, test, vi } from 'vitest'
import { setChapterCommitHandler, useEngineStore } from './engineStore'
import type { RuntimeLog, StudioEngineEvent } from './types'

const runtime = (projectId: string, generation = 1) => ({
  projectId, generation, state: 'running' as const, phase: 'writing', flow: 'writing', elapsedSeconds: 1,
  inputTokens: 0, outputTokens: 0, projectInputTokens: 0, projectOutputTokens: 0,
  runCostUsd: 0, projectCostUsd: 0, agents: [], updatedAt: new Date().toISOString(),
})
const event = (projectId: string, sequence: number, log?: RuntimeLog): StudioEngineEvent => ({
  projectId, generation: 1, runId: 1, sequence, timestamp: new Date().toISOString(), type: 'log', log,
})
const commit = (chapter?: number): RuntimeLog => ({
  ID: 'tool-1', Time: new Date().toISOString(), FinishedAt: new Date().toISOString(), Failed: false,
  Category: 'TOOL', Agent: 'writer', Summary: '章节完成', Detail: '', Tool: 'commit_chapter',
  Chapter: chapter, Level: 'success', Depth: 1,
})

beforeEach(() => {
  useEngineStore.setState({projectId: 'C:/A', runtime: runtime('C:/A'), logs: [], pipeline: {}, lastSequence: 0,
    controlBusy: false, controlError: ''})
  setChapterCommitHandler(undefined)
})

test('章节刷新使用结构化 Chapter 而不解析展示文案', () => {
  const refresh = vi.fn().mockResolvedValue(undefined)
  setChapterCommitHandler(refresh)
  useEngineStore.getState().handleEvent(event('C:/A', 1, commit(23)))
  expect(refresh).toHaveBeenCalledWith(23, expect.objectContaining({sequence: 1}))
})

test('缺少结构化章节号时不猜测提交章节', () => {
  const refresh = vi.fn().mockResolvedValue(undefined)
  setChapterCommitHandler(refresh)
  useEngineStore.getState().handleEvent(event('C:/A', 1, commit()))
  expect(refresh).not.toHaveBeenCalled()
})

test('拒绝旧项目和旧序号事件', () => {
  useEngineStore.getState().handleEvent(event('C:/B', 5, commit(1)))
  useEngineStore.getState().handleEvent(event('C:/A', 1, commit(1)))
  useEngineStore.getState().handleEvent(event('C:/A', 1, commit(2)))
  expect(useEngineStore.getState().logs).toHaveLength(1)
  expect(useEngineStore.getState().lastSequence).toBe(1)
})
