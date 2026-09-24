import { expect, test } from 'vitest'
import { formatRuntimeActivity, runtimeElapsedSeconds } from './components/RuntimeCenter'
import type { Runtime, RuntimeLog } from './types'

const runtime = (updatedAt: string, elapsedSeconds: number): Runtime => ({
  projectId: 'C:/book/output/novel', generation: 1, state: 'running', phase: 'writing', flow: 'writing',
  elapsedSeconds, inputTokens: 0, outputTokens: 0, projectInputTokens: 0, projectOutputTokens: 0,
  runCostUsd: 0, projectCostUsd: 0, agents: [], updatedAt,
})

const log = (overrides: Partial<RuntimeLog>): RuntimeLog => ({
  ID: 'model-1', Time: '2026-09-24T10:00:00.000Z', FinishedAt: '', Failed: false,
  Category: 'MODEL', Agent: 'architect_long', Summary: '模型响应', Detail: '等待模型', Tool: '', Chapter: 0,
  Level: 'info', Depth: 0, ...overrides,
})

test('运行中时长会随最后一次 Core 快照继续显示', () => {
  expect(runtimeElapsedSeconds(runtime('2026-09-24T10:00:00.000Z', 120), new Date('2026-09-24T10:00:07.000Z'))).toBe(127)
})

test('当前活动事件显示 Core 正在做什么', () => {
  expect(formatRuntimeActivity(log({}))).toBe('模型响应 · 等待模型')
})
