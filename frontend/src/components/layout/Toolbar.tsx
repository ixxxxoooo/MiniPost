import { useEffect, useState, useCallback, useRef } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { Input } from "@/components/ui/input"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { ImportCurl } from "../../../wailsjs/go/main/App"
import { WindowControls } from "./WindowControls"
import { WorkspaceHeader } from "./WorkspaceHeader"
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
  const { resolved, setTheme, setSettingsOpen } = useUIStore()
  const { currentProjectId } = useProjectStore()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const [showCurlDialog, setShowCurlDialog] = useState(false)
  const [curlCommand, setCurlCommand] = useState("")
  const [curlError, setCurlError] = useState("")
  const { handleMouseDown: handleTitlebarMouseDown } = useTitlebarDoubleClick()

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === ",") {
        e.preventDefault()
        setSettingsOpen(true)
      }
      if ((e.metaKey || e.ctrlKey) && e.key === "i") {
        e.preventDefault()
        setShowCurlDialog(true)
      }
    }
    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [setSettingsOpen])

  useEffect(() => {
    if (!showCurlDialog) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault()
        setShowCurlDialog(false)
        setCurlError("")
      }
    }
    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [showCurlDialog])

  const handleCurlImport = async () => {
    if (!curlCommand.trim() || !activeTab) return
    setCurlError("")
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
      setShowCurlDialog(false)
    } catch (err) {
      setCurlError(err instanceof Error ? err.message : "cURL 解析失败")
    }
  }

  return (
    <>
      <div
        className={cn(
          "relative z-40 flex h-[var(--size-toolbar)] flex-shrink-0 items-center overflow-visible border-b titlebar-drag",
          "bg-[var(--toolbar-bg)] border-[var(--toolbar-border)] px-2.5"
        )}
        onMouseDown={handleTitlebarMouseDown}
      >
        <div onMouseDown={(e) => e.stopPropagation()}>
          <WindowControls />
        </div>

        <div className="ml-1.5">
          <WorkspaceHeader />
        </div>

        <div className="flex-1" />

        <div className="titlebar-no-drag flex items-center gap-[var(--size-gap-sm)]" onMouseDown={(e) => e.stopPropagation()}>
          <button
            className="flex h-[24px] w-[24px] items-center justify-center rounded-[7px] transition-colors hover:bg-[var(--surface-secondary)]"
            onClick={() => setShowCurlDialog(true)}
            title="导入 cURL (⌘I)"
            type="button"
          >
            <AppIcon name="fileImport" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
          </button>

          <button
            className="flex h-[24px] w-[24px] items-center justify-center rounded-[7px] transition-colors hover:bg-[var(--surface-secondary)]"
            onClick={() => setTheme(resolved === "dark" ? "light" : "dark")}
            title={resolved === "dark" ? "切换为浅色主题" : "切换为深色主题"}
            type="button"
          >
            {resolved === "dark"
              ? <AppIcon name="sun" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
              : <AppIcon name="moon" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
            }
          </button>

          <button
            className="flex h-[24px] w-[24px] items-center justify-center rounded-[7px] transition-colors hover:bg-[var(--surface-secondary)]"
            onClick={() => setSettingsOpen(true)}
            title="设置 (⌘,)"
            type="button"
          >
            <AppIcon name="settings" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
          </button>
        </div>
      </div>

      {/* cURL 导入弹窗 */}
      {showCurlDialog && createPortal(
        <div
          className="fixed inset-0 z-[300] flex items-center justify-center"
          onClick={() => { setShowCurlDialog(false); setCurlError("") }}
        >
          <div className="absolute inset-0 bg-black/50 backdrop-blur-[2px]" />
          <div
            className={cn(
              "relative z-[301] w-[520px] rounded-[12px] border shadow-2xl",
              "bg-[var(--surface)] border-[var(--border-color)]",
              "animate-in fade-in zoom-in-95 duration-150"
            )}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-3 px-6 pt-6 pb-3">
              <div className="h-8 w-8 rounded-[8px] bg-[var(--accent)]/10 flex items-center justify-center flex-shrink-0">
                <AppIcon name="fileImport" size={16} className="text-[var(--accent)]" />
              </div>
              <div className="flex-1">
                <h3 className="text-[14px] font-semibold text-[var(--fg)]">导入 cURL</h3>
                <p className="text-[11px] text-[var(--fg-muted)] mt-0.5">粘贴 cURL 命令以快速创建请求</p>
              </div>
              <button
                className="h-6 w-6 flex items-center justify-center rounded-[6px] hover:bg-[var(--sidebar-hover)] transition-colors"
                onClick={() => { setShowCurlDialog(false); setCurlError("") }}
              >
                <AppIcon name="clear" size={12} className="text-[var(--fg-muted)]" />
              </button>
            </div>

            <div className="px-6 pb-3">
              <textarea
                value={curlCommand}
                onChange={(e) => setCurlCommand(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault()
                    void handleCurlImport()
                  }
                }}
                placeholder={'curl --request GET \\\n  --url https://api.example.com/data \\\n  --header \'Content-Type: application/json\''}
                className={cn(
                  "w-full min-h-[180px] p-3.5 rounded-[8px]",
                  "border border-[var(--border-color)] bg-[var(--surface-secondary)]",
                  "text-[var(--fg)] font-mono text-[12px] leading-relaxed",
                  "placeholder:text-[var(--fg-muted)]/60 resize-y",
                  "focus:outline-none focus:border-[var(--accent)] focus:ring-1 focus:ring-[var(--accent)]/30",
                  "transition-all duration-150"
                )}
                autoFocus
              />
              {curlError && (
                <div className="flex items-center gap-1.5 mt-2 text-[11px] text-[var(--danger)]">
                  <AppIcon name="info" size={12} className="text-[var(--danger)]" />
                  {curlError}
                </div>
              )}
              <div className="mt-2 text-[10px] text-[var(--fg-muted)]">
                Enter 导入，Shift + Enter 换行
              </div>
            </div>

            <div className="flex items-center justify-between px-6 pb-6 pt-1">
              <span className="text-[10px] text-[var(--fg-muted)]">
                <kbd className="px-1 py-0.5 rounded bg-[var(--surface-secondary)] border border-[var(--border-color)] text-[9px] font-mono">⌘I</kbd>
                <span className="ml-1.5">快速唤起</span>
              </span>
              <div className="flex gap-2">
                <button
                  className={cn(
                    "h-[32px] px-4 rounded-[8px] text-[12px] font-medium transition-colors",
                    "border border-[var(--border-color)] text-[var(--fg-secondary)] hover:bg-[var(--surface-secondary)]"
                  )}
                  onClick={() => { setShowCurlDialog(false); setCurlError("") }}
                >
                  取消
                </button>
                <button
                  className={cn(
                    "h-[32px] px-5 rounded-[8px] text-[12px] font-medium transition-colors",
                    "bg-[var(--accent)] text-white hover:brightness-110",
                    "disabled:opacity-40 disabled:pointer-events-none"
                  )}
                  disabled={!curlCommand.trim() || !activeTab}
                  onClick={() => void handleCurlImport()}
                >
                  导入
                </button>
              </div>
            </div>
          </div>
        </div>,
        document.body
      )}
    </>
  )
}
