import { useEffect, useState, useCallback, useRef } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useI18n } from "@/hooks/useI18n"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore } from "@/stores/tabStore"
import { ImportCurl } from "../../../wailsjs/go/main/App"
import { WindowControls } from "./WindowControls"
import { WorkspaceHeader } from "./WorkspaceHeader"
import { cn } from "@/lib/utils"
import { buildDraftRequestFromCurl } from "@/lib/curlImportDraft"
import { info } from "@/lib/logger"

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
  const { t } = useI18n()
  const { resolved, setTheme, setSettingsOpen } = useUIStore()
  const { currentProjectId } = useProjectStore()
  const addTab = useTabStore((s) => s.addTab)
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
    if (!curlCommand.trim()) return
    setCurlError("")
    try {
      if (!currentProjectId) {
        setCurlError(t("请先选择项目后再导入", "Please select a project before importing"))
        return
      }

      const result = await ImportCurl(curlCommand.trim())
      const draftRequest = buildDraftRequestFromCurl(result, {
        projectId: currentProjectId,
        name: "Imported cURL",
      })
      info("CurlImport", "cURL imported into draft tab", {
        source: "toolbar",
        commandLength: curlCommand.trim().length,
        method: draftRequest.method,
        hasUrl: Boolean(draftRequest.url),
        headerCount: draftRequest.headers.filter((item) => item.key.trim()).length,
        bodyType: draftRequest.body.type,
      })
      addTab({
        title: draftRequest.name || "Imported cURL",
        projectId: currentProjectId,
        closable: true,
        dirty: true,
        request: draftRequest,
        response: null,
        responseError: null,
      })

      setCurlCommand("")
      setShowCurlDialog(false)
    } catch (err) {
      setCurlError(err instanceof Error ? err.message : t("cURL 解析失败", "Failed to parse cURL"))
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
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className="flex h-[24px] w-[24px] items-center justify-center rounded-[7px] transition-colors hover:bg-[var(--surface-secondary)]"
                onClick={() => setShowCurlDialog(true)}
                type="button"
              >
                <AppIcon name="fileImport" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("导入 cURL", "Import cURL")} (⌘I)</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className="flex h-[24px] w-[24px] items-center justify-center rounded-[7px] transition-colors hover:bg-[var(--surface-secondary)]"
                onClick={() => setTheme(resolved === "dark" ? "light" : "dark")}
                type="button"
              >
                {resolved === "dark"
                  ? <AppIcon name="sun" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
                  : <AppIcon name="moon" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
                }
              </button>
            </TooltipTrigger>
            <TooltipContent>{resolved === "dark" ? t("切换为浅色主题", "Switch to light theme") : t("切换为深色主题", "Switch to dark theme")}</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className="flex h-[24px] w-[24px] items-center justify-center rounded-[7px] transition-colors hover:bg-[var(--surface-secondary)]"
                onClick={() => setSettingsOpen(true)}
                type="button"
              >
                <AppIcon name="settings" size={14} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("设置", "Settings")} (⌘,)</TooltipContent>
          </Tooltip>
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
              "relative z-[301] w-[500px] rounded-[12px] border shadow-2xl",
              "bg-[var(--surface)] border-[var(--border-color)]",
              "animate-in fade-in zoom-in-95 duration-150"
            )}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-2.5 px-5 pt-4 pb-2">
              <div className="h-7 w-7 rounded-[8px] bg-[var(--accent)]/10 flex items-center justify-center flex-shrink-0">
                <AppIcon name="fileImport" size={15} className="text-[var(--accent)]" />
              </div>
              <div className="flex-1">
                <h3 className="text-[14px] font-semibold text-[var(--fg)]">{t("导入 cURL", "Import cURL")}</h3>
                <p className="text-[11px] text-[var(--fg-muted)] mt-0.5">{t("粘贴 cURL 命令以快速创建请求", "Paste a cURL command to quickly create a request")}</p>
              </div>
              <button
                className="h-5 w-5 flex items-center justify-center rounded-[6px] hover:bg-[var(--sidebar-hover)] transition-colors"
                onClick={() => { setShowCurlDialog(false); setCurlError("") }}
              >
                <AppIcon name="clear" size={11} className="text-[var(--fg-muted)]" />
              </button>
            </div>

            <div className="px-5 pb-2">
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
                  "focus:outline-none focus-visible:outline-none focus:ring-0 focus-visible:ring-0 focus:border-[var(--accent)]",
                  "transition-colors duration-150"
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
                {t("Enter 导入，Shift + Enter 换行", "Press Enter to import, Shift + Enter for newline")}
              </div>
            </div>

            <div className="flex items-center justify-between px-5 pb-4 pt-0.5">
              <span className="text-[10px] text-[var(--fg-muted)]">
                <kbd className="px-1 py-0.5 rounded bg-[var(--surface-secondary)] border border-[var(--border-color)] text-[9px] font-mono">⌘I</kbd>
                <span className="ml-1.5">{t("快速唤起", "Quick launch")}</span>
              </span>
              <div className="flex gap-1.5 flex-shrink-0">
                <button
                  className={cn(
                    "h-[30px] min-w-[84px] px-3.5 rounded-[8px] text-[12px] font-medium whitespace-nowrap transition-colors flex-shrink-0",
                    "border border-[var(--border-color)] text-[var(--fg-secondary)] hover:bg-[var(--surface-secondary)]"
                  )}
                  onClick={() => { setShowCurlDialog(false); setCurlError("") }}
                >
                  {t("取消", "Cancel")}
                </button>
                <button
                  className={cn(
                    "h-[30px] min-w-[84px] px-4.5 rounded-[8px] text-[12px] font-medium whitespace-nowrap transition-colors flex-shrink-0",
                    "bg-[var(--accent)] text-white hover:brightness-110",
                    "disabled:opacity-40 disabled:pointer-events-none"
                  )}
                  disabled={!curlCommand.trim() || !currentProjectId}
                  onClick={() => void handleCurlImport()}
                >
                  {t("导入", "Import")} (Enter)
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
