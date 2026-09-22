import type { StudioBridge } from './types'
export function bridge(): StudioBridge {
  const api = window.go?.bridge.App
  if (!api) throw new Error('请从 Novel Studio 桌面程序打开项目。浏览器预览不连接本地小说。')
  return api
}
