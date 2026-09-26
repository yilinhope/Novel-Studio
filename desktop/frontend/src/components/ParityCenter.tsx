import { useParityStore } from '../parityStore'
import type {
  CastPage, CharacterPage, ForeshadowPage, LayeredChapterPage, LayeredOutlinePage, OutlinePage,
  RelationshipPage, SnapshotPage, StateChangePage, StoryCompass, StoryPremise, StorySummaryPage,
  SnapshotScope, TimelinePage, WorldRulePage,
} from '../types'
import './parity.css'

const title: Record<string, string> = {
  premise: '故事前提', characters: '人物', world: '世界观', outline: '全书大纲', compass: '当前方向', summaries: '故事摘要',
  timeline: '时间线', foreshadow: '伏笔', relationships: '人物关系', states: '人物状态', snapshots: '角色快照', cast: '首次出场',
}

export function ParityCenter() {
  const {section, data, loading, error, offset, selectArc, page, selectedVolume, selectedArc, summaryScope, selectSummaryScope, outlineMode, selectOutlineMode, layeredOutline, snapshotScopes, selectedSnapshotVolume, selectedSnapshotArc, selectSnapshotScope} = useParityStore()
  const pageInfo = data && 'total' in data ? data : null
  return <div className="parity-center">
    <header className="parity-header"><div><p className="eyebrow">CORE READ ONLY</p><h1>{title[section]}</h1><p>内容直接读取当前项目的 Core Store。缺失数据会明确显示为空，不由界面补算。</p></div><span className="source-pill">Core 数据</span></header>
    {section === 'summaries' && <div className="parity-tabs" role="tablist" aria-label="摘要范围">{(['chapter','arc','volume'] as const).map(scope => <button key={scope} className={summaryScope === scope ? 'selected' : ''} onClick={() => void selectSummaryScope(scope)}>{scope === 'chapter' ? '章节摘要' : scope === 'arc' ? '弧摘要' : '卷摘要'}</button>)}</div>}
    {section === 'outline' && <div className="parity-tabs" role="tablist" aria-label="大纲视图"><button className={outlineMode === 'layered' ? 'selected' : ''} onClick={() => void selectOutlineMode('layered')}>分层卷 / 弧大纲</button><button className={outlineMode === 'flat' ? 'selected' : ''} onClick={() => void selectOutlineMode('flat')}>扁平全书大纲</button></div>}
    {section === 'outline' && layeredOutline && <LayerSelector data={layeredOutline} onSelect={selectArc} selectedVolume={selectedVolume} selectedArc={selectedArc}/>}
    {section === 'snapshots' && <SnapshotScopeSelector scopes={snapshotScopes} selectedVolume={selectedSnapshotVolume} selectedArc={selectedSnapshotArc} onSelect={selectSnapshotScope}/ >}
    {loading && <div className="parity-state"><span className="spinner"/>正在读取 Core 数据…</div>}
    {!loading && error && <div className="parity-state error" role="alert"><strong>读取失败</strong><p>{error}</p><button onClick={() => void useParityStore.getState().selectSection(section)}>重新读取</button></div>}
    {!loading && !error && data && <DataView section={section} data={data} outlineMode={outlineMode}/ >}
    {!loading && !error && pageInfo && <div className="parity-pagination"><span>共 {pageInfo.total.toLocaleString()} 项</span><div><button disabled={offset <= 0} onClick={() => void page(Math.max(0, offset - pageInfo.limit))}>上一页</button><span>{pageInfo.total === 0 ? '0' : `${offset + 1}–${Math.min(offset + pageInfo.limit, pageInfo.total)}`}</span><button disabled={!pageInfo.hasMore} onClick={() => void page(offset + pageInfo.limit)}>下一页</button></div></div>}
  </div>
}

function LayerSelector({data, onSelect, selectedVolume, selectedArc}: {data: LayeredOutlinePage; onSelect: (volume: number, arc: number) => void; selectedVolume: number; selectedArc: number}) {
  return <div className="layer-selector"><strong>分层大纲</strong>{data.items.map(volume => <section key={volume.index}><h3>第 {volume.index} 卷 · {volume.title}</h3>{volume.theme && <p>{volume.theme}</p>}<div className="layer-arcs">{volume.arcs.map(arc => <button key={`${volume.index}-${arc.index}`} className={selectedVolume === volume.index && selectedArc === arc.index ? 'selected' : ''} onClick={() => onSelect(volume.index, arc.index)}><span>第 {arc.index} 弧 · {arc.title}</span><small>{arc.chapterCount ? `${arc.chapterCount} 章` : `预计 ${arc.estimatedChapters || 0} 章`}</small></button>)}</div></section>)}</div>
}

