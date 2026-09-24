import { create } from 'zustand'
import { bridge } from './services'
import type { BudgetConfig, ConfigSnapshot, ModelSelection, ProviderDraft, RoleThinking, UsageSnapshot } from './types'

interface ConfigState { config: ConfigSnapshot | null; usage: UsageSnapshot | null; busy: boolean; error: string; load(): Promise<void>; saveProvider(draft: ProviderDraft): Promise<void>; switchModel(selection: ModelSelection): Promise<void>; setThinking(setting: RoleThinking): Promise<void>; saveBudget(budget: BudgetConfig): Promise<void>; loadUsage(): Promise<void> }
let request = 0
export const useConfigStore = create<ConfigState>((set, get) => ({
  config: null, usage: null, busy: false, error: '',
  async load() { const api = bridge(); if (!api.GetConfig) return; const ticket = ++request; set({busy: true, error: ''}); try { const config = await api.GetConfig(); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async saveProvider(draft) { const api = bridge(); if (!api.SaveProviderConfig) return; const ticket = ++request; set({busy: true, error: ''}); try { const config = await api.SaveProviderConfig(draft); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async switchModel(selection) { const api = bridge(); if (!api.SwitchModel) return; const ticket = ++request; set({busy: true, error: ''}); try { const config = await api.SwitchModel(selection); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async setThinking(setting) { const api = bridge(); if (!api.SetRoleThinking) return; const ticket = ++request; set({busy: true, error: ''}); try { const config = await api.SetRoleThinking(setting); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async saveBudget(budget) { const api = bridge(); if (!api.SaveBudgetConfig) return; const ticket = ++request; set({busy: true, error: ''}); try { const config = await api.SaveBudgetConfig(budget); if (ticket === request) set({config}) } catch (error) { if (ticket === request) set({error: String(error)}) } finally { if (ticket === request) set({busy: false}) } },
  async loadUsage() { const api = bridge(); if (!api.GetUsage) return; const ticket = ++request; try { const usage = await api.GetUsage(); if (ticket === request) set({usage}) } catch (error) { if (ticket === request) set({error: String(error)}) } },
}))
