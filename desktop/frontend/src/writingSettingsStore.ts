import { create } from 'zustand'
import { bridge } from './services'
import { useStudio } from './store'
import type { ReadRequest, RuleFile, RuleMutationRequest, RulesWorkspace, StyleMutationRequest, StyleState } from './types'

interface WritingSettingsState {
  rules: RulesWorkspace | null
  style: StyleState | null
  loading: boolean
  saving: boolean
  error: string
  tab: 'rules' | 'style'
  selectedScope: 'global' | 'project' | 'effective'
  selectedRule: RuleFile | null
  dirty: boolean
  load(): Promise<void>
  selectRule(scope: 'global' | 'project', name: string): Promise<void>
  setDraft(content: string): void
  createRule(scope: 'global' | 'project', name: string, content: string): Promise<void>
  saveRule(name?: string): Promise<void>
  deleteRule(scope: 'global' | 'project', name: string): Promise<void>
  renameRule(scope: 'global' | 'project', name: string, newName: string): Promise<void>
  selectStyle(style: string): Promise<void>
  saveAsset(scope: 'global' | 'project', name: string, content: string): Promise<void>
  deleteAsset(scope: 'global' | 'project', name: string): Promise<void>
  reset(): void
}

let sequence = 0
let loadTicket = 0
const request = () => {
  const project = useStudio.getState().project
  if (!project) throw new Error('请先打开小说项目')
  return {project, request: {projectId: project.outputDir, generation: project.generation ?? 0, requestId: `writing-${Date.now()}`, sequence: ++sequence, offset: 0, limit: 50} as ReadRequest}
}
const sameProject = (a: string, b: string) => a.replaceAll('\\', '/').toLocaleLowerCase() === b.replaceAll('\\', '/').toLocaleLowerCase()

export const useWritingSettingsStore = create<WritingSettingsState>((set, get) => ({
  rules: null, style: null, loading: false, saving: false, error: '', tab: 'rules', selectedScope: 'global', selectedRule: null, dirty: false,
  async load() {
    const ticket = ++loadTicket
    const api = bridge()
    if (!api.GetRulesWorkspace || !api.GetStyleState) return
    const {project, request: read} = request()
    set({loading: true, error: ''})
    try {
      const [rules, style] = await Promise.all([api.GetRulesWorkspace(read), api.GetStyleState(read)])
      const active = useStudio.getState().project
      if (ticket !== loadTicket || !active || !sameProject(active.outputDir, project.outputDir)) return
      set({rules, style, loading: false, error: ''})
    } catch (error) { if (ticket === loadTicket) set({loading: false, error: String(error)}) }
  },
  async selectRule(scope, name) {
    const api = bridge()
    if (!api.GetRule) return
    const {project, request: read} = request()
    set({loading: true, error: '', selectedScope: scope})
    try {
      const rule = await api.GetRule(read, scope, name)
      const active = useStudio.getState().project
      if (active && sameProject(active.outputDir, project.outputDir)) set({selectedRule: rule, dirty: false, loading: false})
    } catch (error) { set({loading: false, error: String(error)}) }
  },
  setDraft(content) { const selected = get().selectedRule; if (selected) set({selectedRule: {...selected, content}, dirty: content !== selected.content, error: ''}) },
  async saveRule(name) {
    const selected = get().selectedRule
    const api = bridge()
    if (!selected || !api.SaveRule) return
    const {project, request: read} = request()
    const body: RuleMutationRequest = {...read, scope: selected.scope, name: name ?? selected.name, content: selected.content ?? ''}
    set({saving: true, error: ''})
    try { const rules = await api.SaveRule(body); if (sameProject(useStudio.getState().project?.outputDir ?? '', project.outputDir)) set({rules, saving: false, dirty: false}) }
    catch (error) { set({saving: false, error: String(error)}) }
  },
  async createRule(scope, name, content) {
    const api = bridge()
    if (!api.SaveRule) return
    const {project, request: read} = request(); set({saving: true, error: ''})
    try {
      const rules = await api.SaveRule({...read, scope, name, content})
      if (sameProject(useStudio.getState().project?.outputDir ?? '', project.outputDir)) set({rules, saving: false, selectedRule: {name, scope, path: '', sizeBytes: content.length, modifiedAt: new Date().toISOString(), content}, selectedScope: scope, dirty: false})
    } catch (error) { set({saving: false, error: String(error)}) }
  },
  async deleteRule(scope, name) {
    const api = bridge(); if (!api.DeleteRule) return
    const {project, request: read} = request(); set({saving: true, error: ''})
    try { const rules = await api.DeleteRule({...read, scope, name}); if (sameProject(useStudio.getState().project?.outputDir ?? '', project.outputDir)) set({rules, selectedRule: null, saving: false, dirty: false}) }
    catch (error) { set({saving: false, error: String(error)}) }
  },
  async renameRule(scope, name, newName) {
    const api = bridge(); if (!api.RenameRule) return
    const {project, request: read} = request(); set({saving: true, error: ''})
    try { const rules = await api.RenameRule({...read, scope, name, newName}); if (sameProject(useStudio.getState().project?.outputDir ?? '', project.outputDir)) set({rules, saving: false, dirty: false}) }
    catch (error) { set({saving: false, error: String(error)}) }
  },
  async selectStyle(style) {
    const api = bridge(); if (!api.SaveStyleSelection) return
    const {project, request: read} = request(); set({saving: true, error: ''})
    try { const state = await api.SaveStyleSelection({...read, name: style}); if (sameProject(useStudio.getState().project?.outputDir ?? '', project.outputDir)) set({style: state, saving: false}) }
    catch (error) { set({saving: false, error: String(error)}) }
  },
  async saveAsset(scope, name, content) {
    const api = bridge(); if (!api.SaveStyleAsset) return
    const {project, request: read} = request(); set({saving: true, error: ''})
    try { const state = await api.SaveStyleAsset({...read, scope, name, content}); if (sameProject(useStudio.getState().project?.outputDir ?? '', project.outputDir)) set({style: state, saving: false}) }
    catch (error) { set({saving: false, error: String(error)}) }
  },
  async deleteAsset(scope, name) {
    const api = bridge(); if (!api.DeleteStyleAsset) return
    const {project, request: read} = request(); set({saving: true, error: ''})
    try { const state = await api.DeleteStyleAsset({...read, scope, name}); if (sameProject(useStudio.getState().project?.outputDir ?? '', project.outputDir)) set({style: state, saving: false}) }
    catch (error) { set({saving: false, error: String(error)}) }
  },
  reset() { loadTicket++; set({rules: null, style: null, loading: false, saving: false, error: '', selectedRule: null, dirty: false}) },
}))
