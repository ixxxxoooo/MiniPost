import { useEffect, useCallback, useMemo, useRef, useState } from "react"
import { createPortal } from "react-dom"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { AppIcon } from "@/components/ui/icon"
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
import type { HttpResponse, HttpStreamEntry } from "@/types/response"
import { createKeyValuePair } from "@/types/request"
import { getSuppressedAutoHeaders, isAutoHeaderDisabledMarkerKey, normalizeHeaderName } from "@/lib/autoHeaders"
import { stripJsonComments } from "@/components/ui/CodeEditor"
import { ensureRequestProtocol, resolveTemplateVariables } from "@/lib/variableResolver"
import { areParamsEquivalent, syncParamsWithUrlQuery } from "@/lib/urlQuerySync"
import { buildSaveResponsePayload, suggestResponseFilename } from "@/lib/responseDownload"
import { defaultEditorTabForMethod } from "@/lib/requestEditorTabs"
import { useI18n } from "@/hooks/useI18n"

const TOKEN_HEADER_NAME = "MiniPost-Token"
const REQUEST_EDITOR_TAB_VALUES = ["params", "headers", "body", "auth"] as const
const REQUEST_EDITOR_TAB_STORAGE_KEY = "minipost:request-editor-tabs"

type RequestEditorTabValue = typeof REQUEST_EDITOR_TAB_VALUES[number]

function isRequestEditorTabValue(value: unknown): value is RequestEditorTabValue {
  return typeof value === "string" && REQUEST_EDITOR_TAB_VALUES.includes(value as RequestEditorTabValue)
}

function readPersistedRequestEditorTabs(): Record<string, RequestEditorTabValue> {
  if (typeof window === "undefined") return {}
  try {
    const raw = window.localStorage.getItem(REQUEST_EDITOR_TAB_STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const result: Record<string, RequestEditorTabValue> = {}
    Object.entries(parsed).forEach(([key, value]) => {
      if (isRequestEditorTabValue(value)) result[key] = value
    })
    return result
  } catch {
    return {}
  }
}

function persistRequestEditorTabs(value: Record<string, RequestEditorTabValue>) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(REQUEST_EDITOR_TAB_STORAGE_KEY, JSON.stringify(value))
  } catch {
    // UI preference persistence should never block editing.
  }
}

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

function isMeaningfulKeyValue(item: { key?: string; value?: string; description?: string }): boolean {
  return Boolean((item.key ?? "").trim() || (item.value ?? "").trim() || (item.description ?? "").trim())
}

type FolderTreeOption = {
  id: string
  name: string
  depth: number
}

