import { beforeEach, expect, test, vi } from 'vitest'
import { useStudio } from './store'
import { useRevisionStore } from './revisionStore'
import { setChapterCommitHandler, useEngineStore } from './engineStore'
import { revisionsAllowWriting } from './revisionStore'
import type { Chapter, ChapterSyncResult, Project, StudioBridge } from './types'

const project: Project = {projectRoot:'C:/',outputDir:'C:/小说',overview:{title:'测试小说',synopsis:'简介',path:'C:/小说',phase:'writing',flow:'writing',currentChapter:2,completedChapters:1,plannedChapters:2,wordCount:20,currentVolume:1,currentArc:1},tree:[]}
let api: StudioBridge
beforeEach(() => {
  api = {
    SelectProjectDirectory:vi.fn().mockResolvedValue(''),OpenProject:vi.fn().mockResolvedValue(project),
    GetProjectOverview:vi.fn(),GetProjectTree:vi.fn(),GetChapter:vi.fn(),SaveChapter:vi.fn(),SyncChapterRevisions:vi.fn(),
    GetRuntimeState:vi.fn(),
    GetRevisionStatus:vi.fn().mockResolvedValue({projectId:project.outputDir,state:'unknown',hasUnsynced:false,chapters:[]}),
    CheckChapterRevisions:vi.fn().mockResolvedValue({projectId:project.outputDir,state:'synced',hasUnsynced:false,chapters:[]}),
    ConfirmChapterCommit:vi.fn().mockResolvedValue({confirmed:true,project}),
  }
  vi.stubGlobal('window',{go:{bridge:{App:api}}})
  useStudio.setState({project:null,chapter:null,busy:false,error:'',view:'overview',chapterLoading:false,draftContent:'',savedContent:'',dirty:false,saveBusy:false,saveError:'',syncing:false,syncError:'',syncNotice:''})
  useRevisionStore.setState({projectId:'',generation:0,status:{projectId:'',state:'unknown',hasUnsynced:false,chapters:[]},checking:false,error:''})
  useEngineStore.setState({projectId:'',runtime:{projectId:'',generation:0,state:'idle',phase:'',flow:'',elapsedSeconds:0,inputTokens:0,outputTokens:0,projectInputTokens:0,projectOutputTokens:0,runCostUsd:0,projectCostUsd:0,agents:[],updatedAt:''},logs:[],pipeline:{},lastSequence:0,controlBusy:false,controlError:''})
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
    .mockResolvedValueOnce({number:2,title:'第二章',content:'新正文',wordCount:3,hasContent:true,canEdit:true})
  const first = useStudio.getState().read(1)
  await useStudio.getState().read(2)
  resolveFirst({number:1,title:'第一章',content:'旧正文',wordCount:3,hasContent:true,canEdit:true})
  await first
  expect(useStudio.getState().chapter?.number).toBe(2)
})

test('项目切换使旧章节请求失效',async () => {
  let resolveOld!: (chapter: Chapter) => void
  vi.mocked(api.GetChapter).mockImplementationOnce(() => new Promise(resolve => {resolveOld=resolve}))
  const old = useStudio.getState().read(1)
  await useStudio.getState().open('C:/新项目')
  resolveOld({number:1,title:'旧项目章节',content:'旧正文',wordCount:3,hasContent:true,canEdit:true})
  await old
  expect(useStudio.getState().chapter).toBeNull()
  expect(useStudio.getState().view).toBe('overview')
})

test('打开项目后读取 Store 修订事实',async () => {
  vi.mocked(api.CheckChapterRevisions).mockResolvedValue({
    projectId:project.outputDir,state:'saved_unsynced',hasUnsynced:true,
    chapters:[{chapter:1,acceptedHash:'base',currentHash:'edited'}],
  })
  await useStudio.getState().open('C:/项目')
  await new Promise(resolve => setTimeout(resolve, 0))
  expect(api.GetRevisionStatus).toHaveBeenCalledOnce()
  expect(api.CheckChapterRevisions).toHaveBeenCalledOnce()
  expect(useRevisionStore.getState().status.state).toBe('saved_unsynced')
  expect(useRevisionStore.getState().status.chapters[0]?.currentHash).toBe('edited')
})

