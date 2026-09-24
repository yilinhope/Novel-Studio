import { renderToStaticMarkup } from 'react-dom/server'
import { expect, test } from 'vitest'
import { CreateEventStatus } from './components/M6Centers'

test('CreateCenter 显示 completed CreateEvent 的项目视图刷新提示', () => {
  const html = renderToStaticMarkup(<CreateEventStatus event={{
    projectId: 'D:/book/output/novel',
    generation: 1,
    operation: 'create',
    state: 'completed',
    message: '项目已创建，但项目视图刷新失败：Overview 读取失败',
  }} />)

  expect(html).toContain('项目已创建，但项目视图刷新失败：Overview 读取失败')
  expect(html).toContain('role="status"')
})
