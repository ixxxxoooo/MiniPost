import { useEffect, useMemo, useState } from "react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { AppIcon } from "@/components/ui/icon"
import { ResponseBody } from "./ResponseBody"
import { ResponseHeaders } from "./ResponseHeaders"
import { ResponseCookies } from "./ResponseCookies"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

function formatPhaseDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return "0 ms"
  if (ms < 10) return `${ms.toFixed(2)} ms`
  if (ms < 100) return `${ms.toFixed(1)} ms`
  return `${Math.round(ms)} ms`
}

function statusBadgeClass(code: number, isDark: boolean): string {
  if (isDark) {
    if (code >= 100 && code < 200) return "bg-[#2d2f34] text-[#aeb4bd]"
    if (code >= 200 && code < 300) return "bg-[#123a2c] text-[#7ce0ad]"
    if (code >= 300 && code < 400) return "bg-[#113342] text-[#72c7e4]"
    if (code >= 400 && code < 500) return "bg-[#3f2f16] text-[#f3bf67]"
    return "bg-[#402222] text-[#ff8a83]"
  }
  if (code >= 100 && code < 200) return "bg-[#f1f2f4] text-[#6f7782]"
  if (code >= 200 && code < 300) return "bg-[#e6f6ec] text-[#2b8a57]"
  if (code >= 300 && code < 400) return "bg-[#e6f3f8] text-[#2a7f95]"
  if (code >= 400 && code < 500) return "bg-[#fbf2e6] text-[#a56412]"
  return "bg-[#fbeceb] text-[#b44840]"
}

function statusDotClass(code: number): string {
  if (code >= 200 && code < 300) return "bg-[var(--success)]"
  if (code >= 300 && code < 400) return "bg-[var(--info)]"
  if (code >= 400 && code < 500) return "bg-[var(--warning)]"
  return "bg-[var(--danger)]"
}

function getStatusExplanation(code: number): string {
  const explicit: Record<number, string> = {
    200: "请求成功，服务器已返回预期结果。",
    201: "资源创建成功。",
    204: "请求成功，但响应体为空。",
    301: "资源已永久移动到新地址。",
    302: "资源临时重定向到其他地址。",
    304: "资源未修改，可使用缓存。",
    400: "请求参数或格式错误，服务器无法处理。",
    401: "未通过身份认证，请检查凭据。",
    403: "服务器理解请求，但拒绝访问。",
    404: "请求资源不存在。",
    405: "请求方法不被该资源允许。",
    408: "请求超时，服务器等待客户端过久。",
    409: "请求冲突，通常与资源状态有关。",
    413: "请求体过大，服务器拒绝处理。",
    415: "请求媒体类型不受支持。",
    422: "请求格式正确，但语义校验失败。",
    429: "请求过于频繁，触发限流。",
    500: "服务器内部错误。",
    501: "服务器不支持该请求能力。",
    502: "网关收到上游无效响应。",
    503: "服务暂时不可用。",
    504: "网关等待上游响应超时。",
  }
  if (explicit[code]) return explicit[code]
  if (code >= 200 && code < 300) return "请求已成功处理。"
  if (code >= 300 && code < 400) return "请求被重定向，请关注 Location 或重定向策略。"
  if (code >= 400 && code < 500) return "客户端请求存在问题，请检查 URL、参数、认证和方法。"
  return "服务器处理请求时发生错误，请检查服务端日志或网关链路。"
}

function parseResponseCookies(headers: Record<string, string[]>): Array<{ name: string; value: string; attributes: string }> {
  const setCookieValues: string[] = []
  Object.entries(headers).forEach(([key, values]) => {
    if (key.toLowerCase() === "set-cookie") {
      setCookieValues.push(...values)
    }
  })

  return setCookieValues
    .map((raw) => raw.trim())
    .filter(Boolean)
    .map((raw) => {
      const parts = raw.split(";").map((part) => part.trim()).filter(Boolean)
      const main = parts[0] ?? ""
      const separatorIndex = main.indexOf("=")
      const name = separatorIndex >= 0 ? main.slice(0, separatorIndex).trim() : main
      const value = separatorIndex >= 0 ? main.slice(separatorIndex + 1).trim() : ""
      const attributes = parts.slice(1).join("; ")
      return { name, value, attributes }
    })
}

