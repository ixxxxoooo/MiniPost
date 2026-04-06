import { create } from "zustand"
import { environmentService } from "@/services/environmentService"
import type { model } from "../../wailsjs/go/models"

const ACTIVE_ENV_STORAGE_KEY = "minipost:active-environment-by-project"

type ActiveEnvironmentMap = Record<string, string | null>

function readActiveEnvironmentMap(): ActiveEnvironmentMap {
  if (typeof window === "undefined") return {}
  try {
    const raw = window.localStorage.getItem(ACTIVE_ENV_STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== "object") return {}
    const map: ActiveEnvironmentMap = {}
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      map[key] = typeof value === "string" ? value : null
    }
    return map
  } catch {
    return {}
  }
}

function persistActiveEnvironment(projectId: string, envId: string | null) {
  if (typeof window === "undefined") return
  try {
    const map = readActiveEnvironmentMap()
    map[projectId] = envId
    window.localStorage.setItem(ACTIVE_ENV_STORAGE_KEY, JSON.stringify(map))
  } catch {
    // ignore persistence errors
  }
}

interface EnvironmentState {
  environments: model.Environment[]
  activeEnvironmentId: string | null
  currentProjectId: string | null
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
  currentProjectId: null,
  loading: false,

  loadEnvironments: async (projectId) => {
    set({ loading: true })
    try {
      const envs = await environmentService.listEnvironments(projectId)
      const nextEnvironments = envs ?? []
      const activeMap = readActiveEnvironmentMap()
      const persistedActiveId = activeMap[projectId] ?? null
      const hasPersistedActive = persistedActiveId
        ? nextEnvironments.some((env) => env.id === persistedActiveId)
        : false
      const nextActiveEnvironmentId = hasPersistedActive ? persistedActiveId : null
      set({
        environments: nextEnvironments,
        activeEnvironmentId: nextActiveEnvironmentId,
        currentProjectId: projectId,
        loading: false,
      })
      if (persistedActiveId !== nextActiveEnvironmentId) {
        persistActiveEnvironment(projectId, nextActiveEnvironmentId)
      }
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
      persistActiveEnvironment(projectId, null)
    }
    await get().loadEnvironments(projectId)
  },

  setActiveEnvironment: (envId) => {
    const projectId = get().currentProjectId
    if (projectId) {
      persistActiveEnvironment(projectId, envId)
    }
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
