import { useRef, useCallback, useState, useEffect } from "react"
import { Toolbar } from "./Toolbar"
import { BottomBar } from "./BottomBar"
import { Sidebar } from "./Sidebar"
import { TabBar } from "./TabBar"
import { SettingsPanel } from "./SettingsPanel"
import { RequestEditorBody, RequestEditorToolbar } from "@/components/business/editor/RequestEditor"
import { ResponseViewer } from "@/components/business/response/ResponseViewer"
import { EnvironmentEditorPage } from "@/components/business/environment/EnvironmentEditorPage"
import { ProjectHome } from "./ProjectHome"
import { useI18n } from "@/hooks/useI18n"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { cn } from "@/lib/utils"
import appLogo from "@/assets/images/appicon.png"

export function AppLayout() {
  const { t } = useI18n()
  const { layoutDirection, editingEnvironmentId, workspaceView, sidebarCollapsed } = useUIStore()
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const activeTab = useTabStore(getProjectActiveTabFromState)

  const [splitRatio, setSplitRatio] = useState(0.55)
  const [sidebarPreviewOpen, setSidebarPreviewOpen] = useState(false)
  const splitContainerRef = useRef<HTMLDivElement>(null)
  const resizingRef = useRef(false)
  const splitRafRef = useRef<number | null>(null)
  const pendingRatioRef = useRef<number>(0.55)

  useEffect(() => {
    pendingRatioRef.current = splitRatio
    const container = splitContainerRef.current
    if (container) container.style.setProperty("--split-ratio", String(splitRatio))
  }, [splitRatio])

  useEffect(() => {
    return () => {
      if (splitRafRef.current !== null) {
        window.cancelAnimationFrame(splitRafRef.current)
        splitRafRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    if (!sidebarCollapsed) setSidebarPreviewOpen(false)
  }, [sidebarCollapsed])

  const handleSplitDragStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    resizingRef.current = true
    const container = splitContainerRef.current
    if (!container) return

    const applyPendingRatio = () => {
      splitRafRef.current = null
      const ratio = pendingRatioRef.current
      container.style.setProperty("--split-ratio", String(ratio))
    }

    const onMove = (ev: MouseEvent) => {
      if (!resizingRef.current || !container) return
      const rect = container.getBoundingClientRect()
      let ratio: number
      if (layoutDirection === "vertical") {
        ratio = (ev.clientY - rect.top) / rect.height
      } else {
        ratio = (ev.clientX - rect.left) / rect.width
      }
      pendingRatioRef.current = Math.max(0.2, Math.min(0.8, ratio))
      if (splitRafRef.current === null) {
        splitRafRef.current = window.requestAnimationFrame(applyPendingRatio)
      }
    }
    const onUp = () => {
      resizingRef.current = false
      if (splitRafRef.current !== null) {
        window.cancelAnimationFrame(splitRafRef.current)
        splitRafRef.current = null
      }
      setSplitRatio(pendingRatioRef.current)
      document.removeEventListener("mousemove", onMove)
      document.removeEventListener("mouseup", onUp)
    }
    document.addEventListener("mousemove", onMove)
    document.addEventListener("mouseup", onUp)
  }, [layoutDirection])

  const showEnvEditor = !!editingEnvironmentId
  const showProjectHome = workspaceView === "home"

  const triggerCreateNewRequest = useCallback(() => {
    window.dispatchEvent(new CustomEvent("minipost:new-request"))
  }, [])

  const triggerOpenImport = useCallback(() => {
    window.dispatchEvent(new CustomEvent("minipost:open-import"))
  }, [])

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-[var(--surface)]">
      <Toolbar />

      <div className="relative flex flex-1 min-h-0 overflow-hidden">
        {showProjectHome ? (
          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            <ProjectHome />
            <BottomBar />
          </div>
        ) : (
          <>
            {!sidebarCollapsed && <Sidebar />}

            {sidebarCollapsed && (
              <>
                {sidebarPreviewOpen && (
                  <div
                    className="absolute bottom-0 left-0 top-0 z-40 titlebar-no-drag shadow-[var(--shadow-lg)]"
                    data-testid="sidebar-preview"
                    onMouseLeave={() => setSidebarPreviewOpen(false)}
                  >
                    <Sidebar forceOpen />
                  </div>
                )}
                <button
                  type="button"
                  aria-label={t("临时展开侧边栏", "Temporarily reveal sidebar")}
                  className={cn(
                    "absolute bottom-0 left-0 top-0 z-30 w-2 cursor-default titlebar-no-drag",
                    "bg-transparent transition-colors hover:bg-[var(--accent)]/10 focus-visible:bg-[var(--accent)]/10 focus-visible:outline-none",
                    sidebarPreviewOpen && "pointer-events-none"
                  )}
                  data-testid="sidebar-reveal-rail"
                  onMouseEnter={() => setSidebarPreviewOpen(true)}
                  onFocus={() => setSidebarPreviewOpen(true)}
                  onBlur={() => setSidebarPreviewOpen(false)}
                />
              </>
            )}

            <div className="flex min-w-0 min-h-0 flex-1 flex-col overflow-hidden">
              <TabBar />

              {showEnvEditor ? (
                <EnvironmentEditorPage />
              ) : activeTab ? (
                <div className="flex flex-1 min-h-0 min-w-0 flex-col overflow-hidden">
                  <RequestEditorToolbar />

                  <div
                    ref={splitContainerRef}
                    className={cn(
                      "flex-1 overflow-hidden flex min-h-0 min-w-0",
                      layoutDirection === "vertical" ? "flex-col" : "flex-row"
                    )}
                  >
                    <div
                      className="overflow-hidden min-h-0 min-w-0"
                      style={{
                        [layoutDirection === "vertical" ? "height" : "width"]: `calc(var(--split-ratio, ${splitRatio}) * 100%)`,
                        flexShrink: 0,
                      }}
                    >
                      <RequestEditorBody />
                    </div>

                    <div
                      className={cn(
                        "group relative flex-shrink-0",
                        layoutDirection === "vertical"
                          ? "h-px"
                          : "w-px"
                      )}
                    >
                      <div
                        className={cn(
                          "absolute z-10",
                          layoutDirection === "vertical"
                            ? "left-0 right-0 top-1/2 h-[6px] -translate-y-1/2 cursor-row-resize"
                            : "top-0 bottom-0 left-1/2 w-[6px] -translate-x-1/2 cursor-col-resize"
                        )}
                        onMouseDown={handleSplitDragStart}
                      />
                      <div
                        className={cn(
                          "absolute inset-0 bg-[var(--border-color)] group-hover:bg-[var(--accent)] transition-colors duration-200"
                        )}
                      />
                      <div
                        className={cn(
                          "pointer-events-none bg-transparent group-hover:bg-[var(--accent)]/50 transition-all duration-200 rounded-full absolute",
                          layoutDirection === "vertical"
                            ? "w-6 h-1 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2"
                            : "h-6 w-1 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2"
                        )}
                      />
                    </div>

                    <div className="flex-1 overflow-hidden min-h-0 min-w-0">
                      <ResponseViewer />
                    </div>
                  </div>
                </div>
              ) : currentProjectId ? (
                <div className="flex flex-1 min-h-0 items-center justify-center bg-[var(--surface)]">
                  <div className="w-[300px]">
                    <div className="mb-4 flex flex-col items-center text-center">
                      <img src={appLogo} alt="MiniPost" className="h-12 w-12 object-contain" />
                      <div className="mt-2 text-[13px] text-[var(--fg-secondary)]">MiniPost</div>
                      <div className="mt-1 text-[11px] leading-5 text-[var(--fg-muted)]">
                        {t("一个简洁、快速的 API 调试工具", "A simple and fast API debugging tool")}
                      </div>
                    </div>

                    <button
                      type="button"
                      onClick={triggerCreateNewRequest}
                      className={cn(
                        "flex h-8 w-full items-center justify-between rounded-[7px] px-2 transition-colors",
                        "hover:bg-[var(--surface-secondary)] text-left"
                      )}
                    >
                      <span className="text-[12px] text-[var(--fg-muted)]">{t("新建请求", "Create new")}</span>
                      <span className="inline-flex items-center gap-1 text-[10px] text-[var(--fg-muted)] opacity-90">
                        <kbd className="h-5 min-w-5 rounded-[5px] border border-[var(--border-color)] bg-[var(--surface-secondary)] px-1 font-mono text-[var(--fg-muted)]">⌘</kbd>
                        <kbd className="h-5 min-w-5 rounded-[5px] border border-[var(--border-color)] bg-[var(--surface-secondary)] px-1 font-mono text-[var(--fg-muted)]">N</kbd>
                      </span>
                    </button>

                    <button
                      type="button"
                      onClick={triggerOpenImport}
                      className={cn(
                        "mt-0.5 flex h-8 w-full items-center justify-between rounded-[7px] px-2 transition-colors",
                        "hover:bg-[var(--surface-secondary)] text-left"
                      )}
                    >
                      <span className="text-[12px] text-[var(--fg-muted)]">{t("导入", "Import")}</span>
                      <span className="inline-flex items-center gap-1 text-[10px] text-[var(--fg-muted)] opacity-90">
                        <kbd className="h-5 min-w-5 rounded-[5px] border border-[var(--border-color)] bg-[var(--surface-secondary)] px-1 font-mono text-[var(--fg-muted)]">⌘</kbd>
                        <kbd className="h-5 min-w-5 rounded-[5px] border border-[var(--border-color)] bg-[var(--surface-secondary)] px-1 font-mono text-[var(--fg-muted)]">O</kbd>
                      </span>
                    </button>
                  </div>
                </div>
              ) : (
                <div className="flex flex-1 min-h-0 items-center justify-center bg-[var(--surface)]">
                  <div className="text-center max-w-[320px] px-6">
                    <div className="text-[40px] mb-3 opacity-20">📁</div>
                    <p className="text-[length:var(--size-font-sm)] text-[var(--fg-secondary)] font-medium">{t("请选择工作区", "Please select a workspace")}</p>
                    <p className="text-2xs text-[var(--fg-muted)] mt-1">{t("先在顶部选择一个工作区，然后再打开或创建请求标签", "Select a workspace at the top, then open or create request tabs")}</p>
                  </div>
                </div>
              )}

              <BottomBar />
            </div>
          </>
        )}
      </div>

      <SettingsPanel />
    </div>
  )
}
