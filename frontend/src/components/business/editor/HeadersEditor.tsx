import { useMemo, useCallback } from "react"
import { KeyValueEditor } from "./KeyValueEditor"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { useCookieStore } from "@/stores/cookieStore"
import { createKeyValuePair } from "@/types/request"
import {
  buildAutoHeaderDisabledMarkerKey,
  getSuppressedAutoHeaders,
  isAutoHeaderDisabledMarkerKey,
  normalizeHeaderName,
} from "@/lib/autoHeaders"

const TOKEN_HEADER_NAME = "MiniPost-Token"

function deriveAutoContentType(activeTab: NonNullable<ReturnType<typeof getProjectActiveTabFromState>>) {
  const { body } = activeTab.request
  if (body.type === "json" && (body.json ?? "").trim()) return "application/json"
  if (body.type === "raw" && (body.raw ?? "").trim()) return "text/plain"
  if (body.type === "form-urlencoded") return "application/x-www-form-urlencoded"
  return ""
}

function getHeaderTip(key: string): { tipTitle: string; tipContent: string; tipSettingHint?: string } {
  switch (key.toLowerCase()) {
    case "postman-token":
    case "minipost-token":
      return {
        tipTitle: "建议开启该请求头",
        tipContent: "MiniPost-Token 会在每次发送时生成随机值，便于服务端区分相同参数的多次请求，也有助于排查问题。",
        tipSettingHint: "可在 设置 > 通用 > 发送 MiniPost-Token 请求头 中开启或关闭。",
      }
    case "cache-control":
      return {
        tipTitle: "建议按需开启",
        tipContent: "Cache-Control: no-cache 会要求服务端在返回缓存内容前重新校验，减少拿到陈旧响应的概率。",
        tipSettingHint: "可在 设置 > 通用 > 发送 no-cache 请求头 中开启或关闭。",
      }
    case "content-type":
      return {
        tipTitle: "请求体类型声明",
        tipContent: "用于告诉服务端当前请求体的数据格式（例如 JSON、表单等），通常根据 Body 编辑区的类型自动推导；如需自定义可在 Headers 手动填写同名项覆盖。",
      }
    case "content-length":
      return {
        tipTitle: "请求体长度",
        tipContent: "表示请求体字节长度。运行时会在发送前自动计算；如确有需要，也可以在 Headers 中手动覆盖。",
      }
    case "host":
      return {
        tipTitle: "目标主机标识",
        tipContent: "Host 由请求 URL 自动得出（域名和端口），HTTP/1.1 请求通常会带上该头。",
      }
    case "user-agent":
      return {
        tipTitle: "客户端标识",
        tipContent: "用于标识请求来源客户端，方便服务端统计、日志分析和兼容性处理。",
      }
    case "accept":
      return {
        tipTitle: "可接受的响应类型",
        tipContent: "告知服务端客户端期望的响应媒体类型。默认 */* 代表可接受任意类型。",
      }
    case "accept-encoding":
      return {
        tipTitle: "压缩协商",
        tipContent: "用于声明可接受的压缩算法（如 gzip/br），运行时会按网络栈能力自动协商。",
      }
    case "connection":
      return {
        tipTitle: "连接管理",
        tipContent: "用于控制连接复用或关闭，通常由底层 HTTP 运行时自动设置，不建议手动干预。",
      }
    case "authorization":
      return {
        tipTitle: "认证信息",
        tipContent: "该头由 Auth 面板自动组装（Bearer/Basic），无需在 Headers 再重复手填。",
      }
    case "cookie":
      return {
        tipTitle: "Cookie Jar 注入",
        tipContent: "该头来自 Cookie 管理器，会按当前 URL 自动拼接可用 Cookie。",
        tipSettingHint: "可在 设置 > 通用 > 禁用 Cookies 中关闭自动注入。",
      }
    default:
      return {
        tipTitle: "自动生成请求头",
        tipContent: "该请求头会在发送前由系统自动补齐，用于保持请求行为一致。",
      }
  }
}

