import { SendRequest, SendRequestWithEnv, SendRequestWithEnvStream } from "../../wailsjs/go/main/App"
import type { HttpResponse, HttpStreamEventPayload, StreamEntryKind } from "@/types/response"
import type { RequestData } from "@/types/request"
import { stripJsonComments } from "@/components/ui/CodeEditor"
import { useCookieStore } from "@/stores/cookieStore"
import { useUIStore } from "@/stores/uiStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { getSuppressedAutoHeaders, normalizeHeaderName } from "@/lib/autoHeaders"
import { ensureRequestProtocol, resolveTemplateVariables } from "@/lib/variableResolver"

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

export interface SendRequestStreamOptions {
  streamId: string
  onStreamEvent: (event: HttpStreamEventPayload) => void
}

function dedupeHeaders(headers: Array<{ key: string; value: string }>): Array<{ key: string; value: string }> {
  const result: Array<{ key: string; value: string }> = []
  const indexByName = new Map<string, number>()

  headers.forEach((header) => {
    const key = header.key?.trim()
    if (!key) return
    const normalized = normalizeHeaderName(key)
    const existingIndex = indexByName.get(normalized)
    if (existingIndex === undefined) {
      indexByName.set(normalized, result.length)
      result.push({ key, value: header.value })
      return
    }
    result[existingIndex] = { key, value: header.value }
  })

  return result
}

