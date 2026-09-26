import { create } from 'zustand'
import { bridge } from './services'
import type {
  CastPage, CharacterPage, ForeshadowPage, LayeredChapterPage, LayeredOutlinePage, OutlinePage,
  ParitySection, ReadRequest, RelationshipPage, SnapshotPage, SnapshotScope, StateChangePage, StoryCompass,
  StoryPremise, StorySummaryPage, TimelinePage, WorldRulePage,
} from './types'
import { useStudio } from './store'

type ParityData = StoryPremise | CharacterPage | WorldRulePage | OutlinePage | LayeredOutlinePage | LayeredChapterPage | StoryCompass | StorySummaryPage | TimelinePage | ForeshadowPage | RelationshipPage | StateChangePage | SnapshotPage | CastPage
type ParityState = {
  section: ParitySection
  outlineMode: 'layered' | 'flat'
  layeredOutline: LayeredOutlinePage | null
  snapshotScopes: SnapshotScope[]
  summaryScope: 'chapter' | 'arc' | 'volume'
  data: ParityData | null
  loading: boolean
  error: string
  offset: number
  limit: number
  selectedVolume: number
  selectedArc: number
  selectedSnapshotVolume: number
  selectedSnapshotArc: number
  selectSection(section: ParitySection): Promise<void>
  selectOutlineMode(mode: 'layered' | 'flat'): Promise<void>
  page(offset: number): Promise<void>
  selectArc(volume: number, arc: number): Promise<void>
  selectSnapshotScope(volume: number, arc: number): Promise<void>
  selectSummaryScope(scope: 'chapter' | 'arc' | 'volume'): Promise<void>
  reset(): void
}

let requestSequence = 0
let requestNonce = 0
const PAGE_SIZE = 50
const normalizeProject = (value: string) => value.replaceAll('\\', '/').replace(/\/+$/, '').toLocaleLowerCase()

async function readSection(section: ParitySection, request: ReadRequest, volume: number, arc: number, outlineMode: 'layered' | 'flat', summaryScope: 'chapter' | 'arc' | 'volume'): Promise<ParityData> {
  const api = bridge()
  switch (section) {
    case 'premise': if (api.GetStoryPremise) return api.GetStoryPremise(request); break
    case 'characters': if (api.GetStoryCharacters) return api.GetStoryCharacters(request); break
    case 'world': if (api.GetStoryWorldRules) return api.GetStoryWorldRules(request); break
    case 'outline':
      if (outlineMode === 'layered' && api.GetStoryLayeredOutline) return api.GetStoryLayeredOutline(request)
      if (outlineMode === 'flat' && api.GetStoryOutline) return api.GetStoryOutline(request)
      break
    case 'compass': if (api.GetStoryCompass) return api.GetStoryCompass(request); break
    case 'summaries': if (api.GetStorySummaries) return api.GetStorySummaries(request, summaryScope); break
    case 'timeline': if (api.GetContinuityTimeline) return api.GetContinuityTimeline(request); break
    case 'foreshadow': if (api.GetContinuityForeshadow) return api.GetContinuityForeshadow(request); break
    case 'relationships': if (api.GetContinuityRelationships) return api.GetContinuityRelationships(request); break
    case 'states': if (api.GetContinuityStateChanges) return api.GetContinuityStateChanges(request); break
    case 'snapshots': if (api.GetContinuitySnapshots) return api.GetContinuitySnapshots(request, volume, arc); break
    case 'cast': if (api.GetContinuityCast) return api.GetContinuityCast(request); break
  }
  throw new Error('当前桌面桥接尚未提供此 Story/Continuity 读取能力。')
}

