import { useEffect, useMemo, useState } from 'react'
import { CheckCircle2, Plus, RefreshCw, Save, Trash2, Zap } from 'lucide-react'
import type { ConfigModel, ConfigSnapshot, ProviderDraft } from '../types'

type SaveProvider = (draft: ProviderDraft) => Promise<void>
type DiscoverModels = (draft: ProviderDraft) => Promise<ConfigModel[]>
type TestModelConnection = (draft: ProviderDraft, model: string) => Promise<void>

const providerDefaults: Record<string, {type: string; api: string; baseUrl: string}> = {
  openai: {type: 'openai', api: 'chat', baseUrl: 'https://api.openai.com/v1'},
  deepseek: {type: 'openai', api: 'chat', baseUrl: 'https://api.deepseek.com/v1'},
  gemini: {type: 'gemini', api: 'chat', baseUrl: 'https://generativelanguage.googleapis.com'},
  ollama: {type: 'openai', api: 'chat', baseUrl: 'http://127.0.0.1:11434/v1'},
}

const emptyModel = (): ConfigModel => ({name: '', contextWindow: 0})

function cloneModels(models: ConfigModel[] | undefined): ConfigModel[] {
  return (models ?? []).map(model => ({name: model.name, contextWindow: model.contextWindow, jsonSchema: model.jsonSchema}))
}

