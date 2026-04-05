import { useRef, useCallback, useState } from "react"
import { Toolbar } from "./Toolbar"
import { BottomBar } from "./BottomBar"
import { Sidebar } from "./Sidebar"
import { TabBar } from "./TabBar"
import { RequestEditor } from "@/components/business/editor/RequestEditor"
import { ResponseViewer } from "@/components/business/response/ResponseViewer"
import { useUIStore } from "@/stores/uiStore"
import { useTabStore } from "@/stores/tabStore"
import { cn } from "@/lib/utils"

export function AppLayout() {
  const { sidebarCollapsed, layoutDirection } = useUIStore()
  const activeTab = useTabStore((s) => s.getActiveTab())

  // 请求/响应分割拖拽
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

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-[var(--surface)]">
      <Toolbar />

      <div className="flex flex-1 min-h-0 overflow-hidden">
        <Sidebar />

        <div className="flex min-w-0 min-h-0 flex-1 flex-col overflow-hidden">
          <TabBar />

          {activeTab ? (
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
                <RequestEditor />
              </div>

              <div
                className={cn(
                  "group flex-shrink-0 flex items-center justify-center relative",
                  layoutDirection === "vertical"
                    ? "h-[5px] cursor-row-resize"
                    : "w-[5px] cursor-col-resize"
                )}
                onMouseDown={handleSplitDragStart}
              >
                <div
                  className={cn(
                    "bg-[var(--border-color)] group-hover:bg-[var(--accent)] transition-colors duration-200",
                    layoutDirection === "vertical"
                      ? "h-px w-full absolute top-1/2 -translate-y-1/2"
                      : "w-px h-full absolute left-1/2 -translate-x-1/2"
                  )}
                />
                <div
                  className={cn(
                    "bg-transparent group-hover:bg-[var(--accent)]/50 transition-all duration-200 rounded-full absolute",
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
          ) : (
            <div className="flex flex-1 min-h-0 items-center justify-center bg-[var(--surface)]">
              <div className="text-center">
                <div className="text-[40px] mb-3 opacity-20">⚡</div>
                <p className="text-[length:var(--size-font-sm)] text-[var(--fg-secondary)] font-medium">MiniPost</p>
                <p className="text-2xs text-[var(--fg-muted)] mt-1">选择一个请求或创建新请求开始</p>
              </div>
            </div>
          )}

          <BottomBar />
        </div>
      </div>
    </div>
  )
}
