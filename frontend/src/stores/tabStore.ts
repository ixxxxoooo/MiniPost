import { create } from "zustand"
import type { RequestData } from "@/types/request"
import type { HttpResponse } from "@/types/response"
import { createDefaultRequest } from "@/types/request"

const TAB_STORAGE_KEY = "minipost:project-tabs"

export interface RequestTab {
  id: string
  title: string
  projectId: string
  requestId?: string
  closable: boolean
  dirty: boolean
  request: RequestData
  response: HttpResponse | null
  responseError: string | null
}

interface ProjectTabState {
  tabs: RequestTab[]
  activeTabId: string | null
}

interface TabState {
  projectTabs: Record<string, ProjectTabState>
  currentProjectId: string | null

  setCurrentProject: (projectId: string | null) => void
  addTab: (tab: Omit<RequestTab, "id">) => string
  openRequestTab: (projectId: string, request: RequestData) => void
  addNewUnsavedTab: (projectId: string) => void
  removeTab: (tabId: string) => void
  setActiveTab: (tabId: string) => void
  updateTab: (tabId: string, updates: Partial<RequestTab>) => void
  updateTabRequest: (tabId: string, req: Partial<RequestData>) => void
  setTabResponse: (tabId: string, res: HttpResponse | null) => void
  setTabResponseError: (tabId: string, err: string | null) => void
  markTabDirty: (tabId: string, dirty: boolean) => void
  closeOtherTabs: (tabId: string) => void
  closeAllTabs: (projectId?: string) => void
  deleteProjectTabs: (projectId: string) => void
}

// 稳定的空状态单例，避免 zustand selector 每次返回新引用导致无限 re-render
const EMPTY_PROJECT_TAB_STATE: ProjectTabState = Object.freeze({
  tabs: [] as RequestTab[],
  activeTabId: null,
})

function createEmptyProjectTabState(): ProjectTabState {
  return {
    tabs: [],
    activeTabId: null,
  }
}

function resolveProjectId(state: Pick<TabState, "currentProjectId">, projectId?: string | null): string | null {
  return projectId ?? state.currentProjectId
}

function getScopedProjectState(state: TabState, projectId?: string | null): ProjectTabState {
  const targetProjectId = resolveProjectId(state, projectId)
  if (!targetProjectId) {
    return EMPTY_PROJECT_TAB_STATE
  }
  return state.projectTabs[targetProjectId] ?? EMPTY_PROJECT_TAB_STATE
}

function isValidRequestData(value: unknown): value is RequestData {
  if (!value || typeof value !== "object") {
    return false
  }

  const request = value as Partial<RequestData>
  return typeof request.id === "string"
    && typeof request.name === "string"
    && typeof request.method === "string"
    && typeof request.url === "string"
    && Array.isArray(request.params)
    && Array.isArray(request.headers)
    && typeof request.body === "object"
    && request.body !== null
    && typeof request.auth === "object"
    && request.auth !== null
    && typeof request.createdAt === "string"
    && typeof request.updatedAt === "string"
}

function sanitizeTab(rawTab: unknown, projectId: string): RequestTab | null {
  if (!rawTab || typeof rawTab !== "object") {
    return null
  }

  const tab = rawTab as Partial<RequestTab>
  if (typeof tab.id !== "string" || typeof tab.title !== "string") {
    return null
  }

  if (!isValidRequestData(tab.request)) {
    return null
  }

  const normalizedProjectId = typeof tab.projectId === "string" && tab.projectId.length > 0 ? tab.projectId : projectId
  const normalizedRequest = {
    ...tab.request,
    projectId: tab.request.projectId ?? normalizedProjectId,
  }

  return {
    id: tab.id,
    title: tab.title,
    projectId: normalizedProjectId,
    requestId: typeof tab.requestId === "string" ? tab.requestId : undefined,
    closable: tab.closable ?? true,
    dirty: Boolean(tab.dirty),
    request: normalizedRequest,
    response: tab.response ?? null,
    responseError: typeof tab.responseError === "string" ? tab.responseError : null,
  }
}