function SnapshotScopeSelector({scopes, selectedVolume, selectedArc, onSelect}: {scopes: SnapshotScope[]; selectedVolume: number; selectedArc: number; onSelect: (volume: number, arc: number) => void}) {
  return <div className="parity-tabs snapshot-scopes" role="tablist" aria-label="角色快照范围"><span className="scope-label">快照范围</span><button className={selectedVolume === 0 && selectedArc === 0 ? 'selected' : ''} onClick={() => void onSelect(0, 0)}>最新角色快照</button>{scopes.map(scope => <button key={`${scope.volume}-${scope.arc}`} className={selectedVolume === scope.volume && selectedArc === scope.arc ? 'selected' : ''} onClick={() => void onSelect(scope.volume, scope.arc)}>第 {scope.volume} 卷 · 第 {scope.arc} 弧{scope.title ? ` · ${scope.title}` : ''}</button>)}</div>
}

function DataView({section, data, outlineMode}: {section: string; data: NonNullable<ReturnType<typeof useParityStore.getState>['data']>; outlineMode: 'layered' | 'flat'}) {
  switch (section) {
    case 'premise': return <Premise data={data as StoryPremise}/>
    case 'characters': return <Characters data={data as CharacterPage}/>
    case 'world': return <WorldRules data={data as WorldRulePage}/>
    case 'outline': return 'volume' in data && 'arc' in data ? <LayeredChapters data={data as LayeredChapterPage}/> : outlineMode === 'layered' ? <LayeredOutline data={data as LayeredOutlinePage}/> : <Outline data={data as OutlinePage}/>
    case 'compass': return <Compass data={data as StoryCompass}/>
    case 'summaries': return <Summaries data={data as StorySummaryPage}/>
    case 'timeline': return <Timeline data={data as TimelinePage}/>
    case 'foreshadow': return <Foreshadows data={data as ForeshadowPage}/>
    case 'relationships': return <Relationships data={data as RelationshipPage}/>
    case 'states': return <States data={data as StateChangePage}/>
    case 'snapshots': return <Snapshots data={data as SnapshotPage}/>
    case 'cast': return <Cast data={data as CastPage}/>
    default: return <Empty label="暂无可显示的数据"/>
  }
}

function Premise({data}: {data: StoryPremise}) {
  return <div className="parity-columns">{data.bookAvailable ? <article className="parity-card"><p className="eyebrow">作品简介</p><h2>{data.book.title}</h2><p>{data.book.synopsis}</p></article> : <Empty label="Core 中尚无作品资料"/>}{data.premiseAvailable ? <article className="parity-card premise-card"><p className="eyebrow">创作前提</p><p className="long-text">{data.premise}</p></article> : <Empty label="Core 中尚无故事前提"/>}</div>
}

function Characters({data}: {data: CharacterPage}) {
  if (!data.items.length) return <Empty label="Core 中尚无人物档案"/>
  return <div className="parity-grid">{data.items.map((person, index) => <article className="parity-card" key={`${person.name}-${index}`}><div className="person-title"><div className="avatar">{person.name.slice(0, 1)}</div><div><h2>{person.name}</h2><span>{person.role || '角色'}</span></div><small>{person.tier || ''}</small></div>{person.aliases?.length ? <p className="detail-line"><b>别名</b>{person.aliases.join('、')}</p> : null}<p className="long-text">{person.description || 'Core 未提供人物描述。'}</p>{person.arc && <p className="detail-line"><b>人物弧线</b>{person.arc}</p>}{person.traits?.length ? <div className="chips">{person.traits.map(trait => <span key={trait}>{trait}</span>)}</div> : null}</article>)}</div>
}

function WorldRules({data}: {data: WorldRulePage}) {
  if (!data.items.length) return <Empty label="Core 中尚无世界规则"/>
  return <div className="parity-list">{data.items.map((rule, index) => <article className="parity-card rule-row" key={`${rule.category}-${index}`}><span className="category">{rule.category || '未分类'}</span><div><h3>{rule.rule}</h3>{rule.boundary && <p>{rule.boundary}</p>}</div></article>)}</div>
}

