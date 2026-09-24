import { create } from 'zustand'
import { bridge, subscribeStudioEvent } from './services'
import type { ImportOptions, ImportStatus } from './types'

interface ImportState {
  projectId: string; generation: number; status: ImportStatus | null; busy: boolean; error: string; sequence: number
  start(options: ImportOptions): Promise<void>; cancel(): Promise<void>; refresh(): Promise<void>
}
const same = (left: string, right: string) => left.replaceAll('\\', '/').replace(/\/+$/, '').toLowerCase() === right.replaceAll('\\', '/').replace(/\/+$/, '').toLowerCase()
let bound = false
export const useImportStore = create<ImportState>((set, get) => ({
  projectId: '', generation: 0, status: null, busy: false, error: '', sequence: 0,
  async start(options) {
    if (!bridge().StartImport || get().busy) return
    const ticket = get().sequence + 1
    set({busy: true, error: '', sequence: ticket})
    if (!bound) {
      bound = true
      subscribeStudioEvent<ImportStatus>('studio:import-event', event => {
        const current = get()
        if (!current.projectId || !same(event.projectId, current.projectId) || event.generation < current.generation) return
        set({status: event, busy: event.stage !== 'done' && !event.error})
      })
    }
    try {
      const ack = await bridge().StartImport!(options)
      if (ticket !== get().sequence) return
      set({projectId: ack.projectId, generation: ack.generation, status: {projectId: ack.projectId, generation: ack.generation, active: true, stage: 'starting', current: 0, total: 0, message: '', continued: false}})
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
