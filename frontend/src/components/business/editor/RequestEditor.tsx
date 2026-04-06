import { useEffect, useCallback, useRef } from "react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { UrlBar } from "./UrlBar"
import { ParamsEditor } from "./ParamsEditor"
import { HeadersEditor } from "./HeadersEditor"
import { BodyEditor } from "./BodyEditor"
import { AuthEditor } from "./AuthEditor"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { sendHttpRequest } from "@/services/httpService"
import { useCookieStore } from "@/stores/cookieStore"
import type { RequestData } from "@/types/request"
import type { HttpResponse } from "@/types/response"
import { createKeyValuePair } from "@/types/request"
import { getSuppressedAutoHeaders, isAutoHeaderDisabledMarkerKey, normalizeHeaderName } from "@/lib/autoHeaders"
import { stripJsonComments } from "@/components/ui/CodeEditor"

const TOKEN_HEADER_NAME = "MiniPost-Token"

type StreamStartPayload = {
  statusCode: number
  statusText: string
  headers: Record<string, string[]>
  contentType: string
  protocol?: string
  network?: HttpResponse["network"]
  headerBytes?: number
}

function parseStreamStartPayload(raw: string): StreamStartPayload | null {
  try {
    const parsed = JSON.parse(raw) as Partial<StreamStartPayload>
    if (typeof parsed.statusCode !== "number") return null
    if (typeof parsed.statusText !== "string") return null
    if (!parsed.headers || typeof parsed.headers !== "object") return null
    if (typeof parsed.contentType !== "string") return null
    return {
      statusCode: parsed.statusCode,
      statusText: parsed.statusText,
      headers: parsed.headers as Record<string, string[]>,
      contentType: parsed.contentType,
      protocol: typeof parsed.protocol === "string" ? parsed.protocol : undefined,
      network: parsed.network,
      headerBytes: typeof parsed.headerBytes === "number" ? parsed.headerBytes : undefined,
    }
  } catch {
    return null
  }
}

function dedupeRequestHeaders(headers: RequestData["headers"]): RequestData["headers"] {
  const result: RequestData["headers"] = []
  const indexByName = new Map<string, number>()

  headers.forEach((header) => {
    const normalizedName = normalizeHeaderName(header.key || "")
    if (!normalizedName) return
    const existingIndex = indexByName.get(normalizedName)
    if (existingIndex === undefined) {
      indexByName.set(normalizedName, result.length)
      result.push(header)
      return
    }
    result[existingIndex] = header
  })

  return result
}

function requestWantsStreaming(request: RequestData): boolean {
  const acceptHeader = dedupeRequestHeaders(request.headers).find(
    (header) =>
      header.enabled
      && !isAutoHeaderDisabledMarkerKey(header.key)
      && normalizeHeaderName(header.key) === "accept"
  )
  if (acceptHeader && acceptHeader.value.toLowerCase().includes("text/event-stream")) {
    return true
  }

  const rawCandidate = request.body.type === "json"
    ? request.body.json ?? ""
    : request.body.type === "raw"
      ? request.body.raw ?? ""
      : ""
  const trimmed = rawCandidate.trim()
  if (!trimmed) return false

  try {
    const parsed = JSON.parse(stripJsonComments(trimmed)) as { stream?: unknown }
    return parsed?.stream === true
  } catch {
    return false
  }
}

function buildConsoleRequestBody(request: RequestData): string {
  if (request.body.type === "json") return request.body.json ?? ""
  if (request.body.type === "raw") return request.body.raw ?? ""
  if (request.body.type === "form-urlencoded") {
    const params = new URLSearchParams()
    ;(request.body.formUrlEncoded ?? [])
      .filter((item) => item.enabled && item.key.trim())
      .forEach((item) => params.append(item.key, item.value))
    return params.toString()
  }
  return ""
}

