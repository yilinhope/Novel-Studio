import { useEffect, useState } from 'react'
import { ensureSimulationEvents, useSimulationStore } from '../simulationStore'
import { useWritingSettingsStore } from '../writingSettingsStore'
import type { RuleFile } from '../types'

export function SimulationCenter() {
  const {sources, profile, loading, running, error, event, load, start, importProfile, cancel} = useSimulationStore()
  const [tab, setTab] = useState<'sources' | 'profile'>('sources')
  useEffect(() => { ensureSimulationEvents(); void load() }, [load])
  return <section className="center-card">
    <header className="center-header"><div><p className="eyebrow">CORE SIMULATION</p><h1>参考作品 / 仿写画像</h1><p>语料扫描、画像生成与导入都由 Core 完成，界面不复制或解析业务文件。</p></div><span className="source-pill">{running ? 'Core 运行中' : 'Core 数据'}</span></header>
    <div className="parity-tabs" role="tablist" aria-label="仿写画像视图"><button className={tab === 'sources' ? 'selected' : ''} onClick={() => setTab('sources')}>参考源</button><button className={tab === 'profile' ? 'selected' : ''} onClick={() => setTab('profile')}>Simulation Profile</button></div>
    {error && <div className="error" role="alert">{error}</div>}
    <div className="wave-actions"><button className="primary" disabled={running || loading} onClick={() => void start()}>运行分析</button><button disabled={running} onClick={() => void importProfile()}>导入 Profile</button>{running && <button onClick={() => void cancel()}>取消</button>}<button disabled={loading || running} onClick={() => void load()}>刷新</button></div>
    {event && <div className={`wave-event ${event.state === 'error' ? 'is-error' : ''}`}><strong>{event.state === 'running' ? '进行中' : event.state === 'completed' ? '已完成' : '失败'}</strong><span>{event.stage}</span><p>{event.error || event.message}</p>{event.total > 0 && <small>{event.current} / {event.total}</small>}</div>}
    {loading ? <div className="parity-state"><span className="spinner"/>正在读取 Core 仿写数据…</div> : tab === 'sources' ? <div className="parity-list">{sources?.items.length ? sources.items.map(source => <article className="parity-card rule-row" key={source.relativePath}><span className="category">{source.relativePath.endsWith('.md') || source.relativePath.endsWith('.markdown') ? 'Markdown' : 'TXT'}</span><div><h3>{source.relativePath}</h3><p>{source.sizeBytes.toLocaleString()} bytes · {source.sha256.slice(0, 12)}…</p><small>{source.changed ? '源文件已变化，等待重新分析' : source.analyzedAt ? `已分析于 ${source.analyzedAt}` : '尚未分析'}</small></div></article>) : <div className="parity-empty"><span>—</span><p>simulate 目录暂无 .txt / .md / .markdown 语料。</p></div>}</div> : profile?.available && profile.profile ? <article className="parity-card wave-profile"><p className="eyebrow">Core Simulation Profile</p><p>版本：{profile.profile.version} · 更新时间：{profile.profile.updated_at || '—'}</p><p>已登记源文件：{profile.profile.corpus.sources.length} · 来源报告：{profile.profile.source_reports.length}</p><div className="wave-report-list">{profile.profile.source_reports.slice(0, 12).map((report, index) => <details key={String(report.relative_path ?? index)}><summary>{String(report.title ?? report.relative_path ?? `报告 ${index + 1}`)}</summary>{typeof report.summary === 'string' && <p>{report.summary}</p>}{Array.isArray(report.warnings) && report.warnings.length > 0 && <p className="muted">警告：{report.warnings.map(String).join('；')}</p>}</details>)}</div></article> : <div className="parity-empty"><span>—</span><p>Core 尚未保存 Simulation Profile。</p></div>}
  </section>
}

export function WritingSettingsCenter() {
  const state = useWritingSettingsStore()
  const [tab, setTab] = useState<'rules' | 'style'>('rules')
  useEffect(() => { void state.load() }, [])
  if (state.loading && !state.rules && !state.style) return <div className="parity-state"><span className="spinner"/>正在读取 Core 写作设置…</div>
  return <section className="center-card"><header className="center-header"><div><p className="eyebrow">CORE WRITING SETTINGS</p><h1>写作规则 / 文风</h1><p>显示 Core 当前规则、Style、Voice 与 anti-AI-tone；修改后需新建 Host 才会进入运行时。</p></div><span className="source-pill">Core 配置</span></header>
    <div className="parity-tabs"><button className={tab === 'rules' ? 'selected' : ''} onClick={() => setTab('rules')}>写作规则</button><button className={tab === 'style' ? 'selected' : ''} onClick={() => setTab('style')}>文风 / Voice</button></div>
    {state.error && <div className="error" role="alert">{state.error}</div>}
    {tab === 'rules' ? <RulesPanel/> : <StylePanel/>}
  </section>
}

