import { create } from "zustand"
import { projectService, folderService, requestItemService, collectionService } from "@/services/projectService"
import type { main, model } from "../../wailsjs/go/models"
import { useTabStore } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { useEnvironmentStore } from "@/stores/environmentStore"

type CollectionNodeType = "folder" | "request"

const LAST_PROJECT_STORAGE_KEY = "minipost:last-project-id"

interface ProjectState {
  projects: model.Project[]
  currentProjectId: string | null
  folders: model.Folder[]
  requests: model.RequestItem[]
  treeNodes: model.CollectionNode[]
  loading: boolean
  error: string | null

  loadProjects: () => Promise<void>
  createProject: (name: string) => Promise<model.Project | null>
  renameProject: (id: string, name: string) => Promise<void>
  updateProjectDescription: (id: string, description: string) => Promise<void>
  updateProjectTheme: (id: string, color: string) => Promise<void>
  deleteProject: (id: string) => Promise<void>
  selectProject: (id: string) => Promise<void>

  loadCollections: (projectId: string) => Promise<void>
  createFolder: (parentId: string, name: string) => Promise<void>
  renameFolder: (folderId: string, name: string) => Promise<void>
  deleteFolder: (folderId: string) => Promise<void>
  moveFolder: (folderId: string, targetParentId: string, targetIndex: number) => Promise<void>

  createRequest: (folderId: string, name: string) => Promise<model.RequestItem | null>
  saveRequestToBackend: (request: model.RequestItem) => Promise<void>
  deleteRequest: (requestId: string) => Promise<void>
  moveRequest: (requestId: string, targetFolderId: string, targetIndex: number) => Promise<void>
  moveCollectionNode: (nodeId: string, nodeType: CollectionNodeType, targetParentFolderId: string, targetIndex: number) => Promise<void>
  renameRequest: (requestId: string, name: string) => Promise<void>
  duplicateRequest: (requestId: string) => Promise<void>
  duplicateFolder: (folderId: string) => Promise<void>
  exportProjectJSON: () => Promise<string | null>
  importFromFile: (format: string, content: string) => Promise<void>
  importFromURL: (format: string, sourceURL: string) => Promise<void>
  previewImportFromFile: (format: string, content: string) => Promise<main.ImportPreview>
  previewImportFromURL: (format: string, sourceURL: string) => Promise<main.ImportPreview>
  importWithStrategy: (format: string, content: string, sourceURL: string, strategy: string) => Promise<void>
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
  treeNodes: [],
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
      const shouldResetEnvironmentTabs = resolvedProjectId !== currentProjectId

      set({ projects: nextProjects, loading: false, currentProjectId: resolvedProjectId })