export function HeadersEditor() {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const disableCookies = useUIStore((s) => s.disableCookies)
  const sendNoCacheHeader = useUIStore((s) => s.sendNoCacheHeader)
  const sendPostmanTokenHeader = useUIStore((s) => s.sendPostmanTokenHeader)
  const cookies = useCookieStore((s) => s.cookies)
  const getCookieHeader = useCookieStore((s) => s.getCookieHeader)

  if (!activeTab) return null

  const markerHeaders = useMemo(
    () => activeTab.request.headers.filter((header) => isAutoHeaderDisabledMarkerKey(header.key)),
    [activeTab.request.headers]
  )
  const editableHeaders = useMemo(
    () => activeTab.request.headers.filter((header) => !isAutoHeaderDisabledMarkerKey(header.key)),
    [activeTab.request.headers]
  )

  const handleEditableHeadersChange = useCallback((headers: typeof editableHeaders) => {
    updateTabRequest(activeTab.id, { headers: [...headers, ...markerHeaders] })
  }, [activeTab.id, markerHeaders, updateTabRequest])

  const handleToggleAutoHeader = useCallback((key: string, enabled: boolean) => {
    const normalized = normalizeHeaderName(key)
    const currentHeaders = activeTab.request.headers
    const markerKey = buildAutoHeaderDisabledMarkerKey(normalized)
    const alreadySuppressed = currentHeaders.some((header) => normalizeHeaderName(header.key) === markerKey)

    if (!enabled && !alreadySuppressed) {
      updateTabRequest(activeTab.id, {
        headers: [
          ...currentHeaders,
          createKeyValuePair({
            key: markerKey,
            value: "",
            enabled: false,
            description: "auto-header-disabled",
          }),
        ],
      })
      return
    }
    if (enabled && alreadySuppressed) {
      updateTabRequest(activeTab.id, {
        headers: currentHeaders.filter((header) => normalizeHeaderName(header.key) !== markerKey),
      })
    }
  }, [activeTab.id, activeTab.request.headers, updateTabRequest])

  const autoGeneratedItems = useMemo(() => {
    const normalized = editableHeaders
      .filter((h) => h.enabled && h.key.trim())
      .map((h) => ({
        key: normalizeHeaderName(h.key),
        rawKey: h.key.trim(),
      }))
    const suppressed = getSuppressedAutoHeaders(activeTab.request.headers)

    const hasHeader = (name: string) => normalized.some((h) => h.key === normalizeHeaderName(name))
    const hasCustomHeader = (name: string) => normalized.some((h) => normalizeHeaderName(h.rawKey) === normalizeHeaderName(name))
    const isEnabled = (name: string) => !suppressed.has(normalizeHeaderName(name))
    const hasAnyTokenHeader = hasHeader("minipost-token") || hasHeader("postman-token")

    const items: Array<{ key: string; value: string; enabled: boolean; description: string; tipTitle?: string; tipContent?: string; tipSettingHint?: string }> = []
    const autoContentType = deriveAutoContentType(activeTab)

    const pushAutoHeader = (key: string, value: string, description: string) => {
      const tip = getHeaderTip(key)
      items.push({
        key,
        value,
        enabled: isEnabled(key),
        description,
        tipTitle: tip.tipTitle,
        tipContent: tip.tipContent,
        tipSettingHint: tip.tipSettingHint,
      })
    }

    if (sendPostmanTokenHeader && !hasAnyTokenHeader) {
      pushAutoHeader(TOKEN_HEADER_NAME, "<calculated when request is sent>", "请求发送时自动生成随机 token")
    }
    if (sendNoCacheHeader && !hasHeader("cache-control")) {
      pushAutoHeader("Cache-Control", "no-cache", "根据设置自动注入")
    }
    if (autoContentType && !hasHeader("content-type")) {
      pushAutoHeader("Content-Type", autoContentType, "根据 Body 类型自动生成")
    }
    if ((activeTab.request.body.type !== "none" || autoContentType) && !hasHeader("content-length")) {
      pushAutoHeader("Content-Length", "<calculated when request is sent>", "运行时计算请求体长度")
    }
    if (!hasHeader("host")) {
      pushAutoHeader("Host", "<calculated when request is sent>", "根据请求 URL 自动计算")
    }
    if (!hasHeader("user-agent")) {
      pushAutoHeader("User-Agent", "MiniPost/1.0", "客户端默认标识")
    }
    if (!hasHeader("accept")) {
      pushAutoHeader("Accept", "*/*", "默认接受任意响应类型")
    }
    if (!hasHeader("accept-encoding")) {
      pushAutoHeader("Accept-Encoding", "<calculated by runtime>", "由底层 HTTP 运行时管理压缩协商")
    }
    if (!hasHeader("connection")) {
      pushAutoHeader("Connection", "<calculated by runtime>", "由底层 HTTP 运行时管理连接复用")
    }

    if (!hasHeader("authorization")) {
      const auth = activeTab.request.auth
      if (auth.type === "bearer" && auth.bearer?.token) {
        pushAutoHeader("Authorization", `Bearer ${auth.bearer.token}`, "由 Auth(Bearer) 自动注入")
      } else if (auth.type === "basic" && (auth.basic?.username || auth.basic?.password)) {
        pushAutoHeader("Authorization", "Basic <calculated from auth>", "由 Auth(Basic) 自动注入")
      }
    }

    if (activeTab.request.auth.type === "api-key" && activeTab.request.auth.apiKey?.addTo === "header") {
      const key = (activeTab.request.auth.apiKey.key || "").trim()
      if (key && !hasCustomHeader(key)) {
        const tip = getHeaderTip(key)
        items.push({
          key,
          value: activeTab.request.auth.apiKey.value || "",
          enabled: isEnabled(key),
          description: "由 Auth(API Key) 自动注入到 Header",
          tipTitle: tip.tipTitle,
          tipContent: tip.tipContent,
          tipSettingHint: tip.tipSettingHint,
        })
      }
    }

    if (!disableCookies && !hasHeader("cookie")) {
      const cookieHeader = getCookieHeader(activeTab.request.url)
      if (cookieHeader) {
        pushAutoHeader("Cookie", cookieHeader, "由 Cookie Jar 自动注入")
      }
    }

    return items
  }, [activeTab, disableCookies, editableHeaders, getCookieHeader, cookies, sendNoCacheHeader, sendPostmanTokenHeader])

  return (
    <KeyValueEditor
      items={editableHeaders}
      onChange={handleEditableHeadersChange}
      onToggleAutoGeneratedItem={handleToggleAutoHeader}
      keyPlaceholder="Header 名"
      valuePlaceholder="Header 值"
      autoGeneratedItems={autoGeneratedItems}
      hideAutoGeneratedByDefault
    />
  )
}
