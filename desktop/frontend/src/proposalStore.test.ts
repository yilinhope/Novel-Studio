import { beforeEach, expect, test, vi } from 'vitest'
import { useProposalStore } from './proposalStore'
import type { Proposal, StudioBridge } from './types'

const proposal: Proposal = {
  id: 'old-proposal', projectId: 'C:/旧项目', title: '旧项目建议', source: 'ManualRequest', status: 'Ready',
  changes: [], evidence: [], createdAt: '', updatedAt: '',
}

beforeEach(() => {
  useProposalStore.setState({projectId: 'C:/旧项目', proposals: [], selected: null, diff: [], versions: [], loading: false, busy: false, error: '', notice: ''})
})

test('项目切换后忽略晚到的旧 Proposal 创建响应', async () => {
  let resolveCreate!: (value: Proposal) => void
  const api: StudioBridge = {
    CreateProposal: vi.fn(() => new Promise(resolve => { resolveCreate = resolve })),
  } as unknown as StudioBridge
  vi.stubGlobal('window', {go: {bridge: {App: api}}})

  const creating = useProposalStore.getState().createManual(1, '旧建议', '旧项目候选正文')
  useProposalStore.setState({projectId: 'D:/新项目', proposals: [], selected: null})
  resolveCreate(proposal)
  await creating

  expect(useProposalStore.getState().proposals).toEqual([])
  expect(useProposalStore.getState().selected).toBeNull()
})

