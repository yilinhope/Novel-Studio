import { create } from 'zustand'
import { bridge } from './services'
import { setChapterCommitHandler, useEngineStore } from './engineStore'
import { useRevisionStore } from './revisionStore'
import type { Chapter, Project, StudioEngineEvent } from './types'

interface StudioState {
  project: Project | null; chapter: Chapter | null; busy: boolean; error: string
  view: 'overview' | 'chapter' | 'runtime'; chapterLoading: boolean
  draftContent: string; savedContent: string; dirty: boolean; saveBusy: boolean; saveError: string
  open(path?: string): Promise<void>; read(number: number): Promise<void>; setDraftContent(content: string): void
  saveChapter(): Promise<void>; overview(): void; runtime(): void
}
let request = 0
const normalizePath = (path: string) => path.replaceAll('\\', '/').replace(/\/+$/, '').toLocaleLowerCase()
export const useStudio = create<StudioState>((set, get) => ({
  project: null, chapter: null, busy: false, error: '', view: 'overview', chapterLoading: false,
  draftContent: '', savedContent: '', dirty: false, saveBusy: false, saveError: '',
  async open(path) {
    if (get().busy) return
    const ticket = ++request
    set({busy: true, error: '', chapterLoading: false})
    try {
      const api = bridge()
      const selected = path ?? await api.SelectProjectDirectory()
      if (!selected) return
      const active = get()
      if (active.saveBusy) return
      const sameProject = [active.project?.outputDir, active.project?.projectRoot].some(value => value && normalizePath(value) === normalizePath(selected))
      if (active.dirty && !window.confirm('当前章节有未保存修改。继续打开项目将放弃这些修改，是否继续？')) return
      if (!sameProject && useRevisionStore.getState().status.hasUnsynced) {
        set({error: '当前项目存在未同步人工修改，请先完成同步后再切换项目。'})
        return
      }
      const project = await api.OpenProject(selected)
      if (ticket === request) {
        set({project, chapter: null, draftContent: '', savedContent: '', dirty: false, saveError: '', view: 'overview'})
        await useEngineStore.getState().selectProject(project.overview.path, api.GetRuntimeState)
        await useRevisionStore.getState().selectProject(project.outputDir, () => api.GetRevisionStatus())
        await useEngineStore.getState().refreshRuntime()
        setChapterCommitHandler(async (chapter: number, event: StudioEngineEvent) => {
          if (!api.ConfirmChapterCommit || !event.log) throw new Error('桌面桥接尚未提供章节提交复核')
          const active = useStudio.getState()
          if (normalizePath(active.project?.overview.path ?? '') !== normalizePath(project.overview.path)) return
          const confirmation = await api.ConfirmChapterCommit(chapter, event.log.Time)
          if (!confirmation.confirmed) return
          const current = useStudio.getState()
          if (normalizePath(current.project?.overview.path ?? '') !== normalizePath(project.overview.path)) return
          const selectedChapter = current.view === 'chapter' && current.chapter?.number === chapter
          const refreshedChapter = selectedChapter ? await api.GetChapter(chapter) : current.chapter
          const latest = useStudio.getState()
          if (normalizePath(latest.project?.overview.path ?? '') !== normalizePath(project.overview.path)) return
          const refreshStillSelected = selectedChapter && latest.view === 'chapter' && latest.chapter?.number === chapter
          useStudio.setState({
            project: confirmation.project,
            ...(refreshStillSelected && refreshedChapter ? {chapter: refreshedChapter} : {}),
            ...(refreshStillSelected && refreshedChapter && !latest.dirty
              ? {draftContent: refreshedChapter.content, savedContent: refreshedChapter.content, dirty: false, saveError: ''}
              : {}),
          })
        })
      }
    } catch (error) { if (ticket === request) set({error: String(error)}) }
    finally { if (ticket === request) set({busy: false}) }
  },
  async read(number) {
    if (get().busy) return
    if (get().view === 'chapter' && get().chapter?.number === number) return
    if (get().saveBusy) return
    if (get().dirty && !window.confirm('当前章节有未保存修改。切换章节将放弃这些修改，是否继续？')) return
    const ticket = ++request
    set({chapterLoading: true, error: '', view: 'chapter', chapter: null, draftContent: '', savedContent: '', dirty: false, saveError: ''})
    try {
      const chapter = await bridge().GetChapter(number)
      if (ticket === request) set({chapter, draftContent: chapter.content, savedContent: chapter.content})
    } catch (error) { if (ticket === request) set({error: String(error)}) }
    finally { if (ticket === request) set({chapterLoading: false}) }
  },
  setDraftContent(content) { set({draftContent: content, dirty: content !== get().savedContent, saveError: ''}) },
  async saveChapter() {
    const {project, chapter, draftContent, dirty, saveBusy} = get()
    if (!project || !chapter || !dirty || saveBusy) return
    const projectId = project.outputDir
    const chapterNumber = chapter.number
    set({saveBusy: true, saveError: ''})
    try {
      const result = await bridge().SaveChapter(chapterNumber, draftContent)
      const current = get()
      if (normalizePath(current.project?.outputDir ?? '') !== normalizePath(projectId) || current.chapter?.number !== chapterNumber) return
      const newerDraft = current.draftContent !== draftContent
      set({
        chapter: result.chapter,
        draftContent: newerDraft ? current.draftContent : result.chapter.content,
        savedContent: result.chapter.content,
        dirty: newerDraft && current.draftContent !== result.chapter.content,
        saveError: '',
      })
      useRevisionStore.getState().acceptStatus(result.revision)
      await useEngineStore.getState().refreshRuntime()
    } catch (error) {
      if (normalizePath(get().project?.outputDir ?? '') === normalizePath(projectId) && get().chapter?.number === chapterNumber) {
        set({saveError: String(error)})
      }
    } finally {
      if (normalizePath(get().project?.outputDir ?? '') === normalizePath(projectId) && get().chapter?.number === chapterNumber) set({saveBusy: false})
    }
  },
  overview() {
    if (get().saveBusy || (get().dirty && !window.confirm('当前章节有未保存修改。离开将放弃这些修改，是否继续？'))) return
    ++request
    set({view:'overview', chapterLoading:false, error:'', draftContent:get().savedContent, dirty:false, saveError:''})
  },
  runtime() {
    if (get().saveBusy || (get().dirty && !window.confirm('当前章节有未保存修改。离开将放弃这些修改，是否继续？'))) return
    ++request
    set({view:'runtime', chapterLoading:false, error:'', draftContent:get().savedContent, dirty:false, saveError:''})
  },
}))
