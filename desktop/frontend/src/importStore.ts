import { create } from 'zustand'
import { bridge, subscribeStudioEvent } from './services'
import type { ImportOptions, ImportStatus } from './types'

interface ImportState {
  projectId: string; generation: number; requestId: string; status: ImportStatus | null; pendingStatus: ImportStatus | null; busy: boolean; error: string; sequence: number
  start(options: ImportOptions): Promise<void>; cancel(): Promise<void>; refresh(): Promise<void>
}
const same = (left: string, right: string) => left.replaceAll('\\', '/').replace(/\/+$/, '').toLowerCase() === right.replaceAll('\\', '/').replace(/\/+$/, '').toLowerCase()
const newRequestID = () => globalThis.crypto?.randomUUID?.() ?? `studio-import-${Date.now()}-${Math.random().toString(16).slice(2)}`
const terminal = (status: ImportStatus) => !status.active && (status.stage === 'done' || !!status.error)
let bound = false
export const useImportStore = create<ImportState>((set, get) => ({
  projectId: '', generation: 0, requestId: '', status: null, pendingStatus: null, busy: false, error: '', sequence: 0,
  async start(options) {
    if (!bridge().StartImport || get().busy) return
    const ticket = get().sequence + 1
    const requestId = newRequestID()
    set({busy: true, error: '', sequence: ticket, requestId, projectId: '', generation: 0, status: null, pendingStatus: null})
    if (!bound) {
      bound = true
      subscribeStudioEvent<ImportStatus>('studio:import-event', event => {
        const current = get()
        if (!current.requestId || event.requestId !== current.requestId) return
        if (!current.projectId) {
          set({pendingStatus: event, busy: !terminal(event), error: event.error ?? ''})
          return
        }
        if (!same(event.projectId, current.projectId) || event.generation < current.generation) return
        set({status: event, busy: !terminal(event), error: event.error ?? ''})
      })
    }
    try {
      const ack = await bridge().StartImport!({...options, requestId})
      if (ticket !== get().sequence) return
      const pending = get().pendingStatus
      const matchingPending = pending && pending.requestId === (ack.requestId ?? requestId) && same(pending.projectId, ack.projectId) && pending.generation >= ack.generation ? pending : null
      set({requestId: ack.requestId ?? requestId, projectId: ack.projectId, generation: ack.generation, status: matchingPending ?? {projectId: ack.projectId, generation: ack.generation, requestId: ack.requestId ?? requestId, active: true, stage: 'starting', current: 0, total: 0, message: '', continued: false}, pendingStatus: null, busy: matchingPending ? !terminal(matchingPending) : true})
      await get().refresh()
    } catch (error) { if (ticket === get().sequence) set({busy: false, error: String(error)}) }
  },
  async cancel() {
    if (!bridge().CancelImport) return
    try { await bridge().CancelImport!() } catch (error) { set({error: String(error)}) }
  },
  async refresh() {
    if (!bridge().GetImportStatus) return
    try { const status = await bridge().GetImportStatus!(); if (!get().projectId || same(status.projectId, get().projectId)) set({status, busy: status.active}) }
    catch (error) { set({error: String(error)}) }
  },
}))