function Outline({data}: {data: OutlinePage}) {
  if (!data.items.length) return <Empty label="Core 中尚无扁平大纲"/>
  return <div className="parity-list">{data.items.map(item => <article className="parity-card outline-row" key={item.chapter}><span className="chapter-number">{String(item.chapter).padStart(2, '0')}</span><div><h3>{item.title || `第 ${item.chapter} 章`}</h3>{item.coreEvent && <p>{item.coreEvent}</p>}{item.scenes?.length ? <div className="chips">{item.scenes.map((scene, index) => <span key={`${item.chapter}-${index}`}>{scene}</span>)}</div> : null}</div>{item.hook && <aside><small>章末钩子</small><p>{item.hook}</p></aside>}</article>)}</div>
}

function LayeredOutline({data}: {data: LayeredOutlinePage}) {
  if (!data.items.length) return <Empty label="Core 中尚无分层卷弧大纲"/>
  return <div className="parity-list">{data.items.map(volume => <article className="parity-card layered-volume" key={volume.index}><div><p className="eyebrow">第 {volume.index} 卷{volume.final ? ' · 完结卷' : ''}</p><h2>{volume.title || `第 ${volume.index} 卷`}</h2>{volume.theme && <p>{volume.theme}</p>}</div><div className="layered-arc-list">{volume.arcs.length ? volume.arcs.map(arc => <div className="layered-arc" key={`${volume.index}-${arc.index}`}><strong>第 {arc.index} 弧 · {arc.title}</strong><span>{arc.goal || 'Core 未提供弧目标'} · {arc.chapterCount ? `${arc.chapterCount} 章` : `预计 ${arc.estimatedChapters || 0} 章`}</span></div>) : <p className="muted">Core 尚无弧分解。</p>}</div></article>)}</div>
}

function LayeredChapters({data}: {data: LayeredChapterPage}) {
  if (!data.items.length) return <Empty label="此弧尚无细化章节"/>
  return <div className="parity-list">{data.items.map(item => <article className="parity-card outline-row" key={item.chapter}><span className="chapter-number">{String(item.chapter).padStart(2, '0')}</span><div><h3>{item.title || `第 ${item.chapter} 章`}</h3>{item.coreEvent && <p>{item.coreEvent}</p>}{item.scenes?.length ? <div className="chips">{item.scenes.map((scene, index) => <span key={`${item.chapter}-${index}`}>{scene}</span>)}</div> : null}</div>{item.hook && <aside><small>章末钩子</small><p>{item.hook}</p></aside>}</article>)}</div>
}

function Compass({data}: {data: StoryCompass}) {
  if (!data.available) return <Empty label="Core 尚未提供 Story Compass"/>
  return <article className="parity-card compass-card"><p className="eyebrow">終局方向</p><h2>{data.endingDirection}</h2><div className="compass-meta"><span>预计规模：{data.estimatedScale || '未指定'}</span><span>更新至第 {data.lastUpdated || 0} 章</span></div>{data.openThreads?.length ? <><h3>尚待收束</h3><div className="thread-list">{data.openThreads.map((thread, index) => <p key={`${thread}-${index}`}><span>{String(index + 1).padStart(2, '0')}</span>{thread}</p>)}</div></> : <p className="muted">Core 未记录开放线索。</p>}</article>
}

function Summaries({data}: {data: StorySummaryPage}) {
  const items = data.scope === 'chapter' ? data.chapters ?? [] : data.scope === 'arc' ? data.arcs ?? [] : data.volumes ?? []
  if (!items.length) return <Empty label="Core 中尚无此范围的摘要"/>
  return <div className="parity-list">{items.map((item, index) => {
    const label = 'chapter' in item ? `第 ${item.chapter} 章` : 'arc' in item ? `第 ${item.volume} 卷 · 第 ${item.arc} 弧` : `第 ${item.volume} 卷`
    const characters = 'characters' in item ? item.characters : undefined
    const keyEvents = item.keyEvents
    return <article className="parity-card summary-row" key={`${label}-${index}`}><p className="eyebrow">{label}</p><h3>{item.title}</h3><p>{item.summary}</p>{characters?.length ? <div className="chips">{characters.map(name => <span key={name}>{name}</span>)}</div> : null}{keyEvents?.length ? <ul>{keyEvents.map((event, eventIndex) => <li key={`${label}-${eventIndex}`}>{event}</li>)}</ul> : null}</article>
  })}</div>
}

