import { BookOpen, Layers, AlignLeft } from 'lucide-react'
import type { Overview as ProjectOverview } from '../types'

const phaseNames: Record<string,string> = {init:'初始化',premise:'故事构思',outline:'大纲规划',writing:'正文创作',complete:'已完结'}
export function Overview({data}: {readonly data: ProjectOverview}) {
  return <section className="overview">
    <p className="eyebrow">作品总览 / PROJECT OVERVIEW</p>
    <h1>{data.title}</h1><p className="synopsis">{data.synopsis || '作品简介尚未生成。'}</p>
    <div className="metrics">
      <article><BookOpen size={18}/><span>已完成章节</span><strong>{data.completedChapters}<small>章</small></strong></article>
      <article><Layers size={18}/><span>已规划章节</span><strong>{data.plannedChapters}<small>章</small></strong></article>
      <article><AlignLeft size={18}/><span>已记录总字数</span><strong>{data.wordCount.toLocaleString()}<small>字</small></strong></article>
    </div>
    <section className="panel"><h2>创作进度</h2><dl>
      <dt>项目阶段</dt><dd>{phaseNames[data.phase] ?? (data.phase || '未记录')}</dd>
      <dt>当前章节</dt><dd>{data.currentChapter > 0 ? `第 ${data.currentChapter} 章` : '尚未开始'}</dd>
      <dt>当前卷 / 弧</dt><dd>{data.currentVolume > 0 ? `卷 ${data.currentVolume} / 弧 ${data.currentArc}` : '未采用分层规划'}</dd>
    </dl></section>
    <div className="reading-tip"><BookOpen size={22}/><div><h2>从左侧选择一章，开始阅读</h2><p>显示项目中已保存的正文。尚未生成的章节会保留规划入口。</p></div></div>
  </section>
}