test('编辑并保存只更新正文与 SavedUnsynced/WaitingSync 状态',async () => {
  const savedChapter = {number:1,title:'第一章',content:'人工修订',wordCount:4,hasContent:true,canEdit:true}
  const status = {projectId:project.outputDir,state:'saved_unsynced' as const,hasUnsynced:true,chapters:[{chapter:1,acceptedHash:'old',currentHash:'new'}]}
  vi.mocked(api.GetChapter).mockResolvedValue({number:1,title:'第一章',content:'原正文',wordCount:3,hasContent:true,canEdit:true})
  vi.mocked(api.SaveChapter).mockResolvedValue({chapter:savedChapter,revision:status})
  vi.mocked(api.GetRuntimeState!).mockResolvedValue({projectId:project.outputDir,generation:0,state:'waiting_sync',phase:'',flow:'',elapsedSeconds:0,inputTokens:0,outputTokens:0,projectInputTokens:0,projectOutputTokens:0,runCostUsd:0,projectCostUsd:0,agents:[],updatedAt:''})
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useStudio.getState().setDraftContent('人工修订')
  expect(useStudio.getState().dirty).toBe(true)

  await useStudio.getState().saveChapter()

  expect(api.SaveChapter).toHaveBeenCalledWith(1,'人工修订')
  expect(useStudio.getState().dirty).toBe(false)
  expect(useStudio.getState().savedContent).toBe('人工修订')
  expect(useRevisionStore.getState().status.state).toBe('saved_unsynced')
  expect(useEngineStore.getState().runtime.state).toBe('waiting_sync')
})

test('立即同步刷新项目章节与修订状态并恢复继续创作',async () => {
  const syncedChapter: Chapter = {number:1,title:'第一章',content:'Core 已接纳正文',wordCount:6,hasContent:true,canEdit:true}
  const syncedRevision = {projectId:project.outputDir,state:'synced' as const,hasUnsynced:false,chapters:[]}
  const refreshedProject = {...project,overview:{...project.overview,wordCount:42}}
  const syncResult: ChapterSyncResult = {project:refreshedProject,chapter:syncedChapter,revision:syncedRevision}
  vi.mocked(api.GetChapter).mockResolvedValue({number:1,title:'第一章',content:'本地修订',wordCount:4,hasContent:true,canEdit:true})
  vi.mocked(api.SyncChapterRevisions!).mockResolvedValue(syncResult)
  vi.mocked(api.GetRuntimeState!).mockResolvedValue({projectId:project.outputDir,generation:0,state:'idle',phase:'',flow:'',elapsedSeconds:0,inputTokens:0,outputTokens:0,projectInputTokens:0,projectOutputTokens:0,runCostUsd:0,projectCostUsd:0,agents:[],updatedAt:''})
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useRevisionStore.getState().acceptStatus({projectId:project.outputDir,state:'saved_unsynced',hasUnsynced:true,chapters:[{chapter:1,acceptedHash:'old',currentHash:'new'}]})
  useEngineStore.setState({runtime:{...useEngineStore.getState().runtime,projectId:project.outputDir,state:'waiting_sync'}})

  await useStudio.getState().syncChapterRevisions()

  expect(api.SyncChapterRevisions).toHaveBeenCalledWith(1)
  expect(useStudio.getState().project?.overview.wordCount).toBe(42)
  expect(useStudio.getState().chapter?.content).toBe('Core 已接纳正文')
  expect(useStudio.getState().draftContent).toBe('Core 已接纳正文')
  expect(useRevisionStore.getState().status.state).toBe('synced')
  expect(useEngineStore.getState().runtime.state).toBe('idle')
  expect(revisionsAllowWriting(project.outputDir, useRevisionStore.getState().status, false, '')).toBe(true)
  expect(useStudio.getState().syncError).toBe('')
  expect(useStudio.getState().syncNotice).toContain('同步完成')
  expect(useStudio.getState().syncing).toBe(false)
})