function buildConsoleRequestHeaders(
  request: RequestData,
  options: { disableCookies: boolean; sendNoCacheHeader: boolean; sendPostmanTokenHeader: boolean }
): Record<string, string> {
  const headers: Record<string, string> = {}
  const suppressed = getSuppressedAutoHeaders(request.headers)
  const hasHeader = (name: string) => Object.keys(headers).some((key) => normalizeHeaderName(key) === normalizeHeaderName(name))
  const isSuppressed = (name: string) => suppressed.has(normalizeHeaderName(name))
  const isTokenSuppressed = isSuppressed("minipost-token") || isSuppressed("postman-token")
  dedupeRequestHeaders(request.headers)
    .filter((h) => h.enabled && h.key.trim() && !isAutoHeaderDisabledMarkerKey(h.key))
    .forEach((h) => {
      headers[h.key] = h.value
    })

  if (!isSuppressed("user-agent") && !hasHeader("User-Agent")) {
    headers["User-Agent"] = "MiniPost/1.0"
  }
  if (!isSuppressed("accept") && !hasHeader("Accept")) {
    headers["Accept"] = "*/*"
  }

  if (request.body.type === "json" && !isSuppressed("content-type") && !hasHeader("Content-Type")) {
    headers["Content-Type"] = "application/json"
  }
  if (request.body.type === "form-urlencoded" && !isSuppressed("content-type") && !hasHeader("Content-Type")) {
    headers["Content-Type"] = "application/x-www-form-urlencoded"
  }

  if (!options.disableCookies) {
    const cookieHeader = useCookieStore.getState().getCookieHeader(request.url)
    if (cookieHeader && !isSuppressed("cookie") && !hasHeader("Cookie")) {
      headers["Cookie"] = cookieHeader
    }
  }
  if (options.sendNoCacheHeader && !isSuppressed("cache-control") && !hasHeader("Cache-Control")) {
    headers["Cache-Control"] = "no-cache"
  }
  if (options.sendPostmanTokenHeader && !isTokenSuppressed && !hasHeader("minipost-token") && !hasHeader("postman-token")) {
    headers[TOKEN_HEADER_NAME] = "<calculated when request is sent>"
  }

  return headers
}

function buildRequestWithRuntimeHeaders(
  request: RequestData,
  options: { sendPostmanTokenHeader: boolean }
): RequestData {
  const suppressed = getSuppressedAutoHeaders(request.headers)
  const hasTokenHeader = request.headers.some(
    (header) =>
      header.enabled
      && !isAutoHeaderDisabledMarkerKey(header.key)
      && (normalizeHeaderName(header.key) === "postman-token" || normalizeHeaderName(header.key) === "minipost-token")
  )
  const shouldInjectPostmanToken = options.sendPostmanTokenHeader
    && !suppressed.has("postman-token")
    && !suppressed.has("minipost-token")
    && !hasTokenHeader

  if (!shouldInjectPostmanToken) return request

  return {
    ...request,
    headers: [
      ...request.headers,
      createKeyValuePair({
        key: TOKEN_HEADER_NAME,
        value: crypto.randomUUID(),
        enabled: true,
      }),
    ],
  }
}
 
