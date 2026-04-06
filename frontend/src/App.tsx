import { useCallback, useEffect, useMemo } from "react"
import { TooltipProvider } from "@/components/ui/tooltip"
import { AppLayout } from "@/components/layout/AppLayout"
import { useKeyboardShortcuts } from "@/hooks/useKeyboardShortcuts"
import type { HttpMethod } from "@/lib/constants"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { info, error as logError } from "@/lib/logger"
import { backupService } from "@/services/backupService"
import { applyProjectThemeColor } from "@/lib/projectTheme"

const WINDOW_STATE_STORAGE_KEY = "minipost:window-state"
const AUTO_BACKUP_LAST_RUN_STORAGE_KEY = "minipost:auto-backup-last-run-at"
const WINDOW_MIN_WIDTH = 900
const WINDOW_MIN_HEIGHT = 600

type PersistedWindowState = {
  x: number
  y: number
  width: number
  height: number
}

function normalizeWindowState(input: Partial<PersistedWindowState>): PersistedWindowState | null {
  const x = Number(input.x)
  const y = Number(input.y)
  const width = Number(input.width)
  const height = Number(input.height)
  if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(width) || !Number.isFinite(height)) {
    return null
  }
  return {
    x: Math.round(x),
    y: Math.round(y),
    width: Math.max(WINDOW_MIN_WIDTH, Math.round(width)),
    height: Math.max(WINDOW_MIN_HEIGHT, Math.round(height)),
  }
}

function readWindowState(): PersistedWindowState | null {
  if (typeof window === "undefined") return null
  try {
    const raw = window.localStorage.getItem(WINDOW_STATE_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<PersistedWindowState>
    return normalizeWindowState(parsed)
  } catch {
    return null
  }
}

function persistWindowState(state: PersistedWindowState) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(WINDOW_STATE_STORAGE_KEY, JSON.stringify(state))
  } catch {
    // ignore persistence errors
  }
}

function readLastAutoBackupAt(): number | null {
  if (typeof window === "undefined") return null
  try {
    const raw = window.localStorage.getItem(AUTO_BACKUP_LAST_RUN_STORAGE_KEY)
    if (!raw) return null
    const parsed = Number(raw)
    if (!Number.isFinite(parsed) || parsed <= 0) return null
    return parsed
  } catch {
    return null
  }
}

function persistLastAutoBackupAt(timestamp: number) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(AUTO_BACKUP_LAST_RUN_STORAGE_KEY, String(timestamp))
  } catch {
    // ignore persistence errors
  }
}