      if (resolvedProjectId) {
        useTabStore.getState().setCurrentProject(resolvedProjectId)
        persistLastProjectId(resolvedProjectId)
        if (shouldResetEnvironmentTabs) {
          useUIStore.getState().clearEnvironmentTabs()
        }
        await get().loadCollections(resolvedProjectId)
      } else {
        useTabStore.getState().setCurrentProject(null)
        persistLastProjectId(null)
        useUIStore.getState().clearEnvironmentTabs()
        set({ folders: [], requests: [], treeNodes: [] })
      }
    } catch (err) {
      set({ error: String(err), loading: false })
    }
  },

  createProject: async (name) => {
    try {
      const project = await projectService.createProject(name)
      await get().loadProjects()
      return project
    } catch (err) {
      set({ error: String(err) })
      return null
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

  updateProjectDescription: async (id, description) => {
    try {
      const updatedProject = await projectService.updateProjectDescription(id, description)
      set((state) => ({
        projects: state.projects.map((project) => (
          project.id === id ? { ...project, description: updatedProject?.description ?? description } : project
        )),
      }))
    } catch (err) {
      set({ error: String(err) })
    }
  },

  updateProjectTheme: async (id, color) => {
    try {
      const updatedProject = await projectService.updateProjectTheme(id, color)
      set((state) => ({
        projects: state.projects.map((project) => (
          project.id === id ? { ...project, themeColor: updatedProject?.themeColor ?? color } : project
        )),
      }))
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
        useUIStore.getState().clearEnvironmentTabs()
        set({ currentProjectId: null, folders: [], requests: [], treeNodes: [] })
      }
      useTabStore.getState().deleteProjectTabs(id)
      await get().loadProjects()
    } catch (err) {
      set({ error: String(err) })
    }
  },

  selectProject: async (id) => {
    const uiStore = useUIStore.getState()
    uiStore.clearEnvironmentTabs()
    uiStore.setWorkspaceView("project")
    useTabStore.getState().setCurrentProject(id)
    persistLastProjectId(id)
    set({ currentProjectId: id })
    await get().loadCollections(id)
  },

  loadCollections: async (projectId) => {
    try {
      const data = await collectionService.getCollectionData(projectId)
      set({
        folders: data?.folders ?? [],
        requests: data?.requests ?? [],
        treeNodes: data?.treeNodes ?? [],
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

  moveFolder: async (folderId, targetParentId, targetIndex) => {
    await get().moveCollectionNode(folderId, "folder", targetParentId, targetIndex)
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
      // 删除请求前先关闭该请求对应的所有 tab，removeTab 会自动将焦点切到相邻 tab
      const tabState = useTabStore.getState()
      const projectTabState = tabState.projectTabs[currentProjectId]
      if (projectTabState?.tabs?.length) {
        const tabsToClose = projectTabState.tabs
          .filter((tab) => tab.requestId === requestId)
          .map((tab) => tab.id)

        tabsToClose.forEach((tabId) => {
          tabState.removeTab(tabId)
        })
      }

      await requestItemService.deleteRequest(currentProjectId, requestId)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },

  moveRequest: async (requestId, targetFolderId, targetIndex) => {
    await get().moveCollectionNode(requestId, "request", targetFolderId, targetIndex)
  },

  moveCollectionNode: async (nodeId, nodeType: CollectionNodeType, targetParentFolderId, targetIndex) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await collectionService.moveCollectionNode(currentProjectId, nodeId, nodeType, targetParentFolderId, targetIndex)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },

  renameRequest: async (requestId, name) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await requestItemService.renameRequest(currentProjectId, requestId, name)
      await get().loadCollections(currentProjectId)
      // 同步更新所有关联的 Tab 标题
      const tabState = useTabStore.getState()
      const projectTabs = tabState.projectTabs[currentProjectId]
      if (projectTabs) {
        projectTabs.tabs.forEach((tab) => {
          if (tab.requestId === requestId && tab.title !== name) {
            tabState.updateTab(tab.id, { title: name })
          }
        })
      }
    } catch (err) {
      set({ error: String(err) })
    }
  },

  duplicateRequest: async (requestId) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await requestItemService.duplicateRequest(currentProjectId, requestId)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },

  duplicateFolder: async (folderId) => {
    const { currentProjectId } = get()
    if (!currentProjectId) return
    try {
      await folderService.duplicateFolder(currentProjectId, folderId)
      await get().loadCollections(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
    }
  },

  exportProjectJSON: async () => {
    const { currentProjectId } = get()
    if (!currentProjectId) return null
    try {
      return await projectService.exportProjectJSON(currentProjectId)
    } catch (err) {
      set({ error: String(err) })
      return null
    }
  },

  importFromFile: async (format, content) => {
    const { currentProjectId } = get()
    if (!currentProjectId) {
      throw new Error("请先选择项目后再导入")
    }
    try {
      const envStore = useEnvironmentStore.getState()
      const beforeEnvIds = new Set(
        envStore.currentProjectId === currentProjectId
          ? envStore.environments.map((env) => env.id)
          : []
      )
      await collectionService.importFromFile(currentProjectId, format, content)
      await get().loadCollections(currentProjectId)
      await envStore.loadEnvironments(currentProjectId)
      const importedEnv = useEnvironmentStore.getState().environments.find((env) => !beforeEnvIds.has(env.id))
      if (importedEnv?.id) {
        useEnvironmentStore.getState().setActiveEnvironment(importedEnv.id)
      }
    } catch (err) {
      set({ error: String(err) })
      throw err
    }
  },

  importFromURL: async (format, sourceURL) => {
    const { currentProjectId } = get()
    if (!currentProjectId) {
      throw new Error("请先选择项目后再导入")
    }
    try {
      const envStore = useEnvironmentStore.getState()
      const beforeEnvIds = new Set(
        envStore.currentProjectId === currentProjectId
          ? envStore.environments.map((env) => env.id)
          : []
      )
      await collectionService.importFromURL(currentProjectId, format, sourceURL)
      await get().loadCollections(currentProjectId)
      await envStore.loadEnvironments(currentProjectId)
      const importedEnv = useEnvironmentStore.getState().environments.find((env) => !beforeEnvIds.has(env.id))
      if (importedEnv?.id) {
        useEnvironmentStore.getState().setActiveEnvironment(importedEnv.id)
      }
    } catch (err) {
      set({ error: String(err) })
      throw err
    }
  },

  previewImportFromFile: async (format, content) => {
    const { currentProjectId } = get()
    if (!currentProjectId) {
      throw new Error("请先选择项目后再导入")
    }
    return collectionService.previewImportFromFile(currentProjectId, format, content)
  },

  previewImportFromURL: async (format, sourceURL) => {
    const { currentProjectId } = get()
    if (!currentProjectId) {
      throw new Error("请先选择项目后再导入")
    }
    return collectionService.previewImportFromURL(currentProjectId, format, sourceURL)
  },

  importWithStrategy: async (format, content, sourceURL, strategy) => {
    const { currentProjectId } = get()
    if (!currentProjectId) {
      throw new Error("请先选择项目后再导入")
    }
    try {
      const envStore = useEnvironmentStore.getState()
      const beforeEnvIds = new Set(
        envStore.currentProjectId === currentProjectId
          ? envStore.environments.map((env) => env.id)
          : []
      )
      await collectionService.importWithStrategy(currentProjectId, format, content, sourceURL, strategy)
      await get().loadCollections(currentProjectId)
      await envStore.loadEnvironments(currentProjectId)
      const importedEnv = useEnvironmentStore.getState().environments.find((env) => !beforeEnvIds.has(env.id))
      if (importedEnv?.id) {
        useEnvironmentStore.getState().setActiveEnvironment(importedEnv.id)
      }
    } catch (err) {
      set({ error: String(err) })
      throw err
    }
  },
}))
