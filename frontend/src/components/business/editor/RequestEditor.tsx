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
 
function useRequestEditorActions() {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const setTabResponse = useTabStore((s) => s.setTabResponse)
  const setTabResponseError = useTabStore((s) => s.setTabResponseError)
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
    setIsSending(false)
  }, [setIsSending])

  const handleSend = async (downloadAfter = false) => {
    if (!activeTab || !activeTab.request.url.trim()) return

    if (abortRef.current) abortRef.current.abort()
    abortRef.current = new AbortController()

    setIsSending(true)
    setTabResponseError(activeTab.id, null)

    const reqHeaders: Record<string, string> = {}
    activeTab.request.headers.filter((h) => h.enabled && h.key).forEach((h) => { reqHeaders[h.key] = h.value })

    const logId = addConsoleRequest({
      method: activeTab.request.method,
      url: activeTab.request.url,
      requestHeaders: reqHeaders,
    })

    try {
      const result = await sendHttpRequest(
        activeTab.request,
        currentProjectId ?? undefined,
        activeEnvironmentId ?? undefined,
      )
      setTabResponse(activeTab.id, result)
      updateConsoleResponse(logId, {
        status: result.statusCode,
        duration: result.duration,
        size: result.size,
        responseHeaders: result.headers,
        responseBody: result.body,
      })
      if (downloadAfter) {
        const { SaveResponseToFile } = await import("../../../../wailsjs/go/main/App")
        const filename = `response-${Date.now()}.json`
        await SaveResponseToFile(filename, result.body)
      }
    } catch (err) {
      if ((err as Error)?.name === "AbortError") return
      const msg = err instanceof Error ? err.message : String(err)
      setTabResponseError(activeTab.id, msg)
      updateConsoleError(logId, msg)
    } finally {
      abortRef.current = null
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
      headers: req.headers.filter((h: { key: string }) => h.key).map((h: { key: string; value: string }) => ({ key: h.key, value: h.value })),
      body: {
        type: req.body.type,
        raw: req.body.raw ?? "",
        json: req.body.json ?? "",
        formUrlEncoded: (req.body.formUrlEncoded ?? []).filter((f: { key: string }) => f.key).map((f: { key: string; value: string }) => ({ key: f.key, value: f.value })),
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

        <div className="flex-1 overflow-y-auto pt-1">
          <TabsContent value="params" className="m-0">
            <ParamsEditor />
          </TabsContent>
          <TabsContent value="headers" className="m-0">
            <HeadersEditor />
          </TabsContent>
          <TabsContent value="body" className="m-0">
            <BodyEditor />
          </TabsContent>
          <TabsContent value="auth" className="m-0">
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