function useRequestEditorActions() {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const setTabResponse = useTabStore((s) => s.setTabResponse)
  const setTabResponseError = useTabStore((s) => s.setTabResponseError)
  const resetTabStream = useTabStore((s) => s.resetTabStream)
  const appendTabStreamEntry = useTabStore((s) => s.appendTabStreamEntry)
  const setTabStreamActive = useTabStore((s) => s.setTabStreamActive)
  const markTabDirty = useTabStore((s) => s.markTabDirty)
  const { setIsSending } = useUIStore()
  const { currentProjectId, saveRequestToBackend } = useProjectStore()
  const { activeEnvironmentId } = useEnvironmentStore()

  const { addConsoleRequest, updateConsoleResponse, updateConsoleError } = useUIStore()
  const abortRef = useRef<AbortController | null>(null)

  const handleCancel = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
    if (activeTab) {
      setTabStreamActive(activeTab.id, false)
    }
    setIsSending(false)
  }, [activeTab, setIsSending, setTabStreamActive])

  const handleSend = async (downloadAfter = false) => {
    if (!activeTab || !activeTab.request.url.trim()) return

    const tabId = activeTab.id
    if (abortRef.current) abortRef.current.abort()
    abortRef.current = new AbortController()

    setIsSending(true)
    setTabResponse(tabId, null)
    setTabResponseError(tabId, null)
    resetTabStream(tabId)
    setTabStreamActive(tabId, false)

    const uiState = useUIStore.getState()
    const requestForSend = buildRequestWithRuntimeHeaders(activeTab.request, {
      sendPostmanTokenHeader: uiState.sendPostmanTokenHeader,
    })
    const reqHeaders = buildConsoleRequestHeaders(requestForSend, {
      disableCookies: uiState.disableCookies,
      sendNoCacheHeader: uiState.sendNoCacheHeader,
      sendPostmanTokenHeader: uiState.sendPostmanTokenHeader,
    })
    const reqBody = buildConsoleRequestBody(activeTab.request)
    const requestProtocol = uiState.httpVersion === "http2" ? "HTTP/2.0" : "HTTP/1.1"

    const streamId = crypto.randomUUID()
    const shouldUseStreaming = requestWantsStreaming(requestForSend)
    const logId = addConsoleRequest({
      method: activeTab.request.method,
      url: activeTab.request.url,
      requestHeaders: reqHeaders,
      requestBody: reqBody,
      requestProtocol,
    })

    try {
      let streamBaseResponse: HttpResponse | null = null
      let streamStartMs = 0
      const result = await sendHttpRequest(
        requestForSend,
        currentProjectId ?? undefined,
        activeEnvironmentId ?? undefined,
        shouldUseStreaming ? {
          streamId,
          onStreamEvent: (event) => {
            setTabStreamActive(tabId, true)
            if (event.kind === "response_start") {
              const startPayload = parseStreamStartPayload(event.data)
              if (startPayload) {
                streamStartMs = Date.now()
                streamBaseResponse = {
                  statusCode: startPayload.statusCode,
                  statusText: startPayload.statusText,
                  headers: startPayload.headers,
                  body: "",
                  duration: 0,
                  size: event.bytesTotal ?? startPayload.headerBytes ?? 0,
                  contentType: startPayload.contentType,
                  protocol: startPayload.protocol,
                  network: startPayload.network,
                  warnings: [],
                }
                setTabResponse(tabId, streamBaseResponse)
              }
            } else if (streamBaseResponse) {
              const elapsed = streamStartMs > 0 ? Math.max(0, Date.now() - streamStartMs) : streamBaseResponse.duration
              setTabResponse(tabId, {
                ...streamBaseResponse,
                duration: elapsed,
                size: event.bytesTotal ?? streamBaseResponse.size,
              })
            }
            appendTabStreamEntry(tabId, {
              id: `${event.sequence}-${event.timestamp}`,
              kind: event.kind,
              data: event.data,
              raw: event.raw,
              timestamp: event.timestamp,
              sequence: event.sequence,
              bytesTotal: event.bytesTotal,
            })
          },
        } : undefined
      )
      setTabResponse(tabId, result)
      updateConsoleResponse(logId, {
        status: result.statusCode,
        statusText: result.statusText,
        duration: result.duration,
        size: result.size,
        responseHeaders: result.headers,
        responseBody: result.body,
        responseProtocol: result.protocol,
        warnings: result.warnings ?? [],
      })
      if (downloadAfter) {
        const { SaveResponseToFile } = await import("../../../../wailsjs/go/main/App")
        const filename = `response-${Date.now()}.json`
        await SaveResponseToFile(filename, result.body)
      }
    } catch (err) {
      if ((err as Error)?.name === "AbortError") return
      const msg = err instanceof Error ? err.message : String(err)
      setTabResponse(tabId, null)
      setTabResponseError(tabId, msg)
      updateConsoleError(logId, msg)
    } finally {
      abortRef.current = null
      setTabStreamActive(tabId, false)
      setIsSending(false)
    }
  }

  const handleSave = useCallback(async () => {
    if (!activeTab || !currentProjectId) return

    const req = activeTab.request
    const requestItem = {
      id: req.id,
      name: req.name,
      method: req.method,
      url: req.url,
      params: req.params.filter((p: { key: string }) => p.key).map((p: { key: string; value: string }) => ({ key: p.key, value: p.value })),
      headers: req.headers
        .filter((h: { key: string }) => h.key && !isAutoHeaderDisabledMarkerKey(h.key))
        .map((h: { key: string; value: string }) => ({ key: h.key, value: h.value })),
      body: {
        type: req.body.type,
        raw: req.body.raw ?? "",
        json: req.body.json ?? "",
        formUrlEncoded: (req.body.formUrlEncoded ?? []).filter((f: { key: string }) => f.key).map((f: { key: string; value: string }) => ({ key: f.key, value: f.value })),
        formData: (req.body.formData ?? [])
          .filter((f: { key: string }) => f.key)
          .map((f: { key: string; value: string; type: string; filePath?: string; fileName?: string }) => ({
            key: f.key,
            value: f.value,
            type: f.type,
            filePath: f.filePath ?? "",
            fileName: f.fileName ?? "",
          })),
      },
      auth: {
        type: req.auth.type,
        basic: req.auth.basic ?? { username: "", password: "" },
        bearer: req.auth.bearer ?? { token: "" },
        apiKey: req.auth.apiKey ?? { key: "", value: "", addTo: "header" },
      },
      folderId: req.folderId ?? "",
      projectId: currentProjectId,
      createdAt: req.createdAt,
      updatedAt: new Date().toISOString(),
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await saveRequestToBackend(requestItem as any)
    markTabDirty(activeTab.id, false)
  }, [activeTab, currentProjectId, saveRequestToBackend, markTabDirty])

  useEffect(() => {
    const listener = () => handleSave()
    window.addEventListener("minipost:save", listener)
    return () => window.removeEventListener("minipost:save", listener)
  }, [handleSave])

  useEffect(() => {
    const listener = () => void handleSend()
    window.addEventListener("minipost:send", listener)
    return () => window.removeEventListener("minipost:send", listener)
  }, [handleSend])

  return {
    activeTab,
    handleSend,
    handleCancel,
    handleSave,
  }
}