export function ModelSettingsEditor({
  config,
  busy,
  onSave,
  onDiscover,
  onTest,
}: {
  config: ConfigSnapshot
  busy: boolean
  onSave: SaveProvider
  onDiscover: DiscoverModels
  onTest: TestModelConnection
}) {
  const [providerName, setProviderName] = useState('')
  const [providerType, setProviderType] = useState('')
  const [providerAPI, setProviderAPI] = useState('chat')
  const [baseUrl, setBaseUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [models, setModels] = useState<ConfigModel[]>([])
  const [editingNew, setEditingNew] = useState(false)
  const [saving, setSaving] = useState(false)
  const [discovering, setDiscovering] = useState(false)
  const [testing, setTesting] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const provider = useMemo(() => config.providers.find(item => item.name === providerName), [config.providers, providerName])

  const loadProvider = (name: string) => {
    const selected = config.providers.find(item => item.name === name)
    setProviderName(name)
    setProviderType(selected?.type ?? '')
    setProviderAPI(selected?.api || 'chat')
    setBaseUrl(selected?.baseUrl ?? '')
    setApiKey('')
    setModels(cloneModels(selected?.models))
    setEditingNew(false)
    setMessage('')
    setError('')
  }

  useEffect(() => {
    if (!editingNew && !providerName && config.providers[0]) loadProvider(config.providers[0].name)
  }, [config.providers, editingNew, providerName])

  const startNew = () => {
    setEditingNew(true)
    setProviderName('')
    setProviderType('openai')
    setProviderAPI('chat')
    setBaseUrl('https://api.openai.com/v1')
    setApiKey('')
    setModels([emptyModel()])
    setMessage('')
    setError('')
  }

  const applyProviderPreset = (name: string) => {
    const preset = providerDefaults[name.trim().toLowerCase()]
    if (!preset || baseUrl.trim()) return
    setProviderType(preset.type)
    setProviderAPI(preset.api)
    setBaseUrl(preset.baseUrl)
  }

  const draft = (): ProviderDraft => ({
    provider: providerName.trim(),
    type: providerType,
    api: providerAPI,
    baseUrl: baseUrl.trim(),
    models: models.filter(model => model.name.trim()).map(model => ({
      name: model.name.trim(),
      contextWindow: model.contextWindow && model.contextWindow > 0 ? model.contextWindow : 0,
      jsonSchema: model.jsonSchema,
    })),
    apiKeyAction: apiKey.trim() ? 'replace' : 'keep',
    apiKey: apiKey.trim(),
  })

  const save = async () => {
    const request = draft()
    if (!request.provider) { setError('Provider 名称不能为空。'); return }
    if (!request.models.length) { setError('请至少填写一个模型，或先读取模型列表。'); return }
    setSaving(true)
    setError('')
    setMessage('')
    try {
      await onSave(request)
      setEditingNew(false)
      setMessage('Provider 配置已保存；当前角色模型仍按 Core 选择。')
    } catch (cause) {
      setError(String(cause))
    } finally {
      setSaving(false)
    }
  }

  const discover = async () => {
    const request = draft()
    if (!request.provider) { setError('请先填写 Provider 名称。'); return }
    if (!request.baseUrl) { setError('请先填写 Base URL。'); return }
    setDiscovering(true)
    setError('')
    setMessage('')
    try {
      const discovered = await onDiscover(request)
      setModels(discovered)
      setMessage(`已读取 ${discovered.length} 个模型；确认后点击保存写入 Core 配置。`)
    } catch (cause) {
      setError(String(cause))
    } finally {
      setDiscovering(false)
    }
  }

  const test = async (model: ConfigModel) => {
    const name = model.name.trim()
    if (!name) { setError('请先填写要测试的模型名称。'); return }
    setTesting(name)
    setError('')
    setMessage('')
    try {
      await onTest(draft(), name)
      setMessage(`连接测试成功：${providerName} / ${name}`)
    } catch (cause) {
      setError(String(cause))
    } finally {
      setTesting('')
    }
  }

  return <div className="panel" style={{display: 'grid', gap: 12}}>
    <div style={{display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'center'}}>
      <div><h3 style={{margin: 0}}>{editingNew ? '新建模型配置' : 'Provider / 模型配置'}</h3><p className="muted" style={{margin: '4px 0 0'}}>配置保存在当前项目的 Core 配置层；前端不直接读写 JSON。</p></div>
      {!editingNew && <button onClick={startNew} disabled={busy || saving}><Plus size={14}/>新建配置</button>}
    </div>
    {!editingNew && config.providers.length > 0 && <label>已保存配置<select value={providerName} onChange={event => loadProvider(event.target.value)}>{config.providers.map(item => <option key={item.name} value={item.name}>{item.name}{item.hasApiKey ? ' · 已配置 Key' : ' · 未配置 Key'}</option>)}</select></label>}
    <div style={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)', gap: 10}}>
      <label>显示名称 / Provider ID<input value={providerName} onChange={event => {setProviderName(event.target.value); applyProviderPreset(event.target.value)}} placeholder="例如 openai、deepseek、my-proxy" disabled={!editingNew}/></label>
      <label>调用协议<select value={providerType} onChange={event => setProviderType(event.target.value)}><option value="">Core 自动判断</option><option value="openai">OpenAI-compatible</option><option value="gemini">Gemini</option><option value="anthropic">Anthropic</option></select></label>
    </div>
    <label>API endpoint<select value={providerAPI} onChange={event => setProviderAPI(event.target.value)}><option value="chat">Chat Completions</option><option value="responses">Responses</option></select></label>
    <label>Base URL<input value={baseUrl} onChange={event => setBaseUrl(event.target.value)} placeholder="https://api.openai.com/v1"/></label>
    <label>API Key<input type="password" value={apiKey} onChange={event => setApiKey(event.target.value)} placeholder={provider?.hasApiKey ? `留空保持现有 Key（${provider.apiKeyHint || '已配置'}）` : 'sk-…'} autoComplete="new-password"/></label>
    <div className="panel" style={{display: 'grid', gap: 8, margin: 0}}>
      <div style={{display: 'flex', justifyContent: 'space-between', gap: 8, alignItems: 'center'}}><strong>模型列表</strong><div style={{display: 'flex', gap: 8}}><button onClick={() => void discover()} disabled={busy || saving || discovering || !providerName.trim() || !baseUrl.trim()}><RefreshCw size={14}/>{discovering ? '读取中…' : '读取模型列表'}</button><button onClick={() => setModels([...models, emptyModel()])} disabled={busy || saving}><Plus size={14}/>手动添加</button></div></div>
      <p className="muted" style={{margin: 0}}>读取结果不会自动保存；确认列表后点击下方保存。Ollama 等本地服务可直接手动输入模型名。</p>
      {models.map((model, index) => <div key={`${index}-${model.name}`} style={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) 130px 120px auto auto', gap: 8, alignItems: 'center'}}><input value={model.name} onChange={event => setModels(models.map((item, itemIndex) => itemIndex === index ? {...item, name: event.target.value} : item))} placeholder="model（模型名称）"/><input type="number" min="0" value={model.contextWindow || ''} onChange={event => setModels(models.map((item, itemIndex) => itemIndex === index ? {...item, contextWindow: event.target.value ? Number(event.target.value) : 0} : item))} placeholder="上下文窗口"/><select value={model.jsonSchema === undefined ? '' : model.jsonSchema ? 'true' : 'false'} onChange={event => setModels(models.map((item, itemIndex) => itemIndex === index ? {...item, jsonSchema: event.target.value === '' ? undefined : event.target.value === 'true'} : item))}><option value="">JSON Schema 自动</option><option value="true">支持 JSON Schema</option><option value="false">不支持 JSON Schema</option></select><button onClick={() => void test(model)} disabled={busy || saving || discovering || testing !== '' || !model.name.trim()} title="测试当前草稿，不保存"><Zap size={14}/>{testing === model.name.trim() ? '测试中…' : '测试'}</button><button onClick={() => setModels(models.filter((_, itemIndex) => itemIndex !== index))} disabled={busy || saving || models.length <= 1} title="移除模型"><Trash2 size={14}/></button></div>)}
    </div>
    <div style={{display: 'flex', gap: 8, alignItems: 'center'}}><button className="primary" onClick={() => void save()} disabled={busy || saving || discovering || testing !== ''}><Save size={14}/>{saving ? '保存中…' : '保存 Provider 配置'}</button>{message && <span className="muted" role="status"><CheckCircle2 size={14}/> {message}</span>}</div>
    {error && <div className="editor-error" role="alert">{error}</div>}
  </div>
}