function buildFolderTreeOptions(folders: Array<{ id: string; name: string; parentId?: string; sortOrder: number }>): FolderTreeOption[] {
  const grouped = new Map<string, Array<{ id: string; name: string; parentId?: string; sortOrder: number }>>()
  folders.forEach((folder) => {
    const parentId = folder.parentId || ""
    const list = grouped.get(parentId) ?? []
    list.push(folder)
    grouped.set(parentId, list)
  })
  grouped.forEach((list) => {
    list.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name))
  })

  const result: FolderTreeOption[] = []
  const walk = (parentId: string, depth: number) => {
    const children = grouped.get(parentId) ?? []
    children.forEach((child) => {
      result.push({ id: child.id, name: child.name, depth })
      walk(child.id, depth + 1)
    })
  }

  walk("", 0)
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
    const activeVariables = useEnvironmentStore.getState().getActiveVariables()
    const resolvedUrl = resolveTemplateVariables(request.url, activeVariables)
    const cookieHeader = useCookieStore.getState().getCookieHeader(resolvedUrl || request.url)
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
  const updateTab = useTabStore((s) => s.updateTab)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const setTabResponse = useTabStore((s) => s.setTabResponse)
  const setTabResponseError = useTabStore((s) => s.setTabResponseError)
  const resetTabStream = useTabStore((s) => s.resetTabStream)
  const appendTabStreamEntries = useTabStore((s) => s.appendTabStreamEntries)
  const setTabStreamActive = useTabStore((s) => s.setTabStreamActive)
  const setTabSending = useTabStore((s) => s.setTabSending)
  const markTabDirty = useTabStore((s) => s.markTabDirty)
  const { currentProjectId, saveRequestToBackend, folders } = useProjectStore()
  const { activeEnvironmentId } = useEnvironmentStore()

  const { addConsoleRequest, updateConsoleResponse, updateConsoleError } = useUIStore()
  const abortRefByTab = useRef<Record<string, AbortController>>({})
  const sendingSeqByTabRef = useRef<Record<string, number>>({})
  const [saveDraftDialogOpen, setSaveDraftDialogOpen] = useState(false)
  const [saveDraftName, setSaveDraftName] = useState("")
  const [saveDraftFolderId, setSaveDraftFolderId] = useState("")
  const [saveDraftSaving, setSaveDraftSaving] = useState(false)
  const [saveDraftError, setSaveDraftError] = useState("")
  const folderOptions = useMemo(() => buildFolderTreeOptions(folders), [folders])

  const handleCancel = useCallback(() => {
    if (!activeTab) return
    const tabId = activeTab.id
    abortRefByTab.current[tabId]?.abort()
    delete abortRefByTab.current[tabId]
    sendingSeqByTabRef.current[tabId] = (sendingSeqByTabRef.current[tabId] ?? 0) + 1
    setTabStreamActive(tabId, false)
    setTabSending(tabId, false)
  }, [activeTab, setTabSending, setTabStreamActive])

  const handleSend = async (downloadAfter = false) => {
    if (!activeTab || !activeTab.request.url.trim()) return

    const tabId = activeTab.id
    abortRefByTab.current[tabId]?.abort()
    const abortController = new AbortController()
    abortRefByTab.current[tabId] = abortController

    const sendingSeq = (sendingSeqByTabRef.current[tabId] ?? 0) + 1
    sendingSeqByTabRef.current[tabId] = sendingSeq
    const isCurrentTabSend = () => sendingSeqByTabRef.current[tabId] === sendingSeq

    setTabSending(tabId, true)
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
    const activeVariables = useEnvironmentStore.getState().getActiveVariables()
    const resolvedRequestUrl = ensureRequestProtocol(resolveTemplateVariables(activeTab.request.url, activeVariables) || activeTab.request.url)
    const reqBody = buildConsoleRequestBody(activeTab.request)
    const requestProtocol = uiState.httpVersion === "http2" ? "HTTP/2.0" : "HTTP/1.1"

    const streamId = crypto.randomUUID()
    const shouldUseStreaming = requestWantsStreaming(requestForSend) || requestForSend.method === "GET"
    const logId = addConsoleRequest({
      method: activeTab.request.method,
      url: resolvedRequestUrl,
      requestHeaders: reqHeaders,
      requestBody: reqBody,
      requestProtocol,
    })

    try {
      let streamBaseResponse: HttpResponse | null = null
      let streamStartMs = 0
      let streamIsSSE = false
      let lastProgressUpdateMs = 0
      let pendingEntries: HttpStreamEntry[] = []
      let flushTimer: number | null = null

      const flushPendingEntries = () => {
        flushTimer = null
        if (!isCurrentTabSend() || pendingEntries.length === 0) {
          pendingEntries = []
          return
        }
        appendTabStreamEntries(tabId, pendingEntries)
        pendingEntries = []
      }

      const scheduleFlush = () => {
        if (flushTimer !== null) return
        flushTimer = window.setTimeout(flushPendingEntries, 120)
      }

      const maybeUpdateSSEProgress = (event: {
        kind: string
        bytesTotal?: number
      }) => {
        if (!streamBaseResponse) return
        const now = Date.now()
        const forceUpdate = event.kind === "connection_closed" || event.kind === "error"
        if (!forceUpdate && now - lastProgressUpdateMs < 180) return
        lastProgressUpdateMs = now

        const elapsed = streamStartMs > 0 ? Math.max(0, now - streamStartMs) : streamBaseResponse.duration
        const nextResponse = {
          ...streamBaseResponse,
          duration: elapsed,
          size: event.bytesTotal ?? streamBaseResponse.size,
        }
        setTabResponse(tabId, nextResponse)
        updateConsoleResponse(logId, {
          status: nextResponse.statusCode,
          statusText: nextResponse.statusText,
          duration: nextResponse.duration,
          size: nextResponse.size,
          responseHeaders: nextResponse.headers,
          responseBody: nextResponse.body,
          responseProtocol: nextResponse.protocol,
          warnings: nextResponse.warnings ?? [],
        })
      }

      const result = await sendHttpRequest(
        requestForSend,
        currentProjectId ?? undefined,
        activeEnvironmentId ?? undefined,
        shouldUseStreaming ? {
          streamId,
          onStreamEvent: (event) => {
            if (!isCurrentTabSend()) return
            if (event.kind === "response_start") {
              const startPayload = parseStreamStartPayload(event.data)
              if (!startPayload) return

              streamIsSSE = startPayload.contentType.toLowerCase().includes("text/event-stream")
              if (!streamIsSSE) return

              setTabStreamActive(tabId, true)
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
              updateConsoleResponse(logId, {
                status: startPayload.statusCode,
                statusText: startPayload.statusText,
                duration: 0,
                size: event.bytesTotal ?? startPayload.headerBytes ?? 0,
                responseHeaders: startPayload.headers,
                responseBody: "",
                responseProtocol: startPayload.protocol,
                warnings: [],
              })
            } else {
              if (!streamIsSSE) return
              setTabStreamActive(tabId, true)
              maybeUpdateSSEProgress(event)
            }
            if (!streamIsSSE) return
            pendingEntries.push({
              id: `${event.sequence}-${event.timestamp}`,
              kind: event.kind,
              data: event.data,
              raw: event.raw,
              timestamp: event.timestamp,
              sequence: event.sequence,
              bytesTotal: event.bytesTotal,
            })
            scheduleFlush()
          },
        } : undefined
      )
      if (flushTimer !== null) {
        window.clearTimeout(flushTimer)
        flushTimer = null
      }
      flushPendingEntries()
      if (!isCurrentTabSend()) return
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
        const filename = suggestResponseFilename({
          headers: result.headers,
          contentType: result.contentType,
          requestUrl: resolvedRequestUrl,
        })
        const payload = buildSaveResponsePayload({
          body: result.body,
          bodyBase64: result.bodyBase64,
          bodyIsBinary: result.bodyIsBinary,
        })
        await SaveResponseToFile(filename, payload)
      }
    } catch (err) {
      if (!isCurrentTabSend()) return
      if ((err as Error)?.name === "AbortError") return
      const msg = err instanceof Error ? err.message : String(err)
      setTabResponse(tabId, null)
      setTabResponseError(tabId, msg)
      updateConsoleError(logId, msg)
    } finally {
      // ensure queued stream events are flushed before request settles
      const currentAbortController = abortRefByTab.current[tabId]
      if (currentAbortController === abortController) {
        delete abortRefByTab.current[tabId]
      }
      if (sendingSeqByTabRef.current[tabId] === sendingSeq) {
        setTabStreamActive(tabId, false)
        setTabSending(tabId, false)
      }
    }
  }

  const persistTabRequest = useCallback(async (
    tab: NonNullable<typeof activeTab>,
    overrides?: { name?: string; folderId?: string }
  ) => {
    if (!currentProjectId) return

    const requestName = overrides?.name ?? tab.request.name
    const requestFolderId = overrides?.folderId ?? tab.request.folderId ?? ""
    const req = tab.request
    const requestItem = {
      id: req.id,
      name: requestName,
      method: req.method,
      url: req.url,
      params: req.params
        .filter((p: { key: string }) => p.key)
        .map((p: { key: string; value: string; description?: string }) => ({ key: p.key, value: p.value, description: p.description ?? "" })),
      headers: req.headers
        .filter((h: { key: string }) => h.key && !isAutoHeaderDisabledMarkerKey(h.key))
        .map((h: { key: string; value: string; description?: string }) => ({ key: h.key, value: h.value, description: h.description ?? "" })),
      body: {
        type: req.body.type,
        raw: req.body.raw ?? "",
        json: req.body.json ?? "",
        formUrlEncoded: (req.body.formUrlEncoded ?? [])
          .filter((f: { key: string }) => f.key)
          .map((f: { key: string; value: string; description?: string }) => ({ key: f.key, value: f.value, description: f.description ?? "" })),
        formData: (req.body.formData ?? [])
          .filter((f: { key: string }) => f.key)
          .map((f: { key: string; value: string; description?: string; type: string; filePath?: string; fileName?: string }) => ({
            key: f.key,
            value: f.value,
            description: f.description ?? "",
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
      folderId: requestFolderId,
      projectId: currentProjectId,
      createdAt: req.createdAt,
      updatedAt: new Date().toISOString(),
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await saveRequestToBackend(requestItem as any)
    return { requestName, requestFolderId }
  }, [currentProjectId, saveRequestToBackend])

  const handleSave = useCallback(async () => {
    if (!activeTab || !currentProjectId) return

    if (!activeTab.requestId) {
      setSaveDraftName((activeTab.request.name || activeTab.title || "Untitled").trim())
      setSaveDraftFolderId(activeTab.request.folderId ?? "")
      setSaveDraftError("")
      setSaveDraftDialogOpen(true)
      return
    }

    await persistTabRequest(activeTab)
    markTabDirty(activeTab.id, false)
  }, [activeTab, currentProjectId, markTabDirty, persistTabRequest])

  const handleConfirmDraftSave = useCallback(async () => {
    if (!activeTab || !currentProjectId || saveDraftSaving) return

    const name = saveDraftName.trim() || activeTab.request.name || activeTab.title || "Untitled"
    const folderId = saveDraftFolderId
    setSaveDraftSaving(true)
    setSaveDraftError("")

    try {
      await persistTabRequest(activeTab, { name, folderId })
      updateTab(activeTab.id, { title: name, requestId: activeTab.request.id })
      updateTabRequest(activeTab.id, { name, folderId, projectId: currentProjectId })
      markTabDirty(activeTab.id, false)
      setSaveDraftDialogOpen(false)
    } catch (err) {
      setSaveDraftError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaveDraftSaving(false)
    }
  }, [
    activeTab,
    currentProjectId,
    markTabDirty,
    persistTabRequest,
    saveDraftFolderId,
    saveDraftName,
    saveDraftSaving,
    updateTab,
    updateTabRequest,
  ])

  const handleCancelDraftSave = useCallback(() => {
    if (saveDraftSaving) return
    setSaveDraftDialogOpen(false)
    setSaveDraftError("")
  }, [saveDraftSaving])

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
    saveDraftDialogOpen,
    saveDraftName,
    saveDraftFolderId,
    saveDraftSaving,
    saveDraftError,
    folderOptions,
    setSaveDraftName,
    setSaveDraftFolderId,
    handleConfirmDraftSave,
    handleCancelDraftSave,
  }
}

export function RequestEditorToolbar() {
  const {
    activeTab,
    handleSend,
    handleCancel,
    handleSave,
    saveDraftDialogOpen,
    saveDraftName,
    saveDraftFolderId,
    saveDraftSaving,
    saveDraftError,
    folderOptions,
    setSaveDraftName,
    setSaveDraftFolderId,
    handleConfirmDraftSave,
    handleCancelDraftSave,
  } = useRequestEditorActions()
  const { t } = useI18n()

  if (!activeTab) return null

  const ROOT_FOLDER_VALUE = "__root__"
  const selectedFolderDisplay = saveDraftFolderId
    ? folderOptions.find((option) => option.id === saveDraftFolderId)?.name
    : t("根目录", "Root")

  return (
    <>
      <UrlBar onSend={handleSend} onCancel={handleCancel} onSave={handleSave} />
      {saveDraftDialogOpen && createPortal(
        <div className="fixed inset-0 z-[320] flex items-center justify-center" onClick={() => { if (!saveDraftSaving) handleCancelDraftSave() }}>
          <div className="absolute inset-0 bg-black/35 backdrop-blur-[1px]" />
          <div
            className="relative z-[321] w-[460px] rounded-[12px] border border-[var(--border-color)] bg-[var(--surface)] shadow-[var(--shadow-lg)]"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-[var(--border-subtle)] px-4 py-3">
              <div>
                <div className="text-[16px] font-semibold text-[var(--fg)]">{t("保存请求", "Save Request")}</div>
                <div className="mt-0.5 text-[12px] text-[var(--fg-secondary)]">{t("请选择名称和保存位置", "Choose request name and save location")}</div>
              </div>
              <button
                type="button"
                className="h-6 w-6 inline-flex items-center justify-center rounded-[6px] text-[var(--fg-muted)] hover:bg-[var(--button-bg)] hover:text-[var(--fg)]"
                onClick={handleCancelDraftSave}
                disabled={saveDraftSaving}
              >
                <AppIcon name="clear" size={12} />
              </button>
            </div>
            <div className="space-y-3 px-4 py-3.5">
              <div className="space-y-1">
                <label className="text-[12px] text-[var(--fg-secondary)]">{t("名称", "Name")}</label>
                <input
                  value={saveDraftName}
                  onChange={(event) => setSaveDraftName(event.target.value)}
                  className="h-8 w-full rounded-[7px] border border-[var(--border-color)] bg-[var(--surface)] px-2.5 text-[12px] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
                  placeholder={t("请输入请求名称", "Enter request name")}
                  autoFocus
                />
              </div>
              <div className="space-y-1">
                <label className="text-[12px] text-[var(--fg-secondary)]">{t("位置", "Location")}</label>
                <Select
                  value={saveDraftFolderId || ROOT_FOLDER_VALUE}
                  onValueChange={(value) => setSaveDraftFolderId(value === ROOT_FOLDER_VALUE ? "" : value)}
                >
                  <SelectTrigger className="h-8 text-[12px]">
                    <SelectValue>{selectedFolderDisplay}</SelectValue>
                  </SelectTrigger>
                  <SelectContent className="z-[360]">
                    <SelectItem value={ROOT_FOLDER_VALUE}>{t("根目录", "Root")}</SelectItem>
                    {folderOptions.map((option) => (
                      <SelectItem key={option.id} value={option.id}>
                        {`${"\u3000".repeat(option.depth)}${option.name}`}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {saveDraftError && (
                <div className="rounded-[7px] border border-[var(--danger)]/30 bg-[var(--danger)]/10 px-2.5 py-2 text-[11px] text-[var(--danger)]">
                  {saveDraftError}
                </div>
              )}
            </div>
            <div className="flex items-center justify-end gap-2 border-t border-[var(--border-subtle)] px-4 py-3">
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] border border-[var(--button-border)] text-[12px] text-[var(--fg)] hover:bg-[var(--button-bg)]"
                onClick={handleCancelDraftSave}
                disabled={saveDraftSaving}
              >
                {t("取消", "Cancel")}
              </button>
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] bg-[var(--accent)] text-white text-[12px] font-medium hover:opacity-95 disabled:opacity-60"
                onClick={() => void handleConfirmDraftSave()}
                disabled={saveDraftSaving}
              >
                {saveDraftSaving ? t("保存中...", "Saving...") : t("保存", "Save")}
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </>
  )
}

export function RequestEditorBody() {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const [editorTabByRequestKey, setEditorTabByRequestKey] = useState<Record<string, RequestEditorTabValue>>(readPersistedRequestEditorTabs)
  const querySyncSnapshotRef = useRef<{
    tabId: string
    url: string
    params: RequestData["params"]
  } | null>(null)

  useEffect(() => {
    if (!activeTab) {
      querySyncSnapshotRef.current = null
      return
    }

    const previous = querySyncSnapshotRef.current
    const tabChanged = previous?.tabId !== activeTab.id
    const urlChanged = tabChanged || previous?.url !== activeTab.request.url
    const paramsChanged = Boolean(
      previous
      && !tabChanged
      && !areParamsEquivalent(previous.params, activeTab.request.params)
    )

    querySyncSnapshotRef.current = {
      tabId: activeTab.id,
      url: activeTab.request.url,
      params: activeTab.request.params,
    }

    if (!urlChanged || paramsChanged) return

    const nextParams = syncParamsWithUrlQuery(activeTab.request.url, activeTab.request.params)
    if (areParamsEquivalent(activeTab.request.params, nextParams)) return
    querySyncSnapshotRef.current = {
      tabId: activeTab.id,
      url: activeTab.request.url,
      params: nextParams,
    }
    updateTabRequest(activeTab.id, { params: nextParams })
  }, [activeTab?.id, activeTab?.request.params, activeTab?.request.url, updateTabRequest])

  if (!activeTab) return null

  const { request } = activeTab
  const paramsCount = request.params.filter(isMeaningfulKeyValue).length
  const headersCount = request.headers.filter((item) => isMeaningfulKeyValue(item) && !isAutoHeaderDisabledMarkerKey(item.key ?? "")).length
  const requestEditorTabKey = `${activeTab.projectId || request.projectId || "project"}:${activeTab.requestId || request.id}`
  // 用户没有手动切换过标签时，按请求方法决定默认定位：GET 类停在 Params，POST 类停在 Body
  const activeEditorTab = editorTabByRequestKey[requestEditorTabKey] ?? defaultEditorTabForMethod(request.method)

  return (
    <div className="flex h-full flex-col bg-[var(--surface)] overflow-hidden">
      <Tabs
        value={activeEditorTab}
        onValueChange={(value) => {
          if (!isRequestEditorTabValue(value)) return
          setEditorTabByRequestKey((previous) => {
            const next = {
              ...previous,
              [requestEditorTabKey]: value,
            }
            persistRequestEditorTabs(next)
            return next
          })
        }}
        className="flex-1 flex flex-col overflow-hidden"
      >
        <TabsList className="w-full justify-start px-[var(--size-padding-sm)] py-1">
          <TabsTrigger value="params">
            Params
            {paramsCount > 0 && (
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">({paramsCount})</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="headers">
            Headers
            {headersCount > 0 && (
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">({headersCount})</span>
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
