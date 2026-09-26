import { useEffect, useState } from 'react'
import { ArrowDownToLine, FilePlus2, FolderInput, Play, Save, Settings2 } from 'lucide-react'
import { useStudio } from '../store'
import { useCreateProjectStore } from '../createProjectStore'
import { useImportStore } from '../importStore'
import { useConfigStore } from '../configStore'
import { useExportStore } from '../exportStore'
import type { ConfigSnapshot, CreateEvent, ExportOptions } from '../types'
import { ModelSettingsEditor } from './ModelSettingsEditor'

const cardStyle = {maxWidth: 900, margin: '0 auto', padding: '30px'}

function AgentModelControls({config, onSwitch, onThinking}: {config: ConfigSnapshot; onSwitch: (role: string, provider: string, model: string) => void; onThinking: (role: string, level: string) => void}) {
  const models = config.providers.flatMap(provider => provider.models.map(model => ({provider: provider.name, model: model.name})))
  const levels = ['off', 'low', 'medium', 'high', 'xhigh', 'max']
  return <div className="panel"><h3>Agent Models</h3><p className="muted">仅修改下一次 Core Agent 调用；不会偷偷重启正在运行的 Engine。</p>{Object.entries(config.roles).map(([role, setting]) => <div key={role} style={{display: 'grid', gridTemplateColumns: 'minmax(130px, .7fr) 1fr 1fr', gap: 8, alignItems: 'center', marginTop: 8}}><strong>{role}</strong><select value={`${setting.provider}\0${setting.model}`} onChange={e => {const [provider, model] = e.target.value.split('\0'); onSwitch(role, provider, model)}}>{models.map(item => <option key={`${item.provider}/${item.model}`} value={`${item.provider}\0${item.model}`}>{item.provider} / {item.model}</option>)}</select><select value={setting.reasoningEffort || ''} onChange={e => onThinking(role, e.target.value)}><option value="">继承 Core 默认</option>{levels.map(level => <option key={level} value={level}>{level}</option>)}</select></div>)}</div>
}

export function CreateEventStatus({event}: {event: CreateEvent | null}) {
  if (event?.state !== 'completed' || !event.message) return null
  return <div className="panel" role="status">{event.message}</div>
}

export function CreateCenter() {
  const {open, project} = useStudio()
  const state = useCreateProjectStore()
  const [message, setMessage] = useState('')
  const [sent, setSent] = useState('')
  useEffect(() => { if (state.mode === 'cocreate' && project) void state.loadRecovery() }, [state.mode, project, state.loadRecovery])
  useEffect(() => { if (state.event?.state === 'completed' && state.event.project) void open(state.event.project.outputDir) }, [state.event, open])
  const submit = (event: React.FormEvent) => { event.preventDefault(); void state.start() }
  return <section style={cardStyle}>
    <p className="eyebrow">CREATE WORKSPACE</p><h1>{state.mode === 'cocreate' ? '共创创建' : state.mode === 'outline' ? '从大纲开始' : '快速开始'}</h1>
    <p className="muted">入口只提交 Core 已有的 Prompt / 大纲文件和项目目录，初始化、裁定与写盘仍由 ainovel Core 完成。</p>
    <div style={{display: 'flex', gap: 8, margin: '20px 0'}}>
      <button onClick={() => state.setMode('quick')} className={state.mode === 'quick' ? 'primary' : ''}><Play size={15}/>快速开始</button>
      <button onClick={() => state.setMode('outline')} className={state.mode === 'outline' ? 'primary' : ''}><FilePlus2 size={15}/>从大纲开始</button>
      <button onClick={() => state.setMode('cocreate')} className={state.mode === 'cocreate' ? 'primary' : ''}><Settings2 size={15}/>共创创建</button>
    </div>
    <form onSubmit={submit} style={{display: 'grid', gap: 12}}>
      <label>项目根目录<input value={state.projectRoot} onChange={e => state.setProjectRoot(e.target.value)} placeholder="例如 D:\\小说\\my-book" required/></label>
      <label>{state.mode === 'outline' ? '大纲文件路径' : '创作需求'}{state.mode === 'cocreate' ? <textarea className="chapter-editor" style={{minHeight: 120}} value={state.prompt} onChange={e => state.setPrompt(e.target.value)} placeholder="先告诉 Core 你想创作什么" required/> : <input value={state.prompt} onChange={e => state.setPrompt(e.target.value)} placeholder={state.mode === 'outline' ? '例如 D:\\小说\\outline.md' : '例如：写一部都市悬疑长篇…'} required/>}</label>
      {state.mode === 'cocreate' && <label><input type="checkbox" checked={state.stage} onChange={e => {state.setStage(e.target.checked); setMessage(e.target.checked ? '阶段共创需要当前已打开项目。' : '')}}/> 当前项目进入阶段共创</label>}
      {state.mode === 'outline' && <button type="button" disabled={state.previewing || !state.prompt.trim()} onClick={() => void state.previewOutline()}>{state.previewing ? '正在读取…' : '预览并校验大纲'}</button>}<button className="primary" disabled={state.busy}>{state.busy ? 'Core 正在处理…' : '提交给 Core'}</button>
    </form>
    {message && <p className="muted">{message}</p>}
    {state.error && <div className="editor-error" role="alert">{state.error}</div>}
    <CreateEventStatus event={state.event}/>
    {state.previewText && <div className="panel" style={{marginTop: 20}}><h3>Core 校验后的大纲预览</h3><pre className="prose">{state.previewText}</pre></div>}
    {state.mode === 'cocreate' && state.coCreate && <div className="panel" style={{marginTop: 20}}><h3>Core 共创回复</h3><p style={{whiteSpace: 'pre-wrap'}}>{state.coCreate.reply || state.coCreate.text || 'Core 正在整理…'}</p>{state.coCreate.draft && <details><summary>当前创作指令草稿</summary><pre className="prose">{state.coCreate.draft}</pre></details>}<div style={{display: 'flex', gap: 8, marginTop: 12}}><input value={sent} onChange={e => setSent(e.target.value)} placeholder="继续告诉 Core 你的想法"/><button disabled={state.busy || !state.ack || !sent.trim()} onClick={() => {void state.send(sent); setSent('')}}>发送</button>{state.coCreate.ready && state.ack && <button disabled={state.busy} onClick={() => void state.complete()}>完成并开始创作</button>}</div></div>}
    {state.recovery?.exists && <div className="panel" style={{marginTop: 20}}><h3>发现已有共创会话</h3><p>{state.recovery.interrupted ? '上一轮未完成，Core 已保留可恢复的最后一轮。' : 'Core 已保留最近共创记录。'}</p><small>{state.recovery.history?.length ?? 0} 条已落盘消息</small>{state.recovery.history?.length ? <details open><summary>恢复历史</summary><div className="prose">{state.recovery.history.map((item, index) => <p key={`${item.role}-${index}`}><strong>{item.role === 'assistant' ? 'Core' : '你'}：</strong>{item.content}</p>)}</div></details> : null}{state.recovery.draft && <details open><summary>已落盘草稿</summary><pre className="prose">{state.recovery.draft}</pre></details>}<button className="primary" disabled={state.busy} onClick={() => void state.resumeRecovery()}>继续上次共创</button></div>}
    {state.event?.error && <div className="editor-error">创建失败：{state.event.error}</div>}
  </section>
}

