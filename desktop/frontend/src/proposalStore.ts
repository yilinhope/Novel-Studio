import { create } from 'zustand'
import { bridge } from './services'
import { useRevisionStore } from './revisionStore'
import { useEngineStore } from './engineStore'
import type { ApplyResult, DiffLine, Proposal, VersionSnapshot } from './types'

const sameProject = (left: string, right: string) => left.replaceAll('\\', '/').toLocaleLowerCase() === right.replaceAll('\\', '/').toLocaleLowerCase()

interface ProposalState {
  projectId: string; proposals: Proposal[]; selected: Proposal | null; diff: DiffLine[]; versions: VersionSnapshot[]
  loading: boolean; busy: boolean; error: string; notice: string
  load(projectId: string): Promise<void>; select(id: string): Promise<void>; refreshSelected(): Promise<void>
  createManual(chapter: number, title: string, after: string): Promise<Proposal | null>
  createFromReview(chapter: number, scope: string, issueIndex: number, after: string): Promise<Proposal | null>
  accept(): Promise<void>; reject(): Promise<void>; apply(): Promise<ApplyResult | null>; restoreVersion(id: string): Promise<void>
}

export const useProposalStore = create<ProposalState>((set, get) => ({
  projectId: '', proposals: [], selected: null, diff: [], versions: [], loading: false, busy: false, error: '', notice: '',
  async load(projectId) {
    const getter = bridge().ListProposals
    if (!getter) { set({error: '桌面桥接尚未提供 Proposal API。'}); return }
    set({loading: true, error: '', notice: '', projectId})
    try {
      const proposals = await getter()
      if (!sameProject(get().projectId, projectId)) return
      set({proposals})
      const selected = get().selected
      if (selected && proposals.some(item => item.id === selected.id)) await get().select(selected.id)
      else if (proposals[0]) await get().select(proposals[0].id)
      else set({selected: null, diff: [], versions: []})
    } catch (cause) { if (sameProject(get().projectId, projectId)) set({error: String(cause)}) }
    finally { if (sameProject(get().projectId, projectId)) set({loading: false}) }
  },
  async select(id) {
    const getter = bridge().GetProposal
    const diffGetter = bridge().GetProposalDiff
    const versionsGetter = bridge().GetVersionHistory
    if (!getter) return
    set({loading: true, error: ''})
    try {
      const selected = await getter(id)
      if (get().projectId && !sameProject(selected.projectId, get().projectId)) return
      const [diff, versions] = await Promise.all([
        diffGetter && selected.changes.length ? diffGetter(selected.id, 0) : Promise.resolve([]),
        versionsGetter && selected.changes.length ? versionsGetter(selected.changes[0].chapter) : Promise.resolve([]),
      ])
      set({selected, diff, versions})
    } catch (cause) { set({error: String(cause)}) }
    finally { set({loading: false}) }
  },
  async refreshSelected() {
    const selected = get().selected
    if (selected) await get().select(selected.id)
  },
  async createManual(chapter, title, after) {
    const action = bridge().CreateProposal
    if (!action) { set({error: '桌面桥接尚未提供 CreateProposal。'}); return null }
    const projectIdAtStart = get().projectId
    set({busy: true, error: '', notice: ''})
    try {
      const proposal = await action({projectId: get().projectId, title, source: 'ManualRequest', changes: [{resourceType: 'ChapterText', resourceId: `chapter:${chapter}`, chapter, after}]})
      if (!sameProject(get().projectId, projectIdAtStart)) return null
      set({proposals: [proposal, ...get().proposals], selected: proposal, notice: '建议已创建，请先查看 Diff 与证据。'})
      await get().refreshSelected()
      return proposal
    } catch (cause) { set({error: String(cause)}); return null }
    finally { set({busy: false}) }
  },
  async createFromReview(chapter, scope, issueIndex, after) {
    const action = bridge().CreateProposalFromReviewIssue
    if (!action) { set({error: '桌面桥接尚未提供 ReviewIssue → Proposal。'}); return null }
    const projectIdAtStart = get().projectId
    set({busy: true, error: '', notice: ''})
    try {
      const proposal = await action(chapter, scope, issueIndex, after)
      if (!sameProject(get().projectId, projectIdAtStart)) return null
      set({proposals: [proposal, ...get().proposals], selected: proposal, notice: '已从 Core ReviewIssue 创建建议。'})
      await get().refreshSelected()
      return proposal
    } catch (cause) { set({error: String(cause)}); return null }
    finally { set({busy: false}) }
  },
  async accept() {
    const action = bridge().AcceptProposal
    const selected = get().selected
    if (!action || !selected) return
    const projectIdAtStart = get().projectId
    set({busy: true, error: '', notice: ''})
    try { const updated = await action(selected.id); if (!sameProject(get().projectId, projectIdAtStart)) return; set({selected: updated, proposals: get().proposals.map(item => item.id === updated.id ? updated : item), notice: '建议已接受；正文尚未修改。'}) }
    catch (cause) { set({error: String(cause)}) }
    finally { set({busy: false}) }
  },
  async reject() {
    const action = bridge().RejectProposal
    const selected = get().selected
    if (!action || !selected) return
    const projectIdAtStart = get().projectId
    set({busy: true, error: '', notice: ''})
    try { const updated = await action(selected.id); if (!sameProject(get().projectId, projectIdAtStart)) return; set({selected: updated, proposals: get().proposals.map(item => item.id === updated.id ? updated : item), notice: '建议已拒绝。'}) }
    catch (cause) { set({error: String(cause)}) }
    finally { set({busy: false}) }
  },
  async apply() {
    const action = bridge().ApplyProposal
    const selected = get().selected
    if (!action || !selected) return null
    const projectIdAtStart = get().projectId
    set({busy: true, error: '', notice: ''})
    try {
      const result = await action(selected.id)
      if (!sameProject(get().projectId, projectIdAtStart)) return null
      const getter = bridge().GetProposal
      const updated = getter ? await getter(selected.id) : selected
      set({selected: updated, proposals: get().proposals.map(item => item.id === updated.id ? updated : item), notice: result.message ?? '已应用到工作正文，等待 Sync。'})
      return result
    } catch (cause) { set({error: String(cause)}); return null }
    finally { set({busy: false}) }
  },
  async restoreVersion(id) {
    const action = bridge().RestoreVersion
    if (!action) { set({error: '桌面桥接尚未提供版本恢复。'}); return }
    const projectIdAtStart = get().projectId
    set({busy: true, error: '', notice: ''})
    try {
      await action(id)
      if (!sameProject(get().projectId, projectIdAtStart)) return
      set({notice: '版本已恢复到工作正文，等待 Core Sync。'})
      const statusGetter = bridge().CheckChapterRevisions
      if (statusGetter) {
        const status = await statusGetter()
        if (sameProject(status.projectId, projectIdAtStart) && sameProject(get().projectId, projectIdAtStart)) useRevisionStore.getState().acceptStatus(status)
      }
      await useEngineStore.getState().refreshRuntime()
      await get().refreshSelected()
    }
    catch (cause) { set({error: String(cause)}) }
    finally { set({busy: false}) }
  },
}))
