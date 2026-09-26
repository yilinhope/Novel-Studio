import { create } from 'zustand'
import { bridge, subscribeStudioEvent } from './services'
import { useStudio } from './store'
import type { ReadRequest, SimulationEvent, SimulationProfilePage, SimulationSourcesPage } from './types'

interface SimulationState {
  sources: SimulationSourcesPage | null
  profile: SimulationProfilePage | null
  loading: boolean
  running: boolean
  error: string
  event: SimulationEvent | null
  pendingEvent: SimulationEvent | null
  requestId: string
  load(): Promise<void>
  start(): Promise<void>
  importProfile(path?: string): Promise<void>
  cancel(): Promise<void>
  reset(): void
}

let nonce = 0
let sequence = 0
const sameProject = (left: string, right: string) => left.replaceAll('\\', '/').replace(/\/+$/, '').toLocaleLowerCase() === right.replaceAll('\\', '/').replace(/\/+$/, '').toLocaleLowerCase()
const requestFor = (project: NonNullable<ReturnType<typeof useStudio.getState>['project']>, requestId: string): ReadRequest => ({projectId: project.outputDir, generation: project.generation ?? 0, requestId, sequence: ++sequence, offset: 0, limit: 50})

export const useSimulationStore = create<SimulationState>((set, get) => ({
  sources: null, profile: null, loading: false, running: false, error: '', event: null, pendingEvent: null, requestId: '',
  async load() {
    const project = useStudio.getState().project
    const api = bridge()
    if (!project || !api.GetSimulationSources || !api.GetSimulationProfile) return
    const requestId = `simulation-${++nonce}`
    set({loading: true, error: '', requestId})
    const request = requestFor(project, requestId)
    try {
      const [sources, profile] = await Promise.all([api.GetSimulationSources(request), api.GetSimulationProfile(request)])
      const active = useStudio.getState().project
      if (!active || !sameProject(active.outputDir, project.outputDir) || get().requestId !== requestId) return
      set({sources, profile, loading: false, error: ''})
    } catch (error) {
      if (get().requestId === requestId) set({loading: false, error: String(error)})
    }
  },
  async start() {
    const project = useStudio.getState().project
    const api = bridge()
    if (!project || !api.StartSimulation || get().running) return
    const requestId = `simulation-run-${++nonce}`
    set({running: true, error: '', event: null, pendingEvent: null, requestId})
    try {
      await api.StartSimulation(requestFor(project, requestId))
      const active = useStudio.getState().project
      if (!active || !sameProject(active.outputDir, project.outputDir) || get().requestId !== requestId) return
      const pending = get().pendingEvent
      set({running: pending ? pending.state === 'running' : true, event: pending, pendingEvent: null})
    } catch (error) {
      if (get().requestId === requestId) set({running: false, error: String(error)})
    }
  },
  async importProfile(path) {
    const project = useStudio.getState().project
    const api = bridge()
    if (!project || !api.ImportSimulationProfile || get().running) return
    const selected = path ?? (api.SelectSimulationProfile ? await api.SelectSimulationProfile() : '')
    if (!selected) return
    const requestId = `simulation-import-${++nonce}`
    set({running: true, error: '', event: null, pendingEvent: null, requestId})
    try {
      await api.ImportSimulationProfile({projectId: project.outputDir, generation: project.generation ?? 0, requestId, path: selected})
    } catch (error) {
      if (get().requestId === requestId) set({running: false, error: String(error)})
    }
  },
  async cancel() {
    const action = bridge().CancelSimulation
    if (!action) return
    try { await action() } catch (error) { set({error: String(error)}) }
  },
  reset() { nonce++; set({sources: null, profile: null, loading: false, running: false, error: '', event: null, pendingEvent: null, requestId: ''}) },
}))

let subscribed = false
export function ensureSimulationEvents() {
  if (subscribed) return
  subscribed = true
  subscribeStudioEvent<SimulationEvent>('studio:simulation-event', event => {
    const project = useStudio.getState().project
    const state = useSimulationStore.getState()
    if (!project || !sameProject(event.projectId, project.outputDir) || (state.requestId && event.requestId && state.requestId !== event.requestId)) return
    if (event.state === 'running' && !state.requestId) return
    if (!state.requestId || event.generation === 0) {
      useSimulationStore.setState({pendingEvent: event})
      return
    }
    useSimulationStore.setState({event, running: event.state === 'running', error: event.error ?? (event.state === 'error' ? event.message : '')})
    if (event.state === 'completed') void useSimulationStore.getState().load()
  })
}