test('同步失败后刷新 pending 恢复态并保留错误',async () => {
  vi.mocked(api.SyncChapterRevisions!).mockRejectedValue(new Error('恢复阶段暂时失败'))
  vi.mocked(api.CheckChapterRevisions).mockResolvedValue({
    projectId:project.outputDir,state:'recovery_pending',hasUnsynced:true,pendingStage:'records_applied',chapters:[{chapter:1,acceptedHash:'old',currentHash:'new'}],
  })
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useRevisionStore.getState().acceptStatus({projectId:project.outputDir,state:'saved_unsynced',hasUnsynced:true,chapters:[{chapter:1,acceptedHash:'old',currentHash:'new'}]})

  await useStudio.getState().syncChapterRevisions()

  expect(useRevisionStore.getState().status.state).toBe('recovery_pending')
  expect(useRevisionStore.getState().status.pendingStage).toBe('records_applied')
  expect(useStudio.getState().syncError).toContain('恢复阶段暂时失败')
  expect(useStudio.getState().syncing).toBe(false)
})

test('Sync 返回错误时只读复核的 synced 结果不得解锁 Resume',async () => {
  vi.mocked(api.SyncChapterRevisions!).mockRejectedValue(new Error('Host 返回同步错误'))
  vi.mocked(api.CheckChapterRevisions).mockResolvedValue({projectId:project.outputDir,state:'synced',hasUnsynced:false,chapters:[]})
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useRevisionStore.getState().acceptStatus({projectId:project.outputDir,state:'saved_unsynced',hasUnsynced:true,chapters:[{chapter:1,acceptedHash:'old',currentHash:'new'}]})

  await useStudio.getState().syncChapterRevisions()

  expect(useRevisionStore.getState().status.state).not.toBe('synced')
  expect(useRevisionStore.getState().error).toContain('Host 返回同步错误')
  expect(revisionsAllowWriting(project.outputDir,useRevisionStore.getState().status,useRevisionStore.getState().checking,useRevisionStore.getState().error)).toBe(false)
})

test('脏正文存在时不允许开始同步',async () => {
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useStudio.setState({syncNotice:'章节修订同步完成，项目与章节视图已刷新。'})
  useStudio.getState().setDraftContent('尚未保存')

  await useStudio.getState().syncChapterRevisions()

  expect(api.SyncChapterRevisions).not.toHaveBeenCalled()
  expect(useStudio.getState().dirty).toBe(true)
  expect(useStudio.getState().syncNotice).toBe('')
})

test('章节提交通过 Store 二次确认后刷新选中编辑器正文',async () => {
  vi.mocked(api.GetChapter)
    .mockResolvedValueOnce({number:1,title:'第一章',content:'',wordCount:0,hasContent:false,canEdit:false})
    .mockResolvedValueOnce({number:1,title:'第一章',content:'Engine 新提交正文',wordCount:6,hasContent:true,canEdit:true})
  vi.mocked(api.ConfirmChapterCommit!).mockResolvedValue({confirmed:true,project})
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  expect(useStudio.getState().draftContent).toBe('')

  useEngineStore.getState().handleEvent({
    projectId:project.overview.path,generation:1,runId:1,sequence:1,timestamp:new Date().toISOString(),type:'log',
    log:{ID:'commit-1',Time:new Date().toISOString(),FinishedAt:new Date().toISOString(),Failed:false,Category:'TOOL',Agent:'writer',Summary:'章节已提交',Detail:'',Tool:'commit_chapter',Chapter:1,Level:'info',Depth:0},
  })
  await new Promise(resolve => setTimeout(resolve, 0))

  expect(api.ConfirmChapterCommit).toHaveBeenCalledOnce()
  expect(api.GetChapter).toHaveBeenCalledTimes(2)
  expect(useStudio.getState().chapter?.content).toBe('Engine 新提交正文')
  expect(useStudio.getState().draftContent).toBe('Engine 新提交正文')
  expect(useStudio.getState().savedContent).toBe('Engine 新提交正文')
  expect(useStudio.getState().dirty).toBe(false)
})

