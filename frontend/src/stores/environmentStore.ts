import { create } from "zustand"
import { environmentService } from "@/services/environmentService"
import type { model } from "../../wailsjs/go/models"

interface EnvironmentState {
  environments: model.Environment[]
  activeEnvironmentId: string | null
  loading: boolean

  loadEnvironments: (projectId: string) => Promise<void>
  createEnvironment: (projectId: string, name: string) => Promise<model.Environment | null>
  saveEnvironment: (env: model.Environment) => Promise<void>
  deleteEnvironment: (projectId: string, envId: string) => Promise<void>
  setActiveEnvironment: (envId: string | null) => void

  getActiveVariables: () => { key: string; value: string }[]
}

export const useEnvironmentStore = create<EnvironmentState>((set, get) => ({
  environments: [],
  activeEnvironmentId: null,
  loading: false,

  loadEnvironments: async (projectId) => {
    set({ loading: true })
    try {
      const envs = await environmentService.listEnvironments(projectId)
      set({ environments: envs ?? [], loading: false })
    } catch {
      set({ loading: false })
    }
  },

  createEnvironment: async (projectId, name) => {
    const created = await environmentService.createEnvironment(projectId, name)
    await get().loadEnvironments(projectId)
    if (created?.id) return created as model.Environment
    const envs = get().environments
    const found = envs.find((e) => e.name === name)
    return found ?? null
  },

  saveEnvironment: async (env) => {
    await environmentService.saveEnvironment(env)
    await get().loadEnvironments(env.projectId)
  },

  deleteEnvironment: async (projectId, envId) => {
    await environmentService.deleteEnvironment(projectId, envId)
    const { activeEnvironmentId } = get()
    if (activeEnvironmentId === envId) {
      set({ activeEnvironmentId: null })
    }
    await get().loadEnvironments(projectId)
  },

  setActiveEnvironment: (envId) => {
    set({ activeEnvironmentId: envId })
  },

  getActiveVariables: () => {
    const { environments, activeEnvironmentId } = get()
    if (!activeEnvironmentId) return []
    const env = environments.find((e) => e.id === activeEnvironmentId)
    if (!env) return []
    return (env.variables ?? [])
      .filter((v) => v.enabled && v.key)
      .map((v) => ({ key: v.key, value: v.value }))
  },
}))
