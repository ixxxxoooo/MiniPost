import { useMemo, useCallback } from "react"
import { KeyValueEditor } from "./KeyValueEditor"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { useCookieStore } from "@/stores/cookieStore"
import { createKeyValuePair } from "@/types/request"
import { useI18n } from "@/hooks/useI18n"
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

function getHeaderTip(key: string, isZh: boolean): { tipTitle: string; tipContent: string; tipSettingHint?: string } {
  const t = (zh: string, en: string) => (isZh ? zh : en)
  switch (key.toLowerCase()) {
    case "postman-token":
    case "minipost-token":
      return {
        tipTitle: t("建议开启该请求头", "Recommended to enable this header"),
        tipContent: t(
          "MiniPost-Token 会在每次发送时生成随机值，便于服务端区分相同参数的多次请求，也有助于排查问题。",
          "MiniPost-Token generates a random value on each request, helping distinguish repeated requests and troubleshoot issues."
        ),
        tipSettingHint: t("可在 设置 > 通用 > 发送 MiniPost-Token 请求头 中开启或关闭。", "You can toggle this in Settings > General > Send MiniPost-Token header."),
      }
    case "cache-control":
      return {
        tipTitle: t("建议按需开启", "Enable when needed"),
        tipContent: t(
          "Cache-Control: no-cache 会要求服务端在返回缓存内容前重新校验，减少拿到陈旧响应的概率。",
          "Cache-Control: no-cache asks the server to revalidate cache before returning content, reducing stale responses."
        ),
        tipSettingHint: t("可在 设置 > 通用 > 发送 no-cache 请求头 中开启或关闭。", "You can toggle this in Settings > General > Send no-cache header."),
      }
    case "content-type":
      return {
        tipTitle: t("请求体类型声明", "Request body type"),
        tipContent: t(
          "用于告诉服务端当前请求体的数据格式（例如 JSON、表单等），通常根据 Body 编辑区的类型自动推导；如需自定义可在 Headers 手动填写同名项覆盖。",
          "Tells the server the body format (such as JSON or form). It is usually inferred from Body editor type, and can be overridden in Headers."
        ),
      }
    case "content-length":
      return {
        tipTitle: t("请求体长度", "Request body length"),
        tipContent: t(
          "表示请求体字节长度。运行时会在发送前自动计算；如确有需要，也可以在 Headers 中手动覆盖。",
          "Represents body byte size. It is auto-calculated before sending, but can be manually overridden if needed."
        ),
      }
    case "host":
      return {
        tipTitle: t("目标主机标识", "Target host"),
        tipContent: t("Host 由请求 URL 自动得出（域名和端口），HTTP/1.1 请求通常会带上该头。", "Host is derived from request URL (domain and port), and is usually sent in HTTP/1.1 requests."),
      }
    case "user-agent":
      return {
        tipTitle: t("客户端标识", "Client identity"),
        tipContent: t("用于标识请求来源客户端，方便服务端统计、日志分析和兼容性处理。", "Identifies the client source for server analytics, logging, and compatibility handling."),
      }
    case "accept":
      return {
        tipTitle: t("可接受的响应类型", "Accepted response types"),
        tipContent: t("告知服务端客户端期望的响应媒体类型。默认 */* 代表可接受任意类型。", "Tells the server expected response media types. Default */* means any type."),
      }
    case "accept-encoding":
      return {
        tipTitle: t("压缩协商", "Compression negotiation"),
        tipContent: t("用于声明可接受的压缩算法（如 gzip/br），运行时会按网络栈能力自动协商。", "Declares supported compression algorithms (gzip/br etc.), negotiated automatically by runtime."),
      }
    case "connection":
      return {
        tipTitle: t("连接管理", "Connection management"),
        tipContent: t("用于控制连接复用或关闭，通常由底层 HTTP 运行时自动设置，不建议手动干预。", "Controls connection reuse/close behavior and is usually set by the HTTP runtime."),
      }
    case "authorization":
      return {
        tipTitle: t("认证信息", "Authentication info"),
        tipContent: t("该头由 Auth 面板自动组装（Bearer/Basic），无需在 Headers 再重复手填。", "This header is automatically assembled by Auth panel (Bearer/Basic), so you usually do not need to fill it again."),
      }
    case "cookie":
      return {
        tipTitle: t("Cookie Jar 注入", "Cookie Jar injection"),
        tipContent: t("该头来自 Cookie 管理器，会按当前 URL 自动拼接可用 Cookie。", "This header comes from Cookie manager and is auto-composed based on current URL."),
        tipSettingHint: t("可在 设置 > 通用 > 禁用 Cookies 中关闭自动注入。", "You can disable auto-injection in Settings > General > Disable Cookies."),
      }
    default:
      return {
        tipTitle: t("自动生成请求头", "Auto-generated header"),
        tipContent: t("该请求头会在发送前由系统自动补齐，用于保持请求行为一致。", "This header is auto-filled before sending to keep request behavior consistent."),
      }
  }
}

