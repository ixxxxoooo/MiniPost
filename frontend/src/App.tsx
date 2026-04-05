import { useCallback, useEffect, useMemo } from "react"
import { TooltipProvider } from "@/components/ui/tooltip"
import { AppLayout } from "@/components/layout/AppLayout"
import { useKeyboardShortcuts } from "@/hooks/useKeyboardShortcuts"
import type { HttpMethod } from "@/lib/constants"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { info, error as logError } from "@/lib/logger"

function App() {
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)
  const editingEnvironmentId = useUIStore((s) => s.editingEnvironmentId)
  const closeActiveEnvironmentTab = useUIStore((s) => s.closeActiveEnvironmentTab)
  const { currentProjectId, createRequest } = useProjectStore()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const openRequestTab = useTabStore((s) => s.openRequestTab)
  const removeTab = useTabStore((s) => s.removeTab)

  useEffect(() => {
    info("App", "React App 组件已挂载, 开始加载项目列表")
    useProjectStore.getState().loadProjects()
      .then(() => info("App", "项目列表加载完成"))
      .catch((err) => logError("App", "项目列表加载失败", { error: String(err) }))
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
              type: req.body.type as "none" | "raw" | "json" | "form-urlencoded",
              raw: req.body.raw,
              json: req.body.json,
              formUrlEncoded: (req.body.formUrlEncoded ?? []).map((f) => ({
                id: crypto.randomUUID(),
                key: f.key,
                value: f.value,
                enabled: true,
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
    }
  }, [currentProjectId, createRequest, openRequestTab])

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
    removeTab(activeTab.id)
  }, [activeTab, closeActiveEnvironmentTab, editingEnvironmentId, removeTab])

  const handleSend = useCallback(() => {
    if (!activeTab) return
    window.dispatchEvent(new CustomEvent("minipost:send"))
  }, [activeTab])

  const shortcuts = useMemo(
    () => ({
      "mod+b": toggleSidebar,
      "mod+n": handleNewRequest,
      "mod+s": handleSave,
      "mod+w": handleCloseTab,
      "mod+enter": handleSend,
    }),
    [toggleSidebar, handleNewRequest, handleSave, handleCloseTab, handleSend]
  )

  useKeyboardShortcuts(shortcuts)

  return (
    <TooltipProvider delayDuration={300}>
      <div className="window-frame fixed inset-0 bg-transparent">
        <div className="window-shell h-full w-full bg-[var(--surface)]">
          <AppLayout />
        </div>
      </div>
    </TooltipProvider>
  )
}

export default App
