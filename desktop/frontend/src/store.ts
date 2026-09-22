import { create } from 'zustand'
import { bridge } from './services'
import { useEngineStore } from './engineStore'
import type { Chapter, Project } from './types'

interface StudioState {
  project: Project | null; chapter: Chapter | null; busy: boolean; error: string
  view: 'overview' | 'chapter' | 'runtime'; chapterLoading: boolean
  open(path?: string): Promise<void>; read(number: number): Promise<void>; overview(): void; runtime(): void
}
let request = 0
export const useStudio = create<StudioState>((set, get) => ({
  project: null, chapter: null, busy: false, error: '', view: 'overview', chapterLoading: false,
  async open(path) {
    if (get().busy) return
    const ticket = ++request
    set({busy: true, error: '', chapterLoading: false})
    try {
      const api = bridge()
      const selected = path ?? await api.SelectProjectDirectory()
      if (!selected) return
      const project = await api.OpenProject(selected)
      if (ticket === request) {
        set({project, chapter: null, view: 'overview'})
        void useEngineStore.getState().selectProject(project.overview.path, api.GetRuntimeState)
      }
    } catch (error) { if (ticket === request) set({error: String(error)}) }
    finally { if (ticket === request) set({busy: false}) }
  },
  async read(number) {
    if (get().busy) return
    const ticket = ++request
    set({chapterLoading: true, error: '', view: 'chapter', chapter: null})
    try {
      const chapter = await bridge().GetChapter(number)
      if (ticket === request) set({chapter})
    } catch (error) { if (ticket === request) set({error: String(error)}) }
    finally { if (ticket === request) set({chapterLoading: false}) }
  },
  overview() { ++request; set({view:'overview', chapterLoading:false, error:''}) },
  runtime() { ++request; set({view:'runtime', chapterLoading:false, error:''}) },
}))