function RulesPanel() {
  const {rules, selectedScope, selectedRule, dirty, selectRule, setDraft, saveRule, createRule, deleteRule, renameRule} = useWritingSettingsStore()
  const choose = (scope: 'global' | 'project', name: string) => { if (dirty && !window.confirm('当前规则有未保存修改，继续将放弃修改，是否继续？')) return; void selectRule(scope, name) }
  const create = (scope: 'global' | 'project') => { const name = window.prompt('规则文件名（例如 style.md）', 'new-rule.md'); const content = name ? window.prompt('规则内容', '') : null; if (name && content !== null) void createRule(scope, name, content) }
  const list = (items: RuleFile[], scope: 'global' | 'project') => items.map(item => <button className={`rule-file ${selectedScope === scope && selectedRule?.name === item.name ? 'selected' : ''}`} key={`${scope}-${item.name}`} onClick={() => choose(scope, item.name)}>{item.name}<small>{Math.round(item.sizeBytes / 1024 * 10) / 10} KB</small></button>)
  return <div><div className="wave-columns"><div className="parity-card"><h2>Global Rules</h2><button onClick={() => create('global')}>新建</button>{list(rules?.global ?? [], 'global')} {!rules?.global.length && <p className="muted">暂无全局规则。</p>}</div><div className="parity-card"><h2>Project Rules</h2><button onClick={() => create('project')}>新建</button>{list(rules?.project ?? [], 'project')} {!rules?.project.length && <p className="muted">暂无项目规则。</p>}</div><div className="parity-card"><h2>Effective Rules</h2>{rules?.effectiveAvailable && rules.effective ? <><p>状态：{rules.effective.status}</p><p className="long-text">{rules.effective.preferences || 'Core 当前没有自然语言偏好。'}</p><div className="chips">{rules.effective.sources.map(source => <span key={source}>{source}</span>)}</div></> : <p className="muted">Core 尚未生成有效规则快照。</p>}</div></div>{rules?.effectiveNotice && <p className="muted">{rules.effectiveNotice}</p>}{selectedRule && <div className="parity-card rule-editor"><h2>{selectedRule.scope === 'global' ? '全局' : '项目'}规则：{selectedRule.name}</h2><textarea value={selectedRule.content ?? ''} onChange={event => setDraft(event.target.value)} rows={12}/><div className="wave-actions"><button className="primary" disabled={!dirty} onClick={() => void saveRule()}>保存规则</button><button onClick={() => { const name = window.prompt('新文件名', selectedRule.name); if (name) void renameRule(selectedRule.scope, selectedRule.name, name) }}>重命名</button><button onClick={() => void deleteRule(selectedRule.scope, selectedRule.name)}>删除</button></div></div>}</div>
}

function StylePanel() {
  const {style, selectStyle, saveAsset, deleteAsset} = useWritingSettingsStore()
  const [voice, setVoice] = useState(style?.voiceProject ?? '')
  const [anti, setAnti] = useState(style?.antiAiToneProject ?? '')
  const [scope, setScope] = useState<'global' | 'project'>('project')
  const scopedVoice = scope === 'global' ? style?.voiceGlobal ?? '' : style?.voiceProject ?? ''
  const scopedAnti = scope === 'global' ? style?.antiAiToneGlobal ?? '' : style?.antiAiToneProject ?? ''
  useEffect(() => { setVoice(scopedVoice); setAnti(scopedAnti) }, [scopedVoice, scopedAnti])
  if (!style) return <div className="parity-empty"><span>—</span><p>Core 尚未提供文风状态。</p></div>
  const remove = (title: string, name: string) => { if (window.confirm(`删除当前${scope === 'global' ? '全局' : '项目'}层的 ${title} 覆盖并恢复继承？`)) void deleteAsset(scope, name) }
  return <div><div className="parity-card"><h2>Style</h2><select value={style.selectedStyle} onChange={event => void selectStyle(event.target.value)}>{style.styleNames.map(name => <option key={name} value={name}>{name}</option>)}</select><p className="muted">来源：{style.styleSource}</p><p className="long-text">{style.selectedStyleText || 'Core 未提供当前 Style 文本。'}</p>{style.genreReference && <><p className="muted">题材参考来源：{style.genreReferenceSource}</p><p className="long-text">{style.genreReference}</p></>}</div><div className="parity-card"><label>编辑覆盖范围 <select value={scope} onChange={event => setScope(event.target.value as 'global' | 'project')}><option value="project">项目</option><option value="global">全局</option></select></label><p className="muted">编辑器只修改当前范围的原始覆盖，空白内容请使用删除覆盖来恢复继承。</p></div><div className="wave-columns"><AssetEditor title="Voice" effective={style.effectiveVoice} effectiveSource={style.effectiveVoiceSource} scope={scope} value={voice} onChange={setVoice} onSave={() => void saveAsset(scope, 'voice.md', voice)} onDelete={() => remove('Voice', 'voice.md')}/><AssetEditor title="Anti-AI-Tone" effective={style.effectiveAntiAiTone} effectiveSource={style.effectiveAntiAiToneSource} scope={scope} value={anti} onChange={setAnti} onSave={() => void saveAsset(scope, 'anti-ai-tone.md', anti)} onDelete={() => remove('anti-AI-tone', 'anti-ai-tone.md')}/></div><p className="muted">{style.effectiveNotice}</p></div>
}

function AssetEditor({title, effective, effectiveSource, scope, value, onChange, onSave, onDelete}: {title: string; effective: string; effectiveSource: string; scope: 'global' | 'project'; value: string; onChange: (value: string) => void; onSave: () => void; onDelete: () => void}) { return <div className="parity-card rule-editor"><h2>{title}</h2><p className="muted">当前生效来源：{effectiveSource}</p><textarea value={effective} readOnly rows={6} aria-label={`${title} 当前生效预览`}/><h3>{scope === 'global' ? '全局' : '项目'}层原始覆盖</h3><textarea value={value} onChange={event => onChange(event.target.value)} rows={8} aria-label={`${title} ${scope} 层编辑`}/><div className="wave-actions"><button className="primary" onClick={onSave}>保存 {title} 覆盖</button><button onClick={onDelete}>删除此层覆盖 / 恢复继承</button></div></div> }
