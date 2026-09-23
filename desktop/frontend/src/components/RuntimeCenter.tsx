import { useMemo } from 'react'
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
  const {dirty, saveBusy, syncing, syncError, syncNotice, syncChapterRevisions} = useStudio()
  const canResume = !dirty && !syncing && revisionsAllowWriting(runtime.projectId, revisions.status, revisions.checking, revisions.error)
  const canResumeState = ['idle', 'paused', 'stopped', 'error', 'waiting_sync'].includes(runtime.state)
  const canSyncState = !['running', 'pausing', 'stopping'].includes(runtime.state)
  const canSync = canSyncState && revisions.status.hasUnsynced
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
        <div><span>本次运行时长</span><strong className="runtime-mono">{duration(runtime.elapsedSeconds)}</strong><small>{runtime.flow || '等待创作任务'}</small></div>
      </div>
    </section>

    {(syncError || runtime.error || controlError) && <div className="runtime-error" role="status"><CircleAlert size={16}/><span>{syncError || controlError || runtime.error}</span></div>}

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
