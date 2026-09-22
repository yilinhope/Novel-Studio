import { ChevronRight, FileText, Folder } from 'lucide-react'
import type { TreeNode } from '../types'
import { useStudio } from '../store'

export function ProjectTree({nodes}: {readonly nodes: TreeNode[]}) {
  const {read, chapter, busy} = useStudio()
  return <ul className="tree">{nodes.map(node => <li key={node.id}>
    {node.kind === 'chapter'
      ? <button disabled={busy} aria-current={chapter?.number === node.chapter ? 'page' : undefined} onClick={() => void read(node.chapter)}><FileText size={14}/><span>{String(node.chapter).padStart(3,'0')} · {node.title}</span></button>
      : <details open><summary><ChevronRight size={13}/><Folder size={14}/><span>{node.title || (node.kind === 'volume' ? '未命名卷' : '未命名弧')}</span></summary>{node.children.length ? <ProjectTree nodes={node.children}/> : <small className="tree-empty">尚未展开规划</small>}</details>}
  </li>)}</ul>
}
