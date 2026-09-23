import { beforeEach, expect, test, vi } from 'vitest'
import { useStudio } from './store'
import type { Chapter, Project, StudioBridge } from './types'

const project: Project = {projectRoot:'C:/',outputDir:'C:/小说',overview:{title:'测试小说',synopsis:'简介',path:'C:/小说',phase:'writing',flow:'writing',currentChapter:2,completedChapters:1,plannedChapters:2,wordCount:20,currentVolume:1,currentArc:1},tree:[]}
let api: StudioBridge
beforeEach(() => {
  api = {SelectProjectDirectory:vi.fn().mockResolvedValue(''),OpenProject:vi.fn().mockResolvedValue(project),GetProjectOverview:vi.fn(),GetProjectTree:vi.fn(),GetChapter:vi.fn()}
  vi.stubGlobal('window',{go:{bridge:{App:api}}})
  useStudio.setState({project:null,chapter:null,busy:false,error:'',view:'overview',chapterLoading:false})
})

test('取消选择不清空当前项目',async () => {
  useStudio.setState({project})
  await useStudio.getState().open()
  expect(useStudio.getState().project).toBe(project)
  expect(api.OpenProject).not.toHaveBeenCalled()
  expect(useStudio.getState().busy).toBe(false)
})

test('打开失败保留旧项目并显示错误',async () => {
  useStudio.setState({project})
  vi.mocked(api.OpenProject).mockRejectedValue(new Error('进度文件损坏'))
  await useStudio.getState().open('invalid')
  expect(useStudio.getState().project).toBe(project)
  expect(useStudio.getState().error).toContain('进度文件损坏')
})

test('快速切章时忽略晚到的旧响应',async () => {
  let resolveFirst!: (chapter: Chapter) => void
  vi.mocked(api.GetChapter).mockImplementationOnce(() => new Promise(resolve => {resolveFirst=resolve}))
    .mockResolvedValueOnce({number:2,title:'第二章',content:'新正文',wordCount:3,hasContent:true})
  const first = useStudio.getState().read(1)
  await useStudio.getState().read(2)
  resolveFirst({number:1,title:'第一章',content:'旧正文',wordCount:3,hasContent:true})
  await first
  expect(useStudio.getState().chapter?.number).toBe(2)
})

test('项目切换使旧章节请求失效',async () => {
  let resolveOld!: (chapter: Chapter) => void
  vi.mocked(api.GetChapter).mockImplementationOnce(() => new Promise(resolve => {resolveOld=resolve}))
  const old = useStudio.getState().read(1)
  await useStudio.getState().open('C:/新项目')
  resolveOld({number:1,title:'旧项目章节',content:'旧正文',wordCount:3,hasContent:true})
  await old
  expect(useStudio.getState().chapter).toBeNull()
  expect(useStudio.getState().view).toBe('overview')
})
