import { create } from 'zustand'
import { bridge, subscribeStudioEvent } from './services'
import type { CoCreateEvent, CoCreateMessage, CoCreateRecovery, CoCreateStart, CreateEvent, CreateProjectRequest, OperationAck } from './types'

interface CreateState {
  mode: 'quick' | 'outline' | 'cocreate'
  projectRoot: string
  prompt: string
  stage: boolean
  ack: OperationAck | CoCreateStart | null
  requestId: string
  event: CreateEvent | null
  coCreate: CoCreateEvent | null
  pendingCreateEvent: CreateEvent | null
  pendingCoCreateEvent: CoCreateEvent | null
  recovery: CoCreateRecovery | null
  history: CoCreateMessage[]
  busy: boolean
  previewing: boolean
  previewText: string
  error: string
  setMode(mode: CreateState['mode']): void
  setProjectRoot(root: string): void
  setStage(stage: boolean): void
  setPrompt(prompt: string): void
  start(): Promise<void>
  send(message: string): Promise<void>
  complete(): Promise<void>
  cancel(): Promise<void>
  loadRecovery(): Promise<void>
  resumeRecovery(): Promise<void>
  previewOutline(): Promise<void>
}

let sequence = 0
let unsubscribe: (() => void) | undefined
const same = (left: string, right: string) => left.replaceAll('\\', '/').replace(/\/+$/, '').toLowerCase() === right.replaceAll('\\', '/').replace(/\/+$/, '').toLowerCase()
const newRequestID = () => globalThis.crypto?.randomUUID?.() ?? `studio-${Date.now()}-${Math.random().toString(16).slice(2)}`
const createTerminal = (state: string) => state === 'completed' || state === 'error'
const coCreateTerminal = (state: string) => state === 'reply' || state === 'ready' || state === 'error' || state === 'cancelled'

function matchesRequest(event: {projectId: string; requestId?: string}, current: CreateState) {
  if (event.requestId && event.requestId === current.requestId) return true
  return !event.requestId && !!current.ack && same(event.projectId, current.ack.projectId)
}

function bindEvents(set: (value: Partial<CreateState>) => void, get: () => CreateState) {
  if (unsubscribe) return
  ++sequence
  const createUnsubscribe = subscribeStudioEvent<CreateEvent>('studio:create-event', event => {
    const current = get()
    if (!matchesRequest(event, current) || (current.ack && event.generation < (current.ack.generation ?? 0))) return
    if (!current.ack) {
      set({pendingCreateEvent: event, busy: !createTerminal(event.state), error: event.error ?? ''})
      return
    }
    set({event, busy: event.state === 'started' || event.state === 'running', error: event.error ?? ''})
  })
  const coCreateUnsubscribe = subscribeStudioEvent<CoCreateEvent>('studio:cocreate-event', event => {
    const current = get()
    if (!matchesRequest(event, current) || (current.ack && event.generation < (current.ack.generation ?? 0))) return
    if (!current.ack) {
      set({pendingCoCreateEvent: event, busy: !coCreateTerminal(event.state), error: event.error ?? ''})
      return
    }
    if (event.history) set({history: [...event.history, ...(event.reply ? [{role: 'assistant', content: event.reply} as CoCreateMessage] : [])]})
    set({coCreate: event, busy: event.state !== 'reply' && event.state !== 'error', error: event.error ?? ''})
  })
  unsubscribe = () => { createUnsubscribe(); coCreateUnsubscribe() }
}

