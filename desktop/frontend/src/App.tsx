import { useState } from 'react'
import { BookOpen, FolderOpen, LayoutDashboard, RefreshCw, Library, ShieldCheck } from 'lucide-react'
import { useStudio } from './store'
import { ProjectTree } from './components/ProjectTree'
import { Overview } from './components/Overview'
import { ChapterReader } from './components/ChapterReader'

export default function App() {
  const {project, chapter, busy, error, view, open, overview} = useStudio()
  const [path, setPath] = useState('')
  return <div className="studio">
    <header className="topbar"><button className="brand" onClick={overview} disabled={busy}><BookOpen size={20}/>Novel Studio <small>V1</small></button><span className="breadcrumb">{project?.overview.title ?? '你的长篇创作工作台'}</span><span className="badge">只读浏览</span><button onClick={() => void open()} disabled={busy}><FolderOpen size={15}/>{busy ? '正在打开…' : '打开项目'}</button></header>
    {error && <div className="error" role="alert">{error}</div>}
    {project ? <div className="workspace">
      <aside className="sidebar"><div className="sidebar-heading"><Library size={15}/>项目导航</div><button className={view === 'overview' ? 'nav active' : 'nav'} onClick={overview} disabled={busy}><LayoutDashboard size={16}/>作品总览</button><div className="sidebar-heading">长篇规划</div><ProjectTree nodes={project.tree}/>{!project.tree.length && <p className="muted">尚未生成章节大纲。</p>}</aside>
      <main><div className="tabbar"><span>{view === 'overview' ? '作品总览' : '章节正文'}</span><span className="muted">{view === 'chapter' && chapter ? `第 ${chapter.number} 章` : '本地项目'}</span></div><div className="canvas">{view === 'overview' ? <Overview data={project.overview}/> : <ChapterReader/>}</div></main>
      <aside className="inspector"><h2>项目与上下文</h2><section className="panel"><ShieldCheck size={18}/><h3>只读工作区</h3><p>可以浏览小说结构和正文，创作操作将在后续阶段接入。</p></section><section className="panel"><h3>当前项目</h3><p>{project.overview.title}</p><code className="path">{project.overview.path}</code><button disabled={busy} onClick={() => void open(project.overview.path)}><RefreshCw size={14}/>重新读取项目</button></section>{chapter && view === 'chapter' && <section className="panel"><h3>当前章节</h3><p>第 {chapter.number} 章</p><p>{chapter.hasContent ? '已读取保存的正文' : '尚无已提交正文'}</p><p>{chapter.wordCount.toLocaleString()} 字</p></section>}</aside>
    </div> : <main className="welcome"><BookOpen className="welcome-icon" size={46}/><p className="eyebrow">NOVEL STUDIO</p><h1>让故事，拥有自己的工作台。</h1><p>打开已有小说，沿着卷、弧与章节，回到你的故事。</p><button className="primary" disabled={busy} onClick={() => void open()}><FolderOpen size={18}/>打开本地项目</button><form onSubmit={e => {e.preventDefault(); void open(path)}}><label htmlFor="project-path">或输入小说项目目录</label><div><input id="project-path" value={path} onChange={e => setPath(e.target.value)} placeholder="例如 D:\\小说\\output\\novel"/><button disabled={busy || !path.trim()}>打开</button></div></form><small>选择含 meta/progress.json 的小说输出目录。</small></main>}
    <footer><span className="status-dot"/>只读浏览<span className="footer-right">{project ? `${project.overview.completedChapters} 章已完成 · ${project.overview.wordCount.toLocaleString()} 字` : '尚未打开项目'}</span><span>Novel Studio · M2</span></footer>
  </div>
}