export function HeadersEditor() {
  const { t, isZh } = useI18n()
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
      const tip = getHeaderTip(key, isZh)
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
      pushAutoHeader(
        TOKEN_HEADER_NAME,
        t("<发送请求时自动计算>", "<calculated when request is sent>"),
        t("请求发送时自动生成随机 token", "Automatically generates random token when sending")
      )
    }
    if (sendNoCacheHeader && !hasHeader("cache-control")) {
      pushAutoHeader("Cache-Control", "no-cache", t("根据设置自动注入", "Auto-injected from settings"))
    }
    if (autoContentType && !hasHeader("content-type")) {
      pushAutoHeader("Content-Type", autoContentType, t("根据 Body 类型自动生成", "Auto-generated from Body type"))
    }
    if ((activeTab.request.body.type !== "none" || autoContentType) && !hasHeader("content-length")) {
      pushAutoHeader(
        "Content-Length",
        t("<发送请求时自动计算>", "<calculated when request is sent>"),
        t("运行时计算请求体长度", "Body length calculated at runtime")
      )
    }
    if (!hasHeader("host")) {
      pushAutoHeader("Host", t("<发送请求时自动计算>", "<calculated when request is sent>"), t("根据请求 URL 自动计算", "Auto-calculated from request URL"))
    }
    if (!hasHeader("user-agent")) {
      pushAutoHeader("User-Agent", "MiniPost/1.0", t("客户端默认标识", "Default client identifier"))
    }
    if (!hasHeader("accept")) {
      pushAutoHeader("Accept", "*/*", t("默认接受任意响应类型", "Accept any response type by default"))
    }
    if (!hasHeader("accept-encoding")) {
      pushAutoHeader(
        "Accept-Encoding",
        t("<由运行时自动计算>", "<calculated by runtime>"),
        t("由底层 HTTP 运行时管理压缩协商", "Compression negotiation managed by HTTP runtime")
      )
    }
    if (!hasHeader("connection")) {
      pushAutoHeader(
        "Connection",
        t("<由运行时自动计算>", "<calculated by runtime>"),
        t("由底层 HTTP 运行时管理连接复用", "Connection reuse managed by HTTP runtime")
      )
    }

    if (!hasHeader("authorization")) {
      const auth = activeTab.request.auth
      if (auth.type === "bearer" && auth.bearer?.token) {
        pushAutoHeader("Authorization", `Bearer ${auth.bearer.token}`, t("由 Auth(Bearer) 自动注入", "Auto-injected by Auth (Bearer)"))
      } else if (auth.type === "basic" && (auth.basic?.username || auth.basic?.password)) {
        pushAutoHeader(
          "Authorization",
          t("Basic <由认证信息计算>", "Basic <calculated from auth>"),
          t("由 Auth(Basic) 自动注入", "Auto-injected by Auth (Basic)")
        )
      }
    }

    if (activeTab.request.auth.type === "api-key" && activeTab.request.auth.apiKey?.addTo === "header") {
      const key = (activeTab.request.auth.apiKey.key || "").trim()
      if (key && !hasCustomHeader(key)) {
        const tip = getHeaderTip(key, isZh)
        items.push({
          key,
          value: activeTab.request.auth.apiKey.value || "",
          enabled: isEnabled(key),
          description: t("由 Auth(API Key) 自动注入到 Header", "Auto-injected into header by Auth (API Key)"),
          tipTitle: tip.tipTitle,
          tipContent: tip.tipContent,
          tipSettingHint: tip.tipSettingHint,
        })
      }
    }

    if (!disableCookies && !hasHeader("cookie")) {
      const cookieHeader = getCookieHeader(activeTab.request.url)
      if (cookieHeader) {
        pushAutoHeader("Cookie", cookieHeader, t("由 Cookie Jar 自动注入", "Auto-injected by Cookie Jar"))
      }
    }

    return items
  }, [activeTab, disableCookies, editableHeaders, getCookieHeader, isZh, cookies, sendNoCacheHeader, sendPostmanTokenHeader, t])

  return (
    <KeyValueEditor
      items={editableHeaders}
      onChange={handleEditableHeadersChange}
      onToggleAutoGeneratedItem={handleToggleAutoHeader}
      keyPlaceholder={t("Header 名", "Header Name")}
      valuePlaceholder={t("Header 值", "Header Value")}
      autoGeneratedItems={autoGeneratedItems}
      hideAutoGeneratedByDefault
    />
  )
}
