import { create } from 'zustand'
import { bridge } from './services'
import type { RevisionStatus } from './types'

interface RevisionStore {
  projectId: string
  generation: number
  status: RevisionStatus
  checking: boolean
  error: string
  selectProject(projectId: string, getStatus: () => Promise<RevisionStatus>): Promise<void>
  checkChapterRevisions(): Promise<void>
  acceptStatus(status: RevisionStatus): void
}

const emptyStatus = (projectId = ''): RevisionStatus => ({
  projectId, state: 'unknown', hasUnsynced: false, chapters: [],
})
const normalizePath = (path: string) => path.replaceAll('\\', '/').replace(/\/+$/, '').toLocaleLowerCase()

export function revisionsAllowWriting(projectId: string, status: RevisionStatus, checking: boolean, error: string) {
  return !checking && !error && normalizePath(projectId) !== ''
    && normalizePath(status.projectId) === normalizePath(projectId)
    && status.state === 'synced' && !status.hasUnsynced
}

let generation = 0

export const useRevisionStore = create<RevisionStore>((set, get) => ({
  projectId: '', generation: 0, status: emptyStatus(), checking: false, error: '',
  async selectProject(projectId, getStatus) {
    const ticket = ++generation
    set({projectId, generation: ticket, status: emptyStatus(projectId), checking: false, error: ''})
    try {
      const cached = await getStatus()
      const current = get()
      if (ticket !== generation || normalizePath(current.projectId) !== normalizePath(projectId)) return
      if (normalizePath(cached.projectId) === normalizePath(projectId)) set({status: cached})
      await get().checkChapterRevisions()
    } catch (error) {
      if (ticket === generation) set({error: String(error)})
    }
  },
  async checkChapterRevisions() {
    const current = get()
    if (!current.projectId) return
    const ticket = current.generation
    set({checking: true, error: ''})
    try {
      const status = await bridge().CheckChapterRevisions()
      const latest = get()
      if (ticket !== generation || ticket !== latest.generation || normalizePath(latest.projectId) !== normalizePath(status.projectId)) return
      set({status})
    } catch (error) {
      if (ticket === generation && ticket === get().generation) set({error: String(error)})
    } finally {
      if (ticket === generation && ticket === get().generation) set({checking: false})
    }
  },
  acceptStatus(status) {
    const current = get()
    if (normalizePath(status.projectId) !== normalizePath(current.projectId)) return
    set({status, error: status.error ?? ''})
  },
}))