function sanitizeProjectTabs(rawProjectTabs: unknown): Record<string, ProjectTabState> {
  if (!rawProjectTabs || typeof rawProjectTabs !== "object") {
    return {}
  }

  const sanitizedProjectTabs: Record<string, ProjectTabState> = {}

  for (const [projectId, rawProjectState] of Object.entries(rawProjectTabs as Record<string, unknown>)) {
    if (!projectId || !rawProjectState || typeof rawProjectState !== "object") {
      continue
    }

    const candidateState = rawProjectState as Partial<ProjectTabState>
    const tabs = Array.isArray(candidateState.tabs)
      ? candidateState.tabs
          .map((tab) => sanitizeTab(tab, projectId))
          .filter((tab): tab is RequestTab => tab !== null)
      : []

    const activeTabId = typeof candidateState.activeTabId === "string"
      && tabs.some((tab) => tab.id === candidateState.activeTabId)
      ? candidateState.activeTabId
      : tabs[0]?.id ?? null

    sanitizedProjectTabs[projectId] = {
      tabs,
      activeTabId,
    }
  }

  return sanitizedProjectTabs
}

function readPersistedProjectTabs(): Record<string, ProjectTabState> {
  if (typeof window === "undefined") {
    return {}
  }

  try {
    const raw = window.localStorage.getItem(TAB_STORAGE_KEY)
    if (!raw) {
      return {}
    }

    const parsed = JSON.parse(raw) as unknown
    const sanitized = sanitizeProjectTabs(parsed)
    window.localStorage.setItem(TAB_STORAGE_KEY, JSON.stringify(sanitized))
    return sanitized
  } catch {
    return {}
  }
}

let persistTimer: ReturnType<typeof setTimeout> | null = null

function persistProjectTabs(projectTabs: Record<string, ProjectTabState>) {
  if (typeof window === "undefined") {
    return
  }

  // 防抖：避免高频 state 更新导致 localStorage 写入风暴
  if (persistTimer) {
    clearTimeout(persistTimer)
  }
  persistTimer = setTimeout(() => {
    try {
      const sanitized = sanitizeProjectTabs(projectTabs)
      window.localStorage.setItem(TAB_STORAGE_KEY, JSON.stringify(sanitized))
    } catch {
      // 本地持久化失败时不阻断主流程
    }
  }, 100)
}

function updateProjectState(
  state: TabState,
  projectId: string,
  updater: (projectState: ProjectTabState) => ProjectTabState,
): Pick<TabState, "projectTabs"> {
  const currentProjectState = state.projectTabs[projectId] ?? createEmptyProjectTabState()
  const nextProjectState = updater(currentProjectState)
  const normalizedProjectState: ProjectTabState = {
    tabs: nextProjectState.tabs,
    activeTabId: nextProjectState.activeTabId && nextProjectState.tabs.some((tab) => tab.id === nextProjectState.activeTabId)
      ? nextProjectState.activeTabId
      : nextProjectState.tabs[0]?.id ?? null,
  }
  const projectTabs = {
    ...state.projectTabs,
    [projectId]: normalizedProjectState,
  }
  persistProjectTabs(projectTabs)
  return { projectTabs }
}

type ComparableKeyValue = {
  id: string
  key: string
  value: string
  enabled: boolean
  description: string
}

type ComparableFormData = ComparableKeyValue & {
  type: "text" | "file"
  filePath: string
  fileName: string
}

function isEmptyKeyValueRow(item: { key?: string; value?: string; description?: string }): boolean {
  return (item.key ?? "") === ""
    && (item.value ?? "") === ""
    && (item.description ?? "") === ""
}

function toComparableKeyValues(items: RequestData["params"] | RequestData["headers"] | NonNullable<RequestData["body"]["formUrlEncoded"]>): ComparableKeyValue[] {
  return (items ?? [])
    .filter((item) => !isEmptyKeyValueRow(item))
    .map((item) => ({
      id: item.id,
      key: item.key,
      value: item.value,
      enabled: item.enabled,
      description: item.description ?? "",
    }))
}

