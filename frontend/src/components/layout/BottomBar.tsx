import { useState, useRef, useEffect, useCallback } from "react"
import { cn } from "@/lib/utils"
import { AppIcon } from "@/components/ui/icon"
import { useUIStore, type ConsoleEntry } from "@/stores/uiStore"
import { useCookieStore } from "@/stores/cookieStore"
import { CookiePanel } from "@/components/business/cookie/CookiePanel"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"

function formatTimestamp(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleTimeString("zh-CN", { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" })
}

function getStatusColor(code: number): string {
  if (code >= 200 && code < 300) return "text-[var(--success)]"
  if (code >= 400) return "text-[var(--danger)]"
  return "text-[var(--warning)]"
}

function ConsoleRow({ entry }: { entry: ConsoleEntry }) {
  const [expanded, setExpanded] = useState(false)
  const hasError = !!entry.error
  const hasResponse = entry.status !== undefined

  return (
    <div className="border-b border-[var(--border-color)]/50 last:border-b-0">
      <div
        className="flex items-center gap-2 px-3 py-1.5 text-[11px] font-mono hover:bg-[var(--surface-secondary)] transition-colors cursor-pointer select-none"
        onClick={() => setExpanded(!expanded)}
      >
        <AppIcon name={expanded ? "arrowDown" : "arrowRight"} size={8} className="text-[var(--fg-muted)] flex-shrink-0" />
        <span className={cn(
          "font-bold w-[44px] flex-shrink-0 uppercase",
          METHOD_COLORS[(entry.method as HttpMethod)] || "text-[var(--fg-muted)]"
        )}>
          {entry.method}
        </span>
        <span className="text-[var(--fg)] truncate flex-1">{entry.url}</span>
        {hasError && (
          <span className="text-[var(--danger)] flex-shrink-0 text-[10px]">Error</span>
        )}
        {hasResponse && !hasError && (
          <>
            <span className={cn("font-bold flex-shrink-0", getStatusColor(entry.status!))}>
              {entry.status}
            </span>
            <span className="text-[var(--fg-muted)] flex-shrink-0">{Math.round(entry.duration ?? 0)}ms</span>
            {entry.size !== undefined && (
              <span className="text-[var(--fg-muted)] flex-shrink-0">
                {entry.size < 1024 ? `${entry.size}B` : `${(entry.size / 1024).toFixed(1)}KB`}
              </span>
            )}
          </>
        )}
        {!hasResponse && !hasError && (
          <div className="h-2.5 w-2.5 border-[1.5px] border-[var(--fg-muted)] border-t-transparent rounded-full animate-spin flex-shrink-0" />
        )}
        <span className="text-[var(--fg-muted)] text-[10px] flex-shrink-0 ml-1">{formatTimestamp(entry.timestamp)}</span>
      </div>

      {expanded && (
        <div className="px-6 py-2 text-[10px] font-mono bg-[var(--surface-secondary)]/50 space-y-2">
          <details open>
            <summary className="text-[var(--fg-muted)] cursor-pointer select-none font-medium mb-1">Network</summary>
            <div className="ml-2 space-y-0.5 text-[var(--fg-secondary)]">
              <div>{entry.method} {entry.url}</div>
              {entry.status !== undefined && (
                <div>Status: <span className={getStatusColor(entry.status)}>{entry.status}</span></div>
              )}
              {entry.duration !== undefined && <div>Duration: {Math.round(entry.duration)}ms</div>}
              {entry.size !== undefined && <div>Size: {entry.size < 1024 ? `${entry.size}B` : `${(entry.size / 1024).toFixed(1)}KB`}</div>}
            </div>
          </details>

          {entry.requestHeaders && Object.keys(entry.requestHeaders).length > 0 && (
            <details>
              <summary className="text-[var(--fg-muted)] cursor-pointer select-none font-medium mb-1">Request Headers</summary>
              <div className="ml-2 space-y-0.5 text-[var(--fg-secondary)]">
                {Object.entries(entry.requestHeaders).map(([k, v]) => (
                  <div key={k}><span className="text-[var(--fg)]">{k}</span>: {v}</div>
                ))}
              </div>
            </details>
          )}

          {entry.responseHeaders && Object.keys(entry.responseHeaders).length > 0 && (
            <details>
              <summary className="text-[var(--fg-muted)] cursor-pointer select-none font-medium mb-1">Response Headers</summary>
              <div className="ml-2 space-y-0.5 text-[var(--fg-secondary)]">
                {Object.entries(entry.responseHeaders).map(([k, v]) => (
                  <div key={k}><span className="text-[var(--fg)]">{k}</span>: {Array.isArray(v) ? v.join(", ") : v}</div>
                ))}
              </div>
            </details>
          )}

          {entry.responseBody && (
            <details>
              <summary className="text-[var(--fg-muted)] cursor-pointer select-none font-medium mb-1">Response Body</summary>
              <pre className="ml-2 text-[var(--fg-secondary)] whitespace-pre-wrap break-all max-h-[200px] overflow-auto">
                {(() => {
                  try { return JSON.stringify(JSON.parse(entry.responseBody), null, 2) } catch { return entry.responseBody }
                })()}
              </pre>
            </details>
          )}

          {entry.error && (
            <div className="text-[var(--danger)]">Error: {entry.error}</div>
          )}
        </div>
      )}
    </div>
  )
}

export function BottomBar() {
  const {
    layoutDirection,
    setLayoutDirection,
    sidebarCollapsed,
    toggleSidebar,
    consoleOpen,
    toggleConsole,
    consoleLogs,
    clearConsoleLogs,
    consoleHeight,
    setConsoleHeight,
  } = useUIStore()
  const { cookiePanelOpen, toggleCookiePanel, cookies } = useCookieStore()
  const errorCount = consoleLogs.filter((l) => l.error).length
  const scrollRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef(false)

  useEffect(() => {
    if (consoleOpen && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [consoleLogs.length, consoleOpen])

  const handleDragStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    draggingRef.current = true
    const startY = e.clientY
    const startHeight = consoleHeight

    const onMove = (ev: MouseEvent) => {
      if (!draggingRef.current) return
      const delta = startY - ev.clientY
      setConsoleHeight(startHeight + delta)
    }

    const onUp = () => {
      draggingRef.current = false
      document.removeEventListener("mousemove", onMove)
      document.removeEventListener("mouseup", onUp)
      document.body.style.cursor = ""
      document.body.style.userSelect = ""
    }

    document.body.style.cursor = "row-resize"
    document.body.style.userSelect = "none"
    document.addEventListener("mousemove", onMove)
    document.addEventListener("mouseup", onUp)
  }, [consoleHeight, setConsoleHeight])

  return (
    <div className="flex flex-col flex-shrink-0">
      <CookiePanel />
      {consoleOpen && (
        <div className={cn("flex flex-col", "border border-[var(--border-color)] bg-[var(--surface)]")} style={{ height: consoleHeight }}>
          {/* 拖拽手柄 */}
          <div
            className="group h-px flex-shrink-0 relative"
          >
            <div
              className="absolute inset-x-0 -top-2 h-[5px] cursor-row-resize z-10"
              onMouseDown={handleDragStart}
            />
            <div className="absolute inset-x-0 top-0 h-px bg-transparent group-hover:bg-[var(--accent)] transition-colors" />
            <div className="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-8 h-1 rounded-full bg-transparent group-hover:bg-[var(--accent)]/40 transition-all" />
          </div>
          <div className="flex items-center justify-between h-[25px] px-3 border-b border-[var(--border-color)] bg-[var(--surface-secondary)] flex-shrink-0">
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-medium text-[var(--fg)]">Console</span>
              {consoleLogs.length > 0 && (
                <span className="text-[10px] text-[var(--fg-muted)]">{consoleLogs.length}</span>
              )}
            </div>
            <div className="flex items-center gap-1">
              <button
                className="h-5 px-1.5 flex items-center gap-1 rounded-[4px] text-[10px] text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)] transition-colors"
                onClick={clearConsoleLogs}
                title="清空"
              >
                Clear
              </button>
              <button
                className="h-5 w-5 flex items-center justify-center rounded-[4px] hover:bg-[var(--sidebar-hover)] transition-colors"
                onClick={toggleConsole}
                title="关闭"
              >
                <AppIcon name="clear" size={10} className="text-[var(--fg-muted)]" />
              </button>
            </div>
          </div>
          <div ref={scrollRef} className="flex-1 overflow-y-auto">
            {consoleLogs.length === 0 ? (
              <div className="text-center py-8 text-[11px] text-[var(--fg-muted)]">暂无请求日志</div>
            ) : (
              consoleLogs.map((entry) => <ConsoleRow key={entry.id} entry={entry} />)
            )}
          </div>
        </div>
      )}

      <div
        className={cn(
          "flex h-[25px] items-center justify-between border-t px-3",
          "border-[var(--border-subtle)] bg-[var(--surface-secondary)]"
        )}
      >
        <div className="flex items-center gap-3 text-[11px] text-[var(--fg-muted)]">
          <button
            className={cn("flex items-center gap-1 transition-colors hover:text-[var(--fg-secondary)]", consoleOpen && "text-[var(--fg)]")}
            onClick={toggleConsole}
            type="button"
          >
            <AppIcon name="commandLine" size={11} strokeWidth={1.8} />
            Console
            {errorCount > 0 && (
              <span className="ml-0.5 px-1 py-0 rounded-full bg-[var(--danger)] text-white text-[9px] font-bold leading-[14px]">
                {errorCount}
              </span>
            )}
          </button>
          <button
            className={cn("flex items-center gap-1 transition-colors hover:text-[var(--fg-secondary)]", cookiePanelOpen && "text-[var(--fg)]")}
            onClick={toggleCookiePanel}
            type="button"
          >
            <AppIcon name="cookie" size={11} strokeWidth={1.8} />
            Cookies
            {cookies.length > 0 && (
              <span className="ml-0.5 text-[10px]">({cookies.length})</span>
            )}
          </button>
        </div>

        <div className="flex items-center gap-1">
          <button
            className={cn(
              "flex h-5 w-5 items-center justify-center rounded-[6px] transition-colors hover:bg-[var(--button-bg)]",
              layoutDirection === "vertical" ? "text-[var(--fg)] bg-[var(--button-bg)]" : "text-[var(--fg-secondary)]"
            )}
            onClick={() => setLayoutDirection("vertical")}
            title="上下布局"
            type="button"
          >
            <span className="relative h-[11px] w-[13px] rounded-[2px] border border-current">
              <span className="absolute left-[1px] right-[1px] top-1/2 h-px -translate-y-1/2 bg-current" />
            </span>
          </button>
          <button
            className={cn(
              "flex h-5 w-5 items-center justify-center rounded-[6px] transition-colors hover:bg-[var(--button-bg)]",
              layoutDirection === "horizontal" ? "text-[var(--fg)] bg-[var(--button-bg)]" : "text-[var(--fg-secondary)]"
            )}
            onClick={() => setLayoutDirection("horizontal")}
            title="左右布局"
            type="button"
          >
            <span className="relative h-[11px] w-[13px] rounded-[2px] border border-current">
              <span className="absolute bottom-[1px] left-1/2 top-[1px] w-px -translate-x-1/2 bg-current" />
            </span>
          </button>
          <button
            className="flex h-5 w-5 items-center justify-center rounded-[6px] text-[var(--fg-secondary)] transition-colors hover:bg-[var(--button-bg)] hover:text-[var(--fg)]"
            onClick={toggleSidebar}
            title={sidebarCollapsed ? "展开侧边栏" : "收起侧边栏"}
            type="button"
          >
            <AppIcon name={sidebarCollapsed ? "sidebarExpand" : "sidebarCollapse"} size={11} strokeWidth={1.9} />
          </button>
        </div>
      </div>
    </div>
  )
}
