import { useState } from 'react'
import { Activity, ArrowDownToLine, BookOpen, ClipboardCheck, FilePlus2, FolderInput, FolderOpen, GitPullRequest, LayoutDashboard, Library, Play, RefreshCw, Settings2, ShieldCheck } from 'lucide-react'
import { useStudio } from './store'
import { runtimeLabel, useEngineStore } from './engineStore'
import { ProjectTree } from './components/ProjectTree'
import { Overview } from './components/Overview'
import { ChapterReader } from './components/ChapterReader'
import { RuntimeCenter } from './components/RuntimeCenter'
import { ReviewCenter } from './components/ReviewCenter'
import { ProposalInbox } from './components/ProposalInbox'
import { CreateCenter, ExportCenter, ImportCenter, SettingsCenter } from './components/M6Centers'
import { useCreateProjectStore } from './createProjectStore'
import './review.css'

export default function App() {
  const {project, chapter, busy, error, view, dirty, open, overview, runtime, review, proposals, create, imports, settings, exports} = useStudio()
  const engine = useEngineStore(state => state.runtime)
  const [path, setPath] = useState('')
  const openCreate = (mode: 'quick' | 'outline' | 'cocreate') => { useCreateProjectStore.getState().setMode(mode); create() }
  const center = project
    ? view === 'overview' ? <Overview data={project.overview}/>
      : view === 'runtime' ? <RuntimeCenter/>
      : view === 'review' ? <ReviewCenter/>
      : view === 'proposals' ? <ProposalInbox/>
      : view === 'import' ? <ImportCenter/>
      : view === 'export' ? <ExportCenter/>
      : view === 'settings' ? <SettingsCenter/>
      : <ChapterReader/>
    : view === 'create' ? <CreateCenter/>
      : view === 'import' ? <ImportCenter/>
      : null
  return <div className="studio">
    <header className="topbar"><button className="brand" onClick={overview} disabled={busy}><BookOpen size={20}/>Novel Studio <small>V1</small></button><span className="breadcrumb">{project?.overview.title ?? '你的长篇创作工作台'}</span>{project && <button className={`badge runtime-badge runtime-state-${engine.state}`} onClick={runtime}><i/>{runtimeLabel(engine.state)}</button>}<button onClick={() => void open()} disabled={busy}><FolderOpen size={15}/>{busy ? '正在打开…' : '打开项目'}</button></header>
    {error && <div className="error" role="alert">{error}</div>}
    {project ? <div className="workspace">
      <aside className="sidebar"><div className="sidebar-heading"><Library size={15}/>项目导航</div><button className={view === 'overview' ? 'nav active' : 'nav'} onClick={overview} disabled={busy}><LayoutDashboard size={16}/>作品总览</button><button className={view === 'runtime' ? 'nav active' : 'nav'} onClick={runtime} disabled={busy}><Activity size={16}/>运行中心</button><button className={view === 'review' ? 'nav active' : 'nav'} onClick={review} disabled={busy}><ClipboardCheck size={16}/>审阅中心</button><button className={view === 'proposals' ? 'nav active' : 'nav'} onClick={proposals} disabled={busy}><GitPullRequest size={16}/>建议收件箱</button><button className={view === 'import' ? 'nav active' : 'nav'} onClick={imports} disabled={busy}><FolderInput size={16}/>导入</button><button className={view === 'export' ? 'nav active' : 'nav'} onClick={exports} disabled={busy}><ArrowDownToLine size={16}/>导出</button><button className={view === 'settings' ? 'nav active' : 'nav'} onClick={settings} disabled={busy}><Settings2 size={16}/>模型与预算</button><div className="sidebar-heading">长篇规划</div><ProjectTree key={project.outputDir} nodes={project.tree} currentVolume={project.overview.currentVolume} currentArc={project.overview.currentArc}/>{!project.tree.length && <p className="muted">尚未生成章节大纲。</p>}</aside>
      <main><div className="tabbar"><span>{view === 'overview' ? '作品总览' : view === 'runtime' ? '运行中心' : view === 'review' ? '审阅中心' : view === 'proposals' ? '建议收件箱' : view === 'import' ? '导入作品' : view === 'export' ? '导出作品' : view === 'settings' ? '模型与预算' : '章节编辑'}</span><span className="muted">{view === 'chapter' && chapter ? `第 ${chapter.number} 章${dirty ? ' · 未保存' : ''}` : '本地项目'}</span></div><div className="canvas">{center}</div></main>
      <aside className="inspector"><h2>项目与上下文</h2><section className="panel"><ShieldCheck size={18}/><h3>本地项目</h3><p>Runtime 状态和用量直接来自 ainovel Core。</p></section><section className="panel"><h3>当前项目</h3><p>{project.overview.title}</p><code className="path">{project.overview.path}</code><button disabled={busy} onClick={() => void open(project.overview.path)}><RefreshCw size={14}/>重新读取项目</button></section>{chapter && view === 'chapter' && <section className="panel"><h3>当前章节</h3><p>第 {chapter.number} 章</p><p>{chapter.hasContent ? '已读取保存的正文' : '尚无已提交正文'}</p><p>{chapter.wordCount.toLocaleString()} 字</p></section>}</aside>
    </div> : center ? <><header className="topbar"><button onClick={overview}><BookOpen size={18}/>返回</button></header><main className="canvas">{center}</main></> : <main className="welcome"><BookOpen className="welcome-icon" size={46}/><p className="eyebrow">NOVEL STUDIO</p><h1>让故事，拥有自己的工作台。</h1><p>沿用 ainovel Core，直接开始新书、导入作品或打开已有项目。</p><div style={{display: 'flex', flexWrap: 'wrap', justifyContent: 'center', gap: 8, marginTop: 20}}><button className="primary" disabled={busy} onClick={() => openCreate('quick')}><Play size={16}/>快速开始</button><button disabled={busy} onClick={() => openCreate('cocreate')}><Settings2 size={16}/>共创创建</button><button disabled={busy} onClick={() => openCreate('outline')}><FilePlus2 size={16}/>从大纲开始</button><button disabled={busy} onClick={imports}><FolderInput size={16}/>导入已有小说</button><button disabled={busy} onClick={() => void open()}><FolderOpen size={16}/>打开已有项目</button></div><form onSubmit={e => {e.preventDefault(); void open(path)}}><label htmlFor="project-path">或输入小说项目目录</label><div><input id="project-path" value={path} onChange={e => setPath(e.target.value)} placeholder="例如 D:\小说\output\novel"/><button disabled={busy || !path.trim()}>打开</button></div></form><small>选择含 meta/progress.json 的小说输出目录。</small></main>}
    <footer><span className={`status-dot runtime-state-${engine.state}`}/>{project ? runtimeLabel(engine.state) : '只读浏览'}<span className="footer-right">{project ? `${project.overview.completedChapters} 章已完成 · ${project.overview.wordCount.toLocaleString()} 字` : '尚未打开项目'}</span><span>Novel Studio · V1</span></footer>
  </div>
}
