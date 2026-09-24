import { useEffect, useMemo, useState } from 'react'
import { Activity, Bot, CircleAlert, Clock3, Coins, Cpu, Play, Sparkles, SquareTerminal } from 'lucide-react'
import { runtimeLabel, useEngineStore } from '../engineStore'
import { revisionRecoveryStageLabel, revisionsAllowWriting, useRevisionStore } from '../revisionStore'
import { useStudio } from '../store'
import type { RuntimeLog } from '../types'

const writerSteps = ['novel_context', 'read_chapter', 'plan_chapter', 'draft_chapter', 'check_consistency', 'commit_chapter']
const labels: Record<string, string> = {
  novel_context: '读取创作上下文', read_chapter: '读取相邻章节', plan_chapter: '规划本章',
  draft_chapter: '撰写正文', edit_chapter: '修订正文', check_consistency: '检查一致性', commit_chapter: '提交章节',
}
const finished = (log: RuntimeLog) => Date.parse(log.FinishedAt) > 0

export function runtimeElapsedSeconds(runtime: {state: string; elapsedSeconds: number; updatedAt: string}, now = new Date()) {
  const base = Math.max(0, runtime.elapsedSeconds)
  if (!['running', 'pausing', 'stopping'].includes(runtime.state)) return base
  const updatedAt = Date.parse(runtime.updatedAt)
  if (!Number.isFinite(updatedAt)) return base
  return base + Math.max(0, Math.floor((now.getTime() - updatedAt) / 1000))
}

export function formatRuntimeActivity(log: RuntimeLog) {
  const summary = log.Summary || log.Detail || log.Tool || log.Category || '运行状态更新'
  const detail = log.Detail && log.Detail !== summary ? ` · ${log.Detail}` : ''
  return `${summary}${detail}`
}

function isOngoing(log: RuntimeLog) {
  return Boolean(log.ID) && !finished(log)
}

function ageLabel(seconds: number) {
  if (seconds < 5) return '刚刚'
  if (seconds < 60) return `${seconds} 秒前`
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return rest ? `${minutes} 分 ${rest} 秒前` : `${minutes} 分钟前`
}

function duration(seconds: number) {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const rest = seconds % 60
  return hours ? `${hours}:${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}` : `${minutes}:${String(rest).padStart(2, '0')}`
}

function timeLabel(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit', second: '2-digit'})
}

function LogRow({log}: {readonly log: RuntimeLog}) {
  const ongoing = !finished(log)
  return <div className={`runtime-log-row ${log.Level === 'error' || log.Failed ? 'is-error' : ''}`}>
    <time>{timeLabel(log.Time)}</time>
    <span className="runtime-log-agent">{log.Agent || log.Category}</span>
    <span className="runtime-log-message">{log.Summary || log.Detail || log.Tool || '运行状态更新'}</span>
    {ongoing && <span className="runtime-live">进行中</span>}
    {log.Failed && <CircleAlert size={13} aria-label="失败"/>}
  </div>
}

