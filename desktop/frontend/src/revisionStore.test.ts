import { beforeEach, expect, test, vi } from 'vitest'
import { useRevisionStore } from './revisionStore'
import type { RevisionStatus, StudioBridge } from './types'

const status = (projectId: string, state: RevisionStatus['state']): RevisionStatus => ({
  projectId, state, hasUnsynced: state === 'saved_unsynced', chapters: [],
})

beforeEach(() => {
  useRevisionStore.setState({
    projectId:'', generation:0, status:status('', 'unknown'), checking:false, error:'',
  })
  const api = {
    CheckChapterRevisions:vi.fn().mockResolvedValue(status('C:/A', 'synced')),
  } as unknown as StudioBridge
  vi.stubGlobal('window', {go:{bridge:{App:api}}})
})

test('选择项目时读取缓存并执行只读检查', async () => {
  const getStatus = vi.fn().mockResolvedValue(status('C:/A', 'unknown'))
  const check = status('C:/A', 'saved_unsynced')
  const api = window.go!.bridge.App as unknown as {CheckChapterRevisions: () => Promise<RevisionStatus>}
  api.CheckChapterRevisions = vi.fn().mockResolvedValue(check)
  await useRevisionStore.getState().selectProject('C:/A', getStatus)
  expect(getStatus).toHaveBeenCalledOnce()
  expect(api.CheckChapterRevisions).toHaveBeenCalledOnce()
  expect(useRevisionStore.getState().status).toEqual(check)
})

test('旧项目的晚到检查结果不会覆盖新项目', async () => {
  let resolveOld!: (value: RevisionStatus) => void
  const oldCheck = new Promise<RevisionStatus>(resolve => { resolveOld = resolve })
  const api = window.go!.bridge.App as unknown as {CheckChapterRevisions: () => Promise<RevisionStatus>}
  api.CheckChapterRevisions = vi.fn().mockReturnValueOnce(oldCheck).mockResolvedValueOnce(status('C:/B', 'synced'))
  const oldSelection = useRevisionStore.getState().selectProject('C:/A', async () => status('C:/A', 'unknown'))
  await Promise.resolve()
  await useRevisionStore.getState().selectProject('C:/B', async () => status('C:/B', 'unknown'))
  resolveOld(status('C:/A', 'saved_unsynced'))
  await oldSelection
  expect(useRevisionStore.getState().projectId).toBe('C:/B')
  expect(useRevisionStore.getState().status.projectId).toBe('C:/B')
  expect(useRevisionStore.getState().status.state).toBe('synced')
})

test('检查失败不会把已有待同步状态标记为已同步', async () => {
  const existing = status('C:/A', 'saved_unsynced')
  const api = window.go!.bridge.App as unknown as {CheckChapterRevisions: () => Promise<RevisionStatus>}
  api.CheckChapterRevisions = vi.fn()
    .mockResolvedValueOnce(existing)
    .mockRejectedValueOnce(new Error('进度文件损坏'))
  await useRevisionStore.getState().selectProject('C:/A', async () => existing)
  expect(useRevisionStore.getState().status).toEqual(existing)
  await useRevisionStore.getState().checkChapterRevisions()
  expect(useRevisionStore.getState().status).toEqual(existing)
  expect(useRevisionStore.getState().error).toContain('进度文件损坏')
})
