import { useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Check, ChevronRight, Clock3, GitPullRequest, RefreshCw, RotateCcw, X } from 'lucide-react'
import { useStudio } from '../store'
import { useProposalStore } from '../proposalStore'
import type { ProposalStatus } from '../types'
import '../proposal.css'

const statusLabel: Record<ProposalStatus, string> = {
  Draft: '草稿', Ready: '待审阅', Accepted: '已接受', Rejected: '已拒绝', Stale: '已过期', AppliedWorkingCopy: '已应用', SyncPending: '等待同步', Synced: '已同步', Failed: '失败',
}

function ProposalDetail() {
  const {selected, diff, versions, busy, accept, reject, apply, restoreVersion} = useProposalStore()
  const [showVersions, setShowVersions] = useState(false)
  if (!selected) return <div className="proposal-empty-detail"><GitPullRequest size={34}/><h2>选择一条建议</h2><p>建议接受前不会修改正文；Apply 后仍需从 V1 的 Sync 入口提交 Core。</p></div>
  const canAccept = selected.status === 'Ready' || selected.status === 'Draft'
  const canApply = selected.status === 'Accepted'
  return <article className="proposal-detail">
    <header className="proposal-detail-heading"><div><span className={`proposal-status status-${selected.status.toLowerCase()}`}>{statusLabel[selected.status]}</span><h2>{selected.title}</h2><p>{selected.summary || '没有额外摘要。'}</p></div><span className="proposal-source">来源：{selected.source}</span></header>
    {selected.rationale && <section className="proposal-rationale"><strong>为什么建议修改</strong><p>{selected.rationale}</p></section>}
    {selected.error && <div className="proposal-error"><AlertTriangle size={15}/>{selected.error}</div>}
    <section className="proposal-actions"><button className="primary" onClick={() => void accept()} disabled={!canAccept || busy}><Check size={14}/>接受建议</button><button onClick={() => void reject()} disabled={busy || !canAccept}><X size={14}/>拒绝</button><button className="primary" onClick={() => void apply()} disabled={!canApply || busy}><GitPullRequest size={14}/>应用到正文</button></section>
    <section className="proposal-section"><div className="proposal-section-heading"><h3>Diff</h3><span>{selected.changes.length} 个 change</span></div><div className="proposal-diff">{diff.map((line, index) => <div className={`diff-line diff-${line.kind}`} key={`${index}-${line.kind}`}><span>{line.kind === 'added' ? '+' : line.kind === 'removed' ? '−' : ' '}</span><code>{line.text || ' '}</code></div>)}</div></section>
    <section className="proposal-section"><div className="proposal-section-heading"><h3>Evidence</h3><span>{selected.evidence.length} 条</span></div>{selected.evidence.length ? <ul className="evidence-list">{selected.evidence.map((item, index) => <li key={`${item.resourceId}-${index}`}><div><strong>第 {item.chapter} 章</strong><span>revision {item.revision || '未知'}</span></div><code>{item.contentHash}</code><blockquote>{item.quotePreview || '未提供摘录'}</blockquote><small>正文区间：{item.startOffset}–{item.endOffset}</small></li>)}</ul> : <p className="muted">没有绑定证据。</p>}</section>
    <section className="proposal-section"><button className="text-button" onClick={() => setShowVersions(value => !value)}><Clock3 size={14}/>版本历史（{versions.length}）<ChevronRight size={14} className={showVersions ? 'rotate' : ''}/></button>{showVersions && <ul className="version-list">{versions.map(version => <li key={version.id}><div><strong>{new Date(version.createdAt).toLocaleString()}</strong><span>{version.source}</span></div><code>{version.contentHash}</code><button onClick={() => version.currentHash && void restoreVersion(version.id, version.currentHash)} disabled={busy || !version.currentHash} title={version.currentHash ? '按当前正文版本恢复' : '无法确认当前正文版本'}><RotateCcw size={13}/>恢复</button></li>)}</ul>}</section>
  </article>
}

export function ProposalInbox() {
  const projectId = useStudio(state => state.project?.outputDir ?? '')
  const [filter, setFilter] = useState<ProposalStatus | 'all'>('all')
  const [chapter, setChapter] = useState('1')
  const [title, setTitle] = useState('人工建议')
  const [after, setAfter] = useState('')
  const {proposals, selected, loading, busy, error, notice, load, select, createManual} = useProposalStore()
  useEffect(() => { if (projectId) void load(projectId) }, [projectId, load])
  const visible = useMemo(() => filter === 'all' ? proposals : proposals.filter(item => item.status === filter), [filter, proposals])
  return <section className="proposal-inbox">
    <div className="proposal-heading"><div><p className="eyebrow">V2 PROPOSAL WORKSPACE</p><h1>建议收件箱</h1><p className="proposal-subtitle">Proposal 记录建议与证据；正文修改必须经过 Apply → SavedUnsynced → Core Sync。</p></div><button onClick={() => void load(projectId)} disabled={loading}><RefreshCw size={14}/>{loading ? '读取中…' : '重新读取'}</button></div>
    {error && <div className="proposal-error" role="alert"><AlertTriangle size={15}/>{error}</div>}
    {notice && <div className="proposal-notice" role="status"><Check size={15}/>{notice}</div>}
    <div className="proposal-toolbar"><label>筛选<select value={filter} onChange={event => setFilter(event.target.value as ProposalStatus | 'all')}><option value="all">全部状态</option>{(Object.keys(statusLabel) as ProposalStatus[]).map(status => <option value={status} key={status}>{statusLabel[status]}</option>)}</select></label><span>{visible.length} 条建议</span></div>
    <div className="proposal-layout"><aside className="proposal-list">{visible.length ? visible.map(item => <button key={item.id} className={selected?.id === item.id ? 'proposal-list-item active' : 'proposal-list-item'} onClick={() => void select(item.id)}><span className={`proposal-status status-${item.status.toLowerCase()}`}>{statusLabel[item.status]}</span><strong>{item.title}</strong><small>{item.changes.length} 个 change · {new Date(item.updatedAt).toLocaleString()}</small></button>) : <div className="proposal-empty">暂无建议。可以从审阅中心创建，或在右侧输入人工候选正文。</div>}</aside><ProposalDetail/></div>
    <section className="manual-proposal"><div><h3>创建人工建议</h3><p>第一版由用户提供候选正文，不触发新的模型调用。</p></div><div className="manual-fields"><label>章节<input type="number" min="1" value={chapter} onChange={event => setChapter(event.target.value)}/></label><label>标题<input value={title} onChange={event => setTitle(event.target.value)}/></label></div><textarea value={after} onChange={event => setAfter(event.target.value)} placeholder="输入候选正文…"/><button className="primary" disabled={busy || !after.trim() || !Number(chapter)} onClick={() => void createManual(Number(chapter), title, after)}>创建建议</button></section>
  </section>
}