export function RuntimeCenter() {
  const {runtime, logs, pipeline, controlBusy, controlError, resumeWriting, pauseWriting, stopWriting} = useEngineStore()
  const revisions = useRevisionStore()
  const {dirty, saveBusy, syncing, syncError, syncNotice, refreshWarning, syncChapterRevisions} = useStudio()
  const [steerText, setSteerText] = useState('')
  const [, setClock] = useState(0)
  useEffect(() => {
    if (!['running', 'pausing', 'stopping'].includes(runtime.state)) return
    const timer = window.setInterval(() => setClock(value => value + 1), 1000)
    return () => window.clearInterval(timer)
  }, [runtime.state])
  const now = new Date()
  const activeLog = [...logs].reverse().find(isOngoing)
  const latestLog = logs.at(-1)
  const visibleLog = activeLog ?? latestLog
  const visibleLogAge = visibleLog ? Math.max(0, Math.floor((now.getTime() - Date.parse(visibleLog.Time)) / 1000)) : 0
  const elapsedSeconds = runtimeElapsedSeconds(runtime, now)
  const activityTitle = activeLog
    ? formatRuntimeActivity(activeLog)
    : runtime.error
      ? 'Core 已停止并报告错误'
      : runtime.state === 'running'
        ? 'Core 正在运行，等待下一条事件'
        : '当前没有正在执行的 Core 任务'
  const activityMeta = activeLog
    ? `${activeLog.Agent || activeLog.Category}${activeLog.Tool ? ` · ${labels[activeLog.Tool] ?? activeLog.Tool}` : ''} · ${ageLabel(visibleLogAge)}`
    : visibleLog
      ? `最近事件：${formatRuntimeActivity(visibleLog)} · ${ageLabel(visibleLogAge)}`
      : '尚未收到 Core 事件'
  const staleNotice = runtime.state === 'running' && visibleLog && visibleLogAge >= 45
    ? `已经 ${duration(visibleLogAge)} 没有新事件；这不代表 Core 已失败，可查看运行记录或停止任务。`
    : ''
  const canResume = !dirty && !syncing && revisionsAllowWriting(runtime.projectId, revisions.status, revisions.checking, revisions.error)
  const canResumeState = ['idle', 'paused', 'stopped', 'error', 'waiting_sync'].includes(runtime.state)
  const canSyncState = !['running', 'pausing', 'stopping'].includes(runtime.state)
  const canSync = canSyncState && revisions.status.hasUnsynced
  const canSubmitSteer = ['running', 'paused', 'waiting_review', 'idle', 'stopped', 'error', 'completed'].includes(runtime.state)
    && !dirty && !syncing && revisionsAllowWriting(runtime.projectId, revisions.status, revisions.checking, revisions.error)
  const tools = useMemo(() => {
    const seen = new Set(Object.keys(pipeline))
    return [...writerSteps.slice(0, 4), ...(seen.has('edit_chapter') ? ['edit_chapter'] : []), ...writerSteps.slice(4)]
  }, [pipeline])
  const stateClass = `runtime-state-${runtime.state}`
  return <section className="runtime-center">
    <div className="runtime-heading">
      <div><p className="eyebrow">ENGINE SESSION</p><h1>运行中心</h1><p className="runtime-subtitle">创作流程、用量与运行记录</p></div>
      <div className="runtime-heading-actions">
        <span className={`runtime-state ${stateClass}`}><i/>{runtimeLabel(runtime.state)}</span>
        {runtime.state === 'running' && <button disabled={controlBusy} onClick={() => void pauseWriting()}>暂停</button>}
        {runtime.state === 'pausing' && <button disabled>正在暂停…</button>}
        {runtime.state === 'stopping' && <button disabled>正在停止…</button>}
        {canSync && <button disabled={controlBusy || syncing || dirty || saveBusy || revisions.checking} title={dirty || saveBusy ? '请先保存或放弃未保存正文' : undefined} onClick={() => void syncChapterRevisions()}>{syncing ? revisions.status.state === 'recovery_pending' ? '正在恢复修订…' : '正在同步…' : revisions.status.state === 'recovery_pending' ? '继续恢复同步' : '立即同步'}</button>}
        {revisions.status.state === 'recovery_pending' && <span className="runtime-subtitle" role="status">Core 恢复阶段：{revisionRecoveryStageLabel(revisions.status.pendingStage)}</span>}
        {syncNotice && <span className="runtime-subtitle" role="status">{syncNotice}</span>}
        {refreshWarning && <span className="runtime-subtitle" role="status">Core 操作成功，但视图刷新提醒：{refreshWarning}</span>}
        {canResumeState && <button className="primary runtime-control-primary" disabled={controlBusy || syncing || !canResume} title={!canResume ? '请先完成章节修订检查，并同步所有未同步修订' : undefined} onClick={() => void resumeWriting()}>{runtime.state === 'idle' ? '开始创作' : ['paused', 'waiting_sync'].includes(runtime.state) ? '继续创作' : '恢复创作'}</button>}
        {canResumeState && !canResume && <span className="runtime-subtitle" role="status">{dirty ? '当前章节有未保存修改，请先保存或放弃编辑。' : revisions.checking ? '正在检查章节修订…' : revisions.status.hasUnsynced ? '发现未同步章节修订，请先同步后继续。' : revisions.error ? '修订检查失败，暂不能继续创作。' : '章节修订状态尚未确认。'}</span>}
        {['running', 'pausing', 'paused'].includes(runtime.state) && <button disabled={controlBusy || runtime.state === 'stopping'} onClick={() => void stopWriting()}>停止</button>}
      </div>
    </div>

    <section className="runtime-overview" aria-label="当前运行状态">
      <div className="runtime-now">
        <div className="runtime-now-icon"><Activity size={18}/></div>
        <div><span>当前章节</span><strong>{runtime.chapter ? `第 ${runtime.chapter} 章` : '尚未开始'}</strong><small>{runtime.phase || '等待 Engine Session'}</small></div>
      </div>
      <div className="runtime-now">
        <div className="runtime-now-icon"><Bot size={18}/></div>
        <div><span>当前 Agent / Step</span><strong>{runtime.agent || '—'}</strong><small>{runtime.step ? labels[runtime.step] ?? runtime.step : '暂无活动步骤'}</small></div>
      </div>
      <div className="runtime-now">
        <div className="runtime-now-icon"><Clock3 size={18}/></div>
        <div><span>本次运行时长</span><strong className="runtime-mono">{duration(elapsedSeconds)}</strong><small>{runtime.flow || '等待创作任务'}</small></div>
      </div>
    </section>

    <section className={`runtime-focus ${activeLog ? 'is-active' : ''} ${runtime.error ? 'is-error' : ''}`} aria-label="Core 当前活动">
      <div className="runtime-focus-icon"><Activity size={18}/></div>
      <div className="runtime-focus-copy"><span>Core 当前在做什么</span><strong>{activityTitle}</strong><small>{activityMeta}</small>{staleNotice && <em>{staleNotice}</em>}</div>
      <div className="runtime-focus-state"><span>状态</span><strong>{runtimeLabel(runtime.state)}</strong></div>
    </section>

    {(syncError || runtime.error || controlError) && <div className="runtime-error" role="status"><CircleAlert size={16}/><span>{syncError || controlError || runtime.error}</span></div>}

    <section className="runtime-panel runtime-steer" aria-label="创作方向干预">
      <div className="runtime-panel-heading"><h2>创作指令</h2><span>交由 Core Arbiter 判断如何处理</span></div>
      {runtime.pendingSteer && <p className="runtime-subtitle" role="status">待处理指令：{runtime.pendingSteer}</p>}
      <form onSubmit={event => {
        event.preventDefault()
        if (!steerText.trim()) return
        void useEngineStore.getState().submitSteer(steerText).then(result => {
          if (result) {
            if (result.project) useStudio.setState({project: result.project, refreshWarning: result.refreshWarning ?? ''})
            setSteerText('')
          }
        })
      }}>
        <input aria-label="创作方向指令" value={steerText} onChange={event => setSteerText(event.target.value)} placeholder="告诉 Core 你希望调整的创作方向" disabled={!canSubmitSteer || controlBusy}/>
        <button disabled={!canSubmitSteer || controlBusy || !steerText.trim()}>{controlBusy ? '正在处理…' : '发送创作指令'}</button>
      </form>
      {!canSubmitSteer && <small>{runtime.state === 'waiting_sync' ? '请先完成章节修订同步。' : runtime.state === 'pausing' || runtime.state === 'stopping' ? '请等待 Core 生命周期状态确认后再提交。' : dirty ? '请先保存或放弃当前正文修改。' : revisions.checking ? '正在检查章节修订…' : ''}</small>}
    </section>

    <div className="runtime-grid">
      <section className="runtime-panel runtime-pipeline">
        <div className="runtime-panel-heading"><h2><Play size={15}/>Writer Pipeline</h2><span>按真实工具事件更新</span></div>
        <ol>
          {tools.map((tool, index) => {
            const step = pipeline[tool]
            return <li className={`pipeline-step ${step?.state ?? 'queued'}`} key={tool}>
              <span className="pipeline-marker">{step?.state === 'complete' ? '✓' : step?.state === 'error' ? '!' : String(index + 1).padStart(2, '0')}</span>
              <span className="pipeline-copy"><strong>{labels[tool] ?? tool}</strong><small>{tool}</small></span>
              {step?.state === 'running' && <span className="step-live">运行中</span>}
              {step?.state === 'complete' && <span className="step-done">已完成</span>}
              {step?.state === 'error' && <span className="step-error">失败</span>}
            </li>
          })}
        </ol>
        {!Object.keys(pipeline).length && <p className="runtime-empty-note">Engine 尚未报告工具步骤。</p>}
      </section>

      <section className="runtime-panel runtime-usage">
        <div className="runtime-panel-heading"><h2><Coins size={15}/>本次用量</h2><span>项目累计在下方</span></div>
        <div className="runtime-usage-metrics">
          <article><span>输入 Token</span><strong>{runtime.inputTokens.toLocaleString()}</strong></article>
          <article><span>输出 Token</span><strong>{runtime.outputTokens.toLocaleString()}</strong></article>
          <article><span>本次费用</span><strong>${runtime.runCostUsd.toFixed(4)}</strong></article>
          <article><span>项目累计</span><strong>${runtime.projectCostUsd.toFixed(4)}</strong></article>
        </div>
        <div className="runtime-project-usage"><Cpu size={14}/><span>项目 Token</span><strong>{(runtime.projectInputTokens + runtime.projectOutputTokens).toLocaleString()}</strong></div>
      </section>

      <section className="runtime-panel runtime-agents">
        <div className="runtime-panel-heading"><h2><Sparkles size={15}/>Agent 状态</h2><span>{runtime.agents.length} 个</span></div>
        {runtime.agents.length ? <ul>{runtime.agents.map(agent => <li key={agent.name}>
          <i className={`agent-dot ${agent.state === 'working' ? 'active' : ''}`}/><strong>{agent.name}</strong>
          <span>{agent.tool ? labels[agent.tool] ?? agent.tool : agent.summary || (agent.state === 'working' ? '运行中' : '空闲')}</span>
        </li>)}</ul> : <p className="runtime-empty-note">Core 尚未提供 Agent 状态。</p>}
      </section>

      <section className="runtime-panel runtime-logs">
        <div className="runtime-panel-heading"><h2><SquareTerminal size={15}/>运行记录</h2><span>最近 {logs.length} 条</span></div>
        {logs.length ? <div className="runtime-log-list" role="log" aria-live="polite">{logs.slice(-120).map((log, index) => <LogRow key={`${log.ID}-${index}`} log={log}/>)}</div> : <p className="runtime-empty-note">Engine Session 启动后，运行事件会显示在这里。</p>}
      </section>
    </div>
  </section>
}