export const useCreateProjectStore = create<CreateState>((set, get) => ({
  mode: 'quick', projectRoot: '', prompt: '', stage: false, ack: null, requestId: '', event: null, coCreate: null, pendingCreateEvent: null, pendingCoCreateEvent: null, recovery: null, history: [], busy: false, previewing: false, previewText: '', error: '',
  setMode(mode) { set({mode, error: '', event: null, coCreate: null, recovery: null, pendingCreateEvent: null, pendingCoCreateEvent: null, history: []}) },
  setProjectRoot(projectRoot) { set({projectRoot}) },
  setStage(stage) { set({stage}) },
  setPrompt(prompt) { set({prompt}) },
  async start() {
    const current = get()
    if (current.busy) return
    const api = bridge()
    bindEvents(set, get)
    const ticket = ++sequence
    const requestId = newRequestID()
    set({busy: true, error: '', ack: null, requestId, event: null, coCreate: null, pendingCreateEvent: null, pendingCoCreateEvent: null, history: current.mode === 'cocreate' ? [{role: 'user', content: current.prompt.trim()}] : []})
    try {
      if (current.mode === 'cocreate') {
        if (!api.StartCoCreate) throw new Error('桌面桥接尚未提供 StartCoCreate')
        const ack = await api.StartCoCreate(current.projectRoot, current.prompt, current.stage, requestId)
        if (ticket !== sequence) return
        const pending = get().pendingCoCreateEvent
        set({ack, pendingCoCreateEvent: null, coCreate: pending ?? get().coCreate, busy: pending ? !coCreateTerminal(pending.state) : true, error: pending?.error ?? ''})
        if (pending?.history) set({history: [...pending.history, ...(pending.reply ? [{role: 'assistant' as const, content: pending.reply}] : [])]})
      } else {
        if (!api.StartQuickStart) throw new Error('桌面桥接尚未提供 StartQuickStart')
        const ack = await api.StartQuickStart({projectRoot: current.projectRoot, prompt: current.prompt, mode: current.mode, requestId})
        if (ticket !== sequence) return
        const pending = get().pendingCreateEvent
        set({ack, pendingCreateEvent: null, event: pending ?? get().event, busy: pending ? !createTerminal(pending.state) : true, error: pending?.error ?? ''})
      }
    } catch (error) { if (ticket === sequence) set({busy: false, error: String(error)}) }
  },
  async send(message) {
    const current = get()
    const api = bridge()
    if (current.busy || !current.ack || !api.SendCoCreate) return
    const content = message.trim()
    if (!content) return
    const history = [...current.history, {role: 'user' as const, content}]
    set({busy: true, error: '', history})
    try {
      await api.SendCoCreate(current.projectRoot, current.ack.projectId, current.stage, history)
    } catch (error) { set({busy: false, error: String(error)}) }
  },
  async complete() {
    const current = get()
    const api = bridge()
    if (!current.coCreate?.draft || !api.CompleteCoCreate) return
    set({busy: true, error: ''})
    try { await api.CompleteCoCreate(current.stage, current.coCreate.draft) }
    catch (error) { set({busy: false, error: String(error)}) }
  },
  async cancel() {
    const current = get()
    const api = bridge()
    if (!api.CancelCoCreate) return
    try { await api.CancelCoCreate(current.stage); set({busy: false}) }
    catch (error) { set({error: String(error)}) }
  },
  async loadRecovery() {
    const api = bridge()
    if (!api.GetCoCreateRecovery) return
    try {
      const recovery = await api.GetCoCreateRecovery()
      const history = recovery.history ?? []
      const last = history[history.length - 1]
      set({recovery, history, coCreate: recovery.exists ? {projectId: recovery.projectId, generation: recovery.generation ?? 0, requestId: '', state: 'recovery', reply: last?.role === 'assistant' ? last.content : '', draft: recovery.draft, ready: recovery.ready, suggestions: recovery.suggestions, history, error: recovery.error ?? ''} : null})
    }
    catch (error) { set({error: String(error)}) }
  },
  async resumeRecovery() {
    const current = get()
    const api = bridge()
    if (!current.recovery?.exists || !api.ResumeCoCreate || current.busy) return
    const requestId = newRequestID()
    set({busy: true, error: '', ack: null, requestId, pendingCoCreateEvent: null})
    try {
      const ack = await api.ResumeCoCreate(current.recovery.mode === 'stage', current.history, requestId)
      const pending = get().pendingCoCreateEvent
      set({ack, pendingCoCreateEvent: null, coCreate: pending ?? get().coCreate, busy: pending ? !coCreateTerminal(pending.state) : true, error: pending?.error ?? ''})
      if (pending?.history) set({history: [...pending.history, ...(pending.reply ? [{role: 'assistant' as const, content: pending.reply}] : [])]})
    } catch (error) { set({busy: false, error: String(error)}) }
  },
  async previewOutline() {
    const api = bridge()
    if (!api.PreviewOutline) return
    set({previewing: true, error: ''})
    try { set({previewText: await api.PreviewOutline(get().prompt)}) }
    catch (error) { set({error: String(error), previewText: ''}) }
    finally { set({previewing: false}) }
  },
}))