function Timeline({data}: {data: TimelinePage}) {
  if (!data.items.length) return <Empty label="Core 中尚无时间线事件"/>
  return <div className="timeline-list">{data.items.map((event, index) => <article key={`${event.chapter}-${index}`}><div className="timeline-marker"/><span>第 {event.chapter} 章</span><div className="timeline-content"><small>{event.time || '时间未注明'}</small><h3>{event.event}</h3>{event.characters?.length ? <div className="chips">{event.characters.map(name => <span key={name}>{name}</span>)}</div> : null}</div></article>)}</div>
}

function Foreshadows({data}: {data: ForeshadowPage}) {
  if (!data.items.length) return <Empty label="Core 中尚无伏笔记录"/>
  return <div className="parity-grid">{data.items.map(item => <article className="parity-card" key={item.id}><div className="foreshadow-title"><h3>{item.description}</h3><span className={`status-chip status-${item.status}`}>{statusLabel(item.status)}</span></div><p className="muted">埋设于第 {item.plantedAt} 章{item.resolvedAt ? ` · 回收于第 ${item.resolvedAt} 章` : ''}</p><small>Core ID：{item.id}</small></article>)}</div>
}

function Relationships({data}: {data: RelationshipPage}) {
  if (!data.items.length) return <Empty label="Core 中尚无人物关系记录"/>
  return <div className="parity-grid relation-grid">{data.items.map((item, index) => <article className="parity-card" key={`${item.characterA}-${item.characterB}-${index}`}><div className="relation-pair"><span>{item.characterA}</span><i>↔</i><span>{item.characterB}</span></div><h3>{item.relation}</h3><small>记录于第 {item.chapter} 章</small></article>)}</div>
}

function States({data}: {data: StateChangePage}) {
  if (!data.items.length) return <Empty label="Core 中尚无状态变化记录"/>
  return <div className="parity-list">{data.items.map((item, index) => <article className="parity-card state-row" key={`${item.chapter}-${item.entity}-${index}`}><span className="chapter-number">{item.chapter}</span><div><h3>{item.entity}<small>{item.field}</small></h3><p>{item.oldValue ? `${item.oldValue} → ` : ''}<strong>{item.newValue}</strong></p>{item.reason && <small>{item.reason}</small>}</div></article>)}</div>
}

function Snapshots({data}: {data: SnapshotPage}) {
  if (!data.items.length) return <Empty label="Core 中尚无角色快照"/>
  return <div className="parity-grid">{data.items.map((item, index) => <article className="parity-card" key={`${item.volume}-${item.arc}-${item.name}-${index}`}><p className="eyebrow">第 {item.volume} 卷 · 第 {item.arc} 弧</p><h3>{item.name}</h3><p>{item.status}</p>{item.power && <p className="detail-line"><b>力量</b>{item.power}</p>}<p className="detail-line"><b>动机</b>{item.motivation}</p>{item.relations && <p className="detail-line"><b>关系</b>{item.relations}</p>}</article>)}</div>
}

function Cast({data}: {data: CastPage}) {
  if (!data.items.length) return <Empty label="Core 尚未生成配角首次出场投影"/>
  return <div className="parity-list">{data.items.map(item => <article className="parity-card cast-row" key={item.name}><div className="avatar">{item.name.slice(0, 1)}</div><div><h3>{item.name}</h3><p>{item.briefRole || 'Core 未提供角色简介'}</p></div><div><span>首次出现</span><strong>第 {item.firstSeenChapter} 章</strong></div><div><span>最近出现</span><strong>第 {item.lastSeenChapter} 章</strong></div><div><span>出现次数</span><strong>{item.appearanceCount}</strong></div></article>)}</div>
}

function Empty({label}: {label: string}) { return <div className="parity-empty"><span>—</span><p>{label}</p><small>此处只呈现 Core 已保存的数据。</small></div> }
function statusLabel(value: string) { return ({planted: '已埋设', advanced: '已推进', resolved: '已回收'} as Record<string, string>)[value] ?? value }
