import { create } from 'zustand'
import { bridge } from './services'
import type { BudgetConfig, ConfigModel, ConfigSnapshot, ModelSelection, ProviderDraft, RoleThinking, UsageSnapshot } from './types'

interface ConfigState { config: ConfigSnapshot | null; usage: UsageSnapshot | null; busy: boolean; error: string; load(): Promise<void>; saveProvider(draft: ProviderDraft): Promise<void>; discoverModels(draft: ProviderDraft): Promise<ConfigModel[]>; testModelConnection(draft: ProviderDraft, model: string): Promise<void>; switchModel(selection: ModelSelection): Promise<void>; setThinking(setting: RoleThinking): Promise<void>; saveBudget(budget: BudgetConfig): Promise<void>; loadUsage(): Promise<void> }
let request = 0
let usageRequest = 0
export const useConfigStore = create<ConfigState>((set, get) => ({
  config: null, usage: null, busy: false, error: '',
  async load() { const ticket = ++request; set({busy: true, error: ''}); try { const api = bridge(); if (!api.GetConfig) throw new Error('桌面桥接尚未提供 GetConfig'); const config = await api.GetConfig(); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async saveProvider(draft) { const ticket = ++request; set({busy: true, error: ''}); try { const api = bridge(); if (!api.SaveProviderConfig) throw new Error('桌面桥接尚未提供 SaveProviderConfig'); const config = await api.SaveProviderConfig(draft); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}); throw error } finally { if (ticket === request) set({busy: false}) } },
  async discoverModels(draft) { const api = bridge(); if (!api.DiscoverProviderModels) throw new Error('桌面桥接尚未提供 DiscoverProviderModels'); return api.DiscoverProviderModels(draft) },
  async testModelConnection(draft, model) { const api = bridge(); if (!api.TestModelConnection) throw new Error('桌面桥接尚未提供 TestModelConnection'); return api.TestModelConnection(draft, model) },
  async switchModel(selection) { const ticket = ++request; set({busy: true, error: ''}); try { const api = bridge(); if (!api.SwitchModel) throw new Error('桌面桥接尚未提供 SwitchModel'); const config = await api.SwitchModel(selection); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async setThinking(setting) { const ticket = ++request; set({busy: true, error: ''}); try { const api = bridge(); if (!api.SetRoleThinking) throw new Error('桌面桥接尚未提供 SetRoleThinking'); const config = await api.SetRoleThinking(setting); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async saveBudget(budget) { const ticket = ++request; set({busy: true, error: ''}); try { const api = bridge(); if (!api.SaveBudgetConfig) throw new Error('桌面桥接尚未提供 SaveBudgetConfig'); const config = await api.SaveBudgetConfig(budget); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async loadUsage() { const ticket = ++usageRequest; try { const api = bridge(); if (!api.GetUsage) throw new Error('桌面桥接尚未提供 GetUsage'); const usage = await api.GetUsage(); if (ticket === usageRequest) set({usage}) } catch (error) { if (ticket === usageRequest) set({error: String(error)}) } },
}))
