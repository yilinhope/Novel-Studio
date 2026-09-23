import { useEffect, useState } from 'react'
import { AlertTriangle, ClipboardCheck, RefreshCw } from 'lucide-react'
import { bridge } from '../services'
import { useEngineStore } from '../engineStore'
import { useStudio } from '../store'
import type { ReviewCenter as ReviewCenterData, ReviewEntry } from '../types'

const verdictLabel: Record<string, string> = {accept: '通过', polish: '打磨', rewrite: '返工'}
const scopeLabel: Record<string, string> = {chapter: '章节', arc: '弧', global: '全局'}

function ReviewCard({review}: {review: ReviewEntry}) {
  const issues = review.issues ?? []
  return <article className="review-card">
    <header><div><span className="review-scope">{scopeLabel[review.scope] ?? review.scope}审阅 · 截止第 {review.chapter} 章</span><h2>{verdictLabel[review.verdict] ?? review.verdict}</h2></div><span className="review-issue-count">{issues.length} 项问题</span></header>
    <p className="review-summary">{review.summary || 'Core 未提供总结。'}</p>
    {review.contract_status && <p className="review-meta">契约状态：{review.contract_status}{review.contract_notes ? ` · ${review.contract_notes}` : ''}</p>}
    {!!review.contract_misses?.length && <ul className="review-contract-list">{review.contract_misses.map((item, index) => <li key={`${index}-${item}`}>{item}</li>)}</ul>}
    {!!review.dimensions?.length && <ul className="review-dimensions">{review.dimensions.map((dimension, index) => <li key={`${dimension.dimension}-${index}`}><span>{dimension.dimension}</span><strong>{dimension.score}</strong>{dimension.verdict && <small>{dimension.verdict}</small>}{dimension.comment && <em>{dimension.comment}</em>}</li>)}</ul>}
    {issues.length > 0 && <ul className="review-issues">{issues.map((issue, index) => <li key={`${issue.type}-${index}`}>
      <div><strong>{issue.type || '问题'}</strong><span className={`review-severity severity-${issue.severity}`}>{issue.severity}</span>{issue.requires_change && <span className="review-change-needed">需修改</span>}</div>
      <p>{issue.description}</p>
      {issue.evidence && <blockquote>{issue.evidence}</blockquote>}
      {issue.suggestion && <p className="review-meta">建议：{issue.suggestion}</p>}
      {!!issue.chapters?.length && <small>关联章节：{issue.chapters.join('、')}</small>}
    </li>)}</ul>}
    {!!review.affected_chapters?.length && <p className="review-meta">Core 标记的关联章节：{review.affected_chapters.join('、')}</p>}
  </article>
}

export function ReviewCenter() {
  const projectId = useStudio(state => state.project?.outputDir ?? '')
  const [data, setData] = useState<ReviewCenterData | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const load = async () => {
    const getter = bridge().GetReviewCenter
    if (!getter) { setError('桌面桥接尚未提供 Review Center 数据。'); return }
    setLoading(true); setError('')
    try {
      const result = await getter()
      if (result.projectId.replaceAll('\\', '/').toLocaleLowerCase() !== projectId.replaceAll('\\', '/').toLocaleLowerCase()) return
      setData(result)
      await useEngineStore.getState().refreshRuntime()
    } catch (cause) { setError(String(cause)) }
    finally { setLoading(false) }
  }
  useEffect(() => { void load() }, [])

  return <section className="review-center">
    <div className="review-heading"><div><p className="eyebrow">CORE REVIEW RECORDS</p><h1>审阅中心</h1><p className="review-subtitle">这里只展示 Core 已写入 Store 的 Editor Review；推进许可单独读取，不会生成审阅结果。</p></div><button onClick={() => void load()} disabled={loading}><RefreshCw size={14}/>{loading ? '读取中…' : '重新读取'}</button></div>
    {error && <div className="review-error" role="alert"><AlertTriangle size={15}/>{error}</div>}
    {data && <>
      <section className="advance-gate" aria-label="下一章推进门状态">
        <div className="advance-gate-icon"><ClipboardCheck size={18}/></div>
        <div><span>下一章推进门</span><strong>{data.advanceMode === 'auto' ? '自动推进模式' : !data.canAdvance ? '当前不能放行下一章' : data.requiresAdvancePermit ? `第 ${data.nextChapter} 章等待一次性推进许可` : data.advanceMode === 'review' ? `第 ${data.nextChapter} 章当前无需新增推进许可` : '推进状态未确认'}</strong>
          <small>{data.hasCurrentReview ? '最新已完成章节存在 Core ReviewEntry。' : '最新已完成章节没有 Core ReviewEntry；等待许可不代表已审阅。'}</small>
          {data.advanceBlockedReason && <small>{data.advanceBlockedReason}</small>}
          {data.advanceHoldReason && <small>Core 暂停意图：{data.advanceHoldReason}</small>}
        </div>
      </section>
      <div className="review-list-heading"><h2>真实审阅记录</h2><span>{data.reviews.length} 条</span></div>
      {data.reviews.length ? <div className="review-list">{[...data.reviews].reverse().map((entry, index) => <ReviewCard key={`${entry.scope}-${entry.chapter}-${index}`} review={entry}/>)}</div> : <div className="review-empty">Core 尚未保存 Editor Review。章节推进许可状态不会被当作审阅结果显示。</div>}
    </>}
  </section>
}
