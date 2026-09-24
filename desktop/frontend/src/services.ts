import type { StudioBridge } from './types'
export function bridge(): StudioBridge {
  const api = window.go?.bridge.App
  if (!api) throw new Error('请从 Novel Studio 桌面程序打开项目。浏览器预览不连接本地小说。')
  return api
}

export function subscribeEngineEvents(callback: (payload: import('./types').StudioEngineEvent) => void): () => void {
  const eventsOn = window.runtime?.EventsOn
  if (!eventsOn) return () => {}
  return eventsOn('studio:engine-event', callback as (payload: unknown) => void)
}

export function subscribeStudioEvent<T>(eventName: string, callback: (payload: T) => void): () => void {
  const eventsOn = window.runtime?.EventsOn
  if (!eventsOn) return () => {}
  return eventsOn(eventName, callback as (payload: unknown) => void)
}