test('章节提交刷新晚到时不得替换用户当前编辑的其他章节',async () => {
  let resolveRefresh!: (chapter: Chapter) => void
  vi.mocked(api.GetChapter)
    .mockResolvedValueOnce({number:1,title:'第一章',content:'',wordCount:0,hasContent:false,canEdit:false})
    .mockImplementationOnce(() => new Promise(resolve => {resolveRefresh=resolve}))
    .mockResolvedValueOnce({number:2,title:'第二章',content:'第二章原文',wordCount:5,hasContent:true,canEdit:true})
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useEngineStore.getState().handleEvent({
    projectId:project.overview.path,generation:1,runId:1,sequence:1,timestamp:new Date().toISOString(),type:'log',
    log:{ID:'commit-late',Time:new Date().toISOString(),FinishedAt:new Date().toISOString(),Failed:false,Category:'TOOL',Agent:'writer',Summary:'章节已提交',Detail:'',Tool:'commit_chapter',Chapter:1,Level:'info',Depth:0},
  })
  await vi.waitFor(() => expect(api.GetChapter).toHaveBeenCalledTimes(2))
  await useStudio.getState().read(2)
  useStudio.getState().setDraftContent('第二章尚未保存的新草稿')
  resolveRefresh({number:1,title:'第一章',content:'第一章新提交正文',wordCount:7,hasContent:true,canEdit:true})
  await new Promise(resolve => setTimeout(resolve, 0))

  expect(useStudio.getState().chapter?.number).toBe(2)
  expect(useStudio.getState().draftContent).toBe('第二章尚未保存的新草稿')
  expect(useStudio.getState().dirty).toBe(true)
})

test('保存失败保留脏正文以便重试',async () => {
  vi.mocked(api.GetChapter).mockResolvedValue({number:1,title:'第一章',content:'原正文',wordCount:3,hasContent:true,canEdit:true})
  vi.mocked(api.SaveChapter).mockRejectedValue(new Error('磁盘只读'))
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useStudio.getState().setDraftContent('待保存的新正文')

  await useStudio.getState().saveChapter()

  expect(useStudio.getState().dirty).toBe(true)
  expect(useStudio.getState().draftContent).toBe('待保存的新正文')
  expect(useStudio.getState().saveError).toContain('磁盘只读')
})

test('保存期间继续输入时保留较新的脏草稿',async () => {
  const savedChapter = {number:1,title:'第一章',content:'保存快照 A',wordCount:6,hasContent:true,canEdit:true}
  const status = {projectId:project.outputDir,state:'saved_unsynced' as const,hasUnsynced:true,chapters:[{chapter:1,acceptedHash:'old',currentHash:'new'}]}
  let resolveSave!: (result: {chapter: Chapter; revision: typeof status}) => void
  vi.mocked(api.SaveChapter).mockImplementation(() => new Promise(resolve => {resolveSave=resolve}))
  vi.mocked(api.GetRuntimeState!).mockResolvedValue({projectId:project.outputDir,generation:0,state:'waiting_sync',phase:'',flow:'',elapsedSeconds:0,inputTokens:0,outputTokens:0,projectInputTokens:0,projectOutputTokens:0,runCostUsd:0,projectCostUsd:0,agents:[],updatedAt:''})
  useStudio.setState({project,chapter:{number:1,title:'第一章',content:'原文',wordCount:2,hasContent:true,canEdit:true},draftContent:'保存快照 A',savedContent:'原文',dirty:true})

  const saving = useStudio.getState().saveChapter()
  useStudio.getState().setDraftContent('较新的草稿 B')
  resolveSave({chapter:savedChapter,revision:status})
  await saving

  expect(useStudio.getState().savedContent).toBe('保存快照 A')
  expect(useStudio.getState().draftContent).toBe('较新的草稿 B')
  expect(useStudio.getState().dirty).toBe(true)
  expect(useStudio.getState().saveBusy).toBe(false)
})

test('放弃确认被取消时保留当前章节与脏正文',async () => {
  const confirm = vi.fn().mockReturnValue(false)
  vi.stubGlobal('window',{go:{bridge:{App:api}},confirm})
  vi.mocked(api.GetChapter).mockResolvedValue({number:1,title:'第一章',content:'原正文',wordCount:3,hasContent:true,canEdit:true})
  await useStudio.getState().open('C:/小说')
  await useStudio.getState().read(1)
  useStudio.getState().setDraftContent('未保存的修改')

  await useStudio.getState().read(2)

  expect(confirm).toHaveBeenCalledOnce()
  expect(api.GetChapter).toHaveBeenCalledOnce()
  expect(useStudio.getState().chapter?.number).toBe(1)
  expect(useStudio.getState().dirty).toBe(true)
})