export function ImportCenter() {
  const {open} = useStudio()
  const state = useImportStore()
  const [options, setOptions] = useState({projectRoot: '', sourcePath: '', autoConfirm: false, acceptSegmentation: false, storyResolution: '', continueAfter: false, guidance: ''})
  useEffect(() => { void state.refresh() }, [])
  return <section style={cardStyle}>
    <p className="eyebrow">IMPORT NOVEL</p><h1>导入已有小说</h1>
    <p className="muted">阶段、章节识别、确认、分析、综合与发布全部来自 Core Import Pipeline。</p>
    <form onSubmit={e => {e.preventDefault(); void state.start(options)}} style={{display: 'grid', gap: 12}}>
      <label>新项目根目录（已有项目可留空）<input value={options.projectRoot} onChange={e => setOptions({...options, projectRoot: e.target.value})} placeholder="例如 D:\\小说\\imported"/></label>
      <label>源文件路径<input value={options.sourcePath} onChange={e => setOptions({...options, sourcePath: e.target.value})} placeholder="文本文件（UTF-8 / GB18030）" required/></label>
      <label>切分指导（Core 原生 guidance）<input value={options.guidance} onChange={e => setOptions({...options, guidance: e.target.value})}/></label>
      <div><label><input type="checkbox" checked={options.acceptSegmentation} onChange={e => setOptions({...options, acceptSegmentation: e.target.checked})}/> 已查看并接受章节识别结果</label><label><input type="checkbox" checked={options.continueAfter} onChange={e => setOptions({...options, continueAfter: e.target.checked})}/> 导入完成后按 Core 语义继续创作</label></div>
      <button className="primary" disabled={state.busy}><FolderInput size={15}/>{state.busy ? 'Core 正在导入…' : '开始导入'}</button>
    </form>
    {state.busy && <button onClick={() => void state.cancel()}>取消当前导入</button>}
    {state.error && <div className="editor-error">{state.error}</div>}
    {state.status && <div className="panel" style={{marginTop: 20}}><h3>{state.status.stage || '等待 Core'}</h3><p>{state.status.message || 'Core 正在读取导入工作区。'} {state.status.current || 0}/{state.status.total || 0}</p>{state.status.recoveryHint && <p className="muted">恢复提示：{state.status.recoveryHint}</p>}{state.status.chapters?.map(chapter => <div key={chapter.number}>{chapter.number}. {chapter.title} {chapter.uncertain ? ' · 需确认' : ''}</div>)}{state.status.stage === 'done' && <button onClick={() => void open(state.status?.projectId)}>打开导入结果</button>}</div>}
  </section>
}

