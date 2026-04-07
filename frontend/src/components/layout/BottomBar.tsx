import { useState, useRef, useEffect, useCallback } from "react"
import { cn } from "@/lib/utils"
import { AppIcon } from "@/components/ui/icon"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useI18n } from "@/hooks/useI18n"
import { useUIStore, type ConsoleEntry } from "@/stores/uiStore"
import { useCookieStore } from "@/stores/cookieStore"
import { CookiePanel } from "@/components/business/cookie/CookiePanel"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"

function formatTimestamp(ts: string, locale: string): string {
  const d = new Date(ts)
  return d.toLocaleTimeString(locale, { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit" })
}

function getStatusColor(code: number): string {
  if (code >= 200 && code < 300) return "text-[var(--success)]"
  if (code >= 400) return "text-[var(--danger)]"
  return "text-[var(--warning)]"
}

function parseAppError(raw: string, t: (zh: string, en: string) => string = (zh) => zh): { code: string; message: string; detail: string } {
  const text = raw.trim()
  const match = text.match(/^\[([A-Z0-9_]+)\]\s*([^:]+)(?::\s*(.*))?$/)
  if (!match) {
    return { code: "REQUEST_FAILED", message: t("请求发送失败", "Request failed"), detail: text }
  }
  return {
    code: match[1] || "REQUEST_FAILED",
    message: (match[2] || t("请求发送失败", "Request failed")).trim(),
    detail: (match[3] || "").trim(),
  }
}

function normalizeConsoleError(raw: string, t: (zh: string, en: string) => string = (zh) => zh): string {
  const parsed = parseAppError(raw, t)
  const lowerDetail = parsed.detail.toLowerCase()
  if (parsed.detail) return parsed.detail
  if (parsed.code === "DNS_LOOKUP_FAILED") return "getaddrinfo ENOTFOUND"
  if (parsed.code === "REQUEST_TIMEOUT") return "connect ETIMEDOUT"
  if (parsed.code === "CONNECTION_REFUSED") return "connect ECONNREFUSED"
  if (parsed.code === "TLS_HANDSHAKE_FAILED") return "TLS handshake failed"
  if (lowerDetail.includes("econnrefused")) return parsed.detail
  return parsed.message || raw
}

function getRequestPath(urlText: string): string {
  try {
    const parsed = new URL(urlText)
    return `${parsed.pathname || "/"}${parsed.search || ""}`
  } catch {
    return urlText
  }
}

function buildRawLog(entry: ConsoleEntry, t: (zh: string, en: string) => string): string {
  const requestProtocol = entry.requestProtocol || "HTTP/1.1"
  const requestLine = `${entry.method} ${getRequestPath(entry.url)} ${requestProtocol}`
  const requestHeaders = Object.entries(entry.requestHeaders ?? {})
    .map(([key, value]) => `${key}: ${value}`)
    .join("\n")
  const requestBody = entry.requestBody?.trim() ?? ""

  const blocks: string[] = []
  blocks.push(requestLine)

  if (requestProtocol.startsWith("HTTP/2")) {
    try {
      const parsed = new URL(entry.url)
      blocks.push(`:path: ${parsed.pathname || "/"}${parsed.search || ""}`)
      blocks.push(`:method: ${entry.method}`)
      blocks.push(`:authority: ${parsed.host}`)
      blocks.push(`:scheme: ${parsed.protocol.replace(":", "")}`)
    } catch {
      // ignore parse error for pseudo headers
    }
  }

  if (requestHeaders) blocks.push(requestHeaders)
  if (requestBody) blocks.push("", requestBody)

  if (entry.error) {
    blocks.push("", `Error: ${normalizeConsoleError(entry.error, t)}`)
    return blocks.join("\n")
  }

  const responseProtocol = entry.responseProtocol || "HTTP/1.1"
  const statusText = entry.statusText ? ` ${entry.statusText}` : ""
  const responseLine = `${responseProtocol} ${entry.status ?? ""}${statusText}`.trim()
  const responseHeaders = Object.entries(entry.responseHeaders ?? {})
    .map(([key, value]) => `${key}: ${Array.isArray(value) ? value.join(", ") : value}`)
    .join("\n")

  if (entry.warnings?.length) {
    blocks.unshift(`Warning: ${entry.warnings[0]}`)
  }

  blocks.push("", responseLine)
  if (responseHeaders) blocks.push(responseHeaders)
  if (entry.responseBody) blocks.push("", entry.responseBody)
  return blocks.join("\n")
}

type ConsoleTone = "neutral" | "warning" | "error"

function getConsoleTone(entry: ConsoleEntry): ConsoleTone {
  if (entry.error) {
    const code = parseAppError(entry.error).code
    if (code === "REQUEST_TIMEOUT" || code === "TLS_HANDSHAKE_FAILED") return "warning"
    return "error"
  }
  if (entry.warnings?.length) return "warning"
  return "neutral"
}

function ConsoleRow({ entry }: { entry: ConsoleEntry }) {
  const { t, locale } = useI18n()
  const [expanded, setExpanded] = useState(false)
  const [showRawLog, setShowRawLog] = useState(false)
  const hasError = !!entry.error
  const hasResponse = entry.status !== undefined
  const tone = getConsoleTone(entry)
  const rowToneClass = tone === "error"
    ? "bg-[#fbeceb]/75 hover:bg-[#f7e4e2]"
    : tone === "warning"
      ? "bg-[#fbf2e6]/75 hover:bg-[#f8eedf]"
      : "hover:bg-[var(--surface-secondary)]"
  const expandedToneClass = tone === "error"
    ? "bg-[#fbeceb]/45"
    : tone === "warning"
      ? "bg-[#fbf2e6]/45"
      : "bg-[var(--surface-secondary)]/50"
  const toggleExpanded = () => {
    const selectedText = window.getSelection()?.toString() ?? ""
    if (selectedText.trim().length > 0) return
    setExpanded(!expanded)
  }

  return (
    <div className="border-b border-[var(--border-color)]/50 last:border-b-0">
      <div
        className={cn(
          "flex items-center gap-2 px-3 py-1.5 text-[11px] font-mono transition-colors cursor-pointer",
          rowToneClass
        )}
        onClick={toggleExpanded}
      >
        <AppIcon name={expanded ? "arrowDown" : "arrowRight"} size={8} className="text-[var(--fg-muted)] flex-shrink-0" />
        <span className={cn(
          "font-bold w-[44px] flex-shrink-0 uppercase",
          METHOD_COLORS[(entry.method as HttpMethod)] || "text-[var(--fg-muted)]"
        )}>
          {entry.method}
        </span>
        <span className="text-[var(--fg)] truncate flex-1 select-text">{entry.url}</span>
        {hasError && <span className="text-[var(--danger)] flex-shrink-0 text-[10px]">Error</span>}
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
        <span className="text-[var(--fg-muted)] text-[10px] flex-shrink-0 ml-1">{formatTimestamp(entry.timestamp, locale)}</span>
      </div>

      {expanded && (
        <div className={cn("px-5 py-2 text-[10px] font-mono space-y-2 select-text", expandedToneClass)}>
          {hasError ? (
            <>
              <details open>
                <summary className="mb-1 cursor-pointer select-text text-[10px] text-[var(--fg-muted)] flex items-center justify-between gap-2">
                  <span>Network</span>
                  <button
                    className="text-[10px] text-[var(--accent)] hover:underline"
                    onClick={(event) => {
                      event.preventDefault()
                      event.stopPropagation()
                      setShowRawLog((prev) => !prev)
                    }}
                    type="button"
                  >
                    {showRawLog ? "Show pretty log" : "Show raw log"}
                  </button>
                </summary>
                {showRawLog ? (
                  <pre className="ml-2 whitespace-pre-wrap break-all text-[11px] leading-[1.35] text-[var(--fg-secondary)] select-text">
                    {buildRawLog(entry, t)}
                  </pre>
                ) : (
                  <div className="ml-2 space-y-0.5 text-[10px] text-[var(--fg-secondary)] select-text">
                    <div className="text-[var(--fg)] select-text">{entry.method} {entry.url}</div>
                    <div className="text-[var(--danger)]">Error: {normalizeConsoleError(entry.error || "", t)}</div>
                  </div>
                )}
              </details>
              {!showRawLog && entry.requestHeaders && Object.keys(entry.requestHeaders).length > 0 && (
                <details open>
                  <summary className="text-[var(--fg-muted)] cursor-pointer select-text text-[10px] mb-1">Request Headers</summary>
                  <div className="ml-2 space-y-0.5 text-[10px] text-[var(--fg-secondary)] select-text">
                    {Object.entries(entry.requestHeaders).map(([k, v]) => (
                      <div key={k} className="select-text">
                        <span className="text-[var(--fg)]">{k}</span>: "{v}"
                      </div>
                    ))}
                  </div>
                </details>
              )}
            </>
          ) : (
            <>
              <details open>
                <summary className="mb-1 cursor-pointer select-text text-[10px] text-[var(--fg-muted)] flex items-center justify-between gap-2">
                  <span>Network</span>
                  <button
                    className="text-[10px] text-[var(--accent)] hover:underline"
                    onClick={(event) => {
                      event.preventDefault()
                      event.stopPropagation()
                      setShowRawLog((prev) => !prev)
                    }}
                    type="button"
                  >
                    {showRawLog ? "Show pretty log" : "Show raw log"}
                  </button>
                </summary>
                {showRawLog ? (
                  <pre className="ml-2 whitespace-pre-wrap break-all text-[11px] leading-[1.35] text-[var(--fg-secondary)] select-text">
                    {buildRawLog(entry, t)}
                  </pre>
                ) : (
                  <div className="ml-2 space-y-0.5 text-[10px] text-[var(--fg-secondary)] select-text">
                    <div className="select-text">{entry.method} {entry.url}</div>
                    {entry.warnings?.length ? (
                      <div className="text-[var(--warning)]">Warning: {entry.warnings[0]}</div>
                    ) : null}
                    {entry.status !== undefined && (
                      <div>Status: <span className={getStatusColor(entry.status)}>{entry.status}</span></div>
                    )}
                    {entry.duration !== undefined && <div>Duration: {Math.round(entry.duration)}ms</div>}
                    {entry.size !== undefined && <div>Size: {entry.size < 1024 ? `${entry.size}B` : `${(entry.size / 1024).toFixed(1)}KB`}</div>}
                  </div>
                )}
              </details>

              {!showRawLog && entry.requestHeaders && Object.keys(entry.requestHeaders).length > 0 && (
                <details>
                  <summary className="text-[var(--fg-muted)] cursor-pointer select-text text-[10px] mb-1">Request Headers</summary>
                  <div className="ml-2 space-y-0.5 text-[10px] text-[var(--fg-secondary)] select-text">
                    {Object.entries(entry.requestHeaders).map(([k, v]) => (
                      <div key={k} className="select-text"><span className="text-[var(--fg)]">{k}</span>: {v}</div>
                    ))}
                  </div>
                </details>
              )}

              {!showRawLog && entry.responseHeaders && Object.keys(entry.responseHeaders).length > 0 && (
                <details>
                  <summary className="text-[var(--fg-muted)] cursor-pointer select-text text-[10px] mb-1">Response Headers</summary>
                  <div className="ml-2 space-y-0.5 text-[10px] text-[var(--fg-secondary)] select-text">
                    {Object.entries(entry.responseHeaders).map(([k, v]) => (
                      <div key={k} className="select-text"><span className="text-[var(--fg)]">{k}</span>: {Array.isArray(v) ? v.join(", ") : v}</div>
                    ))}
                  </div>
                </details>
              )}

              {!showRawLog && entry.responseBody && (
                <details>
                  <summary className="text-[var(--fg-muted)] cursor-pointer select-text text-[10px] mb-1">Response Body</summary>
                  <pre className="ml-2 text-[10px] text-[var(--fg-secondary)] whitespace-pre-wrap break-all max-h-[200px] overflow-auto select-text">
                    {(() => {
                      try { return JSON.stringify(JSON.parse(entry.responseBody), null, 2) } catch { return entry.responseBody }
                    })()}
                  </pre>
                </details>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

export function BottomBar() {
  const { t } = useI18n()
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
  const isVerticalLayout = layoutDirection === "vertical"
  const scrollRef = useRef<HTMLDivElement>(null)
  const consolePanelRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef(false)
  const dragHeightRef = useRef<number | null>(null)
  const dragRafRef = useRef<number | null>(null)
  const [dragPreviewHeight, setDragPreviewHeight] = useState<number | null>(null)

  useEffect(() => {
    if (consoleOpen && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [consoleLogs.length, consoleOpen])

  useEffect(() => {
    if (!consoleOpen) return

    const onKeyDown = (event: KeyboardEvent) => {
      if (!(event.metaKey || event.ctrlKey) || event.altKey) return
      if (event.key.toLowerCase() !== "a") return

      const panel = consolePanelRef.current
      if (!panel) return

      const target = event.target as Node | null
      const selection = window.getSelection()
      const anchor = selection?.anchorNode ?? null
      const focus = selection?.focusNode ?? null
      const insideByTarget = target ? panel.contains(target) : false
      const insideBySelection = (anchor ? panel.contains(anchor) : false) || (focus ? panel.contains(focus) : false)
      if (!insideByTarget && !insideBySelection) return

      event.preventDefault()
      event.stopPropagation()
    }

    window.addEventListener("keydown", onKeyDown, true)
    return () => window.removeEventListener("keydown", onKeyDown, true)
  }, [consoleOpen])

  const handleDragStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    draggingRef.current = true
    const startY = e.clientY
    const startHeight = consoleHeight
    dragHeightRef.current = startHeight

    const onMove = (ev: MouseEvent) => {
      if (!draggingRef.current) return
      const delta = startY - ev.clientY
      const next = Math.max(80, Math.min(600, startHeight + delta))
      dragHeightRef.current = next
      if (dragRafRef.current === null) {
        dragRafRef.current = window.requestAnimationFrame(() => {
          dragRafRef.current = null
          setDragPreviewHeight(dragHeightRef.current)
        })
      }
    }

    const onUp = () => {
      draggingRef.current = false
      if (dragRafRef.current !== null) {
        window.cancelAnimationFrame(dragRafRef.current)
        dragRafRef.current = null
      }
      const finalHeight = dragHeightRef.current ?? startHeight
      setConsoleHeight(finalHeight)
      setDragPreviewHeight(null)
      dragHeightRef.current = null
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
        <div ref={consolePanelRef} className={cn("flex flex-col", "border border-[var(--border-color)] bg-[var(--surface)]")} style={{ height: dragPreviewHeight ?? consoleHeight }}>
          {/* 拖拽手柄 */}
          <div
            className="group h-px flex-shrink-0 relative"
          >
            <div
              className="absolute left-0 right-0 top-1/2 h-[6px] -translate-y-1/2 cursor-row-resize z-10"
              onMouseDown={handleDragStart}
            />
            <div className="absolute inset-x-0 top-0 h-px bg-transparent group-hover:bg-[var(--accent)] transition-colors" />
            <div className="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-8 h-1 rounded-full bg-transparent group-hover:bg-[var(--accent)]/40 transition-all" />
          </div>
          <div
            className="flex items-center justify-between h-[25px] px-3 border-b border-[var(--border-color)] bg-[var(--surface-secondary)] flex-shrink-0 cursor-pointer"
            onClick={toggleConsole}
          >
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-medium text-[var(--fg)]">Console</span>
              {consoleLogs.length > 0 && (
                <span className="text-[10px] text-[var(--fg-muted)]">{consoleLogs.length}</span>
              )}
            </div>
            <div className="flex items-center gap-1">
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    className="h-5 px-1.5 flex items-center gap-1 rounded-[4px] text-[10px] text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)] transition-colors"
                    onClick={(event) => {
                      event.stopPropagation()
                      clearConsoleLogs()
                    }}
                  >
                    {t("清空", "Clear")}
                  </button>
                </TooltipTrigger>
                <TooltipContent>{t("清空", "Clear")}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    className="h-5 w-5 flex items-center justify-center rounded-[4px] hover:bg-[var(--sidebar-hover)] transition-colors"
                    onClick={(event) => {
                      event.stopPropagation()
                      toggleConsole()
                    }}
                  >
                    <AppIcon name="clear" size={10} className="text-[var(--fg-muted)]" />
                  </button>
                </TooltipTrigger>
                <TooltipContent>{t("关闭", "Close")}</TooltipContent>
              </Tooltip>
            </div>
          </div>
          <div ref={scrollRef} className="flex-1 overflow-y-auto">
            {consoleLogs.length === 0 ? (
              <div className="text-center py-8 text-[11px] text-[var(--fg-muted)]">{t("暂无请求日志", "No request logs yet")}</div>
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
            className="flex items-center gap-1 transition-colors hover:text-[var(--fg-secondary)]"
            onClick={toggleConsole}
            type="button"
          >
            <AppIcon name="commandLine" size={11} strokeWidth={1.8} />
            Console
            {errorCount > 0 && (
              <span className="ml-0.5 rounded-full border border-[var(--border-color)] bg-[var(--surface)] px-1 py-0 text-[9px] font-medium leading-[14px] text-[var(--fg-muted)]">
                {errorCount}
              </span>
            )}
          </button>
          <button
            className="flex items-center gap-1 transition-colors hover:text-[var(--fg-secondary)]"
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
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className={cn(
                  "flex h-5 w-5 items-center justify-center rounded-[6px] transition-colors hover:bg-[var(--button-bg)]",
                  "text-[var(--fg)] bg-[var(--button-bg)]"
                )}
                onClick={() => setLayoutDirection(isVerticalLayout ? "horizontal" : "vertical")}
                type="button"
              >
                <span className="relative h-[11px] w-[13px] rounded-[2px] border border-current">
                  {isVerticalLayout ? (
                    <span className="absolute bottom-[1px] left-1/2 top-[1px] w-px -translate-x-1/2 bg-current" />
                  ) : (
                    <span className="absolute left-[1px] right-[1px] top-1/2 h-px -translate-y-1/2 bg-current" />
                  )}
                </span>
              </button>
            </TooltipTrigger>
            <TooltipContent>
              {isVerticalLayout
                ? t("切换到左右布局", "Switch to horizontal layout")
                : t("切换到上下布局", "Switch to vertical layout")}
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className={cn(
                  "flex h-5 w-5 items-center justify-center rounded-[6px] transition-colors hover:bg-[var(--button-bg)]",
                  "text-[var(--fg-secondary)] hover:text-[var(--fg)]"
                )}
                onClick={toggleSidebar}
                type="button"
              >
                <span className="relative h-[11px] w-[13px] rounded-[2px] border border-current">
                  {sidebarCollapsed ? (
                    <span className="absolute bottom-[1px] top-[1px] right-[1px] w-[3px] rounded-[1px] bg-current/80" />
                  ) : (
                    <span className="absolute bottom-[1px] top-[1px] left-[1px] w-[3px] rounded-[1px] bg-current/80" />
                  )}
                </span>
              </button>
            </TooltipTrigger>
            <TooltipContent>{sidebarCollapsed ? t("展开侧边栏", "Expand sidebar") : t("收起侧边栏", "Collapse sidebar")}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  )
}
