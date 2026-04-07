import { useEffect, useMemo, useState } from "react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { AppIcon } from "@/components/ui/icon"
import { ResponseBody } from "./ResponseBody"
import { ResponseStream } from "./ResponseStream"
import { ResponseHeaders } from "./ResponseHeaders"
import { ResponseCookies } from "./ResponseCookies"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { useI18n } from "@/hooks/useI18n"
import { ensureRequestProtocol, resolveTemplateVariables } from "@/lib/variableResolver"

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

function formatNetworkValidUntil(value?: string): string {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  const month = date.toLocaleString("en-US", { month: "short", timeZone: "UTC" })
  const day = date.getUTCDate()
  const hh = String(date.getUTCHours()).padStart(2, "0")
  const mm = String(date.getUTCMinutes()).padStart(2, "0")
  const ss = String(date.getUTCSeconds()).padStart(2, "0")
  const year = date.getUTCFullYear()
  return `${month} ${day} ${hh}:${mm}:${ss} ${year} GMT`
}

function displayNetworkValue(value?: string): string {
  if (!value || !value.trim()) return "-"
  return value
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

function getStatusExplanation(code: number, isZh: boolean): string {
  const t = (zh: string, en: string) => (isZh ? zh : en)
  const explicit: Record<number, string> = {
    200: t("请求成功，服务器已返回预期结果。", "Request succeeded and the server returned the expected result."),
    201: t("资源创建成功。", "Resource created successfully."),
    204: t("请求成功，但响应体为空。", "Request succeeded, but the response body is empty."),
    301: t("资源已永久移动到新地址。", "Resource has moved permanently to a new location."),
    302: t("资源临时重定向到其他地址。", "Resource is temporarily redirected to another location."),
    304: t("资源未修改，可使用缓存。", "Resource not modified; cached content can be used."),
    400: t("请求参数或格式错误，服务器无法处理。", "Invalid request params or format; server cannot process it."),
    401: t("未通过身份认证，请检查凭据。", "Authentication failed. Check your credentials."),
    403: t("服务器理解请求，但拒绝访问。", "Server understood the request but refuses access."),
    404: t("请求资源不存在。", "Requested resource was not found."),
    405: t("请求方法不被该资源允许。", "Request method is not allowed for this resource."),
    408: t("请求超时，服务器等待客户端过久。", "Request timed out; server waited too long for the client."),
    409: t("请求冲突，通常与资源状态有关。", "Request conflict, usually related to resource state."),
    413: t("请求体过大，服务器拒绝处理。", "Request body is too large; server rejected it."),
    415: t("请求媒体类型不受支持。", "Unsupported request media type."),
    422: t("请求格式正确，但语义校验失败。", "Request format is valid but semantic validation failed."),
    429: t("请求过于频繁，触发限流。", "Too many requests; rate limit triggered."),
    500: t("服务器内部错误。", "Internal server error."),
    501: t("服务器不支持该请求能力。", "Server does not support this request capability."),
    502: t("网关收到上游无效响应。", "Gateway received an invalid upstream response."),
    503: t("服务暂时不可用。", "Service is temporarily unavailable."),
    504: t("网关等待上游响应超时。", "Gateway timed out waiting for upstream response."),
  }
  if (explicit[code]) return explicit[code]
  if (code >= 200 && code < 300) return t("请求已成功处理。", "Request was handled successfully.")
  if (code >= 300 && code < 400) return t("请求被重定向，请关注 Location 或重定向策略。", "Request was redirected. Check Location or redirect policy.")
  if (code >= 400 && code < 500) return t("客户端请求存在问题，请检查 URL、参数、认证和方法。", "Client request issue. Check URL, params, authentication, and method.")
  return t("服务器处理请求时发生错误，请检查服务端日志或网关链路。", "Server error while processing request. Check server logs or gateway path.")
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

function parseRequestError(raw: string, isZh: boolean): ParsedRequestError {
  const t = (zh: string, en: string) => (isZh ? zh : en)
  const trimmed = raw.trim()
  const match = trimmed.match(/^\[([A-Z0-9_]+)\]\s*([^:]+)(?::\s*(.*))?$/)
  if (!match) {
    return { code: "REQUEST_FAILED", message: t("请求发送失败", "Request failed"), detail: trimmed }
  }
  return {
    code: match[1] || "REQUEST_FAILED",
    message: (match[2] || t("请求发送失败", "Request failed")).trim(),
    detail: (match[3] || "").trim(),
  }
}

function getRequestErrorPresentation(parsed: ParsedRequestError, isZh: boolean): RequestErrorPresentation {
  const t = (zh: string, en: string) => (isZh ? zh : en)
  const detail = parsed.detail || parsed.message
  const lower = detail.toLowerCase()

  if (parsed.code === "DNS_LOOKUP_FAILED" || lower.includes("no such host")) {
    return {
      title: t("无法发送请求", "Could not send request"),
      badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
      badgeText: `Error: ${detail}`,
      description: t("无法解析目标域名，DNS 查询失败。", "Failed to resolve target hostname (DNS lookup failed)."),
      suggestion: t("请检查域名拼写、网络 DNS 设置，或尝试切换网络后重试。", "Check domain spelling and DNS settings, or switch network and retry."),
    }
  }

  if (parsed.code === "CONNECTION_REFUSED" || lower.includes("connection refused") || lower.includes("econnrefused")) {
    return {
      title: t("无法发送请求", "Could not send request"),
      badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
      badgeText: `Error: ${detail}`,
      description: t("连接被目标地址拒绝，服务可能未启动或端口不可达。", "Connection was refused by target. Service may be down or port unreachable."),
      suggestion: t("请检查目标服务状态、端口、代理或防火墙策略。", "Check service status, port, proxy, or firewall policy."),
    }
  }

  if (parsed.code === "REQUEST_TIMEOUT" || lower.includes("timeout") || lower.includes("deadline exceeded")) {
    return {
      title: t("无法发送请求", "Could not send request"),
      badgeToneClass: "bg-[#fbf2e6] text-[#a56412]",
      badgeText: `Error: ${detail}`,
      description: t("请求在超时时间内未完成。", "Request did not complete within timeout."),
      suggestion: t("请增大请求超时、检查网络连通性，或确认服务是否响应较慢。", "Increase timeout, check network connectivity, or verify whether the service responds slowly."),
    }
  }

  if (parsed.code === "TLS_HANDSHAKE_FAILED" || lower.includes("x509") || lower.includes("certificate") || lower.includes("tls")) {
    return {
      title: t("无法发送请求", "Could not send request"),
      badgeToneClass: "bg-[#fbf2e6] text-[#a56412]",
      badgeText: `Error: ${detail}`,
      description: t("TLS 握手或证书校验失败。", "TLS handshake or certificate validation failed."),
      suggestion: t("请检查证书有效期、证书链配置，或在调试时临时关闭 SSL 校验。", "Check certificate validity and chain configuration, or temporarily disable SSL verification for debugging."),
    }
  }

  if (parsed.code === "CONNECTION_CLOSED" || lower.includes("eof")) {
    return {
      title: t("无法发送请求", "Could not send request"),
      badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
      badgeText: `Error: ${detail}`,
      description: t("连接在请求过程中被提前关闭。", "Connection was closed before the request finished."),
      suggestion: t("这通常与网络链路、代理拦截或目标服务异常有关，请查看控制台日志定位。", "Usually related to network path, proxy interception, or target service issues. Check console logs."),
    }
  }

  return {
    title: t("无法发送请求", "Could not send request"),
    badgeToneClass: "bg-[#fbeceb] text-[#b44840]",
    badgeText: `Error: ${detail}`,
    description: t("请求未能成功发送。", "The request could not be sent."),
    suggestion: t("请检查 URL、协议、代理、证书或超时设置后重试。", "Check URL, protocol, proxy, certificate, or timeout settings and retry."),
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
  const { t, isZh } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const { isSending, resolved } = useUIStore()
  const environments = useEnvironmentStore((s) => s.environments)
  const activeEnvironmentId = useEnvironmentStore((s) => s.activeEnvironmentId)
  const [activeTabValue, setActiveTabValue] = useState("body")
  const [liveNow, setLiveNow] = useState(() => Date.now())

  const isDark = resolved === "dark"
  const response = activeTab?.response ?? null
  const responseError = activeTab?.responseError ?? null
  const streamEntries = activeTab?.streamEntries ?? []
  const streamActive = activeTab?.streamActive ?? false
  const hasStreamEntries = streamEntries.length > 0
  const safeHeaders = response?.headers ?? {}
  const parsedError = responseError ? parseRequestError(responseError, isZh) : null
  const errorPresentation = parsedError ? getRequestErrorPresentation(parsedError, isZh) : null
  const canSendFromEmptyState = Boolean(activeTab?.request.url?.trim())
  const rawRequestUrl = activeTab?.request.url ?? ""
  const activeVariables = useMemo(() => {
    if (!activeEnvironmentId) return []
    const env = environments.find((item) => item.id === activeEnvironmentId)
    if (!env) return []
    return (env.variables ?? [])
      .filter((variable) => variable.enabled && variable.key)
      .map((variable) => ({ key: variable.key, value: variable.value }))
  }, [activeEnvironmentId, environments])
  const resolvedRequestUrl = useMemo(() => {
    if (!rawRequestUrl.trim()) return ""
    const withVariables = resolveTemplateVariables(rawRequestUrl, activeVariables)
    return ensureRequestProtocol(withVariables || rawRequestUrl)
  }, [activeVariables, rawRequestUrl])

  const timingRows = useMemo<TimingRow[]>(() => {
    if (!response) return []
    const timings = response.timings
    const phaseRows: Array<Omit<TimingRow, "start">> = [
      { key: "prepare", label: t("准备", "Prepare"), value: timings?.prepare ?? 0, barClass: "bg-[#b7bdc6]" },
      { key: "socketInitialization", label: t("Socket 初始化", "Socket Initialization"), value: timings?.socketInitialization ?? 0, barClass: "bg-[#f2be42]" },
      { key: "dnsLookup", label: t("DNS 查询", "DNS Lookup"), value: timings?.dnsLookup ?? 0, barClass: "bg-[#f2be42]" },
      { key: "tcpHandshake", label: t("TCP 握手", "TCP Handshake"), value: timings?.tcpHandshake ?? 0, barClass: "bg-[#4c89e3]" },
      { key: "sslHandshake", label: t("SSL 握手", "SSL Handshake"), value: timings?.sslHandshake ?? 0, barClass: "bg-[#3d78cf]" },
      { key: "waitingTTFB", label: t("等待首字节 (TTFB)", "Waiting (TTFB)"), value: timings?.waitingTTFB ?? response.duration, barClass: "bg-[#ef8f67]" },
      { key: "download", label: t("下载", "Download"), value: timings?.download ?? 0, barClass: "bg-[#5ca379]" },
      { key: "process", label: t("处理", "Process"), value: timings?.process ?? 0, barClass: "bg-[#a5adb8]" },
    ]
    let cursor = 0
    return phaseRows.map((row) => {
      const normalizedValue = Math.max(0, row.value)
      const next = { ...row, value: normalizedValue, start: cursor }
      cursor += normalizedValue
      return next
    })
  }, [response, t])

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
    if (!isSending || !response || !(hasStreamEntries || streamActive)) return
    const timer = window.setInterval(() => setLiveNow(Date.now()), 120)
    return () => window.clearInterval(timer)
  }, [hasStreamEntries, isSending, response, streamActive])

  const streamStartTimestamp = useMemo(() => {
    const startEntry = streamEntries.find((entry) => entry.kind === "response_start")
    if (startEntry) return startEntry.timestamp
    return streamEntries[0]?.timestamp
  }, [streamEntries])

  const displayDuration = useMemo(() => {
    if (!response) return 0
    const isLiveStreaming = isSending && (hasStreamEntries || streamActive)
    if (!isLiveStreaming) return response.duration
    const startMs = streamStartTimestamp ? new Date(streamStartTimestamp).getTime() : NaN
    if (!Number.isFinite(startMs) || startMs <= 0) return response.duration
    return Math.max(response.duration, liveNow - startMs)
  }, [hasStreamEntries, isSending, liveNow, response, streamActive, streamStartTimestamp])

  const displaySize = useMemo(() => {
    if (!response) return 0
    const latestBytes = streamEntries.reduce<number | undefined>((acc, entry) => {
      if (typeof entry.bytesTotal !== "number" || !Number.isFinite(entry.bytesTotal)) return acc
      return Math.max(acc ?? 0, entry.bytesTotal)
    }, undefined)
    if (latestBytes === undefined) return response.size
    return Math.max(response.size, latestBytes)
  }, [response, streamEntries])

  const hasTLSNetwork = Boolean(
    response?.network?.tlsProtocol
    || response?.network?.cipherName
    || response?.network?.certificateCN
    || response?.network?.issuerCN
    || response?.network?.validUntil
  )

  useEffect(() => {
    if (responseError) {
      setActiveTabValue("body")
    }
  }, [responseError])

  const triggerSendFromEmptyState = () => {
    if (!canSendFromEmptyState || isSending) return
    window.dispatchEvent(new CustomEvent("minipost:send"))
  }

  if (!response && !responseError && !isSending && !hasStreamEntries) {
    return (
      <div className="flex h-full flex-col bg-[var(--surface)]">
        <div className="flex h-[32px] items-center px-[var(--size-padding-sm)]">
          <span className="text-[13px] font-semibold text-[var(--fg)]">{t("响应", "Response")}</span>
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
            <span>{t("发送并获取成功响应", "Send + Get a successful response")}</span>
            <span className="ml-1 inline-flex items-center gap-1 text-[10px] text-[var(--fg-muted)]">
              <kbd className="h-4 min-w-4 rounded-[4px] border border-[var(--button-border)] bg-[var(--surface)] px-1 font-mono">⌘</kbd>
              <kbd className="h-4 min-w-4 rounded-[4px] border border-[var(--button-border)] bg-[var(--surface)] px-1 font-mono">Enter</kbd>
            </span>
          </button>
        </div>
      </div>
    )
  }

  if (!response && !responseError && isSending && !hasStreamEntries) {
    return (
      <div className="relative h-full bg-[var(--surface)]">
        <div className="flex h-[32px] items-center px-[var(--size-padding-sm)]">
          <span className="text-[13px] font-semibold text-[var(--fg)]">{t("响应", "Response")}</span>
        </div>
        <div className="flex h-[calc(100%-32px)] items-center justify-center px-4">
          <span className="text-[13px] text-[var(--fg-secondary)]">{t("发送请求...", "Sending request...")}</span>
        </div>
      </div>
    )
  }

  const headerCount = Object.keys(safeHeaders).length
  const cookies = parseResponseCookies(safeHeaders)
  const cookieCount = cookies.length

  return (
    <div className="flex h-full flex-col bg-[var(--surface)] relative">
      {isSending && (
        <div className="absolute inset-x-0 top-0 z-20 pointer-events-none">
          <LoadingTopShimmer />
        </div>
      )}

      {/* sending overlay */}
      {isSending && !hasStreamEntries && (
        <div className="absolute inset-x-0 bottom-0 top-[32px] z-10 pointer-events-auto bg-[var(--response-loading-overlay)]" />
      )}

      <Tabs value={activeTabValue} onValueChange={setActiveTabValue} className="flex-1 flex flex-col min-h-0">
        <div className="flex items-center h-[32px] px-[var(--size-padding-sm)]">
          <TabsList className="flex-1 justify-start">
            <TabsTrigger value="body">{t("响应体", "Body")}</TabsTrigger>
            <TabsTrigger value="headers">
              {t("响应头", "Headers")}
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">
                ({headerCount})
              </span>
            </TabsTrigger>
            <TabsTrigger value="cookies">
              {t("Cookies", "Cookies")}
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">
                ({cookieCount})
              </span>
            </TabsTrigger>
          </TabsList>

          {response && !responseError && (
            <div className="ml-auto flex items-center gap-2 text-[11px] text-[var(--fg-muted)]">
              {response.warnings && response.warnings.length > 0 && (
                <>
                  <span className="inline-flex h-5 items-center rounded-[7px] bg-[#fbf2e6] px-1.5 text-[10px] font-medium text-[#a56412]">
                    {t("警告", "Warning")}: {response.warnings[0]}
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
                        {getStatusExplanation(response.statusCode, isZh)}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <span className="text-[var(--fg-muted)]">•</span>
              <div className="relative group/time">
                <span className="text-[var(--fg-muted)] [font-variant-numeric:tabular-nums] cursor-default">
                  {formatDuration(displayDuration)}
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
                      <span>{t("响应时间", "Response Time")}</span>
                    </div>
                    <div className="text-[12px] font-semibold text-[var(--fg)] [font-variant-numeric:tabular-nums]">
                      {formatDuration(displayDuration)}
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
                  {formatSize(displaySize)}
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
                        <span>{t("响应大小", "Response Size")}</span>
                      </span>
                      <span className="text-[13px] font-semibold text-[var(--fg)] [font-variant-numeric:tabular-nums]">
                        {formatSize(sizeDetails?.responseTotal ?? displaySize)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>{t("响应头", "Headers")}</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.responseHeaders ?? 0)}</span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>{t("响应体", "Body")}</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.responseBody ?? displaySize)}</span>
                    </div>
                  </div>
                  <div className="my-2 h-px bg-[var(--border-subtle)]" />
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                        <span className="inline-flex items-center gap-2 text-[13px] font-semibold text-[var(--fg)]">
                          <AppIcon name="upload" size={14} className="text-[#a8821f]" />
                        <span>{t("请求大小", "Request Size")}</span>
                      </span>
                      <span className="text-[13px] font-semibold text-[var(--fg)] [font-variant-numeric:tabular-nums]">
                        {formatSize(sizeDetails?.requestTotal ?? 0)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>{t("请求头", "Headers")}</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.requestHeaders ?? 0)}</span>
                    </div>
                    <div className="flex items-center justify-between text-[12px] text-[var(--fg-secondary)]">
                      <span>{t("请求体", "Body")}</span>
                      <span className="[font-variant-numeric:tabular-nums]">{formatSize(sizeDetails?.requestBody ?? 0)}</span>
                    </div>
                  </div>
                </div>
              </div>
              <span className="text-[var(--fg-muted)]">•</span>
              <div className="relative group/network">
                <span className="inline-flex h-5 w-5 items-center justify-center rounded-[6px] text-[var(--fg-muted)]">
                  <span className="relative inline-flex">
                    <AppIcon name="globe" size={13} />
                    {hasTLSNetwork && (
                      <AppIcon
                        name="lock"
                        size={8}
                        className="absolute -bottom-[1px] -right-[3px] rounded-[2px] bg-[var(--surface)]"
                      />
                    )}
                  </span>
                </span>
                <div
                  className={cn(
                    "absolute right-0 top-[calc(100%+8px)] z-20 w-[min(460px,calc(100vw-24px))] rounded-[10px] border shadow-[var(--shadow-lg)]",
                    "bg-[var(--surface)] border-[var(--border-color)] px-3 py-2.5 opacity-0 translate-y-1 pointer-events-none",
                    "transition-all duration-150 group-hover/network:opacity-100 group-hover/network:translate-y-0 group-hover/network:pointer-events-auto"
                  )}
                >
                  <div className="mb-1.5 flex items-center gap-2">
                    <span className="inline-flex items-center justify-center rounded-[6px] bg-[var(--surface-secondary)] p-1">
                      <AppIcon name="globe" size={12} className="text-[var(--fg-secondary)]" />
                    </span>
                    <span className="text-[12px] font-semibold text-[var(--fg)]">{t("网络", "Network")}</span>
                    <span className="relative group/network-info ml-auto inline-flex h-5 w-5 items-center justify-center rounded-[6px] text-[var(--fg-muted)] hover:bg-[var(--surface-secondary)]">
                      <AppIcon name="info" size={11} />
                      <span
                        className={cn(
                          "pointer-events-none absolute right-0 top-[calc(100%+6px)] z-10 w-[300px] rounded-[8px] border border-[var(--border-color)]",
                          "bg-[var(--surface-elevated)] px-2.5 py-2 text-[11px] leading-5 text-[var(--fg-secondary)] shadow-[var(--shadow-md)]",
                          "opacity-0 translate-y-1 transition-all duration-120",
                          "group-hover/network-info:opacity-100 group-hover/network-info:translate-y-0"
                        )}
                      >
                        {t("这里展示本次请求的网络连接与 TLS 协商结果，便于定位代理、证书和链路问题。", "Shows network and TLS negotiation details for this request to help diagnose proxy, certificate, and connection issues.")}
                      </span>
                    </span>
                  </div>

                  <div className="grid grid-cols-[126px_minmax(0,1fr)] gap-x-3 gap-y-1 text-[12px] leading-5">
                    <span className="text-[var(--fg-secondary)]">{t("HTTP 版本", "HTTP Version")}</span>
                    <span className="font-mono text-[var(--fg)]">{displayNetworkValue(response.network?.httpVersion)}</span>
                    <span className="text-[var(--fg-secondary)]">{t("本地地址", "Local Address")}</span>
                    <span className="font-mono text-[var(--fg)]">{displayNetworkValue(response.network?.localAddress)}</span>
                    <span className="text-[var(--fg-secondary)]">{t("远端地址", "Remote Address")}</span>
                    <span className="font-mono text-[var(--fg)]">{displayNetworkValue(response.network?.remoteAddress)}</span>
                  </div>

                  <div className="my-1.5 h-px bg-[var(--border-subtle)]" />

                  <div className="grid grid-cols-[126px_minmax(0,1fr)] gap-x-3 gap-y-1 text-[12px] leading-5">
                    <span className="text-[var(--fg-secondary)]">{t("TLS 协议", "TLS Protocol")}</span>
                    <span className="font-mono text-[var(--fg)]">{displayNetworkValue(response.network?.tlsProtocol)}</span>
                    <span className="text-[var(--fg-secondary)]">{t("加密套件", "Cipher Name")}</span>
                    <span className="font-mono text-[var(--fg)] break-all">{displayNetworkValue(response.network?.cipherName)}</span>
                  </div>

                  <div className="my-1.5 h-px bg-[var(--border-subtle)]" />

                  <div className="grid grid-cols-[126px_minmax(0,1fr)] gap-x-3 gap-y-1 text-[12px] leading-5">
                    <span className="text-[var(--fg-secondary)]">{t("证书 CN", "Certificate CN")}</span>
                    <span className="font-mono text-[var(--fg)] break-all">{displayNetworkValue(response.network?.certificateCN)}</span>
                    <span className="text-[var(--fg-secondary)]">{t("签发者 CN", "Issuer CN")}</span>
                    <span className="font-mono text-[var(--fg)] break-all">{displayNetworkValue(response.network?.issuerCN)}</span>
                    <span className="text-[var(--fg-secondary)]">{t("有效期至", "Valid Until")}</span>
                    <span className="font-mono text-[var(--fg)]">{formatNetworkValidUntil(response.network?.validUntil)}</span>
                  </div>
                </div>
              </div>
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
                      <div className="rounded-[12px] border border-[var(--danger)]/25 bg-[var(--surface-elevated)] px-4 py-3 shadow-[var(--shadow-sm)]">
                        <div className="mb-2 flex items-center gap-2.5">
                          <div className="flex h-6 w-6 items-center justify-center rounded-[999px] bg-[var(--danger)]/12">
                            <AppIcon name="info" size={11} className="text-[var(--danger)]" />
                          </div>
                          <div className="text-[14px] font-semibold text-[var(--fg)] leading-5">
                            {t("无法发送请求", "Could not send request")}
                          </div>
                        </div>

                        <div className="space-y-2.5">
                          <div className="flex items-start gap-2 text-[12px] font-mono">
                            <span className={cn("font-semibold uppercase", METHOD_COLORS[(activeTab?.request.method as HttpMethod)] || "text-[var(--fg-muted)]")}>
                              {activeTab?.request.method || "GET"}
                            </span>
                            <span className="text-[var(--fg)] break-all">{resolvedRequestUrl || rawRequestUrl}</span>
                          </div>
                          <div className={cn("max-w-full rounded-[9px] border px-2.5 py-2 text-[12px] leading-5 break-all", errorPresentation?.badgeToneClass || "bg-[#fbeceb] text-[#b44840]", "border-current/20")}>
                            <span className="font-medium">{t("错误", "Error")}:</span>
                            <span className="ml-1">{errorPresentation?.badgeText?.replace(/^Error:\s*/i, "") || responseError}</span>
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
                              {t("在控制台查看", "View in console")}
                            </button>
                            <span>•</span>
                            <span>{t("错误码", "Code")}: {parsedError?.code || "REQUEST_FAILED"}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              ) : hasStreamEntries || streamActive ? (
                <ResponseStream entries={streamEntries} isDark={isDark} />
              ) : response ? (
                <ResponseBody body={response.body} contentType={response.contentType} isDark={isDark} />
              ) : (
                <div className="flex h-full items-center justify-center text-2xs text-[var(--fg-muted)]">
                  {t("暂无响应内容", "No response content")}
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