type ParsedRequestError = {
  code: string
  message: string
  detail: string
}

type RequestErrorPresentation = {
  title: string
  badgeToneClass: string
  badgeText: string
  description: string
  suggestion: string
}

type TimingRow = {
  key: string
  label: string
  value: number
  start: number
  barClass: string
}

function parseRequestError(raw: string): ParsedRequestError {
  const trimmed = raw.trim()
  const match = trimmed.match(/^\[([A-Z0-9_]+)\]\s*([^:]+)(?::\s*(.*))?$/)
  if (!match) {
    return { code: "REQUEST_FAILED", message: "请求发送失败", detail: trimmed }
  }
  return {
    code: match[1] || "REQUEST_FAILED",
    message: (match[2] || "请求发送失败").trim(),
    detail: (match[3] || "").trim(),
  }
}

function getRequestErrorPresentation(parsed: ParsedRequestError): RequestErrorPresentation {
  const detail = parsed.detail || parsed.message
  const lower = detail.toLowerCase()

  if (parsed.code === "DNS_LOOKUP_FAILED" || lower.includes("no such host")) {
    return {
      title: "Could not send request",
      badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
      badgeText: `Error: ${detail}`,
      description: "无法解析目标域名，DNS 查询失败。",
      suggestion: "请检查域名拼写、网络 DNS 设置，或尝试切换网络后重试。",
    }
  }

  if (parsed.code === "CONNECTION_REFUSED" || lower.includes("connection refused") || lower.includes("econnrefused")) {
    return {
      title: "Could not send request",
      badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
      badgeText: `Error: ${detail}`,
      description: "连接被目标地址拒绝，服务可能未启动或端口不可达。",
      suggestion: "请检查目标服务状态、端口、代理或防火墙策略。",
    }
  }

  if (parsed.code === "REQUEST_TIMEOUT" || lower.includes("timeout") || lower.includes("deadline exceeded")) {
    return {
      title: "Could not send request",
      badgeToneClass: "bg-[#fbf2e6] text-[#a56412]",
      badgeText: `Error: ${detail}`,
      description: "请求在超时时间内未完成。",
      suggestion: "请增大请求超时、检查网络连通性，或确认服务是否响应较慢。",
    }
  }

  if (parsed.code === "TLS_HANDSHAKE_FAILED" || lower.includes("x509") || lower.includes("certificate") || lower.includes("tls")) {
    return {
      title: "Could not send request",
      badgeToneClass: "bg-[#fbf2e6] text-[#a56412]",
      badgeText: `Error: ${detail}`,
      description: "TLS 握手或证书校验失败。",
      suggestion: "请检查证书有效期、证书链配置，或在调试时临时关闭 SSL 校验。",
    }
  }

  if (parsed.code === "CONNECTION_CLOSED" || lower.includes("eof")) {
    return {
      title: "Could not send request",
      badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
      badgeText: `Error: ${detail}`,
      description: "连接在请求过程中被提前关闭。",
      suggestion: "这通常与网络链路、代理拦截或目标服务异常有关，请查看控制台日志定位。",
    }
  }

  return {
    title: "Could not send request",
    badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
    badgeText: `Error: ${detail}`,
    description: "请求未能成功发送。",
    suggestion: "请检查 URL、协议、代理、证书或超时设置后重试。",
  }
}

function LoadingTopShimmer() {
  return (
    <div className="absolute left-0 top-0 h-[3px] w-full overflow-hidden bg-[var(--border-subtle)]/60">
      <div className="response-loading-sheen h-full w-[34%]" />
    </div>
  )
}

