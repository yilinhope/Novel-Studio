import { renderToStaticMarkup } from 'react-dom/server'
import { expect, test } from 'vitest'
import { ProjectTree } from './components/ProjectTree'

test('长篇项目树只默认展开当前卷和弧', () => {
  const html = renderToStaticMarkup(<ProjectTree currentVolume={2} currentArc={1} nodes={[
    {id: 'volume-1', kind: 'volume', title: '第一卷', chapter: 0, children: [
      {id: 'arc-1-1', kind: 'arc', title: '第一弧', chapter: 0, children: [{id: 'chapter-1', kind: 'chapter', title: '开场', chapter: 1, children: []}]},
    ]},
    {id: 'volume-2', kind: 'volume', title: '第二卷', chapter: 0, children: [
      {id: 'arc-2-1', kind: 'arc', title: '当前弧', chapter: 0, children: [{id: 'chapter-2', kind: 'chapter', title: '转折', chapter: 2, children: []}]},
      {id: 'arc-2-2', kind: 'arc', title: '下一弧', chapter: 0, children: [{id: 'chapter-3', kind: 'chapter', title: '后续', chapter: 3, children: []}]},
    ]},
  ]} />)

  expect(html.match(/<details open="">/g)?.length).toBe(2)
  expect(html).toContain('第一卷')
  expect(html).toContain('当前弧')
})
