import { create } from "zustand"
import { projectService, folderService, requestItemService } from "@/services/projectService"
import type { model } from "../../wailsjs/go/models"
import { useTabStore } from "@/stores/tabStore"

const LAST_PROJECT_STORAGE_KEY = "minipost:last-project-id"

interface ProjectState {
  projects: model.Project[]
  currentProjectId: string | null
  folders: model.Folder[]
  requests: model.RequestItem[]
  loading: boolean
  error: string | null

  loadProjects: () => Promise<void>
  createProject: (name: string) => Promise<void>
  renameProject: (id: string, name: string) => Promise<void>
  deleteProject: (id: string) => Promise<void>
  selectProject: (id: string) => Promise<void>

  loadCollections: (projectId: string) => Promise<void>
  createFolder: (parentId: string, name: string) => Promise<void>
  renameFolder: (folderId: string, name: string) => Promise<void>
  deleteFolder: (folderId: string) => Promise<void>

  createRequest: (folderId: string, name: string) => Promise<model.RequestItem | null>
  saveRequestToBackend: (request: model.RequestItem) => Promise<void>
  deleteRequest: (requestId: string) => Promise<void>
}

function readLastProjectId(): string | null {
  if (typeof window === "undefined") {
    return null
  }

  try {
    return window.localStorage.getItem(LAST_PROJECT_STORAGE_KEY)
  } catch {
    return null
  }
}

function persistLastProjectId(projectId: string | null) {
  if (typeof window === "undefined") {
    return
  }

  try {
    if (projectId) {
      window.localStorage.setItem(LAST_PROJECT_STORAGE_KEY, projectId)
      return
    }
    window.localStorage.removeItem(LAST_PROJECT_STORAGE_KEY)
  } catch {
    // 本地持久化失败时不阻断主流程
  }
}

export const useProjectStore = create<ProjectState>((set, get) => ({
  projects: [],
  currentProjectId: null,
  folders: [],
  requests: [],
  loading: false,
  error: null,

  loadProjects: async () => {
    set({ loading: true, error: null })
    try {
      const projects = await projectService.listProjects()
      const nextProjects = projects ?? []
      const lastProjectId = readLastProjectId()
      const currentProjectId = get().currentProjectId
      const resolvedProjectId = currentProjectId && nextProjects.some((project) => project.id === currentProjectId)
        ? currentProjectId
        : lastProjectId && nextProjects.some((project) => project.id === lastProjectId)
          ? lastProjectId
          : nextProjects[0]?.id ?? null

      set({ projects: nextProjects, loading: false, currentProjectId: resolvedProjectId })

      if (resolvedProjectId) {
        useTabStore.getState().setCurrentProject(resolvedProjectId)
        persistLastProjectId(resolvedProjectId)
        await get().loadCollections(resolvedProjectId)
      } else {
        useTabStore.getState().setCurrentProject(null)
        persistLastProjectId(null)
        set({ folders: [], requests: [] })
      }
    } catch (err) {
      set({ error: String(err), loading: false })
    }
  },

  createProject: async (name) => {
    try {
      await projectService.createProject(name)
      await get().loadProjects()
    } catch (err) {
      set({ error: String(err) })
    }
  },

  renameProject: async (id, name) => {
    try {
      await projectService.renameProject(id, name)
      await get().loadProjects()
    } catch (err) {
      set({ error: String(err) })
    }
  },

  deleteProject: async (id) => {
    try {
      await projectService.deleteProject(id)
      const { currentProjectId } = get()
      if (currentProjectId === id) {
        useTabStore.getState().setCurrentProject(null)
        persistLastProjectId(null)
        set({ currentProjectId: null, folders: [], requests: [] })
      }
      useTabStore.getState().deleteProjectTabs(id)
      await get().loadProjects()
    } catch (err) {
      set({ error: String(err) })
    }
  },

  selectProject: async (id) => {
    useTabStore.getState().setCurrentProject(id)
    persistLastProjectId(id)
    set({ currentProjectId: id })
    await get().loadCollections(id)
  },

  loadCollections: async (projectId) => {
    try {
      const [folders, requests] = await Promise.all([
        folderService.listFolders(projectId),
        requestItemService.listRequests(projectId),
      ])
      set({
        folders: folders ?? [],
        requests: requests ?? [],
      })
    } catch (err) {
      set({ error: String(err) })
    }
  },

  createFolder: async (parentId, name) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await folderService.createFolder(currentProjectId, parentId, name)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },

  renameFolder: async (folderId, name) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await folderService.renameFolder(currentProjectId, folderId, name)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },

  deleteFolder: async (folderId) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await folderService.deleteFolder(currentProjectId, folderId)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },

  createRequest: async (folderId, name) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return null
    try {
      const req = await requestItemService.createRequest(currentProjectId, folderId, name)
      await get().loadCollections(currentProjectId)
      return req
    } catch (err) {
      set({ error: String(err) })
      return null
    }
  },

  saveRequestToBackend: async (request) => {
    try {
      await requestItemService.saveRequest(request)
      const { currentProjectId } = get()
      if (currentProjectId) {
        await get().loadCollections(currentProjectId)
      }
    } catch (err) {
      set({ error: String(err) })
    }
  },

  deleteRequest: async (requestId) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await requestItemService.deleteRequest(currentProjectId, requestId)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },
}))