export function ResponseViewer() {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const { isSending, resolved } = useUIStore()
  const [activeTabValue, setActiveTabValue] = useState("body")

  const isDark = resolved === "dark"
  const response = activeTab?.response ?? null
  const responseError = activeTab?.responseError ?? null
  const safeHeaders = response?.headers ?? {}
  const parsedError = responseError ? parseRequestError(responseError) : null
  const errorPresentation = parsedError ? getRequestErrorPresentation(parsedError) : null
  const canSendFromEmptyState = Boolean(activeTab?.request.url?.trim())

  const timingRows = useMemo<TimingRow[]>(() => {
    if (!response) return []
    const timings = response.timings
    const phaseRows: Array<Omit<TimingRow, "start">> = [
      { key: "prepare", label: "Prepare", value: timings?.prepare ?? 0, barClass: "bg-[#b7bdc6]" },
      { key: "socketInitialization", label: "Socket Initialization", value: timings?.socketInitialization ?? 0, barClass: "bg-[#f2be42]" },
      { key: "dnsLookup", label: "DNS Lookup", value: timings?.dnsLookup ?? 0, barClass: "bg-[#f2be42]" },
      { key: "tcpHandshake", label: "TCP Handshake", value: timings?.tcpHandshake ?? 0, barClass: "bg-[#4c89e3]" },
      { key: "sslHandshake", label: "SSL Handshake", value: timings?.sslHandshake ?? 0, barClass: "bg-[#3d78cf]" },
      { key: "waitingTTFB", label: "Waiting (TTFB)", value: timings?.waitingTTFB ?? response.duration, barClass: "bg-[#ef8f67]" },
      { key: "download", label: "Download", value: timings?.download ?? 0, barClass: "bg-[#5ca379]" },
      { key: "process", label: "Process", value: timings?.process ?? 0, barClass: "bg-[#a5adb8]" },
    ]
    let cursor = 0
    return phaseRows.map((row) => {
      const normalizedValue = Math.max(0, row.value)
      const next = { ...row, value: normalizedValue, start: cursor }
      cursor += normalizedValue
      return next
    })
  }, [response])

  const timingTotal = useMemo(() => {
    if (!response) return 1
    if (response.timings && response.timings.total > 0) return response.timings.total
    const sum = timingRows.reduce((acc, row) => acc + row.value, 0)
    if (sum > 0) return sum
    return Math.max(1, response.duration)
  }, [response, timingRows])

  const timingVisualRows = useMemo(() => {
    if (timingRows.length === 0) return []
    const minVisiblePercent = 1.2
    const rowsWithDisplay = timingRows.map((row) => {
      const rawPercent = row.value > 0 ? (row.value / timingTotal) * 100 : 0
      const displayPercent = row.value > 0 ? Math.max(minVisiblePercent, rawPercent) : 0
      return {
        ...row,
        displayPercent,
      }
    })
    const totalDisplay = rowsWithDisplay.reduce((acc, row) => acc + row.displayPercent, 0)
    const scale = totalDisplay > 100 ? 100 / totalDisplay : 1
    let cursor = 0
    return rowsWithDisplay.map((row) => {
      const width = row.displayPercent * scale
      const visualStart = cursor
      cursor += width
      return {
        ...row,
        visualStart,
        visualWidth: width,
      }
    })
  }, [timingRows, timingTotal])

  const sizeDetails = useMemo(() => {
    if (!response) return null
    const details = response.sizeDetails
    const responseBody = details?.responseBody ?? response.size
    const responseHeaders = details?.responseHeaders ?? Math.max(0, (details?.responseTotal ?? response.size) - responseBody)
    const responseTotal = details?.responseTotal ?? (responseHeaders + responseBody)
    const requestHeaders = details?.requestHeaders ?? 0
    const requestBody = details?.requestBody ?? 0
    const requestTotal = details?.requestTotal ?? (requestHeaders + requestBody)

    return {
      responseHeaders,
      responseBody,
      responseTotal,
      requestHeaders,
      requestBody,
      requestTotal,
    }
  }, [response])

  useEffect(() => {
    if (responseError) {
      setActiveTabValue("body")
    }
  }, [responseError])

  const triggerSendFromEmptyState = () => {
    if (!canSendFromEmptyState || isSending) return
    window.dispatchEvent(new CustomEvent("minipost:send"))
  }

  if (!response && !responseError && !isSending) {
    return (
      <div className="flex h-full flex-col bg-[var(--surface)]">
        <div className="flex h-[32px] items-center px-[var(--size-padding-sm)]">
          <span className="text-[13px] font-semibold text-[var(--fg)]">Response</span>
        </div>

        <div className="flex flex-1 items-center justify-center px-4">
          <button
            type="button"
            onClick={triggerSendFromEmptyState}
            disabled={!canSendFromEmptyState}
            className={cn(
              "inline-flex items-center gap-2 rounded-[8px] px-4 py-2 text-[13px] transition-colors",
              canSendFromEmptyState
                ? "text-[var(--fg-secondary)] hover:text-[var(--fg)]"
                : "text-[var(--fg-muted)] opacity-70 cursor-not-allowed"
            )}
            style={{ backgroundColor: resolved === "dark" ? "rgb(52,52,52)" : "rgb(249,249,249)" }}
          >
            <AppIcon name="clock" size={15} />
            <span>Send + Get a successful response</span>
            <span className="ml-1 inline-flex items-center gap-1 text-[10px] text-[var(--fg-muted)]">
              <kbd className="h-4 min-w-4 rounded-[4px] border border-[var(--button-border)] bg-[var(--surface)] px-1 font-mono">⌘</kbd>
              <kbd className="h-4 min-w-4 rounded-[4px] border border-[var(--button-border)] bg-[var(--surface)] px-1 font-mono">Enter</kbd>
            </span>
          </button>
        </div>
      </div>
    )
  }

  if (!response && !responseError && isSending) {
    return (
      <div className="relative h-full bg-[var(--surface)]">
        <div className="flex h-[32px] items-center px-[var(--size-padding-sm)]">
          <span className="text-[13px] font-semibold text-[var(--fg)]">Response</span>
        </div>
        <div className="flex h-[calc(100%-32px)] items-center justify-center px-4">
          <span className="text-[13px] text-[var(--fg-secondary)]">发送请求...</span>
        </div>
      </div>
    )
  }

  const headerCount = isSending ? 0 : Object.keys(safeHeaders).length
  const cookies = parseResponseCookies(safeHeaders)
  const cookieCount = isSending ? 0 : cookies.length

  return (
    <div className="flex h-full flex-col bg-[var(--surface)] relative">
      {isSending && (
        <div className="absolute inset-x-0 top-0 z-20 pointer-events-none">
          <LoadingTopShimmer />
        </div>
      )}

      {/* 发送中的叠加层 */}
      {isSending && (
        <div className="absolute inset-x-0 bottom-0 top-[32px] z-10 pointer-events-auto bg-[var(--response-loading-overlay)]" />
      )}

      <Tabs value={activeTabValue} onValueChange={setActiveTabValue} className="flex-1 flex flex-col min-h-0">
        <div className="flex items-center h-[32px] px-[var(--size-padding-sm)]">
          <TabsList className="flex-1 justify-start">
            <TabsTrigger value="body">Body</TabsTrigger>
            <TabsTrigger value="headers">
              Headers
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">
                ({headerCount})
              </span>
            </TabsTrigger>
            <TabsTrigger value="cookies">
              Cookies
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">
                ({cookieCount})
              </span>
            </TabsTrigger>
          </TabsList>

          {!isSending && response && !responseError && (
            <div className="ml-auto flex items-center gap-2 text-[11px] text-[var(--fg-muted)]">
              {response.warnings && response.warnings.length > 0 && (
                <>
                  <span className="inline-flex h-5 items-center rounded-[7px] bg-[#fbf2e6] px-1.5 text-[10px] font-medium text-[#a56412]">
                    Warning: {response.warnings[0]}
                  </span>
                  <span className="text-[var(--fg-muted)]">•</span>
                </>
              )}
              <div className="relative group/status">
                <span className={cn(
                  "inline-flex h-5 items-center rounded-[7px] px-1.5 text-[10px] font-semibold [font-variant-numeric:tabular-nums]",
                  statusBadgeClass(response.statusCode, isDark)
                )}>
                  {response.statusCode} {response.statusText}
                </span>
                <div
                  className={cn(
                    "pointer-events-none absolute right-0 top-[calc(100%+8px)] z-20 w-[340px] rounded-[10px] border shadow-[var(--shadow-lg)]",
                    "bg-[var(--surface-elevated)] border-[var(--border-color)] p-3 opacity-0 translate-y-1",
                    "transition-all duration-150 group-hover/status:opacity-100 group-hover/status:translate-y-0"
                  )}
                >
                  <div className="flex items-start gap-2">
                    <div className={cn("mt-[5px] h-2 w-2 rounded-full flex-shrink-0", statusDotClass(response.statusCode))} />
                    <div className="min-w-0">
                      <div className="text-[13px] font-semibold text-[var(--fg)] leading-5">
                        {response.statusCode} {response.statusText}
                      </div>
                      <div className="mt-1 text-[12px] text-[var(--fg-secondary)] leading-5">
                        {getStatusExplanation(response.statusCode)}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <span className="text-[var(--fg-muted)]">•</span>
              <div className="relative group/time">
                <span className="text-[var(--fg-muted)] [font-variant-numeric:tabular-nums] cursor-default">
                  {formatDuration(response.duration)}
                </span>
                <div
                  className={cn(
                    "pointer-events-none absolute right-0 top-[calc(100%+8px)] z-20 w-[min(460px,calc(100vw-20px))] rounded-[10px] border shadow-[var(--shadow-lg)]",
                    "max-h-[calc(100vh-140px)] overflow-auto bg-[var(--surface)] border-[var(--border-color)] px-3 py-2 opacity-0 translate-y-1",
                    "transition-all duration-150 group-hover/time:opacity-100 group-hover/time:translate-y-0"
                  )}
                >
                  <div className="mb-1 flex items-center justify-between">
                    <div className="flex items-center gap-2 text-[13px] font-semibold text-[var(--fg)]">
                      <AppIcon name="clock" size={14} className="text-[var(--fg-muted)]" />
                      <span>Response Time</span>
                    </div>
                    <div className="text-[12px] font-semibold text-[var(--fg)] [font-variant-numeric:tabular-nums]">
                      {formatDuration(response.duration)}
                    </div>
                  </div>
                  <div className="space-y-0.5">
                    {timingVisualRows.map((row) => {
                      const startPercent = Math.max(0, Math.min(100, row.visualStart))
                      const clampedWidth = Math.max(0, Math.min(row.visualWidth, 100 - startPercent))
                      const isTTFB = row.key === "waitingTTFB"
                      return (
                        <div key={row.key} className="grid grid-cols-[120px_minmax(140px,1fr)_68px] items-center gap-1.5">
                          <span className="text-[10px] text-[var(--fg-secondary)] truncate">{row.label}</span>
                          <div className="relative h-[20px]">
                            <div className="absolute left-0 top-0 bottom-0 w-px bg-[var(--border-color)]" />
                            <div className="absolute right-0 top-0 bottom-0 w-px bg-[var(--border-color)]" />
                            {clampedWidth > 0 && (
                              isTTFB ? (
                                <div
                                  className="absolute top-1/2 -translate-y-1/2 h-[12px] border border-dashed border-[#e67f71] bg-[#fde9e5]/70"
                                  style={{ left: `${startPercent}%`, width: `${clampedWidth}%` }}
                                />
                              ) : (
                                <div
                                  className={cn("absolute top-1/2 -translate-y-1/2 h-[12px]", row.barClass)}
                                  style={{ left: `${startPercent}%`, width: `${clampedWidth}%` }}
                                />
                              )
                            )}
                          </div>
                          <span className="text-right text-[10px] text-[var(--fg-muted)] [font-variant-numeric:tabular-nums]">
                            {formatPhaseDuration(row.value)}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              </div>
              <span className="text-[var(--fg-muted)]">•</span>
              <div className="relative group/size">
                <span className="text-[var(--fg-muted)] [font-variant-numeric:tabular-nums] cursor-default">
                  {formatSize(response.size)}
                </span>
                <div
                  className={cn(
                    "pointer-events-none absolute right-0 top-[calc(100%+8px)] z-20 w-[min(280px,calc(100vw-24px))] rounded-[10px] border shadow-[var(--shadow-lg)]",
                    "bg-[var(--surface)] border-[var(--border-color)] px-3 py-2.5 opacity-0 translate-y-1",
                    "transition-all duration-150 group-hover/size:opacity-100 group-hover/size:translate-y-0"
                  )}
                >
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="inline-flex items-center gap-2 text-[13px] font-semibold text-[var(--fg)]">
                        <AppIcon name="download" size={14} className="text-[#2f6fd3]" />
                        <span>Response Size</span>
                      </span>
                      <span className="text-[13px] font-semibold text-[var(--fg)] [font-variant-numeric:tabular-nums]">
                        {formatSize(sizeDetails?.responseTotal ?? response.size)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>Headers</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.responseHeaders ?? 0)}</span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>Body</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.responseBody ?? response.size)}</span>
                    </div>
                  </div>
                  <div className="my-2 h-px bg-[var(--border-subtle)]" />
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="inline-flex items-center gap-2 text-[13px] font-semibold text-[var(--fg)]">
                        <AppIcon name="upload" size={14} className="text-[#a8821f]" />
                        <span>Request Size</span>
                      </span>
                      <span className="text-[13px] font-semibold text-[var(--fg)] [font-variant-numeric:tabular-nums]">
                        {formatSize(sizeDetails?.requestTotal ?? 0)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>Headers</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.requestHeaders ?? 0)}</span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>Body</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.requestBody ?? 0)}</span>
                    </div>
                  </div>
                </div>
              </div>
              <span className="text-[var(--fg-muted)]">•</span>
              <span className="max-w-[180px] truncate font-mono text-[var(--fg-muted)]">{response.contentType}</span>
            </div>
          )}
        </div>

        <TabsContent value="body" className="flex-1 m-0 min-h-0 overflow-hidden">
          <div className="h-full p-[var(--size-padding-sm)]">
            <div className="h-full overflow-hidden">
              {responseError ? (
                <div className="h-full overflow-auto">
                  <div className="mx-auto flex h-full w-full max-w-[980px] items-center justify-center px-3">
                    <div className="w-full max-w-[860px] py-2">
                      <div className="flex items-start gap-2.5">
                        <div className="mt-0.5 flex h-6 w-6 items-center justify-center rounded-[999px] bg-[var(--danger)]/12">
                          <AppIcon name="info" size={11} className="text-[var(--danger)]" />
                        </div>
                        <div className="min-w-0 flex-1 space-y-2">
                          <div className="text-[14px] font-semibold text-[var(--fg)] leading-5">
                            无法发送请求
                          </div>
                          <div className="flex items-center gap-2 text-[12px] font-mono">
                            <span className={cn("font-semibold uppercase", METHOD_COLORS[(activeTab?.request.method as HttpMethod)] || "text-[var(--fg-muted)]")}>
                              {activeTab?.request.method || "GET"}
                            </span>
                            <span className="text-[var(--fg)] break-all">{activeTab?.request.url || ""}</span>
                          </div>

                          <div className={cn("inline-flex max-w-full items-center rounded-[8px] px-2.5 py-1.5 text-[12px] break-all", errorPresentation?.badgeToneClass || "bg-[#fbeceb] text-[#b44840]")}>
                            Error: {errorPresentation?.badgeText?.replace(/^Error:\s*/i, "") || responseError}
                          </div>

                          <div className="text-[12px] leading-6 text-[var(--fg-secondary)]">
                            <span>{errorPresentation?.description}</span>
                            <span className="ml-2">{errorPresentation?.suggestion}</span>
                          </div>

                          <div className="flex items-center gap-2 text-[11px] text-[var(--fg-muted)]">
                            <button
                              type="button"
                              className="inline-flex items-center gap-1.5 rounded-[7px] px-2 py-1 hover:bg-[var(--surface-secondary)] hover:text-[var(--fg-secondary)]"
                              onClick={() => useUIStore.setState({ consoleOpen: true })}
                            >
                              <AppIcon name="commandLine" size={12} />
                              View in console
                            </button>
                            <span>•</span>
                            <span>Code: {parsedError?.code || "REQUEST_FAILED"}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              ) : response ? (
                <ResponseBody body={response.body} contentType={response.contentType} isDark={isDark} />
              ) : (
                <div className="flex h-full items-center justify-center text-2xs text-[var(--fg-muted)]">
                  暂无响应内容
                </div>
              )}
            </div>
          </div>
        </TabsContent>

        <TabsContent value="headers" className="flex-1 m-0 min-h-0 overflow-hidden">
          <div className="h-full p-[var(--size-padding-sm)]">
            <ResponseHeaders headers={safeHeaders} />
          </div>
        </TabsContent>

        <TabsContent value="cookies" className="flex-1 m-0 min-h-0 overflow-hidden">
          <div className="h-full p-[var(--size-padding-sm)]">
            <ResponseCookies cookies={cookies} />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