export function RequestEditorToolbar() {
  const { activeTab, handleSend, handleCancel, handleSave } = useRequestEditorActions()

  if (!activeTab) return null

  return <UrlBar onSend={handleSend} onCancel={handleCancel} onSave={handleSave} />
}

export function RequestEditorBody() {
  const activeTab = useTabStore(getProjectActiveTabFromState)

  if (!activeTab) return null

  const { request } = activeTab

  return (
    <div className="flex h-full flex-col bg-[var(--surface)] overflow-hidden">
      <Tabs defaultValue="params" className="flex-1 flex flex-col overflow-hidden">
        <TabsList className="w-full justify-start px-[var(--size-padding-sm)] py-1">
          <TabsTrigger value="params">
            Params
            {request.params.length > 0 && (
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">({request.params.length})</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="headers">
            Headers
            {request.headers.length > 0 && (
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">({request.headers.length})</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="body">Body</TabsTrigger>
          <TabsTrigger value="auth">Auth</TabsTrigger>
        </TabsList>

        <div className="flex-1 min-h-0 pt-1">
          <TabsContent value="params" className="m-0 h-full min-h-0">
            <ParamsEditor />
          </TabsContent>
          <TabsContent value="headers" className="m-0 h-full min-h-0">
            <HeadersEditor />
          </TabsContent>
          <TabsContent value="body" className="m-0 h-full min-h-0">
            <BodyEditor />
          </TabsContent>
          <TabsContent value="auth" className="m-0 h-full min-h-0 overflow-auto">
            <AuthEditor />
          </TabsContent>
        </div>
      </Tabs>
    </div>
  )
}

export function RequestEditor() {
  return (
    <div className="flex h-full min-h-0 flex-col bg-[var(--surface)]">
      <RequestEditorToolbar />
      <div className="flex-1 min-h-0">
        <RequestEditorBody />
      </div>
    </div>
  )
}
