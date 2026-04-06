import { SendRequest, SendRequestWithEnv } from "../../wailsjs/go/main/App"
import type { HttpResponse } from "@/types/response"
import type { RequestData } from "@/types/request"
import { stripJsonComments } from "@/components/ui/CodeEditor"
import { useCookieStore } from "@/stores/cookieStore"
import { useUIStore } from "@/stores/uiStore"
import { getSuppressedAutoHeaders, normalizeHeaderName } from "@/lib/autoHeaders"

const TOKEN_HEADER_NAME = "MiniPost-Token"

export interface SendRequestPayload {
  method: string
  url: string
  params: { key: string; value: string }[]
  headers: { key: string; value: string }[]
  body: {
    type: string
    raw: string
    json: string
    formUrlEncoded: { key: string; value: string }[]
    formData: { key: string; value: string; type: string; filePath?: string; fileName?: string }[]
  }
  auth: {
    type: string
    basic: { username: string; password: string }
    bearer: { token: string }
    apiKey: { key: string; value: string; addTo: string }
  }
}

function buildPayload(request: RequestData): SendRequestPayload {
  return {
    method: request.method,
    url: request.url,
    params: request.params
      .filter((p) => p.enabled && p.key)
      .map((p) => ({ key: p.key, value: p.value })),
    headers: request.headers
      .filter((h) => h.enabled && h.key)
      .map((h) => ({ key: h.key, value: h.value })),
    body: {
      type: request.body.type,
      raw: stripJsonComments(request.body.raw ?? ""),
      json: stripJsonComments(request.body.json ?? ""),
      formUrlEncoded: (request.body.formUrlEncoded ?? [])
        .filter((f) => f.enabled && f.key)
        .map((f) => ({ key: f.key, value: f.value })),
      formData: (request.body.formData ?? [])
        .filter((f) => f.enabled && f.key)
        .map((f) => ({
          key: f.key,
          value: f.value ?? "",
          type: f.type,
          filePath: f.filePath,
          fileName: f.fileName,
        })),
    },
    auth: {
      type: request.auth.type,
      basic: request.auth.basic ?? { username: "", password: "" },
      bearer: request.auth.bearer ?? { token: "" },
      apiKey: request.auth.apiKey ?? { key: "", value: "", addTo: "header" },
    },
  }
}

function mergeCookieHeader(manual: string, fromJar: string): string {
  const map = new Map<string, string>()
  const append = (source: string, overwrite: boolean) => {
    source.split(";").forEach((pair) => {
      const trimmed = pair.trim()
      if (!trimmed) return
      const sep = trimmed.indexOf("=")
      if (sep <= 0) return
      const key = trimmed.slice(0, sep).trim()
      const value = trimmed.slice(sep + 1).trim()
      if (!key) return
      if (overwrite || !map.has(key)) {
        map.set(key, value)
      }
    })
  }
  append(fromJar, false)
  append(manual, true)
  return Array.from(map.entries()).map(([k, v]) => `${k}=${v}`).join("; ")
}

function hasHeader(headers: Array<{ key: string; value: string }>, name: string): boolean {
  const target = normalizeHeaderName(name)
  return headers.some((header) => normalizeHeaderName(header.key) === target)
}

export async function sendHttpRequest(
  request: RequestData,
  projectId?: string,
  envId?: string
): Promise<HttpResponse> {
  const payload = buildPayload(request)
  const uiSettings = useUIStore.getState()
  const cookieStore = useCookieStore.getState()
  const suppressedAutoHeaders = getSuppressedAutoHeaders(request.headers)
  const isSuppressed = (name: string) => suppressedAutoHeaders.has(normalizeHeaderName(name))
  const isTokenSuppressed = isSuppressed("minipost-token") || isSuppressed("postman-token")
  if (!uiSettings.disableCookies) {
    const cookieHeader = cookieStore.getCookieHeader(request.url)
    if (cookieHeader && !isSuppressed("cookie")) {
      const existing = payload.headers.find((h) => h.key.toLowerCase() === "cookie")
      if (!existing) {
        payload.headers.push({ key: "Cookie", value: cookieHeader })
      } else {
        existing.value = mergeCookieHeader(existing.value, cookieHeader)
      }
    }
  }

  if (uiSettings.sendNoCacheHeader && !isSuppressed("cache-control") && !hasHeader(payload.headers, "Cache-Control")) {
    payload.headers.push({ key: "Cache-Control", value: "no-cache" })
  }
  if (uiSettings.sendPostmanTokenHeader && !isTokenSuppressed && !hasHeader(payload.headers, "MiniPost-Token") && !hasHeader(payload.headers, "Postman-Token")) {
    payload.headers.push({ key: TOKEN_HEADER_NAME, value: crypto.randomUUID() })
  }

  payload.headers.push(
    { key: "X-MiniPost-Option-Follow-Redirects", value: uiSettings.followRedirects ? "1" : "0" },
    { key: "X-MiniPost-Option-Timeout-Ms", value: String(Math.max(0, Math.round(uiSettings.requestTimeoutMs))) },
    { key: "X-MiniPost-Option-Max-Response-Size-MB", value: String(Math.max(0, Math.round(uiSettings.maxResponseSizeMB))) },
    { key: "X-MiniPost-Option-SSL-Verify", value: uiSettings.sslCertificateVerification ? "1" : "0" },
    { key: "X-MiniPost-Option-HTTP-Version", value: uiSettings.httpVersion },
    { key: "X-MiniPost-Option-Disable-Default-User-Agent", value: isSuppressed("user-agent") ? "1" : "0" },
    { key: "X-MiniPost-Option-Disable-Default-Accept", value: isSuppressed("accept") ? "1" : "0" },
    { key: "X-MiniPost-Option-Disable-Auto-Content-Type", value: isSuppressed("content-type") ? "1" : "0" },
  )

  // 始终使用 SendRequestWithEnv 以确保历史记录被保存
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = projectId
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ? await SendRequestWithEnv(payload as any, projectId, envId || "")
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    : await SendRequest(payload as any)

  const normalizedResult: HttpResponse = {
    statusCode: result.statusCode,
    statusText: result.statusText,
    headers: result.headers,
    body: result.body,
    duration: result.duration,
    size: result.size,
    contentType: result.contentType,
    protocol: (result as unknown as { protocol?: string }).protocol,
    warnings: (result as unknown as { warnings?: string[] }).warnings ?? [],
    timings: (result as unknown as { timings?: HttpResponse["timings"] }).timings,
    sizeDetails: (result as unknown as { sizeDetails?: HttpResponse["sizeDetails"] }).sizeDetails,
  }

  if (!uiSettings.disableCookies) {
    cookieStore.absorbResponseCookies(request.url, normalizedResult.headers)
  }
  return normalizedResult
}
