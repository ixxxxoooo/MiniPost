import { useRef, useCallback, useState } from "react"
import { Toolbar } from "./Toolbar"
import { BottomBar } from "./BottomBar"
import { Sidebar } from "./Sidebar"
import { TabBar } from "./TabBar"
import { SettingsPanel } from "./SettingsPanel"
import { RequestEditorBody, RequestEditorToolbar } from "@/components/business/editor/RequestEditor"
import { ResponseViewer } from "@/components/business/response/ResponseViewer"
import { EnvironmentEditorPage } from "@/components/business/environment/EnvironmentEditorPage"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { cn } from "@/lib/utils"

export function AppLayout() {
  const { sidebarCollapsed, layoutDirection, editingEnvironmentId } = useUIStore()
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const activeTab = useTabStore(getProjectActiveTabFromState)

  const [splitRatio, setSplitRatio] = useState(0.55)
  const splitContainerRef = useRef<HTMLDivElement>(null)
  const resizingRef = useRef(false)

  const handleSplitDragStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    resizingRef.current = true
    const container = splitContainerRef.current
    if (!container) return

    const onMove = (ev: MouseEvent) => {
      if (!resizingRef.current || !container) return
      const rect = container.getBoundingClientRect()
      let ratio: number
      if (layoutDirection === "vertical") {
        ratio = (ev.clientY - rect.top) / rect.height
      } else {
        ratio = (ev.clientX - rect.left) / rect.width
      }
      setSplitRatio(Math.max(0.2, Math.min(0.8, ratio)))
    }
    const onUp = () => {
      resizingRef.current = false
      document.removeEventListener("mousemove", onMove)
      document.removeEventListener("mouseup", onUp)
    }
    document.addEventListener("mousemove", onMove)
    document.addEventListener("mouseup", onUp)
  }, [layoutDirection])

  const showEnvEditor = !!editingEnvironmentId

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-[var(--surface)]">
      <Toolbar />

      <div className="flex flex-1 min-h-0 overflow-hidden">
        <Sidebar />

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
                    [layoutDirection === "vertical" ? "height" : "width"]: `${splitRatio * 100}%`,
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
                        ? "inset-x-0 -top-2 h-[5px] cursor-row-resize"
                        : "inset-y-0 -left-2 w-[5px] cursor-col-resize"
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
              <div className="text-center">
                <div className="text-[40px] mb-3 opacity-20">⚡</div>
                <p className="text-[length:var(--size-font-sm)] text-[var(--fg-secondary)] font-medium">MiniPost</p>
                <p className="text-2xs text-[var(--fg-muted)] mt-1">选择一个请求或创建新请求开始</p>
              </div>
            </div>
          ) : (
            <div className="flex flex-1 min-h-0 items-center justify-center bg-[var(--surface)]">
              <div className="text-center max-w-[320px] px-6">
                <div className="text-[40px] mb-3 opacity-20">📁</div>
                <p className="text-[length:var(--size-font-sm)] text-[var(--fg-secondary)] font-medium">请选择工作区</p>
                <p className="text-2xs text-[var(--fg-muted)] mt-1">先在顶部选择一个工作区，然后再打开或创建请求标签</p>
              </div>
            </div>
          )}

          <BottomBar />
        </div>
      </div>

      <SettingsPanel />
    </div>
  )
}
