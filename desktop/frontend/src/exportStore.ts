import { create } from 'zustand'
import { bridge } from './services'
import type { ExportOptions, ExportResult } from './types'
interface ExportState { busy: boolean; error: string; result: ExportResult | null; exportProject(options: ExportOptions): Promise<void> }
let request = 0
export const useExportStore = create<ExportState>((set) => ({
  busy: false, error: '', result: null,
  async exportProject(options) { const api = bridge(); if (!api.ExportProject) return; const ticket = ++request; set({busy: true, error: '', result: null}); try { const result = await api.ExportProject(options); if (ticket === request) set({result}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
}))
