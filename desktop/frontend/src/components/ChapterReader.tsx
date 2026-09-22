import { useStudio } from '../store'

export function ChapterReader() {
  const {chapter, chapterLoading, error} = useStudio()
  if (chapterLoading) return <div className="empty" role="status">正在读取章节…</div>
  if (!chapter) return <div className="empty">{error ? '章节读取失败，请重试或选择其他章节。' : '从左侧选择章节。'}</div>
  return <article className="reader"><p className="eyebrow">CHAPTER {String(chapter.number).padStart(3,'0')} / 章节正文</p>
    <h1>{chapter.title}</h1><div className="reader-meta"><span>只读</span><span>{chapter.wordCount.toLocaleString()} 字</span><span>Markdown 原文</span></div>
    {chapter.hasContent ? <pre className="prose">{chapter.content}</pre> : <div className="empty">本章尚无已提交正文。</div>}
  </article>
}
