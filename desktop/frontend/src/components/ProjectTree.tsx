import { useState } from 'react'
import { ChevronRight, FileText, Folder } from 'lucide-react'
import type { TreeNode } from '../types'
import { useStudio } from '../store'

interface ProjectTreeProps {
  readonly nodes: TreeNode[]
  readonly currentVolume?: number
  readonly currentArc?: number
  readonly activeVolumeIndex?: number
}

export function ProjectTree({nodes, currentVolume = 0, currentArc = 0, activeVolumeIndex = -1}: ProjectTreeProps) {
  const {read, chapter, busy} = useStudio()
  const currentVolumeIndex = currentVolume > 0 ? currentVolume - 1 : 0
  return <ul className="tree">{nodes.map((node, index) => <li key={node.id}>
    {node.kind === 'chapter'
      ? <button disabled={busy} aria-current={chapter?.number === node.chapter ? 'page' : undefined} onClick={() => void read(node.chapter)}><FileText size={14}/><span>{String(node.chapter).padStart(3,'0')} · {node.title}</span></button>
      : <TreeBranch node={node} initiallyOpen={node.kind === 'volume' ? index === currentVolumeIndex : activeVolumeIndex === currentVolumeIndex && index === (currentArc > 0 ? currentArc - 1 : 0)} currentVolume={currentVolume} currentArc={currentArc} activeVolumeIndex={node.kind === 'volume' ? index : activeVolumeIndex}/>}
  </li>)}</ul>
}

function TreeBranch({node, initiallyOpen, currentVolume, currentArc, activeVolumeIndex}: {readonly node: TreeNode; readonly initiallyOpen: boolean; readonly currentVolume: number; readonly currentArc: number; readonly activeVolumeIndex: number}) {
  const [open, setOpen] = useState(initiallyOpen)
  return <details open={open} onToggle={event => setOpen(event.currentTarget.open)}><summary><ChevronRight size={13}/><Folder size={14}/><span>{node.title || (node.kind === 'volume' ? '未命名卷' : '未命名弧')}</span></summary>{node.children.length ? <ProjectTree nodes={node.children} currentVolume={currentVolume} currentArc={currentArc} activeVolumeIndex={activeVolumeIndex}/> : <small className="tree-empty">尚未展开规划</small>}</details>
}
