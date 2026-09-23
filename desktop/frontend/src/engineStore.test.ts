import { beforeEach, expect, test, vi } from 'vitest'
import { runtimeLabel, setChapterCommitHandler, useEngineStore } from './engineStore'
import { useRevisionStore } from './revisionStore'
import type { RevisionStatus, RuntimeLog, StudioBridge, StudioEngineEvent } from './types'

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
  useRevisionStore.setState({projectId:'C:/A', generation:1, status:{projectId:'C:/A',state:'synced',hasUnsynced:false,chapters:[]},checking:false,error:''})
  setChapterCommitHandler(undefined)
})

test('waiting_review 使用等待继续确认文案', () => {
  expect(runtimeLabel('waiting_review')).toBe('等待继续确认')
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

test('拒绝旧项目代际事件', () => {
  useEngineStore.setState({runtime: {...runtime('C:/A', 3), state:'paused'}})
  useEngineStore.getState().handleEvent({...event('C:/A', 9, commit(3)), generation:2})
  expect(useEngineStore.getState().logs).toHaveLength(0)
  expect(useEngineStore.getState().lastSequence).toBe(0)
})

test('Auto/Review 切换只调用模式 API，不隐式 Resume', async () => {
  const result = {runtime:{...runtime('C:/A'), state:'paused' as const}, revision:{projectId:'C:/A',state:'synced' as const,hasUnsynced:false,chapters:[]}}
  const setMode = vi.fn().mockResolvedValue(result)
  const resume = vi.fn()
  vi.stubGlobal('window', {go:{bridge:{App:{SetAdvanceMode:setMode, ResumeWriting:resume} as unknown as StudioBridge}}})
  await useEngineStore.getState().setAdvanceMode('review')
  await useEngineStore.getState().setAdvanceMode('auto')
  expect(setMode).toHaveBeenNthCalledWith(1, 'review')
  expect(setMode).toHaveBeenNthCalledWith(2, 'auto')
  expect(resume).not.toHaveBeenCalled()
  expect(useEngineStore.getState().runtime.state).toBe('paused')
})

test('继续下一章使用 Core 结果而不是前端伪造状态', async () => {
  const nextRuntime = {...runtime('C:/A', 2), state:'running' as const, canAdvance:false}
  const advance = vi.fn().mockResolvedValue({runtime:nextRuntime,revision:{projectId:'C:/A',state:'synced',hasUnsynced:false,chapters:[]}})
  vi.stubGlobal('window', {go:{bridge:{App:{AdvanceOneChapter:advance} as unknown as StudioBridge}}})
  await useEngineStore.getState().advanceOneChapter()
  expect(advance).toHaveBeenCalledOnce()
  expect(useEngineStore.getState().runtime).toEqual(nextRuntime)
})

test('有未同步修订时阻止 ResumeWriting 调用', async () => {
  const resume = vi.fn().mockResolvedValue(runtime('C:/A'))
  vi.stubGlobal('window', {go:{bridge:{App:{ResumeWriting:resume} as unknown as StudioBridge}}})
  useRevisionStore.setState({status:{projectId:'C:/A',state:'saved_unsynced',hasUnsynced:true,chapters:[]}})

  await useEngineStore.getState().resumeWriting()

  expect(resume).not.toHaveBeenCalled()
  expect(useEngineStore.getState().controlError).toContain('同步所有未同步修订')
})

test('修订状态尚未确认时阻止 ResumeWriting 调用', async () => {
  const resume = vi.fn().mockResolvedValue(runtime('C:/A'))
  vi.stubGlobal('window', {go:{bridge:{App:{ResumeWriting:resume} as unknown as StudioBridge}}})
  useRevisionStore.setState({status:{projectId:'C:/A',state:'unknown',hasUnsynced:false,chapters:[]}})

  await useEngineStore.getState().resumeWriting()

  expect(resume).not.toHaveBeenCalled()
})
