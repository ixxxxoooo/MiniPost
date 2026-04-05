import { useState } from "react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { AppIcon } from "@/components/ui/icon"
import { ResponseBody } from "./ResponseBody"
import { ResponseHeaders } from "./ResponseHeaders"
import { ResponseCookies } from "./ResponseCookies"
import { cn } from "@/lib/utils"

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

function statusBadgeClass(code: number): string {
  if (code >= 200 && code < 300) return "bg-[var(--success)]/12 text-[var(--success)]"
  if (code >= 300 && code < 400) return "bg-[var(--info)]/12 text-[var(--info)]"
  if (code >= 400 && code < 500) return "bg-[var(--danger)]/12 text-[var(--danger)]"
  return "bg-[var(--danger)]/12 text-[var(--danger)]"
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

function LoadingTopShimmer() {
  return (
    <div className="absolute left-0 top-0 h-[2px] w-full overflow-hidden bg-[var(--border-subtle)]/60">
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

  if (!response && !responseError && !isSending) {
    return (
      <div className="flex h-full items-center justify-center bg-[var(--surface)]">
        <span className="text-[length:var(--size-font-xs)] text-[var(--fg-muted)]">
          发送请求后在此查看响应
        </span>
      </div>
    )
  }

  if (!response && !responseError && isSending) {
    return (
      <div className="relative h-full bg-[var(--surface)]">
        <LoadingTopShimmer />
      </div>
    )
  }

  if (responseError && !response) {
    return (
      <div className="flex h-full items-center justify-center gap-2 px-6 bg-[var(--surface)]">
        <AppIcon name="clear" size={16} className="text-[var(--danger)]" />
        <span className="text-[length:var(--size-font-xs)] text-[var(--danger)]">{responseError}</span>
      </div>
    )
  }

  if (!response) return null
  const headerCount = isSending ? 0 : Object.keys(response.headers).length
  const cookies = parseResponseCookies(response.headers)
  const cookieCount = isSending ? 0 : cookies.length

  return (
    <div className="flex h-full flex-col bg-[var(--surface)] relative">
      {/* 发送中的叠加层 */}
      {isSending && (
        <div className="absolute inset-0 z-10 pointer-events-auto bg-[var(--surface)]/68 backdrop-blur-[1px]">
          <LoadingTopShimmer />
        </div>
      )}

      <Tabs value={activeTabValue} onValueChange={setActiveTabValue} className="flex-1 flex flex-col overflow-hidden">
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

          {!isSending && (
            <div className="ml-auto flex items-center gap-2 text-[11px] text-[var(--fg-muted)]">
              <div className="relative group/status">
                <span className={cn("px-2 py-0.5 rounded-[8px] font-medium", statusBadgeClass(response.statusCode))}>
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
              <span className="text-[var(--fg-muted)] [font-variant-numeric:tabular-nums]">{formatDuration(response.duration)}</span>
              <span className="text-[var(--fg-muted)]">•</span>
              <span className="text-[var(--fg-muted)] [font-variant-numeric:tabular-nums]">{formatSize(response.size)}</span>
              <span className="text-[var(--fg-muted)]">•</span>
              <span className="max-w-[180px] truncate font-mono text-[var(--fg-muted)]">{response.contentType}</span>
            </div>
          )}
        </div>

        <TabsContent value="body" className="flex-1 m-0 min-h-0 overflow-hidden">
          <div className="h-full p-[var(--size-padding-sm)]">
            <div className="h-full overflow-hidden">
              <ResponseBody body={response.body} contentType={response.contentType} isDark={isDark} />
            </div>
          </div>
        </TabsContent>

        <TabsContent value="headers" className="flex-1 m-0 min-h-0 overflow-hidden">
          <div className="h-full p-[var(--size-padding-sm)]">
            <ResponseHeaders headers={response.headers} />
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