function toComparableFormData(items: NonNullable<RequestData["body"]["formData"]>): ComparableFormData[] {
  return (items ?? [])
    .filter((item) =>
      (item.key ?? "") !== ""
      || (item.value ?? "") !== ""
      || (item.filePath ?? "") !== ""
      || (item.fileName ?? "") !== ""
    )
    .map((item) => ({
      id: item.id,
      key: item.key,
      value: item.value,
      enabled: item.enabled,
      description: item.description ?? "",
      type: item.type,
      filePath: item.filePath ?? "",
      fileName: item.fileName ?? "",
    }))
}

function toComparableRequest(request: RequestData) {
  return {
    id: request.id,
    name: request.name,
    method: request.method,
    url: request.url,
    params: toComparableKeyValues(request.params ?? []),
    headers: toComparableKeyValues(request.headers ?? []),
    body: {
      type: request.body.type,
      raw: request.body.raw ?? "",
      json: request.body.json ?? "",
      formUrlEncoded: toComparableKeyValues(request.body.formUrlEncoded ?? []),
      formData: toComparableFormData(request.body.formData ?? []),
    },
    auth: {
      type: request.auth.type,
      basic: request.auth.basic
        ? {
            username: request.auth.basic.username,
            password: request.auth.basic.password,
          }
        : undefined,
      bearer: request.auth.bearer
        ? {
            token: request.auth.bearer.token,
          }
        : undefined,
      apiKey: request.auth.apiKey
        ? {
            key: request.auth.apiKey.key,
            value: request.auth.apiKey.value,
            addTo: request.auth.apiKey.addTo,
          }
        : undefined,
    },
    folderId: request.folderId ?? "",
    projectId: request.projectId ?? "",
    createdAt: request.createdAt,
  }
}

function hasMeaningfulRequestChange(previous: RequestData, next: RequestData): boolean {
  return JSON.stringify(toComparableRequest(previous)) !== JSON.stringify(toComparableRequest(next))
}

export function getProjectTabsFromState(state: TabState, projectId?: string | null): RequestTab[] {
  return getScopedProjectState(state, projectId).tabs
}

export function getProjectActiveTabIdFromState(state: TabState, projectId?: string | null): string | null {
  return getScopedProjectState(state, projectId).activeTabId
}

export function getProjectActiveTabFromState(state: TabState, projectId?: string | null): RequestTab | undefined {
  const projectState = getScopedProjectState(state, projectId)
  return projectState.tabs.find((tab) => tab.id === projectState.activeTabId)
}

