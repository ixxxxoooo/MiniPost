import { create } from "zustand"
import type { RequestData } from "@/types/request"
import type { HttpResponse } from "@/types/response"
import { createDefaultRequest } from "@/types/request"

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

interface TabState {
  tabs: RequestTab[]
  activeTabId: string | null

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
  getActiveTab: () => RequestTab | undefined
}

export const useTabStore = create<TabState>((set, get) => ({
  tabs: [],
  activeTabId: null,

  addTab: (tabData) => {
    const id = crypto.randomUUID()
    const tab: RequestTab = { ...tabData, id }
    set((s) => ({
      tabs: [...s.tabs, tab],
      activeTabId: id,
    }))
    return id
  },

  openRequestTab: (projectId, request) => {
    const { tabs } = get()
    const existing = tabs.find((t) => t.requestId === request.id)
    if (existing) {
      set({ activeTabId: existing.id })
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
      request,
      response: null,
      responseError: null,
    }
    set((s) => ({
      tabs: [...s.tabs, tab],
      activeTabId: id,
    }))
  },

  addNewUnsavedTab: (projectId) => {
    const id = crypto.randomUUID()
    const tab: RequestTab = {
      id,
      title: "Untitled",
      projectId,
      closable: true,
      dirty: false,
      request: createDefaultRequest(),
      response: null,
      responseError: null,
    }
    set((s) => ({
      tabs: [...s.tabs, tab],
      activeTabId: id,
    }))
  },

  removeTab: (tabId) => {
    set((s) => {
      const tabs = s.tabs.filter((t) => t.id !== tabId)
      let activeTabId = s.activeTabId
      if (activeTabId === tabId) {
        const idx = s.tabs.findIndex((t) => t.id === tabId)
        const nextTab = s.tabs[idx + 1] || s.tabs[idx - 1]
        activeTabId = nextTab?.id || null
      }
      return { tabs, activeTabId }
    })
  },

  setActiveTab: (tabId) => set({ activeTabId: tabId }),

  updateTab: (tabId, updates) => {
    set((s) => ({
      tabs: s.tabs.map((t) => (t.id === tabId ? { ...t, ...updates } : t)),
    }))
  },

  updateTabRequest: (tabId, req) => {
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === tabId
          ? { ...t, request: { ...t.request, ...req, updatedAt: new Date().toISOString() }, dirty: true }
          : t
      ),
    }))
  },

  setTabResponse: (tabId, res) => {
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === tabId ? { ...t, response: res, responseError: null } : t
      ),
    }))
  },

  setTabResponseError: (tabId, err) => {
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === tabId ? { ...t, responseError: err } : t
      ),
    }))
  },

  markTabDirty: (tabId, dirty) => {
    set((s) => ({
      tabs: s.tabs.map((t) => (t.id === tabId ? { ...t, dirty } : t)),
    }))
  },

  closeOtherTabs: (tabId) => {
    set((s) => ({
      tabs: s.tabs.filter((t) => t.id === tabId || !t.closable),
      activeTabId: tabId,
    }))
  },

  closeAllTabs: (projectId) => {
    set((s) => ({
      tabs: projectId
        ? s.tabs.filter((t) => t.projectId !== projectId || !t.closable)
        : s.tabs.filter((t) => !t.closable),
      activeTabId: null,
    }))
  },

  getActiveTab: () => {
    const { tabs, activeTabId } = get()
    return tabs.find((t) => t.id === activeTabId)
  },
}))