export const useParityStore = create<ParityState>((set, get) => ({
  section: 'premise', outlineMode: 'layered', layeredOutline: null, snapshotScopes: [], summaryScope: 'chapter', data: null, loading: false, error: '', offset: 0, limit: PAGE_SIZE, selectedVolume: 0, selectedArc: 0, selectedSnapshotVolume: 0, selectedSnapshotArc: 0,
  async selectSection(section) {
    if (!useStudio.getState().project) return
    requestNonce++
    set({section, data: null, layeredOutline: null, snapshotScopes: [], loading: true, error: '', offset: 0, selectedVolume: 0, selectedArc: 0, selectedSnapshotVolume: 0, selectedSnapshotArc: 0})
    const state = useStudio.getState()
    const project = state.project!
    const sequence = ++requestSequence
    const request: ReadRequest = {projectId: project.outputDir, generation: project.generation ?? 0, requestId: `parity-${requestNonce}`, sequence, offset: 0, limit: PAGE_SIZE}
    const nonce = requestNonce
    try {
      const data = await readSection(section, request, 0, 0, get().outlineMode, get().summaryScope)
      const active = useStudio.getState().project
      if (nonce !== requestNonce || normalizeProject(active?.outputDir ?? '') !== normalizeProject(project.outputDir)) return
      set({data, layeredOutline: section === 'outline' && get().outlineMode === 'layered' ? data as LayeredOutlinePage : null, snapshotScopes: section === 'snapshots' ? (data as SnapshotPage).scopes ?? [] : [], loading: false, error: ''})
    } catch (error) {
      if (nonce === requestNonce && normalizeProject(useStudio.getState().project?.outputDir ?? '') === normalizeProject(project.outputDir)) set({loading: false, error: String(error)})
    }
  },
  async selectOutlineMode(mode) {
    set({outlineMode: mode, layeredOutline: null, selectedVolume: 0, selectedArc: 0})
    if (get().section !== 'outline') return
    const project = useStudio.getState().project
    if (!project) return
    requestNonce++
    const nonce = requestNonce
    const sequence = ++requestSequence
    const request: ReadRequest = {projectId: project.outputDir, generation: project.generation ?? 0, requestId: `parity-${nonce}`, sequence, offset: 0, limit: PAGE_SIZE}
    set({data: null, loading: true, error: '', offset: 0})
    try {
      const data = await readSection('outline', request, 0, 0, mode, get().summaryScope)
      if (nonce !== requestNonce || normalizeProject(useStudio.getState().project?.outputDir ?? '') !== normalizeProject(project.outputDir)) return
      set({data, layeredOutline: mode === 'layered' ? data as LayeredOutlinePage : null, loading: false, error: ''})
    } catch (error) {
      if (nonce === requestNonce && normalizeProject(useStudio.getState().project?.outputDir ?? '') === normalizeProject(project.outputDir)) set({loading: false, error: String(error)})
    }
  },
  async page(offset) {
    const {section, selectedVolume, selectedArc, summaryScope, outlineMode, selectedSnapshotVolume, selectedSnapshotArc} = get()
    const state = useStudio.getState()
    const project = state.project
    if (!project) return
    requestNonce++
    const nonce = requestNonce
    const sequence = ++requestSequence
    const request: ReadRequest = {projectId: project.outputDir, generation: project.generation ?? 0, requestId: `parity-${nonce}`, sequence, offset, limit: PAGE_SIZE}
    set({loading: true, error: ''})
    try {
      const api = bridge()
      let data: ParityData
      if (section === 'outline' && selectedVolume > 0 && selectedArc > 0 && api.GetStoryLayeredChapters) data = await api.GetStoryLayeredChapters(request, selectedVolume, selectedArc)
      else if (section === 'summaries' && api.GetStorySummaries) data = await api.GetStorySummaries(request, summaryScope)
      else data = await readSection(section, request, section === 'snapshots' ? selectedSnapshotVolume : selectedVolume, section === 'snapshots' ? selectedSnapshotArc : selectedArc, get().outlineMode, summaryScope)
      const active = useStudio.getState().project
      if (nonce !== requestNonce || normalizeProject(active?.outputDir ?? '') !== normalizeProject(project.outputDir)) return
      const shouldReplaceLayeredOutline = section === 'outline' && outlineMode === 'layered' && selectedVolume === 0 && selectedArc === 0
      set({data, layeredOutline: shouldReplaceLayeredOutline ? data as LayeredOutlinePage : get().layeredOutline, snapshotScopes: section === 'snapshots' ? (data as SnapshotPage).scopes ?? get().snapshotScopes : get().snapshotScopes, loading: false, error: '', offset})
    } catch (error) {
      if (nonce === requestNonce && normalizeProject(useStudio.getState().project?.outputDir ?? '') === normalizeProject(project.outputDir)) set({loading: false, error: String(error)})
    }
  },
  async selectArc(volume, arc) {
    const project = useStudio.getState().project
    const api = bridge()
    if (!project || !api.GetStoryLayeredChapters) return
    requestNonce++
    const nonce = requestNonce
    const sequence = ++requestSequence
    const request: ReadRequest = {projectId: project.outputDir, generation: project.generation ?? 0, requestId: `parity-${nonce}`, sequence, offset: 0, limit: PAGE_SIZE}
    set({loading: true, error: '', selectedVolume: volume, selectedArc: arc, offset: 0})
    try {
      const data = await api.GetStoryLayeredChapters(request, volume, arc)
      const active = useStudio.getState().project
      if (nonce !== requestNonce || normalizeProject(active?.outputDir ?? '') !== normalizeProject(project.outputDir)) return
      set({data, loading: false, error: ''})
    } catch (error) {
      if (nonce === requestNonce && normalizeProject(useStudio.getState().project?.outputDir ?? '') === normalizeProject(project.outputDir)) set({loading: false, error: String(error)})
    }
  },
  async selectSummaryScope(scope) {
    set({summaryScope: scope, offset: 0})
    const state = useStudio.getState()
    const project = state.project
    const api = bridge()
    if (!project || !api.GetStorySummaries) return
    requestNonce++
    const nonce = requestNonce
    const sequence = ++requestSequence
    const request: ReadRequest = {projectId: project.outputDir, generation: project.generation ?? 0, requestId: `parity-${nonce}`, sequence, offset: 0, limit: PAGE_SIZE}
    set({loading: true, error: ''})
    try {
      const data = await api.GetStorySummaries(request, scope)
      if (nonce !== requestNonce || normalizeProject(useStudio.getState().project?.outputDir ?? '') !== normalizeProject(project.outputDir)) return
      set({data, loading: false, error: ''})
    } catch (error) {
      if (nonce === requestNonce && normalizeProject(useStudio.getState().project?.outputDir ?? '') === normalizeProject(project.outputDir)) set({loading: false, error: String(error)})
    }
  },
  async selectSnapshotScope(volume, arc) {
    const project = useStudio.getState().project
    const api = bridge()
    if (!project || !api.GetContinuitySnapshots) return
    requestNonce++
    const nonce = requestNonce
    const sequence = ++requestSequence
    const request: ReadRequest = {projectId: project.outputDir, generation: project.generation ?? 0, requestId: `parity-${nonce}`, sequence, offset: 0, limit: PAGE_SIZE}
    set({loading: true, error: '', selectedSnapshotVolume: volume, selectedSnapshotArc: arc, offset: 0})
    try {
      const data = await api.GetContinuitySnapshots(request, volume, arc)
      if (nonce !== requestNonce || normalizeProject(useStudio.getState().project?.outputDir ?? '') !== normalizeProject(project.outputDir)) return
      set({data, snapshotScopes: data.scopes ?? get().snapshotScopes, loading: false, error: ''})
    } catch (error) {
      if (nonce === requestNonce && normalizeProject(useStudio.getState().project?.outputDir ?? '') === normalizeProject(project.outputDir)) set({loading: false, error: String(error)})
    }
  },
  reset() { requestNonce++; set({data: null, layeredOutline: null, snapshotScopes: [], loading: false, error: '', offset: 0, selectedVolume: 0, selectedArc: 0, selectedSnapshotVolume: 0, selectedSnapshotArc: 0, outlineMode: 'layered', summaryScope: 'chapter'}) },
}))