export const useTabStore = create<TabState>((set, get) => ({
  projectTabs: readPersistedProjectTabs(),
  currentProjectId: null,

  setCurrentProject: (projectId) => {
    set((state) => {
      if (!projectId) {
        return { currentProjectId: null }
      }

      if (state.projectTabs[projectId]) {
        return { currentProjectId: projectId }
      }

      const projectTabs = {
        ...state.projectTabs,
        [projectId]: createEmptyProjectTabState(),
      }
      persistProjectTabs(projectTabs)

      return {
        currentProjectId: projectId,
        projectTabs,
      }
    })
  },

  addTab: (tabData) => {
    const id = crypto.randomUUID()
    const tab: RequestTab = { ...tabData, id }
    const projectId = tab.projectId

    set((state) => updateProjectState(state, projectId, (projectState) => ({
      tabs: [...projectState.tabs, tab],
      activeTabId: id,
    })))

    return id
  },

  openRequestTab: (projectId, request) => {
    const projectState = getScopedProjectState(get(), projectId)
    const existing = projectState.tabs.find((tab) => tab.requestId === request.id)

    if (existing) {
      set((state) => updateProjectState(state, projectId, (currentProjectState) => ({
        ...currentProjectState,
        activeTabId: existing.id,
      })))
      return
    }

    const id = crypto.randomUUID()
    const tab: RequestTab = {
      id,
      title: request.name || "Untitled",
      projectId,
      requestId: request.id,
      closable: true,
      dirty: false,
      request: {
        ...request,
        projectId: request.projectId ?? projectId,
      },
      response: null,
      responseError: null,
    }

    set((state) => updateProjectState(state, projectId, (projectStateForTarget) => ({
      tabs: [...projectStateForTarget.tabs, tab],
      activeTabId: id,
    })))
  },

  addNewUnsavedTab: (projectId) => {
    const id = crypto.randomUUID()
    const tab: RequestTab = {
      id,
      title: "Untitled",
      projectId,
      closable: true,
      dirty: false,
      request: createDefaultRequest({ projectId }),
      response: null,
      responseError: null,
    }

    set((state) => updateProjectState(state, projectId, (projectState) => ({
      tabs: [...projectState.tabs, tab],
      activeTabId: id,
    })))
  },

  removeTab: (tabId) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => {
      const tabs = projectState.tabs.filter((tab) => tab.id !== tabId)
      let activeTabId = projectState.activeTabId

      if (activeTabId === tabId) {
        const currentIndex = projectState.tabs.findIndex((tab) => tab.id === tabId)
        const nextTab = projectState.tabs[currentIndex + 1] || projectState.tabs[currentIndex - 1]
        activeTabId = nextTab?.id || null
      }

      return {
        tabs,
        activeTabId,
      }
    }))
  },

  setActiveTab: (tabId) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => ({
      ...projectState,
      activeTabId: tabId,
    })))
  },

  updateTab: (tabId, updates) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => ({
      ...projectState,
      tabs: projectState.tabs.map((tab) => (tab.id === tabId ? { ...tab, ...updates } : tab)),
    })))
  },

  updateTabRequest: (tabId, req) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => ({
      ...projectState,
      tabs: projectState.tabs.map((tab) =>
        tab.id === tabId
          ? (() => {
              const nextRequest: RequestData = {
                ...tab.request,
                ...req,
                projectId: (req.projectId ?? tab.request.projectId ?? currentProjectId),
                updatedAt: new Date().toISOString(),
              }
              const meaningfulChanged = hasMeaningfulRequestChange(tab.request, nextRequest)
              return {
                ...tab,
                request: nextRequest,
                dirty: tab.dirty || meaningfulChanged,
              }
            })()
          : tab
      ),
    })))
  },

  setTabResponse: (tabId, res) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => ({
      ...projectState,
      tabs: projectState.tabs.map((tab) =>
        tab.id === tabId ? { ...tab, response: res, responseError: null } : tab,
      ),
    })))
  },

  setTabResponseError: (tabId, err) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => ({
      ...projectState,
      tabs: projectState.tabs.map((tab) =>
        tab.id === tabId ? { ...tab, responseError: err } : tab,
      ),
    })))
  },

  markTabDirty: (tabId, dirty) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => ({
      ...projectState,
      tabs: projectState.tabs.map((tab) => (tab.id === tabId ? { ...tab, dirty } : tab)),
    })))
  },

  closeOtherTabs: (tabId) => {
    const currentProjectId = get().currentProjectId
    if (!currentProjectId) return

    set((state) => updateProjectState(state, currentProjectId, (projectState) => ({
      tabs: projectState.tabs.filter((tab) => tab.id === tabId || !tab.closable),
      activeTabId: tabId,
    })))
  },

  closeAllTabs: (projectId) => {
    const targetProjectId = resolveProjectId(get(), projectId)
    if (!targetProjectId) return

    set((state) => updateProjectState(state, targetProjectId, (projectState) => ({
      tabs: projectState.tabs.filter((tab) => !tab.closable),
      activeTabId: null,
    })))
  },

  deleteProjectTabs: (projectId) => {
    set((state) => {
      const projectTabs = { ...state.projectTabs }
      delete projectTabs[projectId]
      persistProjectTabs(projectTabs)
      return { projectTabs }
    })
  },
}))
