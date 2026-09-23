import { useEffect } from 'react'
import { useStudio } from '../store'
import { useEngineStore } from '../engineStore'
import { useRevisionStore } from '../revisionStore'

export function ChapterReader() {
  const {chapter, chapterLoading, error, draftContent, dirty, saveBusy, syncing, saveError, setDraftContent, saveChapter} = useStudio()
  const runtime = useEngineStore(state => state.runtime)
  const revisions = useRevisionStore(state => state.status)
  const runtimeReadOnly = ['running', 'pausing', 'stopping'].includes(runtime.state)
  const readOnly = runtimeReadOnly || syncing || !chapter?.canEdit
  const unsynced = !!chapter && revisions.hasUnsynced && revisions.chapters.some(item => item.chapter === chapter.number)
  const knownSynced = revisions.state === 'synced' && !unsynced
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault()
        void saveChapter()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [saveChapter])
  useEffect(() => {
    if (!dirty) return
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])
  if (chapterLoading) return <div className="empty" role="status">正在读取章节…</div>
  if (!chapter) return <div className="empty">{error ? '章节读取失败，请重试或选择其他章节。' : '从左侧选择章节。'}</div>
  return <article className="reader"><p className="eyebrow">CHAPTER {String(chapter.number).padStart(3,'0')} / 章节正文</p>
    <h1>{chapter.title}</h1><div className="reader-meta"><span>{syncing ? '同步中只读' : runtimeReadOnly ? '运行中只读' : readOnly ? '只读正文' : '可编辑正文'}</span><span>{draftContent.length.toLocaleString()} 字符</span><span>Markdown 原文</span></div>
    <div className="editor-toolbar">
      <span className={dirty ? 'edit-state modified' : unsynced ? 'edit-state unsynced' : 'edit-state'} role="status">
        {syncing ? '正在同步章节修订' : runtimeReadOnly ? 'Writer 正在运行，编辑已锁定' : !chapter.canEdit ? '当前章节暂不可编辑' : dirty ? '● 编辑中 · 尚未保存' : unsynced ? '⚠ 已保存 · 尚未同步' : knownSynced ? '✓ 已同步' : revisions.error ? '修订状态检查失败' : '修订状态未确认'}
      </span>
      <span className="editor-word-count">{draftContent.length.toLocaleString()} 字符</span>
      <button className="primary editor-save" disabled={!dirty || saveBusy || readOnly} onClick={() => void saveChapter()}>{saveBusy ? '保存中…' : '保存'}</button>
    </div>
    {runtimeReadOnly && <p className="editor-hint" role="status">Writer 正在生成，本章正文暂不可编辑。暂停完成后可进行人工修改。</p>}
    {!runtimeReadOnly && !chapter.canEdit && <p className="editor-hint" role="status">仅支持编辑已有正文且 Core 接纳基线完整的已完成章节。</p>}
    {unsynced && !dirty && <p className="editor-hint unsynced-hint" role="status">当前正文已保存到项目，但 ainovel Core 尚未接纳该修改。请先完成同步，再继续创作。</p>}
    {saveError && <p className="editor-error" role="alert">保存失败：{saveError}</p>}
    {revisions.error && !dirty && <p className="editor-error" role="alert">正文已保存，但修订状态检查失败：{revisions.error}</p>}
    {chapter.hasContent || chapter.canEdit ? <textarea className="chapter-editor" aria-label="章节正文编辑器" value={draftContent} readOnly={readOnly} onChange={event => setDraftContent(event.target.value)} spellCheck={false}/> : <div className="empty">本章尚无可编辑的已完成正文。</div>}
    <div className="editor-footer"><span>{draftContent.length.toLocaleString()} 字符</span><span>Markdown 原文 · Ctrl/Cmd+S 保存</span></div>
  </article>
}
