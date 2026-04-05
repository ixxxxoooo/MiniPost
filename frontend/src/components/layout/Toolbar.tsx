import { useEffect, useState, useCallback, useRef } from "react"
import { AppIcon } from "@/components/ui/icon"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { useUIStore, type Theme } from "@/stores/uiStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore } from "@/stores/tabStore"
import { ImportCurl } from "../../../wailsjs/go/main/App"
import { WindowControls } from "./WindowControls"
import { cn } from "@/lib/utils"

function useTitlebarDoubleClick() {
  const lastClickRef = useRef<{ time: number; x: number; y: number }>({
    time: 0, x: 0, y: 0,
  })
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button !== 0) return
    const now = Date.now()
    const last = lastClickRef.current
    if (now - last.time < 400 && Math.abs(e.clientX - last.x) < 5 && Math.abs(e.clientY - last.y) < 5) {
      import("../../../wailsjs/runtime/runtime").then((r) => r.WindowToggleMaximise())
      lastClickRef.current = { time: 0, x: 0, y: 0 }
    } else {
      lastClickRef.current = { time: now, x: e.clientX, y: e.clientY }
    }
  }, [])
  return { handleMouseDown }
}

export function Toolbar() {
  const { theme, resolved, setTheme, sidebarCollapsed, toggleSidebar, layoutDirection, setLayoutDirection } = useUIStore()
  const { currentProjectId } = useProjectStore()
  const activeTab = useTabStore((s) => s.getActiveTab())
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const [showCurlInput, setShowCurlInput] = useState(false)
  const [curlCommand, setCurlCommand] = useState("")
  const { environments, activeEnvironmentId, setActiveEnvironment, loadEnvironments } = useEnvironmentStore()
  const { handleMouseDown: handleTitlebarMouseDown } = useTitlebarDoubleClick()

  useEffect(() => {
    if (currentProjectId) {
      loadEnvironments(currentProjectId)
    }
  }, [currentProjectId, loadEnvironments])

  return (
    <div
      className={cn(
        "relative z-40 flex h-[var(--size-toolbar)] flex-shrink-0 items-center overflow-hidden border-b titlebar-drag",
        "bg-[var(--toolbar-bg)] border-[var(--toolbar-border)]"
      )}
      style={{ paddingRight: "var(--size-padding-sm)" }}
      onMouseDown={handleTitlebarMouseDown}
    >
      <div onMouseDown={(e) => e.stopPropagation()}>
        <WindowControls />
      </div>

      <div className="titlebar-no-drag flex items-center gap-[var(--size-gap-sm)] ml-1" onMouseDown={(e) => e.stopPropagation()}>
        <button
          className="flex items-center justify-center h-[var(--size-btn)] w-[var(--size-btn)] rounded-[var(--radius-btn)] text-[var(--fg-secondary)] hover:bg-[var(--sidebar-hover)] transition-colors"
          onClick={toggleSidebar}
          title={sidebarCollapsed ? "展开侧边栏 (⌘B)" : "折叠侧边栏 (⌘B)"}
        >
          {sidebarCollapsed ? (
            <AppIcon name="sidebarExpand" size={14} className="text-[var(--fg-secondary)]" />
          ) : (
            <AppIcon name="sidebarCollapse" size={14} className="text-[var(--fg-secondary)]" />
          )}
        </button>

        <div className="w-px h-3 bg-[var(--border-color)] mx-0.5" />

        <button
          className="flex items-center justify-center h-[var(--size-btn)] w-[var(--size-btn)] rounded-[var(--radius-btn)] text-[var(--fg-secondary)] hover:bg-[var(--sidebar-hover)] transition-colors"
          onClick={() => setLayoutDirection(layoutDirection === "vertical" ? "horizontal" : "vertical")}
          title={layoutDirection === "vertical" ? "切换为左右布局" : "切换为上下布局"}
        >
          {layoutDirection === "vertical" ? (
            <AppIcon name="arrowLeftRight" size={14} className="text-[var(--fg-secondary)]" />
          ) : (
            <AppIcon name="arrowUpDown" size={14} className="text-[var(--fg-secondary)]" />
          )}
        </button>
      </div>

      {/* 居中 - 当前项目/标题 */}
      <div className="flex-1" />

      {currentProjectId && (
        <div
          className="titlebar-no-drag absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2"
          onMouseDown={(e) => e.stopPropagation()}
        >
          <span className="text-[length:var(--size-font-2xs)] text-[var(--fg-secondary)] font-medium select-none">
            MiniPost
          </span>
        </div>
      )}

      <div className="flex-1" />

      {/* 右侧功能区 */}
      <div className="titlebar-no-drag flex items-center gap-[var(--size-gap-sm)] mr-0.5" onMouseDown={(e) => e.stopPropagation()}>
        {showCurlInput && (
          <Input
            value={curlCommand}
            onChange={(e) => setCurlCommand(e.target.value)}
            placeholder="粘贴 cURL 命令..."
            className="w-64 h-[var(--size-btn-sm)] font-mono text-[length:var(--size-font-2xs)]"
            autoFocus
            onKeyDown={async (e) => {
              if (e.key === "Enter" && curlCommand.trim() && activeTab) {
                try {
                  const result = await ImportCurl(curlCommand.trim())
                  updateTabRequest(activeTab.id, {
                    method: result.method as "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS",
                    url: result.url,
                    headers: (result.headers ?? []).map((h: { key: string; value: string }) => ({
                      id: crypto.randomUUID(), key: h.key, value: h.value, enabled: true,
                    })),
                    body: result.body
                      ? { type: result.body.type as "none" | "raw" | "json" | "form-urlencoded", raw: result.body.raw, json: result.body.json }
                      : { type: "none" as const },
                  })
                  setCurlCommand("")
                  setShowCurlInput(false)
                } catch (err) {
                  console.error("cURL 解析失败:", err)
                }
              }
              if (e.key === "Escape") {
                setShowCurlInput(false)
                setCurlCommand("")
              }
            }}
          />
        )}

        {currentProjectId && environments.length > 0 && (
          <>
            <AppIcon name="globe" size={14} className="text-[var(--fg-muted)]" />
            <Select
              value={activeEnvironmentId ?? "none"}
              onValueChange={(v) => setActiveEnvironment(v === "none" ? null : v)}
            >
              <SelectTrigger className="h-[var(--size-btn-sm)] w-[120px] text-[length:var(--size-font-2xs)] border-0 bg-transparent">
                <SelectValue placeholder="No Env" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">No Environment</SelectItem>
                {environments.map((env) => (
                  <SelectItem key={env.id} value={env.id}>{env.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        )}

        <button
          className="flex items-center justify-center h-[var(--size-btn)] w-[var(--size-btn)] rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors"
          onClick={() => setShowCurlInput(!showCurlInput)}
          title="导入 cURL"
        >
          <AppIcon name="upload" size={14} className="text-[var(--fg-secondary)]" />
        </button>

        <button
          className="flex items-center justify-center h-[var(--size-btn)] w-[var(--size-btn)] rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors"
          onClick={() => setTheme(resolved === "dark" ? "light" : "dark")}
          title={resolved === "dark" ? "切换为浅色主题" : "切换为深色主题"}
        >
          {resolved === "dark"
            ? <AppIcon name="sun" size={14} className="text-[var(--fg-secondary)]" />
            : <AppIcon name="moon" size={14} className="text-[var(--fg-secondary)]" />
          }
        </button>
      </div>
    </div>
  )
}