export function SettingsCenter() {
  const state = useConfigStore()
  const [budget, setBudget] = useState({bookUsd: 0, warnRatio: .8, hardStop: false})
  useEffect(() => { void state.load(); void state.loadUsage() }, [])
  useEffect(() => { if (state.config) setBudget(state.config.budget) }, [state.config])
  const config = state.config
  return <section style={cardStyle}>
    <p className="eyebrow">MODEL & BUDGET</p><h1>模型与 Provider</h1>
    {state.error && <div className="editor-error">{state.error}</div>}
    {config ? <>
      <div className="panel"><h3>默认模型</h3><p>{config.provider} / {config.model}</p><p className="muted">Reasoning：{config.reasoningEffort || '继承 Core 默认'}</p><select value={`${config.provider}\0${config.model}`} onChange={e => {const [providerName, model] = e.target.value.split('\0'); void state.switchModel({role: 'default', provider: providerName, model})}}>{config.providers.flatMap(item => item.models.map(model => <option key={`${item.name}/${model.name}`} value={`${item.name}\0${model.name}`}>{item.name} / {model.name}</option>))}</select></div>
      <AgentModelControls config={config} onSwitch={(role, provider, model) => void state.switchModel({role, provider, model})} onThinking={(role, level) => void state.setThinking({role, level})}/>
      <ModelSettingsEditor config={config} busy={state.busy} onSave={state.saveProvider} onDelete={state.deleteProvider} onDiscover={state.discoverModels} onTest={state.testModelConnection}/>
      <div className="panel"><h3>Budget / Usage</h3><p>${state.usage?.overall.costUsd.toFixed(4) ?? '0.0000'} / ${budget.bookUsd.toFixed(2)} · 输入 {(state.usage?.overall.input ?? 0).toLocaleString()} · 输出 {(state.usage?.overall.output ?? 0).toLocaleString()}</p><label>Book Budget (USD)<input type="number" min="0" step="0.01" value={budget.bookUsd} onChange={e => setBudget({...budget, bookUsd: Number(e.target.value)})}/></label><label>Warn Ratio<input type="number" min="0.01" max="0.99" step="0.01" value={budget.warnRatio} onChange={e => setBudget({...budget, warnRatio: Number(e.target.value)})}/></label><label><input type="checkbox" checked={budget.hardStop} onChange={e => setBudget({...budget, hardStop: e.target.checked})}/> Hard Stop</label><button disabled={state.busy} onClick={() => void state.saveBudget(budget)}><Save size={15}/>保存预算（下一次 Host 生效）</button></div>
    </> : state.busy ? <p className="muted" role="status">正在读取 Core effective config…</p> : <div className="config-load-failed" role="alert"><strong>Core 配置读取失败</strong><p>{state.error || '没有收到配置结果。'}</p><button onClick={() => void state.load()}>重新读取 Core 配置</button></div>}
  </section>
}

export function ExportCenter() {
  const state = useExportStore(); const [options, setOptions] = useState<ExportOptions>({format: 'txt', outPath: '', from: 0, to: 0, overwrite: false})
  return <section style={cardStyle}><p className="eyebrow">EXPORT</p><h1>导出作品</h1><p className="muted">不在前端拼接文本或 EPUB，直接调用 Core Export Service。</p><div className="panel" style={{display: 'grid', gap: 12}}><label>格式<select value={options.format} onChange={e => setOptions({...options, format: e.target.value as 'txt' | 'epub'})}><option value="txt">TXT</option><option value="epub">EPUB</option></select></label><label>目标路径<input value={options.outPath} onChange={e => setOptions({...options, outPath: e.target.value})} placeholder="输出文件路径" required/></label><label>起始章节（0 = 全部）<input type="number" min="0" value={options.from} onChange={e => setOptions({...options, from: Number(e.target.value)})}/></label><label>结束章节（0 = 全部）<input type="number" min="0" value={options.to} onChange={e => setOptions({...options, to: Number(e.target.value)})}/></label><label><input type="checkbox" checked={options.overwrite} onChange={e => setOptions({...options, overwrite: e.target.checked})}/> 允许覆盖</label><button className="primary" disabled={state.busy} onClick={() => void state.exportProject(options)}><ArrowDownToLine size={15}/>{state.busy ? '正在导出…' : '导出'}</button></div>{state.error && <div className="editor-error">{state.error}</div>}{state.result && <div className="panel"><p>已导出：<code className="path">{state.result.path}</code></p><p>{state.result.chapters} 章 · {state.result.bytes.toLocaleString()} bytes</p></div>}</section>
}