function App() {
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)
  const resolvedTheme = useUIStore((s) => s.resolved)
  const setWorkspaceView = useUIStore((s) => s.setWorkspaceView)
  const editingEnvironmentId = useUIStore((s) => s.editingEnvironmentId)
  const closeActiveEnvironmentTab = useUIStore((s) => s.closeActiveEnvironmentTab)
  const autoBackupEnabled = useUIStore((s) => s.autoBackupEnabled)
  const autoBackupIntervalMinutes = useUIStore((s) => s.autoBackupIntervalMinutes)
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const projects = useProjectStore((s) => s.projects)
  const createRequest = useProjectStore((s) => s.createRequest)
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const openRequestTab = useTabStore((s) => s.openRequestTab)
  const currentProjectThemeColor = useMemo(() => {
    if (!currentProjectId) return null
    return projects.find((project) => project.id === currentProjectId)?.themeColor ?? null
  }, [currentProjectId, projects])

  useEffect(() => {
    info("App", "React App 组件已挂载, 开始加载项目列表")
    useProjectStore.getState().loadProjects()
      .then(() => info("App", "项目列表加载完成"))
      .catch((err) => logError("App", "项目列表加载失败", { error: String(err) }))
  }, [])

  useEffect(() => {
    applyProjectThemeColor(currentProjectThemeColor, resolvedTheme)
  }, [currentProjectThemeColor, resolvedTheme])

  useEffect(() => {
    if (!autoBackupEnabled) return

    let disposed = false
    let running = false
    let startupTimer: ReturnType<typeof setTimeout> | null = null
    let intervalTimer: ReturnType<typeof setInterval> | null = null
    const intervalMs = Math.max(5, Math.round(autoBackupIntervalMinutes)) * 60 * 1000

    const runAutoBackup = async () => {
      if (disposed || running) return
      running = true
      try {
        const path = await backupService.createBackup()
        persistLastAutoBackupAt(Date.now())
        info("Backup", "自动备份成功", { path })
      } catch (err) {
        logError("Backup", "自动备份失败", { error: String(err) })
      } finally {
        running = false
      }
    }

    const lastBackupAt = readLastAutoBackupAt()
    const elapsed = lastBackupAt ? Date.now() - lastBackupAt : Number.POSITIVE_INFINITY
    const initialDelay = elapsed >= intervalMs ? 3000 : Math.max(1000, intervalMs - elapsed)

    startupTimer = setTimeout(() => {
      void runAutoBackup()
      intervalTimer = setInterval(() => {
        void runAutoBackup()
      }, intervalMs)
    }, initialDelay)

    return () => {
      disposed = true
      if (startupTimer) clearTimeout(startupTimer)
      if (intervalTimer) clearInterval(intervalTimer)
    }
  }, [autoBackupEnabled, autoBackupIntervalMinutes])

  useEffect(() => {
    let canceled = false
    const restoreWindowState = async () => {
      const stored = readWindowState()
      if (!stored) return
      try {
        const runtime = await import("../wailsjs/runtime/runtime")
        if (canceled) return
        runtime.WindowSetSize(stored.width, stored.height)
        runtime.WindowSetPosition(stored.x, stored.y)
      } catch {
        // ignore restore failures
      }
    }
    void restoreWindowState()
    return () => {
      canceled = true
    }
  }, [])

  useEffect(() => {
    let disposed = false
    let pollTimer: ReturnType<typeof setInterval> | null = null
    let saving = false
    let removeBeforeUnload: (() => void) | null = null

    const startWindowStatePersistence = async () => {
      try {
        const runtime = await import("../wailsjs/runtime/runtime")
        if (disposed) return

        const saveWindowState = async () => {
          if (disposed || saving) return
          saving = true
          try {
            const maximised = await runtime.WindowIsMaximised()
            if (maximised) return
            const [size, position] = await Promise.all([runtime.WindowGetSize(), runtime.WindowGetPosition()])
            const normalized = normalizeWindowState({
              x: position.x,
              y: position.y,
              width: size.w,
              height: size.h,
            })
            if (normalized) {
              persistWindowState(normalized)
            }
          } catch {
            // ignore persistence failures
          } finally {
            saving = false
          }
        }

        const onBeforeUnload = () => {
          void saveWindowState()
        }
        window.addEventListener("beforeunload", onBeforeUnload)
        removeBeforeUnload = () => window.removeEventListener("beforeunload", onBeforeUnload)

        pollTimer = setInterval(() => {
          void saveWindowState()
        }, 1200)
      } catch {
        // ignore setup failures
      }
    }

    void startWindowStatePersistence()

    return () => {
      disposed = true
      if (pollTimer) clearInterval(pollTimer)
      if (removeBeforeUnload) removeBeforeUnload()
    }
  }, [])

  const handleNewRequest = useCallback(async () => {
    if (!currentProjectId) return
    const req = await createRequest("", "New Request")
    if (req) {
      openRequestTab(currentProjectId, {
        id: req.id,
        name: req.name,
        method: req.method as HttpMethod,
        url: req.url,
        params: (req.params ?? []).map((p) => ({
          id: crypto.randomUUID(),
          key: p.key,
          value: p.value,
          enabled: true,
        })),
        headers: (req.headers ?? []).map((h) => ({
          id: crypto.randomUUID(),
          key: h.key,
          value: h.value,
          enabled: true,
        })),
        body: req.body
          ? {
              type: req.body.type as "none" | "raw" | "json" | "form-urlencoded" | "form-data",
              raw: req.body.raw,
              json: req.body.json,
              formUrlEncoded: (req.body.formUrlEncoded ?? []).map((f) => ({
                id: crypto.randomUUID(),
                key: f.key,
                value: f.value,
                enabled: true,
              })),
              formData: (req.body.formData ?? []).map((f) => ({
                id: crypto.randomUUID(),
                key: f.key,
                value: f.value ?? "",
                enabled: true,
                type: (f.type as "text" | "file") || "text",
                filePath: f.filePath,
                fileName: f.fileName,
              })),
            }
          : { type: "none" as const },
        auth: req.auth
          ? {
              type: req.auth.type as "none" | "basic" | "bearer" | "api-key",
              basic: req.auth.basic
                ? { username: req.auth.basic.username, password: req.auth.basic.password }
                : undefined,
              bearer: req.auth.bearer
                ? { token: req.auth.bearer.token }
                : undefined,
              apiKey: req.auth.apiKey
                ? {
                    key: req.auth.apiKey.key,
                    value: req.auth.apiKey.value,
                    addTo: (req.auth.apiKey.addTo as "header" | "query") || "header",
                  }
                : undefined,
            }
          : { type: "none" as const },
        folderId: req.folderId,
        projectId: req.projectId,
        createdAt: req.createdAt,
        updatedAt: req.updatedAt,
      })
      setWorkspaceView("project")
    }
  }, [currentProjectId, createRequest, openRequestTab, setWorkspaceView])

  const handleSave = useCallback(() => {
    if (!editingEnvironmentId && !activeTab) return
    window.dispatchEvent(new CustomEvent("minipost:save"))
  }, [activeTab, editingEnvironmentId])

  const handleCloseTab = useCallback(() => {
    if (editingEnvironmentId) {
      closeActiveEnvironmentTab()
      return
    }
    if (!activeTab?.closable) return
    window.dispatchEvent(new CustomEvent("minipost:close-active-request-tab"))
  }, [activeTab, closeActiveEnvironmentTab, editingEnvironmentId])

  const handleSend = useCallback(() => {
    if (!activeTab) return
    window.dispatchEvent(new CustomEvent("minipost:send"))
  }, [activeTab])

  const handleOpenImport = useCallback(() => {
    window.dispatchEvent(new CustomEvent("minipost:open-import"))
  }, [])

  useEffect(() => {
    const onNewRequest = () => {
      void handleNewRequest()
    }
    window.addEventListener("minipost:new-request", onNewRequest as EventListener)
    return () => window.removeEventListener("minipost:new-request", onNewRequest as EventListener)
  }, [handleNewRequest])

  const shortcuts = useMemo(
    () => ({
      "mod+b": toggleSidebar,
      "mod+n": handleNewRequest,
      "mod+o": handleOpenImport,
      "mod+s": handleSave,
      "mod+w": handleCloseTab,
      "mod+enter": handleSend,
    }),
    [toggleSidebar, handleNewRequest, handleOpenImport, handleSave, handleCloseTab, handleSend]
  )

  useKeyboardShortcuts(shortcuts)

  return (
    <TooltipProvider delayDuration={200} skipDelayDuration={12}>
      <div className="window-frame fixed inset-0 bg-transparent">
        <div className="window-shell h-full w-full bg-[var(--surface)]">
          <AppLayout />
        </div>
      </div>
    </TooltipProvider>
  )
}

export default App