function buildPayload(request: RequestData): SendRequestPayload {
  return {
    method: request.method,
    url: request.url,
    params: request.params
      .filter((p) => p.enabled && p.key)
      .map((p) => ({ key: p.key, value: p.value })),
    headers: dedupeHeaders(
      request.headers
        .filter((h) => h.enabled && h.key)
        .map((h) => ({ key: h.key, value: h.value }))
    ),
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

function resolvePayloadVariables(payload: SendRequestPayload, variables: Array<{ key: string; value: string }>): SendRequestPayload {
  if (!variables.length) return payload

  return {
    ...payload,
    url: resolveTemplateVariables(payload.url, variables),
    params: payload.params.map((item) => ({
      key: resolveTemplateVariables(item.key, variables),
      value: resolveTemplateVariables(item.value, variables),
    })),
    headers: payload.headers.map((item) => ({
      key: resolveTemplateVariables(item.key, variables),
      value: resolveTemplateVariables(item.value, variables),
    })),
    body: {
      ...payload.body,
      raw: resolveTemplateVariables(payload.body.raw, variables),
      json: resolveTemplateVariables(payload.body.json, variables),
      formUrlEncoded: payload.body.formUrlEncoded.map((item) => ({
        key: resolveTemplateVariables(item.key, variables),
        value: resolveTemplateVariables(item.value, variables),
      })),
      formData: payload.body.formData.map((item) => ({
        ...item,
        key: resolveTemplateVariables(item.key, variables),
        value: resolveTemplateVariables(item.value, variables),
        filePath: resolveTemplateVariables(item.filePath ?? "", variables),
        fileName: resolveTemplateVariables(item.fileName ?? "", variables),
      })),
    },
    auth: {
      ...payload.auth,
      basic: {
        username: resolveTemplateVariables(payload.auth.basic.username, variables),
        password: resolveTemplateVariables(payload.auth.basic.password, variables),
      },
      bearer: {
        token: resolveTemplateVariables(payload.auth.bearer.token, variables),
      },
      apiKey: {
        ...payload.auth.apiKey,
        key: resolveTemplateVariables(payload.auth.apiKey.key, variables),
        value: resolveTemplateVariables(payload.auth.apiKey.value, variables),
      },
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

function normalizeStreamEventPayload(input: unknown): HttpStreamEventPayload | null {
  if (!input || typeof input !== "object") return null
  const raw = input as Partial<HttpStreamEventPayload>
  if (typeof raw.streamId !== "string" || !raw.streamId.trim()) return null
  if (!isStreamEntryKind(raw.kind)) return null
  if (typeof raw.data !== "string") return null
  if (typeof raw.timestamp !== "string" || !raw.timestamp.trim()) return null
  if (typeof raw.sequence !== "number" || !Number.isFinite(raw.sequence)) return null
  return {
    streamId: raw.streamId,
    kind: raw.kind,
    data: raw.data,
    raw: typeof raw.raw === "string" ? raw.raw : undefined,
    timestamp: raw.timestamp,
    sequence: raw.sequence,
    bytesTotal: typeof raw.bytesTotal === "number" && Number.isFinite(raw.bytesTotal) ? raw.bytesTotal : undefined,
  }
}

function isStreamEntryKind(value: unknown): value is StreamEntryKind {
  return value === "response_start"
    || value === "data"
    || value === "event"
    || value === "chunk"
    || value === "connection_closed"
    || value === "error"
}

export async function sendHttpRequest(
  request: RequestData,
  projectId?: string,
  envId?: string,
  streamOptions?: SendRequestStreamOptions
): Promise<HttpResponse> {
  const envState = useEnvironmentStore.getState()
  let activeVariables = envState.getActiveVariables()
  if (envId) {
    const matched = envState.environments.find((env) => env.id === envId)
    if (matched) {
      activeVariables = (matched.variables ?? [])
        .filter((variable) => variable.enabled && variable.key)
        .map((variable) => ({ key: variable.key, value: variable.value }))
    }
  }
  const payload = resolvePayloadVariables(buildPayload(request), activeVariables)
  payload.url = ensureRequestProtocol(payload.url)
  const uiSettings = useUIStore.getState()
  const cookieStore = useCookieStore.getState()
  const suppressedAutoHeaders = getSuppressedAutoHeaders(request.headers)
  const isSuppressed = (name: string) => suppressedAutoHeaders.has(normalizeHeaderName(name))
  const isTokenSuppressed = isSuppressed("minipost-token") || isSuppressed("postman-token")
  const resolvedRequestUrl = payload.url || ensureRequestProtocol(request.url)
  if (!uiSettings.disableCookies) {
    const cookieHeader = cookieStore.getCookieHeader(resolvedRequestUrl)
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

  const shouldStream = Boolean(streamOptions?.streamId && streamOptions.onStreamEvent)
  let stopStreamListener: (() => void) | null = null

  if (shouldStream) {
    const runtime = await import("../../wailsjs/runtime/runtime")
    stopStreamListener = runtime.EventsOn("minipost:http-stream", (...args: unknown[]) => {
      const payload = normalizeStreamEventPayload(args[0])
      if (!payload || payload.streamId !== streamOptions!.streamId) return
      streamOptions!.onStreamEvent(payload)
    })
  }

  // 始终优先使用 SendRequestWithEnv，以确保历史记录被保存
  let result: Awaited<ReturnType<typeof SendRequestWithEnv>>
  try {
    if (shouldStream) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      result = await SendRequestWithEnvStream(payload as any, projectId || "", envId || "", streamOptions!.streamId)
    } else if (projectId) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      result = await SendRequestWithEnv(payload as any, projectId, envId || "")
    } else {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      result = await SendRequest(payload as any)
    }
  } finally {
    stopStreamListener?.()
  }

  const normalizedResult: HttpResponse = {
    statusCode: result.statusCode,
    statusText: result.statusText,
    headers: result.headers,
    body: result.body,
    bodyBase64: (result as unknown as { bodyBase64?: string }).bodyBase64,
    bodyIsBinary: (result as unknown as { bodyIsBinary?: boolean }).bodyIsBinary,
    duration: result.duration,
    size: result.size,
    contentType: result.contentType,
    protocol: (result as unknown as { protocol?: string }).protocol,
    warnings: (result as unknown as { warnings?: string[] }).warnings ?? [],
    network: (result as unknown as { network?: HttpResponse["network"] }).network,
    timings: (result as unknown as { timings?: HttpResponse["timings"] }).timings,
    sizeDetails: (result as unknown as { sizeDetails?: HttpResponse["sizeDetails"] }).sizeDetails,
  }

  if (!uiSettings.disableCookies) {
    cookieStore.absorbResponseCookies(resolvedRequestUrl, normalizedResult.headers)
  }
  return normalizedResult
}
