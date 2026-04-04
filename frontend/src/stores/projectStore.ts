import { create } from "zustand"
import { projectService, folderService, requestItemService } from "@/services/projectService"
import type { model } from "../../wailsjs/go/models"

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
      set({ projects: projects ?? [], loading: false })
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
        set({ currentProjectId: null, folders: [], requests: [] })
      }
      await get().loadProjects()
    } catch (err) {
      set({ error: String(err) })
    }
  },

  selectProject: async (id) => {
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
